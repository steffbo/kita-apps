import type { Component } from 'vue';
import {
  CalendarDays,
  ClipboardCheck,
  Clock3,
  CreditCard,
  Database,
  Settings,
  UsersRound,
} from 'lucide-vue-next';

export type PortalRole = 'ADMIN' | 'BEITRAG' | 'VORSTAND' | 'LEITUNG' | 'TEAM' | 'PARENT';

export interface PortalModule {
  id: string;
  label: string;
  description: string;
  routeName: string;
  roles: PortalRole[];
  state: 'ready' | 'planned';
  icon: Component;
}

export const portalModules: PortalModule[] = [
  {
    id: 'parent_work',
    label: 'Elternstunden',
    description: 'Soll, Ist und Einreichungen je Kind',
    routeName: 'parent-work',
    roles: ['ADMIN', 'LEITUNG', 'TEAM', 'PARENT', 'VORSTAND'],
    state: 'ready',
    icon: ClipboardCheck,
  },
  {
    id: 'master_data',
    label: 'Stammdaten',
    description: 'Kinder, Eltern, Haushalte und Gruppen',
    routeName: 'dashboard',
    roles: ['ADMIN', 'BEITRAG', 'LEITUNG', 'VORSTAND'],
    state: 'planned',
    icon: Database,
  },
  {
    id: 'schedule',
    label: 'Dienstplan',
    description: 'Planung und Abwesenheiten',
    routeName: 'dashboard',
    roles: ['ADMIN', 'LEITUNG', 'TEAM'],
    state: 'planned',
    icon: CalendarDays,
  },
  {
    id: 'time_tracking',
    label: 'Zeiterfassung',
    description: 'Kommen, Gehen und Historie',
    routeName: 'dashboard',
    roles: ['ADMIN', 'LEITUNG', 'TEAM'],
    state: 'planned',
    icon: Clock3,
  },
  {
    id: 'fees',
    label: 'Beiträge',
    description: 'Beitragsverwaltung und Erinnerungen',
    routeName: 'dashboard',
    roles: ['ADMIN', 'BEITRAG'],
    state: 'planned',
    icon: CreditCard,
  },
  {
    id: 'admin',
    label: 'Administration',
    description: 'Benutzer, Rollen und Einladungen',
    routeName: 'admin',
    roles: ['ADMIN'],
    state: 'ready',
    icon: Settings,
  },
  {
    id: 'review',
    label: 'Abnahme',
    description: 'Eingereichte Elternstunden prüfen',
    routeName: 'review-queue',
    roles: ['ADMIN', 'LEITUNG', 'TEAM'],
    state: 'ready',
    icon: UsersRound,
  },
];
