import type { UserRole } from '@/api/types';

export function userRoleLabel(role: UserRole): string {
  switch (role) {
    case 'ADMIN': return 'Administrator';
    case 'USER': return 'Benutzer';
    case 'PARENT_WORK': return 'Elternstunden';
    case 'PARENT': return 'Eltern';
  }
}
