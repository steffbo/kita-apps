<script setup lang="ts">
import AppSheet from '@/components/ui/AppSheet.vue';
import { ref, computed } from 'vue';
import { api } from '@/api';
import type { ImportResult, MatchConfirmation } from '@/api/types';
import ImportErrorList from '@/components/ImportErrorList.vue';
import {
  Upload,
  FileSpreadsheet,
  Loader2,
  CheckCircle,
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
  <AppSheet title="CSV hochladen" size="lg" @close="$emit('close')">
      <div class="space-y-6">
        <div v-if="uploadError" class="note note-danger">
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
            isDragging ? 'border-primary bg-primary/5' : 'border-border hover:border-border',
            isUploading ? 'opacity-50 pointer-events-none' : '',
          ]"
        >
          <div v-if="isUploading" class="flex flex-col items-center gap-4">
            <Loader2 class="h-12 w-12 animate-spin text-primary" />
            <p class="text-muted-foreground">CSV wird verarbeitet...</p>
          </div>
          <div v-else class="flex flex-col items-center gap-4">
            <div class="p-4 bg-muted rounded-full">
              <FileSpreadsheet class="h-12 w-12 text-muted-foreground" />
            </div>
            <div>
              <p class="text-lg font-medium text-foreground">CSV-Datei hierher ziehen</p>
              <p class="text-sm text-muted-foreground mt-1">oder klicken um eine Datei auszuwählen</p>
            </div>
            <input type="file" accept=".csv" @change="handleFileSelect" class="hidden" id="file-input" />
            <label
              for="file-input"
              class="btn btn-primary"
            >
              <Upload class="h-4 w-4" />
              Datei auswählen
            </label>
            <p class="text-xs text-muted-foreground mt-2">
              Unterstützt: Deutsche Bankexporte (CSV, Semikolon-getrennt, ISO-8859-1 oder UTF-8)
            </p>
          </div>
        </div>

        <template v-if="importResult">
          <!-- Summary Card -->
          <div class="rounded-xl border p-6">
            <div class="flex items-center justify-between mb-4">
              <div class="flex items-center gap-3">
                <div v-if="importResult.errors?.length" class="p-2 bg-destructive-soft rounded-lg">
                  <AlertTriangle class="h-6 w-6 text-destructive" />
                </div>
                <div v-else class="p-2 bg-success-soft rounded-lg">
                  <CheckCircle class="h-6 w-6 text-success" />
                </div>
                <div>
                  <h2 class="text-base font-semibold">
                    {{ importResult.errors?.length ? 'Import mit Fehlern abgeschlossen' : 'Import erfolgreich' }}
                  </h2>
                  <p class="text-sm text-muted-foreground">{{ importResult.fileName }}</p>
                </div>
              </div>
              <button @click="resetUpload" class="btn btn-ghost btn-sm">
                Neuer Import
              </button>
            </div>
            <div class="grid grid-cols-3 gap-4">
              <div class="p-3 bg-muted rounded-lg text-center">
                <div class="text-2xl font-semibold text-foreground">{{ importResult.totalRows }}</div>
                <div class="text-sm text-muted-foreground">Zeilen gelesen</div>
              </div>
              <div class="p-3 bg-success-soft rounded-lg text-center">
                <div class="text-2xl font-semibold text-success">{{ importResult.imported }}</div>
                <div class="text-sm text-muted-foreground">Importiert</div>
              </div>
              <div class="p-3 bg-muted rounded-lg text-center">
                <div class="text-2xl font-semibold text-muted-foreground">{{ importResult.skipped }}</div>
                <div class="text-sm text-muted-foreground">Übersprungen</div>
              </div>
            </div>
          </div>

          <ImportErrorList v-if="importResult.errors?.length" :errors="importResult.errors" />

          <!-- Confirm Result -->
          <div
            v-if="confirmResult"
            class="border p-4 bg-success-soft border-success-line rounded-lg flex items-start gap-3"
          >
            <CheckCircle class="h-5 w-5 text-success flex-shrink-0 mt-0.5" />
            <div>
              <p class="text-success font-medium">Zuordnungen bestätigt</p>
              <p class="text-sm text-success">
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
                <p class="text-sm text-muted-foreground">
                  {{ selectedMatches.size }} von {{ matchableSuggestions.length }} ausgewählt
                </p>
              </div>
              <div class="flex gap-2">
                <button @click="selectAllMatches" class="btn btn-ghost btn-sm">
                  Alle auswählen
                </button>
                <span class="text-muted-foreground/60">|</span>
                <button @click="deselectAllMatches" class="btn btn-ghost btn-sm">
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
                        ? 'bg-primary border-primary text-primary-foreground'
                        : 'border-border hover:border-border',
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
                        <div class="text-sm text-muted-foreground truncate">
                          {{ suggestion.transaction.description }}
                        </div>
                      </div>
                      <div class="text-right flex-shrink-0">
                        <div class="font-semibold text-success">
                          {{ formatCurrency(suggestion.transaction.amount) }}
                        </div>
                        <div class="text-xs text-muted-foreground">
                          {{ formatDate(suggestion.transaction.bookingDate) }}
                        </div>
                      </div>
                    </div>

                    <div class="mt-3 flex items-center gap-4 text-sm">
                      <span
                        :class="getConfidenceColor(suggestion.confidence)"
                      >
                        {{ getConfidenceLabel(suggestion.confidence) }} ({{ Math.round(suggestion.confidence * 100) }}%)
                      </span>
                      <span class="text-muted-foreground">
                        Erkannt als: {{ getFeeTypeName(suggestion.detectedType) }}
                      </span>
                      <span class="text-muted-foreground">Grund: {{ suggestion.matchedBy }}</span>
                    </div>

                    <button
                      @click="toggleSuggestion(suggestion.transaction.id)"
                      class="btn btn-ghost btn-sm mt-2"
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
                      class="mt-3 p-3 bg-muted rounded-lg text-sm space-y-2"
                    >
                      <div class="grid grid-cols-2 gap-4">
                        <div>
                          <span class="text-muted-foreground">Zugeordnetes Kind:</span>
                          <span class="ml-2 font-medium">
                            {{ suggestion.child?.firstName }} {{ suggestion.child?.lastName }}
                          </span>
                        </div>
                        <div>
                          <span class="text-muted-foreground">Beitragsart:</span>
                          <span class="ml-2 font-medium">
                            {{ getFeeTypeName(suggestion.expectation?.feeType) }}
                          </span>
                        </div>
                        <div>
                          <span class="text-muted-foreground">Erwarteter Betrag:</span>
                          <span class="ml-2 font-medium">
                            {{ formatCurrency(suggestion.expectation?.amount || 0) }}
                          </span>
                        </div>
                        <div>
                          <span class="text-muted-foreground">Zeitraum:</span>
                          <span class="ml-2 font-medium">
                            {{ suggestion.expectation?.month ? suggestion.expectation.month + '/' : '' }}{{ suggestion.expectation?.year }}
                          </span>
                        </div>
                      </div>
                      <div v-if="suggestion.transaction.payerIban">
                        <span class="text-muted-foreground">IBAN:</span>
                        <span class="ml-2 font-mono text-xs">
                          {{ suggestion.transaction.payerIban }}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div class="p-4 border-t bg-muted flex items-center justify-between">
              <p class="text-sm text-muted-foreground">{{ selectedMatches.size }} Zuordnungen ausgewählt</p>
              <button
                @click="confirmMatches"
                :disabled="selectedMatches.size === 0 || isConfirming"
                class="btn btn-primary"
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
                <AlertTriangle class="h-5 w-5 text-warning" />
                <h3 class="font-semibold">Nicht zuordenbar</h3>
              </div>
              <p class="text-sm text-muted-foreground mt-1">
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
                  <div class="text-sm text-muted-foreground truncate max-w-md">
                    {{ suggestion.transaction.description }}
                  </div>
                </div>
                <div class="text-right">
                  <div class="font-semibold">
                    {{ formatCurrency(suggestion.transaction.amount) }}
                  </div>
                  <div class="text-xs text-muted-foreground">
                    {{ formatDate(suggestion.transaction.bookingDate) }}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </template>
      </div>
</AppSheet>
</template>
