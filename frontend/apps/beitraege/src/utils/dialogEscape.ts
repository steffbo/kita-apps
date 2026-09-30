// Esc closes the topmost open dialog in the whole app. Dialogs are overlays
// (`fixed inset-0` or `aria-modal`); the handler clicks their close control:
// `[data-dialog-close]`, a button labelled „Schließen“, or an „Abbrechen“/„Schließen“ button.

const OVERLAY = '[aria-modal="true"], .fixed.inset-0';
const CLOSE_TEXT = /^(abbrechen|schließen)$/i;

function visible(el: HTMLElement): boolean {
  return el.getClientRects().length > 0;
}

function closeControl(dialog: Element): HTMLElement | null {
  const buttons = [...dialog.querySelectorAll<HTMLElement>('[data-dialog-close], button')]
    .filter(b => visible(b) && !(b as HTMLButtonElement).disabled);
  return buttons.find(b => b.hasAttribute('data-dialog-close'))
    ?? buttons.find(b => /schließen/i.test(b.getAttribute('aria-label') ?? ''))
    ?? buttons.find(b => CLOSE_TEXT.test(b.textContent?.trim() ?? ''))
    ?? null;
}

function onKeydown(e: KeyboardEvent): void {
  if (e.key !== 'Escape' || e.defaultPrevented || e.isComposing) return;
  const overlays = [...document.querySelectorAll<HTMLElement>(OVERLAY)]
    .filter(el => visible(el) && el.getAttribute('aria-hidden') !== 'true');
  // Nested dialogs come later in document order; the last one is on top.
  for (const overlay of overlays.reverse()) {
    const control = closeControl(overlay);
    if (!control) continue;
    e.preventDefault();
    // Page-level Esc handlers must not close a second dialog underneath.
    e.stopImmediatePropagation();
    control.click();
    return;
  }
}

export function installDialogEscape(): void {
  window.addEventListener('keydown', onKeydown, { capture: true });
}
