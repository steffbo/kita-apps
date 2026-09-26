<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { api } from '@/api';
import type { BoardTerm, ParentWorkHouseholdOption } from '@/api/types';
import { formatDate, formatDateForInput, todayISO } from '@/utils/format';

const terms = ref<BoardTerm[]>([]);
const households = ref<ParentWorkHouseholdOption[]>([]);
const error = ref('');
const actionError = ref('');
const loading = ref(false);
const saving = ref(false);
const dialog = ref(false);
const editing = ref<BoardTerm | null>(null);
const deleting = ref<BoardTerm | null>(null);
const memberSearch = ref('');
const memberId = ref('');
const office = ref('');
const startDate = ref(todayISO());
const endDate = ref('');
const note = ref('');
const members = computed(() => households.value.flatMap(h => h.members.map(m => ({ ...m, household: h.name }))));
const matches = computed(() => { const q = memberSearch.value.toLocaleLowerCase('de').trim(); return members.value.filter(m => !q || `${m.name} ${m.household}`.toLocaleLowerCase('de').includes(q)).slice(0, 40); });
const selectedMember = computed(() => members.value.find(m => m.id === memberId.value));
function active(term: BoardTerm) { const today = todayISO(); return formatDateForInput(term.startDate) <= today && (!term.endDate || formatDateForInput(term.endDate) >= today); }
async function load() {
  loading.value = true; error.value = '';
  try { const [t, h] = await Promise.all([api.getBoardTerms(), api.getParentWorkHouseholds()]); terms.value = t ?? []; households.value = h ?? []; }
  catch (e) { error.value = e instanceof Error ? e.message : 'Amtszeiten konnten nicht geladen werden'; }
  finally { loading.value = false; }
}
onMounted(load);
function open(term?: BoardTerm) { editing.value = term ?? null; memberId.value = term?.memberId ?? ''; memberSearch.value = ''; office.value = term?.office ?? ''; startDate.value = formatDateForInput(term?.startDate) || todayISO(); endDate.value = formatDateForInput(term?.endDate); note.value = term?.note ?? ''; actionError.value = ''; dialog.value = true; }
async function save() {
  actionError.value = ''; saving.value = true;
  try { const data = { memberId: memberId.value, office: office.value.trim(), startDate: startDate.value, endDate: endDate.value || undefined, note: note.value.trim() || undefined }; if (editing.value) await api.updateBoardTerm(editing.value.id, data); else await api.createBoardTerm(data); dialog.value = false; await load(); }
  catch (e) { actionError.value = e instanceof Error ? e.message : 'Speichern fehlgeschlagen'; }
  finally { saving.value = false; }
}
async function remove() { if (!deleting.value) return; saving.value = true; actionError.value = ''; try { await api.deleteBoardTerm(deleting.value.id); deleting.value = null; await load(); } catch (e) { actionError.value = e instanceof Error ? e.message : 'Löschen fehlgeschlagen'; } finally { saving.value = false; } }
</script>

<template>
  <div class="space-y-6"><div class="flex items-center justify-between gap-3"><div><h1 class="text-2xl font-bold">Vorstand</h1><p class="text-sm text-gray-500">Amtszeiten von Vereinsmitgliedern</p></div><button class="rounded-lg bg-primary px-4 py-2 text-white" @click="open()">Amtszeit anlegen</button></div>
    <p v-if="error" role="alert" class="rounded-lg bg-red-50 p-3 text-red-700">{{ error }}</p>
    <div class="overflow-x-auto rounded-xl border bg-white"><table class="w-full text-sm"><thead class="bg-gray-50 text-left text-gray-600"><tr><th class="px-4 py-3">Mitglied</th><th class="px-4 py-3">Familie</th><th class="px-4 py-3">Amt</th><th class="px-4 py-3">Von</th><th class="px-4 py-3">Bis</th><th class="px-4 py-3">Aktiv</th><th class="px-4 py-3">Aktionen</th></tr></thead><tbody><tr v-for="t in terms" :key="t.id" class="border-t"><td class="px-4 py-3">{{ t.memberName }}</td><td class="px-4 py-3">{{ t.householdName || '—' }}</td><td class="px-4 py-3">{{ t.office }}</td><td class="px-4 py-3">{{ formatDate(t.startDate) }}</td><td class="px-4 py-3">{{ formatDate(t.endDate) }}</td><td class="px-4 py-3">{{ active(t) ? 'Ja' : 'Nein' }}</td><td class="px-4 py-3"><button class="mr-3 text-primary hover:underline" @click="open(t)">Bearbeiten</button><button class="text-red-700 hover:underline" @click="deleting = t; actionError = ''">Löschen</button></td></tr><tr v-if="!terms.length"><td colspan="7" class="px-4 py-8 text-center text-gray-500">{{ loading ? 'Lade Amtszeiten …' : 'Keine Amtszeiten vorhanden.' }}</td></tr></tbody></table></div>
    <div v-if="dialog" class="fixed inset-0 z-50 overflow-y-auto bg-black/50 p-4" @click.self="dialog = false"><form role="dialog" aria-modal="true" :aria-label="editing ? 'Amtszeit bearbeiten' : 'Amtszeit anlegen'" class="mx-auto my-8 max-w-lg rounded-xl bg-white p-6 shadow-xl" @submit.prevent="save"><h2 class="text-xl font-semibold">{{ editing ? 'Amtszeit bearbeiten' : 'Amtszeit anlegen' }}</h2><div class="mt-4 space-y-4"><label class="block text-sm font-medium">Mitglied suchen<input v-model="memberSearch" type="search" placeholder="Name oder Familie" class="mt-1 w-full rounded-lg border px-3 py-2" /></label><label class="block text-sm font-medium">Vereinsmitglied *<select v-model="memberId" required class="mt-1 w-full rounded-lg border px-3 py-2"><option value="">Bitte auswählen</option><option v-for="m in matches" :key="m.id" :value="m.id">{{ m.name }} · {{ m.household }}</option><option v-if="selectedMember && !matches.some(m => m.id === memberId)" :value="selectedMember.id">{{ selectedMember.name }} · {{ selectedMember.household }}</option></select></label><label class="block text-sm font-medium">Amt *<input v-model="office" required class="mt-1 w-full rounded-lg border px-3 py-2" /></label><div class="grid gap-4 sm:grid-cols-2"><label class="block text-sm font-medium">Von *<input v-model="startDate" type="date" required class="mt-1 w-full rounded-lg border px-3 py-2" /></label><label class="block text-sm font-medium">Bis<input v-model="endDate" type="date" :min="startDate" class="mt-1 w-full rounded-lg border px-3 py-2" /></label></div><label class="block text-sm font-medium">Notiz<textarea v-model="note" class="mt-1 w-full rounded-lg border px-3 py-2" /></label><p v-if="actionError" role="alert" class="text-sm text-red-700">{{ actionError }}</p></div><div class="mt-6 flex justify-end gap-2"><button type="button" class="rounded-lg border px-4 py-2" @click="dialog = false">Abbrechen</button><button type="submit" :disabled="saving" class="rounded-lg bg-primary px-4 py-2 text-white">Speichern</button></div></form></div>
    <div v-if="deleting" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="deleting = null"><div role="dialog" aria-modal="true" aria-label="Amtszeit löschen" class="w-full max-w-md rounded-xl bg-white p-6 shadow-xl"><h2 class="text-xl font-semibold">Amtszeit löschen?</h2><p class="mt-3 text-sm">{{ deleting.memberName }} · {{ deleting.office }}</p><p v-if="actionError" role="alert" class="mt-3 text-sm text-red-700">{{ actionError }}</p><div class="mt-6 flex justify-end gap-2"><button class="rounded-lg border px-4 py-2" @click="deleting = null">Abbrechen</button><button :disabled="saving" class="rounded-lg bg-red-700 px-4 py-2 text-white" @click="remove">Löschen</button></div></div></div>
  </div>
</template>
