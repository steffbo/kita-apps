import { test, expect } from './fixtures';

test('admin selects parents and sees individual invitation results', async ({ adminPage: page }) => {
  const anna = '00000000-0000-4000-8000-000000000001';
  const bert = '00000000-0000-4000-8000-000000000002';
  const double = '00000000-0000-4000-8000-000000000003';
  const account = '00000000-0000-4000-8000-000000000004';
  await page.route('**/api/fees/v1/users/invitation-candidates', (route) => route.fulfill({
    status: 200, contentType: 'application/json', body: JSON.stringify([
      { parentId: anna, firstName: 'Anna', lastName: 'Test', email: 'anna@e2e.test', ambiguous: false,
        children: ['Ida Test'] },
      { parentId: bert, firstName: 'Bert', lastName: 'Test', email: 'bert@e2e.test', ambiguous: false,
        children: ['Ida Test', 'Ole Test'] },
      { parentId: double, firstName: 'Doppelt', lastName: 'Test',
        email: 'doppelt@e2e.test', ambiguous: true, children: ['Mia Test'] },
      { parentId: account, firstName: 'Konto', lastName: 'Test', email: 'konto@e2e.test',
        ambiguous: false, children: ['Pia Test'], userId: '00000000-0000-4000-8000-000000000099' },
    ]),
  }));
  let requested: string[] = [];
  await page.route('**/api/fees/v1/users/invitations', async (route) => {
    requested = route.request().postDataJSON().parentIds;
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({
      results: [
        { parentId: anna, email: 'anna@e2e.test', success: true },
        { parentId: bert, email: 'bert@e2e.test', success: false, error: 'SMTP nicht erreichbar' },
      ],
    }) });
  });

  await page.goto('/beitraege/benutzer');
  await page.getByRole('button', { name: 'Eltern einladen' }).click();
  const dialog = page.getByRole('dialog', { name: 'Eltern einladen' });
  await expect(dialog.getByRole('checkbox', { name: /doppelt@e2e.test/ })).toBeDisabled();
  await expect(dialog.getByText('Kinder: Ida Test, Ole Test')).toBeVisible();
  // Temporary: existing accounts are listed but not part of „Alle auswählen“.
  await expect(dialog.getByText(/Hat bereits ein Konto/)).toBeVisible();
  await dialog.getByRole('checkbox', { name: 'Alle auswählen' }).check();
  await expect(dialog.getByRole('checkbox', { name: /konto@e2e.test/ })).not.toBeChecked();
  await dialog.getByRole('button', { name: 'Ausgewählte einladen' }).click();
  await expect(dialog.getByText('Einladung gesendet')).toBeVisible();
  await expect(dialog.getByText('SMTP nicht erreichbar')).toBeVisible();
  expect(requested).toEqual([anna, bert]);
});
