<script setup lang="ts">
import { computed, ref } from 'vue';
import { RouterLink } from 'vue-router';
import api from '@/api/client';
import type { ParentWorkHouseholdOption, ParentWorkImportPreviewRow, ParentWorkImportExecuteRow } from '@/api/types';
import { formatDate, formatHours } from '@/utils/format';
import HouseholdPicker from '@/components/parent-work/HouseholdPicker.vue';

const headers = ref<string[]>([]);
const csvRows = ref<string[][]>([]);
const rows = ref<ParentWorkImportPreviewRow[]>([]);
const mapping = ref<Record<string, number>>({});
const households = ref<ParentWorkHouseholdOption[]>([]);
const selected = ref<Record<number, boolean>>({});
const busy = ref(false);
const error = ref('');
const success = ref('');
const file = ref<File | null>(null);
const fields = [
  { key: 'childName', label: 'Kind', hints: ['kind'] },
  { key: 'memberName', label: 'Mitglied', hints: ['mitglied'] },
  { key: 'workDate', label: 'Datum', hints: ['datum', 'wann'] },
  { key: 'occasion', label: 'Anlass', hints: ['anlass', 'tatigkeit', 'tätigkeit'] },
  { key: 'hours', label: 'Stunden', hints: ['stunden', 'std', 'dauer'] },
];
const canPreview = computed(() => ['workDate', 'hours', 'occasion'].every(k => mapping.value[k] !== undefined) &&
  (mapping.value.childName !== undefined || mapping.value.memberName !== undefined));
function suggested(header: string) {
  const h = header.trim().toLocaleLowerCase('de');
  return fields.find(f => f.hints.some(word => h.includes(word)))?.key;
}
async function chooseFile() {
  error.value = ''; success.value = ''; rows.value = [];
  if (!file.value) return;
  busy.value = true;
  try {
    const parsed = await api.parseParentWorkImport(file.value);
    headers.value = parsed.headers; csvRows.value = parsed.rows;
    mapping.value = {};
    headers.value.forEach((h, i) => { const key = suggested(h); if (key && mapping.value[key] === undefined) mapping.value[key] = i; });
  } catch (e) { error.value = e instanceof Error ? e.message : 'CSV konnte nicht gelesen werden'; }
  finally { busy.value = false; }
}
async function preview() {
  busy.value = true; error.value = ''; success.value = '';
  try {
    const [result, familyList] = await Promise.all([
      api.previewParentWorkImport(headers.value, csvRows.value, mapping.value), api.getParentWorkHouseholds(),
    ]);
    rows.value = result; households.value = familyList ?? [];
    selected.value = Object.fromEntries(result.map(r => [r.index, !r.duplicate && rowSelectable(r)]));
  } catch (e) { error.value = e instanceof Error ? e.message : 'Vorschau fehlgeschlagen'; }
  finally { busy.value = false; }
}
function remainingErrors(row: ParentWorkImportPreviewRow) {
  return row.errors.filter(message => !(row.householdId &&
    (message.includes('Familie auswählen') || message.includes('Familie nicht gefunden'))));
}
function rowSelectable(row: ParentWorkImportPreviewRow) {
  return !!row.householdId && !!row.workDate && !!row.durationMinutes &&
    !!row.occasion?.trim() && remainingErrors(row).length === 0;
}
function chooseHousehold(row: ParentWorkImportPreviewRow, id: string) {
  row.householdId = id || undefined;
  row.householdName = households.value.find(h => h.id === id)?.name;
  selected.value[row.index] = rowSelectable(row) && !row.duplicate;
}
const chosenRows = computed(() => rows.value.filter(r => selected.value[r.index] && rowSelectable(r)));
async function execute() {
  busy.value = true; error.value = ''; success.value = '';
  try {
    const payload: ParentWorkImportExecuteRow[] = chosenRows.value.map(r => ({
      householdId: r.householdId!, workDate: r.workDate!, durationMinutes: r.durationMinutes!, occasion: r.occasion!,
      ...(r.memberName ? { memberName: r.memberName } : {}), ...(r.childName ? { childName: r.childName } : {}),
    }));
    const result = await api.executeParentWorkImport(payload);
    success.value = `${result.created} ${result.created === 1 ? 'Eintrag wurde' : 'Einträge wurden'} importiert.`; rows.value = [];
  } catch (e) { error.value = e instanceof Error ? e.message : 'Import fehlgeschlagen'; }
  finally { busy.value = false; }
}
</script>

<template>
  <section class="space-y-6">
    <div>
      <h2 class="text-xl font-semibold">Import</h2>
      <p class="mt-1 text-sm text-muted-foreground">Excel-Tabelle als CSV speichern (Datei → Speichern unter → CSV UTF-8).</p>
    </div>
    <div class="rounded-lg border bg-card p-4">
      <label class="block text-sm font-medium" for="parent-work-csv">CSV-Datei</label>
      <input id="parent-work-csv" class="mt-2 block w-full text-sm" type="file" accept=".csv,text/csv" @change="file = ($event.target as HTMLInputElement).files?.[0] ?? null; chooseFile()" />
      <p v-if="headers.length" class="mt-2 text-sm text-muted-foreground">{{ csvRows.length }} {{ csvRows.length === 1 ? 'Zeile' : 'Zeilen' }} erkannt.</p>
    </div>
    <div v-if="headers.length" class="rounded-lg border bg-card p-4">
      <h2 class="font-semibold">Spalten zuordnen</h2>
      <div class="mt-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
        <label v-for="field in fields" :key="field.key" class="text-sm">{{ field.label }}<span v-if="['workDate','hours','occasion'].includes(field.key)" class="text-red-600 dark:text-red-300"> *</span>
          <select v-model.number="mapping[field.key]" class="mt-1 w-full rounded border px-2 py-2">
            <option :value="undefined">Nicht zugeordnet</option><option v-for="(header, i) in headers" :key="i" :value="i">{{ header || `Spalte ${i + 1}` }}</option>
          </select>
        </label>
      </div>
      <button class="mt-4 rounded bg-primary px-4 py-2 text-primary-foreground disabled:opacity-50" :disabled="!canPreview || busy" @click="preview">Vorschau erstellen</button>
    </div>
    <p v-if="error" class="rounded border border-red-300 bg-red-50 dark:bg-red-950/40 p-3 text-sm text-red-800 dark:text-red-300" role="alert">{{ error }}</p>
    <p v-if="success" class="rounded border border-green-300 bg-green-50 dark:bg-green-950/40 p-3 text-sm text-green-800 dark:text-green-300" role="status">{{ success }} <RouterLink class="underline" to="/elternstunden">Zur Übersicht</RouterLink></p>
    <div v-if="rows.length" class="space-y-3">
      <div class="flex flex-wrap items-center justify-between gap-3"><h2 class="text-lg font-semibold">Vorschau ({{ chosenRows.length }} ausgewählt)</h2><button class="rounded bg-primary px-4 py-2 text-primary-foreground disabled:opacity-50" :disabled="busy || !chosenRows.length" @click="execute">{{ chosenRows.length }} {{ chosenRows.length === 1 ? 'Eintrag' : 'Einträge' }} importieren</button></div>
      <div class="overflow-x-auto rounded-lg border bg-card">
        <table class="min-w-full text-left text-sm"><thead class="bg-muted"><tr><th class="p-3">Import</th><th class="p-3">Zeile</th><th class="p-3">Datum</th><th class="p-3">Stunden</th><th class="p-3">Anlass</th><th class="p-3">Mitglied / Kind</th><th class="p-3">Familie</th><th class="p-3">Status</th></tr></thead>
          <tbody><tr v-for="row in rows" :key="row.index" class="border-t align-top">
            <td class="p-3"><input v-model="selected[row.index]" type="checkbox" :disabled="!rowSelectable(row)" :aria-label="`Zeile ${row.index} importieren`" /></td>
            <td class="p-3">{{ row.index }}</td><td class="p-3">{{ formatDate(row.workDate) }}</td><td class="p-3">{{ row.durationMinutes ? formatHours(row.durationMinutes) : '—' }}</td><td class="p-3">{{ row.occasion || '—' }}</td><td class="p-3">{{ [row.memberName, row.childName].filter(Boolean).join(' / ') || '—' }}</td>
            <td class="min-w-56 p-3"><HouseholdPicker :model-value="row.householdId" :households="households"
              :label="`Familie für Zeile ${row.index}`" @update:model-value="chooseHousehold(row, $event)" />
              <p v-if="row.matchedBy && row.householdId" class="mt-1 text-xs text-muted-foreground">
                Treffer über {{ row.matchedBy === 'child' ? 'Kind' : 'Mitglied' }}
              </p>
            </td>
            <td class="max-w-64 p-3"><span v-if="!rowSelectable(row)" class="text-red-700 dark:text-red-300">{{ remainingErrors(row).join('; ') || 'Familie auswählen' }}</span><span v-else-if="row.duplicate" class="text-amber-700 dark:text-amber-300">Dublette</span><span v-else class="text-green-700 dark:text-green-300">OK</span></td>
          </tr></tbody>
        </table>
      </div>
    </div>
  </section>
</template>
