import { test, expect, helpers } from '../../fixtures';

test.describe('Schedule Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await helpers.expectLoadingComplete(page);
  });

  test('displays schedule calendar', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /dienstplan/i })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Heute' })).toBeVisible();
  });

  test('can open create entry dialog', async ({ page }) => {
    await page.getByRole('button', { name: 'Eintrag' }).click();
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Neuer Eintrag' })).toBeVisible();
  });

  test('employee dropdown shows options', async ({ page }) => {
    await helpers.expectLoadingComplete(page);
    
    // Open the dialog
    await page.getByRole('button', { name: 'Eintrag' }).click();
    await expect(page.getByRole('dialog')).toBeVisible();
    
    // Find and click the employee dropdown
    const employeeDropdown = page.getByRole('dialog').locator('button[role="combobox"]').first();
    await employeeDropdown.click();
    
    // Wait for dropdown to open
    await page.waitForSelector('[data-radix-popper-content-wrapper]', { timeout: 5000 });
    
    // Check that options are visible - should see Admin Leitung
    await expect(page.getByRole('option', { name: /admin leitung/i })).toBeVisible();
  });

  test('can select an employee from dropdown', async ({ page }) => {
    await helpers.expectLoadingComplete(page);
    
    // Open the dialog
    await page.getByRole('button', { name: 'Eintrag' }).click();
    await expect(page.getByRole('dialog')).toBeVisible();
    
    // Click the employee dropdown
    const employeeDropdown = page.getByRole('dialog').locator('button[role="combobox"]').first();
    await employeeDropdown.click();
    
    // Wait for and click the Admin Leitung option
    await page.waitForSelector('[data-radix-popper-content-wrapper]', { timeout: 5000 });
    await page.getByRole('option', { name: /admin leitung/i }).click();
    
    // The dropdown should close and show the selected value
    await expect(employeeDropdown).toContainText(/admin leitung/i);
  });
});
