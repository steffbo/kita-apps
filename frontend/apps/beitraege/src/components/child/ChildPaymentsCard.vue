<script setup lang="ts">
import { toRefs } from 'vue';
import type { FeeExpectation, MatchSuggestion } from '@/api/types';
import { Loader2, Receipt, CheckCircle, Clock, AlertTriangle, AlertCircle, CreditCard } from 'lucide-vue-next';
import { formatCurrency, formatDate, todayISO } from '@/utils/format';
import {
  formatConfidence,
  formatMatchedBy,
  getFeeMatches,
  getFeeRemainingAmount,
  getFeePeriodLabel,
  getFeeTypeName,
  getTxRemainingAmount,
} from '@/utils/fees';

const props = defineProps<{
  fees: FeeExpectation[];
  likelyTransactions: MatchSuggestion[];
  isLoadingLikelyTransactions: boolean;
  likelyTransactionsError: string | null;
  likelyTransactionsScanned: number;
  formatSuggestionExpectation: (suggestion: MatchSuggestion) => string;
  openFeeGroups: { fee: FeeExpectation; reminders: FeeExpectation[] }[];
  paidFeeGroups: { fee: FeeExpectation; reminders: FeeExpectation[] }[];
  openTransactionModal: (fee: FeeExpectation) => void;
  openAllocationModal: (suggestion: MatchSuggestion) => void;
  getPaymentSummary: (fee: FeeExpectation) => string;
  getReminderSummary: (reminders: FeeExpectation[]) => string;
  canCreateReminder: (fee: FeeExpectation) => boolean;
  openReminderDialog: (fee: FeeExpectation) => void;
}>();

const {
  fees,
  likelyTransactions,
  isLoadingLikelyTransactions,
  likelyTransactionsError,
  likelyTransactionsScanned,
  formatSuggestionExpectation,
  openFeeGroups,
  paidFeeGroups,
  openTransactionModal,
  openAllocationModal,
  getPaymentSummary,
  getReminderSummary,
  canCreateReminder,
  openReminderDialog,
} = toRefs(props);
</script>

<template>
  <div class="bg-card rounded-xl border p-6">
    <div class="flex items-center justify-between mb-6">
      <div class="flex items-center gap-2">
        <Receipt class="h-5 w-5 text-primary" />
        <h2 class="text-lg font-semibold">Beiträge</h2>
      </div>
    </div>

    <!-- Standard View (Open/Paid fees) -->
    <div>
      <!-- Likely unmatched transactions -->
      <div class="mb-6">
        <h3 class="text-sm font-medium text-muted-foreground mb-3 flex items-center gap-2">
          <Receipt class="h-4 w-4" />
          Wahrscheinlich zugehörige Transaktionen
        </h3>

        <div v-if="isLoadingLikelyTransactions" class="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 class="h-4 w-4 animate-spin" />
          Lade Vorschläge...
        </div>
        <div v-else-if="likelyTransactionsError" class="text-sm text-red-600 dark:text-red-300">
          {{ likelyTransactionsError }}
        </div>
        <div v-else-if="likelyTransactions.length === 0" class="text-sm text-muted-foreground">
          Keine offenen Transaktionen mit hoher Wahrscheinlichkeit gefunden.
        </div>
        <div v-else class="space-y-2">
          <div
            v-for="suggestion in likelyTransactions"
            :key="suggestion.transaction.id"
            class="flex items-start justify-between gap-4 p-3 bg-blue-50 dark:bg-blue-950/40 border border-blue-200 rounded-lg"
          >
            <div class="space-y-1">
              <p class="font-medium text-blue-900 dark:text-blue-300">
                {{ suggestion.transaction.payerName || 'Unbekannt' }}
                <span class="text-xs text-blue-600 dark:text-blue-300 ml-2">· {{ formatDate(suggestion.transaction.bookingDate) }}</span>
              </p>
              <p v-if="suggestion.transaction.description" class="text-sm text-blue-800 dark:text-blue-300 break-words">
                {{ suggestion.transaction.description }}
              </p>
              <div class="text-xs text-blue-700 dark:text-blue-300 flex items-center gap-2">
                <span>Konfidenz: {{ formatConfidence(suggestion.confidence) }}</span>
                <span>· Match: {{ formatMatchedBy(suggestion.matchedBy) }}</span>
                <span v-if="formatSuggestionExpectation(suggestion)">
                  · Vorschlag: {{ formatSuggestionExpectation(suggestion) }}
                </span>
              </div>
            </div>
            <div class="text-right space-y-2">
              <p class="font-semibold text-blue-900 dark:text-blue-300">{{ formatCurrency(suggestion.transaction.amount) }}</p>
              <p
                v-if="(suggestion.transaction.matchedAmount ?? 0) > 0"
                class="text-xs text-amber-700 dark:text-amber-300"
              >
                Bereits zugeordnet: {{ formatCurrency(suggestion.transaction.matchedAmount ?? 0) }}
                · Rest: {{ formatCurrency(getTxRemainingAmount(suggestion.transaction)) }}
              </p>
              <button
                @click="openAllocationModal(suggestion)"
                class="inline-flex items-center gap-1 px-2 py-1 text-xs text-blue-700 dark:text-blue-300 bg-card hover:bg-blue-100 dark:hover:bg-blue-900/40 border border-blue-200 rounded transition-colors"
              >
                Zuordnen
              </button>
            </div>
          </div>
          <p v-if="likelyTransactionsScanned > 0" class="text-xs text-muted-foreground">
            {{ likelyTransactions.length }} Treffer aus {{ likelyTransactionsScanned }} offenen Transaktionen.
          </p>
        </div>
      </div>

      <!-- Open fees -->
      <div v-if="openFeeGroups.length > 0" class="mb-6">
        <h3 class="text-sm font-medium text-muted-foreground mb-3 flex items-center gap-2">
          <Clock class="h-4 w-4" />
          Offene Beiträge ({{ openFeeGroups.length }})
        </h3>
        <div class="space-y-2">
          <div
            v-for="group in openFeeGroups"
            :key="group.fee.id"
            :class="[
              'flex items-center justify-between p-3 rounded-lg',
              group.fee.feeType === 'REMINDER'
                ? 'bg-red-50 dark:bg-red-950/40 border border-red-200'
                : 'bg-amber-50 dark:bg-amber-950/40 border border-amber-200'
            ]"
          >
            <div class="flex items-center gap-3">
              <AlertTriangle
                v-if="group.fee.dueDate.slice(0, 10) < todayISO()"
                :class="group.fee.feeType === 'REMINDER' ? 'h-5 w-5 text-red-500 dark:text-red-400' : 'h-5 w-5 text-red-500 dark:text-red-400'"
              />
              <Clock v-else :class="group.fee.feeType === 'REMINDER' ? 'h-5 w-5 text-red-500 dark:text-red-400' : 'h-5 w-5 text-amber-500 dark:text-amber-400'" />
              <div>
                <p :class="['font-medium', group.fee.feeType === 'REMINDER' ? 'text-red-700 dark:text-red-300' : '']">{{ getFeeTypeName(group.fee.feeType) }}</p>
                <p class="text-sm text-muted-foreground">
                  {{ getFeePeriodLabel(group.fee) }}
                  · Fällig: {{ formatDate(group.fee.dueDate) }}
                </p>
                <p v-if="group.fee.matchedAmount && group.fee.matchedAmount > 0" class="text-xs text-amber-700 dark:text-amber-300">
                  Bereits bezahlt: {{ formatCurrency(group.fee.matchedAmount) }} · Rest: {{ formatCurrency(getFeeRemainingAmount(group.fee)) }}
                </p>
                <p
                  v-if="group.reminders.length > 0"
                  :class="[
                    'text-xs',
                    group.reminders.some(rem => !rem.isPaid) ? 'text-red-700 dark:text-red-300' : 'text-green-700 dark:text-green-300'
                  ]"
                >
                  Mahngebühren: {{ getReminderSummary(group.reminders) }}
                </p>
                <p
                  v-if="group.fee.isPaid && group.reminders.some(rem => !rem.isPaid)"
                  class="text-xs text-amber-700 dark:text-amber-300"
                >
                  Beitrag bezahlt · Mahngebühren offen
                </p>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <p :class="['font-semibold', group.fee.feeType === 'REMINDER' ? 'text-red-700 dark:text-red-300' : '']">{{ formatCurrency(group.fee.amount) }}</p>
              <button
                v-if="canCreateReminder(group.fee)"
                @click="openReminderDialog(group.fee)"
                class="p-1.5 text-amber-600 dark:text-amber-300 hover:text-amber-800 dark:text-amber-300 hover:bg-amber-100 dark:hover:bg-amber-900/40 rounded-lg transition-colors"
                title="Mahngebühr erstellen"
              >
                <AlertCircle class="h-4 w-4" />
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Paid fees -->
      <div v-if="paidFeeGroups.length > 0">
        <h3 class="text-sm font-medium text-muted-foreground mb-3 flex items-center gap-2">
          <CheckCircle class="h-4 w-4" />
          Bezahlte Beiträge ({{ paidFeeGroups.length }})
        </h3>
        <div class="space-y-2">
          <button
            v-for="group in paidFeeGroups"
            :key="group.fee.id"
            @click="openTransactionModal(group.fee)"
            :class="[
              'w-full flex items-center justify-between p-3 bg-green-50 dark:bg-green-950/40 border border-green-200 rounded-lg text-left transition-colors',
              getFeeMatches(group.fee).length > 0 ? 'hover:bg-green-100 dark:hover:bg-green-900/40 cursor-pointer' : ''
            ]"
            :disabled="getFeeMatches(group.fee).length === 0"
          >
            <div class="flex items-center gap-3">
              <CheckCircle class="h-5 w-5 text-green-500 dark:text-green-400" />
              <div>
                <p class="font-medium">{{ getFeeTypeName(group.fee.feeType) }}</p>
                <p class="text-sm text-muted-foreground">
                  {{ getFeePeriodLabel(group.fee) }}
                  <span v-if="getPaymentSummary(group.fee)" class="text-green-600 dark:text-green-300">
                    · {{ getPaymentSummary(group.fee) }}
                  </span>
                </p>
                <p v-if="group.reminders.length > 0" class="text-xs text-green-700 dark:text-green-300">
                  Mahngebühren: {{ getReminderSummary(group.reminders) }}
                </p>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <p class="font-semibold text-green-700 dark:text-green-300">{{ formatCurrency(group.fee.amount) }}</p>
              <span
                v-if="getFeeMatches(group.fee).length > 1"
                class="text-xs font-medium text-green-700 dark:text-green-300 bg-green-100 dark:bg-green-950/40 px-2 py-0.5 rounded-full"
              >
                {{ getFeeMatches(group.fee).length }}x
              </span>
              <CreditCard v-if="getFeeMatches(group.fee).length > 0" class="h-4 w-4 text-green-500 dark:text-green-400" />
            </div>
          </button>
        </div>
      </div>

      <div v-if="fees.length === 0" class="text-center py-8 text-muted-foreground">
        Keine Beiträge vorhanden
      </div>
    </div>
  </div>
</template>
