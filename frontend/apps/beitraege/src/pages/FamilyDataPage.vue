<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { api } from '@/api';
import type { OwnOverview, OwnChild, ParentReport } from '@/api/types';
import { useAuthStore } from '@/stores/auth';
import { formatDate } from '@/utils/format';
import ReportDialog from '@/components/family/ReportDialog.vue';
import ReportList from '@/components/family/ReportList.vue';
const auth = useAuthStore();
const overview = ref<OwnOverview | null>(null);
const reports = ref<ParentReport[]>([]);
const error = ref('');
const notice = ref('');
const reportId = ref<string | null>(null);
const showReport = ref(false);
type ContactForm = { email: string; phone: string; street: string; streetNo: string;
  postalCode: string; city: string };
type ParentContact = { id: string; name: string; own: boolean; savedEmail: string; form: ContactForm };
const contacts = ref<ParentContact[]>([]);
// Same 12-column grid for parents and children, so the address fields line up across all cards.
const addressFields = [
  ['street', 'Straße', 'sm:col-span-5'], ['streetNo', 'Hausnummer', 'sm:col-span-2'],
  ['postalCode', 'PLZ', 'sm:col-span-2'], ['city', 'Ort', 'sm:col-span-3'],
] as const;
const contactFields = [
  ['email', 'E-Mail', 'sm:col-span-6'], ['phone', 'Telefon', 'sm:col-span-6'], ...addressFields,
] as const;
const input = 'mt-1 h-9 w-full rounded-lg border px-3 font-normal disabled:opacity-60';
function formFrom(p: OwnOverview['parent']): ContactForm {
  return { email: p.email ?? '', phone: p.phone ?? '', street: p.street ?? '',
    streetNo: p.streetNo ?? '', postalCode: p.postalCode ?? '', city: p.city ?? '' };
}
const children = ref<OwnChild[]>([]);
function dateInput(value: string) { return value.slice(0, 10); }
async function load() {
  try {
    overview.value = await api.getOwnOverview();
    const own = overview.value.parent;
    contacts.value = [
      { id: own.id, name: `${own.firstName} ${own.lastName}`, own: true,
        savedEmail: own.email ?? '', form: formFrom(own) },
      ...overview.value.otherParents.map(p => ({ id: p.id, name: `${p.firstName} ${p.lastName}`,
        own: false, savedEmail: p.email ?? '', form: formFrom(p) })),
    ];
    children.value = overview.value.children.map(c => ({ ...c, birthDate: dateInput(c.birthDate) }));
    reports.value = await api.getOwnReports();
  } catch (e) { error.value = e instanceof Error ? e.message : 'Daten konnten nicht geladen werden'; }
}
onMounted(load);
async function saveContact(c: ParentContact) {
  error.value = ''; notice.value = '';
  try {
    if (c.own) {
      await api.updateOwnContact(c.form);
      if (c.savedEmail !== c.form.email) await auth.fetchUser();
    } else {
      await api.updateParentContact(c.id, c.form);
    }
    notice.value = `Kontaktdaten von ${c.name} gespeichert.`; await load();
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
    <form v-for="c in contacts" :key="c.id" class="rounded-2xl border bg-card p-4"
      @submit.prevent="saveContact(c)">
      <h2 class="text-lg font-bold">{{ c.own ? 'Deine Kontaktdaten' : `Kontaktdaten von ${c.name}` }}</h2>
      <p class="text-sm text-muted-foreground">
        {{ c.own ? 'Deine E-Mail-Adresse ist auch dein Login.'
          : 'Du kannst auch die Daten des anderen Elternteils pflegen, außer der E-Mail-Adresse.' }}</p>
      <div class="mt-3 grid gap-3 sm:grid-cols-12">
        <label v-for="field in contactFields" :key="field[0]" class="text-sm font-medium" :class="field[2]">
          {{ field[1] }}
          <input v-model="c.form[field[0]]" :type="field[0] === 'email' ? 'email' : 'text'"
            :disabled="field[0] === 'email' && !c.own" :class="input" /></label>
      </div>
      <button type="submit" class="mt-3 rounded-lg bg-primary px-4 py-2 text-primary-foreground">
        Kontaktdaten speichern
      </button>
    </form>
    <section v-if="overview" class="space-y-4">
      <h2 class="text-lg font-bold">Kinder</h2>
      <form v-for="child in children" :key="child.id" class="rounded-2xl border bg-card p-4"
        @submit.prevent="saveChild(child)">
        <h3 class="text-lg font-bold">{{ child.firstName }} {{ child.lastName }}</h3>
        <div class="mt-3 grid gap-3 sm:grid-cols-12">
          <label class="text-sm font-medium sm:col-span-4">Vorname
            <input v-model="child.firstName" required :class="input" /></label>
          <label class="text-sm font-medium sm:col-span-4">Nachname
            <input v-model="child.lastName" required :class="input" /></label>
          <label class="text-sm font-medium sm:col-span-4">Geburtsdatum
            <input v-model="child.birthDate" type="date" required :class="input" /></label>
          <label v-for="field in addressFields" :key="field[0]" class="text-sm font-medium"
            :class="field[2]">{{ field[1] }}
            <input v-model="child[field[0]]" :class="input" /></label>
        </div>
        <dl class="mt-3 flex flex-wrap gap-x-6 gap-y-1 text-sm">
          <div v-for="[term, value] in [
            ['Mitgliedsnummer', child.memberNumber],
            ['Eintritt', formatDate(child.entryDate)],
            ['Austritt', child.exitDate ? formatDate(child.exitDate) : '—'],
            ['Betreuungszeit', child.careHours != null ? `${child.careHours} h/Woche` : 'nicht hinterlegt'],
            ['Rechtsanspruch', child.legalHours != null ? `${child.legalHours} h/Woche` : 'nicht hinterlegt'],
          ]" :key="term"><dt class="inline text-muted-foreground">{{ term }}:</dt> <dd class="inline">{{ value }}</dd></div>
        </dl>
        <div class="mt-3 flex flex-wrap items-center gap-3">
          <button type="submit" class="rounded-lg bg-primary px-4 py-2 text-primary-foreground">Speichern</button>
          <button type="button" class="text-primary underline"
            @click="reportId = child.id">Fehler melden</button>
        </div>
      </form>
    </section>
    <section class="rounded-2xl border bg-card p-4">
      <div class="flex flex-wrap items-center justify-between gap-3"><h2 class="text-lg font-bold">Meldungen deiner Familie</h2>
        <button class="text-primary underline" @click="showReport = true">Fehler melden</button></div>
      <p v-if="!reports.length" class="mt-3 text-muted-foreground">Noch keine Meldungen.</p>
      <ReportList class="mt-3" :reports="reports" />
    </section>
    <ReportDialog v-if="showReport || reportId" :topic="reportId ? 'CHILD' : undefined"
      :reference-id="reportId ?? undefined" @close="showReport = false; reportId = null"
      @saved="showReport = false; reportId = null; load()" />
  </div>
</template>
