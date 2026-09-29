<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { RouterLink } from 'vue-router';
import { Plus } from 'lucide-vue-next';
import { api } from '@/api';
import type { ParentWorkAccount, ParentWorkOverview, ParentWorkRule } from '@/api/types';
import { formatHours } from '@/utils/format';
import { useTableSort } from '@/composables/useTableSort';
import EntryDialog from '@/components/parent-work/EntryDialog.vue';
import SortTh from '@/components/SortTh.vue';
import SearchInput from '@/components/SearchInput.vue';

type Filter = 'fulfilled' | 'open' | 'none' | 'submitted';

const overview = ref<ParentWorkOverview | null>(null);
const rules = ref<ParentWorkRule[]>([]);
const year = ref<number | null>(null);
const search = ref('');
const filter = ref<Filter | null>(null);
const showEntry = ref(false);
const entryHouseholdId = ref<string | undefined>();
const loading = ref(false);
const error = ref('');
const years = computed(() => {
  const y = year.value ?? overview.value?.kitaYear;
  if (y === undefined || y === null) return [];
  const ruleYears = rules.value.map(r => Number(r.validFrom.slice(0, 4)));
  const first = Math.min(y, ...ruleYears);
  const last = Math.max(y, ...ruleYears);
  return Array.from({ length: last - first + 1 }, (_, i) => last - i);
});
const households = computed(() => overview.value?.households ?? []);
const matchers: Record<Filter, (h: ParentWorkAccount) => boolean> = {
  fulfilled: h => h.openMinutes <= 0,
  open: h => h.openMinutes > 0,
  none: h => h.openMinutes > 0 && h.doneMinutes === 0,
  submitted: h => h.submittedCount > 0,
};
const cards = computed(() => [
  { key: null, label: 'Familien', value: households.value.length, hint: 'Alle anzeigen' },
  { key: 'fulfilled' as const, label: 'Alle Stunden geleistet', value: households.value.filter(matchers.fulfilled).length },
  { key: 'open' as const, label: 'Stunden noch offen', value: households.value.filter(matchers.open).length },
  { key: 'none' as const, label: 'Noch keine Stunden geleistet', value: households.value.filter(matchers.none).length },
  { key: 'submitted' as const, label: 'Unbestätigte Meldungen', value: overview.value?.submittedTotal ?? 0,
    hint: `in ${households.value.filter(matchers.submitted).length} Familien` },
]);
const rows = computed(() => households.value.filter(h => {
  if (filter.value && !matchers[filter.value](h)) return false;
  const q = search.value.toLocaleLowerCase('de').trim();
  return !q || [h.householdName, ...h.children.map(c => c.name)].some(n => n.toLocaleLowerCase('de').includes(q));
}));
function status(h: ParentWorkAccount) { return h.overrideMinutes !== undefined ? 'manuelles Soll' : h.exemptReason ? 'befreit' : h.openMinutes > 0 ? 'offen' : 'erfüllt'; }
const { sortKey, sortDir, sorted, toggle } = useTableSort(rows, {
  family: h => h.householdName,
  children: h => h.children.map(c => c.name).join(', '),
  required: h => h.requiredMinutes,
  carryIn: h => h.carryInMinutes,
  done: h => h.doneMinutes,
  open: h => h.openMinutes,
  status: h => status(h),
}, { key: 'family' });
function yearLabel(y: number) { return `${y}/${String((y + 1) % 100).padStart(2, '0')}`; }
function selectCard(key: Filter | null) { filter.value = filter.value === key ? null : key; }
function addFor(h: ParentWorkAccount) { entryHouseholdId.value = h.householdId; showEntry.value = true; }
async function load() {
  loading.value = true; error.value = '';
  try { overview.value = await api.getParentWorkOverview(year.value ?? undefined); year.value = overview.value.kitaYear; }
  catch (e) { error.value = e instanceof Error ? e.message : 'Übersicht konnte nicht geladen werden'; }
  finally { loading.value = false; }
}
onMounted(async () => {
  await load();
  try { rules.value = (await api.getParentWorkRules()) ?? []; }
  catch (e) { error.value = e instanceof Error ? e.message : 'Kita-Jahre konnten nicht geladen werden'; }
});
watch(year, (next, old) => { if (old !== null && next !== old) load(); });
async function saved() { showEntry.value = false; await load(); }
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-wrap items-end justify-between gap-4">
      <h1 class="text-2xl font-bold text-foreground">Elternstunden</h1>
      <button class="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-primary-foreground" @click="entryHouseholdId = undefined; showEntry = true"><Plus class="h-4 w-4" />Stunden erfassen</button>
    </div>
    <label class="block w-44 text-sm font-medium">Kita-Jahr
      <select v-model.number="year" class="mt-1 w-full rounded-lg border bg-card px-3 py-2"><option v-for="y in years" :key="y" :value="y">{{ yearLabel(y) }}</option></select>
    </label>
    <p v-if="error" role="alert" class="rounded-lg bg-red-50 dark:bg-red-950/40 p-3 text-red-700 dark:text-red-300">{{ error }}</p>
    <p v-if="overview?.notice" class="rounded-lg border border-amber-200 bg-amber-50 dark:bg-amber-950/40 p-3 text-amber-900 dark:text-amber-300">{{ overview.notice }}</p>
    <div v-if="overview?.unassignedChildren?.length" role="status"
      class="rounded-lg border border-amber-300 bg-amber-50 dark:bg-amber-950/40 p-3 text-amber-900 dark:text-amber-300">
      <p>Diese Kinder zählen nicht zum Soll. Bitte in den Beiträgen einer Familie zuordnen:</p>
      <ul class="mt-1 list-inside list-disc">
        <li v-for="child in overview.unassignedChildren" :key="child.id">{{ child.name }}</li>
      </ul>
    </div>
    <div v-if="overview" class="grid gap-4 sm:grid-cols-2 xl:grid-cols-5">
      <button v-for="card in cards" :key="card.label" type="button" :aria-pressed="filter === card.key"
        class="rounded-xl border bg-card p-5 text-left transition-colors hover:border-primary"
        :class="filter === card.key && card.key !== null ? 'border-primary ring-2 ring-primary/40' : ''"
        @click="selectCard(card.key)">
        <p class="text-sm text-muted-foreground">{{ card.label }}</p>
        <p class="mt-2 text-2xl font-semibold">{{ card.value }}</p>
        <p v-if="card.hint" class="mt-1 text-xs text-muted-foreground">{{ card.hint }}</p>
      </button>
    </div>
    <SearchInput v-model="search" placeholder="Suche nach Familie oder Kind..." label="Suche nach Familie oder Kind" />
    <div class="rounded-xl border bg-card">
      <div class="overflow-x-auto"><table class="w-full text-sm"><thead class="bg-muted text-muted-foreground"><tr>
        <SortTh label="Familie" column="family" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
        <SortTh label="Kinder" column="children" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
        <SortTh label="Soll" column="required" align="right" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
        <SortTh label="Übertrag" column="carryIn" align="right" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
        <SortTh label="Ist" column="done" align="right" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
        <SortTh label="Offen" column="open" align="right" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
        <SortTh label="Status" column="status" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
        <th class="px-4 py-3 text-right"><span class="sr-only">Aktionen</span></th>
      </tr></thead>
        <tbody><tr v-for="h in sorted" :key="h.householdId" class="border-t hover:bg-accent"><td class="px-4 py-3 font-medium"><RouterLink :to="{ name: 'parent-work-detail', params: { id: h.householdId }, query: { jahr: year } }" class="text-primary hover:underline">{{ h.householdName }}</RouterLink><span v-if="h.submittedCount"
          class="ml-2 rounded-full bg-amber-100 px-2 py-1 text-xs text-amber-800
          dark:bg-amber-950/40 dark:text-amber-300">{{ h.submittedCount }} Meldungen</span></td><td class="px-4 py-3">{{ h.children.map(c => c.name).join(', ') || '—' }}</td><td class="px-4 py-3 text-right">{{ formatHours(h.requiredMinutes) }}</td><td class="px-4 py-3 text-right">{{ formatHours(h.carryInMinutes) }}</td><td class="px-4 py-3 text-right">{{ formatHours(h.doneMinutes) }}</td><td class="px-4 py-3 text-right">{{ formatHours(h.openMinutes) }}</td><td class="px-4 py-3"><span class="rounded-full px-2 py-1 text-xs" :class="h.openMinutes > 0 ? 'bg-amber-100 dark:bg-amber-950/40 text-amber-800 dark:text-amber-300' : 'bg-green-100 dark:bg-green-950/40 text-green-800 dark:text-green-300'">{{ status(h) }}</span></td>
          <td class="px-4 py-3 text-right"><button type="button" class="inline-flex h-8 w-8 items-center justify-center rounded-lg border hover:bg-accent"
            :aria-label="`Stunden erfassen für ${h.householdName}`" :title="`Stunden erfassen für ${h.householdName}`" @click="addFor(h)"><Plus class="h-4 w-4" /></button></td></tr>
          <tr v-if="!sorted.length"><td colspan="8" class="px-4 py-8 text-center text-muted-foreground">{{ loading ? 'Lade Familien …' : 'Keine Familien gefunden.' }}</td></tr></tbody></table></div>
    </div>
    <EntryDialog v-if="showEntry" :household-id="entryHouseholdId" @close="showEntry = false" @saved="saved" />
  </div>
</template>
