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
      await form.getByLabel('Stunden').fill('2.5');
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
      await other.getByLabel('Hausnummer').fill('7a');
      await other.getByRole('button', { name: 'Kontaktdaten speichern' }).click();
      await expect(page.getByRole('status')).toContainText('Kontaktdaten von Max');
      await adminPage.goto('/beitraege/');
      await expect(adminPage.getByText(/Telefon geändert: 030 11111 → 030 22222/)).toBeVisible();
      await expect(adminPage.getByText(/hat bei Max .+ Hausnummer geändert: — → 7a/)).toBeVisible();

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
      await adminPage.goto('/beitraege/');
      await expect(adminPage.getByRole('listitem').filter({ hasText: 'E2E Bitte Kontaktdaten prüfen' }))
        .toContainText('erledigt');
      await page.goto('/beitraege/familie');
      const ownReports = page.locator('section').filter({ hasText: 'Meldungen deiner Familie' });
      await expect(ownReports).toContainText('E2E Bitte Kontaktdaten prüfen');
      await expect(ownReports).toContainText('Antwort vom Vorstand: E2E ist korrigiert');
    } finally { await context.close(); }
  });
