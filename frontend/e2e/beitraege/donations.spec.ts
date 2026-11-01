import { test, expect, uniq, loginContext, type Api } from './fixtures';
import { berlinToday, uniqueIban, uploadPayment } from './bank';
import { API } from './env';

type Year = { year: number; total: number; transactions: { payerName: string }[] };
// Playwright normalizes whitespace in the page text, so the no-break space of Intl must be normalized too.
const money = (value: number) => value.toLocaleString('de-DE', { style: 'currency', currency: 'EUR' }).replace(/\s/g, ' ');

test('a payment is assigned as donation, listed per year and can be undone', async ({ adminPage: page, adminApi, playwright, baseURL }) => {
  const recent = `Spender${uniq()}`;
  const old = `Spender${uniq()}`;
  await uploadPayment(playwright, baseURL, { payer: recent, iban: uniqueIban(), purpose: 'Spende Sommerfest', amount: '120,50' });
  await uploadPayment(playwright, baseURL, {
    payer: old, iban: uniqueIban(), purpose: 'Spende Weihnachten', amount: '80,00', date: { year: 2024, month: 12, day: 20 },
  });

  for (const [payer, note] of [[recent, 'Sommerfest'], [old, '']] as const) {
    await page.goto('/beitraege/bankabgleich');
    await page.getByPlaceholder('Zahler oder Beschreibung').fill(payer);
    await expect(page.getByRole('button', { name: 'Zuordnen', exact: true })).toHaveCount(1);
    await page.getByRole('button', { name: 'Zuordnen', exact: true }).click();
    const dialog = page.getByRole('dialog', { name: 'Transaktion manuell zuordnen' });
    await dialog.getByRole('button', { name: 'Spende', exact: true }).click();
    if (note) await dialog.getByLabel('Notiz (optional)').fill(note);
    await dialog.getByRole('button', { name: /Als Spende zuordnen/ }).click();
    await expect(dialog).toBeHidden();
    // No longer open; shown as assigned with the badge.
    await expect(page.getByRole('button', { name: 'Zuordnen', exact: true })).toHaveCount(0);
  }

  await page.goto('/beitraege/spenden');
  const { years } = await adminApi.get<{ years: Year[] }>('/import/donations');
  const currentYear = berlinToday().year;
  const section = (year: number) => page.getByRole('region', { name: `Spenden ${year}` });
  await expect(section(currentYear).getByRole('cell', { name: recent, exact: true })).toBeVisible();
  await expect(section(currentYear).getByRole('cell', { name: 'Sommerfest', exact: true })).toBeVisible();
  await expect(section(2024).getByRole('cell', { name: old, exact: true })).toBeVisible();
  for (const year of [currentYear, 2024]) {
    await expect(section(year).getByText(money(years.find(y => y.year === year)!.total), { exact: true }).first()).toBeVisible(); // year header
  }
  // Newest year first.
  const headings = await page.getByRole('heading', { level: 2 }).allTextContents();
  expect(headings.indexOf(String(currentYear))).toBeLessThan(headings.indexOf('2024'));

  // Undo from the Bankabgleich.
  await page.goto('/beitraege/bankabgleich');
  await page.getByPlaceholder('Zahler oder Beschreibung').fill(old);
  await page.getByRole('button', { name: /^Zugeordnet/ }).click();
  await expect(page.locator('.badge:visible', { hasText: 'Spende' })).toHaveCount(1);
  await page.getByRole('button', { name: 'Aufheben' }).click();
  await page.getByRole('button', { name: 'Ja', exact: true }).click();
  await page.getByRole('button', { name: /^Offen/ }).click();
  await expect(page.getByRole('button', { name: 'Zuordnen', exact: true })).toHaveCount(1);
  await page.goto('/beitraege/spenden');
  await expect(page.getByText(old)).toHaveCount(0);
});

test('USER may assign donations, LEITUNG has no access', async ({ browser, createUser, playwright, baseURL }) => {
  const iban = uniqueIban();
  await uploadPayment(playwright, baseURL, { payer: `Spender${uniq()}`, iban, purpose: 'Spende', amount: '25,00' });
  for (const role of ['USER', 'LEITUNG'] as const) {
    const user = await createUser(role);
    const context = await browser.newContext();
    try {
      const headers = { Authorization: `Bearer ${await loginContext(context, user.email, user.password)}` };
      const list = await context.request.get(`${API}/import/transactions?status=offen&search=${encodeURIComponent(iban)}`, { headers });
      if (role === 'LEITUNG') {
        expect(list.status()).toBe(403);
        expect((await context.request.get(`${API}/import/donations`, { headers })).status()).toBe(403);
        continue;
      }
      const [tx] = (await list.json()).data as { id: string }[];
      const assign = await context.request.post(`${API}/import/transactions/${tx.id}/assign`, { headers, data: { kind: 'DONATION', note: '' } });
      expect(assign.status()).toBe(204);
      const page = await context.newPage();
      await page.goto('/beitraege/spenden');
      await expect(page.getByRole('heading', { name: 'Spenden', level: 1 })).toBeVisible();
      await expect(page.getByRole('navigation', { name: 'Hauptnavigation' }).getByRole('link', { name: 'Spenden' })).toBeVisible();
    } finally {
      await context.close();
    }
  }
});

test('a donation is credited to a household; the next one from the same IBAN suggests it; editable later', async ({ adminPage: page, adminApi, playwright, baseURL }) => {
  const name = `Familie Spende ${uniq()}`;
  await adminApi.post('/households', { name, annualHouseholdIncome: 40000, incomeStatus: 'PROVIDED' });
  const iban = uniqueIban();
  const payer = `Oma${uniq()}`;

  const openDonationDialog = async (purpose: string) => {
    await page.goto('/beitraege/bankabgleich');
    await page.getByPlaceholder('Zahler oder Beschreibung').fill(purpose);
    await expect(page.getByRole('button', { name: 'Zuordnen', exact: true })).toHaveCount(1);
    await page.getByRole('button', { name: 'Zuordnen', exact: true }).click();
    const dialog = page.getByRole('dialog', { name: 'Transaktion manuell zuordnen' });
    await dialog.getByRole('button', { name: 'Spende', exact: true }).click();
    return dialog;
  };

  await uploadPayment(playwright, baseURL, { payer, iban, purpose: `Spende erste ${payer}`, amount: '60,00' });
  let dialog = await openDonationDialog(`erste ${payer}`);
  await expect(dialog.getByText(/Vorschlag über IBAN/)).toHaveCount(0);
  const search = dialog.getByPlaceholder('Familie, Kind oder Mitglied suchen');
  await search.fill(name);
  // Enter in the search must not submit the assignment.
  await search.press('Enter');
  await expect(dialog).toBeVisible();
  await dialog.getByRole('button', { name, exact: true }).click();
  await expect(dialog.getByText(`Familie: ${name}`)).toBeVisible();
  await dialog.getByRole('button', { name: /Als Spende zuordnen/ }).click();
  await expect(dialog).toBeHidden();

  await uploadPayment(playwright, baseURL, { payer, iban, purpose: `Spende zweite ${payer}`, amount: '25,00' });
  dialog = await openDonationDialog(`zweite ${payer}`);
  await expect(dialog.getByText('Vorschlag über IBAN')).toBeVisible();
  await expect(dialog.getByText(`Familie: ${name}`)).toBeVisible();
  await dialog.getByRole('button', { name: /Als Spende zuordnen/ }).click();
  await expect(dialog).toBeHidden();

  await page.goto('/beitraege/spenden');
  await expect(page.getByRole('cell', { name, exact: true })).toHaveCount(2);

  // Remove the household from one donation afterwards.
  await page.getByRole('button', { name: `Spende von ${payer} bearbeiten` }).first().click();
  const edit = page.getByRole('dialog', { name: 'Spende bearbeiten' });
  await edit.getByLabel('Notiz (optional)').fill('Weihnachten');
  await edit.getByRole('button', { name: 'Haushalt entfernen' }).click();
  await edit.getByRole('button', { name: 'Speichern' }).click();
  await expect(edit).toBeHidden();
  await expect(page.getByRole('cell', { name, exact: true })).toHaveCount(1);
  await expect(page.getByRole('cell', { name: 'Weihnachten', exact: true })).toBeVisible();
});

test('parents see the donations of their household without bank details; staff see them on the household', async ({ adminApi, adminPage, browser, playwright, baseURL }) => {
  // A family whose household is created by linking the parent to the child.
  const family = async (label: string) => {
    const suffix = uniq();
    const today = berlinToday();
    const email = `spende-${suffix}@e2e.test`;
    const child = await adminApi.post<{ id: string }>('/children', {
      memberNumber: String(10_000_000 + Math.floor(Math.random() * 90_000_000)), firstName: label, lastName: suffix,
      birthDate: `${today.year - 5}-01-01`, entryDate: `${today.year - 2}-01-01`,
    });
    const parent = await adminApi.post<{ id: string }>('/parents', { firstName: 'Eva', lastName: suffix, email });
    await adminApi.post(`/children/${child.id}/parents`, { parentId: parent.id, isPrimary: true });
    const { householdId } = await adminApi.get<{ householdId: string }>(`/children/${child.id}`);
    return { childId: child.id, email, householdId };
  };
  const donate = async (payer: string, householdId: string, note: string) => {
    const iban = uniqueIban();
    await uploadPayment(playwright, baseURL, { payer, iban, purpose: 'Geheimer Zweck', amount: '75,00' });
    const { data } = await adminApi.get<{ data: { id: string }[] }>(`/import/transactions?status=offen&search=${iban}`);
    await adminApi.post(`/import/transactions/${data[0].id}/assign`, { kind: 'DONATION', note, householdId });
    return iban;
  };

  const own = await family('KindSpende');
  const other = await family('KindAnders');
  const ownPayer = `Oma${uniq()}`;
  const otherPayer = `Opa${uniq()}`;
  const iban = await donate(ownPayer, own.householdId, 'Interne Notiz');
  await donate(otherPayer, other.householdId, '');

  const password = `pw-${uniq()}`;
  await adminApi.post('/users', { email: own.email, firstName: 'Eva', lastName: 'Eltern', role: 'PARENT', isActive: true, password });
  const context = await browser.newContext();
  try {
    const token = await loginContext(context, own.email, password);
    const res = await context.request.get(`${API}/me/donations`, { headers: { Authorization: `Bearer ${token}` } });
    expect(res.ok()).toBeTruthy();
    const body = await res.text();
    expect(body).toContain(ownPayer);
    for (const secret of [otherPayer, iban, 'Interne Notiz', 'Geheimer Zweck']) expect(body).not.toContain(secret);

    const page = await context.newPage();
    await page.goto('/beitraege/familie/beitraege');
    const section = page.getByRole('region', { name: 'Spenden', exact: true });
    await expect(section.getByText(ownPayer)).toBeVisible();
    await expect(section.getByText('75,00 €').first()).toBeVisible();
    await expect(page.getByText(otherPayer)).toHaveCount(0);
    await expect(page.getByText('Interne Notiz')).toHaveCount(0);
  } finally {
    await context.close();
  }

  // Staff: on the child's household and on the household's Elternstunden page.
  await adminPage.goto(`/beitraege/kinder/${own.childId}`);
  const card = adminPage.getByRole('region', { name: 'Spenden des Haushalts' });
  await expect(card.getByText(ownPayer)).toBeVisible();
  await expect(card.getByText(otherPayer)).toHaveCount(0);
  await adminPage.goto(`/beitraege/elternstunden/familien/${own.householdId}`);
  await expect(adminPage.getByRole('region', { name: 'Spenden des Haushalts' }).getByText(ownPayer)).toBeVisible();
});
