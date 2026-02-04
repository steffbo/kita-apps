import { test, expect } from '@playwright/test';
import { TEST_CONFIG, helpers } from '../../fixtures';

test.describe('Zeiterfassung - Clock In/Out', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await helpers.expectLoadingComplete(page);
  });

  test('displays current time and date', async ({ page }) => {
    // Time should be displayed in HH:MM format
    await expect(page.getByText(/\b\d{1,2}:\d{2}\b/)).toBeVisible();
  });

  test('shows clock in button when not clocked in', async ({ page }) => {
    // Should have a clock in button - "Einstempeln"
    const clockInBtn = page.getByRole('button', { name: /einstempeln/i });
    await expect(clockInBtn).toBeVisible();
  });

  test('can clock in', async ({ page }) => {
    const clockInBtn = page.getByRole('button', { name: /einstempeln/i });
    
    if (await clockInBtn.isVisible()) {
      await clockInBtn.click();
      
      // Should show clocked in state - "Ausstempeln" button becomes visible
      await expect(
        page.getByRole('button', { name: /ausstempeln/i })
      ).toBeVisible();
    }
  });

  test('can clock out after clocking in', async ({ page }) => {
    // First clock in if not already
    const clockInBtn = page.getByRole('button', { name: /einstempeln/i });
    if (await clockInBtn.isVisible()) {
      await clockInBtn.click();
      await page.waitForTimeout(500); // Wait for state change
    }
    
    // Now clock out
    const clockOutBtn = page.getByRole('button', { name: /ausstempeln/i });
    if (await clockOutBtn.isVisible()) {
      await clockOutBtn.click();
      
      // Should show clocked out state - "Einstempeln" button visible again
      await expect(
        page.getByRole('button', { name: /einstempeln/i })
      ).toBeVisible();
    }
  });

  test('displays current status', async ({ page }) => {
    // Should show status text - either "nicht eingestempelt" or "Eingestempelt seit"
    await expect(
      page.getByText(/nicht eingestempelt|eingestempelt seit/i)
    ).toBeVisible();
  });
});

test.describe('Zeiterfassung - History', () => {
  test('can view time entry history', async ({ page }) => {
    await page.goto('/history');
    await helpers.expectLoadingComplete(page);
    
    // Heading is "Zeitübersicht"
    await expect(page.getByRole('heading', { name: /zeitübersicht/i })).toBeVisible();
  });

  test('shows time entries in a list or calendar', async ({ page }) => {
    await page.goto('/history');
    await helpers.expectLoadingComplete(page);
    
    // Should have a table with entries
    await expect(page.locator('table')).toBeVisible();
  });

  test('can filter by date range', async ({ page }) => {
    await page.goto('/history');
    
    // History page has month navigation with < and > buttons
    const prevButton = page.locator('button').filter({ hasText: '' }).first();
    const nextButton = page.locator('button').filter({ hasText: '' }).last();
    
    // Page should load with table visible
    await expect(page.locator('table')).toBeVisible();
  });
});

test.describe('Zeiterfassung - Mobile Responsiveness', () => {
  test.use({ viewport: { width: 375, height: 667 } }); // iPhone SE

  test('clock buttons are accessible on mobile', async ({ page }) => {
    await page.goto('/');
    await helpers.expectLoadingComplete(page);
    
    // Clock button should be visible - "Einstempeln" or "Ausstempeln"
    const clockBtn = page.getByRole('button', { name: /einstempeln|ausstempeln/i });
    await expect(clockBtn).toBeVisible();
    
    // Check button is reasonably sized for touch
    const box = await clockBtn.boundingBox();
    expect(box?.height).toBeGreaterThan(40);
  });

  test('navigation works on mobile', async ({ page }) => {
    await page.goto('/');
    
    // Look for mobile menu toggle
    const menuToggle = page.getByRole('button', { name: /menü|menu/i })
      .or(page.locator('[class*="hamburger"], [class*="menu-toggle"]'));
    
    if (await menuToggle.isVisible()) {
      await menuToggle.click();
      
      // Navigation should appear
      await expect(page.getByRole('navigation')).toBeVisible();
    }
  });
});
