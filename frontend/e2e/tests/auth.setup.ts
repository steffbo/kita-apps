import { test as setup, expect } from '@playwright/test';
import { TEST_CONFIG } from '../fixtures';

const authFile = 'e2e/.auth/user.json';

/**
 * Authentication setup - runs before all other tests
 * Saves authentication state to be reused by other tests
 * 
 * Authenticates for both apps (dienstplan and zeiterfassung) since they
 * run on different ports and localStorage is origin-specific.
 */
setup('authenticate', async ({ page, context }) => {
  // === Authenticate Dienstplan App (port 5173) ===
  await page.goto('http://localhost:5173/login');
  
  // Fill in credentials
  await page.getByLabel(/e-mail/i).fill(TEST_CONFIG.testUser.email);
  await page.getByLabel(/passwort/i).fill(TEST_CONFIG.testUser.password);
  
  // Click login button
  await page.getByRole('button', { name: /anmelden/i }).click();
  
  // Wait for successful login - should redirect to dashboard or schedule
  await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 });
  
  // Verify we're logged in by checking for navigation element
  await expect(page.getByRole('navigation').first()).toBeVisible();
  
  // === Authenticate Zeiterfassung App (port 5174) ===
  await page.goto('http://localhost:5174/login');
  
  // Fill in credentials
  await page.getByLabel(/e-mail/i).fill(TEST_CONFIG.testUser.email);
  await page.getByLabel(/passwort/i).fill(TEST_CONFIG.testUser.password);
  
  // Click login button
  await page.getByRole('button', { name: /anmelden/i }).click();
  
  // Wait for successful login
  await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 });
  
  // Save authentication state for both origins
  await context.storageState({ path: authFile });
});
