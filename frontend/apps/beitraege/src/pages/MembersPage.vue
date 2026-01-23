<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue';
import { useRouter } from 'vue-router';
import { api } from '@/api';
import { useAuthStore } from '@/stores/auth';
import type { Member, CreateMemberRequest } from '@/api/types';
import {
  Plus,
  Search,
  Loader2,
  User,
  Hash,
  Mail,
  Phone,
  Calendar,
  X,
  Check,
  AlertTriangle,
  ChevronUp,
  ChevronDown,
  ChevronsUpDown,
  ChevronLeft,
  ChevronRight,
  Trash2,
  UserX,
} from 'lucide-vue-next';

const router = useRouter();
const authStore = useAuthStore();

// Data
const members = ref<Member[]>([]);
const total = ref(0);
const isLoading = ref(true);
const error = ref<string | null>(null);

// Filters
const searchQuery = ref('');
const showInactive = ref(false);

// Pagination
const currentPage = ref(1);
const pageSize = ref(25);
const pageSizeOptions = [10, 25, 50, 100];

// Sorting
type SortField = 'memberNumber' | 'lastName' | 'firstName' | 'email' | 'membershipStart';
type SortDirection = 'asc' | 'desc';
const sortField = ref<SortField>('lastName');
const sortDirection = ref<SortDirection>('asc');

// Bulk selection
const selectedIds = ref<Set<string>>(new Set());
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

// Create form
const createForm = ref<CreateMemberRequest>({
  firstName: '',
  lastName: '',
  email: '',
  phone: '',
  membershipStart: new Date().toISOString().split('T')[0],
});
const isCreating = ref(false);
const createError = ref<string | null>(null);

// Computed
const totalPages = computed(() => Math.ceil(total.value / pageSize.value));
const offset = computed(() => (currentPage.value - 1) * pageSize.value);

async function loadMembers() {
  isLoading.value = true;
  error.value = null;
  try {
    const response = await api.getMembers({
      activeOnly: !showInactive.value,
      search: searchQuery.value || undefined,
      sortBy: sortField.value,
      sortDir: sortDirection.value,
      offset: offset.value,
      limit: pageSize.value,
    });
    members.value = response.data;
    total.value = response.total;
    
    // Clear selection if items no longer exist
    const currentIds = new Set(response.data.map(m => m.id));
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
    loadMembers();
  }, 150);
}

function handleInactiveChange() {
  currentPage.value = 1;
  loadMembers();
}

// Watch for filter changes (backup for programmatic v-model changes)
watch([searchQuery, showInactive], () => {
  currentPage.value = 1;
  loadMembers();
}, { flush: 'post' });

watch([currentPage, pageSize], () => {
  loadMembers();
});

// Reload when sort changes
watch([sortField, sortDirection], () => {
  currentPage.value = 1;
  loadMembers();
});

onMounted(loadMembers);

// ESC key handler to close modals
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (showDeleteDialog.value) showDeleteDialog.value = false;
    else if (showDeactivateDialog.value) showDeactivateDialog.value = false;
    else if (showCreateDialog.value) showCreateDialog.value = false;
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown);
});

// Helpers
function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('de-DE');
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

function getSortIcon(field: SortField) {
  if (sortField.value !== field) return ChevronsUpDown;
  return sortDirection.value === 'asc' ? ChevronUp : ChevronDown;
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

function goToPage(page: number) {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page;
  }
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
      membershipStart: new Date().toISOString().split('T')[0],
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
        <h1 class="text-2xl font-bold text-gray-900">Vereinsmitglieder</h1>
        <p class="text-gray-600 mt-1">{{ total }} Mitglieder registriert</p>
      </div>
      <button
        @click="showCreateDialog = true"
        class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
      >
        <Plus class="h-4 w-4" />
        Mitglied hinzufügen
      </button>
    </div>

    <!-- Filters -->
    <div class="flex flex-col sm:flex-row gap-4 mb-6">
      <div class="relative flex-1">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
        <input
          v-model="searchQuery"
          @input="handleSearchInput"
          type="text"
          placeholder="Suchen nach Name, Mitgliedsnummer oder E-Mail..."
          class="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
        />
      </div>
      <label class="flex items-center gap-2 cursor-pointer">
        <input
          v-model="showInactive"
          @change="handleInactiveChange"
          type="checkbox"
          class="w-4 h-4 text-primary rounded border-gray-300 focus:ring-primary"
        />
        <span class="text-sm text-gray-700">Inaktive anzeigen</span>
      </label>
    </div>

    <!-- Bulk actions bar -->
    <div
      v-if="selectedIds.size > 0"
      class="mb-4 p-3 bg-blue-50 border border-blue-200 rounded-lg flex items-center justify-between"
    >
      <span class="text-sm font-medium text-blue-800">
        {{ selectedIds.size }} {{ selectedIds.size === 1 ? 'Mitglied' : 'Mitglieder' }} ausgewählt
      </span>
      <div class="flex items-center gap-2">
        <button
          @click="showDeactivateDialog = true"
          class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm bg-amber-100 text-amber-800 rounded-lg hover:bg-amber-200 transition-colors"
        >
          <UserX class="h-4 w-4" />
          Deaktivieren
        </button>
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
      <button @click="loadMembers" class="mt-2 text-sm text-red-700 underline">
        Erneut versuchen
      </button>
    </div>

    <!-- Members table -->
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
              <!-- Member number -->
              <th class="px-4 py-3 font-medium">
                <button
                  @click="toggleSort('memberNumber')"
                  class="flex items-center gap-1 hover:text-gray-700"
                >
                  <Hash class="h-4 w-4" />
                  Nr.
                  <component :is="getSortIcon('memberNumber')" class="h-4 w-4" />
                </button>
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
              <!-- Membership Start -->
              <th class="px-4 py-3 font-medium">
                <button
                  @click="toggleSort('membershipStart')"
                  class="flex items-center gap-1 hover:text-gray-700"
                >
                  <Calendar class="h-4 w-4" />
                  Mitglied seit
                  <component :is="getSortIcon('membershipStart')" class="h-4 w-4" />
                </button>
              </th>
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
                'border-t hover:bg-gray-50 cursor-pointer transition-colors',
                selectedIds.has(member.id) ? 'bg-blue-50' : '',
              ]"
            >
              <!-- Checkbox -->
              <td class="px-4 py-3" @click.stop>
                <input
                  type="checkbox"
                  :checked="selectedIds.has(member.id)"
                  @change="toggleSelect(member.id, $event)"
                  class="w-4 h-4 text-primary rounded border-gray-300 focus:ring-primary"
                />
              </td>
              <!-- Member number -->
              <td class="px-4 py-3 font-mono text-sm">{{ member.memberNumber }}</td>
              <!-- Last Name -->
              <td class="px-4 py-3 font-medium">{{ member.lastName }}</td>
              <!-- First Name -->
              <td class="px-4 py-3">{{ member.firstName }}</td>
              <!-- Email -->
              <td class="px-4 py-3 text-gray-600">
                <span v-if="member.email" class="truncate">{{ member.email }}</span>
                <span v-else class="text-gray-400">-</span>
              </td>
              <!-- Phone -->
              <td class="px-4 py-3 text-gray-600">
                <span v-if="member.phone">{{ member.phone }}</span>
                <span v-else class="text-gray-400">-</span>
              </td>
              <!-- Membership Start -->
              <td class="px-4 py-3 text-gray-600">{{ formatDate(member.membershipStart) }}</td>
              <!-- Status -->
              <td class="px-4 py-3">
                <span
                  :class="[
                    'inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium',
                    member.isActive ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600',
                  ]"
                >
                  {{ member.isActive ? 'Aktiv' : 'Inaktiv' }}
                </span>
              </td>
            </tr>
            <tr v-if="members.length === 0">
              <td colspan="8" class="px-4 py-8 text-center text-gray-500">
                Keine Mitglieder gefunden
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

    <!-- Create Dialog -->
    <div
      v-if="showCreateDialog"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="showCreateDialog = false"
    >
      <div class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4 p-6">
        <div class="flex items-center justify-between mb-6">
          <h2 class="text-xl font-semibold">Mitglied hinzufügen</h2>
          <button @click="showCreateDialog = false" class="p-1 hover:bg-gray-100 rounded">
            <X class="h-5 w-5" />
          </button>
        </div>

        <form @submit.prevent="handleCreate" class="space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="firstName" class="block text-sm font-medium text-gray-700 mb-1">Vorname *</label>
              <input
                id="firstName"
                v-model="createForm.firstName"
                required
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div>
              <label for="lastName" class="block text-sm font-medium text-gray-700 mb-1">Nachname *</label>
              <input
                id="lastName"
                v-model="createForm.lastName"
                required
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div>
            <label for="email" class="block text-sm font-medium text-gray-700 mb-1">E-Mail</label>
            <input
              id="email"
              v-model="createForm.email"
              type="email"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>

          <div>
            <label for="phone" class="block text-sm font-medium text-gray-700 mb-1">Telefon</label>
            <input
              id="phone"
              v-model="createForm.phone"
              type="tel"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>

          <div>
            <label for="membershipStart" class="block text-sm font-medium text-gray-700 mb-1">Mitglied seit *</label>
            <input
              id="membershipStart"
              v-model="createForm.membershipStart"
              required
              type="date"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>

          <div v-if="createError" class="p-3 bg-red-50 border border-red-200 rounded-lg">
            <p class="text-sm text-red-600">{{ createError }}</p>
          </div>

          <div class="flex justify-end gap-3 pt-4">
            <button
              type="button"
              @click="showCreateDialog = false"
              class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
            >
              Abbrechen
            </button>
            <button
              type="submit"
              :disabled="isCreating"
              class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Loader2 v-if="isCreating" class="h-4 w-4 animate-spin" />
              <Check v-else class="h-4 w-4" />
              Speichern
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Deactivate Confirmation Dialog -->
    <div
      v-if="showDeactivateDialog"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="showDeactivateDialog = false"
    >
      <div class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4 p-6">
        <div class="flex items-center gap-3 mb-4">
          <div class="w-10 h-10 bg-amber-100 rounded-full flex items-center justify-center">
            <UserX class="h-5 w-5 text-amber-600" />
          </div>
          <h2 class="text-xl font-semibold">Mitglieder deaktivieren</h2>
        </div>
        
        <p class="text-gray-600 mb-6">
          Möchten Sie <strong>{{ selectedIds.size }}</strong> {{ selectedIds.size === 1 ? 'Mitglied' : 'Mitglieder' }} wirklich deaktivieren?
          Deaktivierte Mitglieder werden nicht mehr in den Standardlisten angezeigt.
        </p>

        <div v-if="bulkActionError" class="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg">
          <p class="text-sm text-red-600">{{ bulkActionError }}</p>
        </div>

        <div class="flex justify-end gap-3">
          <button
            @click="showDeactivateDialog = false"
            class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
          >
            Abbrechen
          </button>
          <button
            @click="handleBulkDeactivate"
            :disabled="isBulkActionLoading"
            class="inline-flex items-center gap-2 px-4 py-2 bg-amber-600 text-white rounded-lg hover:bg-amber-700 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isBulkActionLoading" class="h-4 w-4 animate-spin" />
            <UserX v-else class="h-4 w-4" />
            Deaktivieren
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
          <h2 class="text-xl font-semibold text-red-700">Mitglieder löschen</h2>
        </div>
        
        <div class="bg-red-50 border border-red-200 rounded-lg p-4 mb-4">
          <div class="flex items-start gap-2">
            <AlertTriangle class="h-5 w-5 text-red-600 flex-shrink-0 mt-0.5" />
            <div>
              <p class="font-semibold text-red-800">Achtung: Permanente Löschung!</p>
              <p class="text-sm text-red-700 mt-1">
                Diese Aktion kann nicht rückgängig gemacht werden. Alle zugehörigen Daten
                werden ebenfalls gelöscht.
              </p>
            </div>
          </div>
        </div>

        <p class="text-gray-600 mb-6">
          Möchten Sie <strong>{{ selectedIds.size }}</strong> {{ selectedIds.size === 1 ? 'Mitglied' : 'Mitglieder' }} wirklich
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
