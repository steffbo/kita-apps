<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import { useRouter } from 'vue-router';
import { api } from '@/api';
import { useAuthStore } from '@/stores/auth';
import type { Parent, CreateParentRequest } from '@/api/types';
import {
  Plus,
  Loader2,
  User,
  Mail,
  Phone,
  AlertTriangle,
  Trash2,
} from 'lucide-vue-next';
import SearchInput from '@/components/SearchInput.vue';
import SortTh from '@/components/SortTh.vue';
import AppSheet from '@/components/ui/AppSheet.vue';
import AppPagination from '@/components/ui/AppPagination.vue';
import PageHeader from '@/components/ui/PageHeader.vue';
import EmptyState from '@/components/ui/EmptyState.vue';
import { usePagedList } from '@/composables/usePagedList';

const router = useRouter();
const authStore = useAuthStore();

// Filters
const searchQuery = ref('');

// Pagination

// Sorting
type SortField = 'lastName' | 'firstName' | 'email';
type SortDirection = 'asc' | 'desc';
const sortField = ref<SortField>('lastName');
const sortDirection = ref<SortDirection>('asc');

const {
  items: parents,
  total,
  isLoading,
  error,
  currentPage,
  pageSize,
  selectedIds,
  totalPages,
  offset,
  load: loadParents,
  goToPage,
  goToFirstPageAndLoad,
  handleSearchInput,
} = usePagedList<Parent>({
  fetchPage: ({ page, perPage }) => api.getParents({
    search: searchQuery.value || undefined,
    sortBy: sortField.value,
    sortDir: sortDirection.value,
    page,
    perPage,
  }),
});

// Bulk selection
const isAllSelected = computed(() => {
  if (parents.value.length === 0) return false;
  return parents.value.every(p => selectedIds.value.has(p.id));
});
const isSomeSelected = computed(() => {
  return selectedIds.value.size > 0 && !isAllSelected.value;
});

// Dialogs
const showDeleteDialog = ref(false);
const showCreateDialog = ref(false);
const isBulkActionLoading = ref(false);
const bulkActionError = ref<string | null>(null);
const isCreatingParent = ref(false);
const createError = ref<string | null>(null);

const parentForm = ref<CreateParentRequest>({
  firstName: '',
  lastName: '',
  birthDate: '',
  email: '',
  phone: '',
  street: '',
  streetNo: '',
  postalCode: '',
  city: '',
});

// Reload when sort changes
watch([sortField, sortDirection], () => {
  goToFirstPageAndLoad();
});

onMounted(loadParents);

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
    selectedIds.value = new Set(parents.value.map(p => p.id));
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
function goToParent(id: string) {
  router.push(`/eltern/${id}`);
}

function openCreateParentDialog() {
  parentForm.value = {
    firstName: '',
    lastName: '',
    birthDate: '',
    email: '',
    phone: '',
    street: '',
    streetNo: '',
    postalCode: '',
    city: '',
  };
  createError.value = null;
  showCreateDialog.value = true;
}

async function handleCreateParent() {
  if (!parentForm.value.firstName?.trim() || !parentForm.value.lastName?.trim()) {
    createError.value = 'Vorname und Nachname sind erforderlich.';
    return;
  }
  isCreatingParent.value = true;
  createError.value = null;
  try {
    await api.createParent({
      ...parentForm.value,
      firstName: parentForm.value.firstName.trim(),
      lastName: parentForm.value.lastName.trim(),
    });
    showCreateDialog.value = false;
    loadParents();
  } catch (e) {
    createError.value = e instanceof Error ? e.message : 'Fehler beim Erstellen';
  } finally {
    isCreatingParent.value = false;
  }
}

// Bulk delete
async function handleBulkDelete() {
  if (selectedIds.value.size === 0 || !authStore.isAdmin) return;

  isBulkActionLoading.value = true;
  bulkActionError.value = null;

  try {
    const promises = [...selectedIds.value].map(id => api.deleteParent(id));
    await Promise.all(promises);
    showDeleteDialog.value = false;
    selectedIds.value = new Set();
    loadParents();
  } catch (e) {
    bulkActionError.value = e instanceof Error ? e.message : 'Fehler beim Löschen';
  } finally {
    isBulkActionLoading.value = false;
  }
}

</script>

<template>
  <div>
    <PageHeader title="Eltern">
      <template #actions>
        <button data-write type="button" class="btn btn-primary" @click="openCreateParentDialog">
          <Plus />Elternteil hinzufügen
        </button>
      </template>
    </PageHeader>

    <!-- Search -->
    <div class="mb-4 flex flex-col gap-3">
      <SearchInput v-model="searchQuery" placeholder="Suchen nach Name oder E-Mail..." @input="handleSearchInput" />
      <div class="flex gap-2" role="group" aria-label="Filter">
        <span class="chip pointer-events-none border-transparent bg-transparent px-1 text-muted-foreground">
          {{ total }} {{ total === 1 ? 'Elternteil' : 'Eltern' }}
        </span>
      </div>
    </div>

    <!-- Bulk actions bar -->
    <div
      v-if="selectedIds.size > 0"
      class="mb-4 p-3 bg-primary-soft border border-primary-line rounded-lg flex items-center justify-between"
    >
      <span class="text-sm font-medium text-primary">
        {{ selectedIds.size }} {{ selectedIds.size === 1 ? 'Elternteil' : 'Eltern' }} ausgewählt
      </span>
      <div class="flex items-center gap-2">
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
      <button @click="loadParents" class="btn btn-secondary btn-sm">
        Erneut versuchen
      </button>
      </div>
    </div>

    <!-- Parents table -->
    <div v-else class="card overflow-hidden">
      <ul class="divide-y sm:hidden">
        <li class="flex flex-wrap items-center gap-2 px-4 py-3">
          <input data-write type="checkbox" :checked="isAllSelected" :indeterminate="isSomeSelected"
            @change="toggleSelectAll" aria-label="Alle Eltern auswählen" />
          <button type="button" class="chip" :aria-pressed="sortField === 'lastName'"
            @click="toggleSort('lastName')">Nachname</button>
          <button type="button" class="chip" :aria-pressed="sortField === 'firstName'"
            @click="toggleSort('firstName')">Vorname</button>
          <button type="button" class="chip" :aria-pressed="sortField === 'email'"
            @click="toggleSort('email')">E-Mail</button>
        </li>
        <li v-for="parent in parents" :key="parent.id" class="flex items-center gap-3 p-4" @click="goToParent(parent.id)">
          <input data-write type="checkbox" :checked="selectedIds.has(parent.id)" @click.stop
            @change="toggleSelect(parent.id, $event)" aria-label="Elternteil auswählen" />
          <div class="min-w-0 flex-1">
            <p class="font-medium">{{ parent.lastName }}, {{ parent.firstName }}</p>
            <p class="text-[13px] text-muted-foreground">{{ parent.email || '-' }} · {{ parent.phone || '-' }}</p>
            <p v-if="parent.children?.length" class="text-[13px] text-muted-foreground">
              {{ parent.children.map(child => child.firstName).join(', ') }}
            </p>
          </div>
        </li>
        <li v-if="parents.length === 0"><EmptyState :icon="User" text="Keine Eltern gefunden" /></li>
      </ul>
      <div class="hidden overflow-x-auto sm:block">
        <table class="table">
          <thead class="bg-muted">
            <tr class="text-left text-sm text-muted-foreground">
              <!-- Checkbox column -->
              <th class="px-4 py-3 w-12">
                <input data-write
                  type="checkbox"
                  :checked="isAllSelected"
                  :indeterminate="isSomeSelected"
                  @change="toggleSelectAll"
                  class="w-4 h-4 text-primary rounded border-border focus:ring-primary"
                />
              </th>
              <!-- Last Name -->
              <SortTh label="Nachname" column="lastName" :icon="User" :sort-key="sortField" :sort-dir="sortDirection"
                @sort="toggleSort" />
              <!-- First Name -->
              <SortTh label="Vorname" column="firstName" :sort-key="sortField" :sort-dir="sortDirection"
                @sort="toggleSort" />
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
              <!-- Children -->
              <th class="px-4 py-3 font-medium">Kinder</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="parent in parents"
              :key="parent.id"
              @click="goToParent(parent.id)"
              :class="[
                'border-t hover:bg-accent cursor-pointer transition-colors',
                selectedIds.has(parent.id) ? 'bg-primary-soft' : '',
              ]"
            >
              <!-- Checkbox -->
              <td class="px-4 py-3" @click.stop>
                <input data-write
                  type="checkbox"
                  :checked="selectedIds.has(parent.id)"
                  @change="toggleSelect(parent.id, $event)"
                  class="w-4 h-4 text-primary rounded border-border focus:ring-primary"
                />
              </td>
              <!-- Last Name -->
              <td class="px-4 py-3 font-medium">{{ parent.lastName }}</td>
              <!-- First Name -->
              <td class="px-4 py-3">{{ parent.firstName }}</td>
              <!-- Email -->
              <td class="px-4 py-3 text-muted-foreground">
                <span v-if="parent.email" class="truncate">{{ parent.email }}</span>
                <span v-else class="text-muted-foreground">-</span>
              </td>
              <!-- Phone -->
              <td class="px-4 py-3 text-muted-foreground">
                <span v-if="parent.phone">{{ parent.phone }}</span>
                <span v-else class="text-muted-foreground">-</span>
              </td>
              <!-- Children -->
              <td class="px-4 py-3">
                <div v-if="parent.children && parent.children.length > 0" class="flex flex-wrap gap-1">
                  <span
                    v-for="child in parent.children"
                    :key="child.id"
                    class="inline-flex items-center px-2 py-0.5 rounded text-xs bg-muted text-foreground"
                  >
                    {{ child.firstName }}
                  </span>
                </div>
                <span v-else class="text-muted-foreground">-</span>
              </td>
            </tr>
            <tr v-if="parents.length === 0">
              <td colspan="6" class="px-4 py-8 text-center text-muted-foreground">
                Keine Eltern gefunden
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <AppPagination v-model:page-size="pageSize" :current-page="currentPage" :total-pages="totalPages"
        :total="total" :offset="offset" @go="goToPage" />
    </div>

    <!-- Delete Confirmation Dialog (Admin only) -->
    <AppSheet v-if="showDeleteDialog && authStore.isAdmin" title="Eltern löschen" @close="showDeleteDialog = false">

        <div class="note note-danger mb-4">
          <div class="flex items-start gap-2">
            <AlertTriangle class="h-5 w-5 text-destructive flex-shrink-0 mt-0.5" />
            <div>
              <p class="font-semibold text-destructive">Achtung: Permanente Löschung!</p>
              <p class="text-sm text-destructive mt-1">
                Diese Aktion kann nicht rückgängig gemacht werden. Die Verknüpfungen
                zu den Kindern werden ebenfalls entfernt.
              </p>
            </div>
          </div>
        </div>

        <p class="text-muted-foreground mb-6">
          Möchten Sie <strong>{{ selectedIds.size }}</strong> {{ selectedIds.size === 1 ? 'Elternteil' : 'Eltern' }} wirklich
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

    <!-- Create Parent Dialog -->
    <AppSheet v-if="showCreateDialog" title="Elternteil anlegen" @close="showCreateDialog = false">

        <form id="create-parent-form" @submit.prevent="handleCreateParent" class="space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="parent-firstName" class="label">Vorname *</label>
              <input
                id="parent-firstName"
                v-model="parentForm.firstName"
                type="text"
                class="input"
                required
              />
            </div>
            <div>
              <label for="parent-lastName" class="label">Nachname *</label>
              <input
                id="parent-lastName"
                v-model="parentForm.lastName"
                type="text"
                class="input"
                required
              />
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="parent-birthDate" class="label">Geburtsdatum</label>
              <input
                id="parent-birthDate"
                v-model="parentForm.birthDate"
                type="date"
                class="input"
              />
            </div>
            <div>
              <label for="parent-phone" class="label">Telefon</label>
              <input
                id="parent-phone"
                v-model="parentForm.phone"
                type="tel"
                class="input"
              />
            </div>
          </div>

          <div>
            <label for="parent-email" class="label">E-Mail</label>
            <input
              id="parent-email"
              v-model="parentForm.email"
              type="email"
              class="input"
            />
          </div>

          <div class="grid grid-cols-4 gap-4">
            <div class="col-span-3">
              <label for="parent-street" class="label">Straße</label>
              <input
                id="parent-street"
                v-model="parentForm.street"
                type="text"
                class="input"
              />
            </div>
            <div>
              <label for="parent-streetNo" class="label">Hausnr.</label>
              <input
                id="parent-streetNo"
                v-model="parentForm.streetNo"
                type="text"
                class="input"
              />
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="parent-postalCode" class="label">PLZ</label>
              <input
                id="parent-postalCode"
                v-model="parentForm.postalCode"
                type="text"
                class="input"
              />
            </div>
            <div>
              <label for="parent-city" class="label">Ort</label>
              <input
                id="parent-city"
                v-model="parentForm.city"
                type="text"
                class="input"
              />
            </div>
          </div>

          <div v-if="createError" class="note note-danger">
      <AlertTriangle />
      <p class="text-sm text-destructive">{{ createError }}</p>
          </div>


        </form>

    <template #footer><div class="flex justify-end gap-3">
            <button
              type="button"
              @click="showCreateDialog = false"
              class="btn btn-secondary"
            >
              Abbrechen
            </button>
            <button form="create-parent-form" type="submit"
              :disabled="isCreatingParent"
              class="btn btn-primary"
            >
              <Loader2 v-if="isCreatingParent" class="h-4 w-4 animate-spin" />
              <Plus v-else class="h-4 w-4" />
              Speichern
            </button>
          </div></template></AppSheet>
  </div>
</template>
