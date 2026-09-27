<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import { api } from '@/api';
import type { OwnFees } from '@/api/types';
import { formatCurrency, formatDate, formatMonthName, todayISO } from '@/utils/format';
import { getFeeTypeName } from '@/utils/fees';
import ReportDialog from '@/components/family/ReportDialog.vue';
const year = ref(Number(todayISO().slice(0, 4)));
const fees = ref<OwnFees | null>(null);
const error = ref('');
const reportId = ref<string | null>(null);
const years = Array.from({ length: 5 }, (_, i) => year.value + 1 - i);
const status: Record<string, string> = { OPEN: 'Offen', PAID: 'Bezahlt', OVERDUE: 'Überfällig' };
function tone(value: string) {
  return value === 'PAID' ? 'bg-green-100 text-green-800 dark:bg-green-950/40 dark:text-green-300'
    : value === 'OVERDUE' ? 'bg-red-100 text-red-800 dark:bg-red-950/40 dark:text-red-300'
      : 'bg-amber-100 text-amber-800 dark:bg-amber-950/40 dark:text-amber-300';
}
async function load() {
  error.value = '';
  try { fees.value = await api.getOwnFees(year.value); }
  catch (e) { error.value = e instanceof Error ? e.message : 'Beiträge konnten nicht geladen werden'; }
}
onMounted(load); watch(year, load);
</script>
<template>
  <div class="space-y-5">
    <h1 class="text-2xl font-bold">Deine Beiträge</h1>
    <label class="block max-w-40 text-sm font-medium">Jahr
      <select v-model.number="year" class="mt-1 w-full rounded-lg border px-3 py-2">
        <option v-for="y in years" :key="y" :value="y">{{ y }}</option>
      </select>
    </label>
    <p v-if="error" role="alert" class="text-red-700 dark:text-red-300">{{ error }}</p>
    <div v-if="fees" class="grid gap-3 sm:grid-cols-2">
      <div class="rounded-2xl border bg-card p-4">Offen: <strong>{{ formatCurrency(fees.openTotal) }}</strong></div>
      <div class="rounded-2xl border bg-card p-4">Bezahlt: <strong>{{ formatCurrency(fees.paidTotal) }}</strong></div>
    </div>
    <p v-if="fees && !fees.items.length" class="text-muted-foreground">Für dieses Jahr gibt es keine Beiträge.</p>
    <div v-else-if="fees" class="space-y-3">
      <article v-for="fee in fees.items" :key="fee.id" class="rounded-2xl border bg-card p-4">
        <div class="flex flex-wrap items-start justify-between gap-2">
          <div><h2 class="font-bold">{{ fee.childName }}</h2>
            <p class="text-sm text-muted-foreground">{{ getFeeTypeName(fee.feeType) }} ·
              {{ fee.month ? formatMonthName(fee.month) : 'Jahr' }} {{ fee.year }}</p>
          </div>
          <span class="rounded-full px-2 py-1 text-xs font-semibold" :class="tone(fee.status)">
            {{ status[fee.status] ?? fee.status }}
          </span>
        </div>
        <div class="mt-3 flex flex-wrap items-end justify-between gap-3 text-sm">
          <div><p>Betrag: <strong>{{ formatCurrency(fee.amount) }}</strong></p>
            <p>Fällig: {{ formatDate(fee.dueDate) }}</p></div>
          <button class="text-primary underline" @click="reportId = fee.id">Fehler melden</button>
        </div>
      </article>
    </div>
    <ReportDialog v-if="reportId" topic="FEE" :reference-id="reportId"
      @close="reportId = null" @saved="reportId = null" />
  </div>
</template>
