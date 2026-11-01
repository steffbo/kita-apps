import { test, expect, uniq, loginContext } from './fixtures';
import { API } from './env';

test('admin creates a user who can log in without admin rights', async ({ adminPage: page, browser }) => {
  const email = `neu-${uniq()}@e2e.test`;
  const password = `start-${uniq()}`;

  await page.goto('/beitraege/benutzer');
  await expect(page.getByRole('heading', { name: 'Benutzer', exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Benutzer anlegen' }).click();
  const dialog = page.getByRole('dialog', { name: 'Benutzer anlegen' });
  await dialog.getByLabel('E-Mail (Anmeldename) *').fill(email);
  await dialog.getByLabel('Vorname').fill('Erika');
  await dialog.getByLabel('Nachname').fill('Muster');
  await dialog.getByLabel('Elternstunden').uncheck();
  await dialog.getByLabel('Vorstand').check();
  await expect(dialog.getByText(/kann aber nichts ändern/)).toBeVisible();
  await dialog.getByLabel(/Startpasswort/).fill(password);
  await dialog.getByRole('button', { name: 'Anlegen', exact: true }).click();
  await expect(page.getByText('Benutzer angelegt.')).toBeVisible();
  await expect(page.getByRole('row', { name: new RegExp(email) })).toContainText('Aktiv');

  // The new user sees no admin area and cannot open it by URL.
  const context = await browser.newContext();
  await loginContext(context, email, password);
  const userPage = await context.newPage();
  await userPage.goto('/beitraege/');
  await expect(userPage.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
  await expect(userPage.getByRole('link', { name: 'Benutzer' })).toHaveCount(0);
  await userPage.goto('/beitraege/benutzer');
  await expect(userPage.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
  await context.close();
});

test('user changes own password; other sessions end, own session continues', async ({ browser, createUser }) => {
  const user = await createUser();
  const newPassword = `neu-${uniq()}`;

  const other = await browser.newContext();
  await loginContext(other, user.email, user.password);

  const context = await browser.newContext();
  await loginContext(context, user.email, user.password);
  const page = await context.newPage();
  await page.goto('/beitraege/');
  await page.getByRole('button', { name: 'Benutzermenü' }).click();
  await page.getByRole('button', { name: 'Passwort ändern' }).click();

  const dialog = page.getByRole('dialog', { name: 'Passwort ändern' });
  await dialog.getByLabel('Aktuelles Passwort').fill('falsch-falsch');
  await dialog.getByLabel(/^Neues Passwort \(mind/).fill(newPassword);
  await dialog.getByLabel('Neues Passwort wiederholen').fill(newPassword);
  await dialog.getByRole('button', { name: 'Passwort ändern' }).click();
  await expect(dialog.getByRole('alert')).toContainText('Das aktuelle Passwort ist falsch');

  await dialog.getByLabel('Aktuelles Passwort').fill(user.password);
  await dialog.getByRole('button', { name: 'Passwort ändern' }).click();
  await expect(dialog).toBeHidden();

  // Own session keeps working after a reload (new refresh cookie) …
  await page.reload();
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
  // … while the other browser's refresh token was revoked.
  expect((await other.request.post('/api/fees/v1/auth/refresh')).status()).toBe(401);

  // Old password no longer works, the new one does.
  const check = await browser.newContext();
  expect((await check.request.post('/api/fees/v1/auth/login', { data: { email: user.email, password: user.password } })).status()).toBe(401);
  await loginContext(check, user.email, newPassword);
  await Promise.all([other.close(), context.close(), check.close()]);
});

test('deactivated user cannot log in; own account is protected', async ({ adminPage: page, createUser, browser }) => {
  const user = await createUser();

  await page.goto('/beitraege/benutzer');
  await page.getByRole('row', { name: new RegExp(user.email) }).getByRole('button', { name: 'Bearbeiten' }).click();
  const dialog = page.getByRole('dialog', { name: 'Benutzer bearbeiten' });
  await dialog.getByLabel('Aktiv (darf sich anmelden)').uncheck();
  await dialog.getByRole('button', { name: 'Speichern' }).click();
  await expect(page.getByRole('row', { name: new RegExp(user.email) })).toContainText('Deaktiviert');

  const context = await browser.newContext();
  const res = await context.request.post('/api/fees/v1/auth/login', { data: { email: user.email, password: user.password } });
  expect(res.status()).toBe(401);
  await context.close();

  // Own row: the admin role and the active flag are locked, further roles are not.
  await page.getByRole('row', { name: /\(angemeldet\)/ }).getByRole('button', { name: 'Bearbeiten' }).click();
  await expect(dialog.getByLabel('Administrator')).toBeDisabled();
  await expect(dialog.getByLabel('Vorstand')).toBeEnabled();
  await expect(dialog.getByLabel('Aktiv (darf sich anmelden)')).toBeDisabled();
});

test('user list filters by search and role', async ({ adminPage: page, createUser }) => {
  const user = await createUser();
  const worker = await createUser('PARENT_WORK');

  await page.goto('/beitraege/benutzer');
  const userRow = page.getByRole('row', { name: new RegExp(user.email) });
  const workerRow = page.getByRole('row', { name: new RegExp(worker.email) });
  await expect(userRow).toBeVisible();

  await page.getByRole('searchbox', { name: /Suchen nach Name/ }).fill(user.email);
  await expect(userRow).toBeVisible();
  await expect(workerRow).toHaveCount(0);

  await page.getByRole('searchbox', { name: /Suchen nach Name/ }).fill('');
  await page.getByLabel('Rolle').selectOption({ label: 'Elternstunden' });
  await expect(workerRow).toBeVisible();
  await expect(userRow).toHaveCount(0);
});

test('a user with several roles acts in one, switches, and keeps the last one', async ({ browser, adminApi }) => {
  const email = `multi-${uniq()}@e2e.test`;
  const password = `pw-${uniq()}`;
  await adminApi.post('/users', { email, firstName: 'Mia', lastName: 'Mehrfach', roles: ['PARENT_WORK', 'BOARD'], isActive: true, password });
  const context = await browser.newContext();
  try {
    const token = await loginContext(context, email, password);
    const page = await context.newPage();

    // Starts in the most privileged role.
    await page.goto('/beitraege/');
    await expect(page.getByTestId('active-role')).toContainText('Vorstand');
    await expect(page.getByTestId('read-only-indicator')).toBeVisible();

    const refused = await context.request.post(`${API}/auth/switch-role`, {
      headers: { Authorization: `Bearer ${token}` }, data: { role: 'ADMIN' },
    });
    expect(refused.status()).toBe(400);

    await page.getByRole('button', { name: 'Benutzermenü' }).first().click();
    await page.getByRole('radiogroup', { name: 'Rolle' }).getByRole('radio', { name: 'Elternstunden' }).click();
    await expect(page).toHaveURL(/\/elternstunden$/);
    await expect(page.getByTestId('active-role')).toContainText('Elternstunden');
    await expect(page.getByTestId('read-only-indicator')).toHaveCount(0);
    await expect(page.getByRole('button', { name: 'Stunden erfassen', exact: true })).toBeVisible();

    // Stays across a reload and a new login, and the API follows the role.
    await page.reload();
    await expect(page.getByTestId('active-role')).toContainText('Elternstunden');
    const again = await browser.newContext();
    try {
      const fresh = await loginContext(again, email, password);
      const me = await again.request.get(`${API}/auth/me`, { headers: { Authorization: `Bearer ${fresh}` } });
      expect(await me.json()).toMatchObject({ role: 'PARENT_WORK', roles: ['BOARD', 'PARENT_WORK'] });
      const children = await again.request.get(`${API}/children/`, { headers: { Authorization: `Bearer ${fresh}` } });
      expect(children.status()).toBe(403);
    } finally {
      await again.close();
    }
  } finally {
    await context.close();
  }
});
