<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { api } from '@/api';
import type { OwnOverview, OwnWork, OwnWorkEntry } from '@/api/types';
import { formatCurrency, formatDate, formatHours, todayISO } from '@/utils/format';
import ReportDialog from '@/components/family/ReportDialog.vue';
const berlinDate = todayISO();
const currentYear = Number(berlinDate.slice(0, 4));
const year = ref(Number(berlinDate.slice(5, 7)) >= 8 ? currentYear : currentYear - 1);
// Elternstunden are tracked in the app from Kita-Jahr 2025/26 on.
const firstYear = 2025;
const years = Array.from({ length: year.value + 2 - firstYear }, (_, i) => year.value + 1 - i);
const work = ref<OwnWork | null>(null);
const overview = ref<OwnOverview | null>(null);
const error = ref('');
const actionError = ref('');
const showForm = ref(false);
const withdraw = ref<OwnWorkEntry | null>(null);
const reportId = ref<string | null>(null);
const date = ref(todayISO());
const hours = ref<number | null>(null);
const occasion = ref('');
const memberName = ref('');
const childName = ref('');
const saving = ref(false);
const progress = computed(() => work.value?.requiredMinutes
  ? Math.min(100, 100 * work.value.doneMinutes / work.value.requiredMinutes) : 0);
const status: Record<string, string> = {
  SUBMITTED: 'Eingereicht', APPROVED: 'Bestätigt', REJECTED: 'Abgelehnt', VOIDED: 'Storniert',
};
async function load() {
  error.value = '';
  try { work.value = await api.getOwnWork(year.value); }
  catch (e) { error.value = e instanceof Error ? e.message : 'Elternstunden konnten nicht geladen werden'; }
}
onMounted(async () => {
  await load();
  try { overview.value = await api.getOwnOverview(); } catch { /* Work page can still show the account. */ }
});
watch(year, load);
async function submit() {
  const minutes = Math.round((hours.value ?? 0) * 60);
  if (!hours.value || minutes % 15 || minutes / 60 !== hours.value) {
    actionError.value = 'Bitte Stunden in Viertelstunden eingeben.'; return;
  }
  saving.value = true; actionError.value = '';
  try {
    await api.submitOwnWork({ workDate: date.value, durationMinutes: minutes,
      occasion: occasion.value.trim(), memberName: memberName.value || undefined,
      childName: childName.value || undefined });
    showForm.value = false; hours.value = null; occasion.value = ''; await load();
  } catch (e) { actionError.value = e instanceof Error ? e.message : 'Meldung fehlgeschlagen'; }
  finally { saving.value = false; }
}
async function confirmWithdraw() {
  if (!withdraw.value) return;
  saving.value = true; actionError.value = '';
  try { await api.withdrawOwnWork(withdraw.value.id); withdraw.value = null; await load(); }
  catch (e) { actionError.value = e instanceof Error ? e.message : 'Zurückziehen fehlgeschlagen'; }
  finally { saving.value = false; }
}
</script>
<template>
  <div class="space-y-5">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h1 class="text-2xl font-bold">Deine Elternstunden</h1>
      <button class="rounded-lg bg-primary px-4 py-2 text-primary-foreground" @click="showForm = true">
        Stunden melden
      </button>
    </div>
    <label class="block max-w-44 text-sm font-medium">Kita-Jahr
      <select v-model.number="year" class="mt-1 w-full rounded-lg border px-3 py-2">
        <option v-for="y in years" :key="y" :value="y">{{ y }}/{{ String(y + 1).slice(-2) }}</option>
      </select>
    </label>
    <p v-if="error" role="alert" class="text-red-700 dark:text-red-300">{{ error }}</p>
    <template v-if="work">
      <section class="rounded-2xl border bg-card p-5">
        <p v-if="work.exempt" class="font-bold">Du bist in diesem Kita-Jahr befreit.</p>
        <div v-else class="grid grid-cols-2 gap-3 text-sm sm:grid-cols-5">
          <div v-for="item in [
            ['Soll', formatHours(work.requiredMinutes)], ['Ist', formatHours(work.doneMinutes)],
            ['Offen', formatHours(work.openMinutes)], ['Übertrag', formatHours(work.carryInMinutes)],
            ['Fehlbetrag', formatCurrency(work.missingAmountCents / 100)],
          ]" :key="item[0]"><p class="text-muted-foreground">{{ item[0] }}</p><strong>{{ item[1] }}</strong></div>
        </div>
        <div role="progressbar" :aria-valuenow="progress" aria-valuemin="0" aria-valuemax="100"
          class="mt-4 h-3 overflow-hidden rounded-full bg-muted">
          <div class="h-full bg-primary" :style="{ width: `${progress}%` }" />
        </div>
      </section>
      <p class="text-sm text-muted-foreground">Gemeldete Stunden zählen, sobald sie bestätigt sind.</p>
      <h2 class="text-lg font-bold">Einträge</h2>
      <p v-if="!work.entries.length" class="text-muted-foreground">Noch keine Einträge.</p>
      <article v-for="entry in work.entries" :key="entry.id" class="rounded-2xl border bg-card p-4">
        <div class="flex items-start justify-between gap-3">
          <div><h3 class="font-bold">{{ entry.occasion }}</h3>
            <p class="text-sm">{{ formatDate(entry.workDate) }} · {{ formatHours(entry.durationMinutes) }}</p>
            <p v-if="entry.childName" class="text-sm">{{ entry.childName }}</p></div>
          <span class="shrink-0 whitespace-nowrap rounded-full px-2.5 py-0.5 text-xs font-semibold" :class="
            entry.status === 'APPROVED'
              ? 'bg-green-100 text-green-800 dark:bg-green-950/40 dark:text-green-300'
              : entry.status === 'REJECTED'
                ? 'bg-red-100 text-red-800 dark:bg-red-950/40 dark:text-red-300'
                : entry.status === 'SUBMITTED'
                  ? 'bg-amber-100 text-amber-800 dark:bg-amber-950/40 dark:text-amber-300'
                  : 'bg-muted text-muted-foreground'">
            {{ status[entry.status] ?? entry.status }}
          </span>
        </div>
        <p v-if="entry.rejectReason" class="mt-2 text-sm">Grund: {{ entry.rejectReason }}</p>
        <p v-if="entry.voidReason" class="mt-2 text-sm">Grund: {{ entry.voidReason }}</p>
        <div class="mt-3 flex gap-4 text-sm">
          <button v-if="entry.status === 'SUBMITTED'" class="text-primary underline"
            @click="withdraw = entry">Zurückziehen</button>
          <button class="text-primary underline" @click="reportId = entry.id">Fehler melden</button>
        </div>
      </article>
    </template>
    <div v-if="showForm" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
      @click.self="showForm = false">
      <form role="dialog" aria-modal="true" aria-label="Stunden melden"
        class="max-h-[90vh] w-full max-w-lg space-y-3 overflow-y-auto rounded-2xl bg-card p-5"
        @submit.prevent="submit">
        <h2 class="text-xl font-bold">Stunden melden</h2>
        <label class="block text-sm font-medium">Datum
          <input v-model="date" type="date" :max="todayISO()" required
            class="mt-1 w-full rounded-lg border px-3 py-2" /></label>
        <label class="block text-sm font-medium">Stunden
          <input v-model.number="hours" type="number" min="0.25" step="0.25" required
            class="mt-1 w-full rounded-lg border px-3 py-2" /></label>
        <label class="block text-sm font-medium">Anlass
          <input v-model="occasion" required class="mt-1 w-full rounded-lg border px-3 py-2" /></label>
        <label class="block text-sm font-medium">Mitglied (optional)
          <input v-model="memberName" class="mt-1 w-full rounded-lg border px-3 py-2" /></label>
        <label class="block text-sm font-medium">Kind (optional)
          <select v-model="childName" class="mt-1 w-full rounded-lg border px-3 py-2">
            <option value="">Ohne Auswahl</option>
            <option v-for="child in overview?.children ?? []" :key="child.id"
              :value="`${child.firstName} ${child.lastName}`">{{ child.firstName }} {{ child.lastName }}</option>
          </select></label>
        <p v-if="actionError" role="alert" class="text-red-700 dark:text-red-300">{{ actionError }}</p>
        <div class="flex justify-end gap-2"><button type="button" class="rounded-lg border px-4 py-2"
          @click="showForm = false">Abbrechen</button>
          <button type="submit" :disabled="saving" class="rounded-lg bg-primary px-4 py-2
            text-primary-foreground">Melden</button></div>
      </form>
    </div>
    <div v-if="withdraw" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div role="dialog" aria-modal="true" aria-label="Meldung zurückziehen"
        class="w-full max-w-md rounded-2xl bg-card p-5">
        <h2 class="text-lg font-bold">Meldung zurückziehen?</h2>
        <p class="mt-2">{{ withdraw.occasion }} wird storniert.</p>
        <p v-if="actionError" role="alert" class="text-red-700 dark:text-red-300">{{ actionError }}</p>
        <div class="mt-4 flex justify-end gap-2">
          <button class="rounded-lg border px-4 py-2" @click="withdraw = null">Abbrechen</button>
          <button :disabled="saving" class="rounded-lg bg-primary px-4 py-2 text-primary-foreground"
            @click="confirmWithdraw">Zurückziehen</button>
        </div>
      </div>
    </div>
    <ReportDialog v-if="reportId" topic="PARENT_WORK" :reference-id="reportId"
      @close="reportId = null" @saved="reportId = null" />
  </div>
</template>
