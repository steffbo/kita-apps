<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { api } from '@/api';
import type { ParentWorkEntry, ParentWorkHouseholdOption, ParentWorkEntryRequest } from '@/api/types';
import { formatDateForInput, todayISO } from '@/utils/format';
import HouseholdPicker from '@/components/HouseholdPicker.vue';
import DurationPicker from './DurationPicker.vue';
import AppSheet from '@/components/ui/AppSheet.vue';

const props = defineProps<{ entry?: ParentWorkEntry | null; householdId?: string }>();
const emit = defineEmits<{ close: []; saved: [] }>();
const households = ref<ParentWorkHouseholdOption[]>([]);
const householdId = ref(props.entry?.householdId ?? props.householdId ?? '');
const date = ref(formatDateForInput(props.entry?.workDate) || todayISO());
const minutes = ref<number | null>(props.entry?.durationMinutes ?? null);
const occasion = ref(props.entry?.occasion ?? '');
const error = ref('');
const loading = ref(false);
const saving = ref(false);
// Once a family is set (preselected or picked) it stays fixed; changing it means closing the dialog.
const householdName = computed(() => households.value.find(h => h.id === householdId.value)?.name
  ?? (loading.value ? '…' : 'Unbekannte Familie'));

onMounted(async () => {
  loading.value = true;
  try { households.value = (await api.getParentWorkHouseholds()) ?? []; }
  catch (e) { error.value = e instanceof Error ? e.message : 'Familien konnten nicht geladen werden'; }
  finally { loading.value = false; }
});

async function save() {
  error.value = '';
  if (!minutes.value) { error.value = 'Bitte die Dauer wählen.'; return; }
  if (!householdId.value) { error.value = 'Bitte eine Familie auswählen.'; return; }
  saving.value = true;
  try {
    const data: ParentWorkEntryRequest = { householdId: householdId.value, workDate: date.value,
      durationMinutes: minutes.value, occasion: occasion.value.trim(),
      memberName: props.entry?.memberName ?? undefined, childName: props.entry?.childName ?? undefined };
    if (props.entry) await api.updateParentWorkEntry(props.entry.id, data);
    else await api.createParentWorkEntry(data);
    emit('saved');
  } catch (e) { error.value = e instanceof Error ? e.message : 'Speichern fehlgeschlagen'; }
  finally { saving.value = false; }
}
</script>

<template>
  <AppSheet :title="entry ? 'Stunden bearbeiten' : 'Stunden erfassen'" size="md" @close="emit('close')">
    <form id="parent-work-entry" class="space-y-4" @submit.prevent="save">
        <div v-if="householdId">
          <p class="label">Familie</p>
          <p class="flex h-10 items-center rounded-lg border border-input bg-muted px-3 text-sm font-medium">
            {{ householdName }}</p>
        </div>
        <template v-else>
          <HouseholdPicker v-model="householdId" :households="households" label="Familie *" />
          <p v-if="loading" class="text-sm text-muted-foreground">Familien werden geladen …</p>
        </template>
        <label class="label">Datum *<input v-model="date" type="date" required class="input block sm:w-56" /></label>
        <DurationPicker v-model="minutes" label="Dauer *" />
        <label class="label">Anlass *<input v-model="occasion" type="text" required class="input" /></label>
        <p v-if="error" role="alert" class="note note-danger">{{ error }}</p>
    </form>
    <template #footer>
      <button type="button" class="btn btn-secondary" @click="emit('close')">Abbrechen</button>
      <button type="submit" form="parent-work-entry" :disabled="saving || loading" class="btn btn-primary">Speichern</button>
    </template>
  </AppSheet>
</template>
