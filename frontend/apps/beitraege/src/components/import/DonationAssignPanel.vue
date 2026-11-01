<script setup lang="ts">
import { ref } from 'vue';
import { api } from '@/api';
import type { BankTransaction } from '@/api/types';
import { AlertTriangle, Loader2 } from 'lucide-vue-next';
import { formatCurrency } from '@/utils/format';

// Marks an open incoming transaction as donation; undo via „Zuordnung aufheben“.
const props = defineProps<{ transaction: BankTransaction }>();
const emit = defineEmits<{ matched: [] }>();

const note = ref('');
const isAssigning = ref(false);
const error = ref<string | null>(null);

async function assign(): Promise<void> {
  isAssigning.value = true;
  error.value = null;
  try {
    await api.assignTransactionAsDonation(props.transaction.id, note.value);
    emit('matched');
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Zuordnung fehlgeschlagen';
  } finally {
    isAssigning.value = false;
  }
}
</script>

<template>
  <form class="space-y-4" @submit.prevent="assign">
    <div>
      <label for="donation-note" class="label">Notiz (optional)</label>
      <input id="donation-note" v-model="note" class="input" maxlength="500" placeholder="z. B. Sommerfest" />
    </div>

    <div v-if="error" class="note note-danger" role="alert"><AlertTriangle />{{ error }}</div>

    <div class="flex justify-end">
      <button type="submit" class="btn btn-primary" :disabled="isAssigning">
        <Loader2 v-if="isAssigning" class="animate-spin" />Als Spende zuordnen · {{ formatCurrency(transaction.amount) }}
      </button>
    </div>
  </form>
</template>
