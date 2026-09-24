<script setup lang="ts">
import { ref, computed } from 'vue';
import { api } from '@/api';
import type { ImportResult, MatchConfirmation } from '@/api/types';
import ImportErrorList from '@/components/ImportErrorList.vue';
import {
  Upload,
  FileSpreadsheet,
  Loader2,
  CheckCircle,
  XCircle,
  AlertTriangle,
  ChevronDown,
  ChevronUp,
  Check,
} from 'lucide-vue-next';
import { formatCurrency, formatDate } from '@/utils/format';
import { getConfidenceColor, getConfidenceLabel, getFeeTypeName } from '@/utils/fees';

const emit = defineEmits<{
  close: [];
  // A CSV was imported (new batch exists)
  imported: [];
  // Suggested matches were confirmed (transaction list changed)
  confirmed: [];
}>();

const isDragging = ref(false);
const isUploading = ref(false);
const uploadError = ref<string | null>(null);
const importResult = ref<ImportResult | null>(null);
const selectedMatches = ref<Set<string>>(new Set());
const isConfirming = ref(false);
const confirmResult = ref<{ confirmed: number; failed: number } | null>(null);
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
    emit('imported');

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
    emit('confirmed');
  } catch (error) {
    uploadError.value = error instanceof Error ? error.message : 'Bestätigung fehlgeschlagen';
  } finally {
    isConfirming.value = false;
  }
}

function resetUpload(): void {
  importResult.value = null;
  uploadError.value = null;
  confirmResult.value = null;
  selectedMatches.value.clear();
}
</script>

<template>
  <div
    class="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
    @click.self="$emit('close')"
  >
    <div class="bg-white rounded-xl shadow-xl max-w-3xl w-full mx-4 max-h-[90vh] overflow-hidden flex flex-col">
      <div class="p-4 border-b flex items-center justify-between">
        <h2 class="text-lg font-semibold">CSV hochladen</h2>
        <button @click="$emit('close')" class="text-gray-400 hover:text-gray-600">
          <XCircle class="h-5 w-5" />
        </button>
      </div>

      <div class="overflow-y-auto p-4 space-y-6">
        <div v-if="uploadError" class="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
          {{ uploadError }}
        </div>
        <!-- Upload Area -->
        <div
          v-if="!importResult"
          @dragover="handleDragOver"
          @dragleave="handleDragLeave"
          @drop="handleDrop"
          :class="[
            'border-2 border-dashed rounded-xl p-12 text-center transition-colors',
            isDragging ? 'border-primary bg-primary/5' : 'border-gray-300 hover:border-gray-400',
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
              <p class="text-lg font-medium text-gray-700">CSV-Datei hierher ziehen</p>
              <p class="text-sm text-gray-500 mt-1">oder klicken um eine Datei auszuwählen</p>
            </div>
            <input type="file" accept=".csv" @change="handleFileSelect" class="hidden" id="file-input" />
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

        <template v-if="importResult">
          <!-- Summary Card -->
          <div class="rounded-xl border p-6">
            <div class="flex items-center justify-between mb-4">
              <div class="flex items-center gap-3">
                <div v-if="importResult.errors?.length" class="p-2 bg-red-100 rounded-lg">
                  <AlertTriangle class="h-6 w-6 text-red-600" />
                </div>
                <div v-else class="p-2 bg-green-100 rounded-lg">
                  <CheckCircle class="h-6 w-6 text-green-600" />
                </div>
                <div>
                  <h2 class="text-lg font-semibold">
                    {{ importResult.errors?.length ? 'Import mit Fehlern abgeschlossen' : 'Import erfolgreich' }}
                  </h2>
                  <p class="text-sm text-gray-600">{{ importResult.fileName }}</p>
                </div>
              </div>
              <button @click="resetUpload" class="text-sm text-gray-600 hover:text-gray-900 underline">
                Neuer Import
              </button>
            </div>
            <div class="grid grid-cols-3 gap-4">
              <div class="p-3 bg-gray-50 rounded-lg text-center">
                <div class="text-2xl font-bold text-gray-900">{{ importResult.totalRows }}</div>
                <div class="text-sm text-gray-600">Zeilen gelesen</div>
              </div>
              <div class="p-3 bg-green-50 rounded-lg text-center">
                <div class="text-2xl font-bold text-green-600">{{ importResult.imported }}</div>
                <div class="text-sm text-gray-600">Importiert</div>
              </div>
              <div class="p-3 bg-gray-50 rounded-lg text-center">
                <div class="text-2xl font-bold text-gray-500">{{ importResult.skipped }}</div>
                <div class="text-sm text-gray-600">Übersprungen</div>
              </div>
            </div>
          </div>

          <ImportErrorList v-if="importResult.errors?.length" :errors="importResult.errors" />

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
          <div v-if="matchableSuggestions.length > 0" class="rounded-xl border">
            <div class="p-4 border-b flex items-center justify-between">
              <div>
                <h3 class="font-semibold">Zuordnungsvorschläge</h3>
                <p class="text-sm text-gray-600">
                  {{ selectedMatches.size }} von {{ matchableSuggestions.length }} ausgewählt
                </p>
              </div>
              <div class="flex gap-2">
                <button @click="selectAllMatches" class="text-sm text-primary hover:underline">
                  Alle auswählen
                </button>
                <span class="text-gray-300">|</span>
                <button @click="deselectAllMatches" class="text-sm text-gray-600 hover:underline">
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
                      <span class="text-gray-500">Grund: {{ suggestion.matchedBy }}</span>
                    </div>

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

            <div class="p-4 border-t bg-gray-50 flex items-center justify-between">
              <p class="text-sm text-gray-600">{{ selectedMatches.size }} Zuordnungen ausgewählt</p>
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
          <div v-if="unmatchableSuggestions.length > 0" class="rounded-xl border">
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
        </template>
      </div>
    </div>
  </div></template>
