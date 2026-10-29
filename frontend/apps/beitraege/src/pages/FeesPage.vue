<script setup lang="ts">
import PageHeader from '@/components/ui/PageHeader.vue';
import AppSheet from '@/components/ui/AppSheet.vue';
import AppPagination from '@/components/ui/AppPagination.vue';
import EmptyState from '@/components/ui/EmptyState.vue';

import { ref, onMounted, computed, watch } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { api } from '@/api';
import type { FeeExpectation } from '@/api/types';
import {
  Info,
  Loader2,
  Plus,
  CheckCircle,
  Clock,
  AlertTriangle,
  Calendar,
  Trash2,
  AlertCircle,
} from 'lucide-vue-next';
import { formatCurrency, formatDate, todayISO } from '@/utils/format';
import { getFeePeriodLabel, getFeeTypeName } from '@/utils/fees';
import GenerateFeesDialog from '@/components/fees/GenerateFeesDialog.vue';
import CreateFeeDialog from '@/components/fees/CreateFeeDialog.vue';
import SearchInput from '@/components/SearchInput.vue';
import SortTh from '@/components/SortTh.vue';

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
  { value: '', label: 'Alle Typen' },
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
    return {
      icon: CheckCircle, color: 'badge-success', label: 'Bezahlt', bg: 'bg-success-soft',
    };
  }
  const isOverdue = fee.dueDate.slice(0, 10) < todayISO();
  if (isOverdue) {
    return {
      icon: AlertTriangle, color: 'badge-danger', label: 'Überfällig', bg: 'bg-destructive-soft',
    };
  }
  return {
      icon: Clock, color: 'badge-warning', label: 'Offen', bg: 'bg-warning-soft',
    };
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
  const isOverdue = fee.dueDate.slice(0, 10) < todayISO();
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
    <PageHeader title="Beiträge">
      <template #actions>
        <button type="button" @click="showCreateFeeDialog = true" class="btn btn-secondary">
          <Plus class="h-4 w-4" />
          Einzelner Beitrag
        </button>
        <button type="button" @click="showGenerateDialog = true" class="btn btn-primary">
          <Calendar class="h-4 w-4" />
          Beiträge generieren
        </button>
      </template>
    </PageHeader>
    <!-- Selection Action Bar -->
    <div v-if="selectedCount > 0" class="mb-4 flex flex-wrap items-center gap-2 rounded-xl bg-primary-soft px-3 py-2">
      <span class="mr-auto text-sm font-medium text-primary-text">{{ selectedCount }} ausgewählt</span>
      <button type="button" class="btn btn-ghost btn-sm" @click="clearSelection">Auswahl aufheben</button>
      <button
        v-if="selectedDeletableCount > 0"
        type="button"
        class="btn btn-danger btn-sm"
        @click="showDeleteConfirm = true">
        <Trash2 />
        {{ selectedDeletableCount }} löschen
      </button>
      <span v-else class="text-sm text-muted-foreground">Nur unbezahlte Beiträge können gelöscht werden</span>
    </div>
    <div class="mb-4 flex flex-col gap-3">
      <SearchInput v-model="searchQuery" placeholder="Mitgl.-Nr. oder Name" @input="handleSearchInput" />
      <div class="-mx-4 flex gap-2 overflow-x-auto px-4 sm:mx-0 sm:flex-wrap sm:px-0" role="group" aria-label="Filter">
        <span class="chip pointer-events-none border-transparent bg-transparent px-1 text-muted-foreground"
          aria-live="polite">{{ total }} Beiträge</span>
        <button type="button" class="chip" :aria-pressed="selectedStatus === 'open'" title="Offene Beiträge"
          @click="setStatusFilter('open')"><Clock class="h-4 w-4" />Offen</button>
        <button type="button" class="chip" :aria-pressed="selectedStatus === 'paid'" title="Bezahlte Beiträge"
          @click="setStatusFilter('paid')"><CheckCircle class="h-4 w-4" />Bezahlt</button>
        <button type="button" class="chip" :aria-pressed="selectedStatus === 'all'" title="Alle Beiträge"
          @click="setStatusFilter('all')">Alle</button>
        <span class="mx-1 w-px shrink-0 self-stretch bg-border" aria-hidden="true" />
        <button v-for="type in feeTypes" :key="type.value || 'all'" type="button" class="chip"
          :aria-pressed="selectedType === type.value" @click="setTypeFilter(type.value)">{{ type.label }}</button>
      </div>
    </div>
    <div class="mb-4 flex items-center gap-2 sm:hidden">
      <label for="fees-mobile-sort" class="label mb-0 shrink-0">Sortieren nach</label>
      <select
        id="fees-mobile-sort"
        :value="sortBy"
        class="input min-w-0 flex-1"
        @change="toggleSort(($event.target as HTMLSelectElement).value as FeeSortKey)">
        <option value="memberNumber">Mitgl.-Nr.</option>
        <option value="childName">Kind</option>
        <option value="feeType">Typ</option>
        <option value="period">Zeitraum</option>
        <option value="amount">Betrag</option>
      </select>
      <button
        type="button"
        class="btn btn-secondary btn-icon"
        aria-label="Sortierrichtung ändern"
        @click="toggleSort(sortBy)">
        <span aria-hidden="true">{{ sortDir === 'asc' ? '↑' : '↓' }}</span>
      </button>
    </div>
    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>
    <!-- Error state -->
    <div v-else-if="error" class="note note-danger">
      <Info />
      <div class="min-w-0 flex-1">
        <p class="text-destructive">{{ error }}</p>
        <button type="button" @click="loadFees" class="btn btn-secondary">Erneut versuchen</button>
      </div>
    </div>
    <!-- Fees table -->
    <div v-else class="card overflow-hidden">
      <EmptyState v-if="fees.length === 0" :icon="Calendar" text="Keine Beiträge gefunden" />
      <ul v-else class="divide-y sm:hidden">
        <li
          v-for="fee in fees"
          :key="fee.id"
          class="space-y-2 px-4 py-3"
          :class="isSelected(fee.id) && 'bg-primary-soft'">
          <div class="flex items-center gap-3">
            <input
              type="checkbox"
              :checked="isSelected(fee.id)"
              @change="toggleSelect(fee.id)"
              :aria-label="`${fee.child?.firstName} ${fee.child?.lastName} auswählen`"
              class="h-5 w-5 rounded" />
            <button type="button" class="min-w-0 flex-1 text-left" @click="goToChild(fee.childId)">
              <span class="block font-medium">{{ fee.child?.firstName }} {{ fee.child?.lastName }}</span>
              <span class="text-[13px] text-muted-foreground">{{ fee.child?.memberNumber }}</span>
            </button>
            <span class="text-right font-medium tabular-nums">{{ formatCurrency(fee.amount) }}</span>
          </div>
          <p class="text-[13px] text-muted-foreground">
            {{ getFeeTypeName(fee.feeType) }} · {{ getFeePeriodLabel(fee) }} · Fällig: {{ formatDate(fee.dueDate) }}
          </p>
          <div class="flex items-center justify-between gap-2">
            <span class="badge" :class="getStatusInfo(fee).color">
              <component :is="getStatusInfo(fee).icon" class="h-3.5 w-3.5" />
              {{ getStatusInfo(fee).label }}
            </span>
            <div class="flex items-center gap-1">
              <button
                type="button"
                v-if="canCreateReminder(fee)"
                @click="openReminderDialog(fee)"
                class="btn btn-secondary btn-icon btn-sm"
                aria-label="Mahngebühr erstellen"
                title="Mahngebühr erstellen">
                <AlertCircle class="h-4 w-4" />
              </button>
              <button
                type="button"
                v-if="!fee.isPaid"
                @click="deleteSingleFee(fee)"
                class="btn btn-danger-outline btn-icon btn-sm"
                aria-label="Löschen"
                title="Löschen">
                <Trash2 class="h-4 w-4" />
              </button>
            </div>
          </div>
        </li>
      </ul>
      <div v-if="fees.length" class="flex items-center gap-2 border-t px-4 py-2 text-sm sm:hidden">
        <input
          type="checkbox"
          :checked="allSelected"
          :indeterminate="someSelected"
          @change="toggleSelectAll"
          aria-label="Alle auswählen"
          class="h-4 w-4 rounded" />
        Alle auswählen
      </div>
      <div v-if="fees.length" class="hidden overflow-x-auto sm:block">
        <table class="table">
          <thead class="bg-muted">
            <tr class="text-left text-sm text-muted-foreground">
              <th class="px-4 py-3 font-medium w-10">
                <input
                  type="checkbox"
                  :checked="allSelected"
                  :indeterminate="someSelected"
                  @change="toggleSelectAll"
                  class="h-4 w-4 rounded border-border text-primary focus:ring-primary cursor-pointer" />
              </th>
              <SortTh
                label="Mitgl.-Nr."
                column="memberNumber"
                :sort-key="sortBy"
                :sort-dir="sortDir"
                @sort="toggleSort" />
              <SortTh label="Kind" column="childName" :sort-key="sortBy" :sort-dir="sortDir" @sort="toggleSort" />
              <SortTh label="Typ" column="feeType" :sort-key="sortBy" :sort-dir="sortDir" @sort="toggleSort" />
              <SortTh label="Zeitraum" column="period" :sort-key="sortBy" :sort-dir="sortDir" @sort="toggleSort" />
              <SortTh
                label="Betrag"
                column="amount"
                align="right"
                :sort-key="sortBy"
                :sort-dir="sortDir"
                @sort="toggleSort" />
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
              isSelected(fee.id) ? 'bg-primary-soft' : 'hover:bg-accent'
              ]">
              <td class="px-4 py-3">
                <input
                  type="checkbox"
                  :checked="isSelected(fee.id)"
                  @change="toggleSelect(fee.id)"
                  class="h-4 w-4 rounded border-border text-primary focus:ring-primary cursor-pointer" />
              </td>
              <td class="px-4 py-3 text-muted-foreground font-mono text-sm">{{ fee.child?.memberNumber }}</td>
              <td class="px-4 py-3">
                <button type="button" @click="goToChild(fee.childId)" class="text-left font-medium hover:text-primary-text hover:underline">
                  {{ fee.child?.firstName }} {{ fee.child?.lastName }}
                </button>
              </td>
              <td class="px-4 py-3">
                <span
                  :class="[
                  'badge',
                  fee.feeType === 'REMINDER' ? 'badge-danger' : '',
                  ]">
                  {{ getFeeTypeName(fee.feeType) }}
                </span>
              </td>
              <td class="px-4 py-3 text-muted-foreground">{{ getFeePeriodLabel(fee) }}</td>
              <td class="px-4 py-3 text-right font-medium tabular-nums">{{ formatCurrency(fee.amount) }}</td>
              <td class="px-4 py-3 text-muted-foreground">{{ formatDate(fee.dueDate) }}</td>
              <td class="px-4 py-3">
                <div class="badge" :class="getStatusInfo(fee).color">
                  <component :is="getStatusInfo(fee).icon" class="h-3.5 w-3.5" />
                  <span class="text-sm">{{ getStatusInfo(fee).label }}</span>
                </div>
              </td>
              <td class="px-4 py-3">
                <div class="flex items-center gap-1">
                  <button
                    type="button"
                    v-if="canCreateReminder(fee)"
                    @click="openReminderDialog(fee)"
                    class="btn btn-secondary btn-icon btn-sm"
                    aria-label="Mahngebühr erstellen"
                    title="Mahngebühr erstellen">
                    <AlertCircle class="h-4 w-4" />
                  </button>
                  <button
                    type="button"
                    v-if="!fee.isPaid"
                    @click="deleteSingleFee(fee)"
                    class="btn btn-ghost btn-icon btn-sm text-muted-foreground hover:text-destructive"
                    aria-label="Löschen"
                    title="Löschen">
                    <Trash2 class="h-4 w-4" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <AppPagination
        v-if="fees.length"
        v-model:page-size="pageSize"
        :current-page="currentPage"
        :total-pages="totalPages"
        :total="total"
        :offset="offset"
        :page-size-options="pageSizeOptions"
        @go="goToPage" />
    </div>
    <GenerateFeesDialog v-if="showGenerateDialog" @close="showGenerateDialog = false" @generated="loadFees" />
    <!-- Delete Confirmation Dialog -->
    <AppSheet v-if="showDeleteConfirm" title="Beiträge löschen" @close="showDeleteConfirm = false">
      <div class="note note-danger mb-4">
        <Info />
        <div class="min-w-0 flex-1">Diese Aktion kann nicht rückgängig gemacht werden</div>
      </div>
      <p class="text-foreground mb-6">
        Möchten Sie wirklich
        <strong>{{ selectedDeletableCount }} Beiträge</strong>
        löschen?
      </p>
      <template #footer>
        <button type="button" @click="showDeleteConfirm = false" :disabled="isDeleting" class="btn btn-secondary">
          Abbrechen
        </button>
        <button type="button" @click="deleteSelectedFees" :disabled="isDeleting" class="btn btn-danger">
          <Loader2 v-if="isDeleting" class="h-4 w-4 animate-spin" />
          <Trash2 v-else class="h-4 w-4" />
          Löschen
        </button>
      </template>
    </AppSheet>
    <!-- Reminder Confirmation Dialog -->
    <AppSheet v-if="showReminderConfirm" title="Mahngebühr erstellen" @close="showReminderConfirm = false">
      <div class="note note-warning mb-4">
        <Info />
        <div class="min-w-0 flex-1">Eine Mahngebühr von 10,00 € wird erstellt</div>
      </div>
      <div v-if="reminderTargetFee" class="mb-6 p-4 bg-muted rounded-lg">
        <p class="text-sm text-muted-foreground">Für den überfälligen Beitrag:</p>
        <p class="font-medium mt-1">
          {{ getFeeTypeName(reminderTargetFee.feeType) }} -
          {{ reminderTargetFee.child?.firstName }} {{ reminderTargetFee.child?.lastName }}
        </p>
        <p class="text-sm text-muted-foreground mt-1">
          {{ formatCurrency(reminderTargetFee.amount) }} • Fällig: {{ formatDate(reminderTargetFee.dueDate) }}
        </p>
      </div>
      <template #footer>
        <button
          type="button"
          @click="showReminderConfirm = false"
          :disabled="isCreatingReminder"
          class="btn btn-secondary">
          Abbrechen
        </button>
        <button type="button" @click="createReminder" :disabled="isCreatingReminder" class="btn btn-primary">
          <Loader2 v-if="isCreatingReminder" class="h-4 w-4 animate-spin" />
          <AlertCircle v-else class="h-4 w-4" />
          Mahngebühr erstellen
        </button>
      </template>
    </AppSheet>
    <CreateFeeDialog v-if="showCreateFeeDialog" @close="showCreateFeeDialog = false" @created="onFeeCreated" />
  </div>
</template>
