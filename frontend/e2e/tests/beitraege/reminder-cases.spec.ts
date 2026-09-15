import { test, expect, type Page } from '../../fixtures/coverage';

/**
 * Family-based reminder workflow (Erinnerungen).
 *
 * Requires the e2e stack from docs/test-seeding.md: backend-fees on :8081
 * (SMTP disabled is fine) and the beitraege dev server on :5175. Seed data is
 * created via the API in the test body so the suite stays self-contained:
 * overdue fees become actionable on the next worklist load.
 */

const OVERDUE_DUE_DATE = '2026-01-05';
const FUTURE_DUE_DATE = '2099-12-31';

async function loginApi(page: Page): Promise<string> {
  const response = await page.request.post('/api/fees/v1/auth/login', {
    data: { email: 'admin@knirpsenstadt.de', password: 'admin123' },
  });
  expect(response.ok()).toBeTruthy();
  const body = await response.json();
  return body.accessToken as string;
}

interface SeededFamily {
  householdId: string;
  overdueFeeId: string;
  futureFeeId: string;
}

async function seedFamily(page: Page, token: string, name: string, withEmail: boolean): Promise<SeededFamily> {
  const headers = { Authorization: `Bearer ${token}` };

  const householdResponse = await page.request.post('/api/fees/v1/households', {
    headers,
    data: { name },
  });
  expect(householdResponse.ok()).toBeTruthy();
  const household = await householdResponse.json();

  const childResponse = await page.request.post('/api/fees/v1/children', {
    headers,
    data: {
      memberNumber: `T${Math.floor(100000 + Math.random() * 899999)}`,
      firstName: 'Reminder',
      lastName: name,
      birthDate: '2024-02-10',
      entryDate: '2025-08-01',
    },
  });
  expect(childResponse.ok()).toBeTruthy();
  const child = await childResponse.json();

  // Assign child and parent to the household; fees inherit the household.
  const linkChildResponse = await page.request.post(`/api/fees/v1/households/${household.id}/children`, {
    headers,
    data: { childId: child.id },
  });
  expect(linkChildResponse.status()).toBe(204);

  if (withEmail) {
    const parentResponse = await page.request.post('/api/fees/v1/parents', {
      headers,
      data: {
        firstName: 'E2E',
        lastName: name,
        email: `e2e-${Date.now()}@example.test`,
      },
    });
    expect(parentResponse.ok()).toBeTruthy();
    const parent = await parentResponse.json();
    const linkParentResponse = await page.request.post(`/api/fees/v1/households/${household.id}/parents`, {
      headers,
      data: { parentId: parent.id },
    });
    expect(linkParentResponse.status()).toBe(204);
  }

  const createFee = async (dueDate: string) => {
    const feeResponse = await page.request.post('/api/fees/v1/fees', {
      headers,
      data: {
        childId: child.id,
        feeType: 'FOOD',
        year: Number(dueDate.slice(0, 4)),
        month: Number(dueDate.slice(5, 7)),
        dueDate,
      },
    });
    expect(feeResponse.ok()).toBeTruthy();
    return (await feeResponse.json()).id as string;
  };

  return {
    householdId: household.id,
    overdueFeeId: await createFee(OVERDUE_DUE_DATE),
    futureFeeId: await createFee(FUTURE_DUE_DATE),
  };
}

async function openReminders(page: Page): Promise<void> {
  await page.goto('/beitraege/automatisierung');
  await expect(page.getByRole('heading', { name: 'Erinnerungen' })).toBeVisible({ timeout: 15000 });
}

test.describe('Familienbasierter Erinnerungs-Workflow', () => {
  test('Arbeitsliste, Filter, Auswahl, Vorschau und Bestätigung', async ({ page }) => {
    const token = await loginApi(page);
    const familyName = `E2E Reminder ${Date.now()}`;
    const seeded = await seedFamily(page, token, familyName, true);

    await openReminders(page);

    // Worklist shows the new family with overdue fees (Handlungsbedarf).
    const familyRow = page.locator('li button', { hasText: familyName }).first();
    await expect(familyRow).toBeVisible({ timeout: 15000 });

    // Client-side search narrows the list.
    await page.getByPlaceholder('Familie suchen...').fill(familyName);
    await expect(page.locator('li button')).toHaveCount(1);
    await page.getByPlaceholder('Familie suchen...').fill('');

    // Open the family: master-detail on desktop.
    await familyRow.click();
    await expect(page.getByRole('heading', { name: familyName })).toBeVisible();

    // Only actionable fees are preselected; the future fee is visible but unchecked.
    const feeRows = page.locator('table tbody tr');
    await expect(feeRows).toHaveCount(2);
    const checkboxes = feeRows.locator('input[type="checkbox"]');
    await expect(checkboxes.nth(0)).toBeChecked();
    await expect(checkboxes.nth(1)).not.toBeChecked();

    // Status badges show Erinnerung fällig / Nicht fällig.
    await expect(page.getByText('Erinnerung fällig')).toBeVisible();
    await expect(page.getByText('Nicht fällig')).toBeVisible();

    // Live preview appears with the server deadline and unified subject.
    const subjectInput = page.locator('label:has-text("Betreff") + input').first();
    await expect(subjectInput).toHaveValue('Kita Zahlungserinnerung: offene Beiträge', { timeout: 15000 });
    await expect(page.locator('text=Vorschau · Frist:')).toBeVisible();

    // QR toggle regenerates the preview without losing the selection.
    const qrToggle = page.getByLabel('QR-Code');
    await qrToggle.uncheck();
    await expect(page.getByText('QR-Code ist deaktiviert')).toBeVisible();
    await qrToggle.check();

    // Editing the subject marks the text as edited.
    await subjectInput.fill('E2E geänderter Betreff');
    await expect(page.getByText('Text angepasst')).toBeVisible();

    // Switching the stage regenerates the text and resets manual edits with a notice.
    await page.getByRole('button', { name: 'Mahnung', exact: true }).click();
    await expect(page.getByText('Manuelle Textänderungen wurden zurückgesetzt')).toBeVisible({ timeout: 15000 });
    await expect(subjectInput).toHaveValue('Kita Mahnung: offene Beiträge');
    await expect(page.getByText('· Reminder: Essensgeld wurde noch nicht erinnert')).toBeVisible();

    // Send flow opens the confirmation dialog.
    await page.getByRole('button', { name: 'Erinnerung', exact: true }).click();
    await page.getByRole('button', { name: /senden$/ }).click();
    await expect(page.getByRole('heading', { name: /Erinnerung senden\?/ })).toBeVisible();
    await expect(page.getByRole('definition').filter({ hasText: familyName })).toBeVisible();

    // SMTP is disabled in e2e: the backend answers 503 and the UI shows an error.
    await page.getByRole('button', { name: 'Jetzt senden' }).click();
    await expect(page.getByText(/Versand fehlgeschlagen|email service disabled/i)).toBeVisible({ timeout: 15000 });
  });

  test('Familie ohne E-Mail-Adresse ist blockiert', async ({ page }) => {
    const token = await loginApi(page);
    const familyName = `E2E Blockiert ${Date.now()}`;
    await seedFamily(page, token, familyName, false);

    await openReminders(page);
    const familyRow = page.locator('li button', { hasText: familyName }).first();
    await expect(familyRow).toBeVisible({ timeout: 15000 });
    await expect(familyRow.getByText('Keine E-Mail')).toBeVisible();

    await familyRow.click();
    await expect(page.getByText('keine gültige E-Mail-Adresse')).toBeVisible();
    await expect(page.getByRole('button', { name: /senden$/ })).toBeDisabled();
  });

  test('Scope-Filter Alle offenen zeigt wartende Familien', async ({ page }) => {
    await openReminders(page);

    const actionableButton = page.getByRole('button', { name: 'Handlungsbedarf' });
    await expect(actionableButton).toBeVisible();

    await page.getByRole('button', { name: 'Alle offenen' }).click();
    // Both scopes load without error; the list container is present.
    await expect(page.locator('li button').first()).toBeVisible({ timeout: 15000 });
  });
});
