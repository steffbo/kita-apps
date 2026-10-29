<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import { useRouter } from 'vue-router';
import { api } from '@/api';
import { useAuthStore } from '@/stores/auth';
import type { Member, CreateMemberRequest, MemberCountAsOf } from '@/api/types';
import {
  Plus,
  Loader2,
  User,
  Hash,
  Mail,
  Phone,
  Calendar,
  CalendarClock,
  Check,
  AlertTriangle,
  Trash2,
  UserX,
} from 'lucide-vue-next';
import { formatDate, todayISO } from '@/utils/format';
import SearchInput from '@/components/SearchInput.vue';
import SortTh from '@/components/SortTh.vue';
import AppSheet from '@/components/ui/AppSheet.vue';
import AppPagination from '@/components/ui/AppPagination.vue';
import PageHeader from '@/components/ui/PageHeader.vue';
import EmptyState from '@/components/ui/EmptyState.vue';
import { usePagedList } from '@/composables/usePagedList';

const router = useRouter();
const authStore = useAuthStore();

// Children of the household linked to each member (for the Kinder column)
const childNamesByHousehold = ref<Record<string, string[]>>({});

// Filters
const searchQuery = ref('');
const showInactive = ref(false);

// Pagination

// Sorting
type SortField = 'memberNumber' | 'lastName' | 'firstName' | 'email' | 'membershipStart';
type SortDirection = 'asc' | 'desc';
const sortField = ref<SortField>('lastName');
const sortDirection = ref<SortDirection>('asc');

const {
  items: members,
  total,
  isLoading,
  error,
  currentPage,
  pageSize,
  selectedIds,
  totalPages,
  offset,
  load: loadMembers,
  goToPage,
  goToFirstPageAndLoad,
  handleSearchInput,
} = usePagedList<Member>({
  fetchPage: ({ page, perPage }) => api.getMembers({
    activeOnly: !showInactive.value,
    search: searchQuery.value || undefined,
    sortBy: sortField.value,
    sortDir: sortDirection.value,
    page,
    perPage,
  }),
  onLoaded: (memberList) => {
    loadChildrenNames(memberList);
  },
});

// Bulk selection
const isAllSelected = computed(() => {
  if (members.value.length === 0) return false;
  return members.value.every(m => selectedIds.value.has(m.id));
});
const isSomeSelected = computed(() => {
  return selectedIds.value.size > 0 && !isAllSelected.value;
});

// Dialogs
const showCreateDialog = ref(false);
const showDeactivateDialog = ref(false);
const showDeleteDialog = ref(false);
const isBulkActionLoading = ref(false);
const bulkActionError = ref<string | null>(null);

// Stichtagsreport (member count at reference date)
const showStichtagModal = ref(false);
const stichtagDate = ref(todayISO());
const stichtagData = ref<MemberCountAsOf | null>(null);
const isLoadingStichtag = ref(false);
const stichtagError = ref<string | null>(null);

// Create form
const createForm = ref<CreateMemberRequest>({
  firstName: '',
  lastName: '',
  email: '',
  phone: '',
  membershipStart: todayISO(),
});
const isCreating = ref(false);
const createError = ref<string | null>(null);

function handleInactiveChange() {
  goToFirstPageAndLoad();
}

// Sort changes restart at page 1
watch([sortField, sortDirection], () => {
  goToFirstPageAndLoad();
});

onMounted(loadMembers);

// Helpers

async function loadChildrenNames(memberList: Member[]) {
  const ids = [
    ...new Set(
      memberList
        .map(m => m.householdId)
        .filter((id): id is string => !!id && !childNamesByHousehold.value[id])
    ),
  ];
  if (ids.length === 0) return;
  const results = await Promise.allSettled(ids.map(id => api.getHousehold(id)));
  const next = { ...childNamesByHousehold.value };
  ids.forEach((id, i) => {
    const result = results[i];
    next[id] =
      result.status === 'fulfilled'
        ? (result.value.children ?? []).map(c => `${c.firstName} ${c.lastName}`)
        : [];
  });
  childNamesByHousehold.value = next;
}

function childNamesFor(member: Member): string[] {
  if (!member.householdId) return [];
  return childNamesByHousehold.value[member.householdId] ?? [];
}

// Sorting
function toggleSort(field: SortField) {
  if (sortField.value === field) {
    sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc';
  } else {
    sortField.value = field;
    sortDirection.value = 'asc';
  }
}

// Selection
function toggleSelectAll() {
  if (isAllSelected.value) {
    selectedIds.value = new Set();
  } else {
    selectedIds.value = new Set(members.value.map(m => m.id));
  }
}

function toggleSelect(id: string, event: Event) {
  event.stopPropagation();
  if (selectedIds.value.has(id)) {
    selectedIds.value.delete(id);
  } else {
    selectedIds.value.add(id);
  }
  selectedIds.value = new Set(selectedIds.value); // Trigger reactivity
}

// Navigation
function goToMember(id: string) {
  router.push(`/mitglieder/${id}`);
}

// Create
async function handleCreate() {
  isCreating.value = true;
  createError.value = null;
  try {
    await api.createMember(createForm.value);
    showCreateDialog.value = false;
    createForm.value = {
      firstName: '',
      lastName: '',
      email: '',
      phone: '',
      membershipStart: todayISO(),
    };
    loadMembers();
  } catch (e) {
    createError.value = e instanceof Error ? e.message : 'Fehler beim Erstellen';
  } finally {
    isCreating.value = false;
  }
}

// Bulk actions
async function handleBulkDeactivate() {
  if (selectedIds.value.size === 0) return;

  isBulkActionLoading.value = true;
  bulkActionError.value = null;

  try {
    const promises = [...selectedIds.value].map(id =>
      api.updateMember(id, { isActive: false })
    );
    await Promise.all(promises);
    showDeactivateDialog.value = false;
    selectedIds.value = new Set();
    loadMembers();
  } catch (e) {
    bulkActionError.value = e instanceof Error ? e.message : 'Fehler beim Deaktivieren';
  } finally {
    isBulkActionLoading.value = false;
  }
}

async function handleBulkDelete() {
  if (selectedIds.value.size === 0 || !authStore.isAdmin) return;

  isBulkActionLoading.value = true;
  bulkActionError.value = null;

  try {
    const promises = [...selectedIds.value].map(id => api.deleteMember(id));
    await Promise.all(promises);
    showDeleteDialog.value = false;
    selectedIds.value = new Set();
    loadMembers();
  } catch (e) {
    bulkActionError.value = e instanceof Error ? e.message : 'Fehler beim Löschen';
  } finally {
    isBulkActionLoading.value = false;
  }
}

// Stichtagsreport
async function openStichtagModal() {
  showStichtagModal.value = true;
  stichtagDate.value = todayISO();
  await loadStichtagCount();
}

async function loadStichtagCount() {
  isLoadingStichtag.value = true;
  stichtagError.value = null;
  try {
    stichtagData.value = await api.getMemberCountAsOf(stichtagDate.value);
  } catch (e) {
    stichtagError.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoadingStichtag.value = false;
  }
}

function resetStichtagToToday() {
  stichtagDate.value = todayISO();
  loadStichtagCount();
}

</script>

<template>
  <div>
    <PageHeader title="Vereinsmitglieder">
      <template #actions>
        <button type="button" class="btn btn-secondary" @click="openStichtagModal">
          <CalendarClock />Stichtagsreport
        </button>
        <button type="button" class="btn btn-primary" @click="showCreateDialog = true">
          <Plus />Mitglied hinzufügen
        </button>
      </template>
    </PageHeader>

    <div class="mb-4 flex flex-col gap-3">
      <SearchInput v-model="searchQuery" placeholder="Suchen nach Name, Mitgliedsnummer oder E-Mail..."
        @input="handleSearchInput" />
      <div class="flex gap-2" role="group" aria-label="Filter">
        <span class="chip pointer-events-none border-transparent bg-transparent px-1 text-muted-foreground">
          {{ total }} {{ total === 1 ? 'Mitglied' : 'Mitglieder' }}
        </span>
        <button type="button" class="chip" :aria-pressed="showInactive"
          @click="showInactive = !showInactive; handleInactiveChange()">Inaktive anzeigen</button>
      </div>
    </div>

    <!-- Bulk actions bar -->
    <div
      v-if="selectedIds.size > 0"
      class="mb-4 p-3 bg-primary-soft border border-primary-line rounded-lg flex items-center justify-between"
    >
      <span class="text-sm font-medium text-primary">
        {{ selectedIds.size }} {{ selectedIds.size === 1 ? 'Mitglied' : 'Mitglieder' }} ausgewählt
      </span>
      <div class="flex items-center gap-2">
        <button
          @click="showDeactivateDialog = true"
          class="btn btn-secondary btn-sm"
        >
          <UserX class="h-4 w-4" />
          Deaktivieren
        </button>
        <button
          v-if="authStore.isAdmin"
          @click="showDeleteDialog = true"
          class="btn btn-danger-outline btn-sm"
        >
          <Trash2 class="h-4 w-4" />
          Löschen
        </button>
        <button
          @click="selectedIds = new Set()"
          class="btn btn-ghost btn-sm"
        >
          Auswahl aufheben
        </button>
      </div>
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="note note-danger">
      <AlertTriangle />
      <div>
        <p class="text-destructive">{{ error }}</p>
      <button @click="loadMembers" class="btn btn-secondary btn-sm">
        Erneut versuchen
      </button>
      </div>
    </div>

    <!-- Members table -->
    <div v-else class="card overflow-hidden">
      <ul class="divide-y sm:hidden">
        <li class="flex flex-wrap items-center gap-2 px-4 py-3">
          <input type="checkbox" :checked="isAllSelected" :indeterminate="isSomeSelected"
            @change="toggleSelectAll" aria-label="Alle Mitglieder auswählen" />
          <button type="button" class="chip" :aria-pressed="sortField === 'memberNumber'"
            @click="toggleSort('memberNumber')">Nr.</button>
          <button type="button" class="chip" :aria-pressed="sortField === 'lastName'"
            @click="toggleSort('lastName')">Nachname</button>
          <button type="button" class="chip" :aria-pressed="sortField === 'firstName'"
            @click="toggleSort('firstName')">Vorname</button>
          <button type="button" class="chip" :aria-pressed="sortField === 'email'"
            @click="toggleSort('email')">E-Mail</button>
          <button type="button" class="chip" :aria-pressed="sortField === 'membershipStart'"
            @click="toggleSort('membershipStart')">Mitglied seit</button>
        </li>
        <li v-for="member in members" :key="member.id" class="flex items-center gap-3 p-4" @click="goToMember(member.id)">
          <input type="checkbox" :checked="selectedIds.has(member.id)" @click.stop
            @change="toggleSelect(member.id, $event)" aria-label="Mitglied auswählen" />
          <div class="min-w-0 flex-1">
            <p class="font-medium">{{ member.lastName }}, {{ member.firstName }}</p>
            <p class="text-[13px] text-muted-foreground">{{ member.memberNumber }} · {{ childNamesFor(member).join(', ') || '-' }}</p>
            <p class="text-[13px] text-muted-foreground">{{ member.email || '-' }} · {{ member.phone || '-' }}</p>
          </div>
          <span :class="member.isActive ? 'badge badge-success' : 'badge'">{{ member.isActive ? 'Aktiv' : 'Inaktiv' }}</span>
        </li>
        <li v-if="members.length === 0"><EmptyState :icon="User" text="Keine Mitglieder gefunden" /></li>
      </ul>
      <div class="hidden overflow-x-auto sm:block">
        <table class="table">
          <thead class="bg-muted">
            <tr class="text-left text-sm text-muted-foreground">
              <!-- Checkbox column -->
              <th class="px-4 py-3 w-12">
                <input
                  type="checkbox"
                  :checked="isAllSelected"
                  :indeterminate="isSomeSelected"
                  @change="toggleSelectAll"
                  class="w-4 h-4 text-primary rounded border-border focus:ring-primary"
                />
              </th>
              <!-- Member number -->
              <SortTh label="Nr." column="memberNumber" :icon="Hash" :sort-key="sortField" :sort-dir="sortDirection"
                @sort="toggleSort" />
              <!-- Last Name -->
              <SortTh label="Nachname" column="lastName" :icon="User" :sort-key="sortField" :sort-dir="sortDirection"
                @sort="toggleSort" />
              <!-- First Name -->
              <SortTh label="Vorname" column="firstName" :sort-key="sortField" :sort-dir="sortDirection"
                @sort="toggleSort" />
              <!-- Children -->
              <th class="px-4 py-3 font-medium">
                <div class="flex items-center gap-1">
                  <User class="h-4 w-4" />
                  Kinder
                </div>
              </th>
              <!-- Email -->
              <SortTh label="E-Mail" column="email" :icon="Mail" :sort-key="sortField" :sort-dir="sortDirection"
                @sort="toggleSort" />
              <!-- Phone -->
              <th class="px-4 py-3 font-medium">
                <div class="flex items-center gap-1">
                  <Phone class="h-4 w-4" />
                  Telefon
                </div>
              </th>
              <!-- Membership Start -->
              <SortTh label="Mitglied seit" column="membershipStart" :icon="Calendar" :sort-key="sortField" :sort-dir="sortDirection"
                @sort="toggleSort" />
              <!-- Status -->
              <th class="px-4 py-3 font-medium">Status</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="member in members"
              :key="member.id"
              @click="goToMember(member.id)"
              :class="[
                'border-t hover:bg-accent cursor-pointer transition-colors',
                selectedIds.has(member.id) ? 'bg-primary-soft' : '',
              ]"
            >
              <!-- Checkbox -->
              <td class="px-4 py-3" @click.stop>
                <input
                  type="checkbox"
                  :checked="selectedIds.has(member.id)"
                  @change="toggleSelect(member.id, $event)"
                  class="w-4 h-4 text-primary rounded border-border focus:ring-primary"
                />
              </td>
              <!-- Member number -->
              <td class="px-4 py-3 font-mono text-sm">{{ member.memberNumber }}</td>
              <!-- Last Name -->
              <td class="px-4 py-3 font-medium">{{ member.lastName }}</td>
              <!-- First Name -->
              <td class="px-4 py-3">{{ member.firstName }}</td>
              <!-- Children -->
              <td class="px-4 py-3 text-muted-foreground">
                <span v-if="childNamesFor(member).length > 0" class="truncate">
                  {{ childNamesFor(member).join(', ') }}
                </span>
                <span v-else class="text-muted-foreground">-</span>
              </td>
              <!-- Email -->
              <td class="px-4 py-3 text-muted-foreground">
                <span v-if="member.email" class="truncate">{{ member.email }}</span>
                <span v-else class="text-muted-foreground">-</span>
              </td>
              <!-- Phone -->
              <td class="px-4 py-3 text-muted-foreground">
                <span v-if="member.phone">{{ member.phone }}</span>
                <span v-else class="text-muted-foreground">-</span>
              </td>
              <!-- Membership Start -->
              <td class="px-4 py-3 text-muted-foreground">{{ formatDate(member.membershipStart) }}</td>
              <!-- Status -->
              <td class="px-4 py-3">
                <span
                  :class="[
                    member.isActive ? 'badge badge-success' : 'badge',
                  ]"
                >
                  {{ member.isActive ? 'Aktiv' : 'Inaktiv' }}
                </span>
              </td>
            </tr>
            <tr v-if="members.length === 0">
              <td colspan="9" class="px-4 py-8 text-center text-muted-foreground">
                Keine Mitglieder gefunden
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <AppPagination v-model:page-size="pageSize" :current-page="currentPage" :total-pages="totalPages"
        :total="total" :offset="offset" @go="goToPage" />
    </div>

    <!-- Create Dialog -->
    <AppSheet v-if="showCreateDialog" title="Mitglied hinzufügen" @close="showCreateDialog = false">

        <form id="create-member-form" @submit.prevent="handleCreate" class="space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="firstName" class="label">Vorname *</label>
              <input
                id="firstName"
                v-model="createForm.firstName"
                required
                type="text"
                class="input"
              />
            </div>
            <div>
              <label for="lastName" class="label">Nachname *</label>
              <input
                id="lastName"
                v-model="createForm.lastName"
                required
                type="text"
                class="input"
              />
            </div>
          </div>

          <div>
            <label for="email" class="label">E-Mail</label>
            <input
              id="email"
              v-model="createForm.email"
              type="email"
              class="input"
            />
          </div>

          <div>
            <label for="phone" class="label">Telefon</label>
            <input
              id="phone"
              v-model="createForm.phone"
              type="tel"
              class="input"
            />
          </div>

          <div>
            <label for="membershipStart" class="label">Mitglied seit *</label>
            <input
              id="membershipStart"
              v-model="createForm.membershipStart"
              required
              type="date"
              class="input"
            />
          </div>

          <div v-if="createError" class="note note-danger">
      <AlertTriangle />
      <p class="text-sm text-destructive">{{ createError }}</p>
          </div>


        </form>

    <template #footer><div class="flex justify-end gap-3 pt-4">
            <button
              type="button"
              @click="showCreateDialog = false"
              class="btn btn-secondary"
            >
              Abbrechen
            </button>
            <button form="create-member-form" type="submit"
              :disabled="isCreating"
              class="btn btn-primary"
            >
              <Loader2 v-if="isCreating" class="h-4 w-4 animate-spin" />
              <Check v-else class="h-4 w-4" />
              Speichern
            </button>
          </div></template></AppSheet>

    <!-- Deactivate Confirmation Dialog -->
    <AppSheet v-if="showDeactivateDialog" title="Mitglieder deaktivieren" @close="showDeactivateDialog = false">

        <p class="text-muted-foreground mb-6">
          Möchten Sie <strong>{{ selectedIds.size }}</strong> {{ selectedIds.size === 1 ? 'Mitglied' : 'Mitglieder' }} wirklich deaktivieren?
          Deaktivierte Mitglieder werden nicht mehr in den Standardlisten angezeigt.
        </p>

        <div v-if="bulkActionError" class="mb-4 note note-danger">
      <AlertTriangle />
      <p class="text-sm text-destructive">{{ bulkActionError }}</p>
        </div>



    <template #footer><div class="flex justify-end gap-3">
          <button
            @click="showDeactivateDialog = false"
            class="btn btn-secondary"
          >
            Abbrechen
          </button>
          <button
            @click="handleBulkDeactivate"
            :disabled="isBulkActionLoading"
            class="btn btn-secondary"
          >
            <Loader2 v-if="isBulkActionLoading" class="h-4 w-4 animate-spin" />
            <UserX v-else class="h-4 w-4" />
            Deaktivieren
          </button>
        </div></template></AppSheet>

    <!-- Delete Confirmation Dialog (Admin only) -->
    <AppSheet v-if="showDeleteDialog && authStore.isAdmin" title="Mitglieder löschen" @close="showDeleteDialog = false">

        <div class="note note-danger mb-4">
          <div class="flex items-start gap-2">
            <AlertTriangle class="h-5 w-5 text-destructive flex-shrink-0 mt-0.5" />
            <div>
              <p class="font-semibold text-destructive">Achtung: Permanente Löschung!</p>
              <p class="text-sm text-destructive mt-1">
                Diese Aktion kann nicht rückgängig gemacht werden. Alle zugehörigen Daten
                werden ebenfalls gelöscht.
              </p>
            </div>
          </div>
        </div>

        <p class="text-muted-foreground mb-6">
          Möchten Sie <strong>{{ selectedIds.size }}</strong> {{ selectedIds.size === 1 ? 'Mitglied' : 'Mitglieder' }} wirklich
          <strong class="text-destructive">unwiderruflich löschen</strong>?
        </p>

        <div v-if="bulkActionError" class="mb-4 note note-danger">
      <AlertTriangle />
      <p class="text-sm text-destructive">{{ bulkActionError }}</p>
        </div>



    <template #footer><div class="flex justify-end gap-3">
          <button
            @click="showDeleteDialog = false"
            class="btn btn-secondary"
          >
            Abbrechen
          </button>
          <button
            @click="handleBulkDelete"
            :disabled="isBulkActionLoading"
            class="btn btn-danger"
          >
            <Loader2 v-if="isBulkActionLoading" class="h-4 w-4 animate-spin" />
            <Trash2 v-else class="h-4 w-4" />
            Endgültig löschen
          </button>
        </div></template></AppSheet>

    <AppSheet v-if="showStichtagModal" title="Mitglieder-Stichtagsreport" size="lg"
      @close="showStichtagModal = false">
      <div class="space-y-5">
        <p class="note note-info">
          Anzahl der Vereinsmitglieder zum Stichtag, inklusive bereits inaktiver Mitglieder
        </p>
        <div>
          <label for="member-count-date" class="label">Stichtag</label>
          <div class="flex gap-2">
            <input id="member-count-date" v-model="stichtagDate" type="date" class="input"
              @change="loadStichtagCount" />
            <button type="button" class="btn btn-secondary" @click="resetStichtagToToday">Heute</button>
          </div>
        </div>
        <div v-if="isLoadingStichtag" class="flex items-center justify-center py-8">
          <Loader2 class="h-6 w-6 animate-spin text-primary" />
        </div>
        <div v-else-if="stichtagError" class="note note-danger">
      <AlertTriangle />
      <div>
        <p>{{ stichtagError }}</p>
          <button class="btn btn-secondary btn-sm" @click="loadStichtagCount">Erneut versuchen</button>
      </div>
        </div>
        <div v-else-if="stichtagData" class="grid grid-cols-1 gap-3 sm:grid-cols-3">
          <div class="card bg-success-soft p-4">
            <p class="text-sm font-medium text-success">Mitglieder am Stichtag</p>
            <p class="mt-1 text-2xl font-semibold text-success">{{ stichtagData.total }}</p>
            <p class="mt-1 text-xs text-success">am {{ formatDate(stichtagData.asOf) }}</p>
          </div>
          <div class="card bg-muted p-4">
            <p class="text-sm font-medium text-muted-foreground">Davon aktuell aktiv</p>
            <p class="mt-1 text-2xl font-semibold">{{ stichtagData.active }}</p>
          </div>
          <div class="card bg-warning-soft p-4">
            <p class="text-sm font-medium text-warning">Davon inaktiv</p>
            <p class="mt-1 text-2xl font-semibold text-warning">{{ stichtagData.inactive }}</p>
          </div>
        </div>
      </div>
      <template #footer>

      </template>
    </AppSheet>
  </div>
</template>
