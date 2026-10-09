<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue';
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
  ChevronUp,
  ChevronDown,
  ChevronsUpDown,
  ChevronLeft,
  ChevronRight,
  Trash2,
  X,
} from 'lucide-vue-next';
import SearchInput from '@/components/SearchInput.vue';
import { usePagedList } from '@/composables/usePagedList';

const router = useRouter();
const authStore = useAuthStore();

// Filters
const searchQuery = ref('');

// Pagination
const pageSizeOptions = [10, 25, 50, 100];

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

// ESC key handler to close modals
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (showDeleteDialog.value) showDeleteDialog.value = false;
    if (showCreateDialog.value) showCreateDialog.value = false;
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown);
});

// Sorting
function toggleSort(field: SortField) {
  if (sortField.value === field) {
    sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc';
  } else {
    sortField.value = field;
    sortDirection.value = 'asc';
  }
}

function getSortIcon(field: SortField) {
  if (sortField.value !== field) return ChevronsUpDown;
  return sortDirection.value === 'asc' ? ChevronUp : ChevronDown;
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

function goToPage(page: number) {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page;
  }
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

// Pagination display helpers
const visiblePages = computed(() => {
  const pages: (number | '...')[] = [];
  const totalPgs = totalPages.value;
  const current = currentPage.value;
  
  if (totalPgs <= 7) {
    for (let i = 1; i <= totalPgs; i++) pages.push(i);
  } else {
    pages.push(1);
    if (current > 3) pages.push('...');
    
    const start = Math.max(2, current - 1);
    const end = Math.min(totalPgs - 1, current + 1);
    
    for (let i = start; i <= end; i++) pages.push(i);
    
    if (current < totalPgs - 2) pages.push('...');
    pages.push(totalPgs);
  }
  
  return pages;
});
</script>

<template>
  <div>
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
      <div>
        <h1 class="text-2xl font-bold text-foreground">Eltern</h1>
        <p class="text-muted-foreground mt-1">{{ total }} Eltern registriert</p>
      </div>
      <button
        @click="openCreateParentDialog"
        class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors"
      >
        <Plus class="h-4 w-4" />
        Elternteil hinzufügen
      </button>
    </div>

    <!-- Search -->
    <SearchInput v-model="searchQuery" placeholder="Suchen nach Name oder E-Mail..." class="mb-6"
      @input="handleSearchInput" />

    <!-- Bulk actions bar -->
    <div
      v-if="selectedIds.size > 0"
      class="mb-4 p-3 bg-blue-50 dark:bg-blue-950/40 border border-blue-200 rounded-lg flex items-center justify-between"
    >
      <span class="text-sm font-medium text-blue-800 dark:text-blue-300">
        {{ selectedIds.size }} {{ selectedIds.size === 1 ? 'Elternteil' : 'Eltern' }} ausgewählt
      </span>
      <div class="flex items-center gap-2">
        <button
          v-if="authStore.isAdmin"
          @click="showDeleteDialog = true"
          class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm bg-red-100 dark:bg-red-950/40 text-red-800 dark:text-red-300 rounded-lg hover:bg-red-200 dark:hover:bg-red-900/50 transition-colors"
        >
          <Trash2 class="h-4 w-4" />
          Löschen
        </button>
        <button
          @click="selectedIds = new Set()"
          class="px-3 py-1.5 text-sm text-muted-foreground hover:bg-accent rounded-lg transition-colors"
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
    <div v-else-if="error" class="bg-red-50 dark:bg-red-950/40 border border-red-200 rounded-lg p-4">
      <p class="text-red-600 dark:text-red-300">{{ error }}</p>
      <button @click="loadParents" class="mt-2 text-sm text-red-700 dark:text-red-300 underline">
        Erneut versuchen
      </button>
    </div>

    <!-- Parents table -->
    <div v-else class="bg-card rounded-xl border overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full">
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
              <!-- Last Name -->
              <th class="px-4 py-3 font-medium">
                <button
                  @click="toggleSort('lastName')"
                  class="flex items-center gap-1 hover:text-foreground"
                >
                  <User class="h-4 w-4" />
                  Nachname
                  <component :is="getSortIcon('lastName')" class="h-4 w-4" />
                </button>
              </th>
              <!-- First Name -->
              <th class="px-4 py-3 font-medium">
                <button
                  @click="toggleSort('firstName')"
                  class="flex items-center gap-1 hover:text-foreground"
                >
                  Vorname
                  <component :is="getSortIcon('firstName')" class="h-4 w-4" />
                </button>
              </th>
              <!-- Email -->
              <th class="px-4 py-3 font-medium">
                <button
                  @click="toggleSort('email')"
                  class="flex items-center gap-1 hover:text-foreground"
                >
                  <Mail class="h-4 w-4" />
                  E-Mail
                  <component :is="getSortIcon('email')" class="h-4 w-4" />
                </button>
              </th>
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
                selectedIds.has(parent.id) ? 'bg-blue-50 dark:bg-blue-950/40' : '',
              ]"
            >
              <!-- Checkbox -->
              <td class="px-4 py-3" @click.stop>
                <input
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

      <!-- Pagination -->
      <div class="flex items-center justify-between px-4 py-3 border-t bg-muted">
        <div class="flex items-center gap-4">
          <span class="text-sm text-muted-foreground">
            {{ offset + 1 }}-{{ Math.min(offset + pageSize, total) }} von {{ total }}
          </span>
          <select
            v-model="pageSize"
            class="text-sm border border-border rounded px-2 py-1 focus:ring-primary focus:border-primary"
          >
            <option v-for="size in pageSizeOptions" :key="size" :value="size">
              {{ size }} pro Seite
            </option>
          </select>
        </div>
        <div class="flex items-center gap-1">
          <button
            @click="goToPage(currentPage - 1)"
            :disabled="currentPage === 1"
            class="p-1.5 rounded hover:bg-accent disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <ChevronLeft class="h-4 w-4" />
          </button>
          <template v-for="page in visiblePages" :key="page">
            <span v-if="page === '...'" class="px-2 text-muted-foreground">...</span>
            <button
              v-else
              @click="goToPage(page)"
              :class="[
                'px-3 py-1 rounded text-sm',
                page === currentPage
                  ? 'bg-primary text-primary-foreground'
                  : 'hover:bg-accent',
              ]"
            >
              {{ page }}
            </button>
          </template>
          <button
            @click="goToPage(currentPage + 1)"
            :disabled="currentPage === totalPages"
            class="p-1.5 rounded hover:bg-accent disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <ChevronRight class="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>

    <!-- Delete Confirmation Dialog (Admin only) -->
    <div
      v-if="showDeleteDialog && authStore.isAdmin"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="showDeleteDialog = false"
    >
      <div class="bg-card rounded-xl shadow-xl w-full max-w-md mx-4 p-6">
        <div class="flex items-center gap-3 mb-4">
          <div class="w-10 h-10 bg-red-100 dark:bg-red-950/40 rounded-full flex items-center justify-center">
            <Trash2 class="h-5 w-5 text-red-600 dark:text-red-300" />
          </div>
          <h2 class="text-xl font-semibold text-red-700 dark:text-red-300">Eltern löschen</h2>
        </div>
        
        <div class="bg-red-50 dark:bg-red-950/40 border border-red-200 rounded-lg p-4 mb-4">
          <div class="flex items-start gap-2">
            <AlertTriangle class="h-5 w-5 text-red-600 dark:text-red-300 flex-shrink-0 mt-0.5" />
            <div>
              <p class="font-semibold text-red-800 dark:text-red-300">Achtung: Permanente Löschung!</p>
              <p class="text-sm text-red-700 dark:text-red-300 mt-1">
                Diese Aktion kann nicht rückgängig gemacht werden. Die Verknüpfungen 
                zu den Kindern werden ebenfalls entfernt.
              </p>
            </div>
          </div>
        </div>

        <p class="text-muted-foreground mb-6">
          Möchten Sie <strong>{{ selectedIds.size }}</strong> {{ selectedIds.size === 1 ? 'Elternteil' : 'Eltern' }} wirklich
          <strong class="text-red-600 dark:text-red-300">unwiderruflich löschen</strong>?
        </p>

        <div v-if="bulkActionError" class="mb-4 p-3 bg-red-50 dark:bg-red-950/40 border border-red-200 rounded-lg">
          <p class="text-sm text-red-600 dark:text-red-300">{{ bulkActionError }}</p>
        </div>

        <div class="flex justify-end gap-3">
          <button
            @click="showDeleteDialog = false"
            class="px-4 py-2 text-foreground hover:bg-accent rounded-lg transition-colors"
          >
            Abbrechen
          </button>
          <button
            @click="handleBulkDelete"
            :disabled="isBulkActionLoading"
            class="inline-flex items-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isBulkActionLoading" class="h-4 w-4 animate-spin" />
            <Trash2 v-else class="h-4 w-4" />
            Endgültig löschen
          </button>
        </div>
      </div>
    </div>

    <!-- Create Parent Dialog -->
    <div
      v-if="showCreateDialog"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="showCreateDialog = false"
    >
      <div class="bg-card rounded-xl shadow-xl w-full max-w-lg mx-4 p-6">
        <div class="flex items-center justify-between mb-6">
          <h2 class="text-xl font-semibold">Elternteil anlegen</h2>
          <button @click="showCreateDialog = false" class="p-1 hover:bg-accent rounded" aria-label="Schließen">
            <X class="h-5 w-5" />
          </button>
        </div>

        <form @submit.prevent="handleCreateParent" class="space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="parent-firstName" class="block text-sm font-medium text-foreground mb-1">Vorname *</label>
              <input
                id="parent-firstName"
                v-model="parentForm.firstName"
                type="text"
                class="w-full px-3 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
                required
              />
            </div>
            <div>
              <label for="parent-lastName" class="block text-sm font-medium text-foreground mb-1">Nachname *</label>
              <input
                id="parent-lastName"
                v-model="parentForm.lastName"
                type="text"
                class="w-full px-3 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
                required
              />
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="parent-birthDate" class="block text-sm font-medium text-foreground mb-1">Geburtsdatum</label>
              <input
                id="parent-birthDate"
                v-model="parentForm.birthDate"
                type="date"
                class="w-full px-3 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div>
              <label for="parent-phone" class="block text-sm font-medium text-foreground mb-1">Telefon</label>
              <input
                id="parent-phone"
                v-model="parentForm.phone"
                type="tel"
                class="w-full px-3 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div>
            <label for="parent-email" class="block text-sm font-medium text-foreground mb-1">E-Mail</label>
            <input
              id="parent-email"
              v-model="parentForm.email"
              type="email"
              class="w-full px-3 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>

          <div class="grid grid-cols-4 gap-4">
            <div class="col-span-3">
              <label for="parent-street" class="block text-sm font-medium text-foreground mb-1">Straße</label>
              <input
                id="parent-street"
                v-model="parentForm.street"
                type="text"
                class="w-full px-3 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div>
              <label for="parent-streetNo" class="block text-sm font-medium text-foreground mb-1">Hausnr.</label>
              <input
                id="parent-streetNo"
                v-model="parentForm.streetNo"
                type="text"
                class="w-full px-3 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="parent-postalCode" class="block text-sm font-medium text-foreground mb-1">PLZ</label>
              <input
                id="parent-postalCode"
                v-model="parentForm.postalCode"
                type="text"
                class="w-full px-3 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div>
              <label for="parent-city" class="block text-sm font-medium text-foreground mb-1">Ort</label>
              <input
                id="parent-city"
                v-model="parentForm.city"
                type="text"
                class="w-full px-3 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div v-if="createError" class="p-3 bg-red-50 dark:bg-red-950/40 border border-red-200 rounded-lg">
            <p class="text-sm text-red-600 dark:text-red-300">{{ createError }}</p>
          </div>

          <div class="flex justify-end gap-3">
            <button
              type="button"
              @click="showCreateDialog = false"
              class="px-4 py-2 text-foreground hover:bg-accent rounded-lg transition-colors"
            >
              Abbrechen
            </button>
            <button
              type="submit"
              :disabled="isCreatingParent"
              class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Loader2 v-if="isCreatingParent" class="h-4 w-4 animate-spin" />
              <Plus v-else class="h-4 w-4" />
              Speichern
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
