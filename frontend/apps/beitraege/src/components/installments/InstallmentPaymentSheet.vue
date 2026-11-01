<script setup lang="ts">
import { reactive, ref } from 'vue';
import { api } from '@/api';
import type { InstallmentDetail } from '@/api/types';
import { AlertTriangle, Loader2 } from 'lucide-vue-next';
import AppSheet from '@/components/ui/AppSheet.vue';
import { todayISO } from '@/utils/format';

// Manual payment (cash or other); bank transfers are assigned in the Bankabgleich.
const props = defineProps<{ detail: InstallmentDetail }>();
const emit = defineEmits<{ close: []; saved: [] }>();

const form = reactive({
  amount: props.detail.plan.installmentAmount as number | '',
  paidOn: todayISO(),
  note: '',
});
const error = ref<string | null>(null);
const isSaving = ref(false);

async function submit(): Promise<void> {
  if (form.amount === '') return;
  isSaving.value = true;
  error.value = null;
  try {
    await api.addInstallmentPayment(props.detail.plan.id, {
      amount: form.amount,
      paidOn: form.paidOn,
      note: form.note.trim(),
    });
    emit('saved');
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Zahlung konnte nicht gespeichert werden';
  } finally {
    isSaving.value = false;
  }
}
</script>

<template>
  <AppSheet title="Zahlung erfassen" @close="emit('close')">
    <form id="installment-payment-form" class="space-y-4" @submit.prevent="submit">
      <p class="text-sm text-muted-foreground">
        Für Bar- oder sonstige Zahlungen ohne Banktransaktion. Überweisungen kommen über den Bankabgleich.
      </p>
      <div class="grid grid-cols-2 gap-3">
        <div>
          <label for="payment-amount" class="label">Betrag *</label>
          <input id="payment-amount" v-model.number="form.amount" required type="number" min="0.01" step="0.01"
            inputmode="decimal" class="input" />
        </div>
        <div>
          <label for="payment-date" class="label">Datum *</label>
          <input id="payment-date" v-model="form.paidOn" required type="date" class="input" />
        </div>
      </div>
      <div>
        <label for="payment-note" class="label">Notiz</label>
        <input id="payment-note" v-model="form.note" type="text" class="input" placeholder="z. B. Barzahlung im Büro" />
      </div>
      <div v-if="error" class="note note-danger" role="alert"><AlertTriangle />{{ error }}</div>
    </form>
    <template #footer>
      <button type="button" class="btn btn-secondary" @click="emit('close')">Abbrechen</button>
      <button type="submit" form="installment-payment-form" class="btn btn-primary" :disabled="isSaving">
        <Loader2 v-if="isSaving" class="animate-spin" />Speichern
      </button>
    </template>
  </AppSheet>
</template>
