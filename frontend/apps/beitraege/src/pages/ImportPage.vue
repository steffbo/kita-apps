<script setup lang="ts">
import TransactionTable from '@/components/import/TransactionTable.vue';
import { ref, onMounted, onUnmounted } from 'vue';
import { useRoute } from 'vue-router';
import { api } from '@/api';
import type { ImportBatch, BankTransaction, RescanResult } from '@/api/types';
import { useImportTransactions } from '@/composables/useImportTransactions';
import { useImportWarningActions } from '@/composables/useImportWarningActions';
import BankingSyncCard from '@/components/BankingSyncCard.vue';
import { useAuthStore } from '@/stores/auth';
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
  ShieldOff,
  RefreshCw,
} from 'lucide-vue-next';
import { formatDateTime } from '@/utils/format';
import SearchInput from '@/components/SearchInput.vue';

const route = useRoute();
const authStore = useAuthStore();
const uploadError = ref<string | null>(null);
const {
  activeFilter,
  warnings,
  warningsTotal,
  isLoadingTransactions,
  transactionSearch,
  sortField,
  sortDirection,
  page,
  toggleSort,
  offenCount,
  warnungenCount,
  zugeordnetCount,
  totalPages,
  visiblePages,
  pagedRows,
  totalRows,
  allTotal,
  loadTransactions,
  goToPage,
} = useImportTransactions(uploadError);

// Modals (each loads and resets its own state)
const showUploadModal = ref(false);
const showHistoryModal = ref(false);
const showBlacklistModal = ref(false);
const manualMatchTransaction = ref<BankTransaction | null>(null);
// Batch whose errors the history modal expands on open
const expandedBatchId = ref<string | null>(null);

const {
  isResolvingWarning,
  dismissWarningId,
  dismissNote,
  showWarningDismiss,
  cancelWarningDismiss,
  dismissWarning,
  resolveLateFee,
} = useImportWarningActions(warnings, warningsTotal, uploadError, loadTransactions);

// Most recent import (usually the automated banking sync), to surface its errors.
const latestImportBatch = ref<ImportBatch | null>(null);

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

function toggleWarnings(key: string): void {
  if (expandedWarnings.value.has(key)) {
    expandedWarnings.value.delete(key);
  } else {
    expandedWarnings.value.add(key);
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
      return 'bg-amber-100 dark:bg-amber-950/40 text-amber-700 dark:text-amber-300';
    case 'DUPLICATE_PAYMENT':
      return 'bg-red-100 dark:bg-red-950/40 text-red-700 dark:text-red-300';
    case 'UNKNOWN_IBAN':
      return 'bg-muted text-foreground';
    case 'LATE_PAYMENT':
      return 'bg-orange-100 dark:bg-orange-950/40 text-orange-700 dark:text-orange-300';
    case 'OVERPAYMENT':
      return 'bg-purple-100 dark:bg-purple-950/40 text-purple-700 dark:text-purple-300';
    default:
      return 'bg-muted text-foreground';
  }
}
</script>

<template>
  <div>
    <!-- Header -->
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-foreground">Bankabgleich</h1>
      <p class="text-muted-foreground mt-1">
        Banktransaktionen abrufen und Zahlungen zuordnen
      </p>
    </div>

    <!-- Banking Sync -->
    <BankingSyncCard v-if="authStore.isAdmin" @sync-finished="refreshAfterSync" />

    <!-- Rescan Result -->
    <div
      v-if="rescanResult"
      class="mb-4 p-4 bg-blue-50 dark:bg-blue-950/40 border border-blue-200 rounded-lg flex items-start gap-3"
    >
      <CheckCircle class="h-5 w-5 text-blue-500 dark:text-blue-400 flex-shrink-0 mt-0.5" />
      <div>
        <p class="text-blue-700 dark:text-blue-300 font-medium">Erneute Zuordnung abgeschlossen</p>
        <p class="text-sm text-blue-600 dark:text-blue-300">
          {{ rescanResult.scanned }} Transaktionen gescannt<span v-if="rescanResult.autoMatched > 0">, {{ rescanResult.autoMatched }} automatisch zugeordnet</span><span v-if="rescanResult.newMatches > 0">, {{ rescanResult.newMatches }} Vorschläge zur Überprüfung</span>
        </p>
      </div>
      <button
        @click="rescanResult = null"
        class="ml-auto text-blue-500 dark:text-blue-400 hover:text-blue-700 dark:text-blue-300"
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
      class="mb-4 p-4 bg-red-50 dark:bg-red-950/40 border border-red-200 rounded-lg flex items-start gap-3"
      role="alert"
    >
      <AlertTriangle class="h-5 w-5 text-red-500 dark:text-red-400 flex-shrink-0 mt-0.5" />
      <div class="flex-1">
        <p class="text-red-700 dark:text-red-300 font-medium">
          Letzter Import vom {{ formatDateTime(latestImportBatch.importedAt) }}:
          {{ latestImportBatch.errorCount }}
          {{ latestImportBatch.errorCount === 1 ? 'Buchung' : 'Buchungen' }} mit Fehlern
        </p>
        <p class="text-sm text-red-600 dark:text-red-300">{{ latestImportBatch.importedByEmail }} · {{ latestImportBatch.fileName }}</p>
      </div>
      <button @click="openLatestImportErrors" class="text-sm text-red-700 dark:text-red-300 hover:text-red-900 dark:text-red-300 underline">
        Details
      </button>
    </div>

    <!-- Toolbar -->
    <div class="bg-card rounded-xl border p-4 mb-6 space-y-3">
      <div class="flex flex-col lg:flex-row lg:items-center gap-3">
        <div class="flex flex-wrap items-center gap-2">
          <button
            @click="activeFilter = 'offen'"
            :class="[
              'inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm font-medium border transition-colors',
              activeFilter === 'offen'
                ? 'bg-primary text-primary-foreground border-primary'
                : 'bg-card text-muted-foreground border-border hover:bg-accent',
            ]"
          >
            Offen
            <span
              v-if="offenCount > 0"
              :class="[
                'px-1.5 py-0.5 text-xs rounded-full',
                activeFilter === 'offen' ? 'bg-card/20 text-white' : 'bg-amber-100 dark:bg-amber-950/40 text-amber-700 dark:text-amber-300',
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
                ? 'bg-primary text-primary-foreground border-primary'
                : 'bg-card text-muted-foreground border-border hover:bg-accent',
            ]"
          >
            Warnungen
            <span
              v-if="warnungenCount > 0"
              :class="[
                'px-1.5 py-0.5 text-xs rounded-full',
                activeFilter === 'warnungen' ? 'bg-card/20 text-white' : 'bg-orange-100 dark:bg-orange-950/40 text-orange-700 dark:text-orange-300',
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
                ? 'bg-primary text-primary-foreground border-primary'
                : 'bg-card text-muted-foreground border-border hover:bg-accent',
            ]"
          >
            Zugeordnet
            <span
              v-if="zugeordnetCount > 0"
              :class="[
                'px-1.5 py-0.5 text-xs rounded-full',
                activeFilter === 'zugeordnet' ? 'bg-card/20 text-white' : 'bg-green-100 dark:bg-green-950/40 text-green-700 dark:text-green-300',
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
                ? 'bg-primary text-primary-foreground border-primary'
                : 'bg-card text-muted-foreground border-border hover:bg-accent',
            ]"
          >
            Alle
          </button>
        </div>

        <div class="flex-1 min-w-[220px]">
          <SearchInput v-model="transactionSearch" placeholder="Suche nach Zahler oder Beschreibung..."
            class="max-w-md lg:ml-auto" />
        </div>

        <div class="flex items-center gap-2">
          <button
            @click="rescanTransactions"
            :disabled="isRescanning || offenCount === 0"
            class="inline-flex items-center gap-1 px-3 py-2 text-sm border border-border text-foreground rounded-lg hover:bg-accent transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            title="Automatische Zuordnung erneut ausführen"
          >
            <Loader2 v-if="isRescanning" class="h-4 w-4 animate-spin" />
            <RefreshCw v-else class="h-4 w-4" />
            Erneut zuordnen
          </button>
          <button
            @click="showUploadModal = true"
            class="inline-flex items-center gap-1 px-3 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors"
          >
            <Upload class="h-4 w-4" />
            CSV hochladen
          </button>
        </div>
      </div>

      <div class="flex flex-wrap items-center justify-between gap-3 pt-1 border-t text-sm">
        <p class="text-muted-foreground pt-2">
          {{ totalRows }} Transaktionen
          <span v-if="totalRows !== allTotal"> (gefiltert von {{ allTotal }})</span>
        </p>
        <div class="flex items-center gap-4 pt-2">
          <button
            @click="openHistoryModal"
            class="inline-flex items-center gap-1 text-muted-foreground hover:text-foreground underline"
          >
            <History class="h-4 w-4" />
            Import-Historie
          </button>
          <button
            @click="openBlacklistModal"
            class="inline-flex items-center gap-1 text-muted-foreground hover:text-foreground underline"
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
      class="mb-4 p-4 bg-red-50 dark:bg-red-950/40 border border-red-200 rounded-lg flex items-start gap-3"
    >
      <XCircle class="h-5 w-5 text-red-500 dark:text-red-400 flex-shrink-0 mt-0.5" />
      <div>
        <p class="text-red-700 dark:text-red-300 font-medium">Fehler</p>
        <p class="text-sm text-red-600 dark:text-red-300">{{ uploadError }}</p>
      </div>
      <button @click="uploadError = null" class="ml-auto text-red-400 hover:text-red-600 dark:text-red-300">
        <XCircle class="h-4 w-4" />
      </button>
    </div>

    <TransactionTable
      :activeFilter="activeFilter"
      :isLoadingTransactions="isLoadingTransactions"
      :transactionSearch="transactionSearch"
      :sortField="sortField"
      :sortDirection="sortDirection"
      :page="page"
      :toggleSort="toggleSort"
      :totalRows="totalRows"
      :totalPages="totalPages"
      :visiblePages="visiblePages"
      :pagedRows="pagedRows"
      :goToPage="goToPage"
      :isResolvingWarning="isResolvingWarning"
      :dismissWarningId="dismissWarningId"
      v-model:dismissNote="dismissNote"
      :showWarningDismiss="showWarningDismiss"
      :cancelWarningDismiss="cancelWarningDismiss"
      :dismissWarning="dismissWarning"
      :resolveLateFee="resolveLateFee"
      :expandedWarnings="expandedWarnings"
      :isDismissing="isDismissing"
      :dismissConfirmId="dismissConfirmId"
      :isHiding="isHiding"
      :hideConfirmId="hideConfirmId"
      :unmatchConfirmId="unmatchConfirmId"
      :deleteConfirmId="deleteConfirmId"
      :isUnmatching="isUnmatching"
      :isDeletingMatched="isDeletingMatched"
      :toggleWarnings="toggleWarnings"
      :openManualMatch="openManualMatch"
      :showDismissConfirm="showDismissConfirm"
      :cancelDismiss="cancelDismiss"
      :showHideConfirm="showHideConfirm"
      :cancelHide="cancelHide"
      :showUnmatchConfirm="showUnmatchConfirm"
      :showDeleteConfirm="showDeleteConfirm"
      :cancelMatchAction="cancelMatchAction"
      :dismissTransaction="dismissTransaction"
      :hideTransaction="hideTransaction"
      :unmatchTransaction="unmatchTransaction"
      :getWarningTypeLabel="getWarningTypeLabel"
      :getWarningTypeColor="getWarningTypeColor"
    /><ImportUploadModal
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
