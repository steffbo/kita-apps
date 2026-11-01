// Labels, badge styles and small date helpers for Ratenzahlungen (installment plans).
// The schedule itself comes from the API; `schedulePreviewText` is only the live hint in the form.
import type { InstallmentDetail, InstallmentPlan, InstallmentRate } from '@/api/types';
import { formatCurrency, formatDate } from '@/utils/format';

export const INTERVAL_OPTIONS = [
  { value: 1, label: 'monatlich' },
  { value: 2, label: 'alle 2 Monate' },
  { value: 3, label: 'vierteljährlich' },
  { value: 6, label: 'halbjährlich' },
  { value: 12, label: 'jährlich' },
];

export function intervalLabel(months: number): string {
  return INTERVAL_OPTIONS.find(option => option.value === months)?.label ?? `alle ${months} Monate`;
}

const PLAN_STATE_LABELS: Record<string, string> = {
  ACTIVE: 'Läuft',
  OVERDUE: 'Rückstand',
  DONE: 'Abgeschlossen',
  CLOSED: 'Beendet',
};

const PLAN_STATE_BADGES: Record<string, string> = {
  ACTIVE: 'badge badge-primary',
  OVERDUE: 'badge badge-danger',
  DONE: 'badge badge-success',
  CLOSED: 'badge',
};

export function planStateLabel(state: string): string {
  return PLAN_STATE_LABELS[state] ?? state;
}

export function planStateBadgeClass(state: string): string {
  return PLAN_STATE_BADGES[state] ?? 'badge';
}

const RATE_STATUS_LABELS: Record<string, string> = {
  PAID: 'Bezahlt',
  OVERDUE: 'Überfällig',
  PARTIAL: 'Teilweise',
  OPEN: 'Offen',
};

const RATE_STATUS_BADGES: Record<string, string> = {
  PAID: 'badge badge-success',
  OVERDUE: 'badge badge-danger',
  PARTIAL: 'badge badge-warning',
  OPEN: 'badge',
};

export function rateStatusLabel(status: string): string {
  return RATE_STATUS_LABELS[status] ?? status;
}

export function rateStatusBadgeClass(status: string): string {
  return RATE_STATUS_BADGES[status] ?? 'badge';
}

export type PlanFilter = 'active' | 'overdue' | 'done' | 'all';

export const PLAN_FILTERS: { value: PlanFilter; label: string }[] = [
  { value: 'active', label: 'Aktiv' },
  { value: 'overdue', label: 'Rückstand' },
  { value: 'done', label: 'Abgeschlossen' },
  { value: 'all', label: 'Alle' },
];

const FILTER_STATES: Record<PlanFilter, string[] | null> = {
  active: ['ACTIVE', 'OVERDUE'],
  overdue: ['OVERDUE'],
  done: ['DONE', 'CLOSED'],
  all: null,
};

export function matchesPlanFilter(detail: InstallmentDetail, filter: PlanFilter): boolean {
  const states = FILTER_STATES[filter];
  return states === null || states.includes(detail.summary.state);
}

/** Plans that still have a rest to pay (Aktiv and Rückstand). */
export function isOpenPlan(detail: InstallmentDetail): boolean {
  return matchesPlanFilter(detail, 'active') && detail.summary.rest > 0;
}

export function debtorKindLabel(plan: InstallmentPlan): string {
  return plan.householdId ? 'Haushalt' : 'Externer Schuldner';
}

export function overdueRates(detail: InstallmentDetail): InstallmentRate[] {
  return detail.schedule.filter(rate => rate.status === 'OVERDUE');
}

export function paidRateCount(detail: InstallmentDetail): number {
  return detail.schedule.filter(rate => rate.status === 'PAID').length;
}

/** First rate that is not fully paid yet. */
export function nextOpenRate(detail: InstallmentDetail): InstallmentRate | undefined {
  return detail.schedule.find(rate => rate.status !== 'PAID');
}

/** Status part of a list row: next due date, or why there is none. */
export function nextDueText(detail: InstallmentDetail): string {
  if (detail.summary.state === 'CLOSED') return 'beendet';
  const rate = nextOpenRate(detail);
  return rate ? `nächste ${formatDate(rate.dueDate)}` : 'vollständig bezahlt';
}

/** „Rate 3/8 · fällig 19.05.2027“ for the next open rate; null when everything is paid. */
export function nextRateText(detail: InstallmentDetail): string | null {
  const rate = nextOpenRate(detail);
  return rate ? `Rate ${rate.number}/${detail.schedule.length} · fällig ${formatDate(rate.dueDate)}` : null;
}

/** Stage of the next reminder: the first mail is a Zahlungserinnerung, later ones a Mahnung. */
export function nextReminderLabel(detail: InstallmentDetail): string {
  return detail.reminders.length > 0 ? 'Mahnung' : 'Erinnerung';
}

/** Bank payers are written „Last, First“; plans use „First Last“. */
export function debtorNameFromPayer(payer: string): string {
  const trimmed = payer.trim();
  return trimmed.includes(',')
    ? trimmed.split(',').map(part => part.trim()).reverse().join(' ')
    : trimmed;
}

const FILE_NUMBER = /\d+\s*[A-ZÄÖÜ]+\s*\d+\/\d+/;

/** „Az. 10 C 644/24“ when the purpose names a file number, otherwise the purpose itself. */
export function referenceFromPurpose(purpose: string): string {
  const match = FILE_NUMBER.exec(purpose);
  return match ? `Az. ${match[0]}` : purpose.trim();
}

function toIso(year: number, month: number, day: number): string {
  return `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
}

/** Adds calendar months to a `YYYY-MM-DD` date; days past the month end clamp to it. */
function addMonths(iso: string, months: number): string {
  const [year, month, day] = iso.split('-').map(Number);
  const index = month - 1 + months;
  const targetYear = year + Math.floor(index / 12);
  const targetMonth = ((index % 12) + 12) % 12;
  const lastDay = new Date(Date.UTC(targetYear, targetMonth + 1, 0)).getUTCDate();
  return toIso(targetYear, targetMonth + 1, Math.min(day, lastDay));
}

export function addDays(iso: string, days: number): string {
  const date = new Date(`${iso}T00:00:00Z`);
  date.setUTCDate(date.getUTCDate() + days);
  return date.toISOString().slice(0, 10);
}

/** The first day of the month after `iso`, e.g. the default first rate date. */
export function firstOfNextMonth(iso: string): string {
  const [year, month] = iso.split('-').map(Number);
  return toIso(month === 12 ? year + 1 : year, month % 12 + 1, 1);
}

/**
 * Live hint in the plan form, e.g. „8 Raten à 300,00 € · monatlich · letzte Rate 19.05.2027“.
 * Returns null while the inputs are incomplete. Only for the form; the saved schedule comes from the API.
 */
export function schedulePreviewText(
  total: number, rate: number, intervalMonths: number, firstDue: string,
): string | null {
  if (!(total > 0 && rate > 0 && firstDue)) return null;
  const count = Math.ceil(total / rate - 1e-9);
  const lastAmount = Math.round((total - rate * (count - 1)) * 100) / 100;
  const lastDue = formatDate(addMonths(firstDue, (count - 1) * intervalMonths));
  let amounts: string;
  if (count === 1) amounts = `1 Rate à ${formatCurrency(lastAmount)}`;
  else if (lastAmount !== rate) {
    amounts = `${count} Raten (${count - 1} × ${formatCurrency(rate)} + 1 × ${formatCurrency(lastAmount)})`;
  } else amounts = `${count} Raten à ${formatCurrency(rate)}`;
  return `${amounts} · ${intervalLabel(intervalMonths)} · letzte Rate ${lastDue}`;
}
