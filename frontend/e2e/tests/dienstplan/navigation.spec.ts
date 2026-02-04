import { test, expect, NavigationHelper, helpers } from '../../fixtures';

test.describe('Login Flow', () => {
  test.use({ storageState: { cookies: [], origins: [] } }); // Don't use saved auth

  test('shows login page for unauthenticated users', async ({ page }) => {
    await page.goto('/');
    
    // Should redirect to login
    await expect(page).toHaveURL(/\/login/);
    // Login page has "Dienstplan" as heading
    await expect(page.getByRole('heading', { name: /dienstplan/i })).toBeVisible();
  });

  test('successfully logs in with valid credentials', async ({ page }) => {
    await page.goto('/login');
    
    await page.getByLabel(/e-mail/i).fill('admin@knirpsenstadt.de');
    await page.getByLabel(/passwort/i).fill('admin123');
    await page.getByRole('button', { name: /anmelden/i }).click();
    
    // Should redirect away from login
    await expect(page).not.toHaveURL(/\/login/);
    
    // Should show navigation menu
    await expect(page.getByRole('navigation').first()).toBeVisible();
  });

  test('shows error for invalid credentials', async ({ page }) => {
    await page.goto('/login');
    
    await page.getByLabel(/e-mail/i).fill('wrong@example.com');
    await page.getByLabel(/passwort/i).fill('wrongpassword');
    await page.getByRole('button', { name: /anmelden/i }).click();
    
    // Should show error message
    await expect(page.locator('.bg-red-50, [class*="error"]')).toBeVisible();
    
    // Should stay on login page
    await expect(page).toHaveURL(/\/login/);
  });

  test('validates required fields', async ({ page }) => {
    await page.goto('/login');
    
    // Try to submit empty form
    await page.getByRole('button', { name: /anmelden/i }).click();
    
    // Should show validation errors or stay on page
    await expect(page).toHaveURL(/\/login/);
  });
});

test.describe('Navigation', () => {
  test('can navigate to all main sections', async ({ page }) => {
    await page.goto('/');
    
    // We're on the schedule page (root)
    await expect(page.getByRole('navigation').first()).toBeVisible();
    
    // Navigate to employees
    await page.getByRole('link', { name: /mitarbeiter/i }).click();
    await expect(page).toHaveURL(/\/employees/);
    
    // Navigate to groups
    await page.getByRole('link', { name: /gruppen/i }).click();
    await expect(page).toHaveURL(/\/groups/);
    
    // Navigate to special days
    await page.getByRole('link', { name: /besondere tage/i }).click();
    await expect(page).toHaveURL(/\/special-days/);
    
    // Navigate to statistics
    await page.getByRole('link', { name: /statistik/i }).click();
    await expect(page).toHaveURL(/\/statistics/);
    
    // Navigate back to schedule (Dienstplan link)
    await page.getByRole('link', { name: /dienstplan/i }).first().click();
    await expect(page).toHaveURL('/');
  });

  test('can logout', async ({ page }) => {
    await page.goto('/');
    
    // Open user menu first
    await page.getByRole('button', { name: /admin leitung/i }).click();
    
    // Find logout button
    await page.getByRole('button', { name: /abmelden|logout/i }).click();
    
    // Should be on login page
    await expect(page).toHaveURL(/\/login/);
  });
});

test.describe('Dashboard / Schedule View', () => {
  test('displays schedule page', async ({ page }) => {
    await page.goto('/');
    
    // Should be on the schedule page (root path)
    await expect(page).toHaveURL('/');
    
    // Should show the schedule content
    await expect(page.getByRole('navigation').first()).toBeVisible();
  });

  test('can navigate between weeks', async ({ page }) => {
    await page.goto('/');
    
    // Wait for page to load
    await helpers.expectLoadingComplete(page);
    
    // Find navigation buttons (chevrons)
    const nextButton = page.locator('button').filter({ has: page.locator('svg') }).first();
    
    if (await nextButton.isVisible()) {
      await nextButton.click();
      // Page should still be on root
      await expect(page).toHaveURL('/');
    }
  });

  test('shows schedule content after loading', async ({ page }) => {
    await page.goto('/');
    
    // Wait for data to load
    await helpers.expectLoadingComplete(page);
    
    // Should show some content (table, grid, etc.)
    await expect(page.locator('table, [class*="grid"], [class*="schedule"]').first()).toBeVisible();
  });
});
