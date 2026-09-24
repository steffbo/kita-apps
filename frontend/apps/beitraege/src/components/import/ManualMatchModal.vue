<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { api } from '@/api';
import type { BankTransaction, FeeExpectation, MatchSuggestion } from '@/api/types';
import { Loader2, CheckCircle, XCircle, Search } from 'lucide-vue-next';
import { formatCurrency, formatDate } from '@/utils/format';
import { getConfidenceColor, getConfidenceLabel, getFeeTypeName } from '@/utils/fees';

const props = defineProps<{ transaction: BankTransaction }>();
const emit = defineEmits<{ close: []; matched: [] }>();

const PREFILTER_CONFIDENCE = 0.6;

const manualMatchSuggestion = ref<MatchSuggestion | null>(null);
const isLoadingSuggestions = ref(false);
const isLoadingFees = ref(false);
const availableFees = ref<FeeExpectation[]>([]);
const feeSearch = ref('');
const isCreatingMatch = ref(false);
const showAllFees = ref(false);
const matchError = ref<string | null>(null);

onMounted(async () => {
  isLoadingSuggestions.value = true;
  try {
    manualMatchSuggestion.value = await api.getTransactionSuggestions(props.transaction.id);
  } catch (error) {
    console.error('Failed to load suggestions:', error);
  } finally {
    isLoadingSuggestions.value = false;
  }

  await loadAvailableFees();
});

async function loadAvailableFees(): Promise<void> {
  isLoadingFees.value = true;
  try {
    const suggestedChildId = manualMatchSuggestion.value?.child?.id;

    const generalResponse = await api.getFees({
      search: feeSearch.value || undefined,
      perPage: 50,
    });
    let fees = generalResponse.data.filter(f => !f.isPaid);

    if (suggestedChildId && !feeSearch.value) {
      const childFeesResponse = await api.getFees({
        childId: suggestedChildId,
        perPage: 50,
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

async function confirmManualMatch(expectationId: string): Promise<void> {
  isCreatingMatch.value = true;
  matchError.value = null;
  try {
    await api.createManualMatch(props.transaction.id, expectationId);
    emit('matched');
  } catch (error) {
    console.error('Failed to create match:', error);
    matchError.value = error instanceof Error ? error.message : 'Zuordnung fehlgeschlagen';
  } finally {
    isCreatingMatch.value = false;
  }
}
type ScoredFee = {
  fee: FeeExpectation;
  confidence: number;
};

function computeFeeConfidence(fee: FeeExpectation): number {
  const tx = props.transaction;
  if (!tx) return 0;

  const suggestion = manualMatchSuggestion.value;
  const suggestionConfidence = typeof suggestion?.confidence === 'number' ? suggestion.confidence : 0;

  if (suggestion?.expectation?.id && suggestion.expectation.id === fee.id) {
    return 0.99;
  }
  if (suggestion?.expectations?.some(expectation => expectation.id === fee.id)) {
    return 0.99;
  }

  const txDate = tx.bookingDate ? new Date(tx.bookingDate) : null;
  const hasSuggestionChild = !!suggestion?.child?.id && fee.child?.id === suggestion.child.id;
  const amountMatches = Math.abs((fee.amount || 0) - tx.amount) < 0.01;
  const amountScore = amountMatches ? 0.25 : 0;
  const typeScore = suggestion?.detectedType && fee.feeType === suggestion.detectedType ? 0.1 : 0;
  const childScore = hasSuggestionChild ? suggestionConfidence * 0.6 : 0;
  let dateScore = 0;
  if (hasSuggestionChild && txDate && fee.month && fee.year) {
    const txYear = txDate.getFullYear();
    const txMonth = txDate.getMonth() + 1;
    if (txYear === fee.year && txMonth === fee.month) {
      dateScore = 0.25;
    } else {
      const monthDiff = Math.abs((txYear - fee.year) * 12 + (txMonth - fee.month));
      if (monthDiff === 1) {
        dateScore = 0.15;
      } else if (monthDiff === 2) {
        dateScore = 0.05;
      }
    }
  }

  return Math.min(amountScore + typeScore + childScore + dateScore, 0.99);
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
</script>

<template>
  <div
    class="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
    @click.self="$emit('close')"
  >
    <div class="bg-white rounded-xl shadow-xl max-w-3xl w-full mx-4 max-h-[90vh] overflow-hidden flex flex-col">
      <!-- Modal Header -->
      <div class="p-4 border-b">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold">Transaktion manuell zuordnen</h2>
          <button @click="$emit('close')" class="text-gray-400 hover:text-gray-600">
            <XCircle class="h-5 w-5" />
          </button>
        </div>
      </div>

      <!-- Transaction Details -->
      <div class="p-4 bg-gray-50 border-b">
        <div class="grid grid-cols-2 gap-4 text-sm">
          <div>
            <span class="text-gray-500">Zahler:</span>
            <span class="ml-2 font-medium">{{ transaction.payerName || 'Unbekannt' }}</span>
          </div>
          <div>
            <span class="text-gray-500">Betrag:</span>
            <span class="ml-2 font-medium text-green-600">{{ formatCurrency(transaction.amount) }}</span>
          </div>
          <div>
            <span class="text-gray-500">Datum:</span>
            <span class="ml-2">{{ formatDate(transaction.bookingDate) }}</span>
          </div>
          <div v-if="transaction.payerIban">
            <span class="text-gray-500">IBAN:</span>
            <span class="ml-2 font-mono text-xs">{{ transaction.payerIban }}</span>
          </div>
        </div>
        <div v-if="transaction.description" class="mt-2 text-sm text-gray-600">
          {{ transaction.description }}
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

      <div v-if="matchError" class="px-4 pt-4">
        <div class="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">{{ matchError }}</div>
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
        <button @click="$emit('close')" class="px-4 py-2 text-sm text-gray-600 hover:text-gray-900">
          Abbrechen
        </button>
      </div>
    </div>
  </div></template>
