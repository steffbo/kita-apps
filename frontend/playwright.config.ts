import { defineConfig, devices } from '@playwright/test';
import { E2E_PORT } from './e2e/beitraege/env';

/**
 * Playwright E2E suite for the Beiträge app (e2e/beitraege/).
 *
 * `webServer` runs e2e/start-beitraege-stack.sh: a disposable PostgreSQL, the
 * production build (frontend embedded into backend-fees) and a bootstrapped
 * admin. Every run starts with an empty database; tests create their own data
 * with unique names, so they can run in parallel.
 *
 * Legacy Dienstplan/Zeiterfassung tests: playwright.management.config.ts.
 */
export default defineConfig({
  testDir: './e2e/beitraege',
  outputDir: 'test-results/beitraege',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: process.env.CI ? 2 : undefined,
  timeout: 60_000,
  reporter: process.env.CI
    ? [['list'], ['html', { outputFolder: 'playwright-report', open: 'never' }]]
    : [['list'], ['html', { outputFolder: 'playwright-report', open: 'on-failure' }]],

  use: {
    ...devices['Desktop Chrome'],
    baseURL: `http://127.0.0.1:${E2E_PORT}`,
    locale: 'de-DE',
    timezoneId: 'Europe/Berlin',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },

  webServer: {
    command: 'bash e2e/start-beitraege-stack.sh',
    url: `http://127.0.0.1:${E2E_PORT}/health`,
    // Building frontend + backend takes a while on a cold cache.
    timeout: 300_000,
    reuseExistingServer: false,
    // SIGTERM (default is SIGKILL) lets the script remove the DB container.
    gracefulShutdown: { signal: 'SIGTERM', timeout: 15_000 },
    stdout: 'pipe',
    stderr: 'pipe',
    env: { E2E_PORT: String(E2E_PORT) },
  },
});
