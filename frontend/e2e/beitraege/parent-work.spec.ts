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
    await navigation.getByRole('button', { name: 'Elternstunden' }).click();
    await expect(navigation.getByRole('link', { name: 'Verwaltung' })).toBeVisible();
    await expect(navigation.getByRole('link', { name: 'Vorstand' })).toHaveCount(0);
    await page.goto('/beitraege/elternstunden/vorstand');
    await expect(page).toHaveURL(/\/beitraege\/elternstunden$/);
    const terms = await context.request.get(`${API}/parent-work/board-terms`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(terms.status()).toBe(403);
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
  await expect(dialog.getByText('Noch keine Dauer gewählt.')).toBeVisible();
  await dialog.getByRole('group', { name: 'Stunden' }).getByRole('button', { name: '2', exact: true }).click();
  await dialog.getByRole('group', { name: 'Minuten' }).getByRole('button', { name: '30', exact: true }).click();
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
  await page.goto('/beitraege/elternstunden/verwaltung');
  await page.getByLabel('CSV-Datei').setInputFiles({
    name: 'elternstunden.csv', mimeType: 'text/csv', buffer: Buffer.from(csv),
  });
  await expect(page.getByText('1 Zeile erkannt.')).toBeVisible();
  await page.getByRole('button', { name: 'Vorschau erstellen' }).click();
  const row = page.getByRole('table', { name: 'Vorschau' }).getByRole('row', { name: /E2E Aufräumen/ });
  await expect(row).toContainText(family.householdName);
  await expect(row).toContainText('Treffer über Kind');
  await expect(row).toContainText('1,5 h');
  await expect(row.getByRole('checkbox')).toBeChecked();
  await page.getByRole('button', { name: '1 Eintrag importieren' }).click();
  await expect(page.getByRole('status')).toContainText('1 Eintrag wurde importiert.');
  await page.goto(`/beitraege/elternstunden/familien/${family.household.id}?jahr=${family.year}`);
  await expect(page.getByRole('row', { name: /E2E Aufräumen/ })).toContainText('Import');
});

test('CSV import: custom headers, ignored columns, drag and drop, warning for unmatched families', async ({ adminApi, adminPage: page }) => {
  const family = await familyWithChild(adminApi);
  const date = bankDate(family.today);
  const csv = [
    'Lfd. Nr.;Wer;Was;Datum der Arbeit;Dauer in h;Bemerkung',
    `1;${family.childName.toUpperCase()};Beet gejätet;${date};2;egal`,
    `2;Niemand Unbekannt;Fenster geputzt;${date};1;egal`,
  ].join('\n');
  await page.goto('/beitraege/elternstunden/verwaltung');
  await expect(page.getByText('Erwartete Spalten')).toBeVisible();
  await expect(page.getByText('über den Namen, zuerst das Kind')).toBeVisible();

  await page.evaluate((content) => {
    const data = new DataTransfer();
    data.items.add(new File([content], 'drop.csv', { type: 'text/csv' }));
    window.dispatchEvent(new DragEvent('dragenter', { dataTransfer: data, bubbles: true, cancelable: true }));
    window.dispatchEvent(new DragEvent('drop', { dataTransfer: data, bubbles: true, cancelable: true }));
  }, csv);
  await expect(page.getByText('drop.csv')).toBeVisible();
  await expect(page.getByText('2 Zeilen erkannt.')).toBeVisible();

  const preview = page.getByRole('button', { name: 'Vorschau erstellen' });
  await expect(page.getByLabel('Zuordnung für Spalte Datum der Arbeit')).toHaveValue('workDate');
  await expect(page.getByLabel('Zuordnung für Spalte Dauer in h')).toHaveValue('hours');
  await expect(page.getByText('Es fehlt noch: Anlass, Kind oder Mitglied.')).toBeVisible();
  await expect(preview).toBeDisabled();
  await page.getByLabel('Zuordnung für Spalte Wer').selectOption('childName');
  await page.getByLabel('Zuordnung für Spalte Was').selectOption('occasion');
  await expect(page.getByText('(2 von 6 ignoriert)')).toBeVisible();
  await preview.click();

  const warning = page.getByRole('alert').filter({ hasText: '1 Zeile ohne Familie' });
  await expect(warning).toContainText('Niemand Unbekannt');
  await expect(warning).toContainText('Zeile 2');
  const previewTable = page.getByRole('table', { name: 'Vorschau' });
  const matched = previewTable.getByRole('row', { name: /Beet gejätet/ });
  await expect(matched).toContainText(family.householdName);
  await expect(matched.getByRole('checkbox')).toBeChecked();
  await expect(previewTable.getByRole('row', { name: /Fenster geputzt/ }).getByRole('checkbox')).toBeDisabled();
  await page.getByRole('button', { name: '1 Eintrag importieren' }).click();
  await expect(page.getByRole('status')).toContainText('1 Eintrag wurde importiert.');
});

async function addEntry(adminApi: Api, householdId: string, minutes: number, occasion: string) {
  const { today } = workYear();
  const workDate = `${today.year}-${String(today.month).padStart(2, '0')}-${String(today.day).padStart(2, '0')}`;
  await adminApi.post('/parent-work/entries', { householdId, workDate, durationMinutes: minutes, occasion });
}

test('overview: row button opens the entry form with the family preselected', async ({ adminApi, adminPage: page }) => {
  const family = await familyWithChild(adminApi);
  await page.goto('/beitraege/elternstunden');
  await expect(page.getByRole('columnheader', { name: /Fehlbetrag/ })).toHaveCount(0);
  await page.getByLabel('Suche nach Familie oder Kind').fill(family.suffix);
  const row = page.getByRole('row', { name: new RegExp(family.householdName) });
  await row.getByRole('button', { name: `Stunden erfassen für ${family.householdName}` }).click();
  const dialog = page.getByRole('dialog', { name: 'Stunden erfassen' });
  await expect(dialog).toContainText(`Familie: ${family.householdName}`);
  await expect(dialog.getByLabel('Mitglied')).toHaveCount(0);
  await expect(dialog.getByLabel('Kind')).toHaveCount(0);
  await dialog.getByRole('group', { name: 'Stunden' }).getByRole('button', { name: '2', exact: true }).click();
  await dialog.getByLabel('Anlass *').fill('E2E Zeilenbutton');
  await dialog.getByRole('button', { name: 'Speichern' }).click();
  await expect(dialog).toBeHidden();
  await expect(row).toContainText('7 h');
});

test('overview: cards filter the table', async ({ adminApi, adminPage: page }) => {
  const idle = await familyWithChild(adminApi);
  const active = await familyWithChild(adminApi);
  await addEntry(adminApi, active.household.id, 60, 'E2E Karte');
  await page.goto('/beitraege/elternstunden');
  const idleRow = page.getByRole('row', { name: new RegExp(idle.householdName) });
  const activeRow = page.getByRole('row', { name: new RegExp(active.householdName) });
  const none = page.getByRole('button', { name: /Noch keine Stunden geleistet/ });
  await none.click();
  await expect(none).toHaveAttribute('aria-pressed', 'true');
  await expect(idleRow).toBeVisible();
  await expect(activeRow).toHaveCount(0);
  await page.getByRole('button', { name: /Stunden noch offen/ }).click();
  await expect(idleRow).toBeVisible();
  await expect(activeRow).toBeVisible();
  await page.getByRole('button', { name: /Alle Stunden geleistet/ }).click();
  await expect(idleRow).toHaveCount(0);
  await expect(activeRow).toHaveCount(0);
  await page.getByRole('button', { name: /Alle Stunden geleistet/ }).click();
  await expect(idleRow).toBeVisible();
  await expect(page.getByRole('button', { name: /Unbestätigte Meldungen/ })).toBeVisible();
});

test('overview: table sorts by column', async ({ adminApi, adminPage: page }) => {
  const idle = await familyWithChild(adminApi);
  const active = await familyWithChild(adminApi);
  await addEntry(adminApi, active.household.id, 120, 'E2E Sortierung');
  await page.goto('/beitraege/elternstunden');
  const names = () => page.getByRole('row').allInnerTexts();
  const position = async () => {
    const rows = await names();
    return [rows.findIndex(r => r.includes(active.householdName)), rows.findIndex(r => r.includes(idle.householdName))];
  };
  await expect(page.getByRole('row', { name: new RegExp(active.householdName) })).toBeVisible();
  await page.getByRole('button', { name: 'Offen', exact: true }).click();
  await expect(page.getByRole('columnheader', { name: 'Offen' })).toHaveAttribute('aria-sort', 'ascending');
  let [a, i] = await position();
  expect(a).toBeLessThan(i);
  await page.getByRole('button', { name: 'Offen', exact: true }).click();
  await expect(page.getByRole('columnheader', { name: 'Offen' })).toHaveAttribute('aria-sort', 'descending');
  [a, i] = await position();
  expect(a).toBeGreaterThan(i);
});

test('detail: account cards, no board section without term, no amount for parent work role',
  async ({ adminApi, adminPage: page, browser, createUser }) => {
    const family = await familyWithChild(adminApi);
    const url = `/beitraege/elternstunden/familien/${family.household.id}?jahr=${family.year}`;
    await page.goto(url);
    const account = page.getByRole('heading', { name: 'Konto' }).locator('..');
    for (const label of ['Soll', 'Geleistet', 'Übertrag Vorjahr', 'Offen', 'Übertrag Folgejahr', 'Fehlbetrag']) {
      await expect(account.getByText(label, { exact: true })).toBeVisible();
    }
    await expect(page.getByRole('heading', { name: 'Vorstands-Amtszeiten' })).toHaveCount(0);

    const user = await createUser('PARENT_WORK');
    const context = await browser.newContext();
    try {
      await loginContext(context, user.email, user.password);
      const work = await context.newPage();
      await work.goto(url);
      const workAccount = work.getByRole('heading', { name: 'Konto' }).locator('..');
      await expect(workAccount.getByText('Offen', { exact: true })).toBeVisible();
      await expect(workAccount.getByText('Fehlbetrag', { exact: true })).toHaveCount(0);
    } finally { await context.close(); }
  });

test('verwaltung page explains Tertiale and board exemption', async ({ adminPage: page }) => {
  await page.goto('/beitraege/elternstunden/verwaltung');
  await expect(page.getByRole('heading', { name: 'Verwaltung', exact: true })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Regeln', exact: true })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Import', exact: true })).toBeVisible();
  await expect(page.getByText('Tertiale:')).toBeVisible();
  await expect(page.getByText('Vorstandsbetreuung:')).toBeVisible();
});
