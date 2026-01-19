import { test, expect, helpers } from '../../fixtures';

test.describe('Employees Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/employees');
    await helpers.expectLoadingComplete(page);
  });

  test('displays employee list', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /mitarbeiter/i })).toBeVisible();
    
    // Should show at least the admin user in the table
    await expect(page.getByRole('cell', { name: /admin@knirpsenstadt.de/i })).toBeVisible();
  });

  test('can open create employee dialog', async ({ page }) => {
    // Click add button
    await page.getByRole('button', { name: /neuer mitarbeiter|hinzufügen|\+/i }).click();
    
    // Dialog should open
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByLabel(/vorname/i)).toBeVisible();
    await expect(page.getByLabel(/nachname/i)).toBeVisible();
    await expect(page.getByLabel(/e-mail/i)).toBeVisible();
  });

  test('can create a new employee', async ({ page }) => {
    // Open dialog
    await page.getByRole('button', { name: /neuer mitarbeiter|hinzufügen|\+/i }).click();
    
    // Fill form with unique email
    const uniqueEmail = `test.${Date.now()}@knirpsenstadt.de`;
    await page.getByLabel(/vorname/i).fill('Test');
    await page.getByLabel(/nachname/i).fill('Mitarbeiter');
    await page.getByLabel(/e-mail/i).fill(uniqueEmail);
    
    // Set weekly hours if field exists
    const weeklyHoursField = page.getByLabel(/wochenstunden|stunden/i);
    if (await weeklyHoursField.isVisible()) {
      await weeklyHoursField.fill('38');
    }
    
    // Submit
    await page.getByRole('button', { name: /erstellen|speichern|anlegen/i }).click();
    
    // Dialog should close
    await expect(page.getByRole('dialog')).toBeHidden();
    
    // New employee should appear in list - check by email which is unique
    await expect(page.getByRole('cell', { name: uniqueEmail })).toBeVisible();
  });

  test('can edit an employee', async ({ page }) => {
    // Find and click edit button for admin user row
    const adminRow = page.getByRole('row').filter({ hasText: /admin@knirpsenstadt.de/i });
    await adminRow.getByRole('button', { name: /bearbeiten|edit/i }).click();
    
    // Dialog should open with existing data
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByLabel(/vorname/i)).toHaveValue(/admin/i);
  });

  test('shows employee details in table', async ({ page }) => {
    // Table should show key columns
    await expect(page.getByRole('columnheader', { name: /name/i })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: /e-mail/i })).toBeVisible();
    
    // Admin user row should show role badge - use exact match for the badge
    const adminRow = page.getByRole('row').filter({ hasText: /admin@knirpsenstadt.de/i });
    await expect(adminRow.locator('.rounded-full').filter({ hasText: 'Leitung' })).toBeVisible();
  });

  test('can filter or search employees', async ({ page }) => {
    // Look for search field
    const searchField = page.getByPlaceholder(/suche|filter/i);
    
    if (await searchField.isVisible()) {
      await searchField.fill('admin');
      
      // Should filter results
      await expect(page.getByRole('cell', { name: /admin@knirpsenstadt.de/i })).toBeVisible();
    }
  });
});

test.describe('Employee Validation', () => {
  test('validates required fields when creating employee', async ({ page }) => {
    await page.goto('/employees');
    
    // Open dialog
    await page.getByRole('button', { name: /neuer mitarbeiter|hinzufügen|\+/i }).click();
    
    // Try to submit empty form
    await page.getByRole('button', { name: /erstellen|speichern|anlegen/i }).click();
    
    // Dialog should still be visible (form not submitted)
    await expect(page.getByRole('dialog')).toBeVisible();
    
    // Check for HTML5 validation - the required fields should prevent submission
    const emailField = page.getByLabel(/e-mail/i);
    await expect(emailField).toBeVisible();
  });

  test('validates email format', async ({ page }) => {
    await page.goto('/employees');
    
    // Open dialog
    await page.getByRole('button', { name: /neuer mitarbeiter|hinzufügen|\+/i }).click();
    
    // Fill with invalid email
    await page.getByLabel(/vorname/i).fill('Test');
    await page.getByLabel(/nachname/i).fill('User');
    await page.getByLabel(/e-mail/i).fill('invalid-email');
    
    // Try to submit
    await page.getByRole('button', { name: /erstellen|speichern|anlegen/i }).click();
    
    // Dialog should still be open (validation failed)
    await expect(page.getByRole('dialog')).toBeVisible();
  });
});
