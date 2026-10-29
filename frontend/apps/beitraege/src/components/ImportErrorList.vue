<script setup lang="ts">
import { AlertTriangle } from 'lucide-vue-next';
import type { ImportError } from '@/api/types';
import { formatCurrency, formatDate } from '@/utils/format';

withDefaults(
  defineProps<{
    errors: ImportError[];
    title?: string;
    /** Total number of errors, when more occurred than are listed (history keeps max. 100). */
    total?: number;
  }>(),
  { title: 'Fehler beim Import' },
);

</script>

<template>
  <div class="note note-danger" role="alert">
    <div class="flex items-start gap-3">
      <AlertTriangle class="h-5 w-5 text-destructive flex-shrink-0 mt-0.5" />
      <div class="min-w-0 flex-1">
        <p class="text-destructive font-medium">
          {{ title }}: {{ total ?? errors.length }}
          {{ (total ?? errors.length) === 1 ? 'Buchung' : 'Buchungen' }}
        </p>
        <p class="text-sm text-destructive mb-2">
          Diese Buchungen wurden nicht oder nicht vollständig verarbeitet. Nach Behebung der Ursache
          erneut importieren bzw. „Erneut zuordnen“ ausführen.
        </p>
        <ul class="space-y-1 text-sm text-destructive">
          <li v-for="(error, index) in errors" :key="index" class="flex flex-wrap gap-x-2">
            <span v-if="error.bookingDate" class="text-destructive">{{ formatDate(error.bookingDate) }}</span>
            <span v-if="error.payerName" class="font-medium">{{ error.payerName }}</span>
            <span>{{ formatCurrency(error.amount) }}</span>
            <span class="text-destructive">– {{ error.message }}</span>
          </li>
        </ul>
        <p v-if="total && total > errors.length" class="mt-2 text-xs text-destructive">
          {{ total - errors.length }} weitere nicht gespeichert.
        </p>
      </div>
    </div>
  </div>
</template>
