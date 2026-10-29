<script setup lang="ts">
import AppSheet from '@/components/ui/AppSheet.vue';
import { ref, onMounted } from 'vue';
import { api } from '@/api';

const emit = defineEmits<{ close: [] }>();

const reminderPaymentRecipientName = ref('');
const reminderPaymentIBAN = ref('');
const reminderPaymentBIC = ref('');
const isReminderSettingsLoading = ref(false);
const reminderSettingsError = ref<string | null>(null);

async function loadReminderSettings(): Promise<void> {
  isReminderSettingsLoading.value = true;
  reminderSettingsError.value = null;
  try {
    const settings = await api.getReminderSettings();
    reminderPaymentRecipientName.value = settings.payment?.recipientName ?? '';
    reminderPaymentIBAN.value = settings.payment?.iban ?? '';
    reminderPaymentBIC.value = settings.payment?.bic ?? '';
  } catch (e) {
    reminderSettingsError.value = e instanceof Error ? e.message : 'Einstellungen konnten nicht geladen werden';
  } finally {
    isReminderSettingsLoading.value = false;
  }
}

function normalizeIBAN(value: string): string {
  return value.toUpperCase().replace(/\s+/g, '');
}

function normalizeBIC(value: string): string {
  return value.trim().toUpperCase();
}

async function savePaymentSettings(): Promise<void> {
  isReminderSettingsLoading.value = true;
  reminderSettingsError.value = null;
  try {
    await api.updateReminderSettings({
      payment: {
        recipientName: reminderPaymentRecipientName.value.trim(),
        iban: normalizeIBAN(reminderPaymentIBAN.value),
        bic: normalizeBIC(reminderPaymentBIC.value),
      },
    });
    emit('close');
  } catch (e) {
    reminderSettingsError.value = e instanceof Error ? e.message : 'Einstellungen konnten nicht gespeichert werden';
  } finally {
    isReminderSettingsLoading.value = false;
  }
}

onMounted(loadReminderSettings);
</script>

<template>
  <AppSheet title="Zahlungsdaten für QR-Code" size="md" @close="$emit('close')">
      <form id="reminder-payment-settings" class="space-y-3" @submit.prevent="savePaymentSettings">
        <div>
          <label class="label">Empfänger</label>
          <input
            type="text"
            v-model="reminderPaymentRecipientName"
            :disabled="isReminderSettingsLoading"
            placeholder="Knirpsenstadt e.V."
            class="input"
          />
        </div>
        <div>
          <label class="label">IBAN</label>
          <input
            type="text"
            v-model="reminderPaymentIBAN"
            :disabled="isReminderSettingsLoading"
            placeholder="DE33370205000003321400"
            class="input"
          />
        </div>
        <div>
          <label class="label">
            BIC <span class="font-normal text-muted-foreground">(optional)</span>
          </label>
          <input
            type="text"
            v-model="reminderPaymentBIC"
            :disabled="isReminderSettingsLoading"
            placeholder="BFSWDE33XXX"
            class="input"
          />
        </div>
        <p class="text-xs text-muted-foreground">Wenn Felder leer bleiben, werden die Standard-Zahlungsdaten der Kita verwendet.</p>
        <p v-if="reminderSettingsError" role="alert" class="note note-danger">{{ reminderSettingsError }}</p>
      </form>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="$emit('close')">
          Abbrechen
        </button>
        <button
          type="submit"
          form="reminder-payment-settings"
          class="btn btn-primary"
          :disabled="isReminderSettingsLoading"
        >
          Speichern
        </button>
      </template>
  </AppSheet>
</template>
