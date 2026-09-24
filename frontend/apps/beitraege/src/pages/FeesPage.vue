<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { api } from '@/api';
import type { FeeExpectation } from '@/api/types';
import {
  Filter,
  Loader2,
  Plus,
  CheckCircle,
  Clock,
  AlertTriangle,
  Calendar,
  Trash2,
  Search,
  AlertCircle,
  ChevronUp,
  ChevronDown,
  ChevronsUpDown,
  ChevronLeft,
  ChevronRight,
} from 'lucide-vue-next';
import { formatCurrency, formatDate, formatMonthName } from '@/utils/format';
import { getFeeTypeColor, getFeeTypeName } from '@/utils/fees';
import GenerateFeesDialog from '@/components/fees/GenerateFeesDialog.vue';
import CreateFeeDialog from '@/components/fees/CreateFeeDialog.vue';

const router = useRouter();
const route = useRoute();

const fees = ref<FeeExpectation[]>([]);
const total = ref(0);
const isLoading = ref(true);
const error = ref<string | null>(null);

const selectedType = ref<string>('');
const selectedStatus = ref<'open' | 'paid' | 'all'>('open');
const searchQuery = ref('');
const debouncedSearch = ref('');

type FeeSortKey = 'memberNumber' | 'childName' | 'feeType' | 'period' | 'amount';
const sortBy = ref<FeeSortKey>('period');
const sortDir = ref<'asc' | 'desc'>('desc');
const defaultSortDir: Record<FeeSortKey, 'asc' | 'desc'> = {
  memberNumber: 'asc',
  childName: 'asc',
  feeType: 'asc',
  period: 'desc',
  amount: 'desc',
};

const showGenerateDialog = ref(false);

// Pagination
const currentPage = ref(1);
const pageSize = ref(25);
const pageSizeOptions = [10, 25, 50, 100];

// Selection state
const selectedFeeIds = ref<Set<string>>(new Set());
const isDeleting = ref(false);
const showDeleteConfirm = ref(false);

// Reminder state
const showReminderConfirm = ref(false);
const reminderTargetFee = ref<FeeExpectation | null>(null);
const isCreatingReminder = ref(false);

// Single fee creation state
const showCreateFeeDialog = ref(false);

const feeTypes = [
  { value: '', label: 'Alle' },
  { value: 'MEMBERSHIP', label: 'Vereinsbeitrag' },
  { value: 'FOOD', label: 'Essensgeld' },
  { value: 'CHILDCARE', label: 'Platzgeld' },
  { value: 'REMINDER', label: 'Mahngebühr' },
];

// Computed for selection
const allSelected = computed(() => {
  return fees.value.length > 0 && selectedFeeIds.value.size === fees.value.length;
});

const someSelected = computed(() => {
  return selectedFeeIds.value.size > 0 && selectedFeeIds.value.size < fees.value.length;
});

const selectedCount = computed(() => selectedFeeIds.value.size);

const selectedDeletableCount = computed(() => {
  return Array.from(selectedFeeIds.value).filter(id => {
    const fee = fees.value.find(f => f.id === id);
    return fee && !fee.isPaid;
  }).length;
});

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

// Computed pagination helpers
const totalPages = computed(() => Math.ceil(total.value / pageSize.value));
const offset = computed(() => (currentPage.value - 1) * pageSize.value);

// Debounce timer for search
let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null;

function handleSearchInput() {
  if (searchDebounceTimer) {
    clearTimeout(searchDebounceTimer);
  }
  searchDebounceTimer = setTimeout(() => {
    debouncedSearch.value = searchQuery.value;
    currentPage.value = 1;
    loadFees();
  }, 150);
}

let loadFeesSeq = 0;
async function loadFees() {
  const seq = ++loadFeesSeq;
  isLoading.value = true;
  error.value = null;
  selectedFeeIds.value = new Set(); // Clear selection on reload
  try {
    const response = await api.getFees({
      feeType: selectedType.value || undefined,
      status: selectedStatus.value !== 'all' ? selectedStatus.value : undefined,
      search: debouncedSearch.value || undefined,
      page: currentPage.value,
      perPage: pageSize.value,
      sortBy: sortBy.value,
      sortDir: sortDir.value,
    });
    if (seq !== loadFeesSeq) return; // a newer request superseded this one
    fees.value = response.data;
    total.value = response.total;

    // If the current page ran empty (e.g. after deletes), fall back to the last valid page
    if (response.data.length === 0 && currentPage.value > 1 && response.total > 0) {
      currentPage.value = Math.max(1, Math.ceil(response.total / pageSize.value));
      return;
    }
  } catch (e) {
    if (seq !== loadFeesSeq) return;
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    if (seq === loadFeesSeq) isLoading.value = false;
  }
}

function toggleSort(column: FeeSortKey) {
  if (sortBy.value === column) {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc';
  } else {
    sortBy.value = column;
    sortDir.value = defaultSortDir[column];
  }
}

function getSortIcon(column: FeeSortKey) {
  if (sortBy.value !== column) {
    return ChevronsUpDown;
  }
  return sortDir.value === 'asc' ? ChevronUp : ChevronDown;
}

function isSorted(column: FeeSortKey) {
  return sortBy.value === column;
}

function setTypeFilter(value: string) {
  selectedType.value = value;
  currentPage.value = 1;
  loadFees();
}

function setStatusFilter(value: 'open' | 'paid' | 'all') {
  selectedStatus.value = value;
  currentPage.value = 1;
  loadFees();
}

function goToChild(childId: string) {
  router.push(`/kinder/${childId}`);
}

function goToPage(page: number) {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page;
  }
}

// ESC key handler to close modals
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (showDeleteConfirm.value) showDeleteConfirm.value = false;
    else if (showReminderConfirm.value) showReminderConfirm.value = false;
    else if (showCreateFeeDialog.value) showCreateFeeDialog.value = false;
    else if (showGenerateDialog.value) showGenerateDialog.value = false;
  }
}

onMounted(() => {
  // Read URL query params for deep-linking from dashboard
  const feeTypeParam = route.query.feeType as string;
  if (feeTypeParam && ['MEMBERSHIP', 'FOOD', 'CHILDCARE', 'REMINDER'].includes(feeTypeParam)) {
    selectedType.value = feeTypeParam;
  }

  const statusParam = route.query.status as string;
  if (statusParam && ['open', 'paid', 'all'].includes(statusParam)) {
    selectedStatus.value = statusParam as 'open' | 'paid' | 'all';
  }

  loadFees();
  document.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown);
});

// Reload when pagination changes
watch([currentPage, pageSize], () => {
  loadFees();
});

// Reload when sort changes
watch([sortBy, sortDir], () => {
  currentPage.value = 1;
  loadFees();
});

function getStatusInfo(fee: FeeExpectation) {
  if (fee.isPaid) {
    return { icon: CheckCircle, color: 'text-green-500', label: 'Bezahlt', bg: 'bg-green-50' };
  }
  const isOverdue = new Date(fee.dueDate) < new Date();
  if (isOverdue) {
    return { icon: AlertTriangle, color: 'text-red-500', label: 'Überfällig', bg: 'bg-red-50' };
  }
  return { icon: Clock, color: 'text-amber-500', label: 'Offen', bg: 'bg-amber-50' };
}

// Selection functions
function toggleSelectAll() {
  if (allSelected.value) {
    selectedFeeIds.value = new Set();
  } else {
    selectedFeeIds.value = new Set(fees.value.map(f => f.id));
  }
}

function toggleSelect(id: string) {
  const newSet = new Set(selectedFeeIds.value);
  if (newSet.has(id)) {
    newSet.delete(id);
  } else {
    newSet.add(id);
  }
  selectedFeeIds.value = newSet;
}

function isSelected(id: string): boolean {
  return selectedFeeIds.value.has(id);
}

function clearSelection() {
  selectedFeeIds.value = new Set();
}

// Delete functions
async function deleteSingleFee(fee: FeeExpectation) {
  if (fee.isPaid) {
    error.value = 'Bezahlte Beiträge können nicht gelöscht werden';
    return;
  }
  
  if (!confirm(`Beitrag "${getFeeTypeName(fee.feeType)}" für ${fee.child?.firstName} ${fee.child?.lastName} wirklich löschen?`)) {
    return;
  }
  
  try {
    await api.deleteFee(fee.id);
    await loadFees();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Löschen';
  }
}

async function deleteSelectedFees() {
  isDeleting.value = true;
  error.value = null;
  
  const idsToDelete = Array.from(selectedFeeIds.value).filter(id => {
    const fee = fees.value.find(f => f.id === id);
    return fee && !fee.isPaid;
  });
  
  let deleted = 0;
  let failed = 0;
  
  for (const id of idsToDelete) {
    try {
      await api.deleteFee(id);
      deleted++;
    } catch {
      failed++;
    }
  }
  
  showDeleteConfirm.value = false;
  isDeleting.value = false;
  
  if (failed > 0) {
    error.value = `${deleted} gelöscht, ${failed} fehlgeschlagen`;
  }
  
  await loadFees();
}

// Check if fee is overdue and can have a reminder created
function canCreateReminder(fee: FeeExpectation): boolean {
  if (fee.isPaid) return false;
  if (fee.feeType === 'REMINDER') return false;
  const isOverdue = new Date(fee.dueDate) < new Date();
  if (!isOverdue) return false;
  // Check if there's already a reminder for this fee
  const hasReminder = fees.value.some(f => f.reminderForId === fee.id);
  return !hasReminder;
}

function openReminderDialog(fee: FeeExpectation) {
  reminderTargetFee.value = fee;
  showReminderConfirm.value = true;
}

async function createReminder() {
  if (!reminderTargetFee.value) return;
  
  isCreatingReminder.value = true;
  error.value = null;
  
  try {
    await api.createReminder(reminderTargetFee.value.id);
    showReminderConfirm.value = false;
    reminderTargetFee.value = null;
    await loadFees();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Erstellen der Mahngebühr';
  } finally {
    isCreatingReminder.value = false;
  }
}

async function onFeeCreated() {
  showCreateFeeDialog.value = false;
  await loadFees();
}

</script>

<template>
  <div>
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">Beiträge</h1>
        <p class="text-gray-600 mt-1">{{ total }} Beiträge gesamt</p>
      </div>
      <div class="flex gap-2">
        <button
          @click="showCreateFeeDialog = true"
          class="inline-flex items-center gap-2 px-4 py-2 border border-primary text-primary rounded-lg hover:bg-primary/5 transition-colors"
        >
          <Plus class="h-4 w-4" />
          Einzelner Beitrag
        </button>
        <button
          @click="showGenerateDialog = true"
          class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
        >
          <Calendar class="h-4 w-4" />
          Beiträge generieren
        </button>
      </div>
    </div>

    <!-- Selection Action Bar -->
    <div
      v-if="selectedCount > 0"
      class="flex items-center justify-between gap-4 mb-4 p-3 bg-blue-50 border border-blue-200 rounded-lg"
    >
      <div class="flex items-center gap-3">
        <span class="text-sm font-medium text-blue-800">
          {{ selectedCount }} ausgewählt
        </span>
        <button
          @click="clearSelection"
          class="text-sm text-blue-600 hover:text-blue-800 underline"
        >
          Auswahl aufheben
        </button>
      </div>
      <button
        v-if="selectedDeletableCount > 0"
        @click="showDeleteConfirm = true"
        class="inline-flex items-center gap-2 px-3 py-1.5 bg-red-600 text-white text-sm rounded-lg hover:bg-red-700 transition-colors"
      >
        <Trash2 class="h-4 w-4" />
        {{ selectedDeletableCount }} löschen
      </button>
      <span v-else class="text-sm text-gray-500">
        Nur unbezahlte Beiträge können gelöscht werden
      </span>
    </div>

    <!-- Filters -->
    <div class="flex flex-wrap gap-4 mb-6 p-4 bg-white rounded-xl border">
      <div class="flex items-center gap-2">
        <Filter class="h-4 w-4 text-gray-400" />
        <span class="text-sm font-medium text-gray-700">Filter:</span>
      </div>

      <!-- Search Input -->
      <div class="relative">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Mitgl.-Nr. oder Name..."
          @input="handleSearchInput"
          class="pl-9 pr-3 py-1.5 w-48 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
        />
      </div>
      
      <!-- Status Filter Buttons -->
      <div class="flex items-center gap-1 p-1 bg-gray-100 rounded-lg">
        <button
          @click="setStatusFilter('open')"
          :class="[
            'flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-md transition-colors',
            selectedStatus === 'open'
              ? 'bg-white text-amber-600 shadow-sm'
              : 'text-gray-600 hover:text-gray-900'
          ]"
          title="Offene Beiträge"
        >
          <Clock class="h-4 w-4" />
          <span class="hidden sm:inline">Offen</span>
        </button>
        <button
          @click="setStatusFilter('paid')"
          :class="[
            'flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-md transition-colors',
            selectedStatus === 'paid'
              ? 'bg-white text-green-600 shadow-sm'
              : 'text-gray-600 hover:text-gray-900'
          ]"
          title="Bezahlte Beiträge"
        >
          <CheckCircle class="h-4 w-4" />
          <span class="hidden sm:inline">Bezahlt</span>
        </button>
        <button
          @click="setStatusFilter('all')"
          :class="[
            'flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-md transition-colors',
            selectedStatus === 'all'
              ? 'bg-white text-gray-900 shadow-sm'
              : 'text-gray-600 hover:text-gray-900'
          ]"
          title="Alle Beiträge"
        >
          <span>Alle</span>
        </button>
      </div>

      <div class="flex items-center gap-1 p-1 bg-gray-100 rounded-lg">
        <button
          v-for="type in feeTypes"
          :key="type.value || 'all'"
          @click="setTypeFilter(type.value)"
          :class="[
            'px-3 py-1.5 text-sm font-medium rounded-md transition-colors whitespace-nowrap',
            selectedType === type.value
              ? 'bg-white text-primary shadow-sm'
              : 'text-gray-600 hover:text-gray-900'
          ]"
        >
          {{ type.label }}
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
      <button @click="loadFees" class="mt-2 text-sm text-red-700 underline">
        Erneut versuchen
      </button>
    </div>

    <!-- Fees table -->
    <div v-else class="bg-white rounded-xl border overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full">
          <thead class="bg-gray-50">
            <tr class="text-left text-sm text-gray-500">
              <th class="px-4 py-3 font-medium w-10">
                <input
                  type="checkbox"
                  :checked="allSelected"
                  :indeterminate="someSelected"
                  @change="toggleSelectAll"
                  class="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary cursor-pointer"
                />
              </th>
              <th class="px-4 py-3 font-medium">
                <button
                  @click="toggleSort('memberNumber')"
                  class="inline-flex items-center gap-1 hover:text-gray-700"
                  type="button"
                  :aria-sort="isSorted('memberNumber') ? (sortDir === 'asc' ? 'ascending' : 'descending') : 'none'"
                >
                  <span>Mitgl.-Nr.</span>
                  <component
                    :is="getSortIcon('memberNumber')"
                    :class="['h-4 w-4', isSorted('memberNumber') ? 'text-gray-700' : 'text-gray-400']"
                  />
                </button>
              </th>
              <th class="px-4 py-3 font-medium">
                <button
                  @click="toggleSort('childName')"
                  class="inline-flex items-center gap-1 hover:text-gray-700"
                  type="button"
                  :aria-sort="isSorted('childName') ? (sortDir === 'asc' ? 'ascending' : 'descending') : 'none'"
                >
                  <span>Kind</span>
                  <component
                    :is="getSortIcon('childName')"
                    :class="['h-4 w-4', isSorted('childName') ? 'text-gray-700' : 'text-gray-400']"
                  />
                </button>
              </th>
              <th class="px-4 py-3 font-medium">
                <button
                  @click="toggleSort('feeType')"
                  class="inline-flex items-center gap-1 hover:text-gray-700"
                  type="button"
                  :aria-sort="isSorted('feeType') ? (sortDir === 'asc' ? 'ascending' : 'descending') : 'none'"
                >
                  <span>Typ</span>
                  <component
                    :is="getSortIcon('feeType')"
                    :class="['h-4 w-4', isSorted('feeType') ? 'text-gray-700' : 'text-gray-400']"
                  />
                </button>
              </th>
              <th class="px-4 py-3 font-medium">
                <button
                  @click="toggleSort('period')"
                  class="inline-flex items-center gap-1 hover:text-gray-700"
                  type="button"
                  :aria-sort="isSorted('period') ? (sortDir === 'asc' ? 'ascending' : 'descending') : 'none'"
                >
                  <span>Zeitraum</span>
                  <component
                    :is="getSortIcon('period')"
                    :class="['h-4 w-4', isSorted('period') ? 'text-gray-700' : 'text-gray-400']"
                  />
                </button>
              </th>
              <th class="px-4 py-3 font-medium text-right">
                <button
                  @click="toggleSort('amount')"
                  class="inline-flex items-center gap-1 hover:text-gray-700 justify-end w-full"
                  type="button"
                  :aria-sort="isSorted('amount') ? (sortDir === 'asc' ? 'ascending' : 'descending') : 'none'"
                >
                  <span>Betrag</span>
                  <component
                    :is="getSortIcon('amount')"
                    :class="['h-4 w-4', isSorted('amount') ? 'text-gray-700' : 'text-gray-400']"
                  />
                </button>
              </th>
              <th class="px-4 py-3 font-medium">Fällig</th>
              <th class="px-4 py-3 font-medium">Status</th>
              <th class="px-4 py-3 font-medium w-10"></th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="fee in fees"
              :key="fee.id"
              :class="[
                'border-t transition-colors',
                isSelected(fee.id) ? 'bg-blue-50' : 'hover:bg-gray-50'
              ]"
            >
              <td class="px-4 py-3">
                <input
                  type="checkbox"
                  :checked="isSelected(fee.id)"
                  @change="toggleSelect(fee.id)"
                  class="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary cursor-pointer"
                />
              </td>
              <td class="px-4 py-3 text-gray-600 font-mono text-sm">
                {{ fee.child?.memberNumber }}
              </td>
              <td class="px-4 py-3">
                <button
                  @click="goToChild(fee.childId)"
                  class="font-medium text-primary hover:underline text-left"
                >
                  {{ fee.child?.firstName }} {{ fee.child?.lastName }}
                </button>
              </td>
              <td class="px-4 py-3">
                <span
                  :class="[
                    'inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium',
                    getFeeTypeColor(fee.feeType),
                  ]"
                >
                  {{ getFeeTypeName(fee.feeType) }}
                </span>
              </td>
              <td class="px-4 py-3 text-gray-600">
                {{ fee.month ? formatMonthName(fee.month) + ' ' : '' }}{{ fee.year }}
              </td>
              <td class="px-4 py-3 text-right font-medium">
                {{ formatCurrency(fee.amount) }}
              </td>
              <td class="px-4 py-3 text-gray-600">
                {{ formatDate(fee.dueDate) }}
              </td>
              <td class="px-4 py-3">
                <div class="flex items-center gap-1.5">
                  <component
                    :is="getStatusInfo(fee).icon"
                    :class="['h-4 w-4', getStatusInfo(fee).color]"
                  />
                  <span class="text-sm">{{ getStatusInfo(fee).label }}</span>
                </div>
              </td>
              <td class="px-4 py-3">
                <div class="flex items-center gap-1">
                  <button
                    v-if="canCreateReminder(fee)"
                    @click="openReminderDialog(fee)"
                    class="p-1.5 text-gray-400 hover:text-amber-600 hover:bg-amber-50 rounded transition-colors"
                    title="Mahngebühr erstellen"
                  >
                    <AlertCircle class="h-4 w-4" />
                  </button>
                  <button
                    v-if="!fee.isPaid"
                    @click="deleteSingleFee(fee)"
                    class="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded transition-colors"
                    title="Löschen"
                  >
                    <Trash2 class="h-4 w-4" />
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="fees.length === 0">
              <td colspan="9" class="px-4 py-8 text-center text-gray-500">
                Keine Beiträge gefunden
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

    <GenerateFeesDialog v-if="showGenerateDialog" @close="showGenerateDialog = false" @generated="loadFees" />

    <!-- Delete Confirmation Dialog -->
    <div
      v-if="showDeleteConfirm"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="showDeleteConfirm = false"
    >
      <div class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4 p-6">
        <div class="flex items-center gap-3 mb-6">
          <div class="p-2 bg-red-100 rounded-lg">
            <Trash2 class="h-6 w-6 text-red-600" />
          </div>
          <div>
            <h2 class="text-xl font-semibold">Beiträge löschen</h2>
            <p class="text-sm text-gray-600">Diese Aktion kann nicht rückgängig gemacht werden</p>
          </div>
        </div>

        <p class="text-gray-700 mb-6">
          Möchten Sie wirklich <strong>{{ selectedDeletableCount }} Beiträge</strong> löschen?
        </p>

        <div class="flex justify-end gap-3">
          <button
            @click="showDeleteConfirm = false"
            :disabled="isDeleting"
            class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors disabled:opacity-50"
          >
            Abbrechen
          </button>
          <button
            @click="deleteSelectedFees"
            :disabled="isDeleting"
            class="inline-flex items-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isDeleting" class="h-4 w-4 animate-spin" />
            <Trash2 v-else class="h-4 w-4" />
            Löschen
          </button>
        </div>
      </div>
    </div>

    <!-- Reminder Confirmation Dialog -->
    <div
      v-if="showReminderConfirm"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="showReminderConfirm = false"
    >
      <div class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4 p-6">
        <div class="flex items-center gap-3 mb-6">
          <div class="p-2 bg-amber-100 rounded-lg">
            <AlertCircle class="h-6 w-6 text-amber-600" />
          </div>
          <div>
            <h2 class="text-xl font-semibold">Mahngebühr erstellen</h2>
            <p class="text-sm text-gray-600">Eine Mahngebühr von 10,00 € wird erstellt</p>
          </div>
        </div>

        <div v-if="reminderTargetFee" class="mb-6 p-4 bg-gray-50 rounded-lg">
          <p class="text-sm text-gray-600">Für den überfälligen Beitrag:</p>
          <p class="font-medium mt-1">
            {{ getFeeTypeName(reminderTargetFee.feeType) }} - 
            {{ reminderTargetFee.child?.firstName }} {{ reminderTargetFee.child?.lastName }}
          </p>
          <p class="text-sm text-gray-500 mt-1">
            {{ formatCurrency(reminderTargetFee.amount) }} • Fällig: {{ formatDate(reminderTargetFee.dueDate) }}
          </p>
        </div>

        <div class="flex justify-end gap-3">
          <button
            @click="showReminderConfirm = false"
            :disabled="isCreatingReminder"
            class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors disabled:opacity-50"
          >
            Abbrechen
          </button>
          <button
            @click="createReminder"
            :disabled="isCreatingReminder"
            class="inline-flex items-center gap-2 px-4 py-2 bg-amber-600 text-white rounded-lg hover:bg-amber-700 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isCreatingReminder" class="h-4 w-4 animate-spin" />
            <AlertCircle v-else class="h-4 w-4" />
            Mahngebühr erstellen
          </button>
        </div>
      </div>
    </div>

    <CreateFeeDialog v-if="showCreateFeeDialog" @close="showCreateFeeDialog = false" @created="onFeeCreated" />
  </div>
</template>
