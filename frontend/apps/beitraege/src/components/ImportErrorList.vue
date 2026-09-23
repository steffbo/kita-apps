<script setup lang="ts">
import { AlertTriangle } from 'lucide-vue-next';
import type { ImportError } from '@/api/types';

withDefaults(
  defineProps<{
    errors: ImportError[];
    title?: string;
    /** Total number of errors, when more occurred than are listed (history keeps max. 100). */
    total?: number;
  }>(),
  { title: 'Fehler beim Import' },
);

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('de-DE');
}

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR' }).format(amount);
}
</script>

<template>
  <div class="p-4 bg-red-50 border border-red-200 rounded-lg" role="alert">
    <div class="flex items-start gap-3">
      <AlertTriangle class="h-5 w-5 text-red-500 flex-shrink-0 mt-0.5" />
      <div class="min-w-0 flex-1">
        <p class="text-red-700 font-medium">
          {{ title }}: {{ total ?? errors.length }}
          {{ (total ?? errors.length) === 1 ? 'Buchung' : 'Buchungen' }}
        </p>
        <p class="text-sm text-red-600 mb-2">
          Diese Buchungen wurden nicht oder nicht vollständig verarbeitet. Nach Behebung der Ursache
          erneut importieren bzw. „Erneut zuordnen“ ausführen.
        </p>
        <ul class="space-y-1 text-sm text-red-800">
          <li v-for="(error, index) in errors" :key="index" class="flex flex-wrap gap-x-2">
            <span v-if="error.bookingDate" class="text-red-600">{{ formatDate(error.bookingDate) }}</span>
            <span v-if="error.payerName" class="font-medium">{{ error.payerName }}</span>
            <span>{{ formatCurrency(error.amount) }}</span>
            <span class="text-red-700">– {{ error.message }}</span>
          </li>
        </ul>
        <p v-if="total && total > errors.length" class="mt-2 text-xs text-red-600">
          {{ total - errors.length }} weitere nicht gespeichert.
        </p>
      </div>
    </div>
  </div>
</template>
