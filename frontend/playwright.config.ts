import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright configuration for Kita Apps E2E tests
 * @see https://playwright.dev/docs/test-configuration
 */
export default defineConfig({
  testDir: './e2e/tests',
  
  /* Run tests in files in parallel */
  fullyParallel: true,
  
  /* Fail the build on CI if you accidentally left test.only in the source code */
  forbidOnly: !!process.env.CI,
  
  /* Retry on CI only */
  retries: process.env.CI ? 2 : 0,
  
  /* Opt out of parallel tests on CI */
  workers: process.env.CI ? 1 : undefined,
  
  /* Reporter to use */
  reporter: [
    ['html', { outputFolder: 'playwright-report' }],
    ['list'],
  ],
  
  /* Shared settings for all projects */
  use: {
    /* Base URL for navigation actions */
    baseURL: process.env.BASE_URL || 'http://localhost:5173',

    /* Collect trace when retrying the failed test */
    trace: 'on-first-retry',
    
    /* Capture screenshot on failure */
    screenshot: 'only-on-failure',
    
    /* Video recording on failure */
    video: 'on-first-retry',
  },

  /* Configure projects for different apps */
  projects: [
    /* Setup project - runs authentication */
    {
      name: 'setup',
      testMatch: '**/*.setup.ts',
    },
    
    /* Dienstplan App Tests */
    {
      name: 'dienstplan',
      use: { 
        ...devices['Desktop Chrome'],
        baseURL: 'http://localhost:5173',
        storageState: 'e2e/.auth/user.json',
      },
      dependencies: ['setup'],
      testMatch: 'dienstplan/**/*.spec.ts',
    },
    
    /* Zeiterfassung App Tests */
    {
      name: 'zeiterfassung',
      use: { 
        ...devices['Desktop Chrome'],
        baseURL: 'http://localhost:5174',
        storageState: 'e2e/.auth/user.json',
      },
      dependencies: ['setup'],
      testMatch: 'zeiterfassung/**/*.spec.ts',
    },

    /* Beiträge App Tests */
    {
      name: 'beitraege',
      use: { 
        ...devices['Desktop Chrome'],
        baseURL: 'http://localhost:5175',
        storageState: 'e2e/.auth/beitraege.json',
      },
      dependencies: ['beitraege-setup'],
      testMatch: 'beitraege/**/*.spec.ts',
    },
    
    /* Beiträge Auth Setup */
    {
      name: 'beitraege-setup',
      testMatch: '**/beitraege.setup.ts',
    },
    
    /* Mobile viewport tests */
    {
      name: 'mobile',
      use: { 
        ...devices['iPhone 13'],
        baseURL: 'http://localhost:5173',
        storageState: 'e2e/.auth/user.json',
      },
      dependencies: ['setup'],
      testMatch: 'mobile/**/*.spec.ts',
    },
  ],

  /* Run local dev servers before starting the tests */
  webServer: [
    {
      command: 'bun run dev:plan',
      url: 'http://localhost:5173',
      reuseExistingServer: !process.env.CI,
      timeout: 120 * 1000,
    },
    {
      command: 'bun run dev:zeit',
      url: 'http://localhost:5174',
      reuseExistingServer: !process.env.CI,
      timeout: 120 * 1000,
    },
    {
      command: 'bun run dev:beitraege',
      url: 'http://localhost:5175',
      reuseExistingServer: !process.env.CI,
      timeout: 120 * 1000,
    },
  ],
});
