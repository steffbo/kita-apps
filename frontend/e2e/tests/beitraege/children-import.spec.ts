import { test, expect } from '@playwright/test';

test.describe('Beiträge - Children CSV Import', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/kinder');
    // Wait for loading to complete
    await expect(page.locator('.animate-spin')).toBeHidden({ timeout: 10000 });
  });

  test('displays import button on children page', async ({ page }) => {
    await expect(page.getByRole('button', { name: /importieren/i })).toBeVisible();
  });

  test('can navigate to import page', async ({ page }) => {
    await page.getByRole('button', { name: /importieren/i }).click();
    
    // Should be on import page
    await expect(page).toHaveURL(/\/kinder\/import/);
    await expect(page.getByRole('heading', { name: /kinder importieren/i })).toBeVisible();
  });

  test('shows step 1 - file upload initially', async ({ page }) => {
    await page.goto('/kinder/import');
    
    // Step 1 should be active
    await expect(page.getByText('Schritt 1 von 4')).toBeVisible();
    await expect(page.getByText(/CSV-Datei hochladen/i)).toBeVisible();
    
    // Upload area should be visible
    await expect(page.getByText(/datei hierher ziehen/i)).toBeVisible();
  });

  test('can upload CSV file and proceed to step 2', async ({ page }) => {
    await page.goto('/kinder/import');
    
    // Find the file input (it's hidden but we can use setInputFiles)
    const fileInput = page.locator('input[type="file"]');
    
    // Upload the test CSV - path relative to frontend directory
    await fileInput.setInputFiles('e2e/fixtures/test-children-import.csv');
    
    // Should show loading state
    await expect(page.getByText(/wird verarbeitet/i)).toBeVisible({ timeout: 5000 });
    
    // Wait for step 2
    await expect(page.getByText('Schritt 2 von 4')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(/spalten zuordnen/i)).toBeVisible();
    
    // Should show detected columns from CSV
    await expect(page.getByText('Mitgliedsnummer')).toBeVisible();
    await expect(page.getByText('Vorname')).toBeVisible();
    await expect(page.getByText('Nachname')).toBeVisible();
  });

  test('can map columns and proceed to step 3', async ({ page }) => {
    await page.goto('/kinder/import');
    
    // Upload CSV
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles('e2e/fixtures/test-children-import.csv');
    
    // Wait for step 2
    await expect(page.getByText('Schritt 2 von 4')).toBeVisible({ timeout: 10000 });
    
    // The columns should auto-map based on header names
    // Click next to proceed to preview
    await page.getByRole('button', { name: /weiter/i }).click();
    
    // Should show step 3 - preview
    await expect(page.getByText('Schritt 3 von 4')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(/daten prüfen/i)).toBeVisible();
    
    // Should show preview data
    await expect(page.getByText('Emma')).toBeVisible();
    await expect(page.getByText('Müller')).toBeVisible();
    await expect(page.getByText('IMP001')).toBeVisible();
  });

  test('shows validation in preview step', async ({ page }) => {
    await page.goto('/kinder/import');
    
    // Upload CSV
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles('e2e/fixtures/test-children-import.csv');
    
    // Wait for step 2 and proceed
    await expect(page.getByText('Schritt 2 von 4')).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /weiter/i }).click();
    
    // Wait for step 3
    await expect(page.getByText('Schritt 3 von 4')).toBeVisible({ timeout: 10000 });
    
    // Should show validation status (valid rows should have checkmarks)
    await expect(page.locator('[data-testid="row-valid"]').or(page.locator('svg.text-green-500'))).toBeVisible();
    
    // Should show row count
    await expect(page.getByText(/3 zeilen/i).or(page.getByText(/3 kinder/i))).toBeVisible();
  });

  test('can select/deselect rows in preview', async ({ page }) => {
    await page.goto('/kinder/import');
    
    // Upload CSV
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles('e2e/fixtures/test-children-import.csv');
    
    // Wait for step 2 and proceed
    await expect(page.getByText('Schritt 2 von 4')).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /weiter/i }).click();
    
    // Wait for step 3
    await expect(page.getByText('Schritt 3 von 4')).toBeVisible({ timeout: 10000 });
    
    // Find checkboxes for rows
    const checkboxes = page.locator('input[type="checkbox"]');
    const firstRowCheckbox = checkboxes.first();
    
    // Toggle a checkbox
    const initialState = await firstRowCheckbox.isChecked();
    await firstRowCheckbox.click();
    const newState = await firstRowCheckbox.isChecked();
    
    expect(newState).not.toBe(initialState);
  });

  test('can execute import and see results', async ({ page }) => {
    await page.goto('/kinder/import');
    
    // Upload CSV
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles('e2e/fixtures/test-children-import.csv');
    
    // Step 2: mapping
    await expect(page.getByText('Schritt 2 von 4')).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /weiter/i }).click();
    
    // Step 3: preview
    await expect(page.getByText('Schritt 3 von 4')).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /importieren/i }).click();
    
    // Step 4: results
    await expect(page.getByText('Schritt 4 von 4')).toBeVisible({ timeout: 15000 });
    await expect(page.getByText(/import abgeschlossen/i).or(page.getByText(/ergebnis/i))).toBeVisible();
    
    // Should show success info
    await expect(page.getByText(/erfolgreich/i).or(page.getByText(/importiert/i))).toBeVisible();
  });

  test('can go back to children list after import', async ({ page }) => {
    await page.goto('/kinder/import');
    
    // Upload CSV
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles('e2e/fixtures/test-children-import.csv');
    
    // Complete the wizard
    await expect(page.getByText('Schritt 2 von 4')).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /weiter/i }).click();
    
    await expect(page.getByText('Schritt 3 von 4')).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /importieren/i }).click();
    
    await expect(page.getByText('Schritt 4 von 4')).toBeVisible({ timeout: 15000 });
    
    // Click "back to list" button
    await page.getByRole('button', { name: /zur übersicht/i }).or(
      page.getByRole('link', { name: /zur übersicht/i })
    ).click();
    
    // Should be on children page
    await expect(page).toHaveURL(/\/kinder$/);
    
    // Imported children should be visible
    await expect(page.getByText('Emma')).toBeVisible();
  });

  test('can cancel import and go back', async ({ page }) => {
    await page.goto('/kinder/import');
    
    // Click cancel/back button
    await page.getByRole('button', { name: /abbrechen/i }).or(
      page.getByRole('button', { name: /zurück/i })
    ).first().click();
    
    // Should be on children page
    await expect(page).toHaveURL(/\/kinder$/);
  });
});

test.describe('Beiträge - Import Error Handling', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/kinder/import');
  });

  test('shows error for invalid file type', async ({ page }) => {
    // Verify the upload UI is shown
    await expect(page.getByText(/CSV-Datei hochladen/i)).toBeVisible();
    
    // The file input should accept only csv files
    const fileInput = page.locator('input[type="file"]');
    await expect(fileInput).toHaveAttribute('accept', '.csv');
  });
});

test.describe('Beiträge - Import with Existing Data', () => {
  test('detects and warns about duplicate member numbers', async ({ page }) => {
    // First, create a child with a specific member number
    await page.goto('/kinder');
    await expect(page.locator('.animate-spin')).toBeHidden({ timeout: 10000 });
    
    // Create child with member number IMP001 (same as in test CSV)
    await page.getByRole('button', { name: /kind hinzufügen/i }).click();
    await page.getByLabel(/mitgliedsnummer/i).fill('IMP001');
    await page.getByLabel(/vorname/i).fill('Existing');
    await page.getByLabel(/nachname/i).fill('Child');
    await page.getByLabel(/geburtsdatum/i).fill('2021-01-01');
    await page.getByLabel(/eintrittsdatum/i).fill('2024-01-01');
    await page.getByRole('button', { name: /speichern/i }).click();
    
    await expect(page.getByRole('heading', { name: /kind hinzufügen/i })).toBeHidden();
    
    // Now try to import CSV that contains IMP001
    await page.goto('/kinder/import');
    
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles('e2e/fixtures/test-children-import.csv');
    
    // Proceed to preview
    await expect(page.getByText('Schritt 2 von 4')).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /weiter/i }).click();
    
    // In preview, should show warning for duplicate
    await expect(page.getByText('Schritt 3 von 4')).toBeVisible({ timeout: 10000 });
    
    // Should see duplicate warning
    await expect(page.getByText(/duplikat/i).or(page.getByText(/bereits vorhanden/i))).toBeVisible();
  });
});
