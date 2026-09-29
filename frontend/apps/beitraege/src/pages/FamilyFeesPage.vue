<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { CornerDownRight, MessageSquareWarning } from 'lucide-vue-next';
import { api } from '@/api';
import type { OwnFees } from '@/api/types';
import { formatCurrency, formatDate, formatMonthName, todayISO } from '@/utils/format';
import { getFeePeriodLabel, getFeeTypeColor, getFeeTypeName } from '@/utils/fees';
import ReportDialog from '@/components/family/ReportDialog.vue';
type FeeRow = OwnFees['items'][number];
const year = ref(Number(todayISO().slice(0, 4)));
const fees = ref<OwnFees | null>(null);
const error = ref('');
const reportId = ref<string | null>(null);
const onlyOpen = ref(false);
// Fee data in the portal starts with January 2025.
const firstYear = 2025;
const years = Array.from({ length: Math.max(1, year.value - firstYear + 1) }, (_, i) => year.value - i);
const status: Record<string, string> = { OPEN: 'Offen', PAID: 'Bezahlt', OVERDUE: 'Überfällig' };
function tone(value: string) {
  return value === 'PAID' ? 'bg-green-100 text-green-800 dark:bg-green-950/40 dark:text-green-300'
    : value === 'OVERDUE' ? 'bg-red-100 text-red-800 dark:bg-red-950/40 dark:text-red-300'
      : 'bg-amber-100 text-amber-800 dark:bg-amber-950/40 dark:text-amber-300';
}
// A Mahngebühr is listed right below its fee; one without its fee in the list names it instead.
const groups = computed(() => {
  const items = fees.value?.items ?? [];
  const ids = new Set(items.map(f => f.id));
  const reminders = new Map<string, FeeRow[]>();
  for (const f of items) {
    if (f.reminderForId && ids.has(f.reminderForId)) {
      reminders.set(f.reminderForId, [...(reminders.get(f.reminderForId) ?? []), f]);
    }
  }
  return items.filter(f => !f.reminderForId || !ids.has(f.reminderForId))
    .map(fee => ({ fee, reminders: reminders.get(fee.id) ?? [] }))
    .filter(g => !onlyOpen.value || [g.fee, ...g.reminders].some(f => f.status !== 'PAID'));
});
const showChild = computed(() => new Set(fees.value?.items.map(f => f.childId)).size > 1);
function period(fee: FeeRow) {
  return getFeePeriodLabel({ year: fee.year, month: fee.month, reminderFor: fee.baseFeeType
    ? { feeType: fee.baseFeeType, year: fee.baseYear ?? fee.year, month: fee.baseMonth } : undefined });
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
    <h1 class="text-2xl font-bold">Beiträge</h1>
    <label class="block max-w-40 text-sm font-medium">Jahr
      <select v-model.number="year" class="mt-1 w-full rounded-lg border px-3 py-2">
        <option v-for="y in years" :key="y" :value="y">{{ y }}</option>
      </select>
    </label>
    <p v-if="error" role="alert" class="text-red-700 dark:text-red-300">{{ error }}</p>
    <div v-if="fees" class="grid gap-3 sm:grid-cols-2">
      <button type="button" :aria-pressed="onlyOpen" title="Nur offene Beiträge anzeigen"
        class="rounded-2xl border bg-card p-4 text-left transition-colors hover:border-primary"
        :class="onlyOpen ? 'border-primary ring-2 ring-primary' : ''" @click="onlyOpen = !onlyOpen">
        Offen: <strong>{{ formatCurrency(fees.openTotal) }}</strong>
      </button>
      <div class="rounded-2xl border bg-card p-4">Bezahlt: <strong>{{ formatCurrency(fees.paidTotal) }}</strong></div>
    </div>
    <div v-if="onlyOpen" class="flex flex-wrap items-center gap-3 text-sm">
      <span class="text-muted-foreground">Nur offene Beiträge</span>
      <button type="button" class="rounded-lg border bg-card px-3 py-1.5 font-medium hover:bg-muted"
        @click="onlyOpen = false">Alle anzeigen</button>
    </div>
    <p v-if="fees && !groups.length" class="text-muted-foreground">
      {{ onlyOpen ? 'Keine offenen Beiträge.' : 'Für dieses Jahr gibt es keine Beiträge.' }}
    </p>
    <div v-else-if="fees" class="overflow-x-auto rounded-2xl border bg-card">
      <table class="w-full text-sm">
        <thead class="border-b text-left text-muted-foreground">
          <tr>
            <th class="px-4 py-2 font-medium">Art</th>
            <th class="px-4 py-2 font-medium">Zeitraum</th>
            <th v-if="showChild" class="px-4 py-2 font-medium">Kind</th>
            <th class="px-4 py-2 text-right font-medium">Betrag</th>
            <th class="px-4 py-2 font-medium">Fällig</th>
            <th class="px-4 py-2 font-medium">Status</th>
            <th class="px-2 py-2"><span class="sr-only">Aktionen</span></th>
          </tr>
        </thead>
        <tbody>
          <!-- Stripes per fee with its reminders, so the global zebra rule does not split them. -->
          <template v-for="(g, gi) in groups" :key="g.fee.id">
            <tr v-for="(fee, i) in [g.fee, ...g.reminders]" :key="fee.id"
              :class="gi % 2 ? 'bg-muted/45' : 'bg-card'">
              <td class="whitespace-nowrap px-4 py-2">
                <CornerDownRight v-if="i > 0" class="mr-1 inline h-4 w-4 text-muted-foreground" />
                <span class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium"
                  :class="getFeeTypeColor(fee.feeType)">{{ getFeeTypeName(fee.feeType) }}</span>
              </td>
              <td class="whitespace-nowrap px-4 py-2">{{ i > 0 ? '' : period(fee) }}</td>
              <td v-if="showChild" class="whitespace-nowrap px-4 py-2">{{ fee.childName }}</td>
              <td class="whitespace-nowrap px-4 py-2 text-right font-semibold">
                {{ formatCurrency(fee.amount) }}
                <span v-if="fee.status !== 'PAID' && fee.paidAmount > 0"
                  class="block text-xs font-normal text-muted-foreground">
                  bezahlt {{ formatCurrency(fee.paidAmount) }}</span>
              </td>
              <td class="whitespace-nowrap px-4 py-2">{{ formatDate(fee.dueDate) }}</td>
              <td class="whitespace-nowrap px-4 py-2">
                <span class="rounded-full px-2 py-0.5 text-xs font-semibold" :class="tone(fee.status)">
                  {{ status[fee.status] ?? fee.status }}</span>
                <span v-if="fee.paidAt" class="ml-2 text-xs text-muted-foreground">
                  am {{ formatDate(fee.paidAt) }}</span>
              </td>
              <td class="px-2 py-2 text-right">
                <button type="button" title="Fehler melden"
                  :aria-label="`Fehler melden: ${getFeeTypeName(fee.feeType)} ${fee.month
                    ? formatMonthName(fee.month) + ' ' : ''}${fee.year}`"
                  class="rounded p-1.5 text-muted-foreground hover:bg-muted hover:text-primary"
                  @click="reportId = fee.id"><MessageSquareWarning class="h-4 w-4" /></button>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>
    <ReportDialog v-if="reportId" topic="FEE" :reference-id="reportId"
      @close="reportId = null" @saved="reportId = null" />
  </div>
</template>
