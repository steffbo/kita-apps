import { test, expect, uniq } from './fixtures';
import { berlinToday } from './bank';

test('desktop: sidebar, Aufgaben tabs and search with Strg+K', async ({ adminApi, adminPage: page }) => {
  const lastName = `Navi${uniq()}`;
  const today = berlinToday();
  const child = await adminApi.post<{ id: string }>('/children', {
    memberNumber: String(10_000_000 + Math.floor(Math.random() * 90_000_000)),
    firstName: 'Nora', lastName,
    birthDate: `${today.year - 4}-01-01`, entryDate: `${today.year - 2}-08-01`,
  });

  await page.goto('/beitraege/');
  const sidebar = page.getByRole('navigation', { name: 'Hauptnavigation' });
  await expect(page.getByRole('navigation', { name: 'Schnellnavigation' })).toBeHidden();
  await sidebar.getByRole('link', { name: /^Aufgaben/ }).click();
  await expect(page.getByRole('heading', { name: 'Aufgaben', exact: true })).toBeVisible();
  const sections = page.getByRole('navigation', { name: 'Aufgaben-Bereiche' });
  for (const name of [/^Bankabgleich/, /^Meldungen/, /^Änderungen/]) {
    await expect(sections.getByRole('link', { name })).toBeVisible();
  }
  await sections.getByRole('link', { name: /^Meldungen/ }).click();
  await expect(page).toHaveURL(/\/beitraege\/meldungen$/);
  await expect(sidebar.getByRole('link', { name: /^Aufgaben/ })).toHaveAttribute('aria-current', 'page');

  await page.keyboard.press('Control+k');
  const search = page.getByRole('dialog', { name: 'Suchen' });
  await expect(search.getByRole('combobox', { name: 'Suchen' })).toBeFocused();
  await search.getByRole('combobox', { name: 'Suchen' }).fill(lastName);
  await search.getByRole('option', { name: new RegExp(`Nora ${lastName}`) }).click();
  await expect(page).toHaveURL(new RegExp(`/beitraege/kinder/${child.id}$`));
  await expect(search).toHaveCount(0);
  await expect(sidebar.getByRole('link', { name: 'Kinder', exact: true })).toHaveAttribute('aria-current', 'page');
});

test('phone: tab bar, Mehr sheet and account sheet with theme', async ({ adminPage: page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/beitraege/');
  await expect(page.getByRole('navigation', { name: 'Hauptnavigation' })).toBeHidden();
  const tabs = page.getByRole('navigation', { name: 'Schnellnavigation' });
  for (const name of [/^Start/, /^Aufgaben/, /^Kinder/, /^Erinnerungen/]) {
    await expect(tabs.getByRole('link', { name })).toBeVisible();
  }

  await tabs.getByRole('link', { name: /^Erinnerungen/ }).click();
  await expect(page).toHaveURL(/\/beitraege\/automatisierung$/);

  await tabs.getByRole('button', { name: 'Mehr' }).click();
  const more = page.getByRole('dialog', { name: 'Navigation' });
  await more.getByRole('link', { name: 'Benutzer', exact: true }).click();
  await expect(page).toHaveURL(/\/beitraege\/benutzer$/);
  await expect(more).toHaveCount(0);

  await page.getByRole('button', { name: 'Benutzermenü' }).click();
  const account = page.getByRole('dialog', { name: 'Konto' });
  await account.getByRole('radio', { name: 'Dunkel' }).click();
  await expect(page.locator('html')).toHaveClass(/dark/);
  await account.getByRole('radio', { name: 'Hell' }).click();
  await expect(page.locator('html')).not.toHaveClass(/dark/);
  await page.keyboard.press('Escape');
  await expect(account).toHaveCount(0);
});
