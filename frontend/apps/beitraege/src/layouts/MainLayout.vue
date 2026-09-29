<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';
import { useTheme } from '@/composables/useTheme';
import { userRoleLabel } from '@/utils/userRole';
import ChangePasswordDialog from '@/components/ChangePasswordDialog.vue';
import logo from '@/assets/knirpsenstadt-logo.png';
import {
  LayoutDashboard, Users, UserCircle, UserPlus, Receipt, RefreshCw, Bell, LogOut,
  Menu, X, ChevronDown, ClipboardList, NotebookPen, Scale, KeyRound, ShieldCheck,
  Clock, UserCheck, Monitor, Sun, Moon, FileWarning, VenetianMask, Undo2, History,
} from 'lucide-vue-next';

const authStore = useAuthStore();
const route = useRoute();
const router = useRouter();
const { mode, setTheme } = useTheme();
const mobileMenuOpen = ref(false);
const openMenu = ref<string | null>(null);
const showChangePassword = ref(false);
const passwordChanged = ref(false);

const baseNavGroups = [
  { label: 'Täglich', items: [
    { name: 'Dashboard', to: '/', icon: LayoutDashboard },
    { name: 'Bankabgleich', to: '/bankabgleich', icon: RefreshCw },
    { name: 'Meldungen', to: '/meldungen', icon: FileWarning },
  ] },
  { label: 'Verwaltung', items: [
    { name: 'Kinder', to: '/kinder', icon: Users },
    { name: 'Eltern', to: '/eltern', icon: UserCircle },
    { name: 'Mitglieder', to: '/mitglieder', icon: UserPlus },
    { name: 'Notizen', to: '/notizen', icon: NotebookPen },
    { name: 'Änderungen von Eltern', to: '/aenderungen', icon: History },
  ] },
  { label: 'Beiträge', items: [
    { name: 'Beiträge', to: '/beitraege', icon: Receipt },
    { name: 'Einstufungen', to: '/einstufungen', icon: ClipboardList },
    { name: 'Erinnerungen', to: '/automatisierung', icon: Bell },
    { name: 'Beitragsordnung', to: '/beitragsordnung', icon: Scale },
  ] },
];

const parentLinks = [
  { name: 'Übersicht', to: '/familie', icon: LayoutDashboard },
  { name: 'Beiträge', to: '/familie/beitraege', icon: Receipt },
  { name: 'Elternstunden', to: '/familie/elternstunden', icon: Clock },
  { name: 'Meine Daten', to: '/familie/daten', icon: UserCircle },
];
const navGroups = computed(() => [
  ...(authStore.canAccessFees ? baseNavGroups.map(group => ({
    ...group,
    items: group.items.filter(item => !['/automatisierung', '/meldungen', '/aenderungen'].includes(item.to) || authStore.isAdmin),
  })) : []),
  ...(authStore.canAccessParentWork ? [{ label: 'Elternstunden', items: [
    { name: 'Übersicht', to: '/elternstunden', icon: Clock },
    ...(authStore.isAdmin ? [{ name: 'Vorstand', to: '/elternstunden/vorstand', icon: UserCheck }] : []),
    { name: 'Verwaltung', to: '/elternstunden/verwaltung', icon: Scale },
  ] }] : []),
  ...(authStore.isAdmin ? [{ label: 'System', items: [
    { name: 'Benutzer', to: '/benutzer', icon: ShieldCheck },
  ] }] : []),
]);

function isActive(path: string) {
  if (path === '/familie') return route.path === '/familie';
  if (path === '/') return route.path === '/';
  if (path === '/elternstunden') {
    return route.path === path || route.path.startsWith('/elternstunden/familien/');
  }
  return route.path.startsWith(path);
}

function groupActive(items: { to: string }[]) {
  return items.some(item => isActive(item.to));
}

function closeMenus() {
  openMenu.value = null;
  mobileMenuOpen.value = false;
}

function onDocumentClick(event: MouseEvent) {
  if (!(event.target as Element).closest('[data-header-menu]')) openMenu.value = null;
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') closeMenus();
}

watch(() => route.fullPath, closeMenus);
onMounted(() => {
  document.addEventListener('click', onDocumentClick);
  document.addEventListener('keydown', onKeydown);
});
onUnmounted(() => {
  document.removeEventListener('click', onDocumentClick);
  document.removeEventListener('keydown', onKeydown);
});

function cycleTheme() {
  setTheme(mode.value === 'system' ? 'light' : mode.value === 'light' ? 'dark' : 'system');
}

async function handleLogout() {
  closeMenus();
  await authStore.logout();
  router.push('/login');
}

function personName(person: { firstName?: string | null; lastName?: string | null; email: string } | null) {
  if (!person) return '';
  return [person.firstName, person.lastName].filter(Boolean).join(' ') || person.email;
}

function stopImpersonation() {
  closeMenus();
  authStore.stopImpersonation();
}

function openChangePassword() {
  closeMenus();
  showChangePassword.value = true;
}

function onPasswordChanged() {
  showChangePassword.value = false;
  passwordChanged.value = true;
  setTimeout(() => (passwordChanged.value = false), 5000);
}
</script>

<template>
  <div class="min-h-screen min-w-0 bg-background">
    <header class="sticky top-0 z-50 border-b border-brand-200 bg-header text-header-foreground shadow-sm">
      <div class="mx-auto flex max-w-7xl items-center gap-4 px-4 py-2 sm:px-6">
        <RouterLink :to="authStore.isParent ? '/familie' : '/'" class="flex shrink-0 items-center gap-2 rounded-full focus-visible:ring-2 focus-visible:ring-ring">
          <img :src="logo" alt="" class="h-10 w-10 rounded-full object-cover" />
          <span class="hidden font-bold sm:inline">Kita Knirpsenstadt</span>
        </RouterLink>

        <nav aria-label="Hauptnavigation" class="hidden min-w-0 flex-1 items-center justify-center gap-1 lg:flex">
          <RouterLink v-for="item in authStore.isParent ? parentLinks : []" :key="item.to"
            :to="item.to" class="rounded-full px-3 py-2 text-sm font-bold hover:bg-brand-50/70
              dark:hover:bg-brand-800" :class="{ 'bg-brand-50/80 dark:bg-brand-800': isActive(item.to) }">
            {{ item.name }}
          </RouterLink>
          <div v-for="group in navGroups" :key="group.label" class="relative" data-header-menu>
            <RouterLink
              v-if="group.items.length === 1" :to="group.items[0].to"
              class="flex items-center gap-2 rounded-full px-4 py-2 text-sm font-bold hover:bg-brand-50/70 dark:hover:bg-brand-800"
              :class="{ 'bg-brand-50/80 dark:bg-brand-800': groupActive(group.items) }"
            >
              <component :is="group.items[0].icon" class="h-4 w-4" />
              {{ group.items[0].name }}
            </RouterLink>
            <template v-else>
              <!-- Split button: the label opens the group's first page, the chevron lists all pages. -->
              <div
                class="flex items-center rounded-full hover:bg-brand-50/70 dark:hover:bg-brand-800"
                :class="{ 'bg-brand-50/80 dark:bg-brand-800': groupActive(group.items) }"
              >
                <RouterLink
                  :to="group.items[0].to"
                  class="rounded-l-full py-2 pl-4 pr-2 text-sm font-bold"
                  @click="closeMenus"
                >{{ group.label }}</RouterLink>
                <span aria-hidden="true" class="h-5 w-px bg-header-foreground/25" />
                <button
                  type="button" class="rounded-r-full py-2 pl-1.5 pr-3"
                  :aria-label="`${group.label}: alle Seiten`"
                  :aria-expanded="openMenu === group.label"
                  :aria-controls="'menu-' + group.label"
                  @click="openMenu = openMenu === group.label ? null : group.label"
                ><ChevronDown class="h-4 w-4" /></button>
              </div>
              <div
                v-if="openMenu === group.label" :id="'menu-' + group.label"
                class="absolute left-0 top-full z-50 mt-2 min-w-48 rounded-2xl border bg-popover p-2 text-popover-foreground shadow-lg"
              >
                <RouterLink
                  v-for="item in group.items" :key="item.to" :to="item.to"
                  class="flex items-center gap-3 whitespace-nowrap rounded-xl px-3 py-2 text-sm hover:bg-accent"
                  :class="{ 'bg-accent font-bold': isActive(item.to) }" @click="closeMenus"
                ><component :is="item.icon" class="h-4 w-4" />{{ item.name }}</RouterLink>
              </div>
            </template>
          </div>
        </nav>

        <div class="ml-auto flex items-center gap-2">
          <div
            v-if="authStore.isImpersonating" data-testid="impersonation-indicator" role="status"
            :aria-label="`Impersonation: ${personName(authStore.impersonator)} als ${personName(authStore.user)}`"
            class="flex min-w-0 items-center gap-2 rounded-full bg-amber-400 px-3 py-1 text-sm font-bold text-amber-950 ring-2 ring-amber-600 dark:bg-amber-500 dark:ring-amber-300"
          >
            <VenetianMask class="h-4 w-4 shrink-0" />
            <span class="hidden truncate md:inline">
              Impersonation: {{ personName(authStore.impersonator) }} als {{ personName(authStore.user) }}
            </span>
            <span class="md:hidden">Impersonation</span>
          </div>
          <button
            type="button" class="rounded-full p-2 hover:bg-brand-50/70 dark:hover:bg-accent"
            :aria-label="'Design: ' + (mode === 'system' ? 'System' : mode === 'light' ? 'Hell' : 'Dunkel') + '. Umschalten'"
            :title="'Design: ' + (mode === 'system' ? 'System' : mode === 'light' ? 'Hell' : 'Dunkel')"
            @click="cycleTheme"
          >
            <Monitor v-if="mode === 'system'" class="h-5 w-5" />
            <Sun v-else-if="mode === 'light'" class="h-5 w-5" />
            <Moon v-else class="h-5 w-5" />
          </button>
          <div class="relative" data-header-menu>
            <button
              type="button" aria-label="Benutzermenü" :aria-expanded="openMenu === 'Benutzer'"
              class="flex items-center gap-2 rounded-full px-2 py-1 hover:bg-brand-50/70 dark:hover:bg-accent"
              @click="openMenu = openMenu === 'Benutzer' ? null : 'Benutzer'"
            >
              <UserCircle class="h-7 w-7" />
              <span class="hidden max-w-32 text-left sm:block">
                <span class="block truncate text-sm font-bold">{{ authStore.user?.firstName || authStore.user?.email }}</span>
                <span class="block truncate text-xs">{{ authStore.user ? userRoleLabel(authStore.user.role) : '' }}</span>
              </span>
              <ChevronDown class="hidden h-4 w-4 sm:block" />
            </button>
            <div
              v-if="openMenu === 'Benutzer'"
              class="absolute right-0 top-full z-50 mt-2 min-w-48 rounded-2xl border bg-popover p-2 text-popover-foreground shadow-lg"
            >
              <button v-if="authStore.isImpersonating" data-testid="stop-impersonation"
                class="flex w-full items-center gap-2 rounded-xl px-3 py-2 text-left text-sm font-bold hover:bg-accent"
                @click="stopImpersonation"><Undo2 class="h-4 w-4" />Impersonation beenden</button>
              <button v-else class="flex w-full items-center gap-2 rounded-xl px-3 py-2 text-left text-sm hover:bg-accent"
                @click="openChangePassword"><KeyRound class="h-4 w-4" />Passwort ändern</button>
              <button class="flex w-full items-center gap-2 rounded-xl px-3 py-2 text-left text-sm hover:bg-accent"
                @click="handleLogout"><LogOut class="h-4 w-4" />Abmelden</button>
            </div>
          </div>
          <button
            type="button" class="rounded-full p-2 hover:bg-brand-50/70 lg:hidden"
            :aria-label="mobileMenuOpen ? 'Menü schließen' : 'Menü öffnen'"
            :aria-expanded="mobileMenuOpen" @click="mobileMenuOpen = !mobileMenuOpen"
          >
            <X v-if="mobileMenuOpen" class="h-6 w-6" />
            <Menu v-else class="h-6 w-6" />
          </button>
        </div>
      </div>
      <nav v-if="mobileMenuOpen" aria-label="Mobile Navigation"
        class="max-h-[calc(100vh-4rem)] overflow-y-auto border-t bg-header px-4 py-4 lg:hidden">
        <RouterLink v-for="item in authStore.isParent ? parentLinks : []" :key="item.to"
          :to="item.to" class="block rounded-xl px-3 py-2 text-sm font-bold
            hover:bg-brand-50/70 dark:hover:bg-brand-800" @click="closeMenus">{{ item.name }}</RouterLink>
        <div v-for="group in navGroups" :key="group.label" class="mb-4">
          <p class="mb-1 px-3 text-xs font-bold uppercase tracking-wide">{{ group.label }}</p>
          <RouterLink
            v-for="item in group.items" :key="item.to" :to="item.to"
            class="flex items-center gap-3 rounded-xl px-3 py-2 text-sm hover:bg-brand-50/70 dark:hover:bg-brand-800"
            :class="{ 'bg-brand-50/80 font-bold': isActive(item.to) }" @click="closeMenus"
          ><component :is="item.icon" class="h-4 w-4" />{{ item.name }}</RouterLink>
        </div>
      </nav>
    </header>

    <main class="mx-auto min-w-0 max-w-7xl px-4 py-6 sm:px-6 lg:py-8"><RouterView /></main>
    <ChangePasswordDialog v-if="showChangePassword" @close="showChangePassword = false"
      @changed="onPasswordChanged" />
    <div v-if="passwordChanged" role="status"
      class="fixed bottom-4 right-4 z-[60] rounded-2xl border bg-card px-4 py-3 text-sm text-card-foreground shadow">
      Passwort geändert.
    </div>
  </div>
</template>
