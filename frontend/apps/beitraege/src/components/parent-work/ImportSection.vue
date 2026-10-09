<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { RouterLink } from 'vue-router';
import { AlertTriangle, FileSpreadsheet, Upload } from 'lucide-vue-next';
import { api } from '@/api/client';
import type { ParentWorkHouseholdOption, ParentWorkImportPreviewRow, ParentWorkImportExecuteRow } from '@/api/types';
import { formatDate, formatHours } from '@/utils/format';
import HouseholdPicker from '@/components/parent-work/HouseholdPicker.vue';

type FieldKey = 'childName' | 'memberName' | 'workDate' | 'occasion' | 'hours';
const fields: { key: FieldKey; label: string; required: boolean; example: string; hints: string[] }[] = [
  { key: 'workDate', label: 'Datum', required: true, example: '24.05.2026', hints: ['datum', 'wann'] },
  { key: 'hours', label: 'Stunden', required: true, example: '1,5 (Viertelstunden)', hints: ['stunden', 'std', 'dauer'] },
  { key: 'occasion', label: 'Anlass', required: true, example: 'Sommerfest aufgebaut', hints: ['anlass', 'tatigkeit', 'tätigkeit'] },
  { key: 'childName', label: 'Kind', required: false, example: 'Anna Muster', hints: ['kind'] },
  { key: 'memberName', label: 'Mitglied', required: false, example: 'Erika Muster', hints: ['mitglied'] },
];

const headers = ref<string[]>([]);
const csvRows = ref<string[][]>([]);
const assignment = ref<(FieldKey | '')[]>([]);
const rows = ref<ParentWorkImportPreviewRow[]>([]);
const households = ref<ParentWorkHouseholdOption[]>([]);
const selected = ref<Record<number, boolean>>({});
const busy = ref(false);
const error = ref('');
const success = ref('');
const fileName = ref('');
const dragging = ref(false);
let dragDepth = 0;

const mapping = computed(() => {
  const result: Record<string, number> = {};
  assignment.value.forEach((key, i) => { if (key) result[key] = i; });
  return result;
});
const missing = computed(() => {
  const names = fields.filter(f => f.required && mapping.value[f.key] === undefined).map(f => f.label);
  if (mapping.value.childName === undefined && mapping.value.memberName === undefined) names.push('Kind oder Mitglied');
  return names;
});
const canPreview = computed(() => missing.value.length === 0);
const ignoredCount = computed(() => assignment.value.filter(key => !key).length);

function sample(column: number) {
  return csvRows.value.map(r => r[column]?.trim()).filter(Boolean).slice(0, 3).join(' · ');
}
function suggested(header: string): FieldKey | '' {
  const h = header.trim().toLocaleLowerCase('de');
  return fields.find(f => f.hints.some(word => h.includes(word)))?.key ?? '';
}
function assign(column: number, key: FieldKey | '') {
  const next = [...assignment.value];
  if (key) next.forEach((value, i) => { if (value === key) next[i] = ''; });
  next[column] = key;
  assignment.value = next;
  rows.value = [];
}

async function loadFile(file: File) {
  error.value = ''; success.value = ''; rows.value = [];
  if (!/\.(csv|txt)$/i.test(file.name) && !file.type.includes('csv')) {
    error.value = 'Bitte eine CSV-Datei auswählen (Excel: Speichern unter → CSV UTF-8).'; return;
  }
  busy.value = true;
  try {
    const parsed = await api.parseParentWorkImport(file);
    headers.value = parsed.headers; csvRows.value = parsed.rows; fileName.value = file.name;
    const taken = new Set<FieldKey>();
    assignment.value = headers.value.map(h => {
      const key = suggested(h);
      if (!key || taken.has(key)) return '';
      taken.add(key); return key;
    });
  } catch (e) { error.value = e instanceof Error ? e.message : 'CSV konnte nicht gelesen werden'; }
  finally { busy.value = false; }
}
function onPick(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = '';
  if (file) loadFile(file);
}

const hasFiles = (e: DragEvent) => Array.from(e.dataTransfer?.types ?? []).includes('Files');
function onDragEnter(e: DragEvent) { if (!hasFiles(e)) return; e.preventDefault(); dragDepth++; dragging.value = true; }
function onDragOver(e: DragEvent) { if (hasFiles(e)) e.preventDefault(); }
function onDragLeave(e: DragEvent) { if (!hasFiles(e)) return; dragDepth = Math.max(0, dragDepth - 1); if (!dragDepth) dragging.value = false; }
function onDrop(e: DragEvent) {
  if (!hasFiles(e)) return;
  e.preventDefault(); dragDepth = 0; dragging.value = false;
  const file = e.dataTransfer?.files?.[0];
  if (file) loadFile(file);
}
onMounted(() => {
  window.addEventListener('dragenter', onDragEnter); window.addEventListener('dragover', onDragOver);
  window.addEventListener('dragleave', onDragLeave); window.addEventListener('drop', onDrop);
});
onBeforeUnmount(() => {
  window.removeEventListener('dragenter', onDragEnter); window.removeEventListener('dragover', onDragOver);
  window.removeEventListener('dragleave', onDragLeave); window.removeEventListener('drop', onDrop);
});

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
const unmatched = computed(() => {
  const groups = new Map<string, { name: string; lines: number[]; ambiguous: boolean }>();
  for (const row of rows.value.filter(r => !r.householdId)) {
    const name = row.childName || row.memberName || '(ohne Namen)';
    const group = groups.get(name) ?? { name, lines: [], ambiguous: false };
    group.lines.push(row.index);
    group.ambiguous ||= row.errors.some(m => m.includes('mehrdeutig'));
    groups.set(name, group);
  }
  return [...groups.values()];
});
const unmatchedRowCount = computed(() => unmatched.value.reduce((sum, g) => sum + g.lines.length, 0));
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
      <p class="mt-1 text-sm text-muted-foreground">Excel-Tabelle als CSV speichern (Datei → Speichern unter → CSV UTF-8) und hier hochladen.</p>
    </div>

    <div v-if="dragging" class="pointer-events-none fixed inset-0 z-50 flex items-center justify-center bg-primary/20 p-8" aria-hidden="true">
      <div class="rounded-2xl border-2 border-dashed border-primary bg-card px-10 py-8 text-center shadow-xl">
        <Upload class="mx-auto h-10 w-10 text-primary" />
        <p class="mt-3 text-lg font-semibold">CSV-Datei hier ablegen</p>
      </div>
    </div>

    <div class="rounded-lg border bg-card p-4">
      <h3 class="font-semibold">Erwartete Spalten</h3>
      <p class="mt-1 text-sm text-muted-foreground">Die erste Zeile enthält die Spaltennamen. Reihenfolge und Namen sind frei, du ordnest die Spalten nach dem Hochladen zu; weitere Spalten werden ignoriert.</p>
      <ul class="mt-3 grid gap-2 text-sm sm:grid-cols-2 lg:grid-cols-3">
        <li v-for="field in fields" :key="field.key" class="rounded-md bg-muted/50 px-3 py-2">
          <span class="font-medium">{{ field.label }}</span>
          <span v-if="field.required" class="ml-1 text-xs text-red-700 dark:text-red-300">Pflicht</span>
          <span v-else class="ml-1 text-xs text-muted-foreground">Kind oder Mitglied</span>
          <span class="block text-muted-foreground">z. B. {{ field.example }}</span>
        </li>
      </ul>
      <p class="mt-3 text-sm text-muted-foreground">
        <strong class="font-medium text-foreground">Zuordnung zur Familie:</strong> über den Namen, zuerst das Kind, sonst das Mitglied bzw. der Elternteil.
        Groß-/Kleinschreibung, Umlaute (ä = ae) und die Reihenfolge „Nachname Vorname“ spielen keine Rolle; sonst muss der volle Name stimmen.
        Zeilen ohne eindeutigen Treffer markiert die Vorschau, dort wählst du die Familie von Hand.
      </p>
    </div>

    <div class="rounded-lg border-2 border-dashed bg-card p-6 text-center" :class="dragging ? 'border-primary' : 'border-border'">
      <FileSpreadsheet class="mx-auto h-9 w-9 text-muted-foreground" aria-hidden="true" />
      <p v-if="fileName" class="mt-2 text-sm">
        <span class="font-medium">{{ fileName }}</span>
        <span v-if="headers.length" class="text-muted-foreground"> · {{ csvRows.length }} {{ csvRows.length === 1 ? 'Zeile' : 'Zeilen' }} erkannt.</span>
      </p>
      <p v-else class="mt-2 text-sm text-muted-foreground">CSV-Datei hierher ziehen (überall im Fenster) oder auswählen.</p>
      <input id="parent-work-csv" class="sr-only peer" type="file" accept=".csv,.txt,text/csv" :disabled="busy" @change="onPick" />
      <label for="parent-work-csv" class="mt-3 inline-flex cursor-pointer items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 peer-focus-visible:ring-2 peer-focus-visible:ring-ring peer-focus-visible:ring-offset-2 peer-disabled:opacity-50">
        <Upload class="h-4 w-4" aria-hidden="true" />{{ fileName ? 'Andere CSV-Datei wählen' : 'CSV-Datei auswählen' }}
      </label>
    </div>

    <div v-if="headers.length" class="rounded-lg border bg-card p-4">
      <h3 class="font-semibold">Spalten zuordnen</h3>
      <p class="mt-1 text-sm text-muted-foreground">Ordne jeder Spalte deiner Datei ein Feld zu. Nicht benötigte Spalten bleiben auf „Ignorieren“<span v-if="ignoredCount"> ({{ ignoredCount }} von {{ headers.length }} ignoriert)</span>.</p>
      <div class="mt-3 overflow-x-auto">
        <table aria-label="Spaltenzuordnung" class="min-w-full text-left text-sm">
          <thead class="bg-muted"><tr><th class="p-2">Spalte in der Datei</th><th class="p-2">Beispielwerte</th><th class="p-2">Zuordnung</th></tr></thead>
          <tbody>
            <tr v-for="(header, i) in headers" :key="i" class="border-t align-middle" :class="{ 'text-muted-foreground': !assignment[i] }">
              <td class="p-2 font-medium">{{ header || `Spalte ${i + 1}` }}</td>
              <td class="max-w-72 truncate p-2">{{ sample(i) || '—' }}</td>
              <td class="p-2">
                <select :value="assignment[i]" :aria-label="`Zuordnung für Spalte ${header || i + 1}`" class="w-full rounded border bg-card px-2 py-1.5 text-foreground" @change="assign(i, ($event.target as HTMLSelectElement).value as FieldKey | '')">
                  <option value="">Ignorieren</option>
                  <option v-for="field in fields" :key="field.key" :value="field.key">{{ field.label }}{{ field.required ? ' *' : '' }}</option>
                </select>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-if="missing.length" class="mt-3 text-sm text-amber-700 dark:text-amber-300">Es fehlt noch: {{ missing.join(', ') }}.</p>
      <button class="mt-4 rounded bg-primary px-4 py-2 text-primary-foreground disabled:opacity-50" :disabled="!canPreview || busy" @click="preview">Vorschau erstellen</button>
    </div>

    <p v-if="error" class="rounded border border-red-300 bg-red-50 dark:bg-red-950/40 p-3 text-sm text-red-800 dark:text-red-300" role="alert">{{ error }}</p>
    <p v-if="success" class="rounded border border-green-300 bg-green-50 dark:bg-green-950/40 p-3 text-sm text-green-800 dark:text-green-300" role="status">{{ success }} <RouterLink class="underline" to="/elternstunden">Zur Übersicht</RouterLink></p>

    <div v-if="rows.length" class="space-y-3">
      <div v-if="unmatched.length" class="rounded-lg border border-amber-300 bg-amber-50 p-4 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200" role="alert">
        <p class="flex items-center gap-2 font-semibold"><AlertTriangle class="h-4 w-4" aria-hidden="true" />{{ unmatchedRowCount }} {{ unmatchedRowCount === 1 ? 'Zeile' : 'Zeilen' }} ohne Familie</p>
        <p class="mt-1">Diese Zeilen werden nicht importiert, solange keine Familie gewählt ist:</p>
        <ul class="mt-2 list-disc space-y-0.5 pl-5">
          <li v-for="group in unmatched" :key="group.name">
            <strong class="font-medium">{{ group.name }}</strong>
            <span> – {{ group.ambiguous ? 'mehrdeutig' : 'keiner Familie zuzuordnen' }} (Zeile {{ group.lines.join(', ') }})</span>
          </li>
        </ul>
      </div>
      <div class="flex flex-wrap items-center justify-between gap-3"><h3 class="text-lg font-semibold">Vorschau ({{ chosenRows.length }} ausgewählt)</h3><button class="rounded bg-primary px-4 py-2 text-primary-foreground disabled:opacity-50" :disabled="busy || !chosenRows.length" @click="execute">{{ chosenRows.length }} {{ chosenRows.length === 1 ? 'Eintrag' : 'Einträge' }} importieren</button></div>
      <div class="overflow-x-auto rounded-lg border bg-card">
        <table aria-label="Vorschau" class="min-w-full text-left text-sm"><thead class="bg-muted"><tr><th class="p-3">Import</th><th class="p-3">Zeile</th><th class="p-3">Datum</th><th class="p-3">Stunden</th><th class="p-3">Anlass</th><th class="p-3">Mitglied / Kind</th><th class="p-3">Familie</th><th class="p-3">Status</th></tr></thead>
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
