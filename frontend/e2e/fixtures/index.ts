import { test as base, expect } from '@playwright/test';

/**
 * Test configuration
 */
export const TEST_CONFIG = {
  /** Backend API URL */
  apiUrl: process.env.API_URL || 'http://localhost:8080/api',
  
  /** Default test user credentials */
  testUser: {
    email: 'admin@knirpsenstadt.de',
    password: 'admin123',
  },
  
  /** Timeouts */
  timeout: {
    navigation: 30000,
    api: 10000,
  },
};

/**
 * Custom test fixtures extending Playwright's base test
 */
export interface TestFixtures {
  /** Login helper function */
  login: (email?: string, password?: string) => Promise<void>;
  
  /** API request helper */
  apiRequest: (endpoint: string, options?: RequestInit) => Promise<Response>;
}

/**
 * Extended test with custom fixtures
 */
export const test = base.extend<TestFixtures>({
  /**
   * Login fixture - handles authentication via UI
   */
  login: async ({ page }, use) => {
    const loginFn = async (
      email = TEST_CONFIG.testUser.email,
      password = TEST_CONFIG.testUser.password
    ) => {
      await page.goto('/login');
      await page.getByLabel(/e-mail/i).fill(email);
      await page.getByLabel(/passwort/i).fill(password);
      await page.getByRole('button', { name: /anmelden/i }).click();
      
      // Wait for navigation away from login page
      await expect(page).not.toHaveURL(/\/login/);
    };
    
    await use(loginFn);
  },

  /**
   * API request fixture - makes authenticated API calls
   */
  apiRequest: async ({ context }, use) => {
    const requestFn = async (endpoint: string, options?: RequestInit) => {
      const url = `${TEST_CONFIG.apiUrl}${endpoint}`;
      const cookies = await context.cookies();
      const authCookie = cookies.find(c => c.name === 'auth-token');
      
      return fetch(url, {
        ...options,
        headers: {
          'Content-Type': 'application/json',
          ...(authCookie && { Authorization: `Bearer ${authCookie.value}` }),
          ...options?.headers,
        },
      });
    };
    
    await use(requestFn);
  },
});

export { expect };

/**
 * Page Object Models for common UI elements
 */
export class NavigationHelper {
  constructor(private page: base['page']) {}

  async goToEmployees() {
    await this.page.getByRole('link', { name: /mitarbeiter/i }).click();
    await expect(this.page).toHaveURL(/\/employees/);
  }

  async goToSchedule() {
    await this.page.getByRole('link', { name: /dienstplan/i }).click();
    await expect(this.page).toHaveURL(/\/schedule/);
  }

  async goToGroups() {
    await this.page.getByRole('link', { name: /gruppen/i }).click();
    await expect(this.page).toHaveURL(/\/groups/);
  }

  async goToStatistics() {
    await this.page.getByRole('link', { name: /statistiken/i }).click();
    await expect(this.page).toHaveURL(/\/statistics/);
  }

  async goToSpecialDays() {
    await this.page.getByRole('link', { name: /besondere tage/i }).click();
    await expect(this.page).toHaveURL(/\/special-days/);
  }

  async logout() {
    await this.page.getByRole('button', { name: /abmelden/i }).click();
    await expect(this.page).toHaveURL(/\/login/);
  }
}

/**
 * Common assertions and helpers
 */
export const helpers = {
  /**
   * Wait for API response
   */
  async waitForApi(page: base['page'], urlPattern: string | RegExp) {
    return page.waitForResponse(
      response => 
        (typeof urlPattern === 'string' 
          ? response.url().includes(urlPattern) 
          : urlPattern.test(response.url())) &&
        response.status() === 200
    );
  },

  /**
   * Fill a form field by label
   */
  async fillField(page: base['page'], label: string, value: string) {
    await page.getByLabel(label).fill(value);
  },

  /**
   * Select an option from a dropdown
   */
  async selectOption(page: base['page'], label: string, value: string) {
    await page.getByLabel(label).selectOption(value);
  },

  /**
   * Assert toast notification appears
   */
  async expectToast(page: base['page'], message: string | RegExp) {
    await expect(page.getByRole('alert').filter({ hasText: message })).toBeVisible();
  },

  /**
   * Assert loading spinner is visible then hidden
   */
  async expectLoadingComplete(page: base['page']) {
    // Wait for any loading indicators to appear and then disappear
    const loader = page.locator('[class*="animate-spin"]');
    if (await loader.isVisible()) {
      await expect(loader).toBeHidden({ timeout: 10000 });
    }
  },

  /**
   * Format date for input fields (YYYY-MM-DD)
   */
  formatDateForInput(date: Date): string {
    return date.toISOString().split('T')[0];
  },

  /**
   * Get Monday of the current week
   */
  getWeekStart(date = new Date()): Date {
    const d = new Date(date);
    const day = d.getDay();
    const diff = d.getDate() - day + (day === 0 ? -6 : 1);
    return new Date(d.setDate(diff));
  },
};
