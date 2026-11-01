import { test, expect, uniq, loginContext } from './fixtures';
import { API } from './env';

test('board member sees the fee area and the read-only hint, but no admin pages', async ({ browser, createUser }) => {
  const user = await createUser('BOARD');
  const context = await browser.newContext();
  try {
    await loginContext(context, user.email, user.password);
    const page = await context.newPage();

    await page.goto('/beitraege/');
    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
    await expect(page.getByTestId('read-only-indicator')).toBeVisible();
    const navigation = page.getByRole('navigation', { name: 'Hauptnavigation' });
    for (const name of ['Kinder', 'Eltern', 'Beiträge', 'Einstufungen', 'Beitragsordnung', 'Übersicht', 'Vorstand', 'Verwaltung']) {
      await expect(navigation.getByRole('link', { name, exact: true })).toBeVisible();
    }
    for (const name of ['Benutzer', 'Erinnerungen', 'Ratenzahlungen']) {
      await expect(navigation.getByRole('link', { name, exact: true })).toHaveCount(0);
    }

    await page.goto('/beitraege/benutzer');
    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
  } finally {
    await context.close();
  }
});

test('board member reads reports, changes and rules but sees no controls that write', async ({ browser, createUser }) => {
  const user = await createUser('BOARD');
  const context = await browser.newContext();
  try {
    await loginContext(context, user.email, user.password);
    const page = await context.newPage();

    await page.goto('/beitraege/meldungen');
    await expect(page).toHaveURL(/\/meldungen$/);
    await page.goto('/beitraege/aenderungen');
    await expect(page).toHaveURL(/\/aenderungen$/);

    await page.goto('/beitraege/kinder');
    await expect(page.getByRole('heading', { name: 'Kinder' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Kind hinzufügen' })).toBeHidden();

    await page.goto('/beitraege/elternstunden/verwaltung');
    await expect(page.getByRole('heading', { name: 'Regeln' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Neue Version' })).toHaveCount(0);
    await expect(page.getByRole('button', { name: 'Vorschau erstellen' })).toHaveCount(0);

    await page.goto('/beitraege/elternstunden/vorstand');
    await expect(page.getByRole('heading', { name: 'Vorstand' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Amtszeit anlegen' })).toBeHidden();

    for (const path of ['einstufungen/neu', 'kinder/import']) {
      await page.goto(`/beitraege/${path}`);
      await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
    }
  } finally {
    await context.close();
  }
});

test('board member can read everything but every write is refused', async ({ browser, createUser }) => {
  const user = await createUser('BOARD');
  const context = await browser.newContext();
  try {
    const token = await loginContext(context, user.email, user.password);
    const headers = { Authorization: `Bearer ${token}` };

    for (const path of ['/children/', '/fees/', '/einstufungen/', '/parent-work/overview', '/notes']) {
      const res = await context.request.get(`${API}${path}`, { headers });
      expect(res.status(), `GET ${path}`).toBe(200);
    }
    const create = await context.request.post(`${API}/children/`, {
      headers, data: { memberNumber: '1', firstName: 'Kind', lastName: uniq(), birthDate: '2021-03-01', entryDate: '2023-08-01' },
    });
    expect(create.status()).toBe(403);
    expect((await create.json()).message).toContain('Leserechte');
    expect((await context.request.post(`${API}/fees/generate`, { headers, data: {} })).status()).toBe(403);
    expect((await context.request.get(`${API}/users/`, { headers })).status()).toBe(403);
    // Calculations store nothing, so the read-only role may run them.
    const income = await context.request.post(`${API}/einstufungen/calculate-income`, { headers, data: {} });
    expect(income.status()).toBe(200);
  } finally {
    await context.close();
  }
});

test('the import role USER cannot be assigned to an account', async ({ adminApi }) => {
  const res = await adminApi.postRaw('/users', {
    email: `user-${uniq()}@e2e.test`, role: 'USER', isActive: true, password: `pw-${uniq()}`,
  });
  expect(res.status()).toBe(400);
});
