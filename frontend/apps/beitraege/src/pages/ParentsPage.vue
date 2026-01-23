<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue';
import { useRouter } from 'vue-router';
import { api } from '@/api';
import { useAuthStore } from '@/stores/auth';
import type { Parent } from '@/api/types';
import {
  Plus,
  Search,
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
} from 'lucide-vue-next';

const router = useRouter();
const authStore = useAuthStore();

// Data
const parents = ref<Parent[]>([]);
const total = ref(0);
const isLoading = ref(true);
const error = ref<string | null>(null);

// Filters
const searchQuery = ref('');

// Pagination
const currentPage = ref(1);
const pageSize = ref(25);
const pageSizeOptions = [10, 25, 50, 100];

// Sorting
type SortField = 'lastName' | 'firstName' | 'email';
type SortDirection = 'asc' | 'desc';
const sortField = ref<SortField>('lastName');
const sortDirection = ref<SortDirection>('asc');

// Bulk selection
const selectedIds = ref<Set<string>>(new Set());
const isAllSelected = computed(() => {
  if (parents.value.length === 0) return false;
  return parents.value.every(p => selectedIds.value.has(p.id));
});
const isSomeSelected = computed(() => {
  return selectedIds.value.size > 0 && !isAllSelected.value;
});

// Dialogs
const showDeleteDialog = ref(false);
const isBulkActionLoading = ref(false);
const bulkActionError = ref<string | null>(null);

// Computed
const totalPages = computed(() => Math.ceil(total.value / pageSize.value));
const offset = computed(() => (currentPage.value - 1) * pageSize.value);

async function loadParents() {
  isLoading.value = true;
  error.value = null;
  try {
    const response = await api.getParents({
      search: searchQuery.value || undefined,
      sortBy: sortField.value,
      sortDir: sortDirection.value,
      offset: offset.value,
      limit: pageSize.value,
    });
    parents.value = response.data;
    total.value = response.total;
    
    // Clear selection if items no longer exist
    const currentIds = new Set(response.data.map(p => p.id));
    selectedIds.value = new Set([...selectedIds.value].filter(id => currentIds.has(id)));
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoading.value = false;
  }
}

// Debounce timer for search
let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null;

function handleSearchInput() {
  if (searchDebounceTimer) {
    clearTimeout(searchDebounceTimer);
  }
  searchDebounceTimer = setTimeout(() => {
    currentPage.value = 1;
    loadParents();
  }, 150);
}

// Watch for pagination changes
watch([currentPage, pageSize], () => {
  loadParents();
});

// Reload when sort changes
watch([sortField, sortDirection], () => {
  currentPage.value = 1;
  loadParents();
});

onMounted(loadParents);

// ESC key handler to close modals
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (showDeleteDialog.value) showDeleteDialog.value = false;
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
        <h1 class="text-2xl font-bold text-gray-900">Eltern</h1>
        <p class="text-gray-600 mt-1">{{ total }} Eltern registriert</p>
      </div>
      <button
        class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
      >
        <Plus class="h-4 w-4" />
        Elternteil hinzufügen
      </button>
    </div>

    <!-- Search -->
    <div class="relative mb-6">
      <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
      <input
        v-model="searchQuery"
        @input="handleSearchInput"
        type="text"
        placeholder="Suchen nach Name oder E-Mail..."
        class="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
      />
    </div>

    <!-- Bulk actions bar -->
    <div
      v-if="selectedIds.size > 0"
      class="mb-4 p-3 bg-blue-50 border border-blue-200 rounded-lg flex items-center justify-between"
    >
      <span class="text-sm font-medium text-blue-800">
        {{ selectedIds.size }} {{ selectedIds.size === 1 ? 'Elternteil' : 'Eltern' }} ausgewählt
      </span>
      <div class="flex items-center gap-2">
        <button
          v-if="authStore.isAdmin"
          @click="showDeleteDialog = true"
          class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm bg-red-100 text-red-800 rounded-lg hover:bg-red-200 transition-colors"
        >
          <Trash2 class="h-4 w-4" />
          Löschen
        </button>
        <button
          @click="selectedIds = new Set()"
          class="px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
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
    <div v-else-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4">
      <p class="text-red-600">{{ error }}</p>
      <button @click="loadParents" class="mt-2 text-sm text-red-700 underline">
        Erneut versuchen
      </button>
    </div>

    <!-- Parents table -->
    <div v-else class="bg-white rounded-xl border overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full">
          <thead class="bg-gray-50">
            <tr class="text-left text-sm text-gray-500">
              <!-- Checkbox column -->
              <th class="px-4 py-3 w-12">
                <input
                  type="checkbox"
                  :checked="isAllSelected"
                  :indeterminate="isSomeSelected"
                  @change="toggleSelectAll"
                  class="w-4 h-4 text-primary rounded border-gray-300 focus:ring-primary"
                />
              </th>
              <!-- Last Name -->
              <th class="px-4 py-3 font-medium">
                <button
                  @click="toggleSort('lastName')"
                  class="flex items-center gap-1 hover:text-gray-700"
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
                  class="flex items-center gap-1 hover:text-gray-700"
                >
                  Vorname
                  <component :is="getSortIcon('firstName')" class="h-4 w-4" />
                </button>
              </th>
              <!-- Email -->
              <th class="px-4 py-3 font-medium">
                <button
                  @click="toggleSort('email')"
                  class="flex items-center gap-1 hover:text-gray-700"
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
                'border-t hover:bg-gray-50 cursor-pointer transition-colors',
                selectedIds.has(parent.id) ? 'bg-blue-50' : '',
              ]"
            >
              <!-- Checkbox -->
              <td class="px-4 py-3" @click.stop>
                <input
                  type="checkbox"
                  :checked="selectedIds.has(parent.id)"
                  @change="toggleSelect(parent.id, $event)"
                  class="w-4 h-4 text-primary rounded border-gray-300 focus:ring-primary"
                />
              </td>
              <!-- Last Name -->
              <td class="px-4 py-3 font-medium">{{ parent.lastName }}</td>
              <!-- First Name -->
              <td class="px-4 py-3">{{ parent.firstName }}</td>
              <!-- Email -->
              <td class="px-4 py-3 text-gray-600">
                <span v-if="parent.email" class="truncate">{{ parent.email }}</span>
                <span v-else class="text-gray-400">-</span>
              </td>
              <!-- Phone -->
              <td class="px-4 py-3 text-gray-600">
                <span v-if="parent.phone">{{ parent.phone }}</span>
                <span v-else class="text-gray-400">-</span>
              </td>
              <!-- Children -->
              <td class="px-4 py-3">
                <div v-if="parent.children && parent.children.length > 0" class="flex flex-wrap gap-1">
                  <span
                    v-for="child in parent.children"
                    :key="child.id"
                    class="inline-flex items-center px-2 py-0.5 rounded text-xs bg-gray-100 text-gray-700"
                  >
                    {{ child.firstName }}
                  </span>
                </div>
                <span v-else class="text-gray-400">-</span>
              </td>
            </tr>
            <tr v-if="parents.length === 0">
              <td colspan="6" class="px-4 py-8 text-center text-gray-500">
                Keine Eltern gefunden
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div class="flex items-center justify-between px-4 py-3 border-t bg-gray-50">
        <div class="flex items-center gap-4">
          <span class="text-sm text-gray-600">
            {{ offset + 1 }}-{{ Math.min(offset + pageSize, total) }} von {{ total }}
          </span>
          <select
            v-model="pageSize"
            class="text-sm border border-gray-300 rounded px-2 py-1 focus:ring-primary focus:border-primary"
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
            class="p-1.5 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <ChevronLeft class="h-4 w-4" />
          </button>
          <template v-for="page in visiblePages" :key="page">
            <span v-if="page === '...'" class="px-2 text-gray-400">...</span>
            <button
              v-else
              @click="goToPage(page)"
              :class="[
                'px-3 py-1 rounded text-sm',
                page === currentPage
                  ? 'bg-primary text-white'
                  : 'hover:bg-gray-200',
              ]"
            >
              {{ page }}
            </button>
          </template>
          <button
            @click="goToPage(currentPage + 1)"
            :disabled="currentPage === totalPages"
            class="p-1.5 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
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
      <div class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4 p-6">
        <div class="flex items-center gap-3 mb-4">
          <div class="w-10 h-10 bg-red-100 rounded-full flex items-center justify-center">
            <Trash2 class="h-5 w-5 text-red-600" />
          </div>
          <h2 class="text-xl font-semibold text-red-700">Eltern löschen</h2>
        </div>
        
        <div class="bg-red-50 border border-red-200 rounded-lg p-4 mb-4">
          <div class="flex items-start gap-2">
            <AlertTriangle class="h-5 w-5 text-red-600 flex-shrink-0 mt-0.5" />
            <div>
              <p class="font-semibold text-red-800">Achtung: Permanente Löschung!</p>
              <p class="text-sm text-red-700 mt-1">
                Diese Aktion kann nicht rückgängig gemacht werden. Die Verknüpfungen 
                zu den Kindern werden ebenfalls entfernt.
              </p>
            </div>
          </div>
        </div>

        <p class="text-gray-600 mb-6">
          Möchten Sie <strong>{{ selectedIds.size }}</strong> {{ selectedIds.size === 1 ? 'Elternteil' : 'Eltern' }} wirklich
          <strong class="text-red-600">unwiderruflich löschen</strong>?
        </p>

        <div v-if="bulkActionError" class="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg">
          <p class="text-sm text-red-600">{{ bulkActionError }}</p>
        </div>

        <div class="flex justify-end gap-3">
          <button
            @click="showDeleteDialog = false"
            class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
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
  </div>
</template>
