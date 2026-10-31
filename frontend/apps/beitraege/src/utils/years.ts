import { todayISO } from './format';

/** Fees, Einstufungen and Elternstunden are kept in the app from 2025 (Kita-Jahr 2025/26) on. */
export const FIRST_DATA_YEAR = 2025;

/** Current calendar year in Europe/Berlin. */
export function currentYear(): number {
  return Number(todayISO().slice(0, 4));
}

/** Current Kita-Jahr in Europe/Berlin (1 August to 31 July), named by its starting year. */
export function currentKitaYear(): number {
  return kitaYearOf(todayISO());
}

/** Kita-Jahr containing an ISO date (`YYYY-MM-DD…`). */
export function kitaYearOf(date: string): number {
  const year = Number(date.slice(0, 4));
  return Number(date.slice(5, 7)) >= 8 ? year : year - 1;
}

/** `2026/27` for the Kita-Jahr starting in 2026. */
export function kitaYearLabel(year: number): string {
  return `${year}/${String((year + 1) % 100).padStart(2, '0')}`;
}

/**
 * Year choices for a dropdown, newest first: every year from the first year with data up to the
 * year after `current`, plus any further year with data. Independent of the selected year, so
 * picking an old year never hides the current one.
 */
export function yearOptions(current: number, dataYears: number[] = [FIRST_DATA_YEAR]): number[] {
  const first = Math.min(current, ...dataYears);
  const last = Math.max(current + 1, ...dataYears);
  return Array.from({ length: last - first + 1 }, (_, i) => last - i);
}
