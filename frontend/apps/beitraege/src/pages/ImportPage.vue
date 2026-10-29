<script setup lang="ts">
import TransactionTable from '@/components/import/TransactionTable.vue';
import { ref, onMounted } from 'vue';
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

onMounted(() => {
  loadTransactions();
  loadLatestImport();
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
      return 'bg-warning-soft text-warning';
    case 'DUPLICATE_PAYMENT':
      return 'bg-destructive-soft text-destructive';
    case 'UNKNOWN_IBAN':
      return 'bg-muted text-foreground';
    case 'LATE_PAYMENT':
      return 'bg-warning-soft text-warning';
    case 'OVERPAYMENT':
      return 'bg-primary-soft text-primary';
    default:
      return 'bg-muted text-foreground';
  }
}
</script>

<template>
  <div>
    <!-- Banking Sync -->
    <BankingSyncCard v-if="authStore.isAdmin" @sync-finished="refreshAfterSync" />

    <!-- Rescan Result -->
    <div
      v-if="rescanResult"
      class="border mb-4 p-4 bg-primary-soft border-border rounded-lg flex items-start gap-3"
    >
      <CheckCircle class="h-5 w-5 text-primary flex-shrink-0 mt-0.5" />
      <div>
        <p class="text-primary font-medium">Erneute Zuordnung abgeschlossen</p>
        <p class="text-sm text-primary">
          {{ rescanResult.scanned }} Transaktionen gescannt<span v-if="rescanResult.autoMatched > 0">, {{ rescanResult.autoMatched }} automatisch zugeordnet</span><span v-if="rescanResult.newMatches > 0">, {{ rescanResult.newMatches }} Vorschläge zur Überprüfung</span>
        </p>
      </div>
      <button
        @click="rescanResult = null"
        class="ml-auto text-primary hover:text-primary"
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
      class="border mb-4 p-4 bg-destructive-soft border-destructive-line rounded-lg flex items-start gap-3"
      role="alert"
    >
      <AlertTriangle class="h-5 w-5 text-destructive flex-shrink-0 mt-0.5" />
      <div class="flex-1">
        <p class="text-destructive font-medium">
          Letzter Import vom {{ formatDateTime(latestImportBatch.importedAt) }}:
          {{ latestImportBatch.errorCount }}
          {{ latestImportBatch.errorCount === 1 ? 'Buchung' : 'Buchungen' }} mit Fehlern
        </p>
        <p class="text-sm text-destructive">{{ latestImportBatch.importedByEmail }} · {{ latestImportBatch.fileName }}</p>
      </div>
      <button @click="openLatestImportErrors" class="text-sm text-destructive hover:text-destructive underline">
        Details
      </button>
    </div>

    <!-- Toolbar -->
    <div class="card p-4 mb-6 space-y-3">
      <div class="flex flex-col gap-2 sm:flex-row">
        <SearchInput v-model="transactionSearch" placeholder="Zahler oder Beschreibung" class="flex-1" />
        <div class="flex gap-2">
          <button
            @click="rescanTransactions"
            :disabled="isRescanning || offenCount === 0"
            class="btn btn-secondary flex-1 sm:flex-none"
            title="Automatische Zuordnung erneut ausführen"
          >
            <Loader2 v-if="isRescanning" class="animate-spin" />
            <RefreshCw v-else />
            Erneut zuordnen
          </button>
          <button @click="showUploadModal = true" class="btn btn-primary flex-1 sm:flex-none">
            <Upload />
            CSV hochladen
          </button>
        </div>
      </div>
        <div class="flex flex-wrap items-center gap-2">
          <button
            @click="activeFilter = 'offen'"
            class="chip"
            :aria-pressed="activeFilter === 'offen'"
          >
            Offen
            <span
              v-if="offenCount > 0"
              :class="[
                'chip-count',
                activeFilter === 'offen' ? 'text-primary-text' : 'text-warning',
              ]"
            >
              {{ offenCount }}
            </span>
          </button>
          <button
            @click="activeFilter = 'warnungen'"
            class="chip"
            :aria-pressed="activeFilter === 'warnungen'"
          >
            Warnungen
            <span
              v-if="warnungenCount > 0"
              :class="[
                'chip-count',
                activeFilter === 'warnungen' ? 'text-primary-text' : 'text-warning',
              ]"
            >
              {{ warnungenCount }}
            </span>
          </button>
          <button
            @click="activeFilter = 'zugeordnet'"
            class="chip"
            :aria-pressed="activeFilter === 'zugeordnet'"
          >
            Zugeordnet
            <span
              v-if="zugeordnetCount > 0"
              :class="[
                'chip-count',
                activeFilter === 'zugeordnet' ? 'text-primary-text' : 'text-success',
              ]"
            >
              {{ zugeordnetCount }}
            </span>
          </button>
          <button
            @click="activeFilter = 'alle'"
            class="chip"
            :aria-pressed="activeFilter === 'alle'"
          >
            Alle
          </button>
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
      class="border mb-4 p-4 bg-destructive-soft border-destructive-line rounded-lg flex items-start gap-3"
    >
      <XCircle class="h-5 w-5 text-destructive flex-shrink-0 mt-0.5" />
      <div>
        <p class="text-destructive font-medium">Fehler</p>
        <p class="text-sm text-destructive">{{ uploadError }}</p>
      </div>
      <button @click="uploadError = null" class="ml-auto text-destructive hover:text-destructive">
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
