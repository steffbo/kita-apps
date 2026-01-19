import { test, expect, helpers } from '../../fixtures';

test.describe('Groups Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/groups');
    await helpers.expectLoadingComplete(page);
  });

  test('displays group list with default groups', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /gruppen/i })).toBeVisible();
    
    // Wait for groups to load from API - look for group cards
    // Groups are displayed in cards with the group name as h3
    const groupGrid = page.locator('.grid');
    await expect(groupGrid).toBeVisible();
    
    // Should have at least one group card visible after loading
    const groupCards = page.locator('.bg-white.rounded-lg.border');
    await expect(groupCards.first()).toBeVisible({ timeout: 10000 });
  });

  test('groups have color indicators', async ({ page }) => {
    // Wait for groups to load
    const groupCards = page.locator('.bg-white.rounded-lg.border');
    await expect(groupCards.first()).toBeVisible({ timeout: 10000 });
    
    // Groups should have colored circle elements
    const colorIndicator = page.locator('.rounded-full').first();
    await expect(colorIndicator).toBeVisible();
  });

  test('can open create group dialog', async ({ page }) => {
    await page.getByRole('button', { name: /neue gruppe/i }).click();
    
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByLabel(/name/i)).toBeVisible();
  });

  test('can create a new group', async ({ page }) => {
    await page.getByRole('button', { name: /neue gruppe/i }).click();
    
    // Fill form
    const groupName = `Testgruppe ${Date.now()}`;
    await page.getByLabel(/name/i).fill(groupName);
    
    // Set description if available
    const descField = page.getByLabel(/beschreibung/i);
    if (await descField.isVisible()) {
      await descField.fill('Eine Testgruppe');
    }
    
    // Submit
    await page.getByRole('button', { name: /erstellen|speichern/i }).click();
    
    // Dialog should close
    await expect(page.getByRole('dialog')).toBeHidden();
    
    // New group should appear - wait for API response
    await expect(page.getByText(groupName)).toBeVisible({ timeout: 10000 });
  });

  test('can edit a group', async ({ page }) => {
    // Wait for groups to load
    const groupCards = page.locator('.bg-white.rounded-lg.border');
    await expect(groupCards.first()).toBeVisible({ timeout: 10000 });
    
    // Click the first "Bearbeiten" button
    await page.getByRole('button', { name: /bearbeiten/i }).first().click();
    
    // Dialog should open with existing data
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByLabel(/name/i)).toBeVisible();
  });

  test('can delete a group', async ({ page }) => {
    // First create a test group to delete
    await page.getByRole('button', { name: /neue gruppe/i }).click();
    const groupName = `Delete-Test-${Date.now()}`;
    await page.getByLabel(/name/i).fill(groupName);
    await page.getByRole('button', { name: /erstellen|speichern/i }).click();
    await expect(page.getByRole('dialog')).toBeHidden();
    
    // Wait for group to appear
    await expect(page.getByText(groupName)).toBeVisible({ timeout: 10000 });
    
    // Find the card with this group and click delete
    const groupCard = page.locator('.bg-white.rounded-lg.border').filter({ hasText: groupName });
    await groupCard.getByRole('button', { name: /löschen/i }).click();
    
    // Confirm deletion in the modal
    const confirmDialog = page.locator('.fixed.inset-0.z-50');
    await expect(confirmDialog).toBeVisible();
    await confirmDialog.getByRole('button', { name: /löschen/i }).click();
    
    // Wait for dialog to close and group to be removed
    await expect(confirmDialog).toBeHidden({ timeout: 10000 });
    await expect(page.getByText(groupName)).toBeHidden({ timeout: 10000 });
  });
});

test.describe('Special Days Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/special-days');
    await helpers.expectLoadingComplete(page);
  });

  test('displays holidays for current year', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /besondere tage/i })).toBeVisible();
    
    // Should show holidays section - look for the heading
    await expect(page.getByRole('heading', { name: /feiertage brandenburg/i })).toBeVisible();
    
    // Should have some holidays displayed (or a message about no holidays)
    const holidaySection = page.locator('.bg-white.rounded-lg.border').filter({ hasText: /feiertage brandenburg/i });
    await expect(holidaySection).toBeVisible();
  });

  test('can navigate between years', async ({ page }) => {
    // Get current year from the page - use exact match with the year span
    const currentYear = new Date().getFullYear();
    const yearDisplay = page.locator('.font-semibold').filter({ hasText: currentYear.toString() });
    await expect(yearDisplay).toBeVisible();
    
    // Navigate to next year using the > button
    await page.getByRole('button', { name: '>' }).click();
    
    // Year should change - wait for it
    await expect(page.locator('.font-semibold').filter({ hasText: (currentYear + 1).toString() })).toBeVisible();
  });

  test('can create a closure day', async ({ page }) => {
    await page.getByRole('button', { name: /neuer eintrag/i }).click();
    
    await expect(page.getByRole('dialog')).toBeVisible();
    
    // Fill form - use unique date and name to avoid conflicts
    const uniqueSuffix = Date.now();
    const uniqueDate = `2027-08-${String(Math.floor(Math.random() * 28) + 1).padStart(2, '0')}`;
    const closureName = `Test Schließzeit ${uniqueSuffix}`;
    
    await page.getByLabel(/datum/i).fill(uniqueDate);
    await page.getByLabel(/bezeichnung/i).fill(closureName);
    
    // Submit - find the dialog's submit button
    await page.getByRole('dialog').getByRole('button', { name: /erstellen/i }).click();
    
    // Wait for the mutation to complete and dialog to close
    await expect(page.getByRole('dialog')).toBeHidden({ timeout: 10000 });
    
    // Navigate to the year 2027 to see the entry
    await page.getByRole('button', { name: '>' }).click(); // Go to next year (2027)
    
    // Should appear in the list
    await expect(page.getByText(closureName)).toBeVisible({ timeout: 10000 });
  });

  test('shows different day types in sections', async ({ page }) => {
    // Should have sections for different types - check the headings
    await expect(page.getByRole('heading', { name: /feiertage brandenburg/i })).toBeVisible();
    await expect(page.getByRole('heading', { name: /schließzeiten/i })).toBeVisible();
    await expect(page.getByRole('heading', { name: /teamtage.*bildungstage/i })).toBeVisible();
    await expect(page.getByRole('heading', { name: /veranstaltungen/i })).toBeVisible();
  });
});
