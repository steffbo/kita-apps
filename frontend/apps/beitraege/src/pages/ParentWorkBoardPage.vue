<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { api } from '@/api';
import type { BoardTerm, ParentWorkHouseholdOption } from '@/api/types';
import { formatDate, formatDateForInput, todayISO } from '@/utils/format';
import SearchInput from '@/components/SearchInput.vue';
import SortTh from '@/components/SortTh.vue';
import { useTableSort } from '@/composables/useTableSort';
import PageHeader from '@/components/ui/PageHeader.vue';
import AppSheet from '@/components/ui/AppSheet.vue';

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
const search = ref('');
const activeFilter = ref<'' | 'active' | 'inactive'>('');
const filteredTerms = computed(() => {
  const q = search.value.toLocaleLowerCase('de').trim();
  return terms.value.filter(t => (!activeFilter.value || active(t) === (activeFilter.value === 'active'))
    && (!q || [t.memberName, t.householdName ?? '', t.office].some(v => v.toLocaleLowerCase('de').includes(q))));
});
const { sortKey, sortDir, sorted, toggle } = useTableSort(filteredTerms, {
  member: t => t.memberName,
  household: t => t.householdName,
  office: t => t.office,
  startDate: t => formatDateForInput(t.startDate),
  endDate: t => formatDateForInput(t.endDate),
  active: t => (active(t) ? 'Ja' : 'Nein'),
}, { key: 'startDate', dir: 'desc' });
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
  <div class="space-y-6"><PageHeader title="Vorstand"><template #actions><button class="btn btn-primary" @click="open()">Amtszeit anlegen</button></template></PageHeader>
    <p v-if="error" role="alert" class="note note-danger">{{ error }}</p>
    <div class="flex flex-col gap-3 sm:flex-row">
      <SearchInput v-model="search" placeholder="Suchen nach Mitglied, Familie oder Amt..." label="Amtszeiten durchsuchen" class="flex-1" />
      <select v-model="activeFilter" aria-label="Aktiv" class="input sm:w-56">
        <option value="">Alle Amtszeiten</option>
        <option value="active">Nur aktive</option>
        <option value="inactive">Nur nicht aktive</option>
      </select>
    </div>
    <div class="card overflow-hidden"><ul class="divide-y sm:hidden">
      <li v-for="t in sorted" :key="t.id" class="space-y-2 p-4">
        <div class="flex items-start justify-between gap-3"><div><p class="font-medium">{{ t.memberName }}</p><p class="text-[13px] text-muted-foreground">{{ t.householdName || '—' }} · {{ t.office }}</p></div><span class="badge" :class="active(t) ? 'badge-success' : 'badge'">{{ active(t) ? 'Ja' : 'Nein' }}</span></div>
        <p class="text-[13px] text-muted-foreground">{{ formatDate(t.startDate) }} – {{ formatDate(t.endDate) }}</p>
        <div class="flex gap-2"><button class="btn btn-secondary btn-sm" @click="open(t)">Bearbeiten</button><button class="btn btn-danger-outline btn-sm" @click="deleting = t; actionError = ''">Löschen</button></div>
      </li>
      <li v-if="!sorted.length" class="p-6 text-center text-sm text-muted-foreground">{{ loading ? 'Lade Amtszeiten …' : terms.length ? 'Keine Amtszeiten gefunden.' : 'Keine Amtszeiten vorhanden.' }}</li>
    </ul><div class="hidden overflow-x-auto sm:block"><table class="table w-full text-sm"><thead class="bg-muted text-left text-muted-foreground"><tr>
      <SortTh label="Mitglied" column="member" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
      <SortTh label="Familie" column="household" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
      <SortTh label="Amt" column="office" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
      <SortTh label="Von" column="startDate" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
      <SortTh label="Bis" column="endDate" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
      <SortTh label="Aktiv" column="active" :sort-key="sortKey" :sort-dir="sortDir" @sort="toggle" />
      <th class="px-4 py-3">Aktionen</th></tr></thead><tbody><tr v-for="t in sorted" :key="t.id" class="border-t"><td class="px-4 py-3">{{ t.memberName }}</td><td class="px-4 py-3">{{ t.householdName || '—' }}</td><td class="px-4 py-3">{{ t.office }}</td><td class="px-4 py-3">{{ formatDate(t.startDate) }}</td><td class="px-4 py-3">{{ formatDate(t.endDate) }}</td><td class="px-4 py-3">{{ active(t) ? 'Ja' : 'Nein' }}</td><td class="px-4 py-3"><div class="flex gap-2"><button class="btn btn-secondary btn-sm" @click="open(t)">Bearbeiten</button><button class="btn btn-danger-outline btn-sm" @click="deleting = t; actionError = ''">Löschen</button></div></td></tr><tr v-if="!sorted.length"><td colspan="7" class="px-4 py-8 text-center text-muted-foreground">{{ loading ? 'Lade Amtszeiten …' : terms.length ? 'Keine Amtszeiten gefunden.' : 'Keine Amtszeiten vorhanden.' }}</td></tr></tbody></table></div></div>
    <AppSheet v-if="dialog" :title="editing ? 'Amtszeit bearbeiten' : 'Amtszeit anlegen'" @close="dialog = false">
      <form id="board-term" class="space-y-4" @submit.prevent="save">
        <label class="label">Mitglied suchen<SearchInput v-model="memberSearch" placeholder="Name oder Familie" class="mt-1 font-normal" /></label>
        <label class="label">Vereinsmitglied *<select v-model="memberId" required class="input"><option value="">Bitte auswählen</option><option v-for="m in matches" :key="m.id" :value="m.id">{{ m.name }} · {{ m.household }}</option><option v-if="selectedMember && !matches.some(m => m.id === memberId)" :value="selectedMember.id">{{ selectedMember.name }} · {{ selectedMember.household }}</option></select></label>
        <label class="label">Amt *<input v-model="office" required class="input" /></label>
        <div class="grid gap-4 sm:grid-cols-2"><label class="label">Von *<input v-model="startDate" type="date" required class="input" /></label><label class="label">Bis<input v-model="endDate" type="date" :min="startDate" class="input" /></label></div>
        <label class="label">Notiz<textarea v-model="note" class="input" /></label>
        <p v-if="actionError" role="alert" class="note note-danger">{{ actionError }}</p>
      </form>
      <template #footer><button type="button" class="btn btn-secondary" @click="dialog = false">Abbrechen</button><button type="submit" form="board-term" :disabled="saving" class="btn btn-primary">Speichern</button></template>
    </AppSheet>
    <AppSheet v-if="deleting" title="Amtszeit löschen" size="sm" @close="deleting = null">
      <h2 class="text-base font-semibold">Amtszeit löschen?</h2><p class="mt-3 text-sm">{{ deleting.memberName }} · {{ deleting.office }}</p>
      <p v-if="actionError" role="alert" class="mt-3 note note-danger">{{ actionError }}</p>
      <template #footer><button class="btn btn-secondary" @click="deleting = null">Abbrechen</button><button :disabled="saving" class="btn btn-danger" @click="remove">Löschen</button></template>
    </AppSheet>
  </div>
</template>
