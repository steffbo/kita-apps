<script setup lang="ts">
import { ref } from 'vue';
import { api } from '@/api';
import type { InstallmentDetail } from '@/api/types';
import { AlertTriangle, Loader2 } from 'lucide-vue-next';
import AppSheet from '@/components/ui/AppSheet.vue';
import { formatCurrency } from '@/utils/format';

const props = defineProps<{ detail: InstallmentDetail }>();
const emit = defineEmits<{ close: []; saved: [] }>();

const REASONS = ['Restschuld erlassen', 'An Inkasso/Gericht übergeben', 'Irrtümlich angelegt'];

const reason = ref(REASONS[0]);
const error = ref<string | null>(null);
const isSaving = ref(false);

async function submit(): Promise<void> {
  isSaving.value = true;
  error.value = null;
  try {
    await api.closeInstallmentPlan(props.detail.plan.id, { reason: reason.value });
    emit('saved');
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ratenzahlung konnte nicht beendet werden';
  } finally {
    isSaving.value = false;
  }
}
</script>

<template>
  <AppSheet title="Ratenzahlung beenden?" size="sm" @close="emit('close')">
    <form id="installment-close-form" class="space-y-4" @submit.prevent="submit">
      <p class="text-sm">
        Die Ratenzahlung wird geschlossen und taucht nicht mehr in den Erinnerungen auf. Offene Restschuld:
        {{ formatCurrency(detail.summary.rest) }}.
      </p>
      <div>
        <label for="close-reason" class="label">Grund</label>
        <select id="close-reason" v-model="reason" class="input">
          <option v-for="option in REASONS" :key="option" :value="option">{{ option }}</option>
        </select>
      </div>
      <div v-if="error" class="note note-danger" role="alert"><AlertTriangle />{{ error }}</div>
    </form>
    <template #footer>
      <button type="button" class="btn btn-secondary" @click="emit('close')">Abbrechen</button>
      <button type="submit" form="installment-close-form" class="btn btn-danger" :disabled="isSaving">
        <Loader2 v-if="isSaving" class="animate-spin" />Beenden
      </button>
    </template>
  </AppSheet>
</template>
