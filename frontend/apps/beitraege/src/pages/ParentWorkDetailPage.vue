<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { RouterLink, useRoute } from 'vue-router';
import { api } from '@/api';
import type { ParentWorkDetail, ParentWorkEntry } from '@/api/types';
import { formatCurrency, formatDate, formatHours } from '@/utils/format';
import { useAuthStore } from '@/stores/auth';
import { useTableSort } from '@/composables/useTableSort';
import EntryDialog from '@/components/parent-work/EntryDialog.vue';
import SortTh from '@/components/SortTh.vue';

const route = useRoute();
const auth = useAuthStore();
const detail = ref<ParentWorkDetail | null>(null);
const year = ref<number | null>(null);
const error = ref('');
const actionError = ref('');
const loading = ref(false);
const saving = ref(false);
const entryDialog = ref(false);
const editingEntry = ref<ParentWorkEntry | null>(null);
const voidEntry = ref<ParentWorkEntry | null>(null);
const voidReason = ref('');
const rejectEntry = ref<ParentWorkEntry | null>(null);
const rejectReason = ref('');
const overrideDialog = ref(false);
const overrideHours = ref(0);
const overrideReason = ref('');

async function load() {
  loading.value = true; error.value = '';
  try {
    const raw = Number(route.query.jahr);
    if (Number.isInteger(raw) && raw >= 1900 && raw <= 9998) year.value = raw;
    else year.value = (await api.getParentWorkOverview()).kitaYear;
    detail.value = await api.getParentWorkDetail(String(route.params.id), year.value);
  } catch (e) { error.value = e instanceof Error ? e.message : 'Familie konnte nicht geladen werden'; }
  finally { loading.value = false; }
}
onMounted(load);
watch(() => [route.params.id, route.query.jahr], load);
function startOverride() {
  overrideHours.value = (detail.value?.overrideMinutes ?? detail.value?.requiredMinutes ?? 0) / 60;
  overrideReason.value = detail.value?.overrideReason ?? '';
  actionError.value = ''; overrideDialog.value = true;
}
async function saveOverride() {
  const minutes = Math.round(overrideHours.value * 60);
  if (!Number.isFinite(overrideHours.value) || overrideHours.value < 0 || minutes % 15 !== 0 || Math.abs(minutes / 60 - overrideHours.value) > 0.00001) { actionError.value = 'Bitte Stunden in Schritten von 0,25 eingeben.'; return; }
  if (!year.value) return;
  saving.value = true; actionError.value = '';
  try { await api.setParentWorkOverride(String(route.params.id), { kitaYear: year.value, requiredMinutes: minutes, reason: overrideReason.value.trim() }); overrideDialog.value = false; await load(); }
  catch (e) { actionError.value = e instanceof Error ? e.message : 'Speichern fehlgeschlagen'; }
  finally { saving.value = false; }
}
async function removeOverride() {
  if (!year.value) return;
  saving.value = true; actionError.value = '';
  try { await api.removeParentWorkOverride(String(route.params.id), year.value); overrideDialog.value = false; await load(); }
  catch (e) { actionError.value = e instanceof Error ? e.message : 'Entfernen fehlgeschlagen'; }
  finally { saving.value = false; }
}
async function submitVoid() {
  if (!voidEntry.value) return;
  saving.value = true; actionError.value = '';
  try { await api.voidParentWorkEntry(voidEntry.value.id, voidReason.value.trim()); voidEntry.value = null; await load(); }
  catch (e) { actionError.value = e instanceof Error ? e.message : 'Stornieren fehlgeschlagen'; }
  finally { saving.value = false; }
}
async function review(id: string, approve: boolean) {
  saving.value = true; actionError.value = '';
  try {
    if (approve) await api.approveParentWorkEntry(id);
    else await api.rejectParentWorkEntry(id, rejectReason.value.trim());
    rejectEntry.value = null; rejectReason.value = ''; await load();
  } catch (e) { actionError.value = e instanceof Error ? e.message : 'Prüfung fehlgeschlagen'; }
  finally { saving.value = false; }
}
const entries = computed(() => detail.value?.entries ?? []);
const { sortKey, sortDir, sorted: sortedEntries, toggle } = useTableSort(entries, {
  date: e => e.workDate,
  hours: e => e.durationMinutes,
  occasion: e => e.occasion,
  member: e => e.memberName,
  child: e => e.childName,
  source: e => source(e.source),
  status: e => entryStatus(e.status),
}, { key: 'date', dir: 'desc' });
const account = computed(() => {
  const d = detail.value;
  if (!d) return [];
  const cards = [
    { label: 'Soll', value: formatHours(d.requiredMinutes) },
    { label: 'Geleistet', value: formatHours(d.doneMinutes) },
    { label: 'Übertrag Vorjahr', value: formatHours(d.carryInMinutes) },
    { label: 'Offen', value: formatHours(d.openMinutes) },
    { label: 'Übertrag Folgejahr', value: formatHours(d.carryOutMinutes) },
  ];
  if (auth.isAdmin) cards.push({ label: 'Fehlbetrag', value: formatCurrency(d.missingAmountCents / 100) });
  return cards;
});
function entrySaved() { entryDialog.value = false; editingEntry.value = null; load(); }
function entryStatus(status: string) { return ({ APPROVED: 'Bestätigt', SUBMITTED: 'Eingereicht', REJECTED: 'Abgelehnt', VOIDED: 'Storniert' } as Record<string, string>)[status] ?? status; }
function source(source: string) { return source === 'IMPORT' ? 'Import' : source === 'MANUAL' ? 'Manuell' : source === 'PARENT' ? 'Eltern' : source; }
</script>

<template>
  <div class="space-y-6">
    <RouterLink to="/elternstunden" class="text-sm text-primary hover:underline">← Zur Übersicht</RouterLink>
    <div class="flex flex-wrap items-center justify-between gap-3"><div><h1 class="text-2xl font-bold">{{ detail?.householdName ?? 'Familie' }}</h1><p class="text-sm text-muted-foreground">Elternstunden · Kita-Jahr {{ year ? `${year}/${String((year + 1) % 100).padStart(2, '0')}` : '…' }}</p></div><button class="rounded-lg bg-primary px-4 py-2 text-primary-foreground" @click="editingEntry = null; entryDialog = true">Stunden erfassen</button></div>
    <p v-if="error" role="alert" class="rounded-lg bg-red-50 dark:bg-red-950/40 p-3 text-red-700 dark:text-red-300">{{ error }}</p><p v-if="loading" class="text-muted-foreground">Lade Familie …</p>
    <template v-if="detail">
      <div class="grid gap-4 lg:grid-cols-2">
        <section class="rounded-xl border bg-card p-5"><h2 class="text-lg font-semibold">Kinder</h2><table class="mt-3 w-full text-sm"><thead class="text-left text-muted-foreground"><tr><th class="py-2">Kind</th><th class="py-2">Betreuung</th><th class="py-2 text-right">Gezählte Tertiale</th></tr></thead><tbody><tr v-for="c in detail.children" :key="c.id" class="border-t"><td class="py-2">{{ c.name }}</td><td class="py-2">{{ formatDate(c.entryDate) }} – {{ formatDate(c.exitDate) }}</td><td class="py-2 text-right">{{ c.tertials }}</td></tr><tr v-if="!detail.children.length"><td colspan="3" class="py-3 text-muted-foreground">Keine Kinder im Kita-Jahr.</td></tr></tbody></table></section>
        <section class="rounded-xl border bg-card p-5"><div class="flex items-center justify-between gap-2"><h2 class="text-lg font-semibold">Soll</h2><button class="text-sm text-primary hover:underline" @click="startOverride">{{ detail.overrideMinutes !== undefined ? 'Manuelles Soll bearbeiten' : 'Manuelles Soll setzen' }}</button></div><p class="mt-3 text-sm">Berechnetes Soll: <strong>{{ formatHours(detail.calculatedMinutes) }}</strong></p><p v-if="detail.exemptReason" class="mt-2 text-sm">Befreiung: {{ detail.exemptReason }}</p><p v-if="detail.overrideMinutes !== undefined" class="mt-2 text-sm">Manuelles Soll: <strong>{{ formatHours(detail.overrideMinutes) }}</strong> · {{ detail.overrideReason }}</p><p v-if="actionError && !overrideDialog && !voidEntry" role="alert" class="mt-3 text-red-700 dark:text-red-300">{{ actionError }}</p></section>
      </div>
      <section aria-labelledby="account-heading"><h2 id="account-heading" class="text-lg font-semibold">Konto</h2>
        <div class="mt-3 grid gap-4 sm:grid-cols-2 lg:grid-cols-3" :class="account.length === 5 ? 'xl:grid-cols-5' : 'xl:grid-cols-6'">
          <div v-for="item in account" :key="item.label" class="rounded-xl border bg-card p-5"><p class="text-sm text-muted-foreground">{{ item.label }}</p><p class="mt-2 text-2xl font-semibold">{{ item.value }}</p></div>
        </div></section>
      <section class="overflow-hidden rounded-xl border bg-card"><h2 class="p-5 text-lg font-semibold">Einträge</h2><div class="overflow-x-auto"><table class="w-full text-sm"><thead class="bg-muted text-muted-foreground"><tr>
          <SortTh label="Datum" column="date" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
          <SortTh label="Stunden" column="hours" align="right" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
          <SortTh label="Anlass" column="occasion" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
          <SortTh label="Mitglied" column="member" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
          <SortTh label="Kind" column="child" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
          <SortTh label="Quelle" column="source" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
          <SortTh label="Status" column="status" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
          <th class="px-4 py-3 text-left">Aktionen</th></tr></thead><tbody><tr v-for="e in sortedEntries" :key="e.id" class="border-t" :class="e.status === 'VOIDED' ? 'text-muted-foreground bg-muted' : ''"><td class="px-4 py-3">{{ formatDate(e.workDate) }}</td><td class="px-4 py-3 text-right">{{ formatHours(e.durationMinutes) }}</td><td class="px-4 py-3">{{ e.occasion }}</td><td class="px-4 py-3">{{ e.memberName || '—' }}</td><td class="px-4 py-3">{{ e.childName || '—' }}</td><td class="px-4 py-3">{{ source(e.source) }}</td><td class="px-4 py-3">{{ entryStatus(e.status) }}<span v-if="e.reviewedAt" class="block text-xs text-muted-foreground">von {{ e.reviewedByName || 'unbekannt' }}, {{ formatDate(e.reviewedAt) }}</span><span v-if="e.rejectReason" class="block text-xs">Grund: {{ e.rejectReason }}</span><span v-if="e.voidReason" class="block text-xs">Grund: {{ e.voidReason }}</span></td><td class="px-4 py-3"><div v-if="e.status === 'SUBMITTED'" class="flex gap-2">
          <button class="text-primary underline" @click="review(e.id, true)">Bestätigen</button>
          <button class="text-red-700 underline dark:text-red-300" @click="rejectEntry = e">Ablehnen</button>
        </div><div v-else-if="e.status !== 'VOIDED'" class="flex gap-2"><button class="text-primary hover:underline" @click="editingEntry = e; entryDialog = true">Bearbeiten</button><button class="text-red-700 dark:text-red-300 hover:underline" @click="voidEntry = e; voidReason = ''; actionError = ''">Stornieren</button></div></td></tr><tr v-if="!detail.entries?.length"><td colspan="8" class="px-4 py-6 text-center text-muted-foreground">Keine Einträge.</td></tr></tbody></table></div></section>
      <section v-if="detail.boardTerms?.length" class="rounded-xl border bg-card p-5"><h2 class="text-lg font-semibold">Vorstands-Amtszeiten</h2><table class="mt-3 w-full text-sm"><thead class="text-left text-muted-foreground"><tr><th class="py-2">Mitglied</th><th class="py-2">Amt</th><th class="py-2">Von</th><th class="py-2">Bis</th></tr></thead><tbody><tr v-for="term in detail.boardTerms" :key="term.id" class="border-t"><td class="py-2">{{ term.memberName }}</td><td class="py-2">{{ term.office }}</td><td class="py-2">{{ formatDate(term.startDate) }}</td><td class="py-2">{{ formatDate(term.endDate) }}</td></tr></tbody></table></section>
    </template>
    <div v-if="rejectEntry" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <form role="dialog" aria-modal="true" aria-label="Meldung ablehnen"
        class="w-full max-w-md rounded-xl bg-card p-5" @submit.prevent="review(rejectEntry.id, false)">
        <h2 class="text-lg font-bold">Meldung ablehnen</h2>
        <label class="mt-3 block text-sm">Grund *<textarea v-model="rejectReason" required
          class="mt-1 w-full rounded-lg border p-2" /></label>
        <p v-if="actionError" role="alert" class="text-red-700 dark:text-red-300">{{ actionError }}</p>
        <div class="mt-4 flex justify-end gap-2"><button type="button" class="rounded-lg border px-4 py-2"
          @click="rejectEntry = null">Abbrechen</button>
          <button type="submit" :disabled="saving" class="rounded-lg bg-primary px-4 py-2
            text-primary-foreground">Ablehnen</button></div>
      </form>
    </div>
    <EntryDialog v-if="entryDialog" :entry="editingEntry" :household-id="String(route.params.id)" @close="entryDialog = false" @saved="entrySaved" />
    <div v-if="overrideDialog" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="overrideDialog = false"><form role="dialog" aria-modal="true" aria-label="Manuelles Soll" class="w-full max-w-md rounded-xl bg-card p-6 shadow-xl" @submit.prevent="saveOverride"><h2 class="text-xl font-semibold">Manuelles Soll</h2><div class="mt-4 space-y-4"><label class="block text-sm font-medium">Stunden *<input v-model.number="overrideHours" type="number" min="0" step="0.25" required class="mt-1 w-full rounded-lg border px-3 py-2" /></label><label class="block text-sm font-medium">Begründung *<textarea v-model="overrideReason" required class="mt-1 w-full rounded-lg border px-3 py-2" /></label><p v-if="actionError" role="alert" class="text-sm text-red-700 dark:text-red-300">{{ actionError }}</p></div><div class="mt-6 flex flex-wrap justify-end gap-2"><button v-if="detail?.overrideMinutes !== undefined" type="button" :disabled="saving" class="mr-auto text-red-700 dark:text-red-300" @click="removeOverride">Manuelles Soll entfernen</button><button type="button" class="rounded-lg border px-4 py-2" @click="overrideDialog = false">Abbrechen</button><button type="submit" :disabled="saving" class="rounded-lg bg-primary px-4 py-2 text-primary-foreground">Speichern</button></div></form></div>
    <div v-if="voidEntry" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="voidEntry = null"><form role="dialog" aria-modal="true" aria-label="Eintrag stornieren" class="w-full max-w-md rounded-xl bg-card p-6 shadow-xl" @submit.prevent="submitVoid"><h2 class="text-xl font-semibold">Eintrag stornieren</h2><label class="mt-4 block text-sm font-medium">Grund *<textarea v-model="voidReason" required class="mt-1 w-full rounded-lg border px-3 py-2" /></label><p v-if="actionError" role="alert" class="mt-3 text-sm text-red-700 dark:text-red-300">{{ actionError }}</p><div class="mt-6 flex justify-end gap-2"><button type="button" class="rounded-lg border px-4 py-2" @click="voidEntry = null">Abbrechen</button><button type="submit" :disabled="saving" class="rounded-lg bg-red-700 px-4 py-2 text-white">Stornieren</button></div></form></div>
  </div>
</template>
