// Shared helpers for children and households.
import type { IncomeStatus } from '@/api/types';

export function calculateAge(birthDate: string): number {
  const birth = new Date(birthDate);
  const today = new Date();
  let age = today.getFullYear() - birth.getFullYear();
  const m = today.getMonth() - birth.getMonth();
  if (m < 0 || (m === 0 && today.getDate() < birth.getDate())) {
    age--;
  }
  return age;
}

export function isUnderThree(birthDate: string): boolean {
  return calculateAge(birthDate) < 3;
}

export const incomeStatusOptions: { value: IncomeStatus; label: string }[] = [
  { value: '', label: 'Nicht festgelegt' },
  { value: 'PROVIDED', label: 'Einkommen angegeben' },
  { value: 'MAX_ACCEPTED', label: 'Höchstsatz akzeptiert' },
  { value: 'PENDING', label: 'Dokumente ausstehend' },
  { value: 'NOT_REQUIRED', label: 'Nicht erforderlich (Kind >3J bei Eintritt)' },
  { value: 'HISTORIC', label: 'Historisch (Kind jetzt >3J)' },
  { value: 'FOSTER_FAMILY', label: 'Pflegefamilie (Durchschnittsbeitrag)' },
];

export function getIncomeStatusLabel(status?: IncomeStatus): string {
  if (!status) return 'Nicht festgelegt';
  return incomeStatusOptions.find((option) => option.value === status)?.label ?? 'Nicht festgelegt';
}

/** Formats weekly care hours, e.g. "45 Std./Woche". */
export function formatCareHours(careHours?: number | null): string {
  if (careHours === undefined || careHours === null) return 'Unbekannt';
  return `${careHours} Std./Woche`;
}
