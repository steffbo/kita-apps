<script setup lang="ts">
import { ref } from 'vue';
import { api } from '@/api';
import type { BankTransaction, FeeExpectation } from '@/api/types';
import { Trash2, Loader2, X, Unlink, CreditCard } from 'lucide-vue-next';
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
</script>

<template>
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
    @click.self="$emit('close')"
  >
    <div class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4 p-6">
      <div class="flex items-center justify-between mb-6">
        <div class="flex items-center gap-3">
          <div class="p-2 bg-green-100 rounded-lg">
            <CreditCard class="h-6 w-6 text-green-600" />
          </div>
          <h2 class="text-xl font-semibold">Transaktionsdetails</h2>
        </div>
        <button @click="$emit('close')" class="p-1 hover:bg-gray-100 rounded">
          <X class="h-5 w-5" />
        </button>
      </div>

      <div v-if="fee && getFeeMatches(fee).length > 1" class="mb-6">
        <p class="text-sm font-medium text-gray-600 mb-2">Zahlungen ({{ getFeeMatches(fee).length }})</p>
        <div class="space-y-2">
          <button
            v-for="match in getFeeMatches(fee)"
            :key="match.id"
            @click="selectedTransaction = match.transaction ?? null"
            class="w-full flex items-center justify-between p-2 border rounded-lg text-left hover:bg-gray-50"
          >
            <div>
              <p class="text-sm font-medium">{{ match.transaction?.payerName || 'Unbekannt' }}</p>
              <p class="text-xs text-gray-500">
                {{ match.transaction?.bookingDate ? formatDate(match.transaction.bookingDate) : 'Kein Datum' }}
              </p>
            </div>
            <div class="text-sm font-semibold text-green-600">
              {{ formatCurrency(match.amount) }}
            </div>
          </button>
        </div>
        <p v-if="!selectedTransaction" class="text-xs text-gray-500 mt-2">
          Wähle eine Zahlung, um die Details anzuzeigen.
        </p>
      </div>

      <div v-if="selectedTransaction" class="space-y-4">
        <div>
          <p class="text-sm text-gray-500">Zahler</p>
          <p class="font-medium">{{ selectedTransaction.payerName || 'Unbekannt' }}</p>
        </div>

        <div>
          <p class="text-sm text-gray-500">Buchungsdatum</p>
          <p class="font-medium">{{ formatDate(selectedTransaction.bookingDate) }}</p>
        </div>

        <div v-if="selectedTransaction.payerIban">
          <p class="text-sm text-gray-500">IBAN</p>
          <p class="font-mono text-sm">{{ selectedTransaction.payerIban }}</p>
        </div>

        <div v-if="selectedTransaction.description">
          <p class="text-sm text-gray-500">Verwendungszweck</p>
          <p class="text-sm text-gray-700 break-words">{{ selectedTransaction.description }}</p>
        </div>

        <div>
          <p class="text-sm text-gray-500">Betrag</p>
          <p class="font-semibold text-green-600 text-lg">{{ formatCurrency(selectedTransaction.amount) }}</p>
        </div>

        <div>
          <p class="text-sm text-gray-500">Importiert am</p>
          <p class="text-sm text-gray-600">{{ formatDate(selectedTransaction.importedAt) }}</p>
        </div>
      </div>

      <div class="mt-6 space-y-3">
        <div
          v-if="transactionAction"
          :class="[
            'p-3 rounded-lg text-sm',
            transactionAction === 'delete' ? 'bg-red-50 text-red-800' : 'bg-amber-50 text-amber-800'
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
                'px-3 py-1.5 text-xs text-white rounded transition-colors disabled:opacity-50',
                transactionAction === 'delete' ? 'bg-red-600 hover:bg-red-700' : 'bg-amber-600 hover:bg-amber-700'
              ]"
            >
              <Loader2 v-if="isUnmatchingTransaction || isDeletingTransaction" class="h-3 w-3 animate-spin" />
              <span v-else>Ja</span>
            </button>
            <button
              @click="cancelTransactionAction"
              class="px-3 py-1.5 text-xs bg-gray-200 text-gray-700 rounded hover:bg-gray-300"
            >
              Nein
            </button>
          </div>
        </div>

        <p v-if="transactionActionError" class="text-sm text-red-600">
          {{ transactionActionError }}
        </p>

        <div v-if="!transactionAction && selectedTransaction" class="flex flex-col gap-2">
          <button
            @click="requestTransactionAction('unmatch')"
            :disabled="isUnmatchingTransaction || isDeletingTransaction"
            class="inline-flex items-center gap-2 px-3 py-2 text-sm text-amber-700 bg-amber-50 hover:bg-amber-100 rounded-lg transition-colors disabled:opacity-50"
          >
            <Unlink class="h-4 w-4" />
            Zuordnung aufheben
          </button>
          <button
            @click="requestTransactionAction('delete')"
            :disabled="isUnmatchingTransaction || isDeletingTransaction"
            class="inline-flex items-center gap-2 px-3 py-2 text-sm text-red-700 bg-red-50 hover:bg-red-100 rounded-lg transition-colors disabled:opacity-50"
          >
            <Trash2 class="h-4 w-4" />
            Transaktion löschen
          </button>
        </div>

        <div class="flex justify-end">
          <button
            @click="$emit('close')"
            class="px-4 py-2 bg-gray-100 text-gray-700 hover:bg-gray-200 rounded-lg transition-colors"
          >
            Schließen
          </button>
        </div>
      </div>
    </div>
  </div></template>
