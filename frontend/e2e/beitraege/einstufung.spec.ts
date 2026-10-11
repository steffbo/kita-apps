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
      childId: ids.Ida, year: today.year, validFrom: `${today.year}-01-01`,
      highestRateVoluntary: false, careHoursPerWeek: 0, childrenCount: 3,
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

    await page.getByRole('button', { name: 'Einstufung speichern' }).click();
    await expect(page.getByText('Einstufung erfolgreich gespeichert.')).toBeVisible();
    await expect(page).toHaveURL(/\/einstufungen\/[0-9a-f-]{36}$/);

    // A child with an open Einstufung needs a follow-up instead of a second one.
    await page.goto('/beitraege/einstufungen/neu');
    await search.fill(`Ida ${lastName}`);
    await search.press('Enter');
    await expect(page.getByText('Für Ida gibt es bereits eine gültige Einstufung')).toBeVisible();
    await expect(page.getByRole('link', { name: 'Folgeeinstufung erstellen →' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Einstufung speichern' })).toBeDisabled();
  });
