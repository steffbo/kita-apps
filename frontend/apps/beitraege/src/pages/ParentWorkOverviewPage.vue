<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { RouterLink } from 'vue-router';
import { Plus } from 'lucide-vue-next';
import { api } from '@/api';
import type { ParentWorkAccount, ParentWorkOverview, ParentWorkRule } from '@/api/types';
import { formatCurrency, formatHours } from '@/utils/format';
import EntryDialog from '@/components/parent-work/EntryDialog.vue';

const overview = ref<ParentWorkOverview | null>(null);
const rules = ref<ParentWorkRule[]>([]);
const year = ref<number | null>(null);
const search = ref('');
const onlyOpen = ref(false);
const showEntry = ref(false);
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
const rows = computed(() => (overview.value?.households ?? []).filter(h => {
  if (onlyOpen.value && h.openMinutes <= 0) return false;
  const q = search.value.toLocaleLowerCase('de').trim();
  return !q || [h.householdName, ...h.children.map(c => c.name)].some(n => n.toLocaleLowerCase('de').includes(q));
}));
function yearLabel(y: number) { return `${y}/${String((y + 1) % 100).padStart(2, '0')}`; }
function status(h: ParentWorkAccount) { return h.overrideMinutes !== undefined ? 'manuelles Soll' : h.exemptReason ? 'befreit' : h.openMinutes > 0 ? 'offen' : 'erfüllt'; }
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
      <div><h1 class="text-2xl font-bold text-gray-900">Elternstunden</h1><p class="text-sm text-gray-500">Übersicht je Familie und Kita-Jahr</p></div>
      <button class="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-white" @click="showEntry = true"><Plus class="h-4 w-4" />Stunden erfassen</button>
    </div>
    <label class="block w-44 text-sm font-medium">Kita-Jahr
      <select v-model.number="year" class="mt-1 w-full rounded-lg border bg-white px-3 py-2"><option v-for="y in years" :key="y" :value="y">{{ yearLabel(y) }}</option></select>
    </label>
    <p v-if="error" role="alert" class="rounded-lg bg-red-50 p-3 text-red-700">{{ error }}</p>
    <p v-if="overview?.notice" class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-amber-900">{{ overview.notice }}</p>
    <div v-if="overview" class="grid gap-4 sm:grid-cols-2 xl:grid-cols-5">
      <div v-for="card in [
        { label: 'Familien', value: String(overview.households?.length ?? 0) },
        { label: 'Soll gesamt', value: formatHours(overview.requiredMinutes) },
        { label: 'Ist gesamt', value: formatHours(overview.doneMinutes) },
        { label: 'Offen gesamt', value: formatHours(overview.openMinutes) },
        { label: 'Fehlbetrag gesamt', value: formatCurrency(overview.missingAmountCents / 100) },
      ]" :key="card.label" class="rounded-xl border bg-white p-5"><p class="text-sm text-gray-500">{{ card.label }}</p><p class="mt-2 text-2xl font-semibold">{{ card.value }}</p></div>
    </div>
    <div class="rounded-xl border bg-white">
      <div class="flex flex-wrap items-center gap-4 border-b p-4">
        <label class="block flex-1 text-sm font-medium">Suche nach Familie oder Kind<input v-model="search" type="search" class="mt-1 w-full max-w-md rounded-lg border px-3 py-2" /></label>
        <label class="flex items-center gap-2 text-sm"><input v-model="onlyOpen" type="checkbox" />Nur offene</label>
      </div>
      <div class="overflow-x-auto"><table class="w-full text-sm"><thead class="bg-gray-50 text-left text-gray-600"><tr><th class="px-4 py-3">Familie</th><th class="px-4 py-3">Kinder</th><th class="px-4 py-3 text-right">Soll</th><th class="px-4 py-3 text-right">Übertrag</th><th class="px-4 py-3 text-right">Ist</th><th class="px-4 py-3 text-right">Offen</th><th class="px-4 py-3 text-right">Fehlbetrag</th><th class="px-4 py-3">Status</th></tr></thead>
        <tbody><tr v-for="h in rows" :key="h.householdId" class="border-t hover:bg-gray-50"><td class="px-4 py-3 font-medium"><RouterLink :to="{ name: 'parent-work-detail', params: { id: h.householdId }, query: { jahr: year } }" class="text-primary hover:underline">{{ h.householdName }}</RouterLink></td><td class="px-4 py-3">{{ h.children.map(c => c.name).join(', ') || '—' }}</td><td class="px-4 py-3 text-right">{{ formatHours(h.requiredMinutes) }}</td><td class="px-4 py-3 text-right">{{ formatHours(h.carryInMinutes) }}</td><td class="px-4 py-3 text-right">{{ formatHours(h.doneMinutes) }}</td><td class="px-4 py-3 text-right">{{ formatHours(h.openMinutes) }}</td><td class="px-4 py-3 text-right">{{ formatCurrency(h.missingAmountCents / 100) }}</td><td class="px-4 py-3"><span class="rounded-full px-2 py-1 text-xs" :class="h.openMinutes > 0 ? 'bg-amber-100 text-amber-800' : 'bg-green-100 text-green-800'">{{ status(h) }}</span></td></tr>
          <tr v-if="!rows.length"><td colspan="8" class="px-4 py-8 text-center text-gray-500">{{ loading ? 'Lade Familien …' : 'Keine Familien gefunden.' }}</td></tr></tbody></table></div>
    </div>
    <EntryDialog v-if="showEntry" @close="showEntry = false" @saved="saved" />
  </div>
</template>
