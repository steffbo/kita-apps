<script setup lang="ts">
import { computed, ref } from 'vue';
import { api } from '@/api';
import type { BankTransaction } from '@/api/types';
import { AlertTriangle, Link2, Loader2, X } from 'lucide-vue-next';
import HouseholdPicker from '@/components/HouseholdPicker.vue';
import { formatCurrency } from '@/utils/format';

// Marks an open incoming transaction as donation, or edits note and household of an existing one.
// A household known from the payer IBAN is preselected; undo via „Zuordnung aufheben“.
const props = defineProps<{ transaction: BankTransaction }>();
const emit = defineEmits<{ matched: [] }>();

const existing = computed(() => props.transaction.assignment ?? null);
const suggestion = computed(() => (existing.value ? null : props.transaction.householdSuggestion ?? null));

const note = ref(existing.value?.note ?? '');
const householdId = ref<string | null>(existing.value?.householdId ?? suggestion.value?.householdId ?? null);
const isSaving = ref(false);
const error = ref<string | null>(null);

async function save(): Promise<void> {
  isSaving.value = true;
  error.value = null;
  const data = { note: note.value, householdId: householdId.value ?? undefined };
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
  <form class="space-y-4" @submit.prevent="save">
    <div>
      <label for="donation-note" class="label">Notiz (optional)</label>
      <input id="donation-note" v-model="note" class="input" maxlength="500" placeholder="z. B. Sommerfest" />
    </div>

    <div class="space-y-2">
      <div class="flex items-center justify-between">
        <span class="label mb-0">Haushalt (optional)</span>
        <button v-if="householdId" type="button" class="btn btn-ghost btn-sm" @click="householdId = null">
          <X />Entfernen
        </button>
      </div>
      <div v-if="suggestion" class="note note-success">
        <Link2 /><span>Vorschlag über IBAN: {{ suggestion.name }}</span>
      </div>
      <HouseholdPicker v-model="householdId" @error="error = $event" />
    </div>

    <div v-if="error" class="note note-danger" role="alert"><AlertTriangle />{{ error }}</div>

    <div class="flex justify-end">
      <button type="submit" class="btn btn-primary" :disabled="isSaving">
        <Loader2 v-if="isSaving" class="animate-spin" />
        <template v-if="existing">Speichern</template>
        <template v-else>Als Spende zuordnen · {{ formatCurrency(transaction.amount) }}</template>
      </button>
    </div>
  </form>
</template>
