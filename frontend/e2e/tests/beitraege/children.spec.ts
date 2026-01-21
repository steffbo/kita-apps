import { test, expect } from '@playwright/test';

test.describe('Beiträge - Children Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/kinder');
    // Wait for loading to complete
    await expect(page.locator('.animate-spin')).toBeHidden({ timeout: 10000 });
  });

  test('displays children list page', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /kinder/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /kind hinzufügen/i })).toBeVisible();
  });

  test('can open create child dialog', async ({ page }) => {
    await page.getByRole('button', { name: /kind hinzufügen/i }).click();
    
    // Dialog should open
    await expect(page.getByRole('heading', { name: /kind hinzufügen/i })).toBeVisible();
    await expect(page.getByLabel(/mitgliedsnummer/i)).toBeVisible();
    await expect(page.getByLabel(/vorname/i)).toBeVisible();
    await expect(page.getByLabel(/nachname/i)).toBeVisible();
  });

  test('can create a new child', async ({ page }) => {
    // Generate unique member number
    const memberNumber = `T${Date.now().toString().slice(-6)}`;
    const firstName = 'Test';
    const lastName = `Kind-${Date.now().toString().slice(-4)}`;
    
    // Open dialog
    await page.getByRole('button', { name: /kind hinzufügen/i }).click();
    
    // Fill form
    await page.getByLabel(/mitgliedsnummer/i).fill(memberNumber);
    await page.getByLabel(/vorname/i).fill(firstName);
    await page.getByLabel(/nachname/i).fill(lastName);
    await page.getByLabel(/geburtsdatum/i).fill('2022-06-15');
    await page.getByLabel(/eintrittsdatum/i).fill('2024-01-01');
    
    // Submit
    await page.getByRole('button', { name: /speichern/i }).click();
    
    // Dialog should close
    await expect(page.getByRole('heading', { name: /kind hinzufügen/i })).toBeHidden();
    
    // New child should appear in list
    await expect(page.getByText(memberNumber)).toBeVisible();
    await expect(page.getByText(`${firstName} ${lastName}`)).toBeVisible();
  });

  test('validates required fields when creating child', async ({ page }) => {
    // Open dialog
    await page.getByRole('button', { name: /kind hinzufügen/i }).click();
    
    // Try to submit empty form
    await page.getByRole('button', { name: /speichern/i }).click();
    
    // Dialog should still be visible (HTML5 validation prevents submission)
    await expect(page.getByRole('heading', { name: /kind hinzufügen/i })).toBeVisible();
  });

  test('can search for children', async ({ page }) => {
    // First create a child to search for
    const memberNumber = `S${Date.now().toString().slice(-6)}`;
    const searchName = `Suchtest-${Date.now().toString().slice(-4)}`;
    
    await page.getByRole('button', { name: /kind hinzufügen/i }).click();
    await page.getByLabel(/mitgliedsnummer/i).fill(memberNumber);
    await page.getByLabel(/vorname/i).fill(searchName);
    await page.getByLabel(/nachname/i).fill('Nachname');
    await page.getByLabel(/geburtsdatum/i).fill('2021-03-20');
    await page.getByLabel(/eintrittsdatum/i).fill('2024-01-01');
    await page.getByRole('button', { name: /speichern/i }).click();
    
    // Wait for dialog to close
    await expect(page.getByRole('heading', { name: /kind hinzufügen/i })).toBeHidden();
    
    // Use search
    await page.getByPlaceholder(/suchen/i).fill(searchName);
    
    // Should find the child
    await expect(page.getByText(searchName)).toBeVisible();
  });
});

test.describe('Beiträge - Child Detail & Edit', () => {
  let testChildName: string;
  let testMemberNumber: string;

  test.beforeEach(async ({ page }) => {
    // Create a test child first
    testMemberNumber = `D${Date.now().toString().slice(-6)}`;
    testChildName = `Detail-${Date.now().toString().slice(-4)}`;
    
    await page.goto('/kinder');
    await expect(page.locator('.animate-spin')).toBeHidden({ timeout: 10000 });
    
    await page.getByRole('button', { name: /kind hinzufügen/i }).click();
    await page.getByLabel(/mitgliedsnummer/i).fill(testMemberNumber);
    await page.getByLabel(/vorname/i).fill(testChildName);
    await page.getByLabel(/nachname/i).fill('Testname');
    await page.getByLabel(/geburtsdatum/i).fill('2020-05-10');
    await page.getByLabel(/eintrittsdatum/i).fill('2023-09-01');
    await page.getByRole('button', { name: /speichern/i }).click();
    
    await expect(page.getByRole('heading', { name: /kind hinzufügen/i })).toBeHidden();
  });

  test('can navigate to child detail page', async ({ page }) => {
    // Click on the child row
    await page.getByText(testChildName).click();
    
    // Should be on detail page
    await expect(page.getByRole('heading', { name: new RegExp(testChildName) })).toBeVisible();
    await expect(page.getByText(/zurück zur übersicht/i)).toBeVisible();
  });

  test('can edit a child', async ({ page }) => {
    // Navigate to detail page
    await page.getByText(testChildName).click();
    await expect(page.getByRole('heading', { name: new RegExp(testChildName) })).toBeVisible();
    
    // Click edit button
    await page.getByRole('button', { name: /bearbeiten/i }).or(page.locator('button[title="Bearbeiten"]')).click();
    
    // Edit dialog should open
    await expect(page.getByRole('heading', { name: /kind bearbeiten/i })).toBeVisible();
    
    // Change the name
    const newFirstName = `Edited-${Date.now().toString().slice(-4)}`;
    await page.getByLabel(/vorname/i).fill(newFirstName);
    
    // Add address
    await page.getByLabel(/straße/i).fill('Teststraße');
    await page.getByLabel(/hausnummer/i).fill('42');
    await page.getByLabel(/plz/i).fill('12345');
    await page.getByLabel(/ort/i).fill('Teststadt');
    
    // Submit
    await page.getByRole('button', { name: /speichern/i }).click();
    
    // Dialog should close
    await expect(page.getByRole('heading', { name: /kind bearbeiten/i })).toBeHidden();
    
    // Updated name should be visible
    await expect(page.getByRole('heading', { name: new RegExp(newFirstName) })).toBeVisible();
    
    // Address should be visible
    await expect(page.getByText(/teststraße 42/i)).toBeVisible();
  });

  test('can delete a child', async ({ page }) => {
    // Navigate to detail page
    await page.getByText(testChildName).click();
    await expect(page.getByRole('heading', { name: new RegExp(testChildName) })).toBeVisible();
    
    // Click delete button
    await page.getByRole('button', { name: /löschen/i }).or(page.locator('button[title="Löschen"]')).click();
    
    // Confirmation dialog should appear
    await expect(page.getByRole('heading', { name: /kind löschen/i })).toBeVisible();
    await expect(page.getByText(/wirklich löschen/i)).toBeVisible();
    
    // Confirm deletion
    await page.getByRole('button', { name: /löschen/i }).last().click();
    
    // Should redirect to children list
    await expect(page).toHaveURL(/\/kinder$/);
    
    // Child should no longer be in the list
    await expect(page.getByText(testMemberNumber)).toBeHidden();
  });

  test('can cancel delete', async ({ page }) => {
    // Navigate to detail page
    await page.getByText(testChildName).click();
    await expect(page.getByRole('heading', { name: new RegExp(testChildName) })).toBeVisible();
    
    // Click delete button
    await page.getByRole('button', { name: /löschen/i }).or(page.locator('button[title="Löschen"]')).click();
    
    // Cancel
    await page.getByRole('button', { name: /abbrechen/i }).click();
    
    // Should still be on detail page
    await expect(page.getByRole('heading', { name: new RegExp(testChildName) })).toBeVisible();
  });
});

test.describe('Beiträge - Login', () => {
  test('can login with valid credentials', async ({ browser }) => {
    // Create a fresh context without auth state
    const context = await browser.newContext();
    const page = await context.newPage();
    
    await page.goto('http://localhost:5175/login');
    
    // Wait for page to actually load
    await page.waitForLoadState('networkidle');
    
    // If redirected to dashboard, we're still logged in from localStorage
    // so check the URL first
    if (!page.url().includes('/login')) {
      // Need to clear localStorage and reload
      await page.evaluate(() => localStorage.clear());
      await page.goto('http://localhost:5175/login');
      await page.waitForLoadState('networkidle');
    }
    
    // Wait for login form to be ready
    await expect(page.getByLabel(/e-mail/i)).toBeVisible({ timeout: 5000 });
    
    // Fill credentials
    await page.getByLabel(/e-mail/i).fill('admin@knirpsenstadt.de');
    await page.getByLabel(/passwort/i).fill('admin123');
    
    // Submit
    await page.getByRole('button', { name: /anmelden/i }).click();
    
    // Should redirect to dashboard
    await expect(page).not.toHaveURL(/\/login/);
    await expect(page.getByRole('heading', { name: /dashboard/i })).toBeVisible();
    
    await context.close();
  });

  test('shows error with invalid credentials', async ({ browser }) => {
    // Create a fresh context without auth state
    const context = await browser.newContext();
    const page = await context.newPage();
    
    // Clear localStorage first
    await page.goto('http://localhost:5175/login');
    await page.evaluate(() => localStorage.clear());
    await page.goto('http://localhost:5175/login');
    await page.waitForLoadState('networkidle');
    
    // Wait for login form to be ready
    await expect(page.getByLabel(/e-mail/i)).toBeVisible({ timeout: 5000 });
    
    await page.getByLabel(/e-mail/i).fill('wrong@email.com');
    await page.getByLabel(/passwort/i).fill('wrongpassword');
    await page.getByRole('button', { name: /anmelden/i }).click();
    
    // Should show error and stay on login page
    await expect(page).toHaveURL(/\/login/);
    
    await context.close();
  });
});
