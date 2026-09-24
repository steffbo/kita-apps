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
      return 'bg-amber-100 text-amber-700';
    case 'actionable_final':
      return 'bg-red-100 text-red-700';
    case 'waiting':
      return 'bg-blue-100 text-blue-700';
    case 'history_unknown':
      return 'bg-gray-200 text-gray-700';
    default:
      return 'bg-gray-100 text-gray-600';
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
    case 'PASSWORD_RESET':
      return 'Passwort-Reset';
    default:
      return type;
  }
}
