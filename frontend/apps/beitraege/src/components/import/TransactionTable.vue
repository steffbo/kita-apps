<script setup lang="ts">
import type { BankTransaction, TransactionWarning } from '@/api/types';
import { computed, toRefs } from 'vue';
import type { StatusFilter, SortField, TxRow } from '@/composables/useImportTransactions';
import { isPartiallyAllocated, getTxRemaining } from '@/composables/useImportTransactions';
import {
  Loader2,
  CheckCircle,
  XCircle,
  AlertTriangle,
  ChevronDown,
  ChevronUp,
  ChevronLeft,
  ChevronRight,
  Ban,
  Trash2,
  EyeOff,
  Clock,
  Euro,
  LinkIcon,
  Unlink,
  Search,
  ArrowUp,
  ArrowDown,
  ArrowUpDown,
} from 'lucide-vue-next';
import { formatCurrency, formatDate, formatDateTime, formatMonthName } from '@/utils/format';
import { getFeeTypeColor, getFeeTypeName } from '@/utils/fees';

const props = defineProps<{
  activeFilter: StatusFilter;
  isLoadingTransactions: boolean;
  transactionSearch: string;
  sortField: SortField;
  sortDirection: 'asc' | 'desc';
  page: number;
  toggleSort: (field: SortField) => void;
  sortedRows: TxRow[];
  totalPages: number;
  visiblePages: number[];
  pagedRows: TxRow[];
  goToPage: (target: number) => void;
  isResolvingWarning: string | null;
  dismissWarningId: string | null;
  dismissNote: string;
  showWarningDismiss: (warningId: string) => void;
  cancelWarningDismiss: () => void;
  dismissWarning: (warning: TransactionWarning) => Promise<void>;
  resolveLateFee: (warning: TransactionWarning) => Promise<void>;
  expandedWarnings: Set<string>;
  isDismissing: string | null;
  dismissConfirmId: string | null;
  isHiding: string | null;
  hideConfirmId: string | null;
  unmatchConfirmId: string | null;
  deleteConfirmId: string | null;
  isUnmatching: string | null;
  isDeletingMatched: string | null;
  toggleWarnings: (key: string) => void;
  openManualMatch: (transaction: BankTransaction) => void;
  showDismissConfirm: (transactionId: string) => void;
  cancelDismiss: () => void;
  showHideConfirm: (transactionId: string) => void;
  cancelHide: () => void;
  showUnmatchConfirm: (transactionId: string) => void;
  showDeleteConfirm: (transactionId: string) => void;
  cancelMatchAction: () => void;
  dismissTransaction: (transaction: BankTransaction) => Promise<void>;
  hideTransaction: (transaction: BankTransaction) => Promise<void>;
  unmatchTransaction: (transaction: BankTransaction, deleteTransaction?: boolean) => Promise<void>;
  getWarningTypeLabel: (type: string) => string;
  getWarningTypeColor: (type: string) => string;
}>();

const {
  activeFilter,
  isLoadingTransactions,
  transactionSearch,
  sortField,
  sortDirection,
  page,
  toggleSort,
  sortedRows,
  totalPages,
  visiblePages,
  pagedRows,
  goToPage,
  isResolvingWarning,
  dismissWarningId,
  showWarningDismiss,
  cancelWarningDismiss,
  dismissWarning,
  resolveLateFee,
  expandedWarnings,
  isDismissing,
  dismissConfirmId,
  isHiding,
  hideConfirmId,
  unmatchConfirmId,
  deleteConfirmId,
  isUnmatching,
  isDeletingMatched,
  toggleWarnings,
  openManualMatch,
  showDismissConfirm,
  cancelDismiss,
  showHideConfirm,
  cancelHide,
  showUnmatchConfirm,
  showDeleteConfirm,
  cancelMatchAction,
  dismissTransaction,
  hideTransaction,
  unmatchTransaction,
  getWarningTypeLabel,
  getWarningTypeColor,
} = toRefs(props);

const emit = defineEmits<{
  'update:dismissNote': [value: string];
}>();
const dismissNote = computed({
  get: () => props.dismissNote,
  set: (value) => emit('update:dismissNote', value),
});
</script>

<template>
  <!-- Unified Transaction List -->
  <div v-if="isLoadingTransactions" class="flex items-center justify-center py-12">
    <Loader2 class="h-8 w-8 animate-spin text-primary" />
  </div>

  <div v-else-if="pagedRows.length === 0" class="bg-card rounded-xl border text-center py-12">
    <component
      :is="transactionSearch.trim() ? Search : CheckCircle"
      :class="['h-12 w-12 mx-auto mb-4', transactionSearch.trim() ? 'text-muted-foreground/60' : 'text-green-300']"
    />
    <p class="text-muted-foreground">
      {{ transactionSearch.trim() ? 'Keine Transaktionen gefunden' : 'Keine offenen Transaktionen' }}
    </p>
    <p v-if="transactionSearch.trim()" class="text-sm text-muted-foreground mt-1">Versuche einen anderen Suchbegriff</p>
    <p v-else-if="activeFilter === 'warnungen'" class="text-sm text-muted-foreground mt-1">Alle Zahlungen wurden korrekt verarbeitet</p>
    <p v-else-if="activeFilter === 'zugeordnet'" class="text-sm text-muted-foreground mt-1">Noch keine zugeordneten Transaktionen</p>
  </div>

  <div v-else class="bg-card rounded-xl border overflow-hidden">
    <div class="overflow-x-auto">
      <table class="w-full">
        <thead class="bg-muted">
          <tr class="text-left text-sm text-muted-foreground">
            <th
              class="px-3 py-3 font-medium cursor-pointer hover:bg-accent select-none"
              @click="toggleSort('date')"
            >
              <div class="flex items-center gap-1">
                Datum
                <ArrowUp v-if="sortField === 'date' && sortDirection === 'asc'" class="h-4 w-4" />
                <ArrowDown v-else-if="sortField === 'date' && sortDirection === 'desc'" class="h-4 w-4" />
                <ArrowUpDown v-else class="h-4 w-4 text-muted-foreground" />
              </div>
            </th>
            <th
              class="px-3 py-3 font-medium cursor-pointer hover:bg-accent select-none"
              @click="toggleSort('payer')"
            >
              <div class="flex items-center gap-1">
                Zahler
                <ArrowUp v-if="sortField === 'payer' && sortDirection === 'asc'" class="h-4 w-4" />
                <ArrowDown v-else-if="sortField === 'payer' && sortDirection === 'desc'" class="h-4 w-4" />
                <ArrowUpDown v-else class="h-4 w-4 text-muted-foreground" />
              </div>
            </th>
            <th
              class="px-3 py-3 font-medium cursor-pointer hover:bg-accent select-none"
              @click="toggleSort('description')"
            >
              <div class="flex items-center gap-1">
                Beschreibung
                <ArrowUp v-if="sortField === 'description' && sortDirection === 'asc'" class="h-4 w-4" />
                <ArrowDown v-else-if="sortField === 'description' && sortDirection === 'desc'" class="h-4 w-4" />
                <ArrowUpDown v-else class="h-4 w-4 text-muted-foreground" />
              </div>
            </th>
            <th
              class="px-3 py-3 font-medium text-right cursor-pointer hover:bg-accent select-none"
              @click="toggleSort('amount')"
            >
              <div class="flex items-center justify-end gap-1">
                Betrag
                <ArrowUp v-if="sortField === 'amount' && sortDirection === 'asc'" class="h-4 w-4" />
                <ArrowDown v-else-if="sortField === 'amount' && sortDirection === 'desc'" class="h-4 w-4" />
                <ArrowUpDown v-else class="h-4 w-4 text-muted-foreground" />
              </div>
            </th>
            <th class="px-3 py-3 font-medium">Status</th>
            <th class="px-3 py-3 font-medium text-right">Aktionen</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="row in pagedRows" :key="row.key">
            <tr class="border-t hover:bg-accent">
              <td class="px-3 py-3 text-muted-foreground whitespace-nowrap">
                {{ formatDate(row.tx.bookingDate) }}
              </td>
              <td class="px-3 py-3">
                <div class="font-medium">{{ row.tx.payerName || 'Unbekannt' }}</div>
                <div v-if="row.tx.payerIban" class="text-xs text-muted-foreground font-mono whitespace-nowrap">
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
              <td class="px-3 py-3 text-muted-foreground truncate max-w-[14rem]">
                {{ row.tx.description }}
              </td>
              <td
                :class="[
                  'px-3 py-3 text-right font-medium whitespace-nowrap',
                  row.matched ? 'text-green-600 dark:text-green-300' : '',
                ]"
              >
                {{ formatCurrency(row.tx.amount) }}
              </td>
              <td class="px-3 py-3">
                <div class="flex flex-col items-start gap-1.5">
                  <span
                    v-if="isPartiallyAllocated(row)"
                    class="inline-flex items-center gap-1 whitespace-nowrap px-2 py-0.5 rounded-full text-xs font-medium bg-orange-100 dark:bg-orange-950/40 text-orange-700 dark:text-orange-300"
                  >
                    <AlertTriangle class="h-3 w-3 shrink-0" />
                    Teilweise zugeordnet · Rest {{ formatCurrency(getTxRemaining(row.tx)) }}
                  </span>
                  <span
                    v-else-if="row.matched"
                    class="inline-flex items-center gap-1 whitespace-nowrap px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 dark:bg-green-950/40 text-green-700 dark:text-green-300"
                  >
                    <CheckCircle class="h-3 w-3 shrink-0" />
                    Zugeordnet
                  </span>
                  <span
                    v-else
                    class="inline-flex items-center gap-1 whitespace-nowrap px-2 py-0.5 rounded-full text-xs font-medium bg-amber-100 dark:bg-amber-950/40 text-amber-700 dark:text-amber-300"
                  >
                    <AlertTriangle class="h-3 w-3 shrink-0" />
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
                    class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-orange-100 dark:bg-orange-950/40 text-orange-700 dark:text-orange-300 hover:bg-orange-200 dark:hover:bg-orange-900/50 transition-colors"
                  >
                    <AlertTriangle class="h-3 w-3" />
                    {{ row.warnings.length }} {{ row.warnings.length === 1 ? 'Warnung' : 'Warnungen' }}
                    <ChevronDown v-if="!expandedWarnings.has(row.key)" class="h-3 w-3" />
                    <ChevronUp v-else class="h-3 w-3" />
                  </button>
                </div>
              </td>
              <td class="px-3 py-3 text-right">
                <!-- Unmatched actions -->
                <template v-if="!row.matched">
                  <div v-if="dismissConfirmId === row.tx.id" class="flex flex-wrap items-center justify-end gap-1.5">
                    <span class="text-xs text-muted-foreground">Ignorieren?</span>
                    <button
                      @click="dismissTransaction(row.tx)"
                      class="px-2 py-1 text-xs bg-red-500 text-white rounded hover:bg-red-600"
                    >
                      Ja
                    </button>
                    <button
                      @click="cancelDismiss"
                      class="px-2 py-1 text-xs bg-muted text-foreground rounded hover:bg-accent"
                    >
                      Nein
                    </button>
                  </div>
                  <div v-else-if="hideConfirmId === row.tx.id" class="flex flex-wrap items-center justify-end gap-1.5">
                    <span class="text-xs text-muted-foreground">Ausblenden?</span>
                    <button
                      @click="hideTransaction(row.tx)"
                      class="px-2 py-1 text-xs bg-muted text-white rounded hover:bg-accent"
                    >
                      Ja
                    </button>
                    <button
                      @click="cancelHide"
                      class="px-2 py-1 text-xs bg-muted text-foreground rounded hover:bg-accent"
                    >
                      Nein
                    </button>
                  </div>
                  <div v-else class="flex flex-col items-end gap-0.5">
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
                      class="inline-flex items-center gap-1 px-2 py-1 text-xs text-muted-foreground hover:text-foreground hover:bg-accent rounded transition-colors disabled:opacity-50"
                      title="Transaktion ausblenden"
                    >
                      <Loader2 v-if="isHiding === row.tx.id" class="h-3 w-3 animate-spin" />
                      <EyeOff v-else class="h-3 w-3" />
                      Ausblenden
                    </button>
                    <button
                      @click="showDismissConfirm(row.tx.id)"
                      :disabled="isDismissing === row.tx.id"
                      class="inline-flex items-center gap-1 px-2 py-1 text-xs text-muted-foreground hover:text-red-600 dark:text-red-300 hover:bg-red-50 dark:hover:bg-red-900/40 rounded transition-colors disabled:opacity-50"
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
                  <div v-if="unmatchConfirmId === row.tx.id" class="flex flex-wrap items-center justify-end gap-1.5">
                    <span class="text-xs text-muted-foreground">Zuordnung aufheben?</span>
                    <button
                      @click="unmatchTransaction(row.tx)"
                      class="px-2 py-1 text-xs bg-amber-500 text-white rounded hover:bg-amber-600"
                    >
                      Ja
                    </button>
                    <button
                      @click="cancelMatchAction"
                      class="px-2 py-1 text-xs bg-muted text-foreground rounded hover:bg-accent"
                    >
                      Nein
                    </button>
                  </div>
                  <div v-else-if="deleteConfirmId === row.tx.id" class="flex flex-wrap items-center justify-end gap-1.5">
                    <span class="text-xs text-muted-foreground">Transaktion löschen?</span>
                    <button
                      @click="unmatchTransaction(row.tx, true)"
                      class="px-2 py-1 text-xs bg-red-500 text-white rounded hover:bg-red-600"
                    >
                      Ja
                    </button>
                    <button
                      @click="cancelMatchAction"
                      class="px-2 py-1 text-xs bg-muted text-foreground rounded hover:bg-accent"
                    >
                      Nein
                    </button>
                  </div>
                  <div v-else class="flex flex-col items-end gap-0.5">
                    <button
                      @click="showUnmatchConfirm(row.tx.id)"
                      :disabled="isUnmatching === row.tx.id || isDeletingMatched === row.tx.id"
                      class="inline-flex items-center gap-1 px-2 py-1 text-xs text-muted-foreground hover:text-amber-700 dark:text-amber-300 hover:bg-amber-50 dark:hover:bg-amber-900/40 rounded transition-colors disabled:opacity-50"
                      title="Zuordnung aufheben"
                    >
                      <Loader2 v-if="isUnmatching === row.tx.id" class="h-3 w-3 animate-spin" />
                      <Unlink v-else class="h-3 w-3" />
                      Aufheben
                    </button>
                    <button
                      @click="showDeleteConfirm(row.tx.id)"
                      :disabled="isUnmatching === row.tx.id || isDeletingMatched === row.tx.id"
                      class="inline-flex items-center gap-1 px-2 py-1 text-xs text-muted-foreground hover:text-red-600 dark:text-red-300 hover:bg-red-50 dark:hover:bg-red-900/40 rounded transition-colors disabled:opacity-50"
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
            <tr v-if="expandedWarnings.has(row.key) && row.warnings.length > 0" class="border-t bg-orange-50 dark:bg-orange-950/40">
              <td colspan="6" class="px-4 py-4">
                <div class="space-y-4">
                  <div
                    v-for="warning in row.warnings"
                    :key="warning.id"
                    class="bg-card rounded-xl border p-4"
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
                          <span class="text-xs text-muted-foreground">
                            {{ formatDateTime(warning.createdAt) }}
                          </span>
                        </div>

                        <p class="text-foreground mb-3">{{ warning.message }}</p>

                        <div v-if="warning.transaction" class="p-3 bg-muted rounded-lg text-sm space-y-1">
                          <div class="flex justify-between">
                            <span class="text-muted-foreground">Transaktion:</span>
                            <span class="font-medium">{{ warning.transaction.payerName || 'Unbekannt' }}</span>
                          </div>
                          <div class="flex justify-between">
                            <span class="text-muted-foreground">Betrag:</span>
                            <span class="font-medium text-green-600 dark:text-green-300">{{ formatCurrency(warning.transaction.amount) }}</span>
                          </div>
                          <div class="flex justify-between">
                            <span class="text-muted-foreground">Datum:</span>
                            <span>{{ formatDate(warning.transaction.bookingDate) }}</span>
                          </div>
                          <div v-if="warning.transaction.description" class="text-muted-foreground text-xs truncate">
                            {{ warning.transaction.description }}
                          </div>
                        </div>

                        <div
                          v-if="warning.warningType === 'LATE_PAYMENT' && warning.matchedFee"
                          class="mt-2 p-3 bg-orange-50 dark:bg-orange-950/40 rounded-lg text-sm space-y-1"
                        >
                          <div class="flex items-center gap-1 text-orange-700 dark:text-orange-300 font-medium mb-1">
                            <Clock class="h-4 w-4" />
                            Verspätete Zahlung für:
                          </div>
                          <div class="flex justify-between">
                            <span class="text-muted-foreground">Beitragsart:</span>
                            <span class="font-medium">{{ getFeeTypeName(warning.matchedFee.feeType) }}</span>
                          </div>
                          <div class="flex justify-between">
                            <span class="text-muted-foreground">Zeitraum:</span>
                            <span>{{ formatMonthName(warning.matchedFee.month) }} {{ warning.matchedFee.year }}</span>
                          </div>
                          <div class="flex justify-between">
                            <span class="text-muted-foreground">Betrag:</span>
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

                        <div v-if="dismissWarningId === warning.id" class="p-3 bg-muted rounded-lg space-y-2">
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
                              class="px-2 py-1 text-xs bg-muted text-foreground rounded hover:bg-accent"
                            >
                              Abbrechen
                            </button>
                          </div>
                        </div>

                        <button
                          v-else
                          @click="showWarningDismiss(warning.id)"
                          :disabled="isResolvingWarning === warning.id"
                          class="inline-flex items-center gap-1 px-3 py-1.5 text-sm text-muted-foreground hover:text-red-600 dark:text-red-300 hover:bg-red-50 dark:hover:bg-red-900/40 rounded-lg transition-colors disabled:opacity-50"
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
    <div v-if="totalPages > 1" class="flex items-center justify-between px-4 py-3 border-t bg-muted">
      <div class="text-sm text-muted-foreground">
        Seite {{ page }} von {{ totalPages }} ({{ sortedRows.length }} Einträge)
      </div>
      <div class="flex items-center gap-2">
        <button
          @click="goToPage(page - 1)"
          :disabled="page <= 1"
          class="p-1 rounded hover:bg-accent disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <ChevronLeft class="h-5 w-5" />
        </button>
        <button
          v-for="pageNumber in visiblePages"
          :key="pageNumber"
          @click="goToPage(pageNumber)"
          :class="[
            'px-3 py-1 rounded text-sm',
            pageNumber === page ? 'bg-primary text-primary-foreground' : 'hover:bg-accent',
          ]"
        >
          {{ pageNumber }}
        </button>
        <button
          @click="goToPage(page + 1)"
          :disabled="page >= totalPages"
          class="p-1 rounded hover:bg-accent disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <ChevronRight class="h-5 w-5" />
        </button>
      </div>
    </div>
  </div>
</template>
