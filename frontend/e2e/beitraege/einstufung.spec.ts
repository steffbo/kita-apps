import { test, expect, uniq } from './fixtures';
import { berlinToday } from './bank';

function memberNumber() {
  return String(10_000_000 + Math.floor(Math.random() * 90_000_000));
}

test('Neue Einstufung: Kind per Suche wählen, Werte des Geschwisterkinds übernehmen',
  async ({ adminApi, adminPage: page }) => {
    const suffix = uniq();
    const today = berlinToday();
    const lastName = `Stufe${suffix}`;
    const parent = await adminApi.post<{ id: string }>('/parents', {
      firstName: 'Eva', lastName, email: `eltern-${suffix}@e2e.test`,
    });
    const ids: Record<string, string> = {};
    for (const firstName of ['Ida', 'Ole']) {
      const child = await adminApi.post<{ id: string }>('/children', {
        memberNumber: memberNumber(), firstName, lastName,
        birthDate: `${today.year - 1}-06-01`, entryDate: `${today.year}-01-01`,
      });
      await adminApi.post(`/children/${child.id}/parents`, { parentId: parent.id, isPrimary: true });
      ids[firstName] = child.id;
    }
    await adminApi.post('/einstufungen', {
      childId: ids.Ida, year: today.year, incomeBasis: 'INCOME', childrenCount: 3,
      incomeCalculation: { parent1: { grossIncome: 62000, tax: 9000, advertisingCosts: 1500 }, parent2: {} },
    });

    await page.goto('/beitraege/einstufungen/neu');
    const search = page.getByRole('combobox', { name: 'Kind suchen' });
    await search.fill('O');
    await expect(page.getByText('Mindestens 2 Zeichen eingeben.')).toBeVisible();
    await expect(page.getByRole('listbox')).toHaveCount(0);

    await search.fill(`ole stufe${suffix}`);
    await page.getByRole('option', { name: new RegExp(`Ole ${lastName}`) }).click();
    await expect(page.getByText(/Einkommensangaben und Kinderzahl übernommen aus der Einstufung von Ida/))
      .toBeVisible();
    await expect(page.getByLabel('Anzahl unterhaltspflichtiger Kinder')).toHaveValue('3');
    await expect(page.getByRole('region', { name: 'Elternteil 1' }).getByLabel('Bruttoeinkommen'))
      .toHaveValue('62000');
    await expect(page.getByRole('heading', { name: 'Vorschau – noch nicht gespeichert' })).toBeVisible();
    await expect(page.getByText('Einkommensvorschau')).toBeVisible();

    await page.getByRole('button', { name: 'Einstufung speichern' }).click();
    await expect(page.getByText('Einstufung erfolgreich gespeichert.')).toBeVisible();
    await expect(page).toHaveURL(/\/einstufungen\/[0-9a-f-]{36}$/);
    await expect(page.getByRole('heading', { name: 'Ergebnis' })).toBeVisible();

    // Changing the basis previews the result without saving it.
    await page.getByRole('radio', { name: /^Höchstsatz/ }).check();
    await expect(page.getByRole('heading', { name: 'Vorschau – noch nicht gespeichert' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Einkommensberechnung' })).toHaveCount(0);
    await page.getByRole('radio', { name: /^Einkommen/ }).check();
    await expect(page.getByRole('heading', { name: 'Ergebnis' })).toBeVisible();

    // A child with an open Einstufung needs a follow-up instead of a second one.
    await page.goto('/beitraege/einstufungen/neu');
    await search.fill(`Ida ${lastName}`);
    await search.press('Enter');
    await expect(page.getByText('Für Ida gibt es bereits eine gültige Einstufung')).toBeVisible();
    await expect(page.getByRole('link', { name: 'Folgeeinstufung erstellen →' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Einstufung speichern' })).toBeDisabled();
  });

test('Neue Einstufung ohne Einkommensangaben braucht eine Bestätigung',
  async ({ adminApi, adminPage: page }) => {
    const suffix = uniq();
    const today = berlinToday();
    const lastName = `Leer${suffix}`;
    const parent = await adminApi.post<{ id: string }>('/parents', {
      firstName: 'Eva', lastName, email: `eltern-${suffix}@e2e.test`,
    });
    const child = await adminApi.post<{ id: string }>('/children', {
      memberNumber: memberNumber(), firstName: 'Mia', lastName,
      birthDate: `${today.year - 1}-06-01`, entryDate: `${today.year}-01-01`,
    });
    await adminApi.post(`/children/${child.id}/parents`, { parentId: parent.id, isPrimary: true });

    await page.goto(`/beitraege/einstufungen/neu?childId=${child.id}`);
    const warning = page.getByText(/Keine Einkommensangaben eingetragen/);
    await expect(warning).toBeVisible();
    await expect(page.getByRole('radio', { name: /^Nicht erforderlich/ })).toBeDisabled();

    page.once('dialog', (dialog) => void dialog.dismiss());
    await page.getByRole('button', { name: 'Einstufung speichern' }).click();
    await expect(page).toHaveURL(/\/einstufungen\/neu/);

    await page.getByLabel('Vorläufig – Nachweise fehlen noch').check();
    page.once('dialog', (dialog) => void dialog.accept());
    await page.getByRole('button', { name: 'Einstufung speichern' }).click();
    await expect(page.getByText('Einstufung erfolgreich gespeichert.')).toBeVisible();
    await expect(page.getByText('Vorläufig – Nachweise fehlen', { exact: true })).toBeVisible();
  });

test('Einstufungsbrief als PDF laden und per E-Mail senden', async ({ adminApi, adminPage: page }) => {
  const suffix = uniq();
  const today = berlinToday();
  const lastName = `Brief${suffix}`;
  const address = `brief-${suffix}@e2e.test`;
  const parent = await adminApi.post<{ id: string }>('/parents', { firstName: 'Eva', lastName, email: address });
  const child = await adminApi.post<{ id: string }>('/children', {
    memberNumber: memberNumber(), firstName: 'Lea', lastName,
    birthDate: `${today.year - 1}-06-01`, entryDate: `${today.year}-01-01`,
  });
  await adminApi.post(`/children/${child.id}/parents`, { parentId: parent.id, isPrimary: true });
  const created = await adminApi.post<{ id: string }>('/einstufungen', {
    childId: child.id, year: today.year, incomeBasis: 'HIGHEST_RATE', childrenCount: 1,
    incomeCalculation: { parent1: {}, parent2: {} },
  });

  await page.goto(`/beitraege/einstufungen/${created.id}`);
  await expect(page.getByText('Noch nicht per E-Mail versendet.')).toBeVisible();
  const downloadPromise = page.waitForEvent('download');
  await page.getByRole('button', { name: 'PDF herunterladen' }).click();
  const download = await downloadPromise;
  expect(download.suggestedFilename()).toBe(`Einstufung_${today.year}_${lastName}_Lea.pdf`);

  // The e2e stack has no SMTP server: the API refuses instead of pretending to send.
  page.once('dialog', (dialog) => void dialog.accept());
  await page.getByRole('button', { name: 'E-Mail mit PDF senden' }).click();
  await expect(page.getByText('E-Mail-Versand ist auf diesem Server nicht eingerichtet.')).toBeVisible();

  const sentAt = new Date().toISOString();
  let sentBody: { subject?: string; body?: string } = {};
  await page.route(`**/api/fees/v1/einstufungen/${created.id}/send`, async (route) => {
    sentBody = route.request().postDataJSON();
    await route.fulfill({ json: { recipients: [address], sentAt, filename: download.suggestedFilename() } });
  });
  page.once('dialog', (dialog) => void dialog.accept());
  await page.getByRole('button', { name: 'E-Mail mit PDF senden' }).click();
  await expect(page.getByText(new RegExp(`gesendet\\.$`))).toContainText(address);
  await expect(page.getByText(/Zuletzt per E-Mail versendet am/)).toBeVisible();
  expect(sentBody.subject).toBe(`Einstufung Elternbeiträge ${today.year} – ${lastName}`);
  expect(sentBody.body).toContain('Im Anhang findet ihr die Einstufung');
});
