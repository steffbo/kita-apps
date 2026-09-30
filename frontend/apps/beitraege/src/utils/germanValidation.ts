// Native form validation bubbles use the browser language; this replaces their texts
// with German ones for every input, select and textarea in the app.

type Field = HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement;

function message(el: Field): string {
  const v = el.validity;
  if (v.valueMissing) {
    if (el instanceof HTMLSelectElement) return 'Bitte einen Eintrag auswählen.';
    if (el instanceof HTMLInputElement && el.type === 'checkbox') return 'Bitte dieses Kästchen ankreuzen.';
    if (el instanceof HTMLInputElement && el.type === 'radio') return 'Bitte eine Option auswählen.';
    if (el instanceof HTMLInputElement && el.type === 'file') return 'Bitte eine Datei auswählen.';
    return 'Bitte dieses Feld ausfüllen.';
  }
  if (v.typeMismatch) {
    if (el.type === 'email') return 'Bitte eine gültige E-Mail-Adresse eingeben.';
    if (el.type === 'url') return 'Bitte eine gültige Web-Adresse eingeben.';
    return 'Bitte einen gültigen Wert eingeben.';
  }
  if (v.badInput) return el.type === 'number' ? 'Bitte eine Zahl eingeben.' : 'Bitte einen gültigen Wert eingeben.';
  if (v.tooShort) return `Bitte mindestens ${(el as HTMLInputElement).minLength} Zeichen eingeben.`;
  if (v.tooLong) return `Bitte höchstens ${(el as HTMLInputElement).maxLength} Zeichen eingeben.`;
  if (v.rangeUnderflow) return `Der Wert muss mindestens ${(el as HTMLInputElement).min} sein.`;
  if (v.rangeOverflow) return `Der Wert darf höchstens ${(el as HTMLInputElement).max} sein.`;
  if (v.stepMismatch) return 'Bitte einen gültigen Wert eingeben.';
  if (v.patternMismatch) return el.title || 'Bitte das verlangte Format einhalten.';
  return '';
}

function isField(target: EventTarget | null): target is Field {
  return target instanceof HTMLInputElement || target instanceof HTMLSelectElement
    || target instanceof HTMLTextAreaElement;
}

// Only messages set here are reset; custom validity from components stays untouched.
const ours = new WeakSet<Field>();

function reset(e: Event) {
  if (isField(e.target) && ours.has(e.target)) {
    e.target.setCustomValidity('');
    ours.delete(e.target);
  }
}

// A value set by code (v-model, pickers) fires no input event; so before every submit attempt
// (clicks on submit buttons, also the implicit one on Enter) our messages are cleared and the
// browser validates afresh.
function resetForm(e: Event) {
  const button = (e.target as Element | null)?.closest?.('button, input[type="submit"]');
  const form = (button as HTMLButtonElement | null)?.form;
  if (!form || (button as HTMLButtonElement).type !== 'submit') return;
  for (const el of Array.from(form.elements)) {
    if (isField(el) && ours.has(el)) { el.setCustomValidity(''); ours.delete(el); }
  }
}

export function installGermanValidation(): void {
  document.addEventListener('click', resetForm, true);
  document.addEventListener('invalid', (e) => {
    if (!isField(e.target) || (e.target.validity.customError && !ours.has(e.target))) return;
    e.target.setCustomValidity('');
    const text = message(e.target);
    if (text) { e.target.setCustomValidity(text); ours.add(e.target); }
  }, true);
  document.addEventListener('input', reset, true);
  document.addEventListener('change', reset, true);
}
