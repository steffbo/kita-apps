import { test, expect, uniq, loginContext, type Api } from './fixtures';
import { berlinToday } from './bank';

function memberNumber() {
  return String(10_000_000 + Math.floor(Math.random() * 90_000_000));
}
async function family(adminApi: Api, label: string) {
  const suffix = uniq();
  const today = berlinToday();
  const email = `eltern-${suffix}@e2e.test`;
  const child = await adminApi.post<{ id: string }>('/children', {
    memberNumber: memberNumber(), firstName: label, lastName: suffix,
    birthDate: `${today.year - 5}-01-01`, entryDate: `${today.year - 2}-01-01`,
  });
  const parent = await adminApi.post<{ id: string }>('/parents', {
    firstName: 'Eva', lastName: suffix, email, phone: '030 11111',
  });
  await adminApi.post(`/children/${child.id}/parents`, { parentId: parent.id, isPrimary: true });
  return { child, parent, email, childName: `${label} ${suffix}`, parentName: `Eva ${suffix}` };
}

test('Eltern sehen nur die eigene Familie; Meldungen und Änderungen gehen an Staff',
  async ({ adminApi, adminPage, browser }) => {
    const a = await family(adminApi, 'KindA');
    const b = await family(adminApi, 'KindB');
    const partner = await adminApi.post<{ id: string }>('/parents', {
      firstName: 'Max', lastName: a.parentName.split(' ')[1], phone: '030 33333',
    });
    await adminApi.post(`/children/${a.child.id}/parents`, { parentId: partner.id, isPrimary: false });
    const password = `pw-${uniq()}`;
    await adminApi.post('/users', {
      email: a.email, firstName: 'Eva', lastName: 'Eltern', role: 'PARENT',
      isActive: true, password,
    });
    const context = await browser.newContext();
    try {
      await loginContext(context, a.email, password);
      const page = await context.newPage();
      await page.goto('/beitraege/kinder');
      await expect(page).toHaveURL(/\/beitraege\/familie$/);
      await expect(page.getByText(a.childName)).toBeVisible();
      await expect(page.getByText(b.childName)).toHaveCount(0);
      await page.getByRole('link', { name: 'Elternstunden' }).click();
      await page.getByRole('button', { name: 'Stunden melden' }).click();
      const form = page.getByRole('dialog', { name: 'Stunden melden' });
      // Esc closes dialogs app-wide.
      await page.keyboard.press('Escape');
      await expect(form).toHaveCount(0);
      await page.getByRole('button', { name: 'Stunden melden' }).click();
      await expect(form.getByLabel('Kind (optional)')).toHaveCount(0);
      await form.getByRole('button', { name: '2,5 Std.' }).click();
      await expect(form.getByText('2 Std. 30 Min.')).toBeVisible();
      // Native validation bubbles speak German.
      await form.getByRole('button', { name: 'Melden' }).click();
      await expect.poll(() => form.getByLabel('Anlass').evaluate((el) => (el as HTMLInputElement).validationMessage))
        .toBe('Bitte dieses Feld ausfüllen.');
      await form.getByLabel('Anlass').fill('Sommerfest E2E');
      await form.getByRole('button', { name: 'Melden' }).click();
      await expect(page.getByText('Eingereicht')).toBeVisible();
      await expect(page.getByText('Sommerfest E2E')).toBeVisible();

      await adminPage.goto('/beitraege/elternstunden');
      await adminPage.getByRole('button', { name: /Unbestätigte Meldungen/ }).click();
      const row = adminPage.getByRole('row', { name: new RegExp(a.childName) });
      await expect(row).toContainText('Meldungen');
      await row.getByRole('link').click();
      const entry = adminPage.getByRole('row', { name: /Sommerfest E2E/ });
      await expect(entry).toContainText('Eltern');
      await entry.getByRole('button', { name: 'Bestätigen' }).click();
      await expect(entry).toContainText('Bestätigt');
      await expect(entry).toContainText(/von .+, \d{2}\.\d{2}\.\d{4}/);
      await page.reload();
      await expect(page.getByText('Bestätigt', { exact: true })).toBeVisible();
      await expect(page.getByText('2,5 h').first()).toBeVisible();

      await page.goto('/beitraege/familie/daten');
      const own = page.locator('form').filter({ hasText: 'Deine Kontaktdaten' });
      await own.getByLabel('Telefon').fill('030 22222');
      await own.getByRole('button', { name: 'Kontaktdaten speichern' }).click();
      await expect(page.getByRole('status')).toContainText('gespeichert');
      const other = page.locator('form').filter({ hasText: 'Kontaktdaten von Max' });
      await expect(other.getByLabel('E-Mail')).toBeDisabled();
      await other.getByLabel('Telefon').fill('030 44444');
      await other.getByRole('button', { name: 'Kontaktdaten speichern' }).click();
      await expect(page.getByRole('status')).toContainText('Kontaktdaten von Max');
      // Master data is read-only for parents; corrections go through „Fehler melden“.
      const kid = page.locator('article').filter({ hasText: a.childName });
      await expect(kid.getByRole('textbox')).toHaveCount(0);
      await expect(kid).toContainText('Mitgliedsnummer');
      await adminPage.goto('/beitraege/');
      await expect(adminPage.getByText(/\d+ neue Änderungen? von Eltern/)).toBeVisible();
      await adminPage.goto('/beitraege/aenderungen');
      await expect(adminPage.getByText(/Telefon geändert: 030 11111 → 030 22222/)).toBeVisible();
      await expect(adminPage.getByText(/hat bei Max .+ Telefon geändert: 030 33333 → 030 44444/)).toBeVisible();

      await page.getByRole('button', { name: 'Fehler melden' }).first().click();
      const report = page.getByRole('dialog', { name: 'Fehler melden' });
      await report.getByLabel('Nachricht').fill('E2E Bitte Kontaktdaten prüfen');
      await report.getByRole('button', { name: 'Meldung senden' }).click();
      await adminPage.goto('/beitraege/meldungen');
      const reported = adminPage.getByRole('article').filter({ hasText: 'E2E Bitte Kontaktdaten prüfen' });
      await expect(reported).toBeVisible();
      await reported.getByLabel('Antwort an die Eltern (optional)').fill('E2E ist korrigiert');
      await reported.getByRole('button', { name: 'Antworten und erledigen' }).click();
      await expect(reported).toHaveCount(0);
      await adminPage.getByLabel('Status').selectOption('DONE');
      await expect(reported).toContainText('Erledigt');
      await expect(reported).toContainText('E2E ist korrigiert');
      await adminPage.goto('/beitraege/aenderungen');
      await expect(adminPage.getByRole('listitem').filter({ hasText: 'E2E Bitte Kontaktdaten prüfen' }))
        .toContainText('erledigt');
      await page.goto('/beitraege/familie');
      const ownReports = page.locator('section').filter({ hasText: 'Meldungen deiner Familie' });
      await expect(ownReports).toContainText('E2E Bitte Kontaktdaten prüfen');
      await expect(ownReports).toContainText('Antwort vom Vorstand: E2E ist korrigiert');
    } finally { await context.close(); }
  });

test('Beiträge der Familie: Mahngebühr direkt am Beitrag, Filter auf offene', async ({ adminApi, browser }) => {
  const a = await family(adminApi, 'KindFee');
  const { year } = berlinToday();
  const base = await adminApi.post<{ id: string }>('/fees', {
    childId: a.child.id, feeType: 'FOOD', year, month: 1, amount: 45.4, dueDate: `${year}-01-05`,
  });
  await adminApi.post('/fees', {
    childId: a.child.id, feeType: 'FOOD', year, month: 2, amount: 45.4, dueDate: `${year}-02-05`,
  });
  await adminApi.post(`/fees/${base.id}/reminder`);
  const password = `pw-${uniq()}`;
  await adminApi.post('/users', {
    email: a.email, firstName: 'Eva', lastName: 'Eltern', role: 'PARENT', isActive: true, password,
  });
  const context = await browser.newContext();
  try {
    await loginContext(context, a.email, password);
    const page = await context.newPage();
    await page.goto('/beitraege/familie/beitraege');
    await expect(page.getByRole('heading', { name: 'Beiträge', exact: true })).toBeVisible();
    // Other specs may generate fees for the current month, so check the order relative to January.
    const rows = page.locator('tbody tr');
    await expect(rows.filter({ hasText: 'Januar' })).toHaveCount(1);
    const texts = await rows.allTextContents();
    const january = texts.findIndex(t => t.includes('Januar'));
    // Newest due date first; the Mahngebühr follows its January fee although it is due later.
    expect(texts[january - 1]).toContain('Februar');
    expect(texts[january + 1]).toMatch(/Mahngebühr.*10,00/);

    await page.getByRole('button', { name: /Offen:/ }).click();
    await expect(page.getByRole('button', { name: 'Alle anzeigen' })).toBeVisible();
    await expect(rows.filter({ hasText: 'Mahngebühr' })).toHaveCount(1);
    await page.getByRole('button', { name: 'Alle anzeigen' }).click();
    await expect(page.getByRole('button', { name: 'Alle anzeigen' })).toHaveCount(0);
  } finally { await context.close(); }
});
