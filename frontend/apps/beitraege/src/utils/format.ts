// Central display formatters. Business dates follow Europe/Berlin (like the backend's util.Today()).

const TIME_ZONE = 'Europe/Berlin';
const DATE_ONLY = /^(\d{4})-(\d{2})-(\d{2})(?:$|T00:00:00(?:\.0+)?(?:Z|[+-]00:00)?$)/;

const dateFormat = new Intl.DateTimeFormat('de-DE', {
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
  timeZone: TIME_ZONE,
});
const dateTimeFormat = new Intl.DateTimeFormat('de-DE', {
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  timeZone: TIME_ZONE,
});
const currencyFormat = new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR' });
const wholeCurrencyFormat = new Intl.NumberFormat('de-DE', {
  style: 'currency',
  currency: 'EUR',
  maximumFractionDigits: 0,
});

/** Formats a date (`YYYY-MM-DD` or ISO timestamp) as `TT.MM.JJJJ`; empty values yield the fallback. */
export function formatDate(value: string | null | undefined, fallback = '—'): string {
  if (!value) return fallback;
  // Date-only values (and UTC-midnight scans of DATE columns) are calendar dates, not instants.
  const match = DATE_ONLY.exec(value);
  if (match) return `${match[3]}.${match[2]}.${match[1]}`;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? fallback : dateFormat.format(date);
}

/** Formats a timestamp in Berlin time as `TT.MM.JJJJ, hh:mm:ss`. */
export function formatDateTime(value: string | null | undefined, fallback = '—'): string {
  if (!value) return fallback;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? fallback : dateTimeFormat.format(date);
}

/** Returns the `YYYY-MM-DD` part for `<input type="date">`. */
export function formatDateForInput(value: string | null | undefined): string {
  return value ? value.split('T')[0] : '';
}

/** Formats an amount in euros with cents, e.g. `1.234,50 €`. */
export function formatCurrency(amount: number): string {
  return currencyFormat.format(amount);
}

/** Whole minutes as decimal hours, with a German comma. */
export function formatHours(minutes: number): string {
  return `${new Intl.NumberFormat('de-DE', { maximumFractionDigits: 2 }).format(minutes / 60)} h`;
}

/** Formats an amount in whole euros (incomes); empty values yield the fallback. */
export function formatCurrencyWhole(amount: number | null | undefined, fallback = '—'): string {
  if (amount === null || amount === undefined) return fallback;
  return wholeCurrencyFormat.format(amount);
}

/** German month name for 1–12 (`long`: "Januar", `short`: "Jan."); 0/undefined yields ''. */
export function formatMonthName(month: number | null | undefined, style: 'long' | 'short' = 'long'): string {
  if (!month) return '';
  return new Date(2000, month - 1).toLocaleString('de-DE', { month: style });
}

/** "Januar 2025". */
export function formatMonthYear(year: number, month: number): string {
  return new Date(year, month - 1).toLocaleString('de-DE', { month: 'long', year: 'numeric' });
}

const isoDateFormat = new Intl.DateTimeFormat('en-CA', { timeZone: TIME_ZONE });

/** Today's Berlin calendar date as `YYYY-MM-DD` (not the UTC date, which lags after midnight). */
export function todayISO(): string {
  return isoDateFormat.format(new Date());
}

/** Options for month pickers: `{ value: 1, label: 'Januar' }` … */
export const MONTH_OPTIONS = Array.from({ length: 12 }, (_, i) => ({ value: i + 1, label: formatMonthName(i + 1) }));

/** Berlin calendar date of a date or timestamp as `YYYY-MM-DD`, or null when invalid. */
function toBerlinISODate(value: string): string | null {
  const match = DATE_ONLY.exec(value);
  if (match) return `${match[1]}-${match[2]}-${match[3]}`;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? null : isoDateFormat.format(date);
}

/** Berlin calendar days from today to the given date (negative = past), or null when empty/invalid. */
function daysFromToday(value: string | null | undefined): number | null {
  const iso = value ? toBerlinISODate(value) : null;
  return iso ? Math.round((Date.parse(iso) - Date.parse(todayISO())) / 86_400_000) : null;
}

/** Due date relative to today: "heute", "morgen", "in 3 Tagen", "seit 2 Tagen fällig". */
export function formatDueIn(value: string | null | undefined, fallback = '—'): string {
  const days = daysFromToday(value);
  if (days === null) return fallback;
  if (days === 0) return 'heute';
  if (days === 1) return 'morgen';
  if (days > 1) return `in ${days} Tagen`;
  return days === -1 ? 'seit gestern fällig' : `seit ${-days} Tagen fällig`;
}
