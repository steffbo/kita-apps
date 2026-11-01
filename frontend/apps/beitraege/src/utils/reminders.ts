// Labels and badge styles for the reminder worklist and the email log.
import type { ReminderCase, ReminderCaseFee } from '@/api/types';
import { getFeeTypeColor, getFeeTypeName } from '@/utils/fees';

function periodLabel(year: number, month?: number): string {
  if (month && month > 0) return `${month}/${year}`;
  return String(year);
}

/** Period of a fee; a Mahngebühr names its base fee, e.g. "für Essensgeld 9/2026". */
export function formatPeriod(fee: ReminderCaseFee): string {
  if (fee.reminderFor) {
    return `für ${feeTypeLabel(fee.reminderFor.feeType)} ${periodLabel(fee.reminderFor.year, fee.reminderFor.month)}`;
  }
  return periodLabel(fee.year, fee.month);
}

export function feeTypeLabel(feeType: string): string {
  return getFeeTypeName(feeType);
}

export function feeChipClass(feeType: string): string {
  return getFeeTypeColor(feeType);
}

export function feeTypesIn(item: ReminderCase): string[] {
  const types = new Set(item.fees.map((fee) => fee.feeType));
  return Array.from(types);
}

const statusLabels: Record<string, string> = {
  actionable_initial: 'Nicht erinnert',
  actionable_final: 'Frist abgelaufen',
  waiting: 'In Frist',
  never_contacted: 'Nicht fällig',
  history_unknown: 'Historie unbekannt',
};

export function statusLabel(status: string): string {
  return statusLabels[status] ?? status;
}

export function statusBadgeClass(status: string): string {
  switch (status) {
    case 'actionable_initial':
      return 'badge badge-warning';
    case 'actionable_final':
      return 'badge badge-danger';
    case 'waiting':
      return 'badge badge-primary';
    default:
      return 'badge';
  }
}

export function formatEmailType(type: string): string {
  switch (type) {
    case 'REMINDER_INITIAL':
      return 'Zahlungserinnerung';
    case 'REMINDER_FINAL':
      return 'Mahnung';
    case 'MEMBERSHIP_REMINDER_INITIAL':
      return 'Vereinsbeitrag Erinnerung';
    case 'MEMBERSHIP_REMINDER_FINAL':
      return 'Vereinsbeitrag Mahnung';
    case 'ACCOUNT_INVITATION':
      return 'Kontoeinladung';
    case 'INSTALLMENT_REMINDER_INITIAL':
      return 'Ratenzahlung Erinnerung';
    case 'INSTALLMENT_REMINDER_FINAL':
      return 'Ratenzahlung Mahnung';
    case 'EINSTUFUNG':
      return 'Einstufung';
    case 'PASSWORD_RESET':
      return 'Passwort-Reset';
    default:
      return type;
  }
}
