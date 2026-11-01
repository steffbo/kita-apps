import { test, expect, uniq, loginContext, type Api } from './fixtures';
import { berlinToday, uniqueIban, uploadPayment } from './bank';
import { API } from './env';

/** A plan with one overdue rate. */
async function seedOverduePlan(adminApi: Api, emails: string[] = []) {
  const { year, month } = berlinToday();
  const first = new Date(Date.UTC(year, month - 3, 15)).toISOString().slice(0, 10);
  const debtorName = `Schuldner ${uniq()}`;
  const detail = await adminApi.post<{ plan: { id: string } }>('/installment-plans', {
    debtorName, emails, totalAmount: 900, installmentAmount: 300, firstDueDate: first,
  });
  return { id: detail.plan.id, debtorName };
}

test('a bank payment from an unknown payer becomes the first payment of a new plan', async ({ adminPage: page, playwright, baseURL }) => {
  const lastName = `Raten${uniq()}`;
  await uploadPayment(playwright, baseURL, {
    payer: `${lastName}, Daniel`, iban: uniqueIban(), purpose: 'Ratenzahlung nach 10 C 644/24', amount: '300,00',
  });

  await page.goto('/beitraege/bankabgleich');
  await page.getByPlaceholder('Zahler oder Beschreibung').fill(lastName);
  await expect(page.getByRole('button', { name: 'Zuordnen', exact: true })).toHaveCount(1);
  await page.getByRole('button', { name: 'Zuordnen', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: 'Transaktion manuell zuordnen' });
  await dialog.getByRole('button', { name: 'Ratenzahlung', exact: true }).click();
  await dialog.getByRole('button', { name: /Neue Ratenzahlung anlegen/ }).click();

  const form = page.getByRole('dialog', { name: 'Neue Ratenzahlung' });
  await expect(form.getByLabel('Name *')).toHaveValue(`Daniel ${lastName}`);
  await expect(form.getByLabel('Ratenhöhe *')).toHaveValue('300');
  await expect(form.getByLabel(/Bezug/)).toHaveValue('Az. 10 C 644/24');
  await form.getByLabel('Gesamtbetrag *').fill('2400');
  await expect(form.getByText(/8 Raten à 300,00/)).toBeVisible();
  await form.getByRole('button', { name: 'Anlegen & zuordnen' }).click();
  await expect(form).toBeHidden();

  await page.goto('/beitraege/ratenzahlungen');
  await page.getByRole('link', { name: new RegExp(lastName) }).click();
  await expect(page.getByRole('heading', { name: `Daniel ${lastName}` })).toBeVisible();
  await expect(page.getByText('2.100,00 €').first()).toBeVisible();
  await expect(page.getByText(/1\/8 Raten/)).toBeVisible();
  await expect(page.getByText('Noch kein unterschriebener Vertrag hinterlegt.')).toBeVisible();

  // The contract can be uploaded and the rest settled in one final rate.
  await page.locator('input[type=file]').setInputFiles({
    name: 'vertrag.pdf', mimeType: 'application/pdf', buffer: Buffer.from('%PDF-1.4 e2e'),
  });
  await expect(page.getByText('vertrag.pdf')).toBeVisible();
  await page.getByRole('button', { name: 'Schlussrate', exact: true }).click();
  await page.getByRole('dialog', { name: 'Schlussrate vereinbaren' }).getByRole('button', { name: 'Festlegen' }).click();
  await expect(page.getByRole('button', { name: 'Schlussrate aufheben' })).toBeVisible();
  await expect(page.getByText(/Schlussrate · 2\.100,00/)).toBeVisible();
});

test('an IBAN known to a plan is suggested and assigned as a rate', async ({ adminPage: page, adminApi, playwright, baseURL }) => {
  const iban = uniqueIban();
  const lastName = `Vorschlag${uniq()}`;
  const detail = await adminApi.post<{ plan: { id: string } }>('/installment-plans', {
    debtorName: lastName, totalAmount: 600, installmentAmount: 200, firstDueDate: '2030-01-15', payerIban: iban,
  });
  await uploadPayment(playwright, baseURL, { payer: lastName, iban, purpose: 'Rate', amount: '200,00' });

  await page.goto('/beitraege/bankabgleich');
  await page.getByPlaceholder('Zahler oder Beschreibung').fill(lastName);
  await expect(page.getByRole('button', { name: 'Zuordnen', exact: true })).toHaveCount(1);
  await expect(page.getByText(`Rate · ${lastName}`)).toBeVisible();
  await page.getByRole('button', { name: 'Zuordnen', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: 'Transaktion manuell zuordnen' });
  await expect(dialog.getByText(`Vorschlag über IBAN: ${lastName}`)).toBeVisible();
  await dialog.getByRole('button', { name: 'Zuordnen', exact: true }).click();
  await expect(dialog).toBeHidden();

  await page.goto(`/beitraege/ratenzahlungen/${detail.plan.id}`);
  await expect(page.getByText('400,00 €').first()).toBeVisible();
});

test('overdue plans appear in Erinnerungen; without an address sending stays locked until one is added', async ({ adminPage: page, adminApi }) => {
  const { debtorName } = await seedOverduePlan(adminApi);

  await page.goto('/beitraege/automatisierung');
  const row = page.getByRole('button', { name: new RegExp(debtorName) });
  await expect(row.getByText('Ratenzahlung')).toBeVisible();
  await expect(row.getByText('Keine E-Mail')).toBeVisible();
  await row.click();

  const dialog = page.getByRole('dialog', { name: new RegExp(debtorName) });
  await expect(dialog.getByLabel('Betreff')).toHaveValue('Zahlungserinnerung – Ratenzahlung');
  await expect(dialog.getByLabel('Text')).toHaveValue(new RegExp(`Guten Tag ${debtorName}`));
  await expect(dialog.getByRole('button', { name: 'Senden' })).toBeDisabled();

  await dialog.getByRole('button', { name: 'Ergänzen' }).click();
  await expect(page.getByRole('heading', { name: debtorName })).toBeVisible();
});

test('staff without installment access neither see plans nor whose payment it is', async ({ browser, adminApi, createUser, playwright, baseURL }) => {
  const iban = uniqueIban();
  const lastName = `Geheim${uniq()}`;
  const detail = await adminApi.post<{ plan: { id: string } }>('/installment-plans', {
    debtorName: lastName, totalAmount: 600, installmentAmount: 200, firstDueDate: '2030-01-15', payerIban: iban,
  });
  await uploadPayment(playwright, baseURL, { payer: 'Zahler', iban, purpose: 'Rate A', amount: '200,00' });
  await uploadPayment(playwright, baseURL, { payer: 'Zahler', iban, purpose: 'Rate B', amount: '100,00' });
  type Listed = { id: string; description: string; installmentPlanSuggestion?: unknown; installmentPayment?: { debtorName: string } };
  const list = (api: Api, status: string) =>
    api.get<{ data: Listed[] }>(`/import/transactions?status=${status}&search=${encodeURIComponent(iban)}`);
  const open = (await list(adminApi, 'offen')).data;
  expect(open.every(tx => tx.installmentPlanSuggestion)).toBeTruthy();
  const assigned = open.find(tx => tx.description === 'Rate A')!;
  await adminApi.post(`/installment-plans/${detail.plan.id}/payments`, { transactionId: assigned.id });

  const user = await createUser('BOARD');
  const context = await browser.newContext();
  try {
    const token = await loginContext(context, user.email, user.password);
    const headers = { Authorization: `Bearer ${token}` };

    const asUser = async (status: string) => {
      const res = await context.request.get(`${API}/import/transactions?status=${status}&search=${encodeURIComponent(iban)}`, { headers });
      expect(res.ok()).toBeTruthy();
      return (await res.json()).data as Listed[];
    };
    const body = JSON.stringify(await asUser('alle'));
    expect(body).not.toContain(lastName);
    expect(body).not.toContain('installmentPlanSuggestion');
    expect((await asUser('zugeordnet')).find(tx => tx.id === assigned.id)?.installmentPayment?.debtorName).toBe('');

    expect((await context.request.get(`${API}/installment-plans`, { headers })).status()).toBe(403);
    const unmatch = await context.request.post(`${API}/import/transactions/${assigned.id}/unmatch`, { headers, data: {} });
    expect(unmatch.status()).toBe(403);

    const page = await context.newPage();
    await page.goto('/beitraege/');
    await expect(page.getByRole('navigation', { name: 'Hauptnavigation' }).getByRole('link', { name: 'Beiträge', exact: true })).toBeVisible();
    await expect(page.getByRole('navigation', { name: 'Hauptnavigation' }).getByRole('link', { name: 'Ratenzahlungen' })).toHaveCount(0);
    await page.goto('/beitraege/ratenzahlungen');
    await expect(page).not.toHaveURL(/ratenzahlungen/);
  } finally {
    await context.close();
  }
});
