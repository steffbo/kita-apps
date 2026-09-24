<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue';
import { useRoute } from 'vue-router';
import { api } from '@/api';
import type { ImportBatch, BankTransaction, TransactionWarning, RescanResult } from '@/api/types';
import BankingSyncCard from '@/components/BankingSyncCard.vue';
import ImportErrorList from '@/components/ImportErrorList.vue';
import ImportUploadModal from '@/components/import/ImportUploadModal.vue';
import ImportHistoryModal from '@/components/import/ImportHistoryModal.vue';
import ImportBlacklistModal from '@/components/import/ImportBlacklistModal.vue';
import ManualMatchModal from '@/components/import/ManualMatchModal.vue';
import {
  Upload,
  Loader2,
  CheckCircle,
  XCircle,
  AlertTriangle,
  History,
  ChevronDown,
  ChevronUp,
  ChevronLeft,
  ChevronRight,
  Ban,
  Trash2,
  ShieldOff,
  EyeOff,
  Clock,
  Euro,
  LinkIcon,
  Unlink,
  RefreshCw,
  Search,
  ArrowUp,
  ArrowDown,
  ArrowUpDown,
} from 'lucide-vue-next';
import { formatCurrency, formatDate, formatDateTime, formatMonthName } from '@/utils/format';
import { getFeeTypeColor, getFeeTypeName } from '@/utils/fees';

type StatusFilter = 'offen' | 'warnungen' | 'zugeordnet' | 'alle';
type SortField = 'date' | 'payer' | 'description' | 'amount';
type SortDirection = 'asc' | 'desc';

interface TxRow {
  key: string;
  tx: BankTransaction;
  matched: boolean;
  warnings: TransactionWarning[];
}

function isPartiallyAllocated(row: TxRow): boolean {
  return (row.tx.matchedAmount ?? 0) > 0.005 && row.tx.amount - (row.tx.matchedAmount ?? 0) > 0.005;
}

function getTxRemaining(tx: BankTransaction): number {
  const remaining = tx.amount - (tx.matchedAmount ?? 0);
  return remaining > 0 ? remaining : 0;
}

const route = useRoute();
const activeFilter = ref<StatusFilter>('offen');

// Modals (each loads and resets its own state)
const showUploadModal = ref(false);
const showHistoryModal = ref(false);
const showBlacklistModal = ref(false);
const manualMatchTransaction = ref<BankTransaction | null>(null);
// Batch whose errors the history modal expands on open
const expandedBatchId = ref<string | null>(null);

// Page-level error banner
const uploadError = ref<string | null>(null);

// Most recent import (usually the automated banking sync), to surface its errors.
const latestImportBatch = ref<ImportBatch | null>(null);

// Transactions state (unified list)
const unmatchedTransactions = ref<BankTransaction[]>([]);
const unmatchedTotal = ref(0);
const matchedTransactions = ref<BankTransaction[]>([]);
const matchedTotal = ref(0);
const warnings = ref<TransactionWarning[]>([]);
const warningsTotal = ref(0);
const isLoadingTransactions = ref(true);

// Warning actions state
const isResolvingWarning = ref<string | null>(null);
const dismissWarningId = ref<string | null>(null);
const dismissNote = ref('');

// Expanded warning detail rows
const expandedWarnings = ref<Set<string>>(new Set());

// Rescan state
const isRescanning = ref(false);
const rescanResult = ref<RescanResult | null>(null);

// Dismiss/hide/unmatch confirm state
const isDismissing = ref<string | null>(null);
const dismissConfirmId = ref<string | null>(null);
const isHiding = ref<string | null>(null);
const hideConfirmId = ref<string | null>(null);
const unmatchConfirmId = ref<string | null>(null);
const deleteConfirmId = ref<string | null>(null);
const isUnmatching = ref<string | null>(null);
const isDeletingMatched = ref<string | null>(null);

// Search, sort and pagination (client-side over the loaded sets)
const transactionSearch = ref('');
const sortField = ref<SortField>('date');
const sortDirection = ref<SortDirection>('desc');
const page = ref(1);
const pageSize = 50;

function toggleSort(field: SortField): void {
  if (sortField.value === field) {
    sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc';
  } else {
    sortField.value = field;
    sortDirection.value = field === 'date' ? 'desc' : 'asc';
  }
}

watch([activeFilter, transactionSearch], () => {
  page.value = 1;
});

// Unified rows: merge matched, unmatched and open warnings by transaction id
const transactionRows = computed<TxRow[]>(() => {
  const map = new Map<string, TxRow>();
  for (const tx of unmatchedTransactions.value) {
    map.set(tx.id, { key: tx.id, tx, matched: false, warnings: [] });
  }
  for (const tx of matchedTransactions.value) {
    const existing = map.get(tx.id);
    if (existing) {
      existing.matched = true;
    } else {
      map.set(tx.id, { key: tx.id, tx, matched: true, warnings: [] });
    }
  }
  for (const warning of warnings.value) {
    let row = warning.transactionId ? map.get(warning.transactionId) : undefined;
    if (!row && warning.transaction && !map.has(warning.transaction.id)) {
      const tx = warning.transaction;
      row = { key: tx.id || warning.id, tx, matched: false, warnings: [] };
      map.set(row.key, row);
    }
    if (row) {
      row.warnings.push(warning);
    }
  }
  return [...map.values()];
});

const offenCount = computed(() => transactionRows.value.filter(r => !r.matched).length);
const warnungenCount = computed(() => transactionRows.value.filter(r => r.warnings.length > 0).length);
const zugeordnetCount = computed(() => matchedTotal.value);

const filteredRows = computed<TxRow[]>(() => {
  let rows = transactionRows.value;
  if (activeFilter.value === 'offen') {
    rows = rows.filter(r => !r.matched);
  } else if (activeFilter.value === 'warnungen') {
    rows = rows.filter(r => r.warnings.length > 0);
  } else if (activeFilter.value === 'zugeordnet') {
    rows = rows.filter(r => r.matched);
  }

  const search = transactionSearch.value.trim().toLowerCase();
  if (search) {
    rows = rows.filter(
      r =>
        (r.tx.payerName || '').toLowerCase().includes(search) ||
        (r.tx.description || '').toLowerCase().includes(search) ||
        (r.tx.payerIban || '').toLowerCase().includes(search)
    );
  }
  return rows;
});

const sortedRows = computed<TxRow[]>(() => {
  const dir = sortDirection.value === 'asc' ? 1 : -1;
  return [...filteredRows.value].sort((a, b) => {
    switch (sortField.value) {
      case 'payer':
        return dir * (a.tx.payerName || '').localeCompare(b.tx.payerName || '');
      case 'description':
        return dir * (a.tx.description || '').localeCompare(b.tx.description || '');
      case 'amount':
        return dir * (a.tx.amount - b.tx.amount);
      default:
        return dir * (new Date(a.tx.bookingDate).getTime() - new Date(b.tx.bookingDate).getTime());
    }
  });
});

const totalPages = computed(() => Math.max(1, Math.ceil(sortedRows.value.length / pageSize)));

const visiblePages = computed<number[]>(() => {
  const total = totalPages.value;
  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1);
  }
  const start = Math.max(1, Math.min(page.value - 3, total - 6));
  return Array.from({ length: 7 }, (_, i) => start + i);
});

const pagedRows = computed<TxRow[]>(() =>
  sortedRows.value.slice((page.value - 1) * pageSize, page.value * pageSize)
);

function goToPage(target: number): void {
  page.value = Math.min(Math.max(1, target), totalPages.value);
}

function toggleWarnings(key: string): void {
  if (expandedWarnings.value.has(key)) {
    expandedWarnings.value.delete(key);
  } else {
    expandedWarnings.value.add(key);
  }
}

async function loadTransactions(): Promise<void> {
  isLoadingTransactions.value = true;
  try {
    const [unmatchedRes, matchedRes, warningsRes] = await Promise.all([
      api.getUnmatchedTransactions({ page: 1, perPage: 500 }),
      api.getMatchedTransactions({ page: 1, perPage: 1000 }),
      api.getWarnings(1, 200),
    ]);
    unmatchedTransactions.value = unmatchedRes.data;
    unmatchedTotal.value = unmatchedRes.total;
    matchedTransactions.value = matchedRes.data;
    matchedTotal.value = matchedRes.total;
    warnings.value = warningsRes.data;
    warningsTotal.value = warningsRes.total;
  } catch (error) {
    console.error('Failed to load transactions:', error);
    uploadError.value = error instanceof Error ? error.message : 'Transaktionen konnten nicht geladen werden';
  } finally {
    isLoadingTransactions.value = false;
  }
}

async function loadLatestImport(): Promise<void> {
  try {
    const response = await api.getImportHistory(1, 1);
    latestImportBatch.value = response.data[0] ?? null;
  } catch (error) {
    console.error('Failed to load latest import:', error);
  }
}

function openLatestImportErrors(): void {
  if (latestImportBatch.value) {
    expandedBatchId.value = latestImportBatch.value.id;
  }
  showHistoryModal.value = true;
}

function openHistoryModal(): void {
  expandedBatchId.value = null;
  showHistoryModal.value = true;
}

function openBlacklistModal(): void {
  showBlacklistModal.value = true;
}

async function rescanTransactions(): Promise<void> {
  isRescanning.value = true;
  rescanResult.value = null;
  try {
    const result = await api.rescanTransactions();
    rescanResult.value = result;
    await loadTransactions();
  } catch (error) {
    console.error('Failed to rescan transactions:', error);
    uploadError.value = error instanceof Error ? error.message : 'Erneutes Zuordnen fehlgeschlagen';
  } finally {
    isRescanning.value = false;
  }
}

function openManualMatch(transaction: BankTransaction): void {
  manualMatchTransaction.value = transaction;
}

async function onManualMatched(): Promise<void> {
  manualMatchTransaction.value = null;
  await loadTransactions();
}

function handleKeydown(e: KeyboardEvent): void {
  if (e.key !== 'Escape') return;
  if (manualMatchTransaction.value) {
    manualMatchTransaction.value = null;
  } else if (showUploadModal.value) {
    showUploadModal.value = false;
  } else if (showHistoryModal.value) {
    showHistoryModal.value = false;
  } else if (showBlacklistModal.value) {
    showBlacklistModal.value = false;
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown);

  // Map legacy tab query params to the new filter/modal structure
  const tabParam = route.query.tab as string | undefined;
  if (tabParam === 'upload') {
    showUploadModal.value = true;
  } else if (tabParam === 'history') {
    openHistoryModal();
  } else if (tabParam === 'blacklist') {
    openBlacklistModal();
  } else if (tabParam === 'warnings') {
    activeFilter.value = 'warnungen';
  } else if (tabParam === 'matched') {
    activeFilter.value = 'zugeordnet';
  } else if (tabParam === 'unmatched') {
    activeFilter.value = 'offen';
  }

  loadTransactions();
  loadLatestImport();
});

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown);
});

function showWarningDismiss(warningId: string): void {
  dismissWarningId.value = warningId;
  dismissNote.value = '';
}

function cancelWarningDismiss(): void {
  dismissWarningId.value = null;
  dismissNote.value = '';
}

async function dismissWarning(warning: TransactionWarning): Promise<void> {
  isResolvingWarning.value = warning.id;
  try {
    await api.dismissWarning(warning.id, dismissNote.value);
    warnings.value = warnings.value.filter(w => w.id !== warning.id);
    warningsTotal.value = Math.max(0, warningsTotal.value - 1);
    dismissWarningId.value = null;
    dismissNote.value = '';
  } catch (error) {
    console.error('Failed to dismiss warning:', error);
    uploadError.value = error instanceof Error ? error.message : 'Warnung konnte nicht verworfen werden';
  } finally {
    isResolvingWarning.value = null;
  }
}

async function resolveLateFee(warning: TransactionWarning): Promise<void> {
  isResolvingWarning.value = warning.id;
  try {
    await api.resolveLateFee(warning.id);
    warnings.value = warnings.value.filter(w => w.id !== warning.id);
    warningsTotal.value = Math.max(0, warningsTotal.value - 1);
  } catch (error) {
    console.error('Failed to resolve late fee:', error);
    uploadError.value = error instanceof Error ? error.message : 'Mahngebuhr konnte nicht erstellt werden';
  } finally {
    isResolvingWarning.value = null;
  }
}

function showDismissConfirm(transactionId: string): void {
  dismissConfirmId.value = transactionId;
  hideConfirmId.value = null;
}

function cancelDismiss(): void {
  dismissConfirmId.value = null;
}

function showHideConfirm(transactionId: string): void {
  hideConfirmId.value = transactionId;
  dismissConfirmId.value = null;
}

function cancelHide(): void {
  hideConfirmId.value = null;
}

function showUnmatchConfirm(transactionId: string): void {
  unmatchConfirmId.value = transactionId;
  deleteConfirmId.value = null;
}

function showDeleteConfirm(transactionId: string): void {
  deleteConfirmId.value = transactionId;
  unmatchConfirmId.value = null;
}

function cancelMatchAction(): void {
  unmatchConfirmId.value = null;
  deleteConfirmId.value = null;
}

async function dismissTransaction(transaction: BankTransaction): Promise<void> {
  isDismissing.value = transaction.id;
  dismissConfirmId.value = null;
  try {
    await api.dismissTransaction(transaction.id);
    await loadTransactions();
  } catch (error) {
    console.error('Failed to dismiss transaction:', error);
    uploadError.value = error instanceof Error ? error.message : 'Ignorieren fehlgeschlagen';
  } finally {
    isDismissing.value = null;
  }
}

async function hideTransaction(transaction: BankTransaction): Promise<void> {
  isHiding.value = transaction.id;
  hideConfirmId.value = null;
  try {
    await api.hideTransaction(transaction.id);
    await loadTransactions();
  } catch (error) {
    console.error('Failed to hide transaction:', error);
    uploadError.value = error instanceof Error ? error.message : 'Ausblenden fehlgeschlagen';
  } finally {
    isHiding.value = null;
  }
}

async function unmatchTransaction(transaction: BankTransaction, deleteTransaction = false): Promise<void> {
  if (deleteTransaction) {
    isDeletingMatched.value = transaction.id;
  } else {
    isUnmatching.value = transaction.id;
  }
  cancelMatchAction();
  try {
    await api.unmatchTransaction(transaction.id, { deleteTransaction });
    await loadTransactions();
  } catch (error) {
    console.error('Failed to unmatch transaction:', error);
    uploadError.value = error instanceof Error ? error.message : 'Zuordnung konnte nicht aufgehoben werden';
  } finally {
    if (deleteTransaction) {
      isDeletingMatched.value = null;
    } else {
      isUnmatching.value = null;
    }
  }
}

async function refreshAfterSync(): Promise<void> {
  loadLatestImport();
  await loadTransactions();
}

function getWarningTypeLabel(type: string): string {
  switch (type) {
    case 'AMOUNT_MISMATCH':
      return 'Betrag weicht ab';
    case 'DUPLICATE_PAYMENT':
      return 'Doppelte Zahlung';
    case 'UNKNOWN_IBAN':
      return 'Unbekannte IBAN';
    case 'LATE_PAYMENT':
      return 'Verspätete Zahlung';
    case 'OVERPAYMENT':
      return 'Überzahlung';
    default:
      return type;
  }
}

function getWarningTypeColor(type: string): string {
  switch (type) {
    case 'AMOUNT_MISMATCH':
      return 'bg-amber-100 text-amber-700';
    case 'DUPLICATE_PAYMENT':
      return 'bg-red-100 text-red-700';
    case 'UNKNOWN_IBAN':
      return 'bg-gray-100 text-gray-700';
    case 'LATE_PAYMENT':
      return 'bg-orange-100 text-orange-700';
    case 'OVERPAYMENT':
      return 'bg-purple-100 text-purple-700';
    default:
      return 'bg-gray-100 text-gray-700';
  }
}
</script>

<template>
  <div>
    <!-- Header -->
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-gray-900">Bankabgleich</h1>
      <p class="text-gray-600 mt-1">
        Banktransaktionen abrufen und Zahlungen zuordnen
      </p>
    </div>

    <!-- Banking Sync -->
    <BankingSyncCard @sync-finished="refreshAfterSync" />

    <!-- Rescan Result -->
    <div
      v-if="rescanResult"
      class="mb-4 p-4 bg-blue-50 border border-blue-200 rounded-lg flex items-start gap-3"
    >
      <CheckCircle class="h-5 w-5 text-blue-500 flex-shrink-0 mt-0.5" />
      <div>
        <p class="text-blue-700 font-medium">Erneute Zuordnung abgeschlossen</p>
        <p class="text-sm text-blue-600">
          {{ rescanResult.scanned }} Transaktionen gescannt<span v-if="rescanResult.autoMatched > 0">, {{ rescanResult.autoMatched }} automatisch zugeordnet</span><span v-if="rescanResult.newMatches > 0">, {{ rescanResult.newMatches }} Vorschläge zur Überprüfung</span>
        </p>
      </div>
      <button
        @click="rescanResult = null"
        class="ml-auto text-blue-500 hover:text-blue-700"
      >
        <XCircle class="h-4 w-4" />
      </button>
    </div>
    <ImportErrorList
      v-if="rescanResult?.errors?.length"
      class="mb-4"
      title="Fehler bei der erneuten Zuordnung"
      :errors="rescanResult.errors"
    />

    <!-- Errors of the latest (usually automated) import -->
    <div
      v-if="latestImportBatch && latestImportBatch.errorCount > 0"
      class="mb-4 p-4 bg-red-50 border border-red-200 rounded-lg flex items-start gap-3"
      role="alert"
    >
      <AlertTriangle class="h-5 w-5 text-red-500 flex-shrink-0 mt-0.5" />
      <div class="flex-1">
        <p class="text-red-700 font-medium">
          Letzter Import vom {{ formatDateTime(latestImportBatch.importedAt) }}:
          {{ latestImportBatch.errorCount }}
          {{ latestImportBatch.errorCount === 1 ? 'Buchung' : 'Buchungen' }} mit Fehlern
        </p>
        <p class="text-sm text-red-600">{{ latestImportBatch.importedByEmail }} · {{ latestImportBatch.fileName }}</p>
      </div>
      <button @click="openLatestImportErrors" class="text-sm text-red-700 hover:text-red-900 underline">
        Details
      </button>
    </div>

    <!-- Toolbar -->
    <div class="bg-white rounded-xl border p-4 mb-6 space-y-3">
      <div class="flex flex-col lg:flex-row lg:items-center gap-3">
        <div class="flex flex-wrap items-center gap-2">
          <button
            @click="activeFilter = 'offen'"
            :class="[
              'inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm font-medium border transition-colors',
              activeFilter === 'offen'
                ? 'bg-primary text-white border-primary'
                : 'bg-white text-gray-600 border-gray-300 hover:bg-gray-50',
            ]"
          >
            Offen
            <span
              v-if="offenCount > 0"
              :class="[
                'px-1.5 py-0.5 text-xs rounded-full',
                activeFilter === 'offen' ? 'bg-white/20 text-white' : 'bg-amber-100 text-amber-700',
              ]"
            >
              {{ offenCount }}
            </span>
          </button>
          <button
            @click="activeFilter = 'warnungen'"
            :class="[
              'inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm font-medium border transition-colors',
              activeFilter === 'warnungen'
                ? 'bg-primary text-white border-primary'
                : 'bg-white text-gray-600 border-gray-300 hover:bg-gray-50',
            ]"
          >
            Warnungen
            <span
              v-if="warnungenCount > 0"
              :class="[
                'px-1.5 py-0.5 text-xs rounded-full',
                activeFilter === 'warnungen' ? 'bg-white/20 text-white' : 'bg-orange-100 text-orange-700',
              ]"
            >
              {{ warnungenCount }}
            </span>
          </button>
          <button
            @click="activeFilter = 'zugeordnet'"
            :class="[
              'inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm font-medium border transition-colors',
              activeFilter === 'zugeordnet'
                ? 'bg-primary text-white border-primary'
                : 'bg-white text-gray-600 border-gray-300 hover:bg-gray-50',
            ]"
          >
            Zugeordnet
            <span
              v-if="zugeordnetCount > 0"
              :class="[
                'px-1.5 py-0.5 text-xs rounded-full',
                activeFilter === 'zugeordnet' ? 'bg-white/20 text-white' : 'bg-green-100 text-green-700',
              ]"
            >
              {{ zugeordnetCount }}
            </span>
          </button>
          <button
            @click="activeFilter = 'alle'"
            :class="[
              'px-3 py-1.5 rounded-full text-sm font-medium border transition-colors',
              activeFilter === 'alle'
                ? 'bg-primary text-white border-primary'
                : 'bg-white text-gray-600 border-gray-300 hover:bg-gray-50',
            ]"
          >
            Alle
          </button>
        </div>

        <div class="flex-1 min-w-[220px]">
          <div class="relative max-w-md lg:ml-auto">
            <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
            <input
              v-model="transactionSearch"
              type="text"
              placeholder="Suche nach Zahler oder Beschreibung..."
              class="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
        </div>

        <div class="flex items-center gap-2">
          <button
            @click="rescanTransactions"
            :disabled="isRescanning || offenCount === 0"
            class="inline-flex items-center gap-1 px-3 py-2 text-sm border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            title="Automatische Zuordnung erneut ausführen"
          >
            <Loader2 v-if="isRescanning" class="h-4 w-4 animate-spin" />
            <RefreshCw v-else class="h-4 w-4" />
            Erneut zuordnen
          </button>
          <button
            @click="showUploadModal = true"
            class="inline-flex items-center gap-1 px-3 py-2 text-sm bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
          >
            <Upload class="h-4 w-4" />
            CSV hochladen
          </button>
        </div>
      </div>

      <div class="flex flex-wrap items-center justify-between gap-3 pt-1 border-t text-sm">
        <p class="text-gray-500 pt-2">
          {{ sortedRows.length }} Transaktionen
          <span v-if="filteredRows.length !== transactionRows.length"> (gefiltert von {{ transactionRows.length }})</span>
        </p>
        <div class="flex items-center gap-4 pt-2">
          <button
            @click="openHistoryModal"
            class="inline-flex items-center gap-1 text-gray-600 hover:text-gray-900 underline"
          >
            <History class="h-4 w-4" />
            Import-Historie
          </button>
          <button
            @click="openBlacklistModal"
            class="inline-flex items-center gap-1 text-gray-600 hover:text-gray-900 underline"
          >
            <ShieldOff class="h-4 w-4" />
            Blacklist
          </button>
        </div>
      </div>
    </div>

    <!-- Error Banner -->
    <div
      v-if="uploadError"
      class="mb-4 p-4 bg-red-50 border border-red-200 rounded-lg flex items-start gap-3"
    >
      <XCircle class="h-5 w-5 text-red-500 flex-shrink-0 mt-0.5" />
      <div>
        <p class="text-red-700 font-medium">Fehler</p>
        <p class="text-sm text-red-600">{{ uploadError }}</p>
      </div>
      <button @click="uploadError = null" class="ml-auto text-red-400 hover:text-red-600">
        <XCircle class="h-4 w-4" />
      </button>
    </div>

    <!-- Unified Transaction List -->
    <div v-if="isLoadingTransactions" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <div v-else-if="pagedRows.length === 0" class="bg-white rounded-xl border text-center py-12">
      <component
        :is="transactionSearch.trim() ? Search : CheckCircle"
        :class="['h-12 w-12 mx-auto mb-4', transactionSearch.trim() ? 'text-gray-300' : 'text-green-300']"
      />
      <p class="text-gray-600">
        {{ transactionSearch.trim() ? 'Keine Transaktionen gefunden' : 'Keine offenen Transaktionen' }}
      </p>
      <p v-if="transactionSearch.trim()" class="text-sm text-gray-500 mt-1">Versuche einen anderen Suchbegriff</p>
      <p v-else-if="activeFilter === 'warnungen'" class="text-sm text-gray-500 mt-1">Alle Zahlungen wurden korrekt verarbeitet</p>
      <p v-else-if="activeFilter === 'zugeordnet'" class="text-sm text-gray-500 mt-1">Noch keine zugeordneten Transaktionen</p>
    </div>

    <div v-else class="bg-white rounded-xl border overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full">
          <thead class="bg-gray-50">
            <tr class="text-left text-sm text-gray-500">
              <th
                class="px-4 py-3 font-medium cursor-pointer hover:bg-gray-100 select-none"
                @click="toggleSort('date')"
              >
                <div class="flex items-center gap-1">
                  Datum
                  <ArrowUp v-if="sortField === 'date' && sortDirection === 'asc'" class="h-4 w-4" />
                  <ArrowDown v-else-if="sortField === 'date' && sortDirection === 'desc'" class="h-4 w-4" />
                  <ArrowUpDown v-else class="h-4 w-4 text-gray-400" />
                </div>
              </th>
              <th
                class="px-4 py-3 font-medium cursor-pointer hover:bg-gray-100 select-none"
                @click="toggleSort('payer')"
              >
                <div class="flex items-center gap-1">
                  Zahler
                  <ArrowUp v-if="sortField === 'payer' && sortDirection === 'asc'" class="h-4 w-4" />
                  <ArrowDown v-else-if="sortField === 'payer' && sortDirection === 'desc'" class="h-4 w-4" />
                  <ArrowUpDown v-else class="h-4 w-4 text-gray-400" />
                </div>
              </th>
              <th
                class="px-4 py-3 font-medium cursor-pointer hover:bg-gray-100 select-none"
                @click="toggleSort('description')"
              >
                <div class="flex items-center gap-1">
                  Beschreibung
                  <ArrowUp v-if="sortField === 'description' && sortDirection === 'asc'" class="h-4 w-4" />
                  <ArrowDown v-else-if="sortField === 'description' && sortDirection === 'desc'" class="h-4 w-4" />
                  <ArrowUpDown v-else class="h-4 w-4 text-gray-400" />
                </div>
              </th>
              <th
                class="px-4 py-3 font-medium text-right cursor-pointer hover:bg-gray-100 select-none"
                @click="toggleSort('amount')"
              >
                <div class="flex items-center justify-end gap-1">
                  Betrag
                  <ArrowUp v-if="sortField === 'amount' && sortDirection === 'asc'" class="h-4 w-4" />
                  <ArrowDown v-else-if="sortField === 'amount' && sortDirection === 'desc'" class="h-4 w-4" />
                  <ArrowUpDown v-else class="h-4 w-4 text-gray-400" />
                </div>
              </th>
              <th class="px-4 py-3 font-medium">Status</th>
              <th class="px-4 py-3 font-medium text-right">Aktionen</th>
            </tr>
          </thead>
          <tbody>
            <template v-for="row in pagedRows" :key="row.key">
              <tr class="border-t hover:bg-gray-50">
                <td class="px-4 py-3 text-gray-600 whitespace-nowrap">
                  {{ formatDate(row.tx.bookingDate) }}
                </td>
                <td class="px-4 py-3">
                  <div class="font-medium">{{ row.tx.payerName || 'Unbekannt' }}</div>
                  <div v-if="row.tx.payerIban" class="text-xs text-gray-500 font-mono">
                    {{ row.tx.payerIban }}
                  </div>
                  <template v-for="warning in row.warnings" :key="warning.id">
                    <router-link
                      v-if="warning.child"
                      :to="`/kinder/${warning.child.id}`"
                      class="inline-block mt-1 px-2 py-0.5 bg-primary/10 text-primary rounded-full text-xs font-medium hover:bg-primary/20 transition-colors"
                    >
                      {{ warning.child.firstName }} {{ warning.child.lastName }}
                    </router-link>
                  </template>
                </td>
                <td class="px-4 py-3 text-gray-600 truncate max-w-xs">
                  {{ row.tx.description }}
                </td>
                <td
                  :class="[
                    'px-4 py-3 text-right font-medium whitespace-nowrap',
                    row.matched ? 'text-green-600' : '',
                  ]"
                >
                  {{ formatCurrency(row.tx.amount) }}
                </td>
                <td class="px-4 py-3">
                  <div class="flex flex-col items-start gap-1.5">
                    <span
                      v-if="isPartiallyAllocated(row)"
                      class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-orange-100 text-orange-700"
                    >
                      <AlertTriangle class="h-3 w-3" />
                      Teilweise zugeordnet · Rest {{ formatCurrency(getTxRemaining(row.tx)) }}
                    </span>
                    <span
                      v-else-if="row.matched"
                      class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-700"
                    >
                      <CheckCircle class="h-3 w-3" />
                      Zugeordnet
                    </span>
                    <span
                      v-else
                      class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-amber-100 text-amber-700"
                    >
                      <AlertTriangle class="h-3 w-3" />
                      Nicht zugeordnet
                    </span>

                    <div v-if="row.matched && row.tx.matches && row.tx.matches.length > 0" class="flex flex-wrap gap-1">
                      <span
                        v-for="match in row.tx.matches"
                        :key="match.id"
                        :class="[
                          'inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium',
                          getFeeTypeColor(match.expectation?.feeType || ''),
                        ]"
                      >
                        {{ getFeeTypeName(match.expectation?.feeType || '') }}
                        <span class="ml-1 opacity-75">{{ formatCurrency(match.expectation?.amount || 0) }}</span>
                      </span>
                    </div>

                    <button
                      v-if="row.warnings.length > 0"
                      @click="toggleWarnings(row.key)"
                      class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-orange-100 text-orange-700 hover:bg-orange-200 transition-colors"
                    >
                      <AlertTriangle class="h-3 w-3" />
                      {{ row.warnings.length }} {{ row.warnings.length === 1 ? 'Warnung' : 'Warnungen' }}
                      <ChevronDown v-if="!expandedWarnings.has(row.key)" class="h-3 w-3" />
                      <ChevronUp v-else class="h-3 w-3" />
                    </button>
                  </div>
                </td>
                <td class="px-4 py-3 text-right">
                  <!-- Unmatched actions -->
                  <template v-if="!row.matched">
                    <div v-if="dismissConfirmId === row.tx.id" class="flex items-center justify-end gap-2">
                      <span class="text-xs text-gray-500">Ignorieren?</span>
                      <button
                        @click="dismissTransaction(row.tx)"
                        class="px-2 py-1 text-xs bg-red-500 text-white rounded hover:bg-red-600"
                      >
                        Ja
                      </button>
                      <button
                        @click="cancelDismiss"
                        class="px-2 py-1 text-xs bg-gray-200 text-gray-700 rounded hover:bg-gray-300"
                      >
                        Nein
                      </button>
                    </div>
                    <div v-else-if="hideConfirmId === row.tx.id" class="flex items-center justify-end gap-2">
                      <span class="text-xs text-gray-500">Ausblenden?</span>
                      <button
                        @click="hideTransaction(row.tx)"
                        class="px-2 py-1 text-xs bg-gray-700 text-white rounded hover:bg-gray-800"
                      >
                        Ja
                      </button>
                      <button
                        @click="cancelHide"
                        class="px-2 py-1 text-xs bg-gray-200 text-gray-700 rounded hover:bg-gray-300"
                      >
                        Nein
                      </button>
                    </div>
                    <div v-else class="flex items-center justify-end gap-2">
                      <button
                        @click="openManualMatch(row.tx)"
                        class="inline-flex items-center gap-1 px-2 py-1 text-xs text-primary hover:text-primary/80 hover:bg-primary/10 rounded transition-colors"
                        title="Manuell zuordnen"
                      >
                        <LinkIcon class="h-3 w-3" />
                        Zuordnen
                      </button>
                      <button
                        @click="showHideConfirm(row.tx.id)"
                        :disabled="isHiding === row.tx.id"
                        class="inline-flex items-center gap-1 px-2 py-1 text-xs text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded transition-colors disabled:opacity-50"
                        title="Transaktion ausblenden"
                      >
                        <Loader2 v-if="isHiding === row.tx.id" class="h-3 w-3 animate-spin" />
                        <EyeOff v-else class="h-3 w-3" />
                        Ausblenden
                      </button>
                      <button
                        @click="showDismissConfirm(row.tx.id)"
                        :disabled="isDismissing === row.tx.id"
                        class="inline-flex items-center gap-1 px-2 py-1 text-xs text-gray-600 hover:text-red-600 hover:bg-red-50 rounded transition-colors disabled:opacity-50"
                        title="IBAN dauerhaft ignorieren"
                      >
                        <Loader2 v-if="isDismissing === row.tx.id" class="h-3 w-3 animate-spin" />
                        <Ban v-else class="h-3 w-3" />
                        Ignorieren
                      </button>
                    </div>
                  </template>

                  <!-- Matched actions -->
                  <template v-else>
                    <div v-if="unmatchConfirmId === row.tx.id" class="flex items-center justify-end gap-2">
                      <span class="text-xs text-gray-500">Zuordnung aufheben?</span>
                      <button
                        @click="unmatchTransaction(row.tx)"
                        class="px-2 py-1 text-xs bg-amber-500 text-white rounded hover:bg-amber-600"
                      >
                        Ja
                      </button>
                      <button
                        @click="cancelMatchAction"
                        class="px-2 py-1 text-xs bg-gray-200 text-gray-700 rounded hover:bg-gray-300"
                      >
                        Nein
                      </button>
                    </div>
                    <div v-else-if="deleteConfirmId === row.tx.id" class="flex items-center justify-end gap-2">
                      <span class="text-xs text-gray-500">Transaktion löschen?</span>
                      <button
                        @click="unmatchTransaction(row.tx, true)"
                        class="px-2 py-1 text-xs bg-red-500 text-white rounded hover:bg-red-600"
                      >
                        Ja
                      </button>
                      <button
                        @click="cancelMatchAction"
                        class="px-2 py-1 text-xs bg-gray-200 text-gray-700 rounded hover:bg-gray-300"
                      >
                        Nein
                      </button>
                    </div>
                    <div v-else class="flex items-center justify-end gap-2">
                      <button
                        @click="showUnmatchConfirm(row.tx.id)"
                        :disabled="isUnmatching === row.tx.id || isDeletingMatched === row.tx.id"
                        class="inline-flex items-center gap-1 px-2 py-1 text-xs text-gray-600 hover:text-amber-700 hover:bg-amber-50 rounded transition-colors disabled:opacity-50"
                        title="Zuordnung aufheben"
                      >
                        <Loader2 v-if="isUnmatching === row.tx.id" class="h-3 w-3 animate-spin" />
                        <Unlink v-else class="h-3 w-3" />
                        Aufheben
                      </button>
                      <button
                        @click="showDeleteConfirm(row.tx.id)"
                        :disabled="isUnmatching === row.tx.id || isDeletingMatched === row.tx.id"
                        class="inline-flex items-center gap-1 px-2 py-1 text-xs text-gray-600 hover:text-red-600 hover:bg-red-50 rounded transition-colors disabled:opacity-50"
                        title="Transaktion löschen"
                      >
                        <Loader2 v-if="isDeletingMatched === row.tx.id" class="h-3 w-3 animate-spin" />
                        <Trash2 v-else class="h-3 w-3" />
                        Löschen
                      </button>
                    </div>
                  </template>
                </td>
              </tr>

              <!-- Expanded warning details -->
              <tr v-if="expandedWarnings.has(row.key) && row.warnings.length > 0" class="border-t bg-orange-50/40">
                <td colspan="6" class="px-4 py-4">
                  <div class="space-y-4">
                    <div
                      v-for="warning in row.warnings"
                      :key="warning.id"
                      class="bg-white rounded-xl border p-4"
                    >
                      <div class="flex items-start justify-between gap-4">
                        <div class="flex-1">
                          <div class="flex items-center gap-2 mb-2">
                            <span
                              :class="[
                                'px-2 py-0.5 rounded-full text-xs font-medium',
                                getWarningTypeColor(warning.warningType),
                              ]"
                            >
                              {{ getWarningTypeLabel(warning.warningType) }}
                            </span>
                            <router-link
                              v-if="warning.child"
                              :to="`/kinder/${warning.child.id}`"
                              class="px-2 py-0.5 bg-primary/10 text-primary rounded-full text-xs font-medium hover:bg-primary/20 transition-colors"
                            >
                              {{ warning.child.firstName }} {{ warning.child.lastName }}
                            </router-link>
                            <span class="text-xs text-gray-500">
                              {{ formatDateTime(warning.createdAt) }}
                            </span>
                          </div>

                          <p class="text-gray-700 mb-3">{{ warning.message }}</p>

                          <div v-if="warning.transaction" class="p-3 bg-gray-50 rounded-lg text-sm space-y-1">
                            <div class="flex justify-between">
                              <span class="text-gray-500">Transaktion:</span>
                              <span class="font-medium">{{ warning.transaction.payerName || 'Unbekannt' }}</span>
                            </div>
                            <div class="flex justify-between">
                              <span class="text-gray-500">Betrag:</span>
                              <span class="font-medium text-green-600">{{ formatCurrency(warning.transaction.amount) }}</span>
                            </div>
                            <div class="flex justify-between">
                              <span class="text-gray-500">Datum:</span>
                              <span>{{ formatDate(warning.transaction.bookingDate) }}</span>
                            </div>
                            <div v-if="warning.transaction.description" class="text-gray-600 text-xs truncate">
                              {{ warning.transaction.description }}
                            </div>
                          </div>

                          <div
                            v-if="warning.warningType === 'LATE_PAYMENT' && warning.matchedFee"
                            class="mt-2 p-3 bg-orange-50 rounded-lg text-sm space-y-1"
                          >
                            <div class="flex items-center gap-1 text-orange-700 font-medium mb-1">
                              <Clock class="h-4 w-4" />
                              Verspätete Zahlung für:
                            </div>
                            <div class="flex justify-between">
                              <span class="text-gray-500">Beitragsart:</span>
                              <span class="font-medium">{{ getFeeTypeName(warning.matchedFee.feeType) }}</span>
                            </div>
                            <div class="flex justify-between">
                              <span class="text-gray-500">Zeitraum:</span>
                              <span>{{ formatMonthName(warning.matchedFee.month) }} {{ warning.matchedFee.year }}</span>
                            </div>
                            <div class="flex justify-between">
                              <span class="text-gray-500">Betrag:</span>
                              <span class="font-medium">{{ formatCurrency(warning.matchedFee.amount) }}</span>
                            </div>
                          </div>
                        </div>

                        <div class="flex flex-col gap-2">
                          <button
                            v-if="warning.warningType === 'LATE_PAYMENT'"
                            @click="resolveLateFee(warning)"
                            :disabled="isResolvingWarning === warning.id"
                            class="inline-flex items-center gap-1 px-3 py-1.5 text-sm bg-orange-500 text-white rounded-lg hover:bg-orange-600 transition-colors disabled:opacity-50"
                            title="Mahngebuhr von 10 EUR erstellen"
                          >
                            <Loader2 v-if="isResolvingWarning === warning.id" class="h-4 w-4 animate-spin" />
                            <Euro v-else class="h-4 w-4" />
                            Mahngebuhr erstellen
                          </button>

                          <div v-if="dismissWarningId === warning.id" class="p-3 bg-gray-50 rounded-lg space-y-2">
                            <input
                              v-model="dismissNote"
                              type="text"
                              placeholder="Notiz (optional)"
                              class="w-full px-2 py-1 text-sm border rounded"
                            />
                            <div class="flex gap-2">
                              <button
                                @click="dismissWarning(warning)"
                                :disabled="isResolvingWarning === warning.id"
                                class="px-2 py-1 text-xs bg-red-500 text-white rounded hover:bg-red-600 disabled:opacity-50"
                              >
                                Verwerfen
                              </button>
                              <button
                                @click="cancelWarningDismiss"
                                class="px-2 py-1 text-xs bg-gray-200 text-gray-700 rounded hover:bg-gray-300"
                              >
                                Abbrechen
                              </button>
                            </div>
                          </div>

                          <button
                            v-else
                            @click="showWarningDismiss(warning.id)"
                            :disabled="isResolvingWarning === warning.id"
                            class="inline-flex items-center gap-1 px-3 py-1.5 text-sm text-gray-600 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors disabled:opacity-50"
                          >
                            <XCircle class="h-4 w-4" />
                            Verwerfen
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="flex items-center justify-between px-4 py-3 border-t bg-gray-50">
        <div class="text-sm text-gray-600">
          Seite {{ page }} von {{ totalPages }} ({{ sortedRows.length }} Einträge)
        </div>
        <div class="flex items-center gap-2">
          <button
            @click="goToPage(page - 1)"
            :disabled="page <= 1"
            class="p-1 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <ChevronLeft class="h-5 w-5" />
          </button>
          <button
            v-for="pageNumber in visiblePages"
            :key="pageNumber"
            @click="goToPage(pageNumber)"
            :class="[
              'px-3 py-1 rounded text-sm',
              pageNumber === page ? 'bg-primary text-white' : 'hover:bg-gray-200',
            ]"
          >
            {{ pageNumber }}
          </button>
          <button
            @click="goToPage(page + 1)"
            :disabled="page >= totalPages"
            class="p-1 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <ChevronRight class="h-5 w-5" />
          </button>
        </div>
      </div>
    </div>

    <ImportUploadModal
      v-if="showUploadModal"
      @close="showUploadModal = false"
      @imported="loadLatestImport"
      @confirmed="loadTransactions"
    />

    <ImportHistoryModal
      v-if="showHistoryModal"
      :expand-batch-id="expandedBatchId"
      @close="showHistoryModal = false"
      @loaded="latestImportBatch = $event"
    />

    <ImportBlacklistModal v-if="showBlacklistModal" @close="showBlacklistModal = false" />

    <ManualMatchModal
      v-if="manualMatchTransaction"
      :transaction="manualMatchTransaction"
      @close="manualMatchTransaction = null"
      @matched="onManualMatched"
    />
  </div>
</template>
