// Labels and badge styles for the reminder worklist and the email log.
import type { ReminderCase, ReminderCaseFee } from '@/api/types';
import { getFeeTypeColor, getFeeTypeName } from '@/utils/fees';

export function formatPeriod(fee: ReminderCaseFee): string {
  if (fee.month > 0) return `${fee.month}/${fee.year}`;
  return String(fee.year);
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
  actionable_initial: 'Erinnerung fällig',
  actionable_final: 'Mahnung fällig',
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
      return 'bg-amber-100 dark:bg-amber-950/40 text-amber-700 dark:text-amber-300';
    case 'actionable_final':
      return 'bg-red-100 dark:bg-red-950/40 text-red-700 dark:text-red-300';
    case 'waiting':
      return 'bg-blue-100 dark:bg-blue-950/40 text-blue-700 dark:text-blue-300';
    case 'history_unknown':
      return 'bg-muted text-foreground';
    default:
      return 'bg-muted text-muted-foreground';
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
    case 'PASSWORD_RESET':
      return 'Passwort-Reset';
    default:
      return type;
  }
}
