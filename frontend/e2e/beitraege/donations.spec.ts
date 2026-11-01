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
  await expect(section(currentYear).getByRole('cell', { name: recent })).toBeVisible();
  await expect(section(currentYear).getByRole('cell', { name: 'Sommerfest', exact: true })).toBeVisible();
  await expect(section(2024).getByRole('cell', { name: old })).toBeVisible();
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
