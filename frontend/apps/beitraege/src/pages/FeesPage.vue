<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import { useRouter } from 'vue-router';
import { api } from '@/api';
import type { FeeExpectation, GenerateFeeRequest, Child, CreateFeeRequest } from '@/api/types';
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
  User,
} from 'lucide-vue-next';

const router = useRouter();

const fees = ref<FeeExpectation[]>([]);
const total = ref(0);
const isLoading = ref(true);
const error = ref<string | null>(null);

const selectedYear = ref(new Date().getFullYear());
const selectedMonth = ref<number | null>(null);
const selectedType = ref<string>('');
const selectedStatus = ref<'open' | 'paid' | 'all'>('open');
const searchQuery = ref('');
const debouncedSearch = ref('');

const showGenerateDialog = ref(false);
const generateForm = ref<GenerateFeeRequest>({
  year: new Date().getFullYear(),
  month: new Date().getMonth() + 1,
});
const isGenerating = ref(false);
const generateResult = ref<{ created: number; skipped: number } | null>(null);

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
const createFeeForm = ref<CreateFeeRequest>({
  childId: '',
  feeType: 'FOOD',
  year: new Date().getFullYear(),
  month: new Date().getMonth() + 1,
});
const isCreatingFee = ref(false);
const createFeeError = ref<string | null>(null);
const childSearchQuery = ref('');
const childSearchResults = ref<Child[]>([]);
const selectedChild = ref<Child | null>(null);
const isSearchingChildren = ref(false);

const years = computed(() => {
  const currentYear = new Date().getFullYear();
  return [currentYear - 1, currentYear, currentYear + 1];
});

const months = [
  { value: 1, label: 'Januar' },
  { value: 2, label: 'Februar' },
  { value: 3, label: 'März' },
  { value: 4, label: 'April' },
  { value: 5, label: 'Mai' },
  { value: 6, label: 'Juni' },
  { value: 7, label: 'Juli' },
  { value: 8, label: 'August' },
  { value: 9, label: 'September' },
  { value: 10, label: 'Oktober' },
  { value: 11, label: 'November' },
  { value: 12, label: 'Dezember' },
];

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

// Debounce search
let searchTimeout: ReturnType<typeof setTimeout>;
watch(searchQuery, (newVal) => {
  clearTimeout(searchTimeout);
  searchTimeout = setTimeout(() => {
    debouncedSearch.value = newVal;
    loadFees();
  }, 300);
});

async function loadFees() {
  isLoading.value = true;
  error.value = null;
  selectedFeeIds.value = new Set(); // Clear selection on reload
  try {
    const response = await api.getFees({
      year: selectedYear.value,
      month: selectedMonth.value || undefined,
      feeType: selectedType.value || undefined,
      search: debouncedSearch.value || undefined,
      limit: 200,
    });
    // Filter by status on client side
    let data = response.data;
    if (selectedStatus.value === 'open') {
      data = data.filter(f => !f.isPaid);
    } else if (selectedStatus.value === 'paid') {
      data = data.filter(f => f.isPaid);
    }
    fees.value = data;
    total.value = response.total;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoading.value = false;
  }
}

function goToChild(childId: string) {
  router.push(`/kinder/${childId}`);
}

onMounted(loadFees);

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('de-DE');
}

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: 'EUR',
  }).format(amount);
}

function getFeeTypeName(type: string): string {
  switch (type) {
    case 'MEMBERSHIP':
      return 'Vereinsbeitrag';
    case 'FOOD':
      return 'Essensgeld';
    case 'CHILDCARE':
      return 'Platzgeld';
    case 'REMINDER':
      return 'Mahngebühr';
    default:
      return type;
  }
}

function getFeeTypeColor(type: string): string {
  switch (type) {
    case 'MEMBERSHIP':
      return 'bg-purple-100 text-purple-700';
    case 'FOOD':
      return 'bg-orange-100 text-orange-700';
    case 'CHILDCARE':
      return 'bg-blue-100 text-blue-700';
    case 'REMINDER':
      return 'bg-red-100 text-red-700';
    default:
      return 'bg-gray-100 text-gray-700';
  }
}

function getMonthName(month: number): string {
  return months.find(m => m.value === month)?.label || '';
}

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

async function handleGenerate() {
  isGenerating.value = true;
  generateResult.value = null;
  try {
    const result = await api.generateFees(generateForm.value);
    generateResult.value = result;
    await loadFees();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Generieren';
  } finally {
    isGenerating.value = false;
  }
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

// Single fee creation functions
let childSearchTimeout: ReturnType<typeof setTimeout>;

function openCreateFeeDialog() {
  showCreateFeeDialog.value = true;
  createFeeForm.value = {
    childId: '',
    feeType: 'FOOD',
    year: new Date().getFullYear(),
    month: new Date().getMonth() + 1,
  };
  createFeeError.value = null;
  childSearchQuery.value = '';
  childSearchResults.value = [];
  selectedChild.value = null;
}

function closeCreateFeeDialog() {
  showCreateFeeDialog.value = false;
  selectedChild.value = null;
  childSearchQuery.value = '';
  childSearchResults.value = [];
}

watch(childSearchQuery, (newVal) => {
  clearTimeout(childSearchTimeout);
  if (newVal.length < 2) {
    childSearchResults.value = [];
    return;
  }
  childSearchTimeout = setTimeout(async () => {
    isSearchingChildren.value = true;
    try {
      const response = await api.getChildren({ search: newVal, activeOnly: true, limit: 10 });
      childSearchResults.value = response.data;
    } catch {
      childSearchResults.value = [];
    } finally {
      isSearchingChildren.value = false;
    }
  }, 300);
});

function selectChild(child: Child) {
  selectedChild.value = child;
  createFeeForm.value.childId = child.id;
  childSearchQuery.value = '';
  childSearchResults.value = [];
}

function clearSelectedChild() {
  selectedChild.value = null;
  createFeeForm.value.childId = '';
}

const feeTypeRequiresMonth = computed(() => {
  return createFeeForm.value.feeType !== 'MEMBERSHIP';
});

async function handleCreateFee() {
  if (!selectedChild.value) {
    createFeeError.value = 'Bitte wählen Sie ein Kind aus';
    return;
  }
  
  isCreatingFee.value = true;
  createFeeError.value = null;
  
  try {
    const requestData: CreateFeeRequest = {
      childId: createFeeForm.value.childId,
      feeType: createFeeForm.value.feeType,
      year: createFeeForm.value.year,
    };
    
    // Only include month for non-membership fees
    if (feeTypeRequiresMonth.value && createFeeForm.value.month) {
      requestData.month = createFeeForm.value.month;
    }
    
    // Include optional fields if provided
    if (createFeeForm.value.amount) {
      requestData.amount = createFeeForm.value.amount;
    }
    if (createFeeForm.value.dueDate) {
      requestData.dueDate = createFeeForm.value.dueDate;
    }
    
    await api.createFee(requestData);
    closeCreateFeeDialog();
    await loadFees();
  } catch (e) {
    createFeeError.value = e instanceof Error ? e.message : 'Fehler beim Erstellen des Beitrags';
  } finally {
    isCreatingFee.value = false;
  }
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
          @click="openCreateFeeDialog"
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
          class="pl-9 pr-3 py-1.5 w-48 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
        />
      </div>
      
      <!-- Status Filter Buttons -->
      <div class="flex items-center gap-1 p-1 bg-gray-100 rounded-lg">
        <button
          @click="selectedStatus = 'open'; loadFees()"
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
          @click="selectedStatus = 'paid'; loadFees()"
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
          @click="selectedStatus = 'all'; loadFees()"
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

      <select
        v-model="selectedYear"
        @change="loadFees"
        class="px-3 py-1.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
      >
        <option v-for="year in years" :key="year" :value="year">{{ year }}</option>
      </select>
      <select
        v-model="selectedMonth"
        @change="loadFees"
        class="px-3 py-1.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
      >
        <option :value="null">Alle Monate</option>
        <option v-for="month in months" :key="month.value" :value="month.value">
          {{ month.label }}
        </option>
      </select>
      <select
        v-model="selectedType"
        @change="loadFees"
        class="px-3 py-1.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
      >
        <option v-for="type in feeTypes" :key="type.value" :value="type.value">
          {{ type.label }}
        </option>
      </select>
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
            <th class="px-4 py-3 font-medium">Mitgl.-Nr.</th>
            <th class="px-4 py-3 font-medium">Kind</th>
            <th class="px-4 py-3 font-medium">Typ</th>
            <th class="px-4 py-3 font-medium">Zeitraum</th>
            <th class="px-4 py-3 font-medium text-right">Betrag</th>
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
              {{ fee.month ? getMonthName(fee.month) + ' ' : '' }}{{ fee.year }}
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

    <!-- Generate Dialog -->
    <div
      v-if="showGenerateDialog"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="showGenerateDialog = false"
    >
      <div class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4 p-6">
        <div class="flex items-center gap-3 mb-6">
          <div class="p-2 bg-primary/10 rounded-lg">
            <Calendar class="h-6 w-6 text-primary" />
          </div>
          <div>
            <h2 class="text-xl font-semibold">Beiträge generieren</h2>
            <p class="text-sm text-gray-600">Erstellt fehlende Beiträge für alle aktiven Kinder</p>
          </div>
        </div>

        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Jahr</label>
            <select
              v-model="generateForm.year"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            >
              <option v-for="year in years" :key="year" :value="year">{{ year }}</option>
            </select>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Monat</label>
            <select
              v-model="generateForm.month"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            >
              <option :value="undefined">Nur Jahresbeitrag (Vereinsbeitrag)</option>
              <option v-for="month in months" :key="month.value" :value="month.value">
                {{ month.label }} (Essensgeld + ggf. Platzgeld)
              </option>
            </select>
          </div>

          <div v-if="generateResult" class="p-4 bg-green-50 border border-green-200 rounded-lg">
            <p class="text-green-700 font-medium">Erfolgreich generiert!</p>
            <p class="text-sm text-green-600 mt-1">
              {{ generateResult.created }} Beiträge erstellt, {{ generateResult.skipped }} übersprungen
            </p>
          </div>

          <div class="flex justify-end gap-3 pt-4">
            <button
              @click="showGenerateDialog = false"
              class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
            >
              Schließen
            </button>
            <button
              @click="handleGenerate"
              :disabled="isGenerating"
              class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Loader2 v-if="isGenerating" class="h-4 w-4 animate-spin" />
              <Plus v-else class="h-4 w-4" />
              Generieren
            </button>
          </div>
        </div>
      </div>
    </div>

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

    <!-- Create Single Fee Dialog -->
    <div
      v-if="showCreateFeeDialog"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="closeCreateFeeDialog"
    >
      <div class="bg-white rounded-xl shadow-xl w-full max-w-lg mx-4 p-6">
        <div class="flex items-center gap-3 mb-6">
          <div class="p-2 bg-primary/10 rounded-lg">
            <Plus class="h-6 w-6 text-primary" />
          </div>
          <div>
            <h2 class="text-xl font-semibold">Einzelnen Beitrag erstellen</h2>
            <p class="text-sm text-gray-600">Erstellt einen Beitrag für ein bestimmtes Kind</p>
          </div>
        </div>

        <div class="space-y-4">
          <!-- Child Selection -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Kind *</label>
            
            <!-- Selected Child Display -->
            <div v-if="selectedChild" class="flex items-center justify-between p-3 bg-gray-50 border border-gray-200 rounded-lg">
              <div class="flex items-center gap-3">
                <div class="p-2 bg-primary/10 rounded-full">
                  <User class="h-4 w-4 text-primary" />
                </div>
                <div>
                  <p class="font-medium">{{ selectedChild.firstName }} {{ selectedChild.lastName }}</p>
                  <p class="text-sm text-gray-500">Mitgl.-Nr.: {{ selectedChild.memberNumber }}</p>
                </div>
              </div>
              <button
                @click="clearSelectedChild"
                class="text-gray-400 hover:text-gray-600"
              >
                <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            
            <!-- Child Search -->
            <div v-else class="relative">
              <div class="relative">
                <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
                <input
                  v-model="childSearchQuery"
                  type="text"
                  placeholder="Name oder Mitgliedsnummer eingeben..."
                  class="w-full pl-10 pr-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
                />
                <Loader2 v-if="isSearchingChildren" class="absolute right-3 top-1/2 -translate-y-1/2 h-4 w-4 animate-spin text-gray-400" />
              </div>
              
              <!-- Search Results Dropdown -->
              <div
                v-if="childSearchResults.length > 0"
                class="absolute z-10 w-full mt-1 bg-white border border-gray-200 rounded-lg shadow-lg max-h-60 overflow-auto"
              >
                <button
                  v-for="child in childSearchResults"
                  :key="child.id"
                  @click="selectChild(child)"
                  class="w-full flex items-center gap-3 px-4 py-3 hover:bg-gray-50 text-left border-b last:border-b-0"
                >
                  <div class="p-1.5 bg-gray-100 rounded-full">
                    <User class="h-4 w-4 text-gray-600" />
                  </div>
                  <div>
                    <p class="font-medium">{{ child.firstName }} {{ child.lastName }}</p>
                    <p class="text-sm text-gray-500">{{ child.memberNumber }}</p>
                  </div>
                </button>
              </div>
            </div>
          </div>

          <!-- Fee Type -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Beitragsart *</label>
            <select
              v-model="createFeeForm.feeType"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            >
              <option value="FOOD">Essensgeld</option>
              <option value="CHILDCARE">Platzgeld</option>
              <option value="MEMBERSHIP">Vereinsbeitrag</option>
              <option value="REMINDER">Mahngebühr</option>
            </select>
          </div>

          <!-- Year -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Jahr *</label>
            <select
              v-model="createFeeForm.year"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            >
              <option v-for="year in years" :key="year" :value="year">{{ year }}</option>
            </select>
          </div>

          <!-- Month (conditional) -->
          <div v-if="feeTypeRequiresMonth">
            <label class="block text-sm font-medium text-gray-700 mb-1">Monat *</label>
            <select
              v-model="createFeeForm.month"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            >
              <option v-for="month in months" :key="month.value" :value="month.value">
                {{ month.label }}
              </option>
            </select>
          </div>

          <!-- Amount (optional) -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">
              Betrag (optional)
              <span class="font-normal text-gray-500">- wird automatisch berechnet wenn leer</span>
            </label>
            <div class="relative">
              <input
                v-model.number="createFeeForm.amount"
                type="number"
                step="0.01"
                min="0"
                placeholder="z.B. 45.40"
                class="w-full px-3 py-2 pr-10 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
              <span class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400">EUR</span>
            </div>
          </div>

          <!-- Due Date (optional) -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">
              Fälligkeitsdatum (optional)
              <span class="font-normal text-gray-500">- wird automatisch gesetzt wenn leer</span>
            </label>
            <input
              v-model="createFeeForm.dueDate"
              type="date"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>

          <!-- Error Display -->
          <div v-if="createFeeError" class="p-3 bg-red-50 border border-red-200 rounded-lg">
            <p class="text-sm text-red-600">{{ createFeeError }}</p>
          </div>

          <!-- Actions -->
          <div class="flex justify-end gap-3 pt-4">
            <button
              @click="closeCreateFeeDialog"
              class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
            >
              Abbrechen
            </button>
            <button
              @click="handleCreateFee"
              :disabled="isCreatingFee || !selectedChild"
              class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Loader2 v-if="isCreatingFee" class="h-4 w-4 animate-spin" />
              <Plus v-else class="h-4 w-4" />
              Beitrag erstellen
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
