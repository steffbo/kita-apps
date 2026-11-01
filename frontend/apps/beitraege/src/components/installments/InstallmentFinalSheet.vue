<script setup lang="ts">
import { computed, ref } from 'vue';
import { api } from '@/api';
import type { InstallmentDetail } from '@/api/types';
import { AlertTriangle, Loader2 } from 'lucide-vue-next';
import AppSheet from '@/components/ui/AppSheet.vue';
import DetailRow from '@/components/ui/DetailRow.vue';
import { formatCurrency, todayISO } from '@/utils/format';
import { addDays } from '@/utils/installments';

// Schlussrate: the whole rest falls due on one date; all unpaid regular rates are replaced by it.
const props = defineProps<{ detail: InstallmentDetail }>();
const emit = defineEmits<{ close: []; saved: [] }>();

const dueDate = ref(addDays(todayISO(), 14));
const error = ref<string | null>(null);
const isSaving = ref(false);
const openRates = computed(() => props.detail.schedule.filter(rate => rate.status !== 'PAID').length);

async function submit(): Promise<void> {
  isSaving.value = true;
  error.value = null;
  try {
    await api.setInstallmentFinalDueDate(props.detail.plan.id, { dueDate: dueDate.value });
    emit('saved');
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Schlussrate konnte nicht gesetzt werden';
  } finally {
    isSaving.value = false;
  }
}
</script>

<template>
  <AppSheet title="Schlussrate vereinbaren" @close="emit('close')">
    <form id="installment-final-form" class="space-y-4" @submit.prevent="submit">
      <p class="text-sm text-muted-foreground">
        Die gesamte Restschuld wird an einem Tag fällig – auch bereits überfällige Raten. Alle weiteren Raten entfallen.
      </p>
      <div>
        <label for="final-date" class="label">Schlussrate fällig am *</label>
        <input id="final-date" v-model="dueDate" required type="date" class="input" />
      </div>
      <dl class="card divide-y text-sm">
        <DetailRow label="Restschuld heute" value-class="font-medium tabular-nums">{{ formatCurrency(detail.summary.rest) }}</DetailRow>
        <DetailRow label="Schlussrate" value-class="font-semibold tabular-nums">{{ formatCurrency(detail.summary.rest) }}</DetailRow>
        <DetailRow label="Ersetzt" value-class="tabular-nums">{{ openRates }} offene {{ openRates === 1 ? 'Rate' : 'Raten' }}</DetailRow>
      </dl>
      <p class="text-xs text-muted-foreground">
        Wird die Schlussrate nicht bis zum Datum bezahlt, erscheint sie in den Erinnerungen.
      </p>
      <div v-if="error" class="note note-danger" role="alert"><AlertTriangle />{{ error }}</div>
    </form>
    <template #footer>
      <button type="button" class="btn btn-secondary" @click="emit('close')">Abbrechen</button>
      <button type="submit" form="installment-final-form" class="btn btn-primary" :disabled="isSaving">
        <Loader2 v-if="isSaving" class="animate-spin" />Festlegen
      </button>
    </template>
  </AppSheet>
</template>
