import { test as base, expect, type APIRequestContext, type BrowserContext } from '@playwright/test';
import { ADMIN, API } from './env';

/** Short unique suffix so parallel tests never collide on names or emails. */
export function uniq(): string {
  return `${Date.now().toString(36)}${Math.floor(Math.random() * 1e4).toString(36)}`;
}

/** Minimal bearer-token client for seeding data through the real API. */
export class Api {
  constructor(
    private request: APIRequestContext,
    private token: string,
  ) {}

  private async send<T>(method: string, path: string, data?: unknown): Promise<T> {
    const res = await this.request.fetch(`${API}${path}`, {
      method,
      data,
      headers: { Authorization: `Bearer ${this.token}` },
    });
    expect(res.ok(), `${method} ${path} → ${res.status()} ${await res.text()}`).toBeTruthy();
    return res.status() === 204 ? (undefined as T) : ((await res.json()) as T);
  }

  get<T>(path: string) {
    return this.send<T>('GET', path);
  }
  post<T>(path: string, data?: unknown) {
    return this.send<T>('POST', path, data);
  }
  put<T>(path: string, data?: unknown) {
    return this.send<T>('PUT', path, data);
  }
}

export interface TestUser {
  id: string;
  email: string;
  password: string;
}

/**
 * Logs a browser context in through the API. The httpOnly refresh cookie lands
 * in the context's cookie jar, so the app restores the session on page load
 * exactly as after a real login. (A shared storageState would not work: the
 * refresh token rotates, so a saved cookie is dead after its first use.)
 */
export async function loginContext(context: BrowserContext, email: string, password: string): Promise<string> {
  const res = await context.request.post(`${API}/auth/login`, { data: { email, password } });
  expect(res.ok(), `login ${email} → ${res.status()}`).toBeTruthy();
  return (await res.json()).accessToken as string;
}

type Fixtures = {
  /** Page of a browser context logged in as the bootstrapped admin. */
  adminPage: import('@playwright/test').Page;
  /** API client authenticated as admin, for seeding. */
  adminApi: Api;
  /** Creates an active account with role USER (or ADMIN) and returns its credentials. */
  createUser: (role?: 'USER' | 'ADMIN') => Promise<TestUser>;
};

export const test = base.extend<Fixtures>({
  adminPage: async ({ page, context }, use) => {
    await loginContext(context, ADMIN.email, ADMIN.password);
    await use(page);
  },

  adminApi: async ({ playwright, baseURL }, use) => {
    const request = await playwright.request.newContext({ baseURL });
    const res = await request.post(`${API}/auth/login`, { data: ADMIN });
    expect(res.ok(), `admin login → ${res.status()}`).toBeTruthy();
    await use(new Api(request, (await res.json()).accessToken));
    await request.dispose();
  },

  createUser: async ({ adminApi }, use) => {
    await use(async (role = 'USER') => {
      const email = `user-${uniq()}@e2e.test`;
      const password = `pw-${uniq()}`;
      const user = await adminApi.post<{ id: string }>('/users', {
        email,
        firstName: 'E2E',
        lastName: role === 'ADMIN' ? 'Admin' : 'Nutzer',
        role,
        isActive: true,
        password,
      });
      return { id: user.id, email, password };
    });
  },
});

export { expect };
