// Shared settings between playwright.config.ts, the stack script and the tests.
export const E2E_PORT = Number(process.env.E2E_PORT ?? 18081);

/** Bootstrapped by start-beitraege-stack.sh via USER_NAME/USER_PASSWORD. */
export const ADMIN = {
  email: process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test',
  password: process.env.E2E_ADMIN_PASSWORD ?? 'e2e-admin-password',
};

export const API = '/api/fees/v1';
