<script setup lang="ts">
import { ref, computed } from 'vue';
import { api } from '@/api';
import type { FeeExpectation, MatchSuggestion } from '@/api/types';
import { Loader2 } from 'lucide-vue-next';
import { formatCurrency, formatDate, formatMonthName } from '@/utils/format';
import { getFeeRemainingAmount, getFeeTypeName, getTxRemainingAmount } from '@/utils/fees';
import AppSheet from '@/components/ui/AppSheet.vue';
import EmptyState from '@/components/ui/EmptyState.vue';

const props = defineProps<{ suggestion: MatchSuggestion; openFees: FeeExpectation[] }>();
const emit = defineEmits<{ close: []; allocated: [] }>();

// Prefills the rows with the fees the suggestion points at, capped by what is still open.
function initialRows(): { fee: FeeExpectation; amount: number }[] {
  const rows = props.openFees.map(fee => ({ fee, amount: 0 }));
  let remaining = getTxRemainingAmount(props.suggestion.transaction);

  const applyAllocation = (feeId: string, desiredAmount: number) => {
    const row = rows.find(item => item.fee.id === feeId);
    if (!row || remaining <= 0) {
      return;
    }
    const maxAmount = getFeeRemainingAmount(row.fee);
    const amount = Math.min(desiredAmount, maxAmount, remaining);
    if (amount > 0) {
      row.amount = amount;
      remaining -= amount;
    }
  };

  if (props.suggestion.expectations && props.suggestion.expectations.length > 0) {
    for (const expectation of props.suggestion.expectations) {
      applyAllocation(expectation.id, expectation.amount);
    }
  } else if (props.suggestion.expectation) {
    applyAllocation(props.suggestion.expectation.id, props.suggestion.expectation.amount);
  }
  return rows;
}

const allocationRows = ref(initialRows());
const allocationError = ref<string | null>(null);
const isAllocating = ref(false);

const allocationTotal = computed(() =>
  allocationRows.value.reduce((sum, row) => sum + (row.amount || 0), 0)
);

const allocationRemaining = computed(() => props.suggestion.transaction.amount - allocationTotal.value);

function clampAllocationAmount(amount: number, fee: FeeExpectation): number {
  const maxFee = getFeeRemainingAmount(fee);
  const maxTx = getTxRemainingAmount(props.suggestion.transaction);
  if (amount <= 0) return 0;
  return Math.min(amount, maxFee, maxTx);
}

function assignOnlyToFee(feeId: string): void {
  const row = allocationRows.value.find(item => item.fee.id === feeId);
  if (!row) return;
  const amount = clampAllocationAmount(getFeeRemainingAmount(row.fee), row.fee);
  allocationRows.value = allocationRows.value.map(item => ({
    ...item,
    amount: item.fee.id === feeId ? amount : 0,
  }));
}

function assignRemainingToFee(feeId: string): void {
  const row = allocationRows.value.find(item => item.fee.id === feeId);
  if (!row) return;
  const remaining = allocationRemaining.value + (row.amount || 0);
  row.amount = clampAllocationAmount(remaining, row.fee);
}

async function confirmAllocation(): Promise<void> {
  const allocations = allocationRows.value
    .filter(row => row.amount > 0)
    .map(row => ({
      expectationId: row.fee.id,
      amount: row.amount,
    }));

  if (allocations.length === 0) {
    allocationError.value = 'Bitte mindestens einen Betrag zuordnen.';
    return;
  }

  if (allocationRemaining.value < -0.01) {
    allocationError.value = 'Die Summe übersteigt den Transaktionsbetrag.';
    return;
  }

  isAllocating.value = true;
  allocationError.value = null;
  try {
    await api.allocateTransaction(props.suggestion.transaction.id, allocations);
    emit('allocated');
  } catch (e) {
    allocationError.value = e instanceof Error ? e.message : 'Zuordnung fehlgeschlagen';
  } finally {
    isAllocating.value = false;
  }
}
</script>

<template>
  <AppSheet title="Transaktion zuordnen" size="lg" @close="$emit('close')">

      <div class="p-4 bg-primary-soft rounded-lg mb-6">
        <div class="flex justify-between text-sm">
          <span class="text-muted-foreground">Zahler</span>
          <span class="font-medium">{{ suggestion.transaction.payerName || 'Unbekannt' }}</span>
        </div>
        <div class="flex justify-between text-sm">
          <span class="text-muted-foreground">Datum</span>
          <span class="font-medium">{{ formatDate(suggestion.transaction.bookingDate) }}</span>
        </div>
        <div class="flex justify-between text-sm">
          <span class="text-muted-foreground">Betrag</span>
          <span class="font-semibold text-primary">{{ formatCurrency(suggestion.transaction.amount) }}</span>
        </div>
        <div v-if="(suggestion.transaction.matchedAmount ?? 0) > 0" class="flex justify-between text-sm">
          <span class="text-muted-foreground">Bereits zugeordnet</span>
          <span>{{ formatCurrency(suggestion.transaction.matchedAmount ?? 0) }}</span>
        </div>
        <div v-if="suggestion.transaction.description" class="text-xs text-muted-foreground mt-2 break-words">
          {{ suggestion.transaction.description }}
        </div>
      </div>

      <div class="space-y-3">
        <h3 class="text-sm font-medium text-muted-foreground">Offene Beiträge</h3>
        <EmptyState v-if="allocationRows.length === 0" text="Keine offenen Beiträge vorhanden." />
        <div v-else class="space-y-2">
          <div
            v-for="row in allocationRows"
            :key="row.fee.id"
            class="flex items-center justify-between gap-4 p-3 border rounded-lg"
          >
            <div>
              <p class="font-medium">{{ getFeeTypeName(row.fee.feeType) }}</p>
              <p class="text-xs text-muted-foreground">
                {{ row.fee.month ? formatMonthName(row.fee.month) + ' ' : '' }}{{ row.fee.year }}
              </p>
              <p class="text-xs text-muted-foreground">
                Rest: {{ formatCurrency(getFeeRemainingAmount(row.fee)) }}
              </p>
            </div>
            <div class="w-40">
              <input
                v-model.number="row.amount"
                type="number"
                min="0"
                step="0.01"
                :max="getFeeRemainingAmount(row.fee)"
                class="w-full px-2 py-1 border border-border rounded-lg text-sm focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
                placeholder="0,00"
              />
              <div class="flex items-center justify-end gap-2 mt-2">
                <button
                  type="button"
                  @click="assignRemainingToFee(row.fee.id)"
                  class="text-xs text-muted-foreground hover:text-foreground"
                >
                  Restbetrag
                </button>
                <button
                  type="button"
                  @click="assignOnlyToFee(row.fee.id)"
                  class="text-xs text-primary hover:text-primary/80 font-medium"
                >
                  Nur diesen Beitrag
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="mt-6 space-y-2">
        <div class="flex justify-between text-sm">
          <span class="text-muted-foreground">Zugeteilt</span>
          <span class="font-medium">{{ formatCurrency(allocationTotal) }}</span>
        </div>
        <div class="flex justify-between text-sm">
          <span class="text-muted-foreground">Restbetrag</span>
          <span :class="allocationRemaining < -0.01 ? 'text-destructive font-medium' : 'font-medium'">
            {{ formatCurrency(allocationRemaining) }}
          </span>
        </div>
        <p v-if="allocationRemaining > 0.01" class="text-xs text-muted-foreground">
          Der Restbetrag bleibt offen und kann später einem anderen Beitrag oder Kind zugeordnet werden.
        </p>
        <p v-if="allocationError" class="text-sm text-destructive">
          {{ allocationError }}
        </p>
      </div>

      <template #footer>
        <button type="button" class="btn btn-secondary" @click="$emit('close')">Abbrechen</button>
        <button type="button" class="btn btn-primary"
          :disabled="isAllocating || allocationTotal <= 0 || allocationRemaining < -0.01"
          @click="confirmAllocation">
          <Loader2 v-if="isAllocating" class="animate-spin" />Zuordnen
        </button>
      </template>
  </AppSheet>
</template>
