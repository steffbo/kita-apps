<script setup lang="ts">
import { formatCurrency, formatDate } from '@/utils/format';

// Read-only donations by year. Takes the reduced parent payload as well as the staff report.
defineProps<{
  years: {
    year: number;
    total: number;
    transactions: { id: string; bookingDate: string; amount: number; payerName?: string }[];
  }[];
}>();
</script>

<template>
  <div class="space-y-4">
    <section v-for="year in years" :key="year.year" :aria-label="`Spenden ${year.year}`">
      <div class="mb-1 flex items-baseline justify-between gap-3 text-sm">
        <h3 class="font-semibold">{{ year.year }}</h3>
        <span class="font-semibold tabular-nums">{{ formatCurrency(year.total) }}</span>
      </div>
      <ul class="card divide-y">
        <li v-for="tx in year.transactions" :key="tx.id" class="flex items-center gap-3 px-4 py-3">
          <span class="min-w-0 flex-1">
            <span class="block truncate font-medium">{{ tx.payerName || 'Unbekannt' }}</span>
            <span class="block text-[13px] text-muted-foreground">{{ formatDate(tx.bookingDate) }}</span>
          </span>
          <span class="shrink-0 tabular-nums">{{ formatCurrency(tx.amount) }}</span>
        </li>
      </ul>
    </section>
  </div>
</template>
