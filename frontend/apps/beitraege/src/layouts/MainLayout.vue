<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router';
import { Menu, Search, VenetianMask, ChevronsUpDown, Eye } from 'lucide-vue-next';
import { useAuthStore } from '@/stores/auth';
import { useNavigation, TASK_PATHS } from '@/composables/useNavigation';
import { refreshTaskCounts } from '@/composables/useTaskCounts';
import { userRoleLabel } from '@/utils/userRole';
import ChangePasswordDialog from '@/components/ChangePasswordDialog.vue';
import AppSheet from '@/components/ui/AppSheet.vue';
import NavList from '@/components/shell/NavList.vue';
import UserMenuPanel from '@/components/shell/UserMenuPanel.vue';
import CommandPalette from '@/components/shell/CommandPalette.vue';
import logo from '@/assets/knirpsenstadt-logo.png';

// Shell: sidebar from `lg` up; below that a top bar and a bottom tab bar with „Mehr“ for the rest.
const authStore = useAuthStore();
const route = useRoute();
const router = useRouter();
const { tabs, hasMore, isActive, home } = useNavigation();

const userMenuOpen = ref(false);
const moreOpen = ref(false);
const searchOpen = ref(false);
const showChangePassword = ref(false);
const passwordChanged = ref(false);

// The account menu is a popover in the sidebar and a sheet on smaller screens.
const desktopQuery = window.matchMedia('(min-width: 1024px)');
const isDesktop = ref(desktopQuery.matches);
function onDesktopChange(event: MediaQueryListEvent) {
  isDesktop.value = event.matches;
  closeMenus();
}

const initials = computed(() => {
  const user = authStore.user;
  return ((user?.firstName?.[0] ?? '') + (user?.lastName?.[0] ?? '') || user?.email?.[0] || '?').toUpperCase();
});
const displayName = computed(() => personName(authStore.user));
const moreActive = computed(() => !tabs.value.some(tab => isActive(tab)));

function isTaskPath(path: string) {
  return (TASK_PATHS as readonly string[]).includes(path);
}

function closeMenus() {
  userMenuOpen.value = false;
  moreOpen.value = false;
  searchOpen.value = false;
}

watch(() => route.path, (path, previous) => {
  closeMenus();
  // Actions on the task pages change the counts; elsewhere a periodic refresh is enough.
  refreshTaskCounts(isTaskPath(path) || isTaskPath(previous ?? ''));
});

function onKeydown(event: KeyboardEvent) {
  if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
    event.preventDefault();
    searchOpen.value = !searchOpen.value;
  }
}

function onDocumentClick(event: MouseEvent) {
  if (isDesktop.value && !(event.target as Element).closest('[data-user-menu]')) userMenuOpen.value = false;
}

onMounted(() => {
  refreshTaskCounts(true);
  document.addEventListener('keydown', onKeydown);
  document.addEventListener('click', onDocumentClick);
  desktopQuery.addEventListener('change', onDesktopChange);
});
onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown);
  document.removeEventListener('click', onDocumentClick);
  desktopQuery.removeEventListener('change', onDesktopChange);
});

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
  <div class="min-h-dvh lg:flex">
    <!-- Sidebar (desktop) -->
    <aside class="sticky top-0 hidden h-dvh w-60 shrink-0 flex-col border-r bg-card px-3 py-3 lg:flex">
      <RouterLink :to="home" class="mb-3 flex items-center gap-2.5 rounded-lg px-2 py-1 font-semibold tracking-tight">
        <img :src="logo" alt="" class="h-7 w-7 rounded-lg object-cover" />
        Knirpsenstadt
      </RouterLink>
      <button type="button"
        class="mb-4 flex h-9 w-full items-center gap-2 rounded-lg border bg-background px-2.5 text-sm text-muted-foreground hover:border-input"
        @click="searchOpen = true">
        <Search class="h-4 w-4" />Suchen
        <kbd class="ml-auto rounded border px-1.5 font-sans text-[11px]">Strg K</kbd>
      </button>
      <nav aria-label="Hauptnavigation" class="-mx-1 min-h-0 flex-1 overflow-y-auto px-1">
        <NavList />
      </nav>
      <div class="relative mt-2 border-t pt-2" data-user-menu>
        <button type="button" aria-label="Benutzermenü" :aria-expanded="userMenuOpen"
          class="flex h-12 w-full items-center gap-2.5 rounded-lg px-2 text-left hover:bg-accent"
          @click="userMenuOpen = !userMenuOpen">
          <span class="grid h-8 w-8 shrink-0 place-items-center rounded-full bg-muted text-xs font-semibold text-muted-foreground">{{ initials }}</span>
          <span class="min-w-0 flex-1">
            <span class="block truncate text-sm font-medium">{{ displayName }}</span>
            <span class="block truncate text-xs text-muted-foreground">{{ authStore.user ? userRoleLabel(authStore.user.role) : '' }}</span>
          </span>
          <ChevronsUpDown class="h-4 w-4 text-muted-foreground" />
        </button>
        <div v-if="userMenuOpen && isDesktop"
          class="absolute bottom-full left-0 z-50 mb-2 w-72 rounded-xl border bg-popover p-3 text-popover-foreground shadow-lg">
          <UserMenuPanel @change-password="openChangePassword" @stop-impersonation="stopImpersonation" @logout="handleLogout" />
        </div>
      </div>
    </aside>

    <div class="min-w-0 flex-1">
      <!-- Top bar (phone, tablet) -->
      <header class="sticky top-0 z-40 flex h-14 items-center gap-1 border-b bg-card/95 pl-4 pr-2 backdrop-blur lg:hidden">
        <RouterLink :to="home" class="flex min-w-0 flex-1 items-center gap-2.5 font-semibold tracking-tight">
          <img :src="logo" alt="" class="h-7 w-7 rounded-lg object-cover" />
          <span class="truncate">Knirpsenstadt</span>
        </RouterLink>
        <button type="button" aria-label="Suchen" class="grid h-10 w-10 place-items-center rounded-lg hover:bg-accent"
          @click="searchOpen = true"><Search class="h-5 w-5" /></button>
        <button type="button" aria-label="Benutzermenü" :aria-expanded="userMenuOpen"
          class="grid h-10 w-10 place-items-center rounded-lg hover:bg-accent" @click="userMenuOpen = true">
          <span class="grid h-8 w-8 place-items-center rounded-full bg-muted text-xs font-semibold text-muted-foreground">{{ initials }}</span>
        </button>
      </header>

      <div v-if="authStore.isImpersonating" data-testid="impersonation-indicator" role="status"
        :aria-label="`Impersonation: ${personName(authStore.impersonator)} als ${personName(authStore.user)}`"
        class="flex items-center gap-2 border-b border-warning-line bg-warning-soft px-4 py-2 text-sm font-medium text-warning lg:px-8">
        <VenetianMask class="h-4 w-4 shrink-0" />
        <span class="min-w-0 truncate">
          Impersonation: {{ personName(authStore.impersonator) }} als {{ personName(authStore.user) }}
        </span>
        <button type="button" class="ml-auto shrink-0 rounded-md px-2 py-1 underline-offset-2 hover:underline"
          @click="stopImpersonation">Beenden</button>
      </div>

      <div v-if="authStore.isReadOnly" data-testid="read-only-indicator" role="status"
        class="flex items-center gap-2 border-b bg-muted px-4 py-2 text-sm text-muted-foreground lg:px-8">
        <Eye class="h-4 w-4 shrink-0" />
        <span class="min-w-0 truncate">Vorstand: Du siehst alle Daten, kannst aber nichts ändern.</span>
      </div>

      <main class="mx-auto min-w-0 max-w-7xl px-4 pb-[calc(5.5rem+env(safe-area-inset-bottom))] pt-5 sm:px-6 lg:px-8 lg:py-8">
        <RouterView />
      </main>
    </div>

    <!-- Tab bar (phone, tablet) -->
    <nav aria-label="Schnellnavigation"
      class="fixed inset-x-0 bottom-0 z-40 flex justify-around border-t bg-card/95 px-1 pb-[max(0.5rem,env(safe-area-inset-bottom))] pt-1.5 backdrop-blur lg:hidden">
      <RouterLink v-for="tab in tabs" :key="tab.name" :to="tab.to"
        class="relative flex min-w-14 flex-col items-center gap-0.5 text-[11px] font-medium"
        :class="isActive(tab) ? 'text-primary-text' : 'text-muted-foreground'"
        :aria-current="isActive(tab) ? 'page' : undefined">
        <span class="grid h-8 w-14 place-items-center rounded-full transition-colors" :class="isActive(tab) && 'bg-primary-soft'">
          <component :is="tab.icon" class="h-5 w-5" />
        </span>
        {{ tab.short ?? tab.name }}
        <span v-if="tab.count" class="absolute left-[calc(50%+6px)] top-0 grid h-[18px] min-w-[18px] place-items-center rounded-full border-2 border-card bg-destructive px-1 text-[10px] font-semibold text-destructive-foreground tabular-nums">
          {{ tab.count > 99 ? '99+' : tab.count }}
        </span>
      </RouterLink>
      <button v-if="hasMore" type="button" :aria-expanded="moreOpen"
        class="flex min-w-14 flex-col items-center gap-0.5 text-[11px] font-medium"
        :class="moreActive ? 'text-primary-text' : 'text-muted-foreground'" @click="moreOpen = true">
        <span class="grid h-8 w-14 place-items-center rounded-full" :class="moreActive && 'bg-primary-soft'">
          <Menu class="h-5 w-5" />
        </span>
        Mehr
      </button>
    </nav>

    <AppSheet v-if="moreOpen" title="Navigation" hide-title @close="moreOpen = false">
      <nav aria-label="Alle Seiten"><NavList large /></nav>
    </AppSheet>
    <AppSheet v-if="userMenuOpen && !isDesktop" title="Konto" size="sm" @close="userMenuOpen = false">
      <UserMenuPanel @change-password="openChangePassword" @stop-impersonation="stopImpersonation" @logout="handleLogout" />
    </AppSheet>
    <CommandPalette v-if="searchOpen" @close="searchOpen = false" />

    <ChangePasswordDialog v-if="showChangePassword" @close="showChangePassword = false"
      @changed="onPasswordChanged" />
    <div v-if="passwordChanged" role="status"
      class="fixed bottom-24 right-4 z-[60] rounded-xl border bg-card px-4 py-3 text-sm text-card-foreground shadow lg:bottom-4">
      Passwort geändert.
    </div>
  </div>
</template>
