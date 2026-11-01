<script setup lang="ts">
import type { BankTransaction, TransactionWarning } from '@/api/types';
import { computed, toRefs } from 'vue';
import { useAuthStore } from '@/stores/auth';
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
  HandCoins,
  Heart,
  LinkIcon,
  Unlink,
  Search,
} from 'lucide-vue-next';
import SortTh from '@/components/SortTh.vue';
import EmptyState from '@/components/ui/EmptyState.vue';
import { formatCurrency, formatDate, formatDateTime, formatMonthName } from '@/utils/format';
import { getFeeTypeColor, getFeeTypeName } from '@/utils/fees';

const auth = useAuthStore();

const props = defineProps<{
  activeFilter: StatusFilter;
  isLoadingTransactions: boolean;
  transactionSearch: string;
  sortField: SortField;
  sortDirection: 'asc' | 'desc';
  page: number;
  toggleSort: (field: SortField) => void;
  totalRows: number;
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
  totalRows,
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

  <EmptyState v-else-if="pagedRows.length === 0" :icon="transactionSearch.trim() ? Search : CheckCircle"
    :text="transactionSearch.trim() ? 'Keine Transaktionen gefunden' : 'Keine offenen Transaktionen'">
    <p v-if="transactionSearch.trim()">Versuche einen anderen Suchbegriff</p>
    <p v-else-if="activeFilter === 'warnungen'">Alle Zahlungen wurden korrekt verarbeitet</p>
    <p v-else-if="activeFilter === 'zugeordnet'">Noch keine zugeordneten Transaktionen</p>
  </EmptyState>

  <div v-else class="card overflow-hidden">
    <div class="overflow-x-auto">
      <table class="table">
        <thead class="bg-muted">
          <tr class="text-left text-sm text-muted-foreground">
            <SortTh label="Datum" column="date" cell-class="px-3 py-3"
              :sort-key="sortField" :sort-dir="sortDirection" @sort="toggleSort" />
            <SortTh label="Zahler" column="payer" cell-class="px-3 py-3"
              :sort-key="sortField" :sort-dir="sortDirection" @sort="toggleSort" />
            <SortTh label="Beschreibung" column="description" cell-class="px-3 py-3"
              :sort-key="sortField" :sort-dir="sortDirection" @sort="toggleSort" />
            <SortTh label="Betrag" column="amount" align="right" cell-class="px-3 py-3"
              :sort-key="sortField" :sort-dir="sortDirection" @sort="toggleSort" />
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
                  row.matched ? 'text-success' : '',
                ]"
              >
                {{ formatCurrency(row.tx.amount) }}
              </td>
              <td class="px-3 py-3">
                <div class="flex flex-col items-start gap-1.5">
                  <span
                    v-if="isPartiallyAllocated(row)"
                    class="inline-flex items-center gap-1 whitespace-nowrap px-2 py-0.5 rounded-full text-xs font-medium bg-warning-soft text-warning"
                  >
                    <AlertTriangle class="h-3 w-3 shrink-0" />
                    Teilweise zugeordnet · Rest {{ formatCurrency(getTxRemaining(row.tx)) }}
                  </span>
                  <span
                    v-else-if="row.matched"
                    class="inline-flex items-center gap-1 whitespace-nowrap px-2 py-0.5 rounded-full text-xs font-medium bg-success-soft text-success"
                  >
                    <CheckCircle class="h-3 w-3 shrink-0" />
                    Zugeordnet
                  </span>
                  <span
                    v-else
                    class="inline-flex items-center gap-1 whitespace-nowrap px-2 py-0.5 rounded-full text-xs font-medium bg-warning-soft text-warning"
                  >
                    <AlertTriangle class="h-3 w-3 shrink-0" />
                    Nicht zugeordnet
                  </span>

                  <span v-if="row.tx.installmentPlanSuggestion" class="badge badge-primary">
                    <HandCoins class="h-3 w-3" />Rate · {{ row.tx.installmentPlanSuggestion.debtorName }}
                  </span>
                  <template v-if="row.tx.installmentPayment">
                    <router-link v-if="auth.isAdmin" :to="`/ratenzahlungen/${row.tx.installmentPayment.planId}`"
                      class="badge badge-primary">
                      <HandCoins class="h-3 w-3" />Ratenzahlung · {{ row.tx.installmentPayment.debtorName }}
                    </router-link>
                    <span v-else class="badge badge-primary"><HandCoins class="h-3 w-3" />Ratenzahlung</span>
                  </template>

                  <span v-if="row.tx.assignment" class="badge badge-success">
                    <Heart class="h-3 w-3" />Spende<template v-if="row.tx.assignment.householdName"> · {{ row.tx.assignment.householdName }}</template>
                  </span>

                  <div v-if="row.matched && row.tx.matches && row.tx.matches.length > 0" class="flex flex-wrap gap-1">
                    <span
                      v-for="match in row.tx.matches"
                      :key="match.id"
                      :class="getFeeTypeColor(match.expectation?.feeType || '')"
                    >
                      {{ getFeeTypeName(match.expectation?.feeType || '') }}
                      <span class="ml-1 opacity-75">{{ formatCurrency(match.expectation?.amount || 0) }}</span>
                    </span>
                  </div>

                  <button
                    v-if="row.warnings.length > 0"
                    @click="toggleWarnings(row.key)"
                    class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-warning-soft text-warning hover:bg-warning-soft transition-colors"
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
                      class="btn btn-danger btn-sm"
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
                      class="btn btn-secondary btn-sm"
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
                      class="inline-flex items-center gap-1 px-2 py-1 text-xs text-muted-foreground hover:text-destructive hover:bg-destructive-soft rounded transition-colors disabled:opacity-50"
                      title="IBAN dauerhaft ignorieren"
                    >
                      <Loader2 v-if="isDismissing === row.tx.id" class="h-3 w-3 animate-spin" />
                      <Ban v-else class="h-3 w-3" />
                      Ignorieren
                    </button>
                  </div>
                </template>

                <!-- Matched actions; plan payments are only changed by admins -->
                <template v-else-if="auth.isAdmin || !row.tx.installmentPayment">
                  <div v-if="unmatchConfirmId === row.tx.id" class="flex flex-wrap items-center justify-end gap-1.5">
                    <span class="text-xs text-muted-foreground">Zuordnung aufheben?</span>
                    <button
                      @click="unmatchTransaction(row.tx)"
                      class="btn btn-secondary btn-sm"
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
                      class="btn btn-danger btn-sm"
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
                      class="inline-flex items-center gap-1 px-2 py-1 text-xs text-muted-foreground hover:text-warning hover:bg-warning-soft rounded transition-colors disabled:opacity-50"
                      title="Zuordnung aufheben"
                    >
                      <Loader2 v-if="isUnmatching === row.tx.id" class="h-3 w-3 animate-spin" />
                      <Unlink v-else class="h-3 w-3" />
                      Aufheben
                    </button>
                    <button
                      @click="showDeleteConfirm(row.tx.id)"
                      :disabled="isUnmatching === row.tx.id || isDeletingMatched === row.tx.id"
                      class="inline-flex items-center gap-1 px-2 py-1 text-xs text-muted-foreground hover:text-destructive hover:bg-destructive-soft rounded transition-colors disabled:opacity-50"
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
            <tr v-if="expandedWarnings.has(row.key) && row.warnings.length > 0" class="border-t bg-warning-soft">
              <td colspan="6" class="px-4 py-4">
                <div class="space-y-4">
                  <div
                    v-for="warning in row.warnings"
                    :key="warning.id"
                    class="card p-4"
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
                            <span class="font-medium text-success">{{ formatCurrency(warning.transaction.amount) }}</span>
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
                          class="mt-2 p-3 bg-warning-soft rounded-lg text-sm space-y-1"
                        >
                          <div class="flex items-center gap-1 text-warning font-medium mb-1">
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
                          class="btn btn-secondary btn-sm"
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
                              class="btn btn-danger btn-sm"
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
                          class="inline-flex items-center gap-1 px-3 py-1.5 text-sm text-muted-foreground hover:text-destructive hover:bg-destructive-soft rounded-lg transition-colors disabled:opacity-50"
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
        Seite {{ page }} von {{ totalPages }} ({{ totalRows }} Einträge)
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
