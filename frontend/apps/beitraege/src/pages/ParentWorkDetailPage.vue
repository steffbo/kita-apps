<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { api } from '@/api';
import type { ParentWorkDetail, ParentWorkEntry } from '@/api/types';
import { formatCurrency, formatDate, formatHours } from '@/utils/format';
import { useAuthStore } from '@/stores/auth';
import { useTableSort } from '@/composables/useTableSort';
import EntryDialog from '@/components/parent-work/EntryDialog.vue';
import SortTh from '@/components/SortTh.vue';
import PageHeader from '@/components/ui/PageHeader.vue';
import AppSheet from '@/components/ui/AppSheet.vue';
import HouseholdDonationsCard from '@/components/HouseholdDonationsCard.vue';

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
    <PageHeader :title="detail?.householdName ?? 'Familie'" back="/elternstunden">
      <template #meta><span v-if="year" class="badge">Kita-Jahr {{ year }}/{{ String((year + 1) % 100).padStart(2, '0') }}</span></template>
      <template #actions><button class="btn btn-primary" @click="editingEntry = null; entryDialog = true">Stunden erfassen</button></template>
    </PageHeader>
    <p v-if="error" role="alert" class="note note-danger">{{ error }}</p><p v-if="loading" class="text-muted-foreground">Lade Familie …</p>
    <template v-if="detail">
      <div class="grid gap-4 lg:grid-cols-2">
        <section class="card overflow-hidden"><h2 class="px-4 pb-1 pt-4 text-base font-semibold">Kinder</h2><div class="overflow-x-auto"><table class="table"><thead><tr><th>Kind</th><th>Betreuung</th><th class="text-right">Gezählte Tertiale</th></tr></thead><tbody><tr v-for="c in detail.children" :key="c.id"><td>{{ c.name }}</td><td>{{ formatDate(c.entryDate) }} – {{ formatDate(c.exitDate) }}</td><td class="text-right">{{ c.tertials }}</td></tr><tr v-if="!detail.children.length"><td colspan="3" class="text-muted-foreground">Keine Kinder im Kita-Jahr.</td></tr></tbody></table></div></section>
        <section class="card p-5"><div class="flex items-center justify-between gap-2"><h2 class="text-base font-semibold">Soll</h2><button v-if="auth.canManageParentWork" class="btn btn-secondary btn-sm" @click="startOverride">{{ detail.overrideMinutes !== undefined ? 'Manuelles Soll bearbeiten' : 'Manuelles Soll setzen' }}</button></div><p class="mt-3 text-sm">Berechnetes Soll: <strong>{{ formatHours(detail.calculatedMinutes) }}</strong></p><p v-if="detail.exemptReason" class="mt-2 text-sm">Befreiung: {{ detail.exemptReason }}</p><p v-if="detail.overrideMinutes !== undefined" class="mt-2 text-sm">Manuelles Soll: <strong>{{ formatHours(detail.overrideMinutes) }}</strong> · {{ detail.overrideReason }}</p><p v-if="actionError && !overrideDialog && !voidEntry" role="alert" class="mt-3 text-destructive">{{ actionError }}</p></section>
      </div>
      <section aria-labelledby="account-heading"><h2 id="account-heading" class="text-lg font-semibold">Konto</h2>
        <div class="mt-3 grid gap-4 sm:grid-cols-2 lg:grid-cols-3" :class="account.length === 5 ? 'xl:grid-cols-5' : 'xl:grid-cols-6'">
          <div v-for="item in account" :key="item.label" class="card p-5"><p class="text-sm text-muted-foreground">{{ item.label }}</p><p class="mt-2 text-2xl font-semibold">{{ item.value }}</p></div>
        </div></section>
      <section class="card overflow-hidden"><h2 class="p-4 text-lg font-semibold sm:p-5">Einträge</h2>
        <ul class="divide-y sm:hidden">
          <li v-for="e in sortedEntries" :key="e.id" class="space-y-2 p-4" :class="e.status === 'VOIDED' ? 'bg-muted text-muted-foreground' : ''">
            <div class="flex items-start justify-between gap-3"><div><p class="font-medium">{{ e.occasion }}</p><p class="text-[13px] text-muted-foreground">{{ formatDate(e.workDate) }} · {{ formatHours(e.durationMinutes) }} · {{ source(e.source) }}</p></div><span class="badge">{{ entryStatus(e.status) }}</span></div>
            <p class="text-[13px] text-muted-foreground">{{ e.memberName || '—' }} · {{ e.childName || '—' }}</p>
            <p v-if="e.reviewedAt" class="text-xs text-muted-foreground">von {{ e.reviewedByName || 'unbekannt' }}, {{ formatDate(e.reviewedAt) }}</p>
            <p v-if="e.rejectReason" class="text-xs">Grund: {{ e.rejectReason }}</p><p v-if="e.voidReason" class="text-xs">Grund: {{ e.voidReason }}</p>
            <div v-if="auth.canManageParentWork && e.status === 'SUBMITTED'" class="flex gap-2"><button class="btn btn-primary btn-sm" @click="review(e.id, true)">Bestätigen</button><button class="btn btn-secondary btn-sm" @click="rejectEntry = e">Ablehnen</button></div>
            <div v-else-if="auth.canManageParentWork && e.status !== 'VOIDED'" class="flex gap-2"><button class="btn btn-secondary btn-sm" @click="editingEntry = e; entryDialog = true">Bearbeiten</button><button class="btn btn-danger-outline btn-sm" @click="voidEntry = e; voidReason = ''; actionError = ''">Stornieren</button></div>
          </li>
          <li v-if="!detail.entries?.length"><p class="p-6 text-center text-sm text-muted-foreground">Keine Einträge.</p></li>
        </ul>
        <div class="hidden overflow-x-auto sm:block"><table class="table w-full text-sm"><thead class="bg-muted text-muted-foreground"><tr>
          <SortTh label="Datum" column="date" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
          <SortTh label="Stunden" column="hours" align="right" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
          <SortTh label="Anlass" column="occasion" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
          <SortTh label="Mitglied" column="member" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
          <SortTh label="Kind" column="child" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
          <SortTh label="Quelle" column="source" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
          <SortTh label="Status" column="status" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
          <th v-if="auth.canManageParentWork" class="px-4 py-3 text-left">Aktionen</th></tr></thead><tbody><tr v-for="e in sortedEntries" :key="e.id" class="border-t" :class="e.status === 'VOIDED' ? 'text-muted-foreground bg-muted' : ''"><td class="px-4 py-3">{{ formatDate(e.workDate) }}</td><td class="px-4 py-3 text-right">{{ formatHours(e.durationMinutes) }}</td><td class="px-4 py-3">{{ e.occasion }}</td><td class="px-4 py-3">{{ e.memberName || '—' }}</td><td class="px-4 py-3">{{ e.childName || '—' }}</td><td class="px-4 py-3">{{ source(e.source) }}</td><td class="px-4 py-3">{{ entryStatus(e.status) }}<span v-if="e.reviewedAt" class="block text-xs text-muted-foreground">von {{ e.reviewedByName || 'unbekannt' }}, {{ formatDate(e.reviewedAt) }}</span><span v-if="e.rejectReason" class="block text-xs">Grund: {{ e.rejectReason }}</span><span v-if="e.voidReason" class="block text-xs">Grund: {{ e.voidReason }}</span></td><td v-if="auth.canManageParentWork" class="px-4 py-3"><div v-if="e.status === 'SUBMITTED'" class="flex gap-2">
          <button class="btn btn-primary btn-sm" @click="review(e.id, true)">Bestätigen</button>
          <button class="btn btn-secondary btn-sm" @click="rejectEntry = e">Ablehnen</button>
        </div><div v-else-if="auth.canManageParentWork && e.status !== 'VOIDED'" class="flex gap-2"><button class="btn btn-secondary btn-sm" @click="editingEntry = e; entryDialog = true">Bearbeiten</button><button class="btn btn-danger-outline btn-sm" @click="voidEntry = e; voidReason = ''; actionError = ''">Stornieren</button></div></td></tr><tr v-if="!detail.entries?.length"><td :colspan="auth.canManageParentWork ? 8 : 7" class="px-4 py-6 text-center text-muted-foreground">Keine Einträge.</td></tr></tbody></table></div></section>
      <HouseholdDonationsCard v-if="auth.canAccessFees" :household-id="String(route.params.id)" />
      <section v-if="detail.boardTerms?.length" class="card overflow-hidden"><h2 class="px-4 pb-1 pt-4 text-base font-semibold">Vorstands-Amtszeiten</h2><div class="overflow-x-auto"><table class="table"><thead><tr><th>Mitglied</th><th>Amt</th><th>Von</th><th>Bis</th></tr></thead><tbody><tr v-for="term in detail.boardTerms" :key="term.id"><td>{{ term.memberName }}</td><td>{{ term.office }}</td><td>{{ formatDate(term.startDate) }}</td><td>{{ formatDate(term.endDate) }}</td></tr></tbody></table></div></section>
    </template>
    <AppSheet v-if="rejectEntry" title="Meldung ablehnen" @close="rejectEntry = null">
      <form id="reject-entry" class="space-y-3" @submit.prevent="review(rejectEntry.id, false)">
        <label class="label">Grund *<textarea v-model="rejectReason" required class="input" /></label>
        <p v-if="actionError" role="alert" class="note note-danger">{{ actionError }}</p>
      </form>
      <template #footer><button type="button" class="btn btn-secondary" @click="rejectEntry = null">Abbrechen</button><button type="submit" form="reject-entry" :disabled="saving" class="btn btn-primary">Ablehnen</button></template>
    </AppSheet>
    <EntryDialog v-if="entryDialog" :entry="editingEntry" :household-id="String(route.params.id)" @close="entryDialog = false" @saved="entrySaved" />
    <AppSheet v-if="overrideDialog" title="Manuelles Soll" @close="overrideDialog = false">
      <form id="override-form" class="space-y-4" @submit.prevent="saveOverride">
        <label class="label">Stunden *<input v-model.number="overrideHours" type="number" min="0" step="0.25" required class="input" /></label>
        <label class="label">Begründung *<textarea v-model="overrideReason" required class="input" /></label>
        <p v-if="actionError" role="alert" class="note note-danger">{{ actionError }}</p>
      </form>
      <template #footer><button v-if="detail?.overrideMinutes !== undefined" type="button" :disabled="saving" class="btn btn-danger-outline mr-auto" @click="removeOverride">Manuelles Soll entfernen</button><button type="button" class="btn btn-secondary" @click="overrideDialog = false">Abbrechen</button><button type="submit" form="override-form" :disabled="saving" class="btn btn-primary">Speichern</button></template>
    </AppSheet>
    <AppSheet v-if="voidEntry" title="Eintrag stornieren" @close="voidEntry = null">
      <form id="void-entry" class="space-y-4" @submit.prevent="submitVoid">
        <label class="label">Grund *<textarea v-model="voidReason" required class="input" /></label>
        <p v-if="actionError" role="alert" class="note note-danger">{{ actionError }}</p>
      </form>
      <template #footer><button type="button" class="btn btn-secondary" @click="voidEntry = null">Abbrechen</button><button type="submit" form="void-entry" :disabled="saving" class="btn btn-danger">Stornieren</button></template>
    </AppSheet>
  </div>
</template>
