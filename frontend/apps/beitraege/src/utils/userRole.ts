import type { UserRole } from '@/api/types';

export function userRoleLabel(role: UserRole): string {
  switch (role) {
    case 'ADMIN': return 'Administrator';
    case 'BOARD': return 'Vorstand';
    case 'PARENT_WORK': return 'Elternstunden';
    case 'LEITUNG': return 'Leitung';
    case 'PARENT': return 'Eltern';
  }
}
