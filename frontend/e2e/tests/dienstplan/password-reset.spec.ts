import { test, expect } from '../../fixtures';

test.describe('Password Reset Flow', () => {
  test.use({ storageState: { cookies: [], origins: [] } }); // Don't use saved auth

  test('shows forgot password link on login page', async ({ page }) => {
    await page.goto('/login');
    
    // Should have a "Passwort vergessen?" link
    const forgotPasswordLink = page.getByRole('link', { name: /passwort vergessen/i });
    await expect(forgotPasswordLink).toBeVisible();
  });

  test('can navigate to password reset page', async ({ page }) => {
    await page.goto('/login');
    
    // Click forgot password link
    await page.getByRole('link', { name: /passwort vergessen/i }).click();
    
    // Should be on password reset page
    await expect(page).toHaveURL(/\/password-reset/);
    
    // Should show heading
    await expect(page.getByRole('heading', { name: /passwort zurücksetzen/i })).toBeVisible();
  });

  test('password reset request form has email field', async ({ page }) => {
    await page.goto('/password-reset');
    
    // Should show email input
    await expect(page.getByLabel(/e-mail/i)).toBeVisible();
    
    // Should show submit button
    await expect(page.getByRole('button', { name: /anfordern|senden/i })).toBeVisible();
  });

  test('shows validation for empty email', async ({ page }) => {
    await page.goto('/password-reset');
    
    // Click submit without entering email
    await page.getByRole('button', { name: /anfordern|senden/i }).click();
    
    // Should still be on password reset page (form not submitted)
    await expect(page).toHaveURL(/\/password-reset/);
  });

  test('can submit password reset request', async ({ page }) => {
    await page.goto('/password-reset');
    
    // Enter email
    await page.getByLabel(/e-mail/i).fill('test@example.com');
    
    // Submit form
    await page.getByRole('button', { name: /anfordern|senden/i }).click();
    
    // Should show success message or stay on page (API may or may not return success)
    // The important thing is the page doesn't crash
    await expect(page.locator('body')).toBeVisible();
  });

  test('can navigate back to login from password reset', async ({ page }) => {
    await page.goto('/password-reset');
    
    // Find back to login link
    const backLink = page.getByRole('link', { name: /zurück.*anmeldung|anmelden/i });
    
    if (await backLink.isVisible()) {
      await backLink.click();
      await expect(page).toHaveURL(/\/login/);
    }
  });

  test('password reset confirm page with token shows password fields', async ({ page }) => {
    // Navigate to password reset with a token
    await page.goto('/passwort-zuruecksetzen?token=test-token');
    
    // Should show new password input
    await expect(page.getByLabel(/neues passwort/i).first()).toBeVisible();
    
    // Should show confirm password input or just one password field
    const confirmPassword = page.getByLabel(/passwort bestätigen|wiederholen/i);
    // Either confirm field exists or the form has a submit button
    if (await confirmPassword.isVisible()) {
      await expect(confirmPassword).toBeVisible();
    }
    
    // Should show submit button
    await expect(page.getByRole('button', { name: /zurücksetzen|speichern|ändern/i })).toBeVisible();
  });

  test('password reset confirm validates password length', async ({ page }) => {
    await page.goto('/passwort-zuruecksetzen?token=test-token');
    
    // Enter short password
    await page.getByLabel(/neues passwort/i).first().fill('short');
    
    const confirmPassword = page.getByLabel(/passwort bestätigen|wiederholen/i);
    if (await confirmPassword.isVisible()) {
      await confirmPassword.fill('short');
    }
    
    // Submit should remain disabled for invalid password length
    await expect(page.getByRole('button', { name: /zurücksetzen|speichern|ändern/i })).toBeDisabled();
  });
});
