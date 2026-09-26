<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { api } from '@/api';
import type { ParentWorkEntry, ParentWorkHouseholdOption, ParentWorkEntryRequest } from '@/api/types';
import { formatDateForInput, todayISO } from '@/utils/format';

const props = defineProps<{ entry?: ParentWorkEntry | null; householdId?: string }>();
const emit = defineEmits<{ close: []; saved: [] }>();
const households = ref<ParentWorkHouseholdOption[]>([]);
const search = ref('');
const householdId = ref(props.entry?.householdId ?? props.householdId ?? '');
const date = ref(formatDateForInput(props.entry?.workDate) || todayISO());
const hours = ref(props.entry ? props.entry.durationMinutes / 60 : 0.25);
const occasion = ref(props.entry?.occasion ?? '');
const memberName = ref(props.entry?.memberName ?? '');
const childName = ref(props.entry?.childName ?? '');
const status = ref<ParentWorkEntryRequest['status']>(props.entry?.status === 'SUBMITTED' || props.entry?.status === 'REJECTED' ? props.entry.status : 'APPROVED');
const error = ref('');
const loading = ref(false);
const saving = ref(false);
const selected = computed(() => households.value.find(h => h.id === householdId.value));
const matches = computed(() => {
  const q = search.value.toLocaleLowerCase('de').trim();
  return households.value.filter(h => !q || [h.name, ...h.children.map(c => c.name), ...h.members.map(m => m.name), ...h.parents.map(p => p.name)].some(n => n.toLocaleLowerCase('de').includes(q))).slice(0, 30);
});

onMounted(async () => {
  loading.value = true;
  try { households.value = (await api.getParentWorkHouseholds()) ?? []; }
  catch (e) { error.value = e instanceof Error ? e.message : 'Familien konnten nicht geladen werden'; }
  finally { loading.value = false; }
});
watch(householdId, (id, oldId) => { if (oldId && id !== oldId) { memberName.value = ''; childName.value = ''; } });

async function save() {
  error.value = '';
  const minutes = Math.round(hours.value * 60);
  if (!Number.isFinite(hours.value) || hours.value <= 0 || minutes % 15 !== 0 || Math.abs(minutes / 60 - hours.value) > 0.00001) {
    error.value = 'Bitte Stunden in Schritten von 0,25 eingeben.'; return;
  }
  if (!householdId.value) { error.value = 'Bitte eine Familie auswählen.'; return; }
  saving.value = true;
  try {
    const data: ParentWorkEntryRequest = { householdId: householdId.value, workDate: date.value, durationMinutes: minutes, occasion: occasion.value.trim(), memberName: memberName.value.trim() || undefined, childName: childName.value.trim() || undefined, status: status.value };
    if (props.entry) await api.updateParentWorkEntry(props.entry.id, data);
    else await api.createParentWorkEntry(data);
    emit('saved');
  } catch (e) { error.value = e instanceof Error ? e.message : 'Speichern fehlgeschlagen'; }
  finally { saving.value = false; }
}
</script>

<template>
  <div class="fixed inset-0 z-50 overflow-y-auto bg-black/50 p-4" @click.self="emit('close')">
    <form role="dialog" aria-modal="true" :aria-label="entry ? 'Stunden bearbeiten' : 'Stunden erfassen'" class="mx-auto my-8 max-w-xl rounded-xl bg-white p-6 shadow-xl" @submit.prevent="save">
      <h2 class="text-xl font-semibold">{{ entry ? 'Stunden bearbeiten' : 'Stunden erfassen' }}</h2>
      <div class="mt-5 space-y-4">
        <label class="block text-sm font-medium">Familie *
          <input v-model="search" type="search" placeholder="Familie, Kind oder Mitglied suchen" class="mt-1 w-full rounded-lg border px-3 py-2" />
        </label>
        <p v-if="selected" class="text-sm text-gray-600">Ausgewählt: {{ selected.name }}</p>
        <label class="block text-sm font-medium">Familie auswählen *
          <select v-model="householdId" required class="mt-1 w-full rounded-lg border px-3 py-2">
            <option value="">Bitte auswählen</option>
            <option v-for="h in matches" :key="h.id" :value="h.id">{{ h.name }}</option>
            <option v-if="selected && !matches.some(h => h.id === householdId)" :value="selected.id">{{ selected.name }}</option>
          </select>
        </label>
        <p v-if="loading" class="text-sm text-gray-500">Familien werden geladen …</p>
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="block text-sm font-medium">Datum *<input v-model="date" type="date" required class="mt-1 w-full rounded-lg border px-3 py-2" /></label>
          <label class="block text-sm font-medium">Stunden *<input v-model.number="hours" type="number" min="0.25" step="0.25" required class="mt-1 w-full rounded-lg border px-3 py-2" /></label>
        </div>
        <label class="block text-sm font-medium">Anlass *<input v-model="occasion" type="text" required class="mt-1 w-full rounded-lg border px-3 py-2" /></label>
        <label class="block text-sm font-medium">Mitglied
          <input v-model="memberName" list="parent-work-members" type="text" class="mt-1 w-full rounded-lg border px-3 py-2" />
          <datalist id="parent-work-members"><option v-for="m in selected?.members ?? []" :key="m.id" :value="m.name" /></datalist>
        </label>
        <label class="block text-sm font-medium">Kind
          <input v-model="childName" list="parent-work-children" type="text" class="mt-1 w-full rounded-lg border px-3 py-2" />
          <datalist id="parent-work-children"><option v-for="c in selected?.children ?? []" :key="c.id" :value="c.name" /></datalist>
        </label>
        <label class="block text-sm font-medium">Status
          <select v-model="status" class="mt-1 w-full rounded-lg border px-3 py-2"><option value="APPROVED">Bestätigt</option><option value="SUBMITTED">Eingereicht</option><option value="REJECTED">Abgelehnt</option></select>
        </label>
        <p v-if="error" role="alert" class="rounded-lg bg-red-50 p-3 text-sm text-red-700">{{ error }}</p>
      </div>
      <div class="mt-6 flex justify-end gap-3"><button type="button" class="rounded-lg border px-4 py-2" @click="emit('close')">Abbrechen</button><button type="submit" :disabled="saving || loading" class="rounded-lg bg-primary px-4 py-2 text-white disabled:opacity-50">Speichern</button></div>
    </form>
  </div>
</template>
