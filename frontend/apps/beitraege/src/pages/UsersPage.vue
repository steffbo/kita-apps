<script setup lang="ts">
import { RouterLink } from 'vue-router';
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { KeyRound, Loader2, MailPlus, Pencil, Plus, Send, VenetianMask } from 'lucide-vue-next';
import { api } from '@/api';
import type { UserAccount, UserRole } from '@/api/types';
import { useAuthStore } from '@/stores/auth';
import { formatDate } from '@/utils/format';
import { userRoleLabel } from '@/utils/userRole';
import { useTableSort } from '@/composables/useTableSort';
import SearchInput from '@/components/SearchInput.vue';
import SortTh from '@/components/SortTh.vue';
import UserFormDialog from '@/components/users/UserFormDialog.vue';
import SetUserPasswordDialog from '@/components/users/SetUserPasswordDialog.vue';
import InviteParentsDialog from '@/components/users/InviteParentsDialog.vue';

const authStore = useAuthStore();

const users = ref<UserAccount[]>([]);
const isLoading = ref(true);
const loadError = ref<string | null>(null);
const notice = ref<string | null>(null);
const actionError = ref<string | null>(null);
const impersonatingId = ref<string | null>(null);

// Dialog state: `formUser === null` with showForm = create, otherwise edit.
const showForm = ref(false);
const formUser = ref<UserAccount | null>(null);
const passwordUser = ref<UserAccount | null>(null);
const showInvitations = ref(false);
const resendingId = ref<string | null>(null);

const search = ref('');
const roleFilter = ref<UserRole | ''>('');
const roles: UserRole[] = ['ADMIN', 'USER', 'PARENT_WORK', 'PARENT'];
const filteredUsers = computed(() => {
  const q = search.value.trim().toLocaleLowerCase('de');
  return users.value.filter(u => (!roleFilter.value || u.role === roleFilter.value)
    && (!q || [displayName(u), u.email, u.parentName ?? ''].some(v => v.toLocaleLowerCase('de').includes(q))));
});
const { sortKey, sortDir, sorted, toggle } = useTableSort(filteredUsers, {
  name: u => displayName(u),
  email: u => u.email,
  role: u => userRoleLabel(u.role),
  parent: u => u.parentName,
  status: u => statusLabel(u),
  createdAt: u => u.createdAt,
}, { key: 'name' });

async function loadUsers() {
  isLoading.value = true;
  loadError.value = null;
  try {
    users.value = await api.getUsers();
  } catch (e) {
    loadError.value = e instanceof Error ? e.message : 'Benutzer konnten nicht geladen werden';
  } finally {
    isLoading.value = false;
  }
}

function displayName(user: UserAccount): string {
  return [user.firstName, user.lastName].filter(Boolean).join(' ') || '—';
}

function statusLabel(user: UserAccount): string {
  if (user.invitationPending) return 'Einladung ausstehend';
  return user.isActive ? 'Aktiv' : 'Deaktiviert';
}

function statusClass(user: UserAccount): string {
  if (user.invitationPending) {
    return 'bg-amber-100 dark:bg-amber-950/40 text-amber-700 dark:text-amber-300';
  }
  return user.isActive
    ? 'bg-green-100 dark:bg-green-950/40 text-green-700 dark:text-green-300'
    : 'bg-muted text-muted-foreground';
}

function isSelf(user: UserAccount | null): boolean {
  return !!user && user.id === authStore.user?.id;
}

function canImpersonate(user: UserAccount): boolean {
  return user.isActive && user.role !== 'ADMIN' && !isSelf(user);
}

async function impersonate(user: UserAccount) {
  actionError.value = null;
  impersonatingId.value = user.id;
  try {
    await authStore.startImpersonation(user.id);
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : 'Anmeldung als Benutzer fehlgeschlagen';
    impersonatingId.value = null;
  }
}

function openCreate() {
  formUser.value = null;
  showForm.value = true;
}

function openEdit(user: UserAccount) {
  formUser.value = user;
  showForm.value = true;
}

async function onSaved() {
  const self = isSelf(formUser.value);
  notice.value = formUser.value ? 'Benutzer gespeichert.' : 'Benutzer angelegt.';
  showForm.value = false;
  await loadUsers();
  if (self) await authStore.fetchUser();
}

function onPasswordSaved() {
  notice.value = `Passwort für ${passwordUser.value?.email} neu gesetzt.`;
  passwordUser.value = null;
}

async function resendInvitation(user: UserAccount) {
  actionError.value = null;
  resendingId.value = user.id;
  try {
    await api.resendInvitation(user.id);
    notice.value = `Einladung an ${user.email} erneut gesendet.`;
  } catch (e) {
    actionError.value = e instanceof Error ? e.message : 'Einladung konnte nicht gesendet werden.';
  } finally {
    resendingId.value = null;
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key !== 'Escape') return;
  showForm.value = false;
  passwordUser.value = null;
  showInvitations.value = false;
}

onMounted(() => {
  loadUsers();
  document.addEventListener('keydown', handleKeydown);
});
onUnmounted(() => document.removeEventListener('keydown', handleKeydown));
</script>

<template>
  <div>
    <div class="mb-6 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div>
        <h1 class="text-2xl font-bold text-foreground">Benutzer</h1>
        <p class="text-muted-foreground mt-1 max-w-3xl">
          Zugänge zur Beitragsverwaltung. Deaktivierte Benutzer können sich nicht mehr anmelden;
          laufende Sitzungen enden spätestens nach 15 Minuten.
        </p>
      </div>
      <div class="flex flex-wrap gap-2">
        <button type="button" @click="showInvitations = true"
          class="inline-flex items-center gap-2 rounded-lg border px-4 py-2 hover:bg-accent">
          <MailPlus class="h-4 w-4" /> Eltern einladen
        </button>
        <button @click="openCreate"
          class="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2
            text-primary-foreground hover:bg-primary/90">
          <Plus class="h-4 w-4" /> Benutzer anlegen
        </button>
      </div>
    </div>

    <div v-if="notice" class="mb-4 p-3 bg-green-50 dark:bg-green-950/40 border border-green-200 rounded-lg text-sm text-green-700 dark:text-green-300 flex justify-between gap-2">
      <span>{{ notice }}</span>
      <button class="text-green-700 dark:text-green-300 underline" @click="notice = null">OK</button>
    </div>

    <div v-if="actionError" class="mb-4 p-3 bg-red-50 dark:bg-red-950/40 border border-red-200 rounded-lg text-sm text-red-600 dark:text-red-300 flex justify-between gap-2" role="alert">
      <span>{{ actionError }}</span>
      <button class="underline" @click="actionError = null">OK</button>
    </div>

    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>
    <div v-else-if="loadError" class="bg-red-50 dark:bg-red-950/40 border border-red-200 rounded-lg p-4" role="alert">
      <p class="text-red-600 dark:text-red-300">{{ loadError }}</p>
      <button @click="loadUsers()" class="mt-2 text-sm text-red-700 dark:text-red-300 underline">Erneut versuchen</button>
    </div>

    <div v-else class="space-y-4">
      <div class="flex flex-col gap-3 sm:flex-row">
        <SearchInput v-model="search" placeholder="Suchen nach Name, E-Mail oder Elternteil..." class="flex-1" />
        <select v-model="roleFilter" aria-label="Rolle" class="rounded-lg border px-3 py-2">
          <option value="">Alle Rollen</option>
          <option v-for="role in roles" :key="role" :value="role">{{ userRoleLabel(role) }}</option>
        </select>
      </div>
      <div class="bg-card rounded-xl border overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="bg-muted text-left text-muted-foreground">
            <tr>
              <SortTh label="Name" column="name" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
              <SortTh label="E-Mail" column="email" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
              <SortTh label="Rolle" column="role" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
              <SortTh label="Elternteil" column="parent" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
              <SortTh label="Status" column="status" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
              <SortTh label="Angelegt" column="createdAt" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
              <th class="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody class="divide-y">
            <tr v-for="user in sorted" :key="user.id" :class="{ 'text-muted-foreground': !user.isActive }">
              <td class="px-4 py-3 font-medium">
                {{ displayName(user) }}
                <span v-if="isSelf(user)" class="ml-1 text-xs text-muted-foreground">(angemeldet)</span>
              </td>
              <td class="px-4 py-3">{{ user.email }}</td>
              <td class="px-4 py-3">{{ userRoleLabel(user.role) }}</td>
              <td class="px-4 py-3"><RouterLink v-if="user.parentId" :to="`/eltern/${user.parentId}`"
                class="text-primary underline">{{ user.parentName || 'Elternteil' }}</RouterLink>
                <span v-else>—</span></td>
              <td class="px-4 py-3">
                <span
                  class="px-2 py-0.5 rounded-full text-xs font-medium"
                  :class="statusClass(user)"
                >
                  {{ statusLabel(user) }}
                </span>
              </td>
              <td class="px-4 py-3">{{ formatDate(user.createdAt) }}</td>
              <td class="px-4 py-3">
                <div class="flex justify-end gap-1">
                  <button
                    @click="openEdit(user)"
                    class="rounded-lg border p-2 hover:bg-accent"
                    aria-label="Bearbeiten" title="Bearbeiten"
                  >
                    <Pencil class="h-4 w-4" />
                  </button>
                  <button
                    v-if="!isSelf(user)"
                    @click="passwordUser = user"
                    class="rounded-lg border p-2 hover:bg-accent"
                    aria-label="Passwort neu setzen" title="Passwort neu setzen"
                  >
                    <KeyRound class="h-4 w-4" />
                  </button>
                  <button v-if="user.invitationPending" type="button"
                    class="rounded-lg border p-2 hover:bg-accent disabled:opacity-50"
                    :disabled="resendingId !== null" @click="resendInvitation(user)"
                    aria-label="Einladung erneut senden" title="Einladung erneut senden">
                    <Send class="h-4 w-4" />
                  </button>
                  <button
                    v-if="canImpersonate(user)"
                    @click="impersonate(user)"
                    :disabled="impersonatingId !== null"
                    class="rounded-lg border p-2 hover:bg-accent disabled:opacity-50"
                    aria-label="Als diesen Benutzer anmelden" title="Als diesen Benutzer anmelden"
                  >
                    <VenetianMask class="h-4 w-4" />
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="!sorted.length">
              <td colspan="7" class="px-4 py-8 text-center text-muted-foreground">Keine Benutzer gefunden.</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <InviteParentsDialog v-if="showInvitations" @close="showInvitations = false"
      @changed="loadUsers" />
    <UserFormDialog
      v-if="showForm"
      :user="formUser"
      :is-self="isSelf(formUser)"
      @close="showForm = false"
      @saved="onSaved"
    />
    <SetUserPasswordDialog
      v-if="passwordUser"
      :user="passwordUser"
      @close="passwordUser = null"
      @saved="onPasswordSaved"
    />
  </div>
</template>
