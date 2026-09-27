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
      await expect(adminPage.getByText(/eingereichte Meldungen warten/)).toBeVisible();
      await adminPage.getByLabel('Nur mit Meldungen').check();
      const row = adminPage.getByRole('row', { name: new RegExp(a.childName) });
      await expect(row).toContainText('Meldungen');
      await row.getByRole('link').click();
      const entry = adminPage.getByRole('row', { name: /Sommerfest E2E/ });
      await expect(entry).toContainText('Eltern');
      await entry.getByRole('button', { name: 'Bestätigen' }).click();
      await expect(entry).toContainText('Bestätigt');
      await page.reload();
      await expect(page.getByText('Bestätigt', { exact: true })).toBeVisible();
      await expect(page.getByText('2,5 h').first()).toBeVisible();

      await page.goto('/beitraege/familie/daten');
      await page.getByLabel('Telefon').fill('030 22222');
      await page.getByRole('button', { name: 'Kontaktdaten speichern' }).click();
      await expect(page.getByRole('status')).toContainText('gespeichert');
      await adminPage.goto('/beitraege/');
      await expect(adminPage.getByText(/Telefon geändert: 030 11111 → 030 22222/)).toBeVisible();

      await page.getByRole('button', { name: 'Fehler melden' }).first().click();
      const report = page.getByRole('dialog', { name: 'Fehler melden' });
      await report.getByLabel('Nachricht').fill('E2E Bitte Kontaktdaten prüfen');
      await report.getByRole('button', { name: 'Meldung senden' }).click();
      await adminPage.goto('/beitraege/meldungen');
      const reported = adminPage.getByRole('article').filter({ hasText: 'E2E Bitte Kontaktdaten prüfen' });
      await expect(reported).toBeVisible();
      await reported.getByRole('button', { name: 'Erledigt' }).click();
      await expect(reported).toHaveCount(0);
      await adminPage.getByLabel('Status').selectOption('DONE');
      await expect(reported).toContainText('Erledigt');
    } finally { await context.close(); }
  });
