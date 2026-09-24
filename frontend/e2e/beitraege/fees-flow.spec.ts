import { test, expect, uniq } from './fixtures';
import { bankCsv, bankDate, berlinToday, uniqueIban } from './bank';

test('child + parent, generate monthly fees, bank import auto-matches the food fee', async ({ adminPage: page }) => {
  const { year, month, day } = berlinToday();
  const memberNumber = String(10000 + (Date.now() % 90000));
  const lastName = `Kind-${uniq()}`;

  // Create the child.
  await page.goto('/beitraege/kinder');
  await page.getByRole('button', { name: 'Kind hinzufügen' }).click();
  await page.getByLabel('Mitgliedsnummer *').fill(memberNumber);
  await page.getByLabel('Vorname *').fill('Test');
  await page.getByLabel('Nachname *').fill(lastName);
  await page.getByLabel('Geburtsdatum *').fill('2020-05-10');
  await page.getByLabel('Eintrittsdatum *').fill(`${year - 1}-01-01`);
  const created = page.waitForResponse((r) => r.url().endsWith('/api/fees/v1/children') && r.request().method() === 'POST');
  await page.getByRole('button', { name: 'Speichern' }).click();
  const childRes = await created;
  expect(childRes.status()).toBe(201);
  const childId = (await childRes.json()).id as string;

  // Add a parent; this creates the household.
  await page.goto(`/beitraege/kinder/${childId}`);
  await expect(page.getByRole('heading', { name: new RegExp(lastName) })).toBeVisible();
  await page.getByRole('button', { name: 'Elternteil', exact: true }).click();
  await page.getByLabel('Vorname *').fill('Eva');
  await page.getByLabel('Nachname *').fill(lastName);
  await page.getByLabel('E-Mail').fill(`eltern-${uniq()}@e2e.test`);
  await page.getByRole('button', { name: 'Anlegen & Verknüpfen' }).click();
  await expect(page.getByText(`Eva ${lastName}`).first()).toBeVisible();

  // Generate fees for the current month.
  await page.goto('/beitraege/beitraege');
  await page.getByRole('button', { name: 'Beiträge generieren' }).click();
  await page.getByLabel('Jahr').selectOption(String(year));
  await page.getByLabel('Monat').selectOption(String(month));
  const generated = page.waitForResponse((r) => r.url().includes('/fees/generate') && r.request().method() === 'POST');
  await page.getByRole('button', { name: 'Generieren', exact: true }).click();
  expect((await generated).status()).toBe(201);

  // Upload a bank CSV paying the food fee; the member number in the purpose matches the child.
  await page.goto('/beitraege/import');
  await page.getByRole('button', { name: 'CSV hochladen' }).click();
  const date = bankDate({ year, month, day });
  const uploaded = page.waitForResponse((r) => r.url().includes('/import/upload'));
  await page.setInputFiles('#file-input', {
    name: 'e2e-bank.csv',
    mimeType: 'text/csv',
    buffer: Buffer.from(
      bankCsv({
        date,
        payer: `Eva ${lastName}`,
        iban: uniqueIban(),
        purpose: `Essensgeld Test ${lastName} ${memberNumber}`,
        amount: '45,40',
      }),
    ),
  });
  const upload = await uploaded;
  expect(upload.status()).toBe(200);
  expect((await upload.json()).imported).toBe(1);

  // The child's account shows the food fee as paid.
  await page.goto(`/beitraege/kinder/${childId}`);
  await expect(page.getByText('Bezahlte Beiträge (1)')).toBeVisible();
});
