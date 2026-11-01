import { computed, type Component } from 'vue';
import { useRoute } from 'vue-router';
import {
  LayoutDashboard, Inbox, Users, UserCircle, UserPlus, NotebookPen, Receipt, ClipboardList, Bell,
  Scale, Clock, UserCheck, Settings2, ShieldCheck, Home, IdCard, HandCoins, Heart,
} from 'lucide-vue-next';
import { useAuthStore } from '@/stores/auth';
import { useTaskCounts } from '@/composables/useTaskCounts';

export interface NavItem {
  name: string;
  /** Label in the mobile tab bar, if shorter than `name`. */
  short?: string;
  to: string;
  icon: Component;
  count?: number;
  /** Extra path prefixes that mark this item as active. */
  match?: string[];
}

export interface NavGroup {
  label: string;
  items: NavItem[];
}

/** Pages bundled under „Aufgaben“; each keeps its own URL. */
export const TASK_PATHS = ['/bankabgleich', '/meldungen', '/aenderungen'] as const;

export function useNavigation() {
  const auth = useAuthStore();
  const route = useRoute();
  const counts = useTaskCounts();

  const taskTabs = computed<NavItem[]>(() => [
    ...(auth.canAccessFees ? [{ name: 'Bankabgleich', to: '/bankabgleich', icon: Inbox, count: counts.bank.value }] : []),
    ...(auth.canAccessFees ? [
      { name: 'Meldungen', to: '/meldungen', icon: Inbox, count: counts.reports.value },
      { name: 'Änderungen', to: '/aenderungen', icon: Inbox, count: counts.changes.value },
    ] : []),
  ]);

  const taskTotal = computed(() => taskTabs.value.reduce((sum, tab) => sum + (tab.count ?? 0), 0));

  /** „Aufgaben“ opens the first section with open items. */
  const tasksTarget = computed(() => taskTabs.value.find(tab => tab.count)?.to ?? taskTabs.value[0]?.to ?? '/');

  const groups = computed<NavGroup[]>(() => {
    if (auth.isParent) {
      return [{ label: '', items: [
        { name: 'Übersicht', to: '/familie', icon: Home },
        { name: 'Beiträge', to: '/familie/beitraege', icon: Receipt },
        { name: 'Elternstunden', short: 'Stunden', to: '/familie/elternstunden', icon: Clock },
        { name: 'Meine Daten', short: 'Daten', to: '/familie/daten', icon: IdCard },
      ] }];
    }
    const result: NavGroup[] = [];
    if (auth.canAccessFees) {
      result.push(
        { label: '', items: [
          { name: 'Start', to: '/', icon: LayoutDashboard },
          { name: 'Aufgaben', to: tasksTarget.value, icon: Inbox, count: taskTotal.value, match: [...TASK_PATHS, '/aufgaben'] },
        ] },
        { label: 'Stammdaten', items: [
          { name: 'Kinder', to: '/kinder', icon: Users },
          { name: 'Eltern', to: '/eltern', icon: UserCircle },
          { name: 'Mitglieder', to: '/mitglieder', icon: UserPlus },
          { name: 'Notizen', to: '/notizen', icon: NotebookPen },
        ] },
        { label: 'Beiträge', items: [
          { name: 'Beiträge', to: '/beitraege', icon: Receipt },
          { name: 'Einstufungen', to: '/einstufungen', icon: ClipboardList },
          { name: 'Spenden', to: '/spenden', icon: Heart },
          ...(auth.isAdmin ? [
            { name: 'Ratenzahlungen', to: '/ratenzahlungen', icon: HandCoins },
            { name: 'Erinnerungen', to: '/automatisierung', icon: Bell },
          ] : []),
          { name: 'Beitragsordnung', to: '/beitragsordnung', icon: Scale },
        ] },
      );
    }
    if (!auth.canAccessFees && auth.isLeitung) {
      // Leitung: Stichtag-Dashboard, Stammdaten ohne Notizen, keine Beitrags- oder Bankbereiche.
      result.push({ label: '', items: [{ name: 'Start', to: '/', icon: LayoutDashboard }] });
      result.push({ label: 'Stammdaten', items: [
        { name: 'Kinder', to: '/kinder', icon: Users },
        { name: 'Eltern', to: '/eltern', icon: UserCircle },
        { name: 'Mitglieder', to: '/mitglieder', icon: UserPlus },
      ] });
    }
    if (auth.canAccessParentWork) {
      result.push({ label: 'Elternstunden', items: [
        { name: 'Übersicht', short: auth.isLeitung ? 'Stunden' : undefined, to: '/elternstunden', icon: Clock,
          match: ['/elternstunden/familien'] },
        ...(auth.canAccessFees ? [{ name: 'Vorstand', to: '/elternstunden/vorstand', icon: UserCheck }] : []),
        ...(auth.canViewParentWorkAdmin
          ? [{ name: 'Verwaltung', to: '/elternstunden/verwaltung', icon: Settings2 }] : []),
      ] });
    }
    if (auth.isAdmin) {
      result.push({ label: 'System', items: [{ name: 'Benutzer', to: '/benutzer', icon: ShieldCheck }] });
    }
    return result;
  });

  const allItems = computed(() => groups.value.flatMap(group => group.items));

  /** Quick tabs on phones: four destinations, everything else under „Mehr“. */
  const tabs = computed<NavItem[]>(() => {
    const find = (to: string) => allItems.value.find(item => item.to === to || item.match?.includes(to));
    let picks: (NavItem | undefined)[];
    if (auth.isLeitung) picks = [find('/'), find('/kinder'), find('/eltern'), find('/elternstunden')];
    else if (auth.isParent || !auth.canAccessFees) picks = allItems.value.slice(0, 4);
    else if (auth.isAdmin) picks = [find('/'), find('/aufgaben'), find('/kinder'), find('/automatisierung')];
    else picks = [find('/'), find('/aufgaben'), find('/kinder'), find('/beitraege')];
    return picks.filter((item): item is NavItem => !!item);
  });

  const hasMore = computed(() => allItems.value.length > tabs.value.length);

  function isActive(item: Pick<NavItem, 'to' | 'match'>) {
    const path = route.path;
    const prefixes = [item.to, ...(item.match ?? [])];
    return prefixes.some((prefix) => {
      // Overview pages only match themselves; their children belong to other items or are listed in `match`.
      if (prefix === '/' || prefix === '/familie' || prefix === '/elternstunden') return path === prefix;
      // Whole path segments: '/eltern' must not light up on '/elternstunden'.
      return path === prefix || path.startsWith(`${prefix}/`);
    });
  }

  const home = computed(() => {
    if (auth.isParent) return '/familie';
    return auth.canAccessFees || auth.isLeitung ? '/' : '/elternstunden';
  });

  return { groups, allItems, tabs, hasMore, taskTabs, taskTotal, isActive, home };
}
