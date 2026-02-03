<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue';
import { useRoute } from 'vue-router';
import { api } from '@/api';
import type { ImportResult, ImportBatch, BankTransaction, MatchConfirmation, KnownIBAN, TransactionWarning, MatchSuggestion, FeeExpectation } from '@/api/types';
import {
  Upload,
  FileSpreadsheet,
  Loader2,
  CheckCircle,
  XCircle,
  AlertTriangle,
  History,
  Link2,
  ChevronDown,
  ChevronUp,
  ChevronLeft,
  ChevronRight,
  RefreshCw,
  Check,
  Ban,
  Trash2,
  ShieldOff,
  Clock,
  Euro,
  LinkIcon,
  Unlink,
  Search,
  ArrowUp,
  ArrowDown,
  ArrowUpDown,
} from 'lucide-vue-next';

type TabType = 'upload' | 'history' | 'unmatched' | 'matched' | 'warnings' | 'blacklist';

const route = useRoute();
const activeTab = ref<TabType>('upload');

// Upload state
const isDragging = ref(false);
const isUploading = ref(false);
const uploadError = ref<string | null>(null);
const importResult = ref<ImportResult | null>(null);
const selectedMatches = ref<Set<string>>(new Set());
const isConfirming = ref(false);
const confirmResult = ref<{ confirmed: number; failed: number } | null>(null);

// History state
const importHistory = ref<ImportBatch[]>([]);
const historyTotal = ref(0);
const isLoadingHistory = ref(false);

// Unmatched transactions state
const unmatchedTransactions = ref<BankTransaction[]>([]);
const unmatchedTotal = ref(0);
const isLoadingUnmatched = ref(false);

// Blacklist state
const blacklistedIBANs = ref<KnownIBAN[]>([]);
const blacklistTotal = ref(0);
const isLoadingBlacklist = ref(false);

// Warnings state
const warnings = ref<TransactionWarning[]>([]);
const warningsTotal = ref(0);
const isLoadingWarnings = ref(false);
const isResolvingWarning = ref<string | null>(null);
const dismissWarningId = ref<string | null>(null);
const dismissNote = ref('');

// Matched transactions state
const matchedTransactions = ref<BankTransaction[]>([]);
const matchedTotal = ref(0);
const isLoadingMatched = ref(false);

// Manual match modal state
const manualMatchTransaction = ref<BankTransaction | null>(null);
const manualMatchSuggestion = ref<MatchSuggestion | null>(null);
const isLoadingSuggestions = ref(false);
const isLoadingFees = ref(false);
const availableFees = ref<FeeExpectation[]>([]);
const feeSearch = ref('');
const isCreatingMatch = ref(false);
const showAllFees = ref(false);

const PREFILTER_CONFIDENCE = 0.6;

// Rescan state
const isRescanning = ref(false);
const rescanResult = ref<{ scanned: number; autoMatched: number; newMatches: number; suggestions: MatchSuggestion[] } | null>(null);

// Dismiss state
const isDismissing = ref<string | null>(null);
const dismissConfirmId = ref<string | null>(null);
const unmatchConfirmId = ref<string | null>(null);
const deleteConfirmId = ref<string | null>(null);
const isUnmatching = ref<string | null>(null);
const isDeletingMatched = ref<string | null>(null);

// Transaction search and sorting state (server-side)
const transactionSearch = ref('');
const debouncedTransactionSearch = ref('');
type TransactionSortField = 'date' | 'payer' | 'description' | 'amount';
type SortDirection = 'asc' | 'desc';
const unmatchedSortField = ref<TransactionSortField>('date');
const unmatchedSortDirection = ref<SortDirection>('desc');
const matchedSortField = ref<TransactionSortField>('date');
const matchedSortDirection = ref<SortDirection>('desc');

// Pagination state
const unmatchedPage = ref(1);
const matchedPage = ref(1);
const pageSize = 20;

// Debounce search
let searchTimeout: ReturnType<typeof setTimeout>;
watch(transactionSearch, (newVal) => {
  clearTimeout(searchTimeout);
  searchTimeout = setTimeout(() => {
    debouncedTransactionSearch.value = newVal;
    unmatchedPage.value = 1;
    matchedPage.value = 1;

    const reloadActions: Record<string, () => Promise<void>> = {
      unmatched: loadUnmatched,
      matched: loadMatched,
    };

    const reloadFn = reloadActions[activeTab.value];
    if (reloadFn) {
      reloadFn();
    }
  }, 300);
});

function toggleSort(
  field: TransactionSortField,
  currentField: Ref<TransactionSortField>,
  currentDirection: Ref<SortDirection>,
  pageRef: Ref<number>,
  reloadFn: () => Promise<void>
): void {
  if (currentField.value === field) {
    currentDirection.value = currentDirection.value === 'asc' ? 'desc' : 'asc';
  } else {
    currentField.value = field;
    currentDirection.value = 'asc';
  }
  pageRef.value = 1;
  reloadFn();
}

function toggleUnmatchedSort(field: TransactionSortField): void {
  toggleSort(field, unmatchedSortField, unmatchedSortDirection, unmatchedPage, loadUnmatched);
}

function toggleMatchedSort(field: TransactionSortField): void {
  toggleSort(field, matchedSortField, matchedSortDirection, matchedPage, loadMatched);
}

const unmatchedTotalPages = computed(() => Math.ceil(unmatchedTotal.value / pageSize));
const matchedTotalPages = computed(() => Math.ceil(matchedTotal.value / pageSize));

function goToPage(page: number, pageRef: Ref<number>, totalPages: number, reloadFn: () => Promise<void>): void {
  if (page >= 1 && page <= totalPages) {
    pageRef.value = page;
    reloadFn();
  }
}

function goToUnmatchedPage(page: number): void {
  goToPage(page, unmatchedPage, unmatchedTotalPages.value, loadUnmatched);
}

function goToMatchedPage(page: number): void {
  goToPage(page, matchedPage, matchedTotalPages.value, loadMatched);
}

const expandedSuggestions = ref<Set<string>>(new Set());

function toggleSuggestion(id: string): void {
  if (expandedSuggestions.value.has(id)) {
    expandedSuggestions.value.delete(id);
  } else {
    expandedSuggestions.value.add(id);
  }
}

function handleDragOver(e: DragEvent): void {
  e.preventDefault();
  isDragging.value = true;
}

function handleDragLeave(): void {
  isDragging.value = false;
}

async function handleDrop(e: DragEvent): Promise<void> {
  e.preventDefault();
  isDragging.value = false;
  const files = e.dataTransfer?.files;
  if (files && files.length > 0) {
    await uploadFile(files[0]);
  }
}

async function handleFileSelect(e: Event): Promise<void> {
  const input = e.target as HTMLInputElement;
  if (input.files && input.files.length > 0) {
    await uploadFile(input.files[0]);
    input.value = '';
  }
}

async function uploadFile(file: File): Promise<void> {
  if (!file.name.endsWith('.csv')) {
    uploadError.value = 'Bitte nur CSV-Dateien hochladen';
    return;
  }

  isUploading.value = true;
  uploadError.value = null;
  importResult.value = null;
  selectedMatches.value.clear();
  confirmResult.value = null;

  try {
    const result = await api.uploadCSV(file);
    importResult.value = result;

    // Pre-select high-confidence matches
    for (const suggestion of result.suggestions) {
      if (suggestion.confidence >= 0.8 && suggestion.expectation) {
        selectedMatches.value.add(suggestion.transaction.id);
      }
    }
  } catch (error) {
    uploadError.value = error instanceof Error ? error.message : 'Upload fehlgeschlagen';
  } finally {
    isUploading.value = false;
  }
}

function toggleMatch(transactionId: string): void {
  if (selectedMatches.value.has(transactionId)) {
    selectedMatches.value.delete(transactionId);
  } else {
    selectedMatches.value.add(transactionId);
  }
}

function selectAllMatches(): void {
  if (!importResult.value) return;
  for (const suggestion of importResult.value.suggestions) {
    if (suggestion.expectation) {
      selectedMatches.value.add(suggestion.transaction.id);
    }
  }
}

function deselectAllMatches(): void {
  selectedMatches.value.clear();
}

const matchableSuggestions = computed(() => {
  if (!importResult.value) return [];
  return importResult.value.suggestions.filter(s => s.expectation);
});

const unmatchableSuggestions = computed(() => {
  if (!importResult.value) return [];
  return importResult.value.suggestions.filter(s => !s.expectation);
});

async function confirmMatches(): Promise<void> {
  if (!importResult.value || selectedMatches.value.size === 0) return;

  isConfirming.value = true;
  try {
    const matches: MatchConfirmation[] = [];
    for (const suggestion of importResult.value.suggestions) {
      if (selectedMatches.value.has(suggestion.transaction.id) && suggestion.expectation) {
        matches.push({
          transactionId: suggestion.transaction.id,
          expectationId: suggestion.expectation.id,
        });
      }
    }

    const result = await api.confirmMatches(matches);
    confirmResult.value = result;

    // Remove confirmed matches from the list
    if (importResult.value) {
      importResult.value.suggestions = importResult.value.suggestions.filter(
        s => !selectedMatches.value.has(s.transaction.id)
      );
    }
    selectedMatches.value.clear();
  } catch (error) {
    uploadError.value = error instanceof Error ? error.message : 'Bestätigung fehlgeschlagen';
  } finally {
    isConfirming.value = false;
  }
}

async function loadHistory(): Promise<void> {
  isLoadingHistory.value = true;
  try {
    const response = await api.getImportHistory(0, 50);
    importHistory.value = response.data;
    historyTotal.value = response.total;
  } catch (error) {
    console.error('Failed to load history:', error);
  } finally {
    isLoadingHistory.value = false;
  }
}

async function loadUnmatched(): Promise<void> {
  isLoadingUnmatched.value = true;
  try {
    const response = await api.getUnmatchedTransactions({
      offset: (unmatchedPage.value - 1) * pageSize,
      limit: pageSize,
      search: debouncedTransactionSearch.value || undefined,
      sortBy: unmatchedSortField.value,
      sortDir: unmatchedSortDirection.value,
    });
    unmatchedTransactions.value = response.data;
    unmatchedTotal.value = response.total;
  } catch (error) {
    console.error('Failed to load unmatched transactions:', error);
  } finally {
    isLoadingUnmatched.value = false;
  }
}

async function loadBlacklist(): Promise<void> {
  isLoadingBlacklist.value = true;
  try {
    const response = await api.getBlacklist(0, 100);
    blacklistedIBANs.value = response.data;
    blacklistTotal.value = response.total;
  } catch (error) {
    console.error('Failed to load blacklist:', error);
  } finally {
    isLoadingBlacklist.value = false;
  }
}

async function loadWarnings(): Promise<void> {
  isLoadingWarnings.value = true;
  try {
    const response = await api.getWarnings(0, 100);
    warnings.value = response.data;
    warningsTotal.value = response.total;
  } catch (error) {
    console.error('Failed to load warnings:', error);
  } finally {
    isLoadingWarnings.value = false;
  }
}

async function loadMatched(): Promise<void> {
  isLoadingMatched.value = true;
  try {
    const response = await api.getMatchedTransactions({
      offset: (matchedPage.value - 1) * pageSize,
      limit: pageSize,
      search: debouncedTransactionSearch.value || undefined,
      sortBy: matchedSortField.value,
      sortDir: matchedSortDirection.value,
    });
    matchedTransactions.value = response.data;
    matchedTotal.value = response.total;
  } catch (error) {
    console.error('Failed to load matched transactions:', error);
  } finally {
    isLoadingMatched.value = false;
  }
}

async function rescanTransactions(): Promise<void> {
  isRescanning.value = true;
  rescanResult.value = null;
  try {
    const result = await api.rescanTransactions();
    rescanResult.value = result;
    await loadUnmatched();
  } catch (error) {
    console.error('Failed to rescan transactions:', error);
    uploadError.value = error instanceof Error ? error.message : 'Erneutes Zuordnen fehlgeschlagen';
  } finally {
    isRescanning.value = false;
  }
}

async function openManualMatch(transaction: BankTransaction): Promise<void> {
  manualMatchTransaction.value = transaction;
  manualMatchSuggestion.value = null;
  isLoadingSuggestions.value = true;
  feeSearch.value = '';
  availableFees.value = [];
  showAllFees.value = false;

  try {
    const suggestion = await api.getTransactionSuggestions(transaction.id);
    manualMatchSuggestion.value = suggestion;
  } catch (error) {
    console.error('Failed to load suggestions:', error);
  } finally {
    isLoadingSuggestions.value = false;
  }

  await loadAvailableFees();
}

async function loadAvailableFees(): Promise<void> {
  isLoadingFees.value = true;
  try {
    const suggestedChildId = manualMatchSuggestion.value?.child?.id;

    const generalResponse = await api.getFees({
      search: feeSearch.value || undefined,
      limit: 50,
    });
    let fees = generalResponse.data.filter(f => !f.isPaid);

    if (suggestedChildId && !feeSearch.value) {
      const childFeesResponse = await api.getFees({
        childId: suggestedChildId,
        limit: 50,
      });
      const childFees = childFeesResponse.data.filter(f => !f.isPaid);

      const feeIds = new Set(fees.map(f => f.id));
      for (const fee of childFees) {
        if (!feeIds.has(fee.id)) {
          fees.push(fee);
        }
      }
    }

    availableFees.value = fees;
  } catch (error) {
    console.error('Failed to load fees:', error);
  } finally {
    isLoadingFees.value = false;
  }
}

function closeManualMatch(): void {
  manualMatchTransaction.value = null;
  manualMatchSuggestion.value = null;
  availableFees.value = [];
  feeSearch.value = '';
  showAllFees.value = false;
}

function handleKeydown(e: KeyboardEvent): void {
  if (e.key === 'Escape' && manualMatchTransaction.value) {
    closeManualMatch();
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown);
});

async function confirmManualMatch(expectationId: string): Promise<void> {
  if (!manualMatchTransaction.value) return;

  isCreatingMatch.value = true;
  try {
    await api.createManualMatch(manualMatchTransaction.value.id, expectationId);
    unmatchedTransactions.value = unmatchedTransactions.value.filter(
      tx => tx.id !== manualMatchTransaction.value?.id
    );
    unmatchedTotal.value = Math.max(0, unmatchedTotal.value - 1);
    closeManualMatch();
  } catch (error) {
    console.error('Failed to create match:', error);
    uploadError.value = error instanceof Error ? error.message : 'Zuordnung fehlgeschlagen';
  } finally {
    isCreatingMatch.value = false;
  }
}

type ScoredFee = {
  fee: FeeExpectation;
  confidence: number;
};

function computeFeeConfidence(fee: FeeExpectation): number {
  const tx = manualMatchTransaction.value;
  if (!tx) return 0;

  const suggestion = manualMatchSuggestion.value;
  const suggestionConfidence = typeof suggestion?.confidence === 'number' ? suggestion.confidence : 0;

  if (suggestion?.expectation?.id && suggestion.expectation.id === fee.id) {
    return 0.99;
  }
  if (suggestion?.expectations?.some(expectation => expectation.id === fee.id)) {
    return 0.99;
  }

  const amountMatches = Math.abs((fee.amount || 0) - tx.amount) < 0.01;
  const amountScore = amountMatches ? 0.25 : 0;
  const typeScore = suggestion?.detectedType && fee.feeType === suggestion.detectedType ? 0.1 : 0;
  const childScore = suggestion?.child?.id && fee.child?.id === suggestion.child.id ? suggestionConfidence * 0.6 : 0;

  return Math.min(amountScore + typeScore + childScore, 0.99);
}

const scoredFees = computed<ScoredFee[]>(() =>
  availableFees.value.map(fee => ({
    fee,
    confidence: computeFeeConfidence(fee),
  }))
);

const searchedFees = computed<ScoredFee[]>(() => {
  if (!feeSearch.value.trim()) return scoredFees.value;
  const search = feeSearch.value.toLowerCase();
  return scoredFees.value.filter(({ fee }) => {
    const childName = `${fee.child?.firstName || ''} ${fee.child?.lastName || ''}`.toLowerCase();
    const feeType = getFeeTypeName(fee.feeType).toLowerCase();
    return childName.includes(search) || feeType.includes(search);
  });
});

// Collect IDs of fees already shown in the suggestion section
const suggestionFeeIds = computed<Set<string>>(() => {
  const ids = new Set<string>();
  if (manualMatchSuggestion.value?.expectation?.id) {
    ids.add(manualMatchSuggestion.value.expectation.id);
  }
  if (manualMatchSuggestion.value?.expectations) {
    for (const exp of manualMatchSuggestion.value.expectations) {
      ids.add(exp.id);
    }
  }
  return ids;
});

// Filter out suggestion fees first, then apply confidence filter
const feesWithoutSuggestion = computed<ScoredFee[]>(() =>
  searchedFees.value.filter(candidate => !suggestionFeeIds.value.has(candidate.fee.id))
);

const highConfidenceFees = computed<ScoredFee[]>(() =>
  feesWithoutSuggestion.value.filter(candidate => candidate.confidence >= PREFILTER_CONFIDENCE)
);

const isConfidencePrefiltered = computed(() => {
  const hasSuggestionConfidence = (manualMatchSuggestion.value?.confidence || 0) > 0;
  return !showAllFees.value && !feeSearch.value.trim() && hasSuggestionConfidence && highConfidenceFees.value.length > 0;
});

const displayedFeeCandidates = computed<ScoredFee[]>(() => {
  const source = isConfidencePrefiltered.value ? highConfidenceFees.value : feesWithoutSuggestion.value;

  return [...source].sort((a, b) => {
    if (a.confidence !== b.confidence) return b.confidence - a.confidence;
    const dueA = a.fee.dueDate ? new Date(a.fee.dueDate).getTime() : Number.MAX_SAFE_INTEGER;
    const dueB = b.fee.dueDate ? new Date(b.fee.dueDate).getTime() : Number.MAX_SAFE_INTEGER;
    if (dueA !== dueB) return dueA - dueB;
    const lastA = a.fee.child?.lastName || '';
    const lastB = b.fee.child?.lastName || '';
    return lastA.localeCompare(lastB);
  });
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
}

function cancelDismiss(): void {
  dismissConfirmId.value = null;
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
    const result = await api.dismissTransaction(transaction.id);
    unmatchedTransactions.value = unmatchedTransactions.value.filter(
      tx => tx.payerIban !== transaction.payerIban
    );
    unmatchedTotal.value = Math.max(0, unmatchedTotal.value - result.transactionsRemoved);
    if (activeTab.value === 'blacklist') {
      loadBlacklist();
    }
  } catch (error) {
    console.error('Failed to dismiss transaction:', error);
    uploadError.value = error instanceof Error ? error.message : 'Ignorieren fehlgeschlagen';
  } finally {
    isDismissing.value = null;
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
    await loadMatched();
    await loadUnmatched();
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

async function removeFromBlacklist(iban: string): Promise<void> {
  try {
    await api.removeFromBlacklist(iban);
    blacklistedIBANs.value = blacklistedIBANs.value.filter(item => item.iban !== iban);
    blacklistTotal.value = Math.max(0, blacklistTotal.value - 1);
  } catch (error) {
    console.error('Failed to remove from blacklist:', error);
    uploadError.value = error instanceof Error ? error.message : 'Entfernen fehlgeschlagen';
  }
}

function switchTab(tab: TabType): void {
  activeTab.value = tab;

  const tabActions: Record<TabType, (() => Promise<void>) | null> = {
    upload: null,
    history: loadHistory,
    unmatched: loadUnmatched,
    matched: loadMatched,
    warnings: loadWarnings,
    blacklist: loadBlacklist,
  };

  const loadFn = tabActions[tab];
  if (loadFn) {
    loadFn();
  }
}

onMounted(() => {
  // Check for query param to auto-switch tab
  const tabParam = route.query.tab as TabType | undefined;
  if (tabParam && ['upload', 'history', 'unmatched', 'matched', 'warnings', 'blacklist'].includes(tabParam)) {
    activeTab.value = tabParam;
    // Load the tab's data
    switchTab(tabParam);
  } else {
    // Default behavior: pre-load counts in background
    loadHistory();
    loadWarnings();
    loadUnmatched();
  }
});

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('de-DE');
}

function formatDateTime(dateStr: string): string {
  return new Date(dateStr).toLocaleString('de-DE');
}

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: 'EUR',
  }).format(amount);
}

function getMonthName(month?: number): string {
  if (!month) return '';
  return new Date(2000, month - 1).toLocaleString('de-DE', { month: 'long' });
}

function getConfidenceColor(confidence: number): string {
  if (confidence >= 0.8) return 'text-green-600 bg-green-100';
  if (confidence >= 0.5) return 'text-amber-600 bg-amber-100';
  return 'text-red-600 bg-red-100';
}

function getConfidenceLabel(confidence: number): string {
  if (confidence >= 0.8) return 'Hoch';
  if (confidence >= 0.5) return 'Mittel';
  return 'Niedrig';
}

function getFeeTypeName(type?: string): string {
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
      return type || 'Unbekannt';
  }
}

function getFeeTypeColor(type?: string): string {
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

function resetUpload(): void {
  importResult.value = null;
  uploadError.value = null;
  confirmResult.value = null;
  selectedMatches.value.clear();
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
    default:
      return 'bg-gray-100 text-gray-700';
  }
}
</script>

<template>
  <div>
    <!-- Header -->
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-gray-900">CSV-Import</h1>
      <p class="text-gray-600 mt-1">
        Kontoauszüge importieren und Zahlungen zuordnen
      </p>
    </div>

    <!-- Tabs -->
    <div class="flex border-b mb-6">
      <button
        @click="switchTab('upload')"
        :class="[
          'px-4 py-2 text-sm font-medium border-b-2 transition-colors',
          activeTab === 'upload'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-600 hover:text-gray-900',
        ]"
      >
        <div class="flex items-center gap-2">
          <Upload class="h-4 w-4" />
          Upload
        </div>
      </button>
      <button
        @click="switchTab('history')"
        :class="[
          'px-4 py-2 text-sm font-medium border-b-2 transition-colors',
          activeTab === 'history'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-600 hover:text-gray-900',
        ]"
      >
        <div class="flex items-center gap-2">
          <History class="h-4 w-4" />
          Historie
        </div>
      </button>
      <button
        @click="switchTab('unmatched')"
        :class="[
          'px-4 py-2 text-sm font-medium border-b-2 transition-colors',
          activeTab === 'unmatched'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-600 hover:text-gray-900',
        ]"
      >
        <div class="flex items-center gap-2">
          <Link2 class="h-4 w-4" />
          Nicht zugeordnet
          <span v-if="unmatchedTotal > 0" class="px-1.5 py-0.5 text-xs bg-amber-100 text-amber-700 rounded-full">
            {{ unmatchedTotal }}
          </span>
        </div>
      </button>
      <button
        @click="switchTab('matched')"
        :class="[
          'px-4 py-2 text-sm font-medium border-b-2 transition-colors',
          activeTab === 'matched'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-600 hover:text-gray-900',
        ]"
      >
        <div class="flex items-center gap-2">
          <CheckCircle class="h-4 w-4" />
          Zugeordnet
        </div>
      </button>
      <button
        @click="switchTab('warnings')"
        :class="[
          'px-4 py-2 text-sm font-medium border-b-2 transition-colors',
          activeTab === 'warnings'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-600 hover:text-gray-900',
        ]"
      >
        <div class="flex items-center gap-2">
          <AlertTriangle class="h-4 w-4" />
          Warnungen
          <span v-if="warningsTotal > 0" class="px-1.5 py-0.5 text-xs bg-orange-100 text-orange-700 rounded-full">
            {{ warningsTotal }}
          </span>
        </div>
      </button>
      <button
        @click="switchTab('blacklist')"
        :class="[
          'px-4 py-2 text-sm font-medium border-b-2 transition-colors',
          activeTab === 'blacklist'
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-600 hover:text-gray-900',
        ]"
      >
        <div class="flex items-center gap-2">
          <Ban class="h-4 w-4" />
          Blacklist
          <span v-if="blacklistTotal > 0" class="px-1.5 py-0.5 text-xs bg-gray-100 text-gray-700 rounded-full">
            {{ blacklistTotal }}
          </span>
        </div>
      </button>
    </div>

    <!-- Upload Tab -->
    <div v-if="activeTab === 'upload'">
      <!-- Upload Area -->
      <div
        v-if="!importResult"
        @dragover="handleDragOver"
        @dragleave="handleDragLeave"
        @drop="handleDrop"
        :class="[
          'border-2 border-dashed rounded-xl p-12 text-center transition-colors',
          isDragging
            ? 'border-primary bg-primary/5'
            : 'border-gray-300 hover:border-gray-400',
          isUploading ? 'opacity-50 pointer-events-none' : '',
        ]"
      >
        <div v-if="isUploading" class="flex flex-col items-center gap-4">
          <Loader2 class="h-12 w-12 animate-spin text-primary" />
          <p class="text-gray-600">CSV wird verarbeitet...</p>
        </div>
        <div v-else class="flex flex-col items-center gap-4">
          <div class="p-4 bg-gray-100 rounded-full">
            <FileSpreadsheet class="h-12 w-12 text-gray-400" />
          </div>
          <div>
            <p class="text-lg font-medium text-gray-700">
              CSV-Datei hierher ziehen
            </p>
            <p class="text-sm text-gray-500 mt-1">
              oder klicken um eine Datei auszuwählen
            </p>
          </div>
          <input
            type="file"
            accept=".csv"
            @change="handleFileSelect"
            class="hidden"
            id="file-input"
          />
          <label
            for="file-input"
            class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 cursor-pointer transition-colors"
          >
            <Upload class="h-4 w-4" />
            Datei auswählen
          </label>
          <p class="text-xs text-gray-400 mt-2">
            Unterstützt: Deutsche Bankexporte (CSV, Semikolon-getrennt, ISO-8859-1 oder UTF-8)
          </p>
        </div>
      </div>

      <!-- Upload Error -->
      <div
        v-if="uploadError"
        class="mt-4 p-4 bg-red-50 border border-red-200 rounded-lg flex items-start gap-3"
      >
        <XCircle class="h-5 w-5 text-red-500 flex-shrink-0 mt-0.5" />
        <div>
          <p class="text-red-700 font-medium">Fehler beim Upload</p>
          <p class="text-sm text-red-600">{{ uploadError }}</p>
        </div>
      </div>

      <!-- Import Result -->
      <div v-if="importResult" class="space-y-6">
        <!-- Summary Card -->
        <div class="bg-white rounded-xl border p-6">
          <div class="flex items-center justify-between mb-4">
            <div class="flex items-center gap-3">
              <div class="p-2 bg-green-100 rounded-lg">
                <CheckCircle class="h-6 w-6 text-green-600" />
              </div>
              <div>
                <h2 class="text-lg font-semibold">Import erfolgreich</h2>
                <p class="text-sm text-gray-600">{{ importResult.fileName }}</p>
              </div>
            </div>
            <button
              @click="resetUpload"
              class="text-sm text-gray-600 hover:text-gray-900 underline"
            >
              Neuer Import
            </button>
          </div>
          <div class="grid grid-cols-3 gap-4">
            <div class="p-3 bg-gray-50 rounded-lg text-center">
              <div class="text-2xl font-bold text-gray-900">
                {{ importResult.totalRows }}
              </div>
              <div class="text-sm text-gray-600">Zeilen gelesen</div>
            </div>
            <div class="p-3 bg-green-50 rounded-lg text-center">
              <div class="text-2xl font-bold text-green-600">
                {{ importResult.imported }}
              </div>
              <div class="text-sm text-gray-600">Importiert</div>
            </div>
            <div class="p-3 bg-gray-50 rounded-lg text-center">
              <div class="text-2xl font-bold text-gray-500">
                {{ importResult.skipped }}
              </div>
              <div class="text-sm text-gray-600">Übersprungen</div>
            </div>
          </div>
        </div>

        <!-- Confirm Result -->
        <div
          v-if="confirmResult"
          class="p-4 bg-green-50 border border-green-200 rounded-lg flex items-start gap-3"
        >
          <CheckCircle class="h-5 w-5 text-green-500 flex-shrink-0 mt-0.5" />
          <div>
            <p class="text-green-700 font-medium">Zuordnungen bestätigt</p>
            <p class="text-sm text-green-600">
              {{ confirmResult.confirmed }} Zahlungen wurden als bezahlt markiert
              <span v-if="confirmResult.failed > 0">, {{ confirmResult.failed }} fehlgeschlagen</span>
            </p>
          </div>
        </div>

        <!-- Match Suggestions -->
        <div v-if="matchableSuggestions.length > 0" class="bg-white rounded-xl border">
          <div class="p-4 border-b flex items-center justify-between">
            <div>
              <h3 class="font-semibold">Zuordnungsvorschläge</h3>
              <p class="text-sm text-gray-600">
                {{ selectedMatches.size }} von {{ matchableSuggestions.length }} ausgewählt
              </p>
            </div>
            <div class="flex gap-2">
              <button
                @click="selectAllMatches"
                class="text-sm text-primary hover:underline"
              >
                Alle auswählen
              </button>
              <span class="text-gray-300">|</span>
              <button
                @click="deselectAllMatches"
                class="text-sm text-gray-600 hover:underline"
              >
                Keine
              </button>
            </div>
          </div>
          
          <div class="divide-y">
            <div
              v-for="suggestion in matchableSuggestions"
              :key="suggestion.transaction.id"
              class="p-4"
            >
              <div class="flex items-start gap-3">
                <!-- Checkbox -->
                <button
                  @click="toggleMatch(suggestion.transaction.id)"
                  :class="[
                    'mt-1 w-5 h-5 rounded border flex items-center justify-center flex-shrink-0 transition-colors',
                    selectedMatches.has(suggestion.transaction.id)
                      ? 'bg-primary border-primary text-white'
                      : 'border-gray-300 hover:border-gray-400',
                  ]"
                >
                  <Check v-if="selectedMatches.has(suggestion.transaction.id)" class="h-3 w-3" />
                </button>

                <!-- Main Content -->
                <div class="flex-1 min-w-0">
                  <div class="flex items-start justify-between gap-4">
                    <div>
                      <div class="font-medium">
                        {{ suggestion.transaction.payerName || 'Unbekannt' }}
                      </div>
                      <div class="text-sm text-gray-600 truncate">
                        {{ suggestion.transaction.description }}
                      </div>
                    </div>
                    <div class="text-right flex-shrink-0">
                      <div class="font-semibold text-green-600">
                        {{ formatCurrency(suggestion.transaction.amount) }}
                      </div>
                      <div class="text-xs text-gray-500">
                        {{ formatDate(suggestion.transaction.bookingDate) }}
                      </div>
                    </div>
                  </div>

                  <!-- Match Details -->
                  <div class="mt-3 flex items-center gap-4 text-sm">
                    <span
                      :class="[
                        'px-2 py-0.5 rounded-full text-xs font-medium',
                        getConfidenceColor(suggestion.confidence),
                      ]"
                    >
                      {{ getConfidenceLabel(suggestion.confidence) }} ({{ Math.round(suggestion.confidence * 100) }}%)
                    </span>
                    <span class="text-gray-500">
                      Erkannt als: {{ getFeeTypeName(suggestion.detectedType) }}
                    </span>
                    <span class="text-gray-500">
                      Grund: {{ suggestion.matchedBy }}
                    </span>
                  </div>

                  <!-- Expand Details -->
                  <button
                    @click="toggleSuggestion(suggestion.transaction.id)"
                    class="mt-2 text-sm text-primary flex items-center gap-1"
                  >
                    <ChevronDown
                      v-if="!expandedSuggestions.has(suggestion.transaction.id)"
                      class="h-4 w-4"
                    />
                    <ChevronUp v-else class="h-4 w-4" />
                    {{ expandedSuggestions.has(suggestion.transaction.id) ? 'Weniger' : 'Details' }}
                  </button>

                  <div
                    v-if="expandedSuggestions.has(suggestion.transaction.id)"
                    class="mt-3 p-3 bg-gray-50 rounded-lg text-sm space-y-2"
                  >
                    <div class="grid grid-cols-2 gap-4">
                      <div>
                        <span class="text-gray-500">Zugeordnetes Kind:</span>
                        <span class="ml-2 font-medium">
                          {{ suggestion.child?.firstName }} {{ suggestion.child?.lastName }}
                        </span>
                      </div>
                      <div>
                        <span class="text-gray-500">Beitragsart:</span>
                        <span class="ml-2 font-medium">
                          {{ getFeeTypeName(suggestion.expectation?.feeType) }}
                        </span>
                      </div>
                      <div>
                        <span class="text-gray-500">Erwarteter Betrag:</span>
                        <span class="ml-2 font-medium">
                          {{ formatCurrency(suggestion.expectation?.amount || 0) }}
                        </span>
                      </div>
                      <div>
                        <span class="text-gray-500">Zeitraum:</span>
                        <span class="ml-2 font-medium">
                          {{ suggestion.expectation?.month ? suggestion.expectation.month + '/' : '' }}{{ suggestion.expectation?.year }}
                        </span>
                      </div>
                    </div>
                    <div v-if="suggestion.transaction.payerIban">
                      <span class="text-gray-500">IBAN:</span>
                      <span class="ml-2 font-mono text-xs">
                        {{ suggestion.transaction.payerIban }}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Confirm Button -->
          <div class="p-4 border-t bg-gray-50 flex items-center justify-between">
            <p class="text-sm text-gray-600">
              {{ selectedMatches.size }} Zuordnungen ausgewählt
            </p>
            <button
              @click="confirmMatches"
              :disabled="selectedMatches.size === 0 || isConfirming"
              class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Loader2 v-if="isConfirming" class="h-4 w-4 animate-spin" />
              <CheckCircle v-else class="h-4 w-4" />
              Zuordnungen bestätigen
            </button>
          </div>
        </div>

        <!-- Unmatched from this import -->
        <div v-if="unmatchableSuggestions.length > 0" class="bg-white rounded-xl border">
          <div class="p-4 border-b">
            <div class="flex items-center gap-2">
              <AlertTriangle class="h-5 w-5 text-amber-500" />
              <h3 class="font-semibold">Nicht zuordenbar</h3>
            </div>
            <p class="text-sm text-gray-600 mt-1">
              Diese Transaktionen konnten keinem offenen Beitrag zugeordnet werden
            </p>
          </div>
          
          <div class="divide-y">
            <div
              v-for="suggestion in unmatchableSuggestions"
              :key="suggestion.transaction.id"
              class="p-4 flex items-center justify-between"
            >
              <div>
                <div class="font-medium">
                  {{ suggestion.transaction.payerName || 'Unbekannt' }}
                </div>
                <div class="text-sm text-gray-600 truncate max-w-md">
                  {{ suggestion.transaction.description }}
                </div>
              </div>
              <div class="text-right">
                <div class="font-semibold">
                  {{ formatCurrency(suggestion.transaction.amount) }}
                </div>
                <div class="text-xs text-gray-500">
                  {{ formatDate(suggestion.transaction.bookingDate) }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- History Tab -->
    <div v-if="activeTab === 'history'">
      <div v-if="isLoadingHistory" class="flex items-center justify-center py-12">
        <Loader2 class="h-8 w-8 animate-spin text-primary" />
      </div>

      <div v-else-if="importHistory.length === 0" class="text-center py-12">
        <History class="h-12 w-12 text-gray-300 mx-auto mb-4" />
        <p class="text-gray-600">Noch keine Importe durchgeführt</p>
      </div>

      <div v-else class="bg-white rounded-xl border overflow-hidden">
        <table class="w-full">
          <thead class="bg-gray-50">
            <tr class="text-left text-sm text-gray-500">
              <th class="px-4 py-3 font-medium">Datei</th>
              <th class="px-4 py-3 font-medium">Zeitraum</th>
              <th class="px-4 py-3 font-medium">Transaktionen</th>
              <th class="px-4 py-3 font-medium">Zugeordnet</th>
              <th class="px-4 py-3 font-medium">Importiert am</th>
              <th class="px-4 py-3 font-medium">Von</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="batch in importHistory"
              :key="batch.id"
              class="border-t hover:bg-gray-50"
            >
              <td class="px-4 py-3">
                <div class="flex items-center gap-2">
                  <FileSpreadsheet class="h-4 w-4 text-gray-400" />
                  <span class="font-medium">{{ batch.fileName }}</span>
                </div>
              </td>
              <td class="px-4 py-3 text-gray-600 text-sm">
                <span v-if="batch.dateFrom && batch.dateTo">
                  {{ formatDate(batch.dateFrom) }} - {{ formatDate(batch.dateTo) }}
                </span>
                <span v-else class="text-gray-400">-</span>
              </td>
              <td class="px-4 py-3">{{ batch.transactionCount }}</td>
              <td class="px-4 py-3">
                <span
                  :class="[
                    'px-2 py-0.5 rounded-full text-xs font-medium',
                    batch.matchedCount === batch.transactionCount
                      ? 'bg-green-100 text-green-700'
                      : batch.matchedCount > 0
                        ? 'bg-amber-100 text-amber-700'
                        : 'bg-gray-100 text-gray-700',
                  ]"
                >
                  {{ batch.matchedCount }} / {{ batch.transactionCount }}
                </span>
              </td>
              <td class="px-4 py-3 text-gray-600">
                {{ formatDateTime(batch.importedAt) }}
              </td>
              <td class="px-4 py-3 text-gray-600">
                {{ batch.importedByEmail || batch.importedBy }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Unmatched Tab -->
    <div v-if="activeTab === 'unmatched'">
      <div class="flex items-center justify-between mb-4">
        <p class="text-sm text-gray-600">
          {{ unmatchedTotal }} nicht zugeordnete Transaktionen
        </p>
        <div class="flex items-center gap-2">
          <button
            @click="rescanTransactions"
            :disabled="isRescanning || unmatchedTotal === 0"
            class="inline-flex items-center gap-1 px-3 py-1.5 text-sm bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <Loader2 v-if="isRescanning" class="h-4 w-4 animate-spin" />
            <RefreshCw v-else class="h-4 w-4" />
            Erneut zuordnen
          </button>
          <button
            @click="loadUnmatched"
            class="inline-flex items-center gap-1 text-sm text-primary hover:underline"
          >
            <RefreshCw class="h-4 w-4" />
            Aktualisieren
          </button>
        </div>
      </div>

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

      <!-- Search Bar -->
      <div class="mb-4">
        <div class="relative">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
          <input
            v-model="transactionSearch"
            type="text"
            placeholder="Suche nach Zahler oder Beschreibung..."
            class="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          />
        </div>
      </div>

      <div v-if="isLoadingUnmatched" class="flex items-center justify-center py-12">
        <Loader2 class="h-8 w-8 animate-spin text-primary" />
      </div>

      <div v-else-if="unmatchedTransactions.length === 0" class="text-center py-12">
        <component :is="debouncedTransactionSearch ? Search : CheckCircle"
                   :class="debouncedTransactionSearch ? 'h-12 w-12 text-gray-300 mx-auto mb-4' : 'h-12 w-12 text-green-300 mx-auto mb-4'" />
        <p class="text-gray-600">
          {{ debouncedTransactionSearch ? 'Keine Transaktionen gefunden' : 'Alle Transaktionen sind zugeordnet' }}
        </p>
        <p v-if="debouncedTransactionSearch" class="text-sm text-gray-500 mt-1">Versuche einen anderen Suchbegriff</p>
      </div>

      <div v-else class="bg-white rounded-xl border overflow-hidden">
        <table class="w-full">
          <thead class="bg-gray-50">
            <tr class="text-left text-sm text-gray-500">
              <th
                class="px-4 py-3 font-medium cursor-pointer hover:bg-gray-100 select-none"
                @click="toggleUnmatchedSort('date')"
              >
                <div class="flex items-center gap-1">
                  Datum
                  <ArrowUp v-if="unmatchedSortField === 'date' && unmatchedSortDirection === 'asc'" class="h-4 w-4" />
                  <ArrowDown v-else-if="unmatchedSortField === 'date' && unmatchedSortDirection === 'desc'" class="h-4 w-4" />
                  <ArrowUpDown v-else class="h-4 w-4 text-gray-400" />
                </div>
              </th>
              <th
                class="px-4 py-3 font-medium cursor-pointer hover:bg-gray-100 select-none"
                @click="toggleUnmatchedSort('payer')"
              >
                <div class="flex items-center gap-1">
                  Zahler
                  <ArrowUp v-if="unmatchedSortField === 'payer' && unmatchedSortDirection === 'asc'" class="h-4 w-4" />
                  <ArrowDown v-else-if="unmatchedSortField === 'payer' && unmatchedSortDirection === 'desc'" class="h-4 w-4" />
                  <ArrowUpDown v-else class="h-4 w-4 text-gray-400" />
                </div>
              </th>
              <th
                class="px-4 py-3 font-medium cursor-pointer hover:bg-gray-100 select-none"
                @click="toggleUnmatchedSort('description')"
              >
                <div class="flex items-center gap-1">
                  Beschreibung
                  <ArrowUp v-if="unmatchedSortField === 'description' && unmatchedSortDirection === 'asc'" class="h-4 w-4" />
                  <ArrowDown v-else-if="unmatchedSortField === 'description' && unmatchedSortDirection === 'desc'" class="h-4 w-4" />
                  <ArrowUpDown v-else class="h-4 w-4 text-gray-400" />
                </div>
              </th>
              <th
                class="px-4 py-3 font-medium text-right cursor-pointer hover:bg-gray-100 select-none"
                @click="toggleUnmatchedSort('amount')"
              >
                <div class="flex items-center justify-end gap-1">
                  Betrag
                  <ArrowUp v-if="unmatchedSortField === 'amount' && unmatchedSortDirection === 'asc'" class="h-4 w-4" />
                  <ArrowDown v-else-if="unmatchedSortField === 'amount' && unmatchedSortDirection === 'desc'" class="h-4 w-4" />
                  <ArrowUpDown v-else class="h-4 w-4 text-gray-400" />
                </div>
              </th>
              <th class="px-4 py-3 font-medium text-right">Aktionen</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="tx in unmatchedTransactions"
              :key="tx.id"
              class="border-t hover:bg-gray-50"
            >
              <td class="px-4 py-3 text-gray-600">
                {{ formatDate(tx.bookingDate) }}
              </td>
              <td class="px-4 py-3">
                <div class="font-medium">{{ tx.payerName || 'Unbekannt' }}</div>
                <div v-if="tx.payerIban" class="text-xs text-gray-500 font-mono">
                  {{ tx.payerIban }}
                </div>
              </td>
              <td class="px-4 py-3 text-gray-600 truncate max-w-xs">
                {{ tx.description }}
              </td>
              <td class="px-4 py-3 text-right font-medium">
                {{ formatCurrency(tx.amount) }}
              </td>
              <td class="px-4 py-3 text-right">
                <!-- Confirm Dialog -->
                <div v-if="dismissConfirmId === tx.id" class="flex items-center justify-end gap-2">
                  <span class="text-xs text-gray-500">Ignorieren?</span>
                  <button
                    @click="dismissTransaction(tx)"
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
                <!-- Action Buttons -->
                <div v-else class="flex items-center justify-end gap-2">
                  <!-- Manual Match Button -->
                  <button
                    @click="openManualMatch(tx)"
                    class="inline-flex items-center gap-1 px-2 py-1 text-xs text-primary hover:text-primary/80 hover:bg-primary/10 rounded transition-colors"
                    title="Manuell zuordnen"
                  >
                    <LinkIcon class="h-3 w-3" />
                    Zuordnen
                  </button>
                  <!-- Dismiss Button -->
                  <button
                    @click="showDismissConfirm(tx.id)"
                    :disabled="isDismissing === tx.id"
                    class="inline-flex items-center gap-1 px-2 py-1 text-xs text-gray-600 hover:text-red-600 hover:bg-red-50 rounded transition-colors disabled:opacity-50"
                    title="IBAN dauerhaft ignorieren"
                  >
                    <Loader2 v-if="isDismissing === tx.id" class="h-3 w-3 animate-spin" />
                    <Ban v-else class="h-3 w-3" />
                    Ignorieren
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>

        <!-- Pagination -->
        <div v-if="unmatchedTotalPages > 1" class="flex items-center justify-between px-4 py-3 border-t bg-gray-50">
          <div class="text-sm text-gray-600">
            Seite {{ unmatchedPage }} von {{ unmatchedTotalPages }} ({{ unmatchedTotal }} Einträge)
          </div>
          <div class="flex items-center gap-2">
            <button
              @click="goToUnmatchedPage(unmatchedPage - 1)"
              :disabled="unmatchedPage <= 1"
              class="p-1 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <ChevronLeft class="h-5 w-5" />
            </button>
            <button
              v-for="page in Math.min(5, unmatchedTotalPages)"
              :key="page"
              @click="goToUnmatchedPage(page)"
              :class="[
                'px-3 py-1 rounded text-sm',
                page === unmatchedPage ? 'bg-primary text-white' : 'hover:bg-gray-200'
              ]"
            >
              {{ page }}
            </button>
            <span v-if="unmatchedTotalPages > 5" class="text-gray-400">...</span>
            <button
              @click="goToUnmatchedPage(unmatchedPage + 1)"
              :disabled="unmatchedPage >= unmatchedTotalPages"
              class="p-1 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <ChevronRight class="h-5 w-5" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Warnings Tab -->
    <div v-if="activeTab === 'warnings'">
      <div class="flex items-center justify-between mb-4">
        <div>
          <p class="text-sm text-gray-600">
            {{ warningsTotal }} offene Warnungen
          </p>
          <p class="text-xs text-gray-500 mt-1">
            Warnungen zu Zahlungen, die manuell uberpruft werden sollten
          </p>
        </div>
        <button
          @click="loadWarnings"
          class="inline-flex items-center gap-1 text-sm text-primary hover:underline"
        >
          <RefreshCw class="h-4 w-4" />
          Aktualisieren
        </button>
      </div>

      <div v-if="isLoadingWarnings" class="flex items-center justify-center py-12">
        <Loader2 class="h-8 w-8 animate-spin text-primary" />
      </div>

      <div v-else-if="warnings.length === 0" class="text-center py-12">
        <CheckCircle class="h-12 w-12 text-green-300 mx-auto mb-4" />
        <p class="text-gray-600">Keine offenen Warnungen</p>
        <p class="text-sm text-gray-500 mt-1">
          Alle Zahlungen wurden korrekt verarbeitet
        </p>
      </div>

      <div v-else class="space-y-4">
        <div
          v-for="warning in warnings"
          :key="warning.id"
          class="bg-white rounded-xl border p-4"
        >
          <div class="flex items-start justify-between gap-4">
            <!-- Warning Info -->
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
                <!-- Child Name Badge -->
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
              
              <!-- Transaction Details -->
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

              <!-- Matched Fee Details (for LATE_PAYMENT) -->
              <div v-if="warning.warningType === 'LATE_PAYMENT' && warning.matchedFee" class="mt-2 p-3 bg-orange-50 rounded-lg text-sm space-y-1">
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
                  <span>{{ getMonthName(warning.matchedFee.month) }} {{ warning.matchedFee.year }}</span>
                </div>
                <div class="flex justify-between">
                  <span class="text-gray-500">Betrag:</span>
                  <span class="font-medium">{{ formatCurrency(warning.matchedFee.amount) }}</span>
                </div>
              </div>
            </div>

            <!-- Actions -->
            <div class="flex flex-col gap-2">
              <!-- Late Payment: Create Late Fee Button -->
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

              <!-- Dismiss Dialog -->
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

              <!-- Dismiss Button -->
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
    </div>

    <!-- Blacklist Tab -->
    <div v-if="activeTab === 'blacklist'">
      <div class="flex items-center justify-between mb-4">
        <div>
          <p class="text-sm text-gray-600">
            {{ blacklistTotal }} ignorierte IBANs
          </p>
          <p class="text-xs text-gray-500 mt-1">
            Transaktionen von diesen IBANs werden beim Import automatisch ignoriert
          </p>
        </div>
        <button
          @click="loadBlacklist"
          class="inline-flex items-center gap-1 text-sm text-primary hover:underline"
        >
          <RefreshCw class="h-4 w-4" />
          Aktualisieren
        </button>
      </div>

      <div v-if="isLoadingBlacklist" class="flex items-center justify-center py-12">
        <Loader2 class="h-8 w-8 animate-spin text-primary" />
      </div>

      <div v-else-if="blacklistedIBANs.length === 0" class="text-center py-12">
        <ShieldOff class="h-12 w-12 text-gray-300 mx-auto mb-4" />
        <p class="text-gray-600">Keine IBANs auf der Blacklist</p>
        <p class="text-sm text-gray-500 mt-1">
          Klicken Sie bei nicht zugeordneten Transaktionen auf "Ignorieren", um IBANs zur Blacklist hinzuzufugen
        </p>
      </div>

      <div v-else class="bg-white rounded-xl border overflow-hidden">
        <table class="w-full">
          <thead class="bg-gray-50">
            <tr class="text-left text-sm text-gray-500">
              <th class="px-4 py-3 font-medium">IBAN</th>
              <th class="px-4 py-3 font-medium">Zahler</th>
              <th class="px-4 py-3 font-medium">Letzte Transaktion</th>
              <th class="px-4 py-3 font-medium">Hinzugefugt am</th>
              <th class="px-4 py-3 font-medium text-right">Aktionen</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="item in blacklistedIBANs"
              :key="item.iban"
              class="border-t hover:bg-gray-50"
            >
              <td class="px-4 py-3">
                <span class="font-mono text-sm">{{ item.iban }}</span>
              </td>
              <td class="px-4 py-3">
                <span class="font-medium">{{ item.payerName || 'Unbekannt' }}</span>
              </td>
              <td class="px-4 py-3 text-gray-600">
                <div v-if="item.originalDescription" class="truncate max-w-xs text-sm">
                  {{ item.originalDescription }}
                </div>
                <div v-if="item.originalAmount" class="text-xs text-gray-500">
                  {{ formatCurrency(item.originalAmount) }}
                </div>
              </td>
              <td class="px-4 py-3 text-gray-600 text-sm">
                {{ formatDate(item.createdAt) }}
              </td>
              <td class="px-4 py-3 text-right">
                <button
                  @click="removeFromBlacklist(item.iban)"
                  class="inline-flex items-center gap-1 px-2 py-1 text-xs text-gray-600 hover:text-green-600 hover:bg-green-50 rounded transition-colors"
                  title="Von Blacklist entfernen"
                >
                  <Trash2 class="h-3 w-3" />
                  Entfernen
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Matched Tab -->
    <div v-if="activeTab === 'matched'">
      <div class="flex items-center justify-between mb-4">
        <p class="text-sm text-gray-600">
          {{ matchedTotal }} zugeordnete Transaktionen
        </p>
        <button
          @click="loadMatched"
          class="inline-flex items-center gap-1 text-sm text-primary hover:underline"
        >
          <RefreshCw class="h-4 w-4" />
          Aktualisieren
        </button>
      </div>

      <!-- Search Bar -->
      <div class="mb-4">
        <div class="relative">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
          <input
            v-model="transactionSearch"
            type="text"
            placeholder="Suche nach Zahler oder Beschreibung..."
            class="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          />
        </div>
      </div>

      <div v-if="isLoadingMatched" class="flex items-center justify-center py-12">
        <Loader2 class="h-8 w-8 animate-spin text-primary" />
      </div>

      <div v-else-if="matchedTransactions.length === 0" class="text-center py-12">
        <component :is="debouncedTransactionSearch ? Search : Link2"
                   class="h-12 w-12 text-gray-300 mx-auto mb-4" />
        <p class="text-gray-600">
          {{ debouncedTransactionSearch ? 'Keine Transaktionen gefunden' : 'Noch keine zugeordneten Transaktionen' }}
        </p>
        <p v-if="debouncedTransactionSearch" class="text-sm text-gray-500 mt-1">Versuche einen anderen Suchbegriff</p>
      </div>

      <div v-else class="bg-white rounded-xl border overflow-hidden">
        <table class="w-full">
          <thead class="bg-gray-50">
            <tr class="text-left text-sm text-gray-500">
              <th
                class="px-4 py-3 font-medium cursor-pointer hover:bg-gray-100 select-none"
                @click="toggleMatchedSort('date')"
              >
                <div class="flex items-center gap-1">
                  Datum
                  <ArrowUp v-if="matchedSortField === 'date' && matchedSortDirection === 'asc'" class="h-4 w-4" />
                  <ArrowDown v-else-if="matchedSortField === 'date' && matchedSortDirection === 'desc'" class="h-4 w-4" />
                  <ArrowUpDown v-else class="h-4 w-4 text-gray-400" />
                </div>
              </th>
              <th
                class="px-4 py-3 font-medium cursor-pointer hover:bg-gray-100 select-none"
                @click="toggleMatchedSort('payer')"
              >
                <div class="flex items-center gap-1">
                  Zahler
                  <ArrowUp v-if="matchedSortField === 'payer' && matchedSortDirection === 'asc'" class="h-4 w-4" />
                  <ArrowDown v-else-if="matchedSortField === 'payer' && matchedSortDirection === 'desc'" class="h-4 w-4" />
                  <ArrowUpDown v-else class="h-4 w-4 text-gray-400" />
                </div>
              </th>
              <th
                class="px-4 py-3 font-medium cursor-pointer hover:bg-gray-100 select-none"
                @click="toggleMatchedSort('description')"
              >
                <div class="flex items-center gap-1">
                  Beschreibung
                  <ArrowUp v-if="matchedSortField === 'description' && matchedSortDirection === 'asc'" class="h-4 w-4" />
                  <ArrowDown v-else-if="matchedSortField === 'description' && matchedSortDirection === 'desc'" class="h-4 w-4" />
                  <ArrowUpDown v-else class="h-4 w-4 text-gray-400" />
                </div>
              </th>
              <th
                class="px-4 py-3 font-medium text-right cursor-pointer hover:bg-gray-100 select-none"
                @click="toggleMatchedSort('amount')"
              >
                <div class="flex items-center justify-end gap-1">
                  Betrag
                  <ArrowUp v-if="matchedSortField === 'amount' && matchedSortDirection === 'asc'" class="h-4 w-4" />
                  <ArrowDown v-else-if="matchedSortField === 'amount' && matchedSortDirection === 'desc'" class="h-4 w-4" />
                  <ArrowUpDown v-else class="h-4 w-4 text-gray-400" />
                </div>
              </th>
              <th class="px-4 py-3 font-medium">Zugeordnete Beiträge</th>
              <th class="px-4 py-3 font-medium text-right">Aktionen</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="tx in matchedTransactions"
              :key="tx.id"
              class="border-t hover:bg-gray-50"
            >
              <td class="px-4 py-3 text-gray-600">
                {{ formatDate(tx.bookingDate) }}
              </td>
              <td class="px-4 py-3">
                <div class="font-medium">{{ tx.payerName || 'Unbekannt' }}</div>
                <div v-if="tx.payerIban" class="text-xs text-gray-500 font-mono">
                  {{ tx.payerIban }}
                </div>
              </td>
              <td class="px-4 py-3 text-gray-600 truncate max-w-xs">
                {{ tx.description }}
              </td>
              <td class="px-4 py-3 text-right font-medium text-green-600">
                {{ formatCurrency(tx.amount) }}
              </td>
              <td class="px-4 py-3">
                <div v-if="tx.matches && tx.matches.length > 0" class="flex flex-wrap gap-1">
                  <span
                    v-for="match in tx.matches"
                    :key="match.id"
                    :class="[
                      'inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium',
                      getFeeTypeColor(match.expectation?.feeType || '')
                    ]"
                  >
                    {{ getFeeTypeName(match.expectation?.feeType || '') }}
                    <span class="ml-1 opacity-75">{{ formatCurrency(match.expectation?.amount || 0) }}</span>
                  </span>
                </div>
                <span v-else class="text-gray-400 text-sm">-</span>
              </td>
              <td class="px-4 py-3 text-right">
                <!-- Confirm Dialogs -->
                <div v-if="unmatchConfirmId === tx.id" class="flex items-center justify-end gap-2">
                  <span class="text-xs text-gray-500">Zuordnung aufheben?</span>
                  <button
                    @click="unmatchTransaction(tx)"
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
                <div v-else-if="deleteConfirmId === tx.id" class="flex items-center justify-end gap-2">
                  <span class="text-xs text-gray-500">Transaktion löschen?</span>
                  <button
                    @click="unmatchTransaction(tx, true)"
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
                <!-- Action Buttons -->
                <div v-else class="flex items-center justify-end gap-2">
                  <button
                    @click="showUnmatchConfirm(tx.id)"
                    :disabled="isUnmatching === tx.id || isDeletingMatched === tx.id"
                    class="inline-flex items-center gap-1 px-2 py-1 text-xs text-gray-600 hover:text-amber-700 hover:bg-amber-50 rounded transition-colors disabled:opacity-50"
                    title="Zuordnung aufheben"
                  >
                    <Loader2 v-if="isUnmatching === tx.id" class="h-3 w-3 animate-spin" />
                    <Unlink v-else class="h-3 w-3" />
                    Aufheben
                  </button>
                  <button
                    @click="showDeleteConfirm(tx.id)"
                    :disabled="isUnmatching === tx.id || isDeletingMatched === tx.id"
                    class="inline-flex items-center gap-1 px-2 py-1 text-xs text-gray-600 hover:text-red-600 hover:bg-red-50 rounded transition-colors disabled:opacity-50"
                    title="Transaktion löschen"
                  >
                    <Loader2 v-if="isDeletingMatched === tx.id" class="h-3 w-3 animate-spin" />
                    <Trash2 v-else class="h-3 w-3" />
                    Löschen
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>

        <!-- Pagination -->
        <div v-if="matchedTotalPages > 1" class="flex items-center justify-between px-4 py-3 border-t bg-gray-50">
          <div class="text-sm text-gray-600">
            Seite {{ matchedPage }} von {{ matchedTotalPages }} ({{ matchedTotal }} Einträge)
          </div>
          <div class="flex items-center gap-2">
            <button
              @click="goToMatchedPage(matchedPage - 1)"
              :disabled="matchedPage <= 1"
              class="p-1 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <ChevronLeft class="h-5 w-5" />
            </button>
            <button
              v-for="page in Math.min(5, matchedTotalPages)"
              :key="page"
              @click="goToMatchedPage(page)"
              :class="[
                'px-3 py-1 rounded text-sm',
                page === matchedPage ? 'bg-primary text-white' : 'hover:bg-gray-200'
              ]"
            >
              {{ page }}
            </button>
            <span v-if="matchedTotalPages > 5" class="text-gray-400">...</span>
            <button
              @click="goToMatchedPage(matchedPage + 1)"
              :disabled="matchedPage >= matchedTotalPages"
              class="p-1 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <ChevronRight class="h-5 w-5" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Manual Match Modal -->
    <div
      v-if="manualMatchTransaction"
      class="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
      @click.self="closeManualMatch"
    >
      <div class="bg-white rounded-xl shadow-xl max-w-3xl w-full mx-4 max-h-[90vh] overflow-hidden flex flex-col">
        <!-- Modal Header -->
        <div class="p-4 border-b">
          <div class="flex items-center justify-between">
            <h2 class="text-lg font-semibold">Transaktion manuell zuordnen</h2>
            <button
              @click="closeManualMatch"
              class="text-gray-400 hover:text-gray-600"
            >
              <XCircle class="h-5 w-5" />
            </button>
          </div>
        </div>

        <!-- Transaction Details -->
        <div class="p-4 bg-gray-50 border-b">
          <div class="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span class="text-gray-500">Zahler:</span>
              <span class="ml-2 font-medium">{{ manualMatchTransaction.payerName || 'Unbekannt' }}</span>
            </div>
            <div>
              <span class="text-gray-500">Betrag:</span>
              <span class="ml-2 font-medium text-green-600">{{ formatCurrency(manualMatchTransaction.amount) }}</span>
            </div>
            <div>
              <span class="text-gray-500">Datum:</span>
              <span class="ml-2">{{ formatDate(manualMatchTransaction.bookingDate) }}</span>
            </div>
            <div v-if="manualMatchTransaction.payerIban">
              <span class="text-gray-500">IBAN:</span>
              <span class="ml-2 font-mono text-xs">{{ manualMatchTransaction.payerIban }}</span>
            </div>
          </div>
          <div v-if="manualMatchTransaction.description" class="mt-2 text-sm text-gray-600">
            {{ manualMatchTransaction.description }}
          </div>
        </div>

        <!-- Suggestion (if available) -->
        <div v-if="isLoadingSuggestions" class="p-4 border-b">
          <div class="flex items-center gap-2 text-gray-500">
            <Loader2 class="h-4 w-4 animate-spin" />
            Lade Vorschlage...
          </div>
        </div>
        <div v-else-if="manualMatchSuggestion?.expectation" class="p-4 border-b">
          <h3 class="text-sm font-medium text-gray-700 mb-2">Vorschlag</h3>
          <div
            class="p-3 bg-green-50 border border-green-200 rounded-lg flex items-center justify-between cursor-pointer hover:bg-green-100"
            @click="confirmManualMatch(manualMatchSuggestion.expectation!.id)"
          >
            <div class="flex-1">
              <div class="font-medium">
                {{ manualMatchSuggestion.child?.firstName }} {{ manualMatchSuggestion.child?.lastName }}
              </div>
              <div class="text-sm text-gray-600">
                {{ getFeeTypeName(manualMatchSuggestion.expectation?.feeType) }}
                - {{ manualMatchSuggestion.expectation?.month }}/{{ manualMatchSuggestion.expectation?.year }}
                - {{ formatCurrency(manualMatchSuggestion.expectation?.amount || 0) }}
              </div>
            </div>
            <div class="flex items-center gap-2">
              <span
                :class="[
                  'px-2 py-0.5 rounded-full text-xs font-medium',
                  getConfidenceColor(manualMatchSuggestion.confidence),
                ]"
              >
                {{ Math.round(manualMatchSuggestion.confidence * 100) }}%
              </span>
              <CheckCircle class="h-5 w-5 text-green-500" />
            </div>
          </div>
        </div>

        <!-- Fee Search -->
        <div class="p-4 border-b">
          <div class="flex items-center justify-between mb-2">
            <h3 class="text-sm font-medium text-gray-700">Offene Beitrage durchsuchen</h3>
            <button
              v-if="manualMatchSuggestion?.confidence && highConfidenceFees.length > 0 && !feeSearch.trim()"
              @click="showAllFees = !showAllFees"
              class="text-xs text-primary hover:underline"
            >
              {{ showAllFees ? 'Nur hohe Konfidenz' : 'Alle offenen Beitrage anzeigen' }}
            </button>
          </div>
          <p v-if="isConfidencePrefiltered" class="text-xs text-gray-500 mb-2">
            Gefiltert nach hoher Konfidenz (>= {{ Math.round(PREFILTER_CONFIDENCE * 100) }}%).
          </p>
          <div class="relative">
            <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
            <input
              v-model="feeSearch"
              type="text"
              placeholder="Nach Kind oder Beitragsart suchen..."
              class="w-full pl-10 pr-4 py-2 border rounded-lg text-sm"
              @input="loadAvailableFees"
            />
          </div>
        </div>

        <!-- Fee List -->
        <div class="flex-1 overflow-y-auto p-4">
          <div v-if="isLoadingFees" class="flex items-center justify-center py-8">
            <Loader2 class="h-6 w-6 animate-spin text-primary" />
          </div>
          <div v-else-if="displayedFeeCandidates.length === 0" class="text-center py-8 text-gray-500">
            Keine offenen Beitrage gefunden
          </div>
          <div v-else class="space-y-2">
            <div
              v-for="candidate in displayedFeeCandidates"
              :key="candidate.fee.id"
              class="p-3 border rounded-lg hover:bg-gray-50 cursor-pointer flex items-center justify-between"
              @click="confirmManualMatch(candidate.fee.id)"
            >
              <div>
                <div class="font-medium">
                  {{ candidate.fee.child?.firstName }} {{ candidate.fee.child?.lastName }}
                </div>
                <div class="text-sm text-gray-600">
                  {{ getFeeTypeName(candidate.fee.feeType) }}
                  - {{ candidate.fee.month ? candidate.fee.month + '/' : '' }}{{ candidate.fee.year }}
                </div>
              </div>
              <div class="text-right">
                <div class="flex items-center justify-end gap-2">
                  <span
                    v-if="candidate.confidence > 0"
                    :class="[
                      'px-2 py-0.5 rounded-full text-xs font-medium',
                      getConfidenceColor(candidate.confidence),
                    ]"
                  >
                    {{ getConfidenceLabel(candidate.confidence) }} ({{ Math.round(candidate.confidence * 100) }}%)
                  </span>
                  <div class="font-medium">{{ formatCurrency(candidate.fee.amount) }}</div>
                </div>
                <div class="text-xs text-gray-500">Fallig: {{ formatDate(candidate.fee.dueDate) }}</div>
              </div>
            </div>
          </div>
        </div>

        <!-- Modal Footer -->
        <div class="p-4 border-t bg-gray-50 flex justify-end">
          <button
            @click="closeManualMatch"
            class="px-4 py-2 text-sm text-gray-600 hover:text-gray-900"
          >
            Abbrechen
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
