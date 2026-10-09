<script setup lang="ts">
import { toRefs } from 'vue';
import type { ReminderCase } from '@/api/types';
import { Mail, Clock } from 'lucide-vue-next';
import { formatCurrency, formatDate, formatDueIn } from '@/utils/format';
import { feeChipClass, feeTypeLabel, feeTypesIn } from '@/utils/reminders';

const props = defineProps<{
  selectedHouseholdId: string | null;
  filteredCases: ReminderCase[];
  selectedCase: ReminderCase | null;
  lastContactOf: (item: ReminderCase) => string | null;
  hasBlockedEmail: (item: ReminderCase) => boolean;
  openCase: (householdId: string) => void;
}>();

const {
  selectedHouseholdId,
  filteredCases,
  selectedCase,
  lastContactOf,
  hasBlockedEmail,
  openCase,
} = toRefs(props);
</script>

<template>
  <div :class="selectedCase ? 'hidden lg:block' : ''">
    <ul class="space-y-2">
      <li v-for="item in filteredCases" :key="item.householdId">
        <button
          class="w-full text-left p-4 border rounded-xl bg-card transition-colors hover:bg-accent"
          :class="item.householdId === selectedHouseholdId ? 'border-primary ring-1 ring-primary' : 'border-border'"
          @click="openCase(item.householdId)"
        >
          <div class="flex items-center justify-between gap-3">
            <span class="font-medium text-foreground">{{ item.householdName }}</span>
            <span class="font-semibold text-foreground whitespace-nowrap">{{ formatCurrency(item.totalRemaining) }}</span>
          </div>
          <div class="flex flex-wrap items-center gap-1.5 mt-2">
            <span
              v-for="feeType in feeTypesIn(item)"
              :key="feeType"
              class="px-2 py-0.5 text-xs rounded-full font-medium"
              :class="feeChipClass(feeType)"
            >
              {{ feeTypeLabel(feeType) }}
            </span>
            <span
              v-if="hasBlockedEmail(item)"
              class="px-2 py-0.5 text-xs rounded-full font-medium bg-red-600 text-white"
            >
              Keine E-Mail
            </span>
          </div>
          <div class="flex flex-wrap items-center gap-4 mt-2 text-xs text-muted-foreground">
            <span class="inline-flex items-center gap-1" :title="formatDate(item.nextActionAt)">
              <Clock class="h-3.5 w-3.5" />
              Nächste Aktion: {{ formatDueIn(item.nextActionAt) }}
            </span>
            <span v-if="lastContactOf(item)" class="inline-flex items-center gap-1">
              <Mail class="h-3.5 w-3.5" />
              Letzter Kontakt: {{ formatDate(lastContactOf(item) ?? undefined) }}
            </span>
          </div>
        </button>
      </li>
    </ul>
  </div>
</template>
