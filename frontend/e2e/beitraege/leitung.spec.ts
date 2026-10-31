import { test, expect, uniq, loginContext } from './fixtures';
import { API } from './env';
import { berlinToday } from './bank';

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

test('leitung sees the Stichtag dashboard, master data and Elternstunden but no fee areas', async ({ browser, createUser, adminApi }) => {
  const { child, childName, suffix } = await seedFamily(adminApi);
  const user = await createUser('LEITUNG');
  const context = await browser.newContext();
  try {
    await loginContext(context, user.email, user.password);
    const page = await context.newPage();

    await page.goto('/beitraege/');
    await expect(page).toHaveURL(/\/beitraege\/$/);
    const navigation = page.getByRole('navigation', { name: 'Hauptnavigation' });
    for (const name of ['Start', 'Kinder', 'Eltern', 'Mitglieder', 'Übersicht']) {
      await expect(navigation.getByRole('link', { name, exact: true })).toBeVisible();
    }
    for (const name of ['Notizen', 'Einstufungen', 'Beitragsordnung', 'Erinnerungen', 'Benutzer', 'Vorstand',
      'Verwaltung']) {
      await expect(navigation.getByRole('link', { name, exact: true })).toHaveCount(0);
    }
    for (const group of ['Aufgaben', 'Beiträge', 'System']) {
      await expect(navigation.getByText(group, { exact: true })).toHaveCount(0);
    }

    // Fee pages bounce back to the Leitung dashboard.
    for (const path of ['einstufungen', 'bankabgleich', 'notizen', 'beitraege', 'benutzer']) {
      await page.goto(`/beitraege/${path}`);
      await expect(page).toHaveURL(/\/beitraege\/$/);
    }

    // The dashboard lists the enrolled children for the Stichtagsmeldung; the seeded child is Ü3.
    const report = page.getByRole('region', { name: 'Stichtagsmeldung' });
    await expect(report.getByText('Stichtagsmeldung Landkreis')).toBeVisible();
    const list = page.getByRole('region', { name: 'Kinder zum Stichtag' });
    const childRow = list.getByRole('row', { name: new RegExp(suffix) });
    await expect(childRow).toBeVisible();
    await report.getByRole('button', { name: /^U3 \(Krippe\)/ }).click();
    await expect(childRow).toHaveCount(0);
    await report.getByRole('button', { name: /^Ü3 \(Kindergarten\)/ }).click();
    await expect(childRow).toBeVisible();
    const stichtag = await report.getByTestId('stichtag').textContent();
    await report.getByRole('button', { name: 'Vorheriger Stichtag' }).click();
    await expect(report.getByTestId('stichtag')).not.toHaveText(stichtag ?? '');
    await expect(report.getByTestId('stichtag')).toHaveText(/^01\.(03|06|09|12)\./);

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

    // The Stichtagsmeldung to the Landkreis is open; a child born 24.07.2020 is still in the Kita
    // on 01.09.2026 although school starts for it in 2026, so it counts as zurückgestellt.
    const deferred = await adminApi.post<{ id: string }>('/children', {
      memberNumber: memberNumber(), firstName: 'Rücksteller', lastName: suffix,
      birthDate: '2020-07-24', entryDate: '2023-08-01',
    });
    const jugendamtRes = await context.request.get(`${API}/stichtagsmeldung/jugendamt?date=2026-09-01`, { headers });
    expect(jugendamtRes.status()).toBe(200);
    const jugendamt = await jugendamtRes.json() as {
      stichtag: string; lastYear: number; deferred: number; children: { id: string; group: string }[] };
    expect(jugendamt.stichtag).toBe('2026-09-01');
    expect(jugendamt.children.find(c => c.id === deferred.id)?.group).toBe('DEFERRED');
    expect(jugendamt.children.find(c => c.id === child.id)?.group).toBe('LAST_YEAR');
    expect(jugendamt.lastYear).toBeGreaterThanOrEqual(jugendamt.deferred + 1);

    // Fee, Einstufung, bank and note data is off limits, and so is deleting.
    for (const path of ['/stichtagsmeldung/report?date=2026-09-01', '/stichtagsmeldung/children']) {
      const res = await context.request.get(`${API}${path}`, { headers });
      expect(res.status(), `GET ${path}`).toBe(403);
    }
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

test('leitung sees parent-work accounts and records hours but cannot correct them', async ({ browser, createUser, adminApi }) => {
  const { household, suffix } = await seedFamily(adminApi);
  const householdName = `Familie Leitung ${suffix}`;
  const today = berlinToday();
  const workDate = `${today.year}-${String(today.month).padStart(2, '0')}-${String(today.day).padStart(2, '0')}`;
  const entry = await adminApi.post<{ id: string }>('/parent-work/entries', {
    householdId: household.id, workDate, durationMinutes: 60, occasion: 'E2E Admin-Eintrag',
  });
  const user = await createUser('LEITUNG');
  const context = await browser.newContext();
  try {
    const token = await loginContext(context, user.email, user.password);
    const page = await context.newPage();

    await page.goto('/beitraege/elternstunden/verwaltung');
    await expect(page).toHaveURL(/\/beitraege\/elternstunden$/);

    await page.goto(`/beitraege/elternstunden/familien/${household.id}`);
    await expect(page.getByRole('heading', { name: householdName })).toBeVisible();
    await expect(page.getByRole('row', { name: /E2E Admin-Eintrag/ })).toBeVisible();
    for (const name of ['Manuelles Soll setzen', 'Bearbeiten', 'Stornieren']) {
      await expect(page.getByRole('button', { name })).toHaveCount(0);
    }
    await expect(page.getByRole('columnheader', { name: 'Aktionen' })).toHaveCount(0);

    await page.getByRole('button', { name: 'Stunden erfassen' }).click();
    const dialog = page.getByRole('dialog', { name: 'Stunden erfassen' });
    await expect(dialog.getByText(householdName, { exact: true })).toBeVisible();
    await dialog.getByRole('button', { name: '2 Std.', exact: true }).click();
    await dialog.getByLabel('Anlass *').fill('E2E Leitung');
    await dialog.getByRole('button', { name: 'Speichern' }).click();
    await expect(dialog).toBeHidden();
    await expect(page.getByRole('row', { name: /E2E Leitung/ })).toBeVisible();

    const headers = { Authorization: `Bearer ${token}` };
    const forbidden = [
      context.request.put(`${API}/parent-work/entries/${entry.id}`, { headers, data: {} }),
      context.request.post(`${API}/parent-work/entries/${entry.id}/void`, { headers, data: { reason: 'x' } }),
      context.request.post(`${API}/parent-work/entries/${entry.id}/approve`, { headers }),
      context.request.put(`${API}/parent-work/households/${household.id}/override`, { headers, data: {} }),
      context.request.post(`${API}/parent-work/import/parse`, { headers, data: {} }),
    ];
    for (const res of await Promise.all(forbidden)) expect(res.status(), res.url()).toBe(403);
  } finally { await context.close(); }
});
