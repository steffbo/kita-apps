import { test, expect, uniq } from './fixtures';
import { berlinToday } from './bank';

test('admin sets the exit date for several selected children at once', async ({ adminApi, adminPage: page }) => {
  const suffix = uniq();
  const today = berlinToday();
  const ids: string[] = [];
  for (const firstName of ['Anna', 'Ben']) {
    const child = await adminApi.post<{ id: string }>('/children', {
      memberNumber: String(10_000_000 + Math.floor(Math.random() * 90_000_000)),
      firstName, lastName: `Austritt${suffix}`,
      birthDate: `${today.year - 6}-01-01`, entryDate: `${today.year - 3}-08-01`,
    });
    ids.push(child.id);
  }

  await page.goto('/beitraege/kinder');
  await page.getByPlaceholder('Suchen nach Name oder Mitgliedsnummer...').fill(`Austritt${suffix}`);
  const rows = page.getByRole('row', { name: new RegExp(`Austritt${suffix}`) });
  await expect(rows).toHaveCount(2);
  for (const row of await rows.all()) await row.getByRole('checkbox').check();
  await expect(page.getByText('2 Kinder ausgewählt')).toBeVisible();

  await page.getByRole('button', { name: 'Austrittsdatum setzen' }).click();
  const dialog = page.getByRole('dialog', { name: 'Austrittsdatum setzen' });
  const exitDate = `${today.year + 5}-07-31`;
  await dialog.getByLabel('Austrittsdatum').fill(`${today.year - 4}-07-31`);
  await expect(dialog.getByRole('alert')).toContainText(/Liegt vor dem Eintritt von .*Anna/);
  await expect(dialog.getByRole('button', { name: 'Austrittsdatum setzen' })).toBeDisabled();
  await dialog.getByLabel('Austrittsdatum').fill(exitDate);
  await dialog.getByRole('button', { name: 'Austrittsdatum setzen' }).click();
  await expect(dialog).toBeHidden();

  for (const id of ids) {
    const child = await adminApi.get<{ exitDate?: string }>(`/children/${id}`);
    expect(child.exitDate?.slice(0, 10)).toBe(exitDate);
  }
});
