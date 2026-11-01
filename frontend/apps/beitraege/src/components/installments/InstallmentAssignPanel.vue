<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { api } from '@/api';
import type { BankTransaction, InstallmentDetail } from '@/api/types';
import { AlertTriangle, Check, Link2, Loader2, Plus } from 'lucide-vue-next';
import InstallmentPlanForm from '@/components/installments/InstallmentPlanForm.vue';
import { formatCurrency } from '@/utils/format';
import { debtorKindLabel, isOpenPlan, nextRateText } from '@/utils/installments';

// Assigns a bank transaction as a payment of an open plan, or opens the plan form prefilled from it.
const props = defineProps<{ transaction: BankTransaction }>();
const emit = defineEmits<{ matched: [] }>();

const openPlans = ref<InstallmentDetail[]>([]);
const selectedPlanId = ref<string | null>(props.transaction.installmentPlanSuggestion?.planId ?? null);
const isLoading = ref(true);
const isAssigning = ref(false);
const showForm = ref(false);
const error = ref<string | null>(null);

const selectedPlan = computed(() => openPlans.value.find(item => item.plan.id === selectedPlanId.value) ?? null);
const settlesRest = computed(() => !!selectedPlan.value && props.transaction.amount >= selectedPlan.value.summary.rest);

onMounted(async () => {
  try {
    openPlans.value = (await api.getInstallmentPlans()).filter(isOpenPlan);
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ratenzahlungen konnten nicht geladen werden';
  } finally {
    isLoading.value = false;
  }
});

async function assign(): Promise<void> {
  if (!selectedPlanId.value) return;
  isAssigning.value = true;
  error.value = null;
  try {
    await api.addInstallmentPayment(selectedPlanId.value, { transactionId: props.transaction.id });
    emit('matched');
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Zuordnung fehlgeschlagen';
  } finally {
    isAssigning.value = false;
  }
}
</script>

<template>
  <div class="space-y-4">
    <div v-if="transaction.installmentPlanSuggestion" class="note note-success">
      <Link2 />
      <span>Vorschlag über IBAN: {{ transaction.installmentPlanSuggestion.debtorName }}</span>
    </div>

    <div v-if="isLoading" class="flex justify-center py-6">
      <Loader2 class="h-5 w-5 animate-spin text-muted-foreground" />
    </div>
    <div v-else class="card divide-y">
      <label v-for="item in openPlans" :key="item.plan.id"
        class="flex min-h-14 cursor-pointer items-center gap-3 px-4 py-2.5 hover:bg-accent">
        <input v-model="selectedPlanId" type="radio" name="installment-plan" :value="item.plan.id"
          class="h-4 w-4" />
        <span class="min-w-0 flex-1">
          <span class="block truncate font-medium">{{ item.plan.debtorName }}</span>
          <span class="block truncate text-xs text-muted-foreground">
            {{ debtorKindLabel(item.plan) }}<template v-if="nextRateText(item)"> · {{ nextRateText(item) }}</template>
          </span>
        </span>
        <span class="whitespace-nowrap text-sm font-semibold tabular-nums">{{ formatCurrency(item.summary.rest) }}</span>
      </label>
      <button type="button" class="flex min-h-14 w-full items-center gap-3 px-4 py-2.5 text-left hover:bg-accent"
        @click="showForm = true">
        <Plus class="h-4 w-4 shrink-0 text-primary-text" />
        <span class="min-w-0 flex-1">
          <span class="block font-medium text-primary-text">Neue Ratenzahlung anlegen</span>
          <span class="block text-xs text-muted-foreground">Mit Daten aus dieser Überweisung vorbefüllt</span>
        </span>
      </button>
    </div>

    <div v-if="selectedPlan && settlesRest" class="note note-success">
      <Check />
      <span>Tilgt die Restschuld von {{ formatCurrency(selectedPlan.summary.rest) }} vollständig – die Ratenzahlung ist danach abgeschlossen.</span>
    </div>
    <div v-else-if="selectedPlan" class="note">
      <span>Wird der ältesten offenen Rate angerechnet. Ein Überschuss läuft in die nächste Rate.</span>
    </div>

    <div v-if="error" class="note note-danger" role="alert"><AlertTriangle />{{ error }}</div>

    <div class="flex justify-end">
      <button type="button" class="btn btn-primary" :disabled="!selectedPlanId || isAssigning" @click="assign">
        <Loader2 v-if="isAssigning" class="animate-spin" />Zuordnen
      </button>
    </div>

    <InstallmentPlanForm v-if="showForm" :prefill="transaction" @close="showForm = false" @saved="emit('matched')" />
  </div>
</template>
