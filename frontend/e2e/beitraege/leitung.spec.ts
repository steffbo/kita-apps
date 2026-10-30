import { test, expect, uniq, loginContext } from './fixtures';
import { API } from './env';

/** Unique member number within the varchar(10) column. */
function memberNumber() {
  return String(10_000_000 + Math.floor(Math.random() * 90_000_000));
}

const INCOME = 48000;

/** A household with known income and one child, seeded as admin. */
async function seedFamily(adminApi: import('./fixtures').Api) {
  const suffix = uniq();
  const household = await adminApi.post<{ id: string }>('/households', {
    name: `Familie Leitung ${suffix}`, annualHouseholdIncome: INCOME, incomeStatus: 'PROVIDED',
  });
  const child = await adminApi.post<{ id: string }>('/children', {
    memberNumber: memberNumber(), firstName: 'Kind', lastName: suffix,
    birthDate: '2021-03-01', entryDate: '2023-08-01',
  });
  await adminApi.post(`/households/${household.id}/children`, { childId: child.id });
  return { household, child, suffix, childName: `Kind ${suffix}` };
}

test('leitung sees master data and Elternstunden but no fee areas', async ({ browser, createUser, adminApi }) => {
  const { child, childName } = await seedFamily(adminApi);
  const user = await createUser('LEITUNG');
  const context = await browser.newContext();
  try {
    await loginContext(context, user.email, user.password);
    const page = await context.newPage();

    await page.goto('/beitraege/');
    await expect(page).toHaveURL(/\/beitraege\/kinder$/);
    const navigation = page.getByRole('navigation', { name: 'Hauptnavigation' });
    for (const name of ['Kinder', 'Eltern', 'Mitglieder', 'Verwaltung']) {
      await expect(navigation.getByRole('link', { name, exact: true })).toBeVisible();
    }
    for (const name of ['Notizen', 'Einstufungen', 'Beitragsordnung', 'Erinnerungen', 'Benutzer', 'Vorstand']) {
      await expect(navigation.getByRole('link', { name, exact: true })).toHaveCount(0);
    }
    for (const group of ['Aufgaben', 'Beiträge', 'System']) {
      await expect(navigation.getByText(group, { exact: true })).toHaveCount(0);
    }

    // Fee pages bounce back to the child list.
    for (const path of ['einstufungen', 'bankabgleich', 'notizen', 'beitraege', 'benutzer']) {
      await page.goto(`/beitraege/${path}`);
      await expect(page).toHaveURL(/\/beitraege\/kinder$/);
    }

    // The child page shows the household but no income, fee or note sections.
    await page.goto(`/beitraege/kinder/${child.id}`);
    await expect(page.getByRole('heading', { name: childName })).toBeVisible();
    await expect(page.getByText('Haushaltsname')).toBeVisible();
    for (const text of ['Einkommensstatus', 'Jahreshaushaltseinkommen', 'Platzgeld', 'Einstufung erstellen']) {
      await expect(page.getByText(text)).toHaveCount(0);
    }
  } finally { await context.close(); }
});

test('leitung api hides income and rejects fee, bank, note and delete routes', async ({ browser, createUser, adminApi }) => {
  const { household, child, suffix } = await seedFamily(adminApi);
  const user = await createUser('LEITUNG');
  const context = await browser.newContext();
  try {
    const token = await loginContext(context, user.email, user.password);
    const headers = { Authorization: `Bearer ${token}` };

    // Master data is readable, without income or fee fields at any depth.
    const householdRes = await context.request.get(`${API}/households/${household.id}`, { headers });
    expect(householdRes.status()).toBe(200);
    const householdBody = await householdRes.text();
    expect(householdBody).toContain(household.id);
    expect(householdBody).not.toMatch(/annualHouseholdIncome|incomeStatus|childrenCountForFees/);

    const childRes = await context.request.get(`${API}/children/${child.id}`, { headers });
    expect(childRes.status()).toBe(200);
    expect(await childRes.text()).not.toMatch(/annualHouseholdIncome|incomeStatus|openFeesCount/);

    // The open-fees filter is ignored, so the child still shows up.
    const listRes = await context.request.get(`${API}/children?search=${suffix}&hasOpenFees=true`, { headers });
    expect(listRes.status()).toBe(200);
    const listBody = await listRes.text();
    expect(listBody).toContain(child.id);
    expect(listBody).not.toMatch(/openFeesCount|annualHouseholdIncome|incomeStatus/);

    // Fee, Einstufung, bank and note data is off limits, and so is deleting.
    for (const path of [
      '/fees', '/fees/overview', '/einstufungen', '/fee-schedules', '/stichtagsmeldung/stats',
      '/import/transactions', '/import/history', '/banking-sync/status', '/notes',
      `/children/${child.id}/ledger`, `/children/${child.id}/timeline?year=2026`, `/children/${child.id}/notes`,
    ]) {
      const res = await context.request.get(`${API}${path}`, { headers });
      expect(res.status(), `GET ${path}`).toBe(403);
    }
    for (const path of [`/children/${child.id}`, `/households/${household.id}`]) {
      const res = await context.request.delete(`${API}${path}`, { headers });
      expect(res.status(), `DELETE ${path}`).toBe(403);
    }

    // Edits go through, but income sent along is ignored.
    const update = await context.request.put(`${API}/households/${household.id}`, {
      headers, data: { name: 'Familie umbenannt', annualHouseholdIncome: 1, incomeStatus: 'MAX_ACCEPTED' },
    });
    expect(update.status()).toBe(200);
    const stored = await adminApi.get<{ name: string; annualHouseholdIncome: number; incomeStatus: string }>(
      `/households/${household.id}`);
    expect(stored.name).toBe('Familie umbenannt');
    expect(stored.annualHouseholdIncome).toBe(INCOME);
    expect(stored.incomeStatus).toBe('PROVIDED');
  } finally { await context.close(); }
});
