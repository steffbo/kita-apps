import { test, expect, uniq } from './fixtures';
import { bankCsv, bankDate, berlinToday, uniqueIban } from './bank';
import { API, IMPORT_TOKEN } from './env';

// The daily banking-sync run (banking-sync/upload.js, sync.js): no login, only
// the X-Import-Token header and a multipart field "file" to /import/upload.

test('banking-sync upload with import token auto-matches a payment', async ({ adminApi, playwright, baseURL }) => {
  const today = berlinToday();
  const memberNumber = String(10000 + (Date.now() % 90000));
  const lastName = `Sync-${uniq()}`;

  const child = await adminApi.post<{ id: string }>('/children', {
    memberNumber,
    firstName: 'Test',
    lastName,
    birthDate: '2020-05-10',
    entryDate: `${today.year - 1}-01-01`,
  });
  const parent = await adminApi.post<{ id: string }>('/parents', { firstName: 'Eva', lastName });
  await adminApi.post(`/children/${child.id}/parents`, { parentId: parent.id, isPrimary: true });
  await adminApi.post('/fees/generate', { year: today.year, month: today.month });

  // Fresh context without cookies or bearer token, like the banking-sync container.
  const sync = await playwright.request.newContext({ baseURL });
  const csv = bankCsv({
    date: bankDate(today),
    payer: `Eva ${lastName}`,
    iban: uniqueIban(),
    purpose: `Essensgeld Test ${lastName} ${memberNumber}`,
    amount: '45,40',
  });
  const upload = (token: string) =>
    sync.post(`${API}/import/upload`, {
      headers: { 'X-Import-Token': token },
      multipart: { file: { name: 'umsaetze.csv', mimeType: 'text/csv', buffer: Buffer.from(csv) } },
    });

  expect((await upload('wrong-token')).status()).toBe(401);

  const res = await upload(IMPORT_TOKEN);
  expect(res.status(), await res.text()).toBe(200);
  const result = await res.json();
  expect(result).toMatchObject({ imported: 1, autoMatched: 1, warnings: 0 });

  // Same file again (banking-sync may re-download overlapping days): skipped, not duplicated.
  const again = await (await upload(IMPORT_TOKEN)).json();
  expect(again).toMatchObject({ imported: 0, autoMatched: 0, skipped: 1 });

  await sync.dispose();
});
