<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { api } from '@/api';
import type { ParentWorkEntry, ParentWorkHouseholdOption, ParentWorkEntryRequest } from '@/api/types';
import { formatDateForInput, todayISO } from '@/utils/format';
import HouseholdPicker from './HouseholdPicker.vue';

const props = defineProps<{ entry?: ParentWorkEntry | null; householdId?: string }>();
const emit = defineEmits<{ close: []; saved: [] }>();
const households = ref<ParentWorkHouseholdOption[]>([]);
const householdId = ref(props.entry?.householdId ?? props.householdId ?? '');
const date = ref(formatDateForInput(props.entry?.workDate) || todayISO());
const hours = ref<number | null>(props.entry ? props.entry.durationMinutes / 60 : null);
const occasion = ref(props.entry?.occasion ?? '');
const error = ref('');
const loading = ref(false);
const saving = ref(false);

onMounted(async () => {
  loading.value = true;
  try { households.value = (await api.getParentWorkHouseholds()) ?? []; }
  catch (e) { error.value = e instanceof Error ? e.message : 'Familien konnten nicht geladen werden'; }
  finally { loading.value = false; }
});

async function save() {
  error.value = '';
  const value = hours.value ?? 0;
  const minutes = Math.round(value * 60);
  if (!Number.isFinite(value) || value <= 0 || minutes % 15 !== 0 || Math.abs(minutes / 60 - value) > 0.00001) {
    error.value = 'Bitte Stunden in Schritten von 0,25 eingeben.'; return;
  }
  if (!householdId.value) { error.value = 'Bitte eine Familie auswählen.'; return; }
  saving.value = true;
  try {
    const data: ParentWorkEntryRequest = { householdId: householdId.value, workDate: date.value,
      durationMinutes: minutes, occasion: occasion.value.trim(),
      memberName: props.entry?.memberName ?? undefined, childName: props.entry?.childName ?? undefined };
    if (props.entry) await api.updateParentWorkEntry(props.entry.id, data);
    else await api.createParentWorkEntry(data);
    emit('saved');
  } catch (e) { error.value = e instanceof Error ? e.message : 'Speichern fehlgeschlagen'; }
  finally { saving.value = false; }
}
</script>

<template>
  <div class="fixed inset-0 z-50 overflow-y-auto bg-black/50 p-4" @click.self="emit('close')">
    <form role="dialog" aria-modal="true" :aria-label="entry ? 'Stunden bearbeiten' : 'Stunden erfassen'" class="mx-auto my-8 max-w-xl rounded-xl bg-card p-6 shadow-xl" @submit.prevent="save">
      <h2 class="text-xl font-semibold">{{ entry ? 'Stunden bearbeiten' : 'Stunden erfassen' }}</h2>
      <div class="mt-5 space-y-4">
        <HouseholdPicker v-model="householdId" :households="households" label="Familie *" />
        <p v-if="loading" class="text-sm text-muted-foreground">Familien werden geladen …</p>
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="block text-sm font-medium">Datum *<input v-model="date" type="date" required class="mt-1 w-full rounded-lg border px-3 py-2" /></label>
          <label class="block text-sm font-medium">Stunden *<input v-model.number="hours" type="number" min="0.25" step="0.25" required class="mt-1 w-full rounded-lg border px-3 py-2" /></label>
        </div>
        <label class="block text-sm font-medium">Anlass *<input v-model="occasion" type="text" required class="mt-1 w-full rounded-lg border px-3 py-2" /></label>
        <p v-if="error" role="alert" class="rounded-lg bg-red-50 dark:bg-red-950/40 p-3 text-sm text-red-700 dark:text-red-300">{{ error }}</p>
      </div>
      <div class="mt-6 flex justify-end gap-3"><button type="button" class="rounded-lg border px-4 py-2" @click="emit('close')">Abbrechen</button><button type="submit" :disabled="saving || loading" class="rounded-lg bg-primary px-4 py-2 text-primary-foreground disabled:opacity-50">Speichern</button></div>
    </form>
  </div>
</template>
