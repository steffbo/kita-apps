import { test, expect } from './fixtures';
import { ADMIN } from './env';

async function fillLogin(page: import('@playwright/test').Page, email: string, password: string) {
  await page.getByLabel('E-Mail').fill(email);
  await page.getByLabel('Passwort').fill(password);
  await page.getByRole('button', { name: 'Anmelden' }).click();
}

test('wrong password shows a readable error', async ({ page }) => {
  await page.goto('/beitraege/login');
  await fillLogin(page, ADMIN.email, 'definitely-wrong');
  await expect(page.getByText('E-Mail oder Passwort ist falsch')).toBeVisible();
  await expect(page).toHaveURL(/\/login/);
});

test('session lives in an httpOnly cookie and survives a reload', async ({ page, context }) => {
  await page.goto('/beitraege/');
  await expect(page).toHaveURL(/\/login/);

  await fillLogin(page, ADMIN.email, ADMIN.password);
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();

  // No token is readable by page scripts.
  const stored = await page.evaluate(() => JSON.stringify(localStorage));
  expect(stored).not.toMatch(/token/i);
  const cookie = (await context.cookies()).find((c) => c.name === 'fees_refresh');
  expect(cookie?.httpOnly).toBe(true);
  expect(cookie?.sameSite).toBe('Strict');
  expect(cookie?.path).toBe('/api/fees/v1/auth');
  expect(await page.evaluate(() => document.cookie)).not.toContain('fees_refresh');

  // Reload: the app restores the session via /auth/refresh.
  await page.reload();
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();

  // Logout clears the cookie; a reload stays logged out.
  await page.getByRole('button', { name: 'Benutzermenü' }).click();
  await page.getByRole('button', { name: 'Abmelden' }).click();
  await expect(page).toHaveURL(/\/login/);
  expect((await context.cookies()).find((c) => c.name === 'fees_refresh')).toBeUndefined();
  await page.goto('/beitraege/kinder');
  await expect(page).toHaveURL(/\/login/);
});

test('deep link redirects to login and back', async ({ page }) => {
  await page.goto('/beitraege/kinder');
  await expect(page).toHaveURL(/\/login\?redirect=/);
  await fillLogin(page, ADMIN.email, ADMIN.password);
  await expect(page.getByRole('heading', { name: 'Kinder' })).toBeVisible();
});

test('repeated wrong passwords are throttled', async ({ page, createUser }) => {
  // Own account, so the admin used by other tests is never blocked.
  const user = await createUser();
  await page.goto('/beitraege/login');
  for (let i = 0; i < 5; i++) {
    await fillLogin(page, user.email, 'wrong-password');
    await expect(page.getByText('E-Mail oder Passwort ist falsch')).toBeVisible();
  }
  // Blocked now, even with the correct password.
  await fillLogin(page, user.email, user.password);
  await expect(page.getByText(/Zu viele fehlgeschlagene Versuche\. Bitte in \d+ Minuten? erneut versuchen\./)).toBeVisible();
  await expect(page).toHaveURL(/\/login/);
});
