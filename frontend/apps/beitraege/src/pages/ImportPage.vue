<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { api } from '@/api';
import type { ImportResult, ImportBatch, BankTransaction, MatchConfirmation, KnownIBAN } from '@/api/types';
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
  RefreshCw,
  Check,
  Ban,
  Trash2,
  ShieldOff,
} from 'lucide-vue-next';

type TabType = 'upload' | 'history' | 'unmatched' | 'blacklist';

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

// Dismiss state
const isDismissing = ref<string | null>(null);
const dismissConfirmId = ref<string | null>(null);

const expandedSuggestions = ref<Set<string>>(new Set());

function toggleSuggestion(id: string) {
  if (expandedSuggestions.value.has(id)) {
    expandedSuggestions.value.delete(id);
  } else {
    expandedSuggestions.value.add(id);
  }
}

function handleDragOver(e: DragEvent) {
  e.preventDefault();
  isDragging.value = true;
}

function handleDragLeave() {
  isDragging.value = false;
}

async function handleDrop(e: DragEvent) {
  e.preventDefault();
  isDragging.value = false;
  const files = e.dataTransfer?.files;
  if (files && files.length > 0) {
    await uploadFile(files[0]);
  }
}

async function handleFileSelect(e: Event) {
  const input = e.target as HTMLInputElement;
  if (input.files && input.files.length > 0) {
    await uploadFile(input.files[0]);
    input.value = '';
  }
}

async function uploadFile(file: File) {
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
  } catch (e) {
    uploadError.value = e instanceof Error ? e.message : 'Upload fehlgeschlagen';
  } finally {
    isUploading.value = false;
  }
}

function toggleMatch(transactionId: string) {
  if (selectedMatches.value.has(transactionId)) {
    selectedMatches.value.delete(transactionId);
  } else {
    selectedMatches.value.add(transactionId);
  }
}

function selectAllMatches() {
  if (!importResult.value) return;
  for (const suggestion of importResult.value.suggestions) {
    if (suggestion.expectation) {
      selectedMatches.value.add(suggestion.transaction.id);
    }
  }
}

function deselectAllMatches() {
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

async function confirmMatches() {
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
  } catch (e) {
    uploadError.value = e instanceof Error ? e.message : 'Bestätigung fehlgeschlagen';
  } finally {
    isConfirming.value = false;
  }
}

async function loadHistory() {
  isLoadingHistory.value = true;
  try {
    const response = await api.getImportHistory(0, 50);
    importHistory.value = response.data;
    historyTotal.value = response.total;
  } catch (e) {
    console.error('Failed to load history:', e);
  } finally {
    isLoadingHistory.value = false;
  }
}

async function loadUnmatched() {
  isLoadingUnmatched.value = true;
  try {
    const response = await api.getUnmatchedTransactions(0, 100);
    unmatchedTransactions.value = response.data;
    unmatchedTotal.value = response.total;
  } catch (e) {
    console.error('Failed to load unmatched:', e);
  } finally {
    isLoadingUnmatched.value = false;
  }
}

async function loadBlacklist() {
  isLoadingBlacklist.value = true;
  try {
    const response = await api.getBlacklist(0, 100);
    blacklistedIBANs.value = response.data;
    blacklistTotal.value = response.total;
  } catch (e) {
    console.error('Failed to load blacklist:', e);
  } finally {
    isLoadingBlacklist.value = false;
  }
}

function showDismissConfirm(transactionId: string) {
  dismissConfirmId.value = transactionId;
}

function cancelDismiss() {
  dismissConfirmId.value = null;
}

async function dismissTransaction(transaction: BankTransaction) {
  isDismissing.value = transaction.id;
  dismissConfirmId.value = null;
  try {
    const result = await api.dismissTransaction(transaction.id);
    // Remove all transactions with this IBAN from the list
    unmatchedTransactions.value = unmatchedTransactions.value.filter(
      tx => tx.payerIban !== transaction.payerIban
    );
    unmatchedTotal.value = Math.max(0, unmatchedTotal.value - result.transactionsRemoved);
    // Refresh blacklist if on that tab
    if (activeTab.value === 'blacklist') {
      loadBlacklist();
    }
  } catch (e) {
    console.error('Failed to dismiss transaction:', e);
    uploadError.value = e instanceof Error ? e.message : 'Ignorieren fehlgeschlagen';
  } finally {
    isDismissing.value = null;
  }
}

async function removeFromBlacklist(iban: string) {
  try {
    await api.removeFromBlacklist(iban);
    blacklistedIBANs.value = blacklistedIBANs.value.filter(item => item.iban !== iban);
    blacklistTotal.value = Math.max(0, blacklistTotal.value - 1);
  } catch (e) {
    console.error('Failed to remove from blacklist:', e);
    uploadError.value = e instanceof Error ? e.message : 'Entfernen fehlgeschlagen';
  }
}

function switchTab(tab: TabType) {
  activeTab.value = tab;
  if (tab === 'history') {
    loadHistory();
  } else if (tab === 'unmatched') {
    loadUnmatched();
  } else if (tab === 'blacklist') {
    loadBlacklist();
  }
}

onMounted(() => {
  // Pre-load history in background
  loadHistory();
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
    default:
      return type || 'Unbekannt';
  }
}

function resetUpload() {
  importResult.value = null;
  uploadError.value = null;
  confirmResult.value = null;
  selectedMatches.value.clear();
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
              <td class="px-4 py-3 text-gray-600">{{ batch.importedBy }}</td>
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
        <button
          @click="loadUnmatched"
          class="inline-flex items-center gap-1 text-sm text-primary hover:underline"
        >
          <RefreshCw class="h-4 w-4" />
          Aktualisieren
        </button>
      </div>

      <div v-if="isLoadingUnmatched" class="flex items-center justify-center py-12">
        <Loader2 class="h-8 w-8 animate-spin text-primary" />
      </div>

      <div v-else-if="unmatchedTransactions.length === 0" class="text-center py-12">
        <CheckCircle class="h-12 w-12 text-green-300 mx-auto mb-4" />
        <p class="text-gray-600">Alle Transaktionen sind zugeordnet</p>
      </div>

      <div v-else class="bg-white rounded-xl border overflow-hidden">
        <table class="w-full">
          <thead class="bg-gray-50">
            <tr class="text-left text-sm text-gray-500">
              <th class="px-4 py-3 font-medium">Datum</th>
              <th class="px-4 py-3 font-medium">Zahler</th>
              <th class="px-4 py-3 font-medium">Beschreibung</th>
              <th class="px-4 py-3 font-medium text-right">Betrag</th>
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
                <!-- Dismiss Button -->
                <button
                  v-else
                  @click="showDismissConfirm(tx.id)"
                  :disabled="isDismissing === tx.id"
                  class="inline-flex items-center gap-1 px-2 py-1 text-xs text-gray-600 hover:text-red-600 hover:bg-red-50 rounded transition-colors disabled:opacity-50"
                  title="IBAN dauerhaft ignorieren"
                >
                  <Loader2 v-if="isDismissing === tx.id" class="h-3 w-3 animate-spin" />
                  <Ban v-else class="h-3 w-3" />
                  Ignorieren
                </button>
              </td>
            </tr>
          </tbody>
        </table>
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
  </div>
</template>
