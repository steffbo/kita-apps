<script setup lang="ts">
import { toRefs } from 'vue';
import type { CareHoursHistoryEntry, LegalHoursHistoryEntry } from '@/api/types';
import { formatCareHours } from '@/utils/child';

const props = defineProps<{
  careHoursHistory: CareHoursHistoryEntry[];
  legalHoursHistory: LegalHoursHistoryEntry[];
  formatHistoryRange: (entry: { effectiveFrom: string; effectiveUntil?: string | null }) => string;
}>();

const {
  careHoursHistory,
  legalHoursHistory,
  formatHistoryRange,
} = toRefs(props);
</script>

<template>
  <div v-if="legalHoursHistory.length > 0" class="mt-2 space-y-1">
    <p class="text-xs uppercase tracking-wide text-muted-foreground">Historie Rechtsanspruch</p>
    <div
      v-for="entry in legalHoursHistory"
      :key="entry.id"
      class="flex items-center justify-between text-sm text-muted-foreground"
    >
      <span>{{ formatHistoryRange(entry) }}</span>
      <span class="font-medium text-foreground">{{ formatCareHours(entry.legalHours) }}</span>
    </div>
  </div>
  <div v-if="careHoursHistory.length > 0" class="mt-2 space-y-1">
    <p class="text-xs uppercase tracking-wide text-muted-foreground">Historie Betreuungszeit</p>
    <div
      v-for="entry in careHoursHistory"
      :key="entry.id"
      class="flex items-center justify-between text-sm text-muted-foreground"
    >
      <span>{{ formatHistoryRange(entry) }}</span>
      <span class="font-medium text-foreground">{{ formatCareHours(entry.careHours) }}</span>
    </div>
  </div>
</template>
