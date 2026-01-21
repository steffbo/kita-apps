import { test as setup, expect } from '@playwright/test';

const authFile = 'e2e/.auth/beitraege.json';

/**
 * Authentication setup for Beiträge app
 * Uses separate JWT auth system on port 8081
 */
setup('authenticate beitraege', async ({ page, context }) => {
  await page.goto('http://localhost:5175/login');
  
  // Fill in credentials
  await page.getByLabel(/e-mail/i).fill('admin@knirpsenstadt.de');
  await page.getByLabel(/passwort/i).fill('admin123');
  
  // Click login button
  await page.getByRole('button', { name: /anmelden/i }).click();
  
  // Wait for successful login - should redirect to dashboard
  await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 });
  
  // Verify we're logged in by checking for navigation element
  await expect(page.getByRole('navigation').first()).toBeVisible();
  
  // Save authentication state
  await context.storageState({ path: authFile });
});
