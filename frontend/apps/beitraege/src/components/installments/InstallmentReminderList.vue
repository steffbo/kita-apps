<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { api } from '@/api';
import type { InstallmentDetail } from '@/api/types';
import { AlertTriangle, Loader2 } from 'lucide-vue-next';
import { formatCurrency } from '@/utils/format';
import { nextReminderLabel, overdueRates } from '@/utils/installments';

// Plans with overdue rates that are due for a reminder. The parent remounts this list to reload it.
const emit = defineEmits<{ open: [detail: InstallmentDetail] }>();

const cases = ref<InstallmentDetail[]>([]);
const isLoading = ref(true);
const error = ref<string | null>(null);

onMounted(async () => {
  try {
    cases.value = await api.getInstallmentReminderCases();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ratenzahlungen konnten nicht geladen werden';
  } finally {
    isLoading.value = false;
  }
});
</script>

<template>
  <div v-if="isLoading" class="flex justify-center py-4">
    <Loader2 class="h-5 w-5 animate-spin text-muted-foreground" />
  </div>
  <div v-else-if="error" class="note note-danger" role="alert"><AlertTriangle />{{ error }}</div>
  <ul v-else-if="cases.length" class="mb-4 space-y-2">
    <li v-for="item in cases" :key="item.plan.id">
      <button type="button" class="w-full rounded-xl border bg-card p-4 text-left transition-colors hover:bg-accent"
        @click="emit('open', item)">
        <div class="flex items-center justify-between gap-3">
          <span class="font-medium">{{ item.plan.debtorName }}</span>
          <span class="whitespace-nowrap font-semibold">{{ formatCurrency(item.summary.overdue) }}</span>
        </div>
        <div class="mt-2 flex flex-wrap items-center gap-1.5">
          <span class="badge badge-primary">Ratenzahlung</span>
          <span class="badge" :class="nextReminderLabel(item) === 'Mahnung' ? 'badge-danger' : 'badge-warning'">
            {{ nextReminderLabel(item) }}
          </span>
          <span v-if="item.plan.emails.length === 0" class="badge badge-danger">Keine E-Mail</span>
          <span class="text-xs text-muted-foreground">
            {{ overdueRates(item).length }} {{ overdueRates(item).length === 1 ? 'Rate' : 'Raten' }} überfällig
          </span>
        </div>
      </button>
    </li>
  </ul>
</template>
