import { test, expect } from '../../fixtures/coverage';
import type { Page } from '@playwright/test';

function expectHasKeys(obj: Record<string, unknown>, keys: string[]) {
  for (const key of keys) {
    expect(obj).toHaveProperty(key);
  }
}

function formatGermanDate(day: number, month: number, year: number): string {
  const dd = String(day).padStart(2, '0');
  const mm = String(month).padStart(2, '0');
  return `${dd}.${mm}.${year}`;
}

function buildBankCsvRow(params: {
  bookingDate: string;
  valueDate: string;
  payerName: string;
  payerIban: string;
  description: string;
  amount: string;
  currency?: string;
}): string {
  const currency = params.currency ?? 'EUR';
  return [
    'Test',
    'DE1234',
    'BIC',
    'Bank',
    params.bookingDate,
    params.valueDate,
    params.payerName,
    params.payerIban,
    'BIC',
    'Transfer',
    params.description,
    params.amount,
    currency,
    '1000,00',
  ].join(';');
}

async function waitForResponseByMethod(page: Page, path: string, method: string, status = 200) {
  return page.waitForResponse(
    response =>
      response.url().includes(path) &&
      response.request().method() === method &&
      response.status() === status,
    { timeout: 15000 }
  );
}

test.describe('Beiträge - Core UX Flows with API Coverage', () => {
  test('create child + parent, generate monthly fees, import CSV, auto-match', async ({ page }) => {
    await page.goto('/beitraege/kinder');
    await expect(page.locator('.animate-spin')).toBeHidden({ timeout: 10000 });

    const memberNumber = String(10000 + Math.floor(Math.random() * 90000));
    const childFirstName = 'Test';
    const childLastName = `Kind-${Date.now().toString().slice(-4)}`;
    const birthDate = '2018-05-10';
    const entryDate = '2024-01-01';

    const createChildResponsePromise = waitForResponseByMethod(page, '/children', 'POST', 201);

    await page.getByRole('button', { name: /kind hinzufügen/i }).click();
    await page.getByLabel(/mitgliedsnummer/i).fill(memberNumber);
    await page.getByLabel(/vorname/i).fill(childFirstName);
    await page.getByLabel(/nachname/i).fill(childLastName);
    await page.getByLabel(/geburtsdatum/i).fill(birthDate);
    await page.getByLabel(/eintrittsdatum/i).fill(entryDate);
    await page.getByRole('button', { name: /speichern/i }).click();

    const createChildResponse = await createChildResponsePromise;
    const child = await createChildResponse.json();

    expectHasKeys(child, [
      'id',
      'memberNumber',
      'firstName',
      'lastName',
      'birthDate',
      'entryDate',
      'isActive',
      'createdAt',
      'updatedAt',
    ]);
    expect(child.memberNumber).toBe(memberNumber);

    const childId = child.id as string;

    await page.goto(`/beitraege/kinder/${childId}`);
    await expect(page.getByRole('heading', { name: new RegExp(childFirstName) })).toBeVisible();

    const parentFirstName = `Eltern-${Date.now().toString().slice(-4)}`;
    const parentLastName = 'Elternteil';
    const parentEmail = `test-${Date.now().toString().slice(-4)}@example.com`;
    const parentPhone = '0123456789';
    const parentStreet = 'Teststraße';
    const parentStreetNo = '42';
    const parentPostal = '12345';
    const parentCity = 'Teststadt';

    const createParentResponsePromise = waitForResponseByMethod(page, '/parents', 'POST', 201);
    const linkParentRequestPromise = page.waitForRequest(req =>
      req.url().includes(`/children/${childId}/parents`) && req.method() === 'POST'
    );

    await page.getByRole('button', { name: /elternteil/i }).click();
    await page.getByLabel(/vorname \*/i).fill(parentFirstName);
    await page.getByLabel(/nachname \*/i).fill(parentLastName);
    await page.getByLabel(/e-mail/i).fill(parentEmail);
    await page.getByLabel(/telefon/i).fill(parentPhone);
    await page.getByLabel(/straße/i).fill(parentStreet);
    await page.getByLabel(/hausnr\./i).fill(parentStreetNo);
    await page.getByLabel(/plz/i).fill(parentPostal);
    await page.getByLabel(/ort/i).fill(parentCity);
    await page.getByRole('button', { name: /anlegen & verknüpfen/i }).click();

    const createParentResponse = await createParentResponsePromise;
    const parent = await createParentResponse.json();

    expectHasKeys(parent, [
      'id',
      'firstName',
      'lastName',
      'email',
      'phone',
      'street',
      'streetNo',
      'postalCode',
      'city',
      'createdAt',
      'updatedAt',
    ]);
    expect(parent.email).toBe(parentEmail);
    expect(parent.phone).toBe(parentPhone);

    const linkRequest = await linkParentRequestPromise;
    const linkPayload = JSON.parse(linkRequest.postData() || '{}');
    expectHasKeys(linkPayload, ['parentId', 'isPrimary']);

    await expect(page.getByText(parentFirstName)).toBeVisible();

    // Generate monthly fees
    await page.goto('/beitraege/beitraege');
    await expect(page.locator('.animate-spin')).toBeHidden({ timeout: 10000 });

    const now = new Date();
    const targetYear = now.getFullYear();
    const targetMonth = now.getMonth() + 1;

    const generateResponsePromise = waitForResponseByMethod(page, '/fees/generate', 'POST', 201);

    await page.getByRole('button', { name: /beiträge generieren/i }).click();
    await page.getByLabel(/jahr/i).selectOption(String(targetYear));
    await page.getByLabel(/monat/i).selectOption(String(targetMonth));
    await page.getByRole('button', { name: /^generieren$/i }).click();

    const generateResponse = await generateResponsePromise;
    const generateResult = await generateResponse.json();

    expectHasKeys(generateResult, ['created', 'skipped']);
    expect(generateResult.created).toBeGreaterThan(0);

    // Import CSV with matching transaction
    await page.goto('/beitraege/import');
    await expect(page.locator('.animate-spin')).toBeHidden({ timeout: 10000 });

    const bookingDate = formatGermanDate(2, targetMonth, targetYear);
    const csvContent = [
      'Bezeichnung Auftragskonto;IBAN Auftragskonto;BIC Auftragskonto;Bankname Auftragskonto;Buchungstag;Valutadatum;Name Zahlungsbeteiligter;IBAN Zahlungsbeteiligter;BIC (SWIFT-Code) Zahlungsbeteiligter;Buchungstext;Verwendungszweck;Betrag;Waehrung;Saldo nach Buchung',
      buildBankCsvRow({
        bookingDate,
        valueDate: bookingDate,
        payerName: 'Test Zahler',
        payerIban: 'DE50500105175432192422',
        description: `Essensgeld ${childFirstName} ${childLastName} ${memberNumber}`,
        amount: '45,40',
      }),
      '',
    ].join('\n');

    const uploadResponsePromise = waitForResponseByMethod(page, '/import/upload', 'POST', 200);

    await page.setInputFiles('#file-input', {
      name: 'e2e-import.csv',
      mimeType: 'text/csv',
      buffer: Buffer.from(csvContent, 'utf-8'),
    });

    const uploadResponse = await uploadResponsePromise;
    const uploadResult = await uploadResponse.json();

    expectHasKeys(uploadResult, ['batchId', 'fileName', 'totalRows', 'imported', 'skipped', 'warnings', 'blacklisted']);
    expect(uploadResult.imported).toBeGreaterThanOrEqual(1);

    await page.goto(`/beitraege/kinder/${childId}`);

    // Fetch fees directly to avoid race with cached UI data
    const accessToken = await page.evaluate(() => localStorage.getItem('fees_access_token'));
    expect(accessToken).toBeTruthy();
    let matchedFee: any | undefined;
    let feesPayload: any | undefined;
    for (let attempt = 0; attempt < 10; attempt += 1) {
      const feesResponse = await page.request.get(`/api/fees/v1/fees?childId=${childId}&page=1&perPage=50`, {
        headers: {
          Authorization: `Bearer ${accessToken}`,
        },
      });
      feesPayload = await feesResponse.json();
      matchedFee = feesPayload?.data?.find((fee: any) => fee.feeType === 'FOOD');
      if (matchedFee?.matchedAmount !== undefined) {
        break;
      }
      await page.waitForTimeout(500);
    }

    expectHasKeys(feesPayload, ['data', 'total', 'page', 'perPage', 'totalPages']);
    expect(Array.isArray(feesPayload.data)).toBe(true);
    expect(matchedFee).toBeTruthy();
    expect(matchedFee.matchedAmount).toBeDefined();
    expect(matchedFee.isPaid).toBe(true);

    const monthLabel = new Date(2000, targetMonth - 1, 1).toLocaleString('de-DE', { month: 'long' });
    await expect(page.getByRole('heading', { name: /bezahlte beiträge/i })).toBeVisible();
    await expect(page.getByText(new RegExp(`Essensgeld.*${monthLabel}.*${targetYear}`, 'i'))).toBeVisible();
  });
});
