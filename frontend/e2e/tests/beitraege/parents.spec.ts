import { test, expect, Page, Locator } from '@playwright/test';

// Helper to fill a search input and trigger Vue's v-model properly
async function fillSearchInput(page: Page, locator: Locator, value: string) {
  // Set up response listener BEFORE dispatching the event
  const responsePromise = page.waitForResponse(
    resp => resp.url().includes('/children') && resp.status() === 200,
    { timeout: 10000 }
  );
  
  // Dispatch the input event
  await locator.evaluate((el: HTMLInputElement, v: string) => {
    const nativeInputValueSetter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value')!.set!;
    nativeInputValueSetter.call(el, v);
    el.dispatchEvent(new InputEvent('input', { bubbles: true, data: v }));
  }, value);
  
  // Wait for debounce (150ms in component) + response
  await responsePromise;
  await expect(page.locator('.animate-spin')).toBeHidden({ timeout: 10000 });
}

// Helper to find and click on a child by searching
async function navigateToChild(page: Page, childName: string) {
  // Search for the child
  const searchInput = page.getByPlaceholder(/suchen/i);
  await fillSearchInput(page, searchInput, childName);
  
  // Click on the first matching row in the table
  await page.getByRole('cell', { name: new RegExp(childName) }).first().click();
  await expect(page.getByRole('heading', { name: new RegExp(childName) })).toBeVisible();
}

test.describe('Beitraege - Parent Cards on Child Detail Page', () => {
  let testChildName: string;
  let testMemberNumber: string;

  test.beforeEach(async ({ page }) => {
    // Create a test child first
    testMemberNumber = `P${Date.now().toString().slice(-6)}`;
    testChildName = `Parent-Test-${Date.now().toString().slice(-4)}`;

    await page.goto('/kinder');
    await expect(page.locator('.animate-spin')).toBeHidden({ timeout: 10000 });

    await page.getByRole('button', { name: /kind hinzufügen/i }).click();
    await page.getByLabel(/mitgliedsnummer/i).fill(testMemberNumber);
    await page.getByLabel(/vorname/i).fill(testChildName);
    await page.getByLabel(/nachname/i).fill('Testname');
    await page.getByLabel(/geburtsdatum/i).fill('2020-05-10');
    await page.getByLabel(/eintrittsdatum/i).fill('2023-09-01');
    await page.getByRole('button', { name: /speichern/i }).click();

    await expect(page.getByRole('heading', { name: /kind hinzufügen/i })).toBeHidden();
  });

  test('displays "No parents" message when child has no parents', async ({ page }) => {
    // Navigate to detail page by searching
    await navigateToChild(page, testChildName);

    // Should show "No parents" message
    await expect(page.getByText(/keine eltern zugeordnet/i)).toBeVisible();

    // Should show buttons to add or link parents
    await expect(page.getByRole('button', { name: /neu anlegen/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /verknüpfen/i }).first()).toBeVisible();
  });

  test('can open create parent dialog', async ({ page }) => {
    await navigateToChild(page, testChildName);

    // Click "Neu anlegen" button
    await page.getByRole('button', { name: /neu anlegen/i }).click();

    // Dialog should open
    await expect(page.getByRole('heading', { name: /elternteil anlegen/i })).toBeVisible();

    // Check form fields are visible
    await expect(page.getByLabel(/vorname \*/i)).toBeVisible();
    await expect(page.getByLabel(/nachname \*/i)).toBeVisible();
    await expect(page.getByLabel(/e-mail/i)).toBeVisible();
    await expect(page.getByLabel(/telefon/i)).toBeVisible();
  });

  test('can create a new parent from child detail page', async ({ page }) => {
    const parentFirstName = `Eltern-${Date.now().toString().slice(-4)}`;
    const parentLastName = 'Testperson';
    const parentEmail = `test-${Date.now()}@example.com`;
    const parentPhone = '0123456789';

    await navigateToChild(page, testChildName);

    // Open create dialog
    await page.getByRole('button', { name: /neu anlegen/i }).click();
    await expect(page.getByRole('heading', { name: /elternteil anlegen/i })).toBeVisible();

    // Fill form
    await page.getByLabel(/vorname \*/i).fill(parentFirstName);
    await page.getByLabel(/nachname \*/i).fill(parentLastName);
    await page.getByLabel(/e-mail/i).fill(parentEmail);
    await page.getByLabel(/telefon/i).fill(parentPhone);

    // Submit
    await page.getByRole('button', { name: /anlegen & verknüpfen/i }).click();

    // Dialog should close
    await expect(page.getByRole('heading', { name: /elternteil anlegen/i })).toBeHidden();

    // Parent card should appear
    await expect(page.getByText(`${parentFirstName} ${parentLastName}`)).toBeVisible();
    await expect(page.getByText(parentEmail)).toBeVisible();
    await expect(page.getByText(parentPhone)).toBeVisible();

    // "No parents" message should be gone
    await expect(page.getByText(/keine eltern zugeordnet/i)).toBeHidden();
  });

  test('parent card shows contact information', async ({ page }) => {
    const parentFirstName = `Card-${Date.now().toString().slice(-4)}`;
    const parentLastName = 'Display';
    const parentEmail = `card-test-${Date.now()}@example.com`;
    const parentPhone = '0987654321';

    await navigateToChild(page, testChildName);

    // Create a parent with all details
    await page.getByRole('button', { name: /neu anlegen/i }).click();
    await page.getByLabel(/vorname \*/i).fill(parentFirstName);
    await page.getByLabel(/nachname \*/i).fill(parentLastName);
    await page.getByLabel(/e-mail/i).fill(parentEmail);
    await page.getByLabel(/telefon/i).fill(parentPhone);
    await page.locator('#parent-street').fill('Teststraße');
    await page.locator('#parent-streetNo').fill('42');
    await page.locator('#parent-postalCode').fill('12345');
    await page.locator('#parent-city').fill('Teststadt');
    await page.getByRole('button', { name: /anlegen & verknüpfen/i }).click();

    // Wait for dialog to close
    await expect(page.getByRole('heading', { name: /elternteil anlegen/i })).toBeHidden();

    // Verify all contact info is displayed on the card
    await expect(page.getByText(`${parentFirstName} ${parentLastName}`)).toBeVisible();
    await expect(page.getByText(parentEmail)).toBeVisible();
    await expect(page.getByText(parentPhone)).toBeVisible();
    await expect(page.getByText(/teststraße 42/i)).toBeVisible();
  });

  test('can switch between create and link modes in dialog', async ({ page }) => {
    await navigateToChild(page, testChildName);

    // Open dialog via "Neu anlegen" button in the empty state
    await page.getByRole('button', { name: /neu anlegen/i }).click();
    await expect(page.getByRole('heading', { name: /elternteil anlegen/i })).toBeVisible();

    // Should be in create mode - check for form field
    await expect(page.locator('#parent-firstName')).toBeVisible();

    // Switch to link mode using the tab button inside the dialog
    await page.locator('.fixed').getByRole('button', { name: /vorhandenen verknüpfen/i }).click();

    // Should show search field
    await expect(page.getByPlaceholder(/name eingeben/i)).toBeVisible();
    await expect(page.getByText(/mindestens 2 zeichen/i)).toBeVisible();

    // Switch back to create mode using the tab button
    await page.locator('.fixed').getByRole('button', { name: /neu anlegen/i }).click();

    // Should show create form again
    await expect(page.locator('#parent-firstName')).toBeVisible();
  });

  test('can open link parent dialog directly', async ({ page }) => {
    await navigateToChild(page, testChildName);

    // Click "Verknüpfen" button
    await page.getByRole('button', { name: /verknüpfen/i }).first().click();

    // Dialog should open in link mode
    await expect(page.getByRole('heading', { name: /elternteil verknüpfen/i })).toBeVisible();
    await expect(page.getByPlaceholder(/name eingeben/i)).toBeVisible();
  });

  test('can search for existing parents in link mode', async ({ page }) => {
    // First create a parent via a different child to have something to search for
    const searchParentName = `Suchbar-${Date.now().toString().slice(-4)}`;
    
    // Create another child to attach the parent to first
    await page.goto('/kinder');
    await expect(page.locator('.animate-spin')).toBeHidden({ timeout: 10000 });
    
    const helperChildName = `SearchHelper-${Date.now().toString().slice(-4)}`;
    const helperMemberNumber = `H${Date.now().toString().slice(-6)}`;
    await page.getByRole('button', { name: /kind hinzufügen/i }).click();
    await page.getByLabel(/mitgliedsnummer/i).fill(helperMemberNumber);
    await page.getByLabel(/vorname/i).fill(helperChildName);
    await page.getByLabel(/nachname/i).fill('Child');
    await page.getByLabel(/geburtsdatum/i).fill('2021-01-01');
    await page.getByLabel(/eintrittsdatum/i).fill('2024-01-01');
    await page.getByRole('button', { name: /speichern/i }).click();
    await expect(page.getByRole('heading', { name: /kind hinzufügen/i })).toBeHidden();
    
    // Navigate to helper child and create a parent there
    await navigateToChild(page, helperChildName);
    
    await page.getByRole('button', { name: /neu anlegen/i }).click();
    await page.locator('#parent-firstName').fill(searchParentName);
    await page.locator('#parent-lastName').fill('Elternteil');
    await page.getByRole('button', { name: /anlegen & verknüpfen/i }).click();
    await expect(page.getByRole('heading', { name: /elternteil anlegen/i })).toBeHidden();

    // Now go to our test child and try to link
    await page.goto('/kinder');
    await expect(page.locator('.animate-spin')).toBeHidden({ timeout: 10000 });

    await navigateToChild(page, testChildName);

    // Open link dialog
    await page.getByRole('button', { name: /verknüpfen/i }).first().click();
    await expect(page.getByRole('heading', { name: /elternteil verknüpfen/i })).toBeVisible();

    // Search for the parent
    await page.getByPlaceholder(/name eingeben/i).fill(searchParentName);

    // Wait for search results
    await expect(page.getByText(`${searchParentName} Elternteil`)).toBeVisible({ timeout: 5000 });
  });

  test('can link an existing parent to a child', async ({ page }) => {
    // First create a parent via a different child
    const linkParentName = `Link-${Date.now().toString().slice(-4)}`;

    // Create another child to attach the parent to first
    await page.goto('/kinder');
    await expect(page.locator('.animate-spin')).toBeHidden({ timeout: 10000 });
    
    const helperChildName = `LinkHelper-${Date.now().toString().slice(-4)}`;
    const helperMemberNumber = `L${Date.now().toString().slice(-6)}`;
    await page.getByRole('button', { name: /kind hinzufügen/i }).click();
    await page.getByLabel(/mitgliedsnummer/i).fill(helperMemberNumber);
    await page.getByLabel(/vorname/i).fill(helperChildName);
    await page.getByLabel(/nachname/i).fill('Child');
    await page.getByLabel(/geburtsdatum/i).fill('2021-01-01');
    await page.getByLabel(/eintrittsdatum/i).fill('2024-01-01');
    await page.getByRole('button', { name: /speichern/i }).click();
    await expect(page.getByRole('heading', { name: /kind hinzufügen/i })).toBeHidden();
    
    // Navigate to helper child and create a parent there
    await navigateToChild(page, helperChildName);
    
    await page.getByRole('button', { name: /neu anlegen/i }).click();
    await page.locator('#parent-firstName').fill(linkParentName);
    await page.locator('#parent-lastName').fill('Zuverknüpfen');
    await page.getByRole('button', { name: /anlegen & verknüpfen/i }).click();
    await expect(page.getByRole('heading', { name: /elternteil anlegen/i })).toBeHidden();

    // Go to our test child and link the parent
    await page.goto('/kinder');
    await expect(page.locator('.animate-spin')).toBeHidden({ timeout: 10000 });

    await navigateToChild(page, testChildName);

    // Open link dialog
    await page.getByRole('button', { name: /verknüpfen/i }).first().click();

    // Search and select
    await page.getByPlaceholder(/name eingeben/i).fill(linkParentName);
    await expect(page.getByText(`${linkParentName} Zuverknüpfen`)).toBeVisible({ timeout: 5000 });
    await page.getByText(`${linkParentName} Zuverknüpfen`).click();

    // Should show selected preview
    await expect(page.getByText(/ausgewählt:/i)).toBeVisible();

    // Click link button
    await page.getByRole('button', { name: /verknüpfen/i }).last().click();

    // Dialog should close
    await expect(page.getByRole('heading', { name: /elternteil verknüpfen/i })).toBeHidden();

    // Parent should appear on the page
    await expect(page.getByText(`${linkParentName} Zuverknüpfen`)).toBeVisible();
  });

  test('can unlink a parent from a child', async ({ page }) => {
    const unlinkParentName = `Unlink-${Date.now().toString().slice(-4)}`;

    await navigateToChild(page, testChildName);

    // Create and link a parent first
    await page.getByRole('button', { name: /neu anlegen/i }).click();
    await page.getByLabel(/vorname \*/i).fill(unlinkParentName);
    await page.getByLabel(/nachname \*/i).fill('Zuentfernen');
    await page.getByRole('button', { name: /anlegen & verknüpfen/i }).click();
    await expect(page.getByRole('heading', { name: /elternteil anlegen/i })).toBeHidden();

    // Verify parent is shown
    await expect(page.getByText(`${unlinkParentName} Zuentfernen`)).toBeVisible();

    // Click unlink button (the span with Unlink icon on the parent card)
    await page.locator('span[title="Verknüpfung aufheben"]').click();

    // Confirmation dialog should appear
    await expect(page.getByRole('heading', { name: /verknüpfung aufheben/i })).toBeVisible();
    await expect(page.getByText(/möchtest du die verknüpfung/i)).toBeVisible();

    // Confirm
    await page.getByRole('button', { name: /aufheben/i }).last().click();

    // Dialog should close
    await expect(page.getByRole('heading', { name: /verknüpfung aufheben/i })).toBeHidden();

    // Parent should be gone from the page
    await expect(page.getByText(`${unlinkParentName} Zuentfernen`)).toBeHidden();

    // "No parents" message should appear again
    await expect(page.getByText(/keine eltern zugeordnet/i)).toBeVisible();
  });

  test('can cancel unlink dialog', async ({ page }) => {
    const cancelUnlinkParent = `CancelUnlink-${Date.now().toString().slice(-4)}`;

    await navigateToChild(page, testChildName);

    // Create and link a parent
    await page.getByRole('button', { name: /neu anlegen/i }).click();
    await page.getByLabel(/vorname \*/i).fill(cancelUnlinkParent);
    await page.getByLabel(/nachname \*/i).fill('Bleiben');
    await page.getByRole('button', { name: /anlegen & verknüpfen/i }).click();
    await expect(page.getByRole('heading', { name: /elternteil anlegen/i })).toBeHidden();

    // Click unlink
    await page.locator('span[title="Verknüpfung aufheben"]').click();
    await expect(page.getByRole('heading', { name: /verknüpfung aufheben/i })).toBeVisible();

    // Cancel
    await page.getByRole('button', { name: /abbrechen/i }).click();

    // Dialog should close
    await expect(page.getByRole('heading', { name: /verknüpfung aufheben/i })).toBeHidden();

    // Parent should still be visible
    await expect(page.getByText(`${cancelUnlinkParent} Bleiben`)).toBeVisible();
  });

  test('can add multiple parents to a child', async ({ page }) => {
    const parent1Name = `Multi1-${Date.now().toString().slice(-4)}`;
    const parent2Name = `Multi2-${Date.now().toString().slice(-4)}`;

    await navigateToChild(page, testChildName);

    // Create first parent
    await page.getByRole('button', { name: /neu anlegen/i }).click();
    await page.getByLabel(/vorname \*/i).fill(parent1Name);
    await page.getByLabel(/nachname \*/i).fill('Erster');
    await page.getByRole('button', { name: /anlegen & verknüpfen/i }).click();
    await expect(page.getByRole('heading', { name: /elternteil anlegen/i })).toBeHidden();

    // Verify first parent is shown
    await expect(page.getByText(`${parent1Name} Erster`)).toBeVisible();

    // Create second parent using the header "Neu" button
    await page.getByRole('button', { name: /^neu$/i }).click();
    await page.getByLabel(/vorname \*/i).fill(parent2Name);
    await page.getByLabel(/nachname \*/i).fill('Zweiter');
    await page.getByRole('button', { name: /anlegen & verknüpfen/i }).click();
    await expect(page.getByRole('heading', { name: /elternteil anlegen/i })).toBeHidden();

    // Both parents should be visible
    await expect(page.getByText(`${parent1Name} Erster`)).toBeVisible();
    await expect(page.getByText(`${parent2Name} Zweiter`)).toBeVisible();
  });

  test('validates required fields when creating parent', async ({ page }) => {
    await navigateToChild(page, testChildName);

    // Open create dialog
    await page.getByRole('button', { name: /neu anlegen/i }).click();
    await expect(page.getByRole('heading', { name: /elternteil anlegen/i })).toBeVisible();

    // Try to submit without filling required fields
    await page.getByRole('button', { name: /anlegen & verknüpfen/i }).click();

    // Dialog should still be visible (HTML5 validation prevents submission)
    await expect(page.getByRole('heading', { name: /elternteil anlegen/i })).toBeVisible();
  });

  test('can close parent dialog with X button', async ({ page }) => {
    await navigateToChild(page, testChildName);

    // Open dialog
    await page.getByRole('button', { name: /neu anlegen/i }).click();
    await expect(page.getByRole('heading', { name: /elternteil anlegen/i })).toBeVisible();

    // Close with X button - it's the button next to the dialog title
    await page.locator('.fixed .bg-white button').first().click();

    // Dialog should close
    await expect(page.getByRole('heading', { name: /elternteil anlegen/i })).toBeHidden();
  });

  test('can close parent dialog with Cancel button', async ({ page }) => {
    await navigateToChild(page, testChildName);

    // Open dialog
    await page.getByRole('button', { name: /neu anlegen/i }).click();
    await expect(page.getByRole('heading', { name: /elternteil anlegen/i })).toBeVisible();

    // Close with Cancel button
    await page.getByRole('button', { name: /abbrechen/i }).click();

    // Dialog should close
    await expect(page.getByRole('heading', { name: /elternteil anlegen/i })).toBeHidden();
  });
});
