<script setup lang="ts">
import type { ParentReport } from '@/api/types';
import { formatDate } from '@/utils/format';
defineProps<{ reports: ParentReport[] }>();
</script>
<template>
  <ul class="space-y-3">
    <li v-for="report in reports" :key="report.id" class="border-t pt-3">
      <div class="flex items-start justify-between gap-3">
        <p class="whitespace-pre-wrap font-semibold">{{ report.message }}</p>
        <span class="shrink-0 whitespace-nowrap rounded-full px-2.5 py-0.5 text-xs font-semibold" :class="
          report.status === 'OPEN'
            ? 'bg-amber-100 text-amber-800 dark:bg-amber-950/40 dark:text-amber-300'
            : 'bg-green-100 text-green-800 dark:bg-green-950/40 dark:text-green-300'">
          {{ report.status === 'OPEN' ? 'Offen' : '✓ Erledigt' }}
        </span>
      </div>
      <p class="text-xs text-muted-foreground">
        {{ formatDate(report.createdAt) }}<template v-if="report.parentName"> · von {{ report.parentName }}</template>
      </p>
      <p v-if="report.response" class="mt-2 rounded-lg bg-muted p-3 text-sm">
        <span class="font-semibold">Antwort vom Vorstand:</span> {{ report.response }}
      </p>
    </li>
  </ul>
</template>
