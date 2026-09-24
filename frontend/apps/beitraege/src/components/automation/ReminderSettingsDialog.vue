<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { api } from '@/api';
import { X } from 'lucide-vue-next';

const emit = defineEmits<{ close: [] }>();

const reminderAutoEnabled = ref(false);
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
    reminderAutoEnabled.value = settings.autoEnabled;
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
      autoEnabled: reminderAutoEnabled.value,
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
  <div
    class="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4"
    @click.self="$emit('close')"
  >
    <div class="bg-white rounded-xl shadow-xl w-full max-w-lg max-h-[90vh] flex flex-col">
      <div class="flex items-start justify-between gap-4 p-5 border-b">
        <h3 class="text-lg font-semibold text-gray-900">Zahlungsdaten für QR-Code</h3>
        <button type="button" class="rounded-lg p-2 text-gray-500 hover:bg-gray-100 hover:text-gray-900" @click="$emit('close')">
          <X class="h-5 w-5" />
        </button>
      </div>
      <div class="overflow-y-auto p-5 space-y-3">
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">Empfänger</label>
          <input
            type="text"
            v-model="reminderPaymentRecipientName"
            :disabled="isReminderSettingsLoading"
            placeholder="Knirpsenstadt e.V."
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">IBAN</label>
          <input
            type="text"
            v-model="reminderPaymentIBAN"
            :disabled="isReminderSettingsLoading"
            placeholder="DE33370205000003321400"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">
            BIC <span class="font-normal text-gray-400">(optional)</span>
          </label>
          <input
            type="text"
            v-model="reminderPaymentBIC"
            :disabled="isReminderSettingsLoading"
            placeholder="BFSWDE33XXX"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          />
        </div>
        <p class="text-xs text-gray-500">Wenn Felder leer bleiben, werden die Standard-Zahlungsdaten der Kita verwendet.</p>
        <div v-if="reminderSettingsError" class="text-sm text-red-600">{{ reminderSettingsError }}</div>
      </div>
      <div class="flex justify-end gap-3 p-5 border-t">
        <button class="px-4 py-2 rounded-lg border text-sm font-medium hover:bg-gray-50" @click="$emit('close')">
          Abbrechen
        </button>
        <button
          class="px-4 py-2 rounded-lg bg-primary text-white text-sm font-medium hover:bg-primary/90 disabled:opacity-50"
          :disabled="isReminderSettingsLoading"
          @click="savePaymentSettings"
        >
          Speichern
        </button>
      </div>
    </div>
  </div>
</template>
