<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
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
// All cards use the same two equal columns, so every field has the same width.
const addressFields = [
  ['street', 'Straße'], ['streetNo', 'Hausnummer'], ['postalCode', 'PLZ'], ['city', 'Ort'],
] as const;
type Address = Record<(typeof addressFields)[number][0], string>;
const input = 'mt-1 h-9 w-full rounded-lg border px-3 font-normal disabled:opacity-60';
const grid = 'mt-3 grid max-w-2xl gap-3 sm:grid-cols-2';
function formFrom(p: OwnOverview['parent']): ContactForm {
  return { email: p.email ?? '', phone: p.phone ?? '', street: p.street ?? '',
    streetNo: p.streetNo ?? '', postalCode: p.postalCode ?? '', city: p.city ?? '' };
}
const children = ref<OwnChild[]>([]);
function dateInput(value: string) { return value.slice(0, 10); }
// Addresses stay per person in the backend. When everyone in the family has the same non-empty
// address, the page shows it once and saves it for everybody.
const familyAddress = ref<Address>(emptyAddress());
const separateAddresses = ref(true);
function emptyAddress(): Address { return { street: '', streetNo: '', postalCode: '', city: '' }; }
function addressOf(a: Partial<Record<keyof Address, string | null>>): Address {
  return { street: a.street ?? '', streetNo: a.streetNo ?? '', postalCode: a.postalCode ?? '', city: a.city ?? '' };
}
function addressKey(a: Address) {
  return addressFields.map(([key]) => a[key].trim().toLowerCase()).join('|');
}
function sharedAddress(): Address | null {
  const all = [...contacts.value.map(c => addressOf(c.form)), ...children.value.map(addressOf)];
  const first = all[0];
  if (!first || addressKey(first) === '|||') return null;
  return all.every(a => addressKey(a) === addressKey(first)) ? first : null;
}
const ownAddress = computed(() => contacts.value[0] ? addressOf(contacts.value[0].form) : emptyAddress());
const hasOwnAddress = computed(() => addressKey(ownAddress.value) !== '|||');
function takeOwnAddress(target: Partial<Address>) { Object.assign(target, ownAddress.value); }
async function saveFamilyAddress() {
  error.value = ''; notice.value = '';
  const address = { ...familyAddress.value };
  const saved = overview.value;
  if (!saved) return;
  // Start from the loaded data, so unsaved edits in other cards are not sent along.
  try {
    await api.updateOwnContact({ ...formFrom(saved.parent), ...address });
    for (const p of saved.otherParents) await api.updateParentContact(p.id, { ...formFrom(p), ...address });
    for (const child of saved.children) {
      await api.updateOwnChild(child.id, { firstName: child.firstName, lastName: child.lastName,
        birthDate: dateInput(child.birthDate), ...address });
    }
    notice.value = 'Adresse der Familie gespeichert.'; await load();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Speichern fehlgeschlagen';
    await load();
  }
}
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
    const shared = sharedAddress();
    familyAddress.value = shared ?? emptyAddress();
    separateAddresses.value = !shared;
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
      <div :class="grid">
        <label class="text-sm font-medium">E-Mail
          <input v-model="c.form.email" type="email" :disabled="!c.own" :class="input" /></label>
        <label class="text-sm font-medium">Telefon
          <input v-model="c.form.phone" :class="input" /></label>
        <template v-if="separateAddresses">
          <label v-for="[key, label] in addressFields" :key="key" class="text-sm font-medium">{{ label }}
            <input v-model="c.form[key]" :class="input" /></label>
        </template>
      </div>
      <div class="mt-3 flex flex-wrap items-center gap-3">
        <button type="submit" class="rounded-lg bg-primary px-4 py-2 text-primary-foreground">
          Kontaktdaten speichern
        </button>
        <button v-if="separateAddresses && !c.own && hasOwnAddress" type="button"
          class="text-primary underline" @click="takeOwnAddress(c.form)">Adresse übernehmen</button>
      </div>
    </form>
    <form v-if="overview && !separateAddresses" class="rounded-2xl border bg-card p-4"
      @submit.prevent="saveFamilyAddress">
      <h2 class="text-lg font-bold">Adresse der Familie</h2>
      <p class="text-sm text-muted-foreground">
        Gilt für {{ [...contacts.map(c => c.name), ...children.map(c => `${c.firstName} ${c.lastName}`)].join(', ') }}.
      </p>
      <div :class="grid">
        <label v-for="[key, label] in addressFields" :key="key" class="text-sm font-medium">{{ label }}
          <input v-model="familyAddress[key]" :class="input" /></label>
      </div>
      <div class="mt-3 flex flex-wrap items-center gap-3">
        <button type="submit" class="rounded-lg bg-primary px-4 py-2 text-primary-foreground">Adresse speichern</button>
        <button type="button" class="text-primary underline" @click="separateAddresses = true">
          Abweichende Adresse für einzelne Personen</button>
      </div>
    </form>
    <section v-if="overview" class="space-y-4">
      <h2 class="text-lg font-bold">Kinder</h2>
      <form v-for="child in children" :key="child.id" class="rounded-2xl border bg-card p-4"
        @submit.prevent="saveChild(child)">
        <h3 class="text-lg font-bold">{{ child.firstName }} {{ child.lastName }}</h3>
        <div :class="grid">
          <label class="text-sm font-medium">Vorname
            <input v-model="child.firstName" required :class="input" /></label>
          <label class="text-sm font-medium">Nachname
            <input v-model="child.lastName" required :class="input" /></label>
          <label class="text-sm font-medium">Geburtsdatum
            <input v-model="child.birthDate" type="date" required :class="input" /></label>
          <template v-if="separateAddresses">
            <span class="hidden sm:block" />
            <label v-for="[key, label] in addressFields" :key="key" class="text-sm font-medium">{{ label }}
              <input v-model="child[key]" :class="input" /></label>
          </template>
        </div>
        <dl class="mt-4 grid max-w-md grid-cols-[auto_1fr] gap-x-6 gap-y-1 text-sm">
          <template v-for="[term, value] in [
            ['Mitgliedsnummer', child.memberNumber],
            ['Eintritt', formatDate(child.entryDate)],
            ['Austritt', child.exitDate ? formatDate(child.exitDate) : '—'],
            ['Betreuungszeit', child.careHours != null ? `${child.careHours} h/Woche` : 'nicht hinterlegt'],
            ['Rechtsanspruch', child.legalHours != null ? `${child.legalHours} h/Woche` : 'nicht hinterlegt'],
          ]" :key="term"><dt class="text-muted-foreground">{{ term }}</dt><dd>{{ value }}</dd></template>
        </dl>
        <div class="mt-3 flex flex-wrap items-center gap-3">
          <button type="submit" class="rounded-lg bg-primary px-4 py-2 text-primary-foreground">Speichern</button>
          <button v-if="separateAddresses && hasOwnAddress" type="button" class="text-primary underline"
            @click="takeOwnAddress(child)">Adresse übernehmen</button>
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
