<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { api } from '@/api';
import { useAuthStore } from '@/stores/auth';
import type { Child, CreateChildRequest } from '@/api/types';
import {
  Plus,
  Loader2,
  Check,
  Upload,
  AlertTriangle,
  Coins,
  Trash2,
  UserX,
  CalendarX,
  Users,
} from 'lucide-vue-next';
import { formatDate, todayISO } from '@/utils/format';
import { calculateAge, isUnderThree } from '@/utils/child';
import SearchInput from '@/components/SearchInput.vue';
import SortTh from '@/components/SortTh.vue';
import AppSheet from '@/components/ui/AppSheet.vue';
import AppPagination from '@/components/ui/AppPagination.vue';
import PageHeader from '@/components/ui/PageHeader.vue';
import EmptyState from '@/components/ui/EmptyState.vue';
import { usePagedList } from '@/composables/usePagedList';

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();

// Filters
const searchQuery = ref('');
const showInactive = ref(false);
const showOnlyU3 = ref(false);
const showOnlyWarnings = ref(false);
const showOnlyOpenFees = ref(false);

// Pagination
const pageSizeOptions = [10, 25, 50, 100];

// Sorting
type SortField = 'memberNumber' | 'firstName' | 'lastName' | 'birthDate' | 'age' | 'entryDate';
type SortDirection = 'asc' | 'desc';
const sortField = ref<SortField>('lastName');
const sortDirection = ref<SortDirection>('asc');

const {
  items: children,
  total,
  isLoading,
  error,
  currentPage,
  pageSize,
  selectedIds,
  totalPages,
  offset,
  load: loadChildren,
  goToFirstPageAndLoad,
  handleSearchInput,
} = usePagedList<Child>({
  fetchPage: ({ page, perPage }) => api.getChildren({
    activeOnly: !showInactive.value,
    u3Only: showOnlyU3.value,
    hasWarnings: showOnlyWarnings.value,
    hasOpenFees: showOnlyOpenFees.value,
    search: searchQuery.value || undefined,
    sortBy: sortField.value,
    sortDir: sortDirection.value,
    page,
    perPage,
  }),
});

// Bulk selection
const isAllSelected = computed(() => {
  if (children.value.length === 0) return false;
  return children.value.every(c => selectedIds.value.has(c.id));
});
const isSomeSelected = computed(() => {
  return selectedIds.value.size > 0 && !isAllSelected.value;
});

// Dialogs
const showCreateDialog = ref(false);
const showDeactivateDialog = ref(false);
const showDeleteDialog = ref(false);
const showExitDateDialog = ref(false);
const bulkExitDate = ref('');
const isBulkActionLoading = ref(false);
const bulkActionError = ref<string | null>(null);

// Create form
const createForm = ref<CreateChildRequest>({
  memberNumber: '',
  firstName: '',
  lastName: '',
  birthDate: '',
  entryDate: '',
});
const isCreating = ref(false);
const createError = ref<string | null>(null);

// Filter chips; each change restarts at page 1.
const filters = [
  { label: 'Inaktive', flag: showInactive },
  { label: 'U3', flag: showOnlyU3 },
  { label: 'Mit Hinweisen', flag: showOnlyWarnings },
  { label: 'Offene Beiträge', flag: showOnlyOpenFees },
];

function toggleFilter(flag: { value: boolean }) {
  flag.value = !flag.value;
  goToFirstPageAndLoad();
}

// Sort changes restart at page 1
watch([sortField, sortDirection], () => {
  currentPage.value = 1;
  loadChildren();
});

onMounted(() => {
  // Check for query params to auto-enable filters
  if (route.query.openFees === 'true') {
    showOnlyOpenFees.value = true;
  }
  if (route.query.warnings === 'true') {
    showOnlyWarnings.value = true;
  }
  loadChildren();
});

// Helpers

// Warning checks
function getChildWarnings(child: Child): string[] {
  const warnings: string[] = [];
  
  // U3 without household income
  if (isUnderThree(child.birthDate)) {
    const household = child.household;
    const hasIncome = household && (
      household.annualHouseholdIncome !== undefined && household.annualHouseholdIncome !== null ||
      household.incomeStatus === 'MAX_ACCEPTED' ||
      household.incomeStatus === 'NOT_REQUIRED' ||
      household.incomeStatus === 'FOSTER_FAMILY' ||
      household.incomeStatus === 'HISTORIC'
    );
    if (!hasIncome) {
      warnings.push('U3 ohne Haushaltseinkommen');
    }
  }
  
  // No Rechtsanspruch or Betreuungszeit
  if (!child.legalHours && !child.careHours) {
    warnings.push('Keine Betreuungsstunden');
  } else if (!child.legalHours) {
    warnings.push('Kein Rechtsanspruch');
  } else if (!child.careHours) {
    warnings.push('Keine Betreuungszeit');
  }
  
  // No parents linked
  if (!child.parents || child.parents.length === 0) {
    warnings.push('Keine Eltern verknüpft');
  }
  
  return warnings;
}

function hasOpenFees(child: Child): boolean {
  return (child.openFeesCount ?? 0) > 0;
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
    selectedIds.value = new Set(children.value.map(c => c.id));
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
function goToChild(id: string) {
  router.push(`/kinder/${id}`);
}

function goToPage(page: number) {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page;
  }
}

// Create
async function openCreateDialog() {
  createError.value = null;
  createForm.value = {
    memberNumber: '',
    firstName: '',
    lastName: '',
    birthDate: '',
    entryDate: '',
  };
  showCreateDialog.value = true;
  try {
    const result = await api.getNextChildMemberNumber();
    createForm.value.memberNumber = result.memberNumber;
  } catch (e) {
    createError.value = e instanceof Error ? e.message : 'Mitgliedsnummer konnte nicht geladen werden';
  }
}

async function handleCreate() {
  isCreating.value = true;
  createError.value = null;
  try {
    await api.createChild(createForm.value);
    showCreateDialog.value = false;
    createForm.value = {
      memberNumber: '',
      firstName: '',
      lastName: '',
      birthDate: '',
      entryDate: '',
    };
    loadChildren();
  } catch (e) {
    createError.value = e instanceof Error ? e.message : 'Fehler beim Erstellen';
  } finally {
    isCreating.value = false;
  }
}

// Bulk actions
// Vorschlag: Ende des laufenden Kita-Jahres (31.07.), typischer Austritt bei Schulbeginn.
function openExitDateDialog() {
  const [year, month] = todayISO().split('-').map(Number);
  bulkExitDate.value = `${month >= 8 ? year + 1 : year}-07-31`;
  bulkActionError.value = null;
  showExitDateDialog.value = true;
}

// Kinder aus der Auswahl, deren Eintritt nach dem gewählten Austrittsdatum liegt.
const exitBeforeEntry = computed(() => children.value.filter(c =>
  selectedIds.value.has(c.id) && bulkExitDate.value && c.entryDate.slice(0, 10) > bulkExitDate.value));

async function handleBulkExitDate() {
  if (selectedIds.value.size === 0 || !bulkExitDate.value || exitBeforeEntry.value.length) return;

  isBulkActionLoading.value = true;
  bulkActionError.value = null;

  try {
    await Promise.all([...selectedIds.value].map(id => api.updateChild(id, { exitDate: bulkExitDate.value })));
    showExitDateDialog.value = false;
    selectedIds.value = new Set();
    loadChildren();
  } catch (e) {
    bulkActionError.value = e instanceof Error ? e.message : 'Fehler beim Setzen des Austrittsdatums';
  } finally {
    isBulkActionLoading.value = false;
  }
}

async function handleBulkDeactivate() {
  if (selectedIds.value.size === 0) return;
  
  isBulkActionLoading.value = true;
  bulkActionError.value = null;
  
  try {
    const promises = [...selectedIds.value].map(id => 
      api.updateChild(id, { isActive: false })
    );
    await Promise.all(promises);
    showDeactivateDialog.value = false;
    selectedIds.value = new Set();
    loadChildren();
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
    const promises = [...selectedIds.value].map(id => api.deleteChild(id));
    await Promise.all(promises);
    showDeleteDialog.value = false;
    selectedIds.value = new Set();
    loadChildren();
  } catch (e) {
    bulkActionError.value = e instanceof Error ? e.message : 'Fehler beim Löschen';
  } finally {
    isBulkActionLoading.value = false;
  }
}

// Pagination display helpers
</script>

<template>
  <div>
    <PageHeader title="Kinder">
      <template #actions>
        <button type="button" class="btn btn-secondary" @click="router.push('/kinder/import')">
          <Upload />Importieren
        </button>
        <button type="button" class="btn btn-primary" @click="openCreateDialog">
          <Plus />Kind hinzufügen
        </button>
      </template>
    </PageHeader>

    <div class="mb-4 flex flex-col gap-3">
      <SearchInput v-model="searchQuery" placeholder="Name oder Mitgliedsnummer" @input="handleSearchInput" />
      <div class="-mx-4 flex gap-2 overflow-x-auto px-4 sm:mx-0 sm:flex-wrap sm:px-0" role="group" aria-label="Filter">
        <span class="chip pointer-events-none border-transparent bg-transparent px-1 text-muted-foreground"
          aria-live="polite">{{ total }} {{ total === 1 ? 'Kind' : 'Kinder' }}</span>
        <button v-for="filter in filters" :key="filter.label" type="button" class="chip"
          :aria-pressed="filter.flag.value" @click="toggleFilter(filter.flag)">{{ filter.label }}</button>
      </div>
    </div>

    <div v-if="selectedIds.size > 0"
      class="mb-4 flex flex-wrap items-center gap-2 rounded-xl border border-transparent bg-primary-soft px-3 py-2">
      <span class="mr-auto text-sm font-medium text-primary-text">
        {{ selectedIds.size }} {{ selectedIds.size === 1 ? 'Kind' : 'Kinder' }} ausgewählt
      </span>
      <button type="button" class="btn btn-secondary btn-sm" @click="openExitDateDialog">
        <CalendarX />Austrittsdatum setzen
      </button>
      <button type="button" class="btn btn-secondary btn-sm" @click="showDeactivateDialog = true">
        <UserX />Deaktivieren
      </button>
      <button v-if="authStore.isAdmin" type="button" class="btn btn-danger-outline btn-sm" @click="showDeleteDialog = true">
        <Trash2 />Löschen
      </button>
      <button type="button" class="btn btn-ghost btn-sm" @click="selectedIds = new Set()">Auswahl aufheben</button>
    </div>

    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
    </div>

    <div v-else-if="error" class="note note-danger" role="alert">
      <AlertTriangle />
      <div class="flex-1">{{ error }}</div>
      <button type="button" class="font-medium underline" @click="loadChildren">Erneut versuchen</button>
    </div>

    <div v-else class="card overflow-hidden">
      <EmptyState v-if="children.length === 0" :icon="Users" text="Keine Kinder gefunden" />

      <!-- Phone: one row per child -->
      <ul v-else class="divide-y sm:hidden">
        <li v-for="child in children" :key="child.id" class="flex items-center gap-3 px-4 py-3"
          :class="selectedIds.has(child.id) && 'bg-primary-soft'">
          <input type="checkbox" :checked="selectedIds.has(child.id)" :aria-label="`${child.firstName} ${child.lastName} auswählen`"
            class="h-5 w-5 shrink-0 rounded" @change="toggleSelect(child.id, $event)" />
          <button type="button" class="flex min-w-0 flex-1 items-center gap-3 text-left" @click="goToChild(child.id)">
            <span class="min-w-0 flex-1">
              <span class="block truncate font-medium">{{ child.lastName }}, {{ child.firstName }}</span>
              <span class="block truncate text-[13px] tabular-nums text-muted-foreground">
                {{ child.memberNumber }} · {{ calculateAge(child.birthDate) }} J.<template
                  v-if="child.careHours"> · {{ child.careHours }} h</template>
              </span>
            </span>
            <span class="flex shrink-0 items-center gap-1.5">
              <span v-if="getChildWarnings(child).length" class="badge badge-warning">
                <AlertTriangle class="h-3 w-3" />{{ getChildWarnings(child).length }}
              </span>
              <span v-if="hasOpenFees(child)" class="badge">
                <Coins class="h-3 w-3" />{{ child.openFeesCount ?? 0 }}
              </span>
              <span v-if="!child.isActive" class="badge">Inaktiv</span>
            </span>
          </button>
        </li>
      </ul>

      <div v-if="children.length" class="hidden overflow-x-auto sm:block">
        <table class="table">
          <thead>
            <tr>
              <th class="w-12">
                <input type="checkbox" aria-label="Alle auswählen" :checked="isAllSelected" :indeterminate="isSomeSelected"
                  class="h-4 w-4 rounded" @change="toggleSelectAll" />
              </th>
              <SortTh label="Nr." column="memberNumber" :sort-key="sortField" :sort-dir="sortDirection" @sort="toggleSort" />
              <SortTh label="Nachname" column="lastName" :sort-key="sortField" :sort-dir="sortDirection" @sort="toggleSort" />
              <SortTh label="Vorname" column="firstName" :sort-key="sortField" :sort-dir="sortDirection" @sort="toggleSort" />
              <SortTh label="Geburtsdatum" column="birthDate" :sort-key="sortField" :sort-dir="sortDirection"
                @sort="toggleSort" />
              <th>Alter</th>
              <th>Stunden</th>
              <th>Hinweise</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="child in children" :key="child.id" class="cursor-pointer transition-colors hover:bg-accent"
              :class="selectedIds.has(child.id) && 'bg-primary-soft hover:bg-primary-soft'" @click="goToChild(child.id)">
              <td @click.stop>
                <input type="checkbox" :checked="selectedIds.has(child.id)" class="h-4 w-4 rounded"
                  @change="toggleSelect(child.id, $event)" />
              </td>
              <td class="tabular-nums text-muted-foreground">{{ child.memberNumber }}</td>
              <td class="font-medium">{{ child.lastName }}</td>
              <td>{{ child.firstName }}</td>
              <td class="tabular-nums text-muted-foreground">{{ formatDate(child.birthDate) }}</td>
              <td class="tabular-nums">{{ calculateAge(child.birthDate) }} J.</td>
              <td class="tabular-nums text-muted-foreground">{{ child.careHours ? `${child.careHours} h` : '–' }}</td>
              <td>
                <div class="flex items-center gap-1.5">
                  <div v-if="getChildWarnings(child).length > 0" class="group relative">
                    <span class="badge badge-warning"><AlertTriangle class="h-3 w-3" />{{ getChildWarnings(child).length }}</span>
                    <div class="absolute left-0 top-7 z-10 hidden w-60 rounded-lg border bg-popover p-3 shadow-lg group-hover:block">
                      <ul class="space-y-1 text-xs">
                        <li v-for="(warning, idx) in getChildWarnings(child)" :key="idx" class="flex items-start gap-2">
                          <AlertTriangle class="mt-0.5 h-3 w-3 shrink-0 text-warning" />
                          <span>{{ warning }}</span>
                        </li>
                      </ul>
                    </div>
                  </div>
                  <span v-if="hasOpenFees(child)" class="badge"
                    :title="`${child.openFeesCount ?? 0} offene Beiträge`">
                    <Coins class="h-3 w-3" />{{ child.openFeesCount ?? 0 }}
                  </span>
                  <Check v-if="getChildWarnings(child).length === 0 && !hasOpenFees(child)"
                    class="h-4 w-4 text-success" aria-label="Keine Hinweise" />
                </div>
              </td>
              <td>
                <span class="badge" :class="child.isActive && 'badge-success'">{{ child.isActive ? 'Aktiv' : 'Inaktiv' }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <AppPagination v-if="children.length" v-model:page-size="pageSize" :current-page="currentPage"
        :total-pages="totalPages" :total="total" :offset="offset" :page-size-options="pageSizeOptions" @go="goToPage" />
    </div>

    <AppSheet v-if="showCreateDialog" title="Kind hinzufügen" @close="showCreateDialog = false">
      <form id="create-child" class="space-y-4" @submit.prevent="handleCreate">
        <div>
          <label for="memberNumber" class="label">Mitgliedsnummer *</label>
          <input id="memberNumber" v-model="createForm.memberNumber" required type="text" placeholder="z. B. 11072"
            class="input" />
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="firstName" class="label">Vorname *</label>
            <input id="firstName" v-model="createForm.firstName" required type="text" class="input" />
          </div>
          <div>
            <label for="lastName" class="label">Nachname *</label>
            <input id="lastName" v-model="createForm.lastName" required type="text" class="input" />
          </div>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="birthDate" class="label">Geburtsdatum *</label>
            <input id="birthDate" v-model="createForm.birthDate" required type="date" class="input" />
          </div>
          <div>
            <label for="entryDate" class="label">Eintrittsdatum *</label>
            <input id="entryDate" v-model="createForm.entryDate" required type="date" class="input" />
          </div>
        </div>
        <div v-if="createError" class="note note-danger" role="alert"><AlertTriangle />{{ createError }}</div>
      </form>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="showCreateDialog = false">Abbrechen</button>
        <button type="submit" form="create-child" class="btn btn-primary" :disabled="isCreating">
          <Loader2 v-if="isCreating" class="animate-spin" />Speichern
        </button>
      </template>
    </AppSheet>

    <AppSheet v-if="showExitDateDialog" title="Austrittsdatum setzen" @close="showExitDateDialog = false">
      <form id="bulk-exit-date" class="space-y-4" @submit.prevent="handleBulkExitDate">
        <p class="text-sm text-muted-foreground">
          Gilt für {{ selectedIds.size }} {{ selectedIds.size === 1 ? 'Kind' : 'Kinder' }} und überschreibt ein
          eingetragenes Datum.
        </p>
        <label class="label">Austrittsdatum
          <input v-model="bulkExitDate" type="date" required class="input mt-1.5" />
        </label>
        <div v-if="exitBeforeEntry.length" role="alert" class="note note-warning">
          <AlertTriangle />Liegt vor dem Eintritt von {{ exitBeforeEntry.map(c => `${c.firstName} ${c.lastName}`).join(', ') }}.
        </div>
        <div v-if="bulkActionError" class="note note-danger"><AlertTriangle />{{ bulkActionError }}</div>
      </form>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="showExitDateDialog = false">Abbrechen</button>
        <button type="submit" form="bulk-exit-date" class="btn btn-primary"
          :disabled="isBulkActionLoading || !bulkExitDate || exitBeforeEntry.length > 0">
          <Loader2 v-if="isBulkActionLoading" class="animate-spin" />Austrittsdatum setzen
        </button>
      </template>
    </AppSheet>

    <AppSheet v-if="showDeactivateDialog" title="Kinder deaktivieren" size="sm" @close="showDeactivateDialog = false">
      <p class="text-sm text-muted-foreground">
        {{ selectedIds.size }} {{ selectedIds.size === 1 ? 'Kind' : 'Kinder' }} deaktivieren? Deaktivierte Kinder
        erscheinen nicht mehr in den Standardlisten.
      </p>
      <div v-if="bulkActionError" class="note note-danger mt-4"><AlertTriangle />{{ bulkActionError }}</div>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="showDeactivateDialog = false">Abbrechen</button>
        <button type="button" class="btn btn-primary" :disabled="isBulkActionLoading" @click="handleBulkDeactivate">
          <Loader2 v-if="isBulkActionLoading" class="animate-spin" />Deaktivieren
        </button>
      </template>
    </AppSheet>

    <AppSheet v-if="showDeleteDialog && authStore.isAdmin" title="Kinder löschen" size="sm"
      @close="showDeleteDialog = false">
      <div class="note note-danger">
        <AlertTriangle />
        <div>
          {{ selectedIds.size }} {{ selectedIds.size === 1 ? 'Kind' : 'Kinder' }} unwiderruflich löschen? Alle
          zugehörigen Daten (Elternverknüpfungen, Gebühreneinträge usw.) werden mitgelöscht.
        </div>
      </div>
      <div v-if="bulkActionError" class="note note-danger mt-4"><AlertTriangle />{{ bulkActionError }}</div>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="showDeleteDialog = false">Abbrechen</button>
        <button type="button" class="btn btn-danger" :disabled="isBulkActionLoading" @click="handleBulkDelete">
          <Loader2 v-if="isBulkActionLoading" class="animate-spin" />Endgültig löschen
        </button>
      </template>
    </AppSheet>
  </div>
</template>
