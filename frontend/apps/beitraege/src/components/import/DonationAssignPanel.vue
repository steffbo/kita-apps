<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { api } from '@/api';
import type { BankTransaction, ParentWorkHouseholdOption } from '@/api/types';
import { AlertTriangle, Link2, Loader2, X } from 'lucide-vue-next';
import HouseholdPicker from '@/components/HouseholdPicker.vue';
import { formatCurrency } from '@/utils/format';

// Marks an open incoming transaction as donation, or edits note and household of an existing one.
// A household known from the payer IBAN is preselected; undo via „Zuordnung aufheben“.
// With `formId` the surrounding sheet renders the submit button in its footer.
// `defaultHouseholdId` preselects the household the panel was opened from, ahead of the IBAN suggestion.
const props = defineProps<{ transaction: BankTransaction; formId?: string; defaultHouseholdId?: string }>();
const emit = defineEmits<{ matched: [] }>();

const existing = computed(() => props.transaction.assignment ?? null);
const suggestion = computed(() => (existing.value ? null : props.transaction.householdSuggestion ?? null));

const note = ref(existing.value?.note ?? '');
const householdId = ref<string | undefined>(existing.value?.householdId ?? props.defaultHouseholdId ?? suggestion.value?.householdId);
const households = ref<ParentWorkHouseholdOption[]>([]);
const isSaving = ref(false);
const error = ref<string | null>(null);

onMounted(async () => {
  try {
    households.value = await api.getHouseholdOptions();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Haushalte konnten nicht geladen werden';
  }
});

async function save(): Promise<void> {
  if (isSaving.value) return;
  isSaving.value = true;
  error.value = null;
  const data = { note: note.value, householdId: householdId.value };
  try {
    if (existing.value) await api.updateDonation(props.transaction.id, data);
    else await api.assignTransactionAsDonation(props.transaction.id, data);
    emit('matched');
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Zuordnung fehlgeschlagen';
  } finally {
    isSaving.value = false;
  }
}
</script>

<template>
  <form :id="formId" class="space-y-4" @submit.prevent="save">
    <div>
      <label for="donation-note" class="label">Notiz (optional)</label>
      <input id="donation-note" v-model="note" class="input" maxlength="500" placeholder="z. B. Sommerfest" />
    </div>

    <div class="space-y-2">
      <HouseholdPicker v-model="householdId" :households="households" label="Haushalt (optional)" optional />
      <div v-if="suggestion && suggestion.householdId === householdId" class="note note-success">
        <Link2 /><span>Vorschlag über IBAN</span>
      </div>
      <button v-if="householdId" type="button" class="btn btn-ghost btn-sm" @click="householdId = undefined">
        <X />Haushalt entfernen
      </button>
    </div>

    <div v-if="error" class="note note-danger" role="alert"><AlertTriangle />{{ error }}</div>

    <div v-if="!formId" class="flex justify-end">
      <button type="submit" class="btn btn-primary" :disabled="isSaving">
        <Loader2 v-if="isSaving" class="animate-spin" />Als Spende zuordnen · {{ formatCurrency(transaction.amount) }}
      </button>
    </div>
  </form>
</template>
