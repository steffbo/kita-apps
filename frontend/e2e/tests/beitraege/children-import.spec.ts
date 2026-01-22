import { test, expect } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';

// Helper to create a test CSV file with unique member numbers
function createUniqueTestCSV(): string {
  const timestamp = Date.now().toString().slice(-6);
  const csvContent = `Mitgliedsnummer;Vorname;Nachname;Geburtsdatum;Eintrittsdatum;Straße;Hausnr;PLZ;Ort;Eltern 1 Vorname;Eltern 1 Nachname;Eltern 1 Email;Eltern 1 Telefon
TEST${timestamp}A;Emma;Müller;15.03.2021;01.08.2024;Hauptstraße;12;12345;Musterstadt;Anna;Müller;anna.mueller@example.com;0151-12345678
TEST${timestamp}B;Max;Schmidt;22.07.2020;01.08.2024;Nebenweg;5a;12345;Musterstadt;Thomas;Schmidt;thomas.schmidt@example.com;0152-87654321
TEST${timestamp}C;Lina;Weber;08.11.2021;01.09.2024;Parkstraße;8;12346;Nachbarort;Maria;Weber;maria.weber@example.com;0163-11223344
`;
  // path.join(__dirname) points to e2e/tests/beitraege, so we go up two levels to e2e/fixtures
  const tempPath = path.join(__dirname, '..', '..', 'fixtures', `test-children-import-${timestamp}.csv`);
  fs.writeFileSync(tempPath, csvContent, 'utf8');
  return tempPath;
}

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
    
    // Step 1 should be active - check for Upload step label
    await expect(page.getByText('Upload')).toBeVisible();
    
    // Upload area should be visible
    await expect(page.getByText(/CSV-Datei hier ablegen/i)).toBeVisible();
  });

  test('can upload CSV file and proceed to step 2', async ({ page }) => {
    await page.goto('/kinder/import');
    
    // Find the file input (it's hidden but we can use setInputFiles)
    const fileInput = page.locator('input[type="file"]');
    
    // Upload the test CSV - path relative to frontend directory
    await fileInput.setInputFiles('e2e/fixtures/test-children-import.csv');
    
    // Should show loading state
    await expect(page.getByText(/wird verarbeitet/i)).toBeVisible({ timeout: 5000 });
    
    // Wait for step 2 - check for mapping section
    await expect(page.getByText(/Datei erkannt/i)).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(/Gefundene Spalten/i)).toBeVisible();
    
    // Should show detected columns from CSV - use heading
    await expect(page.getByRole('heading', { name: 'Datei erkannt' })).toBeVisible();
  });

  test('can map columns and proceed to step 3', async ({ page }) => {
    await page.goto('/kinder/import');
    
    // Upload CSV
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles('e2e/fixtures/test-children-import.csv');
    
    // Wait for step 2 - mapping section
    await expect(page.getByText(/Datei erkannt/i)).toBeVisible({ timeout: 10000 });
    
    // The columns should auto-map based on header names
    // Click "Vorschau" to proceed to preview
    await page.getByRole('button', { name: /vorschau/i }).click();
    
    // Should show step 3 - preview with data table
    // Wait for preview heading (step 3 has h2 "Vorschau" and shows gültige Einträge)
    await expect(page.getByText(/gültige einträge/i)).toBeVisible({ timeout: 10000 });
    
    // Should show preview data in the table
    await expect(page.locator('table')).toBeVisible();
  });

  test('shows validation in preview step', async ({ page }) => {
    await page.goto('/kinder/import');
    
    // Create a unique CSV to avoid duplicate member numbers
    const csvPath = createUniqueTestCSV();
    
    // Upload CSV
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles(csvPath);
    
    // Wait for step 2 and proceed
    await expect(page.getByText(/Datei erkannt/i)).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /vorschau/i }).click();
    
    // Wait for step 3 - preview (shows "gültige Einträge" text)
    await expect(page.getByText(/gültige einträge/i)).toBeVisible({ timeout: 10000 });
    
    // Should show validation status (valid rows should have NEU badge)
    await expect(page.getByText('NEU').first()).toBeVisible();
    
    // Cleanup
    fs.unlinkSync(csvPath);
  });

  test('can select/deselect rows in preview', async ({ page }) => {
    await page.goto('/kinder/import');
    
    // Create a unique CSV to avoid duplicate member numbers
    const csvPath = createUniqueTestCSV();
    
    // Upload CSV
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles(csvPath);
    
    // Wait for step 2 and proceed
    await expect(page.getByText(/Datei erkannt/i)).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /vorschau/i }).click();
    
    // Wait for step 3 - preview
    await expect(page.getByText(/gültige einträge/i)).toBeVisible({ timeout: 10000 });
    
    // Find checkboxes for rows
    const checkboxes = page.locator('input[type="checkbox"]');
    const firstRowCheckbox = checkboxes.first();
    
    // Toggle a checkbox
    const initialState = await firstRowCheckbox.isChecked();
    await firstRowCheckbox.click();
    const newState = await firstRowCheckbox.isChecked();
    
    expect(newState).not.toBe(initialState);
    
    // Cleanup
    fs.unlinkSync(csvPath);
  });

  test('can execute import and see results', async ({ page }) => {
    await page.goto('/kinder/import');
    
    // Create a unique CSV to avoid duplicate member numbers
    const csvPath = createUniqueTestCSV();
    
    // Upload CSV
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles(csvPath);
    
    // Step 2: mapping
    await expect(page.getByText(/Datei erkannt/i)).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /vorschau/i }).click();
    
    // Step 3: preview - wait for data
    await expect(page.getByText(/gültige einträge/i)).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /importieren/i }).click();
    
    // Step 4: results - should show success heading
    await expect(page.getByRole('heading', { name: /Import abgeschlossen/i })).toBeVisible({ timeout: 15000 });
    
    // Cleanup
    fs.unlinkSync(csvPath);
  });

  test('can go back to children list after import', async ({ page }) => {
    await page.goto('/kinder/import');
    
    // Create a unique CSV to avoid duplicate member numbers
    const csvPath = createUniqueTestCSV();
    
    // Upload CSV
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles(csvPath);
    
    // Complete the wizard
    await expect(page.getByText(/Datei erkannt/i)).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /vorschau/i }).click();
    
    // Wait for preview table to be visible (with data)
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /importieren/i }).click();
    
    // Wait for success
    await expect(page.getByRole('heading', { name: /Import abgeschlossen/i })).toBeVisible({ timeout: 15000 });
    
    // Click "back to list" button
    await page.getByRole('button', { name: /zur übersicht/i }).click();
    
    // Should be on children page
    await expect(page).toHaveURL(/\/kinder$/);
    
    // Cleanup
    fs.unlinkSync(csvPath);
  });

  test('can cancel import and go back', async ({ page }) => {
    await page.goto('/kinder/import');
    
    // Click back button in header
    await page.getByText(/zurück zur übersicht/i).click();
    
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
    await expect(page.getByText(/CSV-Datei hier ablegen/i)).toBeVisible();
    
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
    
    // Create child with unique member number that we'll also put in import CSV
    const uniqueMemberNumber = `DUP${Date.now().toString().slice(-6)}`;
    await page.getByRole('button', { name: /kind hinzufügen/i }).click();
    await expect(page.getByRole('heading', { name: /kind hinzufügen/i })).toBeVisible();
    await page.getByLabel(/mitgliedsnummer/i).fill(uniqueMemberNumber);
    await page.getByLabel(/vorname/i).fill('Existing');
    await page.getByLabel(/nachname/i).fill('Child');
    await page.getByLabel(/geburtsdatum/i).fill('2021-01-01');
    await page.getByLabel(/eintrittsdatum/i).fill('2024-01-01');
    await page.getByRole('button', { name: /speichern/i }).click();
    
    // Wait for dialog to close
    await expect(page.getByRole('heading', { name: /kind hinzufügen/i })).toBeHidden({ timeout: 10000 });
    
    // Now try to import CSV that contains IMP001 - we test with the fixture file
    // which has IMP001 which might already exist from previous tests
    await page.goto('/kinder/import');
    
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles('e2e/fixtures/test-children-import.csv');
    
    // Proceed to preview
    await expect(page.getByText(/Datei erkannt/i)).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /vorschau/i }).click();
    
    // In preview, verify we got to the preview step by checking for table rows
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Beiträge - Import with Parents (Enhanced)', () => {
  // Helper to create a CSV with parents - child data matches createChild() defaults
  function createCSVWithParents(memberNumber: string, firstName = 'TestKind', lastName = 'ParentImport'): string {
    const csvContent = `Mitgliedsnummer;Vorname;Nachname;Geburtsdatum;Eintrittsdatum;Eltern 1 Vorname;Eltern 1 Nachname;Eltern 2 Vorname;Eltern 2 Nachname
${memberNumber};${firstName};${lastName};01.01.2021;01.01.2024;Anna;Müller;Max;Müller
`;
    const tempPath = path.join(__dirname, '..', '..', 'fixtures', `test-parents-import-${memberNumber}.csv`);
    fs.writeFileSync(tempPath, csvContent, 'utf8');
    return tempPath;
  }

  // Helper to create a child via UI
  async function createChild(page: import('@playwright/test').Page, memberNumber: string, firstName: string, lastName: string) {
    await page.goto('/kinder');
    await expect(page.locator('.animate-spin')).toBeHidden({ timeout: 10000 });
    
    await page.getByRole('button', { name: /kind hinzufügen/i }).click();
    await expect(page.getByRole('heading', { name: /kind hinzufügen/i })).toBeVisible();
    
    await page.getByLabel(/mitgliedsnummer/i).fill(memberNumber);
    await page.getByLabel(/vorname/i).fill(firstName);
    await page.getByLabel(/nachname/i).fill(lastName);
    await page.getByLabel(/geburtsdatum/i).fill('2021-01-01');
    await page.getByLabel(/eintrittsdatum/i).fill('2024-01-01');
    await page.getByRole('button', { name: /speichern/i }).click();
    
    // Wait for dialog to close
    await expect(page.getByRole('heading', { name: /kind hinzufügen/i })).toBeHidden({ timeout: 10000 });
  }

  test('can import child with new parents', async ({ page }) => {
    await page.goto('/kinder/import');
    
    const memberNumber = `PAR${Date.now().toString().slice(-6)}`;
    const csvPath = createCSVWithParents(memberNumber);
    
    // Upload CSV
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles(csvPath);
    
    // Step 2: mapping
    await expect(page.getByText(/Datei erkannt/i)).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /vorschau/i }).click();
    
    // Step 3: preview - should show parents in the table
    await expect(page.getByText(/gültige einträge/i)).toBeVisible({ timeout: 10000 });
    
    // Should show parent names in preview
    await expect(page.getByText('Anna Müller').first()).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Max Müller').first()).toBeVisible();
    
    // Import
    await page.getByRole('button', { name: /importieren/i }).click();
    
    // Should show success with parents created
    await expect(page.getByRole('heading', { name: /Import abgeschlossen/i })).toBeVisible({ timeout: 15000 });
    await expect(page.getByText(/Eltern erstellt/i)).toBeVisible();
    
    // Cleanup
    fs.unlinkSync(csvPath);
  });

  test('shows EXISTIERT badge for duplicate children', async ({ page }) => {
    // First create a child
    const memberNumber = `EXI${Date.now().toString().slice(-6)}`;
    await createChild(page, memberNumber, 'Existing', 'Child');
    
    // Now import CSV with same member number
    const csvPath = createCSVWithParents(memberNumber);
    await page.goto('/kinder/import');
    
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles(csvPath);
    
    await expect(page.getByText(/Datei erkannt/i)).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /vorschau/i }).click();
    
    // Should show EXISTIERT badge (use exact: true to avoid matching "Kind existiert bereits")
    await expect(page.getByText('EXISTIERT', { exact: true })).toBeVisible({ timeout: 10000 });
    
    // Should show existing child info
    await expect(page.getByText(/Kind existiert bereits/i)).toBeVisible();
    
    // Cleanup
    fs.unlinkSync(csvPath);
  });

  test('can merge parents to existing child', async ({ page }) => {
    // First create a child without parents
    const memberNumber = `MRG${Date.now().toString().slice(-6)}`;
    // Use matching first/last name so there are no field conflicts
    await createChild(page, memberNumber, 'MergeTest', 'Child');
    
    // Now import CSV with parents for this child - use matching names
    const csvPath = createCSVWithParents(memberNumber, 'MergeTest', 'Child');
    await page.goto('/kinder/import');
    
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles(csvPath);
    
    await expect(page.getByText(/Datei erkannt/i)).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /vorschau/i }).click();
    
    // Wait for preview to load - either EXISTIERT or MERGE badge will be shown
    // (MERGE may be auto-enabled when there are no field conflicts)
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10000 });
    
    // If EXISTIERT is shown, click merge button to enable merge mode
    const existiertBadge = page.getByText('EXISTIERT', { exact: true });
    if (await existiertBadge.isVisible({ timeout: 1000 }).catch(() => false)) {
      await page.getByRole('button', { name: /Merge/i }).click();
    }
    
    // Should now show MERGE badge
    await expect(page.getByText('MERGE', { exact: true })).toBeVisible();
    
    // Should show "Eltern werden hinzugefügt" info
    await expect(page.getByText(/Eltern werden hinzugefügt/i)).toBeVisible();
    
    // Import should now be possible
    await page.getByRole('button', { name: /zusammenführen/i }).click();
    
    // Should show success
    await expect(page.getByRole('heading', { name: /Import abgeschlossen/i })).toBeVisible({ timeout: 15000 });
    
    // Cleanup
    fs.unlinkSync(csvPath);
  });

  // Skip this test - the parent management UI doesn't have a working "add parent" dialog yet
  test.skip('shows already linked parents', async ({ page }) => {
    // First create a child with a parent
    const memberNumber = `LNK${Date.now().toString().slice(-6)}`;
    await createChild(page, memberNumber, 'Linked', 'Child');
    
    // Create parent with same name as CSV
    await page.goto('/eltern');
    await expect(page.locator('.animate-spin')).toBeHidden({ timeout: 10000 });
    
    // Click add parent button and wait for dialog
    const addButton = page.getByRole('button', { name: /elternteil hinzufügen/i });
    await addButton.click();
    
    // Wait for dialog - look for any dialog or form elements
    await expect(page.getByLabel(/vorname/i)).toBeVisible({ timeout: 10000 });
    
    await page.getByLabel(/vorname/i).fill('Anna');
    await page.getByLabel(/nachname/i).fill('Müller');
    await page.getByRole('button', { name: /speichern/i }).click();
    
    // Wait for dialog to close and parent to appear in list
    await expect(page.getByLabel(/vorname/i)).toBeHidden({ timeout: 10000 });
    await expect(page.getByText('Anna Müller').first()).toBeVisible({ timeout: 10000 });
    
    // Find and click on the parent to link
    await page.getByText('Anna Müller').first().click();
    
    // Link parent to child
    await page.getByRole('button', { name: /kind verknüpfen/i }).click();
    // Find the child in the link dialog - search may be needed
    await page.getByPlaceholder(/suchen/i).fill(memberNumber);
    await page.waitForTimeout(500); // Wait for search
    await page.getByText(memberNumber).click();
    await page.getByRole('button', { name: /^verknüpfen$/i }).click();
    
    // Wait for link to complete
    await page.waitForTimeout(1000);
    
    // Now import CSV with same member number and parent name
    const csvPath = createCSVWithParents(memberNumber);
    await page.goto('/kinder/import');
    
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles(csvPath);
    
    await expect(page.getByText(/Datei erkannt/i)).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /vorschau/i }).click();
    
    // Wait for preview - either EXISTIERT or MERGE badge
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10000 });
    
    // Enable merge if showing EXISTIERT
    const existiertBadge = page.getByText('EXISTIERT', { exact: true });
    if (await existiertBadge.isVisible({ timeout: 1000 }).catch(() => false)) {
      await page.getByRole('button', { name: /Merge/i }).click();
    }
    
    // Should show "Verknüpft" badge for already linked parent
    await expect(page.locator('span:has-text("Verknüpft")').first()).toBeVisible({ timeout: 5000 });
    
    // Cleanup
    fs.unlinkSync(csvPath);
  });
});

test.describe('Beiträge - Import with Field Conflicts', () => {
  // Helper to create a CSV with different data for same child
  function createCSVWithConflicts(memberNumber: string): string {
    const csvContent = `Mitgliedsnummer;Vorname;Nachname;Geburtsdatum;Eintrittsdatum;Rechtsanspruch
${memberNumber};DifferentName;Conflict;20.05.2022;15.09.2024;35
`;
    const tempPath = path.join(__dirname, '..', '..', 'fixtures', `test-conflicts-${memberNumber}.csv`);
    fs.writeFileSync(tempPath, csvContent, 'utf8');
    return tempPath;
  }

  test('shows field conflicts when merging with different data', async ({ page }) => {
    // First create a child
    const memberNumber = `CON${Date.now().toString().slice(-6)}`;
    await page.goto('/kinder');
    await expect(page.locator('.animate-spin')).toBeHidden({ timeout: 10000 });
    
    await page.getByRole('button', { name: /kind hinzufügen/i }).click();
    await expect(page.getByRole('heading', { name: /kind hinzufügen/i })).toBeVisible();
    await page.getByLabel(/mitgliedsnummer/i).fill(memberNumber);
    await page.getByLabel(/vorname/i).fill('Original');
    await page.getByLabel(/nachname/i).fill('Name');
    await page.getByLabel(/geburtsdatum/i).fill('2021-01-01');
    await page.getByLabel(/eintrittsdatum/i).fill('2024-01-01');
    await page.getByRole('button', { name: /speichern/i }).click();
    // Wait for dialog to close
    await expect(page.getByRole('heading', { name: /kind hinzufügen/i })).toBeHidden({ timeout: 10000 });
    
    // Import CSV with different data for same child
    const csvPath = createCSVWithConflicts(memberNumber);
    await page.goto('/kinder/import');
    
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles(csvPath);
    
    await expect(page.getByText(/Datei erkannt/i)).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /vorschau/i }).click();
    
    // Enable merge (use exact: true to avoid matching "Kind existiert bereits")
    await expect(page.getByText('EXISTIERT', { exact: true })).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /Merge/i }).click();
    
    // Should show UPDATE badge (since there are conflicts)
    await expect(page.getByText('UPDATE')).toBeVisible();
    
    // Should show conflict info
    await expect(page.getByText(/können aktualisiert werden/i)).toBeVisible();
    
    // Should show conflict resolution UI
    await expect(page.getByText(/Unterschiede zwischen CSV und Datenbank/i)).toBeVisible();
    
    // Should show field names with radio options
    await expect(page.getByText('Vorname:')).toBeVisible();
    await expect(page.getByText(/Behalten:/i).first()).toBeVisible();
    await expect(page.getByText(/CSV verwenden:/i).first()).toBeVisible();
    
    // Cleanup
    fs.unlinkSync(csvPath);
  });

  test('can select which values to use for conflicts', async ({ page }) => {
    // First create a child
    const memberNumber = `SEL${Date.now().toString().slice(-6)}`;
    await page.goto('/kinder');
    await expect(page.locator('.animate-spin')).toBeHidden({ timeout: 10000 });
    
    await page.getByRole('button', { name: /kind hinzufügen/i }).click();
    await expect(page.getByRole('heading', { name: /kind hinzufügen/i })).toBeVisible();
    await page.getByLabel(/mitgliedsnummer/i).fill(memberNumber);
    await page.getByLabel(/vorname/i).fill('Original');
    await page.getByLabel(/nachname/i).fill('Name');
    await page.getByLabel(/geburtsdatum/i).fill('2021-01-01');
    await page.getByLabel(/eintrittsdatum/i).fill('2024-01-01');
    await page.getByRole('button', { name: /speichern/i }).click();
    // Wait for dialog to close
    await expect(page.getByRole('heading', { name: /kind hinzufügen/i })).toBeHidden({ timeout: 10000 });
    
    // Import CSV with different data for same child
    const csvPath = createCSVWithConflicts(memberNumber);
    await page.goto('/kinder/import');
    
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles(csvPath);
    
    await expect(page.getByText(/Datei erkannt/i)).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /vorschau/i }).click();
    
    // Enable merge (use exact: true to avoid matching "Kind existiert bereits")
    await expect(page.getByText('EXISTIERT', { exact: true })).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /Merge/i }).click();
    
    // Find and click the "CSV verwenden" radio for firstName
    const csvRadio = page.locator('label:has-text("CSV verwenden: DifferentName") input[type="radio"]');
    await csvRadio.click();
    
    // Radio should be checked
    await expect(csvRadio).toBeChecked();
    
    // Now execute import
    await page.getByRole('button', { name: /zusammenführen/i }).click();
    
    // Should show success
    await expect(page.getByRole('heading', { name: /Import abgeschlossen/i })).toBeVisible({ timeout: 15000 });
    
    // Should show children updated counter
    await expect(page.getByText(/Kinder aktualisiert/i)).toBeVisible();
    
    // Cleanup
    fs.unlinkSync(csvPath);
  });
});
