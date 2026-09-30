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
// Parents change only e-mail and phone themselves; everything else is corrected via „Fehler melden“.
const report = ref<{ topic: 'CONTACT' | 'CHILD'; referenceId?: string } | null>(null);
const showReport = ref(false);
type Parent = OwnOverview['parent'];
type ParentContact = { id: string; name: string; own: boolean; parent: Parent; email: string; phone: string };
const contacts = ref<ParentContact[]>([]);
const children = ref<OwnChild[]>([]);
const input = 'mt-1 h-9 w-full rounded-lg border px-3 font-normal disabled:opacity-60';
const card = 'rounded-2xl border bg-card p-4';
type Address = Pick<Parent, 'street' | 'streetNo' | 'postalCode' | 'city'>;
function addressLine(a: Address) {
  const line = [[a.street, a.streetNo].filter(Boolean).join(' '),
    [a.postalCode, a.city].filter(Boolean).join(' ')].filter(Boolean).join(', ');
  return line || 'nicht hinterlegt';
}
// Everybody in the family usually lives at the same address; then it is shown once.
const sharedAddress = computed(() => {
  const all = [...contacts.value.map(c => c.parent), ...children.value];
  const key = (a: Address) => addressLine(a).toLowerCase();
  if (!all.length || addressLine(all[0]) === 'nicht hinterlegt') return null;
  return all.every(a => key(a) === key(all[0])) ? addressLine(all[0]) : null;
});
async function load() {
  try {
    overview.value = await api.getOwnOverview();
    const own = overview.value.parent;
    contacts.value = [own, ...overview.value.otherParents].map(p => ({ id: p.id,
      name: `${p.firstName} ${p.lastName}`, own: p.id === own.id, parent: p,
      email: p.email ?? '', phone: p.phone ?? '' }));
    children.value = overview.value.children;
    reports.value = await api.getOwnReports();
  } catch (e) { error.value = e instanceof Error ? e.message : 'Daten konnten nicht geladen werden'; }
}
onMounted(load);
async function saveContact(c: ParentContact) {
  error.value = ''; notice.value = '';
  try {
    if (c.own) {
      await api.updateOwnContact({ email: c.email, phone: c.phone });
      if ((c.parent.email ?? '') !== c.email) await auth.fetchUser();
    } else {
      await api.updateParentContact(c.id, { phone: c.phone });
    }
    notice.value = `Kontaktdaten von ${c.name} gespeichert.`; await load();
  } catch (e) { error.value = e instanceof Error ? e.message : 'Speichern fehlgeschlagen'; }
}
function childFacts(child: OwnChild): [string, string][] {
  return [
    ['Geburtsdatum', formatDate(child.birthDate)],
    ...(sharedAddress.value ? [] : [['Adresse', addressLine(child)] as [string, string]]),
    ['Mitgliedsnummer', child.memberNumber],
    ['Eintritt', formatDate(child.entryDate)],
    ['Austritt', child.exitDate ? formatDate(child.exitDate) : '—'],
    ['Betreuungszeit', child.careHours != null ? `${child.careHours} h/Woche` : 'nicht hinterlegt'],
    ['Rechtsanspruch', child.legalHours != null ? `${child.legalHours} h/Woche` : 'nicht hinterlegt'],
  ];
}
</script>
<template>
  <div class="space-y-5">
    <h1 class="text-2xl font-bold">Meine Daten</h1>
    <p v-if="error" role="alert" class="text-red-700 dark:text-red-300">{{ error }}</p>
    <p v-if="notice" role="status" class="text-green-800 dark:text-green-300">{{ notice }}</p>
    <div class="grid gap-4 lg:grid-cols-2">
      <form v-for="c in contacts" :key="c.id" :class="card" @submit.prevent="saveContact(c)">
        <h2 class="text-lg font-bold">{{ c.own ? 'Deine Kontaktdaten' : `Kontaktdaten von ${c.name}` }}</h2>
        <div class="mt-3 space-y-3">
          <label class="block text-sm font-medium">E-Mail
            <span class="font-normal text-muted-foreground">{{ c.own ? '– gleichzeitig dein Login!'
              : '– kann nicht von dir geändert werden, da sie zum Login dient' }}</span>
            <input v-model="c.email" type="email" :disabled="!c.own" :class="input" /></label>
          <label class="block text-sm font-medium">Telefon
            <input v-model="c.phone" type="tel" :class="input" /></label>
          <dl v-if="!sharedAddress" class="text-sm">
            <dt class="font-medium">Adresse</dt><dd class="mt-1">{{ addressLine(c.parent) }}</dd>
          </dl>
        </div>
        <div class="mt-4 flex flex-wrap items-center gap-3">
          <button type="submit" class="rounded-lg bg-primary px-4 py-2 text-primary-foreground">
            Kontaktdaten speichern
          </button>
          <button type="button" class="text-primary underline"
            @click="report = { topic: 'CONTACT' }">Fehler melden</button>
        </div>
      </form>
      <section v-if="sharedAddress" :class="card">
        <h2 class="text-lg font-bold">Adresse der Familie</h2>
        <p class="mt-3">{{ sharedAddress }}</p>
        <p class="mt-1 text-sm text-muted-foreground">Gilt für alle Eltern und Kinder.</p>
        <button type="button" class="mt-4 text-primary underline"
          @click="report = { topic: 'CONTACT' }">Fehler melden</button>
      </section>
    </div>
    <section v-if="overview" class="space-y-4">
      <h2 class="text-lg font-bold">Kinder</h2>
      <div class="grid gap-4 lg:grid-cols-2">
        <article v-for="child in children" :key="child.id" :class="card">
          <h3 class="text-lg font-bold">{{ child.firstName }} {{ child.lastName }}</h3>
          <dl class="mt-3 grid grid-cols-[auto_1fr] gap-x-6 gap-y-1 text-sm">
            <template v-for="[term, value] in childFacts(child)" :key="term">
              <dt class="text-muted-foreground">{{ term }}</dt><dd>{{ value }}</dd>
            </template>
          </dl>
          <button type="button" class="mt-4 text-primary underline"
            @click="report = { topic: 'CHILD', referenceId: child.id }">Fehler melden</button>
        </article>
      </div>
    </section>
    <section :class="card">
      <div class="flex flex-wrap items-center justify-between gap-3"><h2 class="text-lg font-bold">Meldungen deiner Familie</h2>
        <button class="text-primary underline" @click="showReport = true">Fehler melden</button></div>
      <p v-if="!reports.length" class="mt-3 text-muted-foreground">Noch keine Meldungen.</p>
      <ReportList class="mt-3" :reports="reports" />
    </section>
    <ReportDialog v-if="showReport || report" :topic="report?.topic"
      :reference-id="report?.referenceId" @close="showReport = false; report = null"
      @saved="showReport = false; report = null; load()" />
  </div>
</template>
