<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { api } from '@/api';
import type { OwnWork, OwnWorkEntry } from '@/api/types';
import { formatCurrency, formatDate, formatHours, todayISO } from '@/utils/format';
import ReportDialog from '@/components/family/ReportDialog.vue';
import DurationPicker from '@/components/parent-work/DurationPicker.vue';
import PageHeader from '@/components/ui/PageHeader.vue';
import AppSheet from '@/components/ui/AppSheet.vue';
import EmptyState from '@/components/ui/EmptyState.vue';
import { currentKitaYear, kitaYearLabel, yearOptions } from '@/utils/years';
const year = ref(currentKitaYear());
const years = yearOptions(year.value);
const work = ref<OwnWork | null>(null);
const error = ref('');
const actionError = ref('');
const showForm = ref(false);
const withdraw = ref<OwnWorkEntry | null>(null);
const reportId = ref<string | null>(null);
const date = ref(todayISO());
const minutes = ref<number | null>(null);
const occasion = ref('');
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
onMounted(load);
watch(year, load);
async function submit() {
  if (!minutes.value) { actionError.value = 'Bitte die Dauer wählen.'; return; }
  saving.value = true; actionError.value = '';
  try {
    await api.submitOwnWork({ workDate: date.value, durationMinutes: minutes.value,
      occasion: occasion.value.trim() });
    showForm.value = false; minutes.value = null; occasion.value = ''; await load();
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
    <PageHeader title="Deine Elternstunden">
      <template #actions><button class="btn btn-primary" @click="showForm = true">Stunden melden</button></template>
    </PageHeader>
    <label class="label block max-w-44">Kita-Jahr
      <select v-model.number="year" class="input">
        <option v-for="y in years" :key="y" :value="y">{{ kitaYearLabel(y) }}</option>
      </select>
    </label>
    <p v-if="error" role="alert" class="note note-danger">{{ error }}</p>
    <template v-if="work">
      <section class="card p-4 sm:p-5">
        <p v-if="work.exempt" class="font-semibold">Du bist in diesem Kita-Jahr befreit.</p>
        <div v-else class="grid grid-cols-2 gap-3 text-sm sm:grid-cols-5">
          <div v-for="item in [
            ['Soll', formatHours(work.requiredMinutes)], ['Ist', formatHours(work.doneMinutes)],
            ['Offen', formatHours(work.openMinutes)], ['Übertrag', formatHours(work.carryInMinutes)],
            ['Fehlbetrag', formatCurrency(work.missingAmountCents / 100)],
          ]" :key="item[0]"><p class="text-muted-foreground">{{ item[0] }}</p><strong>{{ item[1] }}</strong></div>
        </div>
        <div role="progressbar" :aria-valuenow="progress" aria-valuemin="0" aria-valuemax="100"
          class="mt-4 h-3 overflow-hidden rounded-lg bg-muted">
          <div class="h-full bg-primary" :style="{ width: `${progress}%` }" />
        </div>
      </section>
      <p class="text-sm text-muted-foreground">Gemeldete Stunden zählen, sobald sie bestätigt sind.</p>
      <h2 class="text-base font-semibold">Einträge</h2>
      <EmptyState v-if="!work.entries.length" text="Noch keine Einträge." />
      <article v-for="entry in work.entries" :key="entry.id" class="card p-4">
        <div class="flex items-start justify-between gap-3">
          <div><h3 class="font-semibold">{{ entry.occasion }}</h3>
            <p class="text-sm">{{ formatDate(entry.workDate) }} · {{ formatHours(entry.durationMinutes) }}</p>
            <p v-if="entry.childName" class="text-sm">{{ entry.childName }}</p></div>
          <span class="badge shrink-0" :class="
            entry.status === 'APPROVED'
              ? 'badge-success'
              : entry.status === 'REJECTED'
                ? 'badge-danger'
                : entry.status === 'SUBMITTED'
                  ? 'badge-warning'
                  : ''">
            {{ status[entry.status] ?? entry.status }}
          </span>
        </div>
        <p v-if="entry.rejectReason" class="mt-2 text-sm">Grund: {{ entry.rejectReason }}</p>
        <p v-if="entry.voidReason" class="mt-2 text-sm">Grund: {{ entry.voidReason }}</p>
        <div class="mt-3 flex gap-4 text-sm">
          <button v-if="entry.status === 'SUBMITTED'" class="btn btn-secondary"
            @click="withdraw = entry">Zurückziehen</button>
          <button class="btn btn-secondary" @click="reportId = entry.id">Fehler melden</button>
        </div>
      </article>
    </template>
    <AppSheet v-if="showForm" title="Stunden melden" @close="showForm = false">
      <form id="own-work-form" class="space-y-3" @submit.prevent="submit">
        <label class="label">Datum
          <input v-model="date" type="date" :max="todayISO()" required
            class="input sm:w-56" /></label>
        <DurationPicker v-model="minutes" label="Dauer" />
        <label class="label">Anlass
          <input v-model="occasion" required class="input" /></label>
        <p v-if="actionError" role="alert" class="note note-danger">{{ actionError }}</p>
      </form>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="showForm = false">Abbrechen</button>
        <button type="submit" form="own-work-form" :disabled="saving" class="btn btn-primary">Melden</button>
      </template>
    </AppSheet>
    <AppSheet v-if="withdraw" title="Meldung zurückziehen" @close="withdraw = null">
        <h2 class="sr-only">Meldung zurückziehen?</h2>
        <p class="mt-2">{{ withdraw.occasion }} wird storniert.</p>
        <p v-if="actionError" role="alert" class="note note-danger">{{ actionError }}</p>
      <template #footer>
        <button class="btn btn-secondary" @click="withdraw = null">Abbrechen</button>
        <button :disabled="saving" class="btn btn-primary" @click="confirmWithdraw">Zurückziehen</button>
      </template>
    </AppSheet>
    <ReportDialog v-if="reportId" topic="PARENT_WORK" :reference-id="reportId"
      @close="reportId = null" @saved="reportId = null" />
  </div>
</template>
