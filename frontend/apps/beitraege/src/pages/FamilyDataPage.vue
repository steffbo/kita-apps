<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { api } from '@/api';
import type { OwnOverview, OwnChild, ParentReport } from '@/api/types';
import { useAuthStore } from '@/stores/auth';
import { formatDate } from '@/utils/format';
import ReportDialog from '@/components/family/ReportDialog.vue';
const auth = useAuthStore();
const overview = ref<OwnOverview | null>(null);
const reports = ref<ParentReport[]>([]);
const error = ref('');
const notice = ref('');
const reportId = ref<string | null>(null);
const showReport = ref(false);
const contact = ref({ email: '', phone: '', street: '', streetNo: '', postalCode: '', city: '' });
const children = ref<OwnChild[]>([]);
function dateInput(value: string) { return value.slice(0, 10); }
async function load() {
  try {
    overview.value = await api.getOwnOverview();
    const p = overview.value.parent;
    contact.value = { email: p.email ?? '', phone: p.phone ?? '', street: p.street ?? '',
      streetNo: p.streetNo ?? '', postalCode: p.postalCode ?? '', city: p.city ?? '' };
    children.value = overview.value.children.map(c => ({ ...c, birthDate: dateInput(c.birthDate) }));
    reports.value = await api.getOwnReports();
  } catch (e) { error.value = e instanceof Error ? e.message : 'Daten konnten nicht geladen werden'; }
}
onMounted(load);
async function saveContact() {
  error.value = ''; notice.value = '';
  try {
    const previous = overview.value?.parent.email;
    await api.updateOwnContact(contact.value);
    if (previous !== contact.value.email) await auth.fetchUser();
    notice.value = 'Kontaktdaten gespeichert.'; await load();
  } catch (e) { error.value = e instanceof Error ? e.message : 'Speichern fehlgeschlagen'; }
}
async function saveChild(child: OwnChild) {
  error.value = ''; notice.value = '';
  try {
    await api.updateOwnChild(child.id, { firstName: child.firstName, lastName: child.lastName,
      birthDate: child.birthDate, street: child.street ?? '', streetNo: child.streetNo ?? '',
      postalCode: child.postalCode ?? '', city: child.city ?? '' });
    notice.value = 'Kinderdaten gespeichert.'; await load();
  } catch (e) { error.value = e instanceof Error ? e.message : 'Speichern fehlgeschlagen'; }
}
</script>
<template>
  <div class="space-y-5">
    <h1 class="text-2xl font-bold">Meine Daten</h1>
    <p v-if="error" role="alert" class="text-red-700 dark:text-red-300">{{ error }}</p>
    <p v-if="notice" role="status" class="text-green-800 dark:text-green-300">{{ notice }}</p>
    <form v-if="overview" class="rounded-2xl border bg-card p-5" @submit.prevent="saveContact">
      <h2 class="text-lg font-bold">Kontaktdaten</h2>
      <p class="mt-1 text-sm text-muted-foreground">Deine E-Mail-Adresse ist auch dein Login.</p>
      <div class="mt-4 grid gap-3 sm:grid-cols-2">
        <label v-for="field in [
          ['email', 'E-Mail'], ['phone', 'Telefon'], ['street', 'Straße'],
          ['streetNo', 'Nr.'], ['postalCode', 'PLZ'], ['city', 'Ort'],
        ] as const" :key="field[0]" class="block text-sm font-medium">{{ field[1] }}
          <input v-model="contact[field[0]]" :type="field[0] === 'email' ? 'email' : 'text'"
            class="mt-1 w-full rounded-lg border px-3 py-2" /></label>
      </div>
      <button type="submit" class="mt-4 rounded-lg bg-primary px-4 py-2 text-primary-foreground">
        Kontaktdaten speichern
      </button>
    </form>
    <section v-if="overview" class="space-y-4">
      <h2 class="text-lg font-bold">Kinder</h2>
      <form v-for="child in children" :key="child.id" class="rounded-2xl border bg-card p-5"
        @submit.prevent="saveChild(child)">
        <h3 class="font-bold">{{ child.firstName }} {{ child.lastName }}</h3>
        <div class="mt-4 grid gap-3 sm:grid-cols-2">
          <label class="text-sm font-medium">Vorname
            <input v-model="child.firstName" required class="mt-1 w-full rounded-lg border px-3 py-2" /></label>
          <label class="text-sm font-medium">Nachname
            <input v-model="child.lastName" required class="mt-1 w-full rounded-lg border px-3 py-2" /></label>
          <label class="text-sm font-medium">Geburtsdatum
            <input v-model="child.birthDate" type="date" required
              class="mt-1 w-full rounded-lg border px-3 py-2" /></label>
          <label v-for="field in [
            ['street', 'Straße'], ['streetNo', 'Nr.'], ['postalCode', 'PLZ'], ['city', 'Ort'],
          ] as const" :key="field[0]" class="text-sm font-medium">{{ field[1] }}
            <input v-model="child[field[0]]" class="mt-1 w-full rounded-lg border px-3 py-2" /></label>
        </div>
        <div class="mt-4 text-sm text-muted-foreground">
          <p>Mitgliedsnummer: {{ child.memberNumber }}</p>
          <p>Eintritt: {{ formatDate(child.entryDate) }}</p>
          <p>Austritt: {{ child.exitDate ? formatDate(child.exitDate) : '—' }}</p>
          <p>Betreuungszeit: {{ child.careHours != null ? `${child.careHours} h/Woche` : 'nicht hinterlegt' }}</p>
          <p>Rechtsanspruch: {{ child.legalHours != null ? `${child.legalHours} h/Woche` : 'nicht hinterlegt' }}</p>
          <p class="mt-2">Änderungen daran bitte über „Fehler melden“.</p>
        </div>
        <div class="mt-4 flex flex-wrap gap-3">
          <button type="submit" class="rounded-lg bg-primary px-4 py-2 text-primary-foreground">Speichern</button>
          <button type="button" class="text-primary underline"
            @click="reportId = child.id">Fehler melden</button>
        </div>
      </form>
    </section>
    <section class="rounded-2xl border bg-card p-5">
      <div class="flex flex-wrap items-center justify-between gap-3"><h2 class="text-lg font-bold">Meine Meldungen</h2>
        <button class="text-primary underline" @click="showReport = true">Fehler melden</button></div>
      <p v-if="!reports.length" class="mt-3 text-muted-foreground">Noch keine Meldungen.</p>
      <ul class="mt-3 space-y-3"><li v-for="report in reports" :key="report.id" class="border-t pt-3">
        <div class="flex justify-between gap-3"><strong>{{ report.message }}</strong>
          <span>{{ report.status === 'OPEN' ? 'Offen' : 'Erledigt' }}</span></div>
        <p class="text-xs text-muted-foreground">{{ formatDate(report.createdAt) }}</p>
      </li></ul>
    </section>
    <ReportDialog v-if="showReport || reportId" :topic="reportId ? 'CHILD' : undefined"
      :reference-id="reportId ?? undefined" @close="showReport = false; reportId = null"
      @saved="showReport = false; reportId = null; load()" />
  </div>
</template>
