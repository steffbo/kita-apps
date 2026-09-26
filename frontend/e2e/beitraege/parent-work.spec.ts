import { test, expect, uniq, loginContext, type Api } from './fixtures';
import { berlinToday, bankDate } from './bank';
import { API } from './env';

/** Unique member number within the varchar(10) column. */
function memberNumber() {
  return String(10_000_000 + Math.floor(Math.random() * 90_000_000));
}

function workYear() {
  const today = berlinToday();
  return { today, year: today.month >= 8 ? today.year : today.year - 1 };
}

async function familyWithChild(adminApi: Api) {
  const suffix = uniq();
  const householdName = `Familie Elternstunden ${suffix}`;
  const childName = `Kind ${suffix}`;
  const household = await adminApi.post<{ id: string }>('/households', { name: householdName });
  const { today, year } = workYear();
  const child = await adminApi.post<{ id: string }>('/children', {
    memberNumber: memberNumber(), firstName: 'Kind', lastName: suffix,
    birthDate: `${year - 5}-01-01`, entryDate: `${year - 2}-01-01`,
  });
  await adminApi.post(`/households/${household.id}/children`, { childId: child.id });
  return { household, householdName, childName, today, year, suffix };
}

test('parent work role sees only its area and cannot fetch children', async ({ browser, createUser }) => {
  const user = await createUser('PARENT_WORK');
  const context = await browser.newContext();
  try {
    const token = await loginContext(context, user.email, user.password);
    const page = await context.newPage();
    await page.goto('/beitraege/');
    await expect(page).toHaveURL(/\/beitraege\/elternstunden$/);
    await expect(page.getByRole('heading', { name: 'Elternstunden', exact: true })).toBeVisible();
    const navigation = page.getByRole('navigation');
    await expect(navigation.getByText('Elternstunden', { exact: true })).toBeVisible();
    for (const group of ['Täglich', 'Verwaltung', 'Beiträge', 'System']) {
      await expect(navigation.getByText(group, { exact: true })).toHaveCount(0);
    }
    await expect(navigation.getByRole('link', { name: 'Vorstand' })).toBeVisible();
    await page.goto('/beitraege/kinder');
    await expect(page).toHaveURL(/\/beitraege\/elternstunden$/);
    const response = await context.request.get(`${API}/children`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(response.status()).toBe(403);
  } finally { await context.close(); }
});

test('work entry reduces the open account and voiding restores it', async ({ adminApi, adminPage: page }) => {
  const family = await familyWithChild(adminApi);
  await page.goto('/beitraege/elternstunden');
  await page.getByLabel('Suche nach Familie oder Kind').fill(family.suffix);
  const row = page.getByRole('row', { name: new RegExp(family.householdName) });
  await expect(row).toContainText('9 h');
  await row.getByRole('link', { name: family.householdName }).click();
  const account = page.getByRole('heading', { name: 'Konto' }).locator('..');
  const open = account.getByText('Offen', { exact: true }).locator('..');
  const missing = account.getByText('Fehlbetrag', { exact: true }).locator('..');
  await expect(open).toContainText('9 h');
  await page.getByRole('button', { name: 'Stunden erfassen' }).click();
  const dialog = page.getByRole('dialog', { name: 'Stunden erfassen' });
  await expect(dialog).toContainText(`Familie: ${family.householdName}`);
  await expect(dialog.getByLabel('Stunden *')).toHaveValue('');
  await dialog.getByLabel('Stunden *').fill('2.5');
  await dialog.getByLabel('Anlass *').fill('E2E Gartenarbeit');
  await dialog.getByRole('button', { name: 'Speichern' }).click();
  await expect(dialog).toBeHidden();
  await expect(open).toContainText('6,5 h');
  await expect(missing).toContainText('195,00 €');
  const entry = page.getByRole('row', { name: /E2E Gartenarbeit/ });
  await entry.getByRole('button', { name: 'Stornieren' }).click();
  const voidDialog = page.getByRole('dialog', { name: 'Eintrag stornieren' });
  await voidDialog.getByLabel('Grund *').fill('E2E Korrektur');
  await voidDialog.getByRole('button', { name: 'Stornieren' }).click();
  await expect(open).toContainText('9 h');
  await expect(entry).toContainText('Storniert');
});

test('board term exempts the family', async ({ adminApi, adminPage: page }) => {
  const family = await familyWithChild(adminApi);
  const member = await adminApi.post<{ id: string }>('/members', {
    memberNumber: memberNumber(), firstName: 'Mira', lastName: family.suffix,
    membershipStart: `${family.year - 2}-01-01`, householdId: family.household.id,
  });
  await page.goto('/beitraege/elternstunden/vorstand');
  await page.getByRole('button', { name: 'Amtszeit anlegen' }).click();
  const dialog = page.getByRole('dialog', { name: 'Amtszeit anlegen' });
  await dialog.getByLabel('Mitglied suchen').fill(family.suffix);
  await dialog.getByLabel('Vereinsmitglied *').selectOption(member.id);
  await dialog.getByLabel('Amt *').fill('Vorsitz');
  await dialog.getByLabel('Von *').fill(`${family.year}-08-01`);
  await dialog.getByRole('button', { name: 'Speichern' }).click();
  await expect(dialog).toBeHidden();
  await page.goto('/beitraege/elternstunden');
  await page.getByLabel('Suche nach Familie oder Kind').fill(family.suffix);
  const row = page.getByRole('row', { name: new RegExp(family.householdName) });
  await expect(row).toContainText('befreit');
  await expect(row).toContainText('0 h');
});

test('CSV preview matches child and imports an entry', async ({ adminApi, adminPage: page }) => {
  const family = await familyWithChild(adminApi);
  const csv = [
    'Name des Mitgliedes;Name des Kindes;Wann;Anlass;Stunden',
    `;${family.childName};${bankDate(family.today)};E2E Aufräumen;1,5`,
  ].join('\n');
  await page.goto('/beitraege/elternstunden/import');
  await page.getByLabel('CSV-Datei').setInputFiles({
    name: 'elternstunden.csv', mimeType: 'text/csv', buffer: Buffer.from(csv),
  });
  await expect(page.getByText('1 Zeile erkannt.')).toBeVisible();
  await page.getByRole('button', { name: 'Vorschau erstellen' }).click();
  const row = page.getByRole('row', { name: /E2E Aufräumen/ });
  await expect(row).toContainText(family.householdName);
  await expect(row).toContainText('Treffer über Kind');
  await expect(row).toContainText('1,5 h');
  await expect(row.getByRole('checkbox')).toBeChecked();
  await page.getByRole('button', { name: '1 Eintrag importieren' }).click();
  await expect(page.getByRole('status')).toContainText('1 Eintrag wurde importiert.');
  await page.goto(`/beitraege/elternstunden/familien/${family.household.id}?jahr=${family.year}`);
  await expect(page.getByRole('row', { name: /E2E Aufräumen/ })).toContainText('Import');
});
