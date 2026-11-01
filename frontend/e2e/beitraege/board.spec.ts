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
    for (const name of ['Kinder', 'Eltern', 'Beiträge', 'Einstufungen', 'Beitragsordnung', 'Übersicht']) {
      await expect(navigation.getByRole('link', { name, exact: true })).toBeVisible();
    }
    for (const name of ['Benutzer', 'Erinnerungen', 'Verwaltung']) {
      await expect(navigation.getByRole('link', { name, exact: true })).toHaveCount(0);
    }

    await page.goto('/beitraege/benutzer');
    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
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
