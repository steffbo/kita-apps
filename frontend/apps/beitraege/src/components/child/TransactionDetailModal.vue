<script setup lang="ts">
import { ref } from 'vue';
import { api } from '@/api';
import type { BankTransaction, FeeExpectation } from '@/api/types';
import { Trash2, Loader2, Unlink, CreditCard } from 'lucide-vue-next';
import { formatCurrency, formatDate } from '@/utils/format';
import { getFeeMatches } from '@/utils/fees';

const props = defineProps<{ fee: FeeExpectation }>();
const emit = defineEmits<{ close: []; changed: [] }>();

const selectedTransaction = ref<BankTransaction | null>(getFeeMatches(props.fee)[0]?.transaction ?? null);
const transactionAction = ref<'unmatch' | 'delete' | null>(null);
const isUnmatchingTransaction = ref(false);
const isDeletingTransaction = ref(false);
const transactionActionError = ref<string | null>(null);

function requestTransactionAction(action: 'unmatch' | 'delete'): void {
  transactionAction.value = action;
  transactionActionError.value = null;
}

function cancelTransactionAction(): void {
  transactionAction.value = null;
  transactionActionError.value = null;
}

async function confirmTransactionAction(): Promise<void> {
  if (!selectedTransaction.value || !transactionAction.value) return;
  const deleteTransaction = transactionAction.value === 'delete';
  if (deleteTransaction) {
    isDeletingTransaction.value = true;
  } else {
    isUnmatchingTransaction.value = true;
  }
  transactionActionError.value = null;
  try {
    await api.unmatchTransaction(selectedTransaction.value.id, { deleteTransaction });
    emit('changed');
  } catch (e) {
    transactionActionError.value = e instanceof Error ? e.message : 'Aktion fehlgeschlagen';
  } finally {
    if (deleteTransaction) {
      isDeletingTransaction.value = false;
    } else {
      isUnmatchingTransaction.value = false;
    }
  }
}
import AppSheet from '@/components/ui/AppSheet.vue';
</script>

<template>
  <AppSheet title="Transaktionsdetails" hide-title @close="$emit('close')">

      <div class="flex items-center justify-between mb-6">
        <div class="flex items-center gap-3">
          <div class="p-2 bg-success-soft rounded-lg">
            <CreditCard class="h-6 w-6 text-success" />
          </div>
          <h2 class="text-base font-semibold">Transaktionsdetails</h2>
        </div>
      </div>

      <div v-if="fee && getFeeMatches(fee).length > 1" class="mb-6">
        <p class="text-sm font-medium text-muted-foreground mb-2">Zahlungen ({{ getFeeMatches(fee).length }})</p>
        <div class="space-y-2">
          <button
            v-for="match in getFeeMatches(fee)"
            :key="match.id"
            @click="selectedTransaction = match.transaction ?? null"
            class="w-full flex items-center justify-between p-2 border rounded-lg text-left hover:bg-accent"
          >
            <div>
              <p class="text-sm font-medium">{{ match.transaction?.payerName || 'Unbekannt' }}</p>
              <p class="text-xs text-muted-foreground">
                {{ match.transaction?.bookingDate ? formatDate(match.transaction.bookingDate) : 'Kein Datum' }}
              </p>
            </div>
            <div class="text-sm font-semibold text-success">
              {{ formatCurrency(match.amount) }}
            </div>
          </button>
        </div>
        <p v-if="!selectedTransaction" class="text-xs text-muted-foreground mt-2">
          Wähle eine Zahlung, um die Details anzuzeigen.
        </p>
      </div>

      <div v-if="selectedTransaction" class="space-y-4">
        <div>
          <p class="text-sm text-muted-foreground">Zahler</p>
          <p class="font-medium">{{ selectedTransaction.payerName || 'Unbekannt' }}</p>
        </div>

        <div>
          <p class="text-sm text-muted-foreground">Buchungsdatum</p>
          <p class="font-medium">{{ formatDate(selectedTransaction.bookingDate) }}</p>
        </div>

        <div v-if="selectedTransaction.payerIban">
          <p class="text-sm text-muted-foreground">IBAN</p>
          <p class="font-mono text-sm">{{ selectedTransaction.payerIban }}</p>
        </div>

        <div v-if="selectedTransaction.description">
          <p class="text-sm text-muted-foreground">Verwendungszweck</p>
          <p class="text-sm text-foreground break-words">{{ selectedTransaction.description }}</p>
        </div>

        <div>
          <p class="text-sm text-muted-foreground">Betrag</p>
          <p class="font-semibold text-success text-lg">{{ formatCurrency(selectedTransaction.amount) }}</p>
        </div>

        <div>
          <p class="text-sm text-muted-foreground">Importiert am</p>
          <p class="text-sm text-muted-foreground">{{ formatDate(selectedTransaction.importedAt) }}</p>
        </div>
      </div>

      <div class="mt-6 space-y-3">
        <div
          v-if="transactionAction"
          :class="[
            'p-3 rounded-lg text-sm',
            transactionAction === 'delete' ? 'bg-destructive-soft text-destructive' : 'bg-warning-soft text-warning'
          ]"
        >
          <p class="font-medium">
            {{ transactionAction === 'delete'
              ? 'Transaktion wirklich löschen?'
              : 'Zuordnung wirklich aufheben?' }}
          </p>
          <p class="text-xs mt-1">
            {{ transactionAction === 'delete'
              ? 'Die Transaktion wird gelöscht (inklusive aller Zuordnungen).'
              : 'Der Beitrag wird wieder als offen geführt. Falls mehrere Beiträge mit der Transaktion verknüpft sind, werden alle Zuordnungen aufgehoben.' }}
          </p>
          <div class="flex justify-end gap-2 mt-3">
            <button
              @click="confirmTransactionAction"
              :disabled="isUnmatchingTransaction || isDeletingTransaction"
              :class="[
                'px-3 py-1.5 text-xs text-primary-foreground rounded transition-colors disabled:opacity-50',
                transactionAction === 'delete' ? 'bg-destructive hover:bg-destructive' : 'bg-warning hover:bg-warning'
              ]"
            >
              <Loader2 v-if="isUnmatchingTransaction || isDeletingTransaction" class="h-3 w-3 animate-spin" />
              <span v-else>Ja</span>
            </button>
            <button
              @click="cancelTransactionAction"
              class="px-3 py-1.5 text-xs bg-muted text-foreground rounded hover:bg-accent"
            >
              Nein
            </button>
          </div>
        </div>

        <p v-if="transactionActionError" class="text-sm text-destructive">
          {{ transactionActionError }}
        </p>

        <div v-if="!transactionAction && selectedTransaction" class="flex flex-col gap-2">
          <button
            @click="requestTransactionAction('unmatch')"
            :disabled="isUnmatchingTransaction || isDeletingTransaction"
            class="inline-flex items-center gap-2 px-3 py-2 text-sm text-warning bg-warning-soft hover:bg-warning-soft rounded-lg transition-colors disabled:opacity-50"
          >
            <Unlink class="h-4 w-4" />
            Zuordnung aufheben
          </button>
          <button
            @click="requestTransactionAction('delete')"
            :disabled="isUnmatchingTransaction || isDeletingTransaction"
            class="inline-flex items-center gap-2 px-3 py-2 text-sm text-destructive bg-destructive-soft hover:bg-destructive-soft rounded-lg transition-colors disabled:opacity-50"
          >
            <Trash2 class="h-4 w-4" />
            Transaktion löschen
          </button>
        </div>

      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="$emit('close')">Schließen</button>
      </template>
  </AppSheet>
</template>
