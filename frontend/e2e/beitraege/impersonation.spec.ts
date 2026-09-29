import { test, expect, uniq, type Api } from './fixtures';
import { berlinToday } from './bank';

async function parentUser(adminApi: Api) {
  const suffix = uniq();
  const today = berlinToday();
  const email = `imp-eltern-${suffix}@e2e.test`;
  const child = await adminApi.post<{ id: string }>('/children', {
    memberNumber: String(10_000_000 + Math.floor(Math.random() * 90_000_000)),
    firstName: 'ImpKind', lastName: suffix,
    birthDate: `${today.year - 5}-01-01`, entryDate: `${today.year - 2}-01-01`,
  });
  const parent = await adminApi.post<{ id: string }>('/parents', {
    firstName: 'Eva', lastName: suffix, email, phone: '030 11111',
  });
  await adminApi.post(`/children/${child.id}/parents`, { parentId: parent.id, isPrimary: true });
  await adminApi.post('/users', {
    email, firstName: 'Eva', lastName: suffix, role: 'PARENT', isActive: true, password: `pw-${uniq()}`,
  });
  return { email, childName: `ImpKind ${suffix}` };
}

test('admin impersonates a parent, keeps the mode on reload and returns to the admin session',
  async ({ adminApi, adminPage: page }) => {
    const parent = await parentUser(adminApi);

    await page.goto('/beitraege/benutzer');
    const row = page.getByRole('row', { name: new RegExp(parent.email) });
    await row.getByRole('button', { name: 'Als diesen Benutzer anmelden' }).click();

    const indicator = page.getByTestId('impersonation-indicator');
    await expect(page).toHaveURL(/\/beitraege\/familie$/);
    await expect(indicator).toBeVisible();
    await expect(indicator).toContainText(/Impersonation: .* als Eva /);
    await expect(page.getByText(parent.childName)).toBeVisible();
    await expect(page.getByRole('link', { name: 'Benutzer' })).toHaveCount(0);

    await page.goto('/beitraege/benutzer');
    await expect(page).toHaveURL(/\/beitraege\/familie$/);

    await page.reload();
    await expect(indicator).toBeVisible();
    await expect(page.getByText(parent.childName)).toBeVisible();

    await page.getByRole('button', { name: 'Benutzermenü' }).click();
    await expect(page.getByRole('button', { name: 'Passwort ändern' })).toHaveCount(0);
    await page.getByTestId('stop-impersonation').click();

    await expect(page).toHaveURL(/\/beitraege\/benutzer$/);
    await expect(page.getByRole('heading', { name: 'Benutzer', exact: true })).toBeVisible();
    await expect(indicator).toHaveCount(0);
    await page.reload();
    await expect(page.getByRole('heading', { name: 'Benutzer', exact: true })).toBeVisible();
    await expect(indicator).toHaveCount(0);
  });

test('impersonation button is offered only for active non-admin users other than oneself',
  async ({ adminPage: page, createUser, adminApi }) => {
    const user = await createUser();
    const admin = await createUser('ADMIN');
    const inactive = await createUser();
    await adminApi.put(`/users/${inactive.id}`, {
      email: inactive.email, firstName: 'E2E', lastName: 'Nutzer', role: 'USER', isActive: false,
    });

    await page.goto('/beitraege/benutzer');
    const button = { name: 'Als diesen Benutzer anmelden' };
    await expect(page.getByRole('row', { name: new RegExp(user.email) }).getByRole('button', button)).toBeVisible();
    await expect(page.getByRole('row', { name: new RegExp(admin.email) }).getByRole('button', { name: 'Bearbeiten' })).toBeVisible();
    await expect(page.getByRole('row', { name: new RegExp(admin.email) }).getByRole('button', button)).toHaveCount(0);
    await expect(page.getByRole('row', { name: new RegExp(inactive.email) }).getByRole('button', button)).toHaveCount(0);
    await expect(page.getByRole('row', { name: /\(angemeldet\)/ }).getByRole('button', button)).toHaveCount(0);
  });

test('impersonating a regular user shows the fee area and ends with logout', async ({ adminPage: page, createUser }) => {
  const user = await createUser();

  await page.goto('/beitraege/benutzer');
  await page.getByRole('row', { name: new RegExp(user.email) })
    .getByRole('button', { name: 'Als diesen Benutzer anmelden' }).click();
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
  await expect(page.getByTestId('impersonation-indicator')).toBeVisible();
  await expect(page.getByRole('link', { name: 'Benutzer' })).toHaveCount(0);

  await page.getByRole('button', { name: 'Benutzermenü' }).click();
  await page.getByRole('button', { name: 'Abmelden' }).click();
  await expect(page).toHaveURL(/\/beitraege\/login/);
  await page.goto('/beitraege/');
  await expect(page).toHaveURL(/\/beitraege\/login/);
});

test('an expired token while impersonating is renewed for the target user, not the admin',
  async ({ adminApi, adminPage: page }) => {
    const parent = await parentUser(adminApi);
    await page.goto('/beitraege/benutzer');
    await page.getByRole('row', { name: new RegExp(parent.email) })
      .getByRole('button', { name: 'Als diesen Benutzer anmelden' }).click();
    await expect(page.getByText(parent.childName)).toBeVisible();

    let expired = false;
    await page.route('**/api/fees/v1/me/fees*', (route) => {
      if (expired) return route.continue();
      expired = true;
      return route.fulfill({ status: 401, body: '{}' });
    });
    const retried = page.waitForResponse((res) =>
      res.url().includes('/me/fees') && res.request().headers()['authorization'] !== undefined && res.status() === 200);
    await page.getByRole('link', { name: 'Beiträge', exact: true }).click();
    await expect(page).toHaveURL(/\/beitraege\/familie\/beitraege$/);
    await retried;
    expect(expired).toBe(true);
    await expect(page.getByTestId('impersonation-indicator')).toBeVisible();
    await expect(page.getByRole('alert')).toHaveCount(0);
    await page.getByRole('link', { name: 'Übersicht', exact: true }).click();
    await expect(page.getByText(parent.childName)).toBeVisible();
    await expect(page.getByRole('link', { name: 'Benutzer' })).toHaveCount(0);
  });
