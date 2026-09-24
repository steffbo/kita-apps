import { test, expect, uniq, loginContext } from './fixtures';

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

  // Own row: role and active flag are locked.
  await page.getByRole('row', { name: /\(angemeldet\)/ }).getByRole('button', { name: 'Bearbeiten' }).click();
  await expect(dialog.getByLabel('Rolle')).toBeDisabled();
  await expect(dialog.getByLabel('Aktiv (darf sich anmelden)')).toBeDisabled();
});
