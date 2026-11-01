import type { UserRole } from '@/api/types';

/** Roles an account can hold, most privileged first (domain.AssignableRoles). */
export const ASSIGNABLE_ROLES: UserRole[] = ['ADMIN', 'BOARD', 'LEITUNG', 'PARENT_WORK', 'PARENT'];

export function userRoleLabel(role: UserRole): string {
  switch (role) {
    case 'ADMIN': return 'Administrator';
    case 'BOARD': return 'Vorstand';
    case 'PARENT_WORK': return 'Elternstunden';
    case 'LEITUNG': return 'Leitung';
    case 'PARENT': return 'Eltern';
  }
}
