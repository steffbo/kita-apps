<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { api } from '@/api';
import type { ParentWorkRule } from '@/api/types';
import { useAuthStore } from '@/stores/auth';
import { formatCurrency, formatDate, formatDateForInput, formatHours, todayISO } from '@/utils/format';

const auth = useAuthStore();
const rules = ref<ParentWorkRule[]>([]);
const error = ref('');
const actionError = ref('');
const loading = ref(false);
const saving = ref(false);
const dialog = ref(false);
const editing = ref<ParentWorkRule | null>(null);
const validFrom = ref('');
const hoursPerChild = ref<number | undefined>();
const missingRate = ref<number | undefined>();
const maxCarry = ref<number | undefined>();
const nextAugust = (() => { const current = todayISO(); const year = Number(current.slice(0, 4)); const candidate = `${year}-08-01`; return candidate > current ? candidate : `${year + 1}-08-01`; })();
function editable(rule: ParentWorkRule) { return formatDateForInput(rule.validFrom) > todayISO(); }
async function load() { loading.value = true; error.value = ''; try { rules.value = (await api.getParentWorkRules()) ?? []; } catch (e) { error.value = e instanceof Error ? e.message : 'Regeln konnten nicht geladen werden'; } finally { loading.value = false; } }
onMounted(load);
function open(rule?: ParentWorkRule) { const values = rule ?? rules.value[rules.value.length - 1]; editing.value = rule ?? null; validFrom.value = formatDateForInput(rule?.validFrom) || nextAugust; hoursPerChild.value = values ? values.hoursPerChildMinutes / 60 : undefined; missingRate.value = values ? values.missingHourRateCents / 100 : undefined; maxCarry.value = values ? values.maxCarryOverMinutes / 60 : undefined; actionError.value = ''; dialog.value = true; }
function toMinutes(value: number | undefined) { if (value === undefined) return null; const minutes = Math.round(value * 60); return Number.isFinite(value) && value >= 0 && Math.abs(minutes / 60 - value) < 0.00001 ? minutes : null; }
async function save() {
  const child = toMinutes(hoursPerChild.value), carry = toMinutes(maxCarry.value), cents = Math.round((missingRate.value ?? NaN) * 100);
  if (child === null || carry === null || missingRate.value === undefined || !Number.isFinite(missingRate.value) || missingRate.value < 0 || Math.abs(cents / 100 - missingRate.value) > 0.00001) { actionError.value = 'Bitte gültige Stunden und Eurobeträge eingeben.'; return; }
  saving.value = true; actionError.value = '';
  try { const data = { validFrom: validFrom.value, hoursPerChildMinutes: child, missingHourRateCents: cents, maxCarryOverMinutes: carry }; if (editing.value) await api.updateParentWorkRule(editing.value.id, data); else await api.createParentWorkRule(data); dialog.value = false; await load(); }
  catch (e) { actionError.value = e instanceof Error ? e.message : 'Speichern fehlgeschlagen'; }
  finally { saving.value = false; }
}
</script>

<template>
  <section class="space-y-6"><div class="flex items-center justify-between gap-3"><div><h2 class="text-xl font-semibold">Regeln</h2><p class="text-sm text-muted-foreground">Versionen des Regelwerks</p></div><button v-if="auth.isAdmin" class="rounded-lg bg-primary px-4 py-2 text-primary-foreground" @click="open()">Neue Version</button></div>
    <div class="rounded-xl border bg-card p-5 text-sm">
      <h3 class="font-semibold">So wird das Soll berechnet</h3>
      <ul class="mt-3 list-disc space-y-2 pl-5">
        <li><strong>Tertiale:</strong> Das Kita-Jahr (1. August bis 31. Juli) besteht aus drei Tertialen:
          1. Tertial 1.8.–30.11., 2. Tertial 1.12.–31.3., 3. Tertial 1.4.–31.7.</li>
        <li>Ein Kind zählt für ein Tertial, wenn es darin mindestens einen Tag betreut wird. Das Soll je Kind
          ist „Stunden je Kind“ × gezählte Tertiale ÷ 3, das Soll der Familie die Summe über alle ihre Kinder.</li>
        <li><strong>Vorstandsbetreuung:</strong> Familien, aus denen ein Mitglied im Kita-Jahr mindestens einen Tag
          im Vorstand ist, sind für das gesamte Kita-Jahr befreit (Soll 0 Stunden). Die Amtszeiten pflegen
          Admins unter „Vorstand“.</li>
        <li>Nur bestätigte Stunden zählen. Ein manuelles Soll je Familie ersetzt die Berechnung.</li>
        <li>Überschüssige Stunden werden bis zum maximalen Übertrag ins Folgejahr übernommen. Für jede am Ende offene
          Stunde gilt der Satz je Fehlstunde.</li>
      </ul>
    </div>
    <p v-if="error" role="alert" class="rounded-lg bg-red-50 dark:bg-red-950/40 p-3 text-red-700 dark:text-red-300">{{ error }}</p>
    <div class="overflow-x-auto rounded-xl border bg-card"><table class="w-full text-sm"><thead class="bg-muted text-left text-muted-foreground"><tr><th class="px-4 py-3">Gültig ab</th><th class="px-4 py-3">Stunden je Kind</th><th class="px-4 py-3">Satz je Fehlstunde</th><th class="px-4 py-3">Max. Übertrag</th><th v-if="auth.isAdmin" class="px-4 py-3">Aktion</th></tr></thead><tbody><tr v-for="r in rules" :key="r.id" class="border-t"><td class="px-4 py-3">{{ formatDate(r.validFrom) }}</td><td class="px-4 py-3">{{ formatHours(r.hoursPerChildMinutes) }}</td><td class="px-4 py-3">{{ formatCurrency(r.missingHourRateCents / 100) }}</td><td class="px-4 py-3">{{ formatHours(r.maxCarryOverMinutes) }}</td><td v-if="auth.isAdmin" class="px-4 py-3"><button v-if="editable(r)" class="text-primary hover:underline" @click="open(r)">Bearbeiten</button></td></tr><tr v-if="!rules.length"><td :colspan="auth.isAdmin ? 5 : 4" class="px-4 py-8 text-center text-muted-foreground">{{ loading ? 'Lade Regeln …' : 'Noch kein Regelwerk vorhanden.' }}</td></tr></tbody></table></div>
    <div v-if="dialog" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="dialog = false"><form role="dialog" aria-modal="true" :aria-label="editing ? 'Regelwerk bearbeiten' : 'Regelwerk anlegen'" class="w-full max-w-lg rounded-xl bg-card p-6 shadow-xl" @submit.prevent="save"><h2 class="text-xl font-semibold">{{ editing ? 'Regelwerk bearbeiten' : 'Neue Regelwerk-Version' }}</h2><div class="mt-4 space-y-4"><label class="block text-sm font-medium">Gültig ab (01.08.) *<input v-model="validFrom" type="date" required :min="nextAugust" class="mt-1 w-full rounded-lg border px-3 py-2" /></label><label class="block text-sm font-medium">Stunden je Kind *<input v-model.number="hoursPerChild" type="number" min="0" step="0.25" required class="mt-1 w-full rounded-lg border px-3 py-2" /></label><label class="block text-sm font-medium">Satz je Fehlstunde (€) *<input v-model.number="missingRate" type="number" min="0" step="0.01" required class="mt-1 w-full rounded-lg border px-3 py-2" /></label><label class="block text-sm font-medium">Maximaler Übertrag (Stunden) *<input v-model.number="maxCarry" type="number" min="0" step="0.25" required class="mt-1 w-full rounded-lg border px-3 py-2" /></label><p v-if="actionError" role="alert" class="text-sm text-red-700 dark:text-red-300">{{ actionError }}</p></div><div class="mt-6 flex justify-end gap-2"><button type="button" class="rounded-lg border px-4 py-2" @click="dialog = false">Abbrechen</button><button type="submit" :disabled="saving" class="rounded-lg bg-primary px-4 py-2 text-primary-foreground">Speichern</button></div></form></div>
  </section>
</template>
