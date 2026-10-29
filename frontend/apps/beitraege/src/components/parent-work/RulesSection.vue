<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { api } from '@/api';
import type { ParentWorkRule } from '@/api/types';
import { useAuthStore } from '@/stores/auth';
import { formatCurrency, formatDate, formatDateForInput, formatHours, todayISO } from '@/utils/format';
import AppSheet from '@/components/ui/AppSheet.vue';

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
  <section class="space-y-6"><div class="flex items-center justify-between gap-3"><div><h2 class="text-xl font-semibold">Regeln</h2><p class="text-sm text-muted-foreground">Versionen des Regelwerks</p></div><button v-if="auth.isAdmin" class="btn btn-primary" @click="open()">Neue Version</button></div>
    <div class="card p-5 text-sm">
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
    <p v-if="error" role="alert" class="note note-danger">{{ error }}</p>
    <div class="card overflow-x-auto"><table class="table"><thead class="bg-muted text-left text-muted-foreground"><tr><th>Gültig ab</th><th>Stunden je Kind</th><th>Satz je Fehlstunde</th><th>Max. Übertrag</th><th v-if="auth.isAdmin">Aktion</th></tr></thead><tbody><tr v-for="r in rules" :key="r.id"><td>{{ formatDate(r.validFrom) }}</td><td>{{ formatHours(r.hoursPerChildMinutes) }}</td><td>{{ formatCurrency(r.missingHourRateCents / 100) }}</td><td>{{ formatHours(r.maxCarryOverMinutes) }}</td><td v-if="auth.isAdmin"><button v-if="editable(r)" class="btn btn-secondary btn-sm" @click="open(r)">Bearbeiten</button></td></tr><tr v-if="!rules.length"><td :colspan="auth.isAdmin ? 5 : 4" class="py-8 text-center text-muted-foreground">{{ loading ? 'Lade Regeln …' : 'Noch kein Regelwerk vorhanden.' }}</td></tr></tbody></table></div>
    <AppSheet v-if="dialog" :title="editing ? 'Regelwerk bearbeiten' : 'Regelwerk anlegen'" @close="dialog = false">
      <form id="parent-work-rules" class="space-y-4" @submit.prevent="save">
        <label class="label">Gültig ab (01.08.) *<input v-model="validFrom" type="date" required :min="nextAugust" class="input" /></label>
        <label class="label">Stunden je Kind *<input v-model.number="hoursPerChild" type="number" min="0" step="0.25" required class="input" /></label>
        <label class="label">Satz je Fehlstunde (€) *<input v-model.number="missingRate" type="number" min="0" step="0.01" required class="input" /></label>
        <label class="label">Maximaler Übertrag (Stunden) *<input v-model.number="maxCarry" type="number" min="0" step="0.25" required class="input" /></label>
        <p v-if="actionError" role="alert" class="note note-danger">{{ actionError }}</p>
      </form>
      <template #footer><button type="button" class="btn btn-secondary" @click="dialog = false">Abbrechen</button><button type="submit" form="parent-work-rules" :disabled="saving" class="btn btn-primary">Speichern</button></template>
    </AppSheet>
  </section>
</template>
