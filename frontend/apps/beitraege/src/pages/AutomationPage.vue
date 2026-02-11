<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue';
import { api } from '@/api';
import type { BankingSyncStatus, ReminderRunResponse, EmailLog } from '@/api/types';
import { Loader2, RefreshCw, Square } from 'lucide-vue-next';
import { useAuthStore } from '@/stores/auth';

const authStore = useAuthStore();

// Banking sync state
const bankingSyncStatus = ref<BankingSyncStatus | null>(null);
const bankingSyncError = ref<string | null>(null);
const isStartingBankingSync = ref(false);
const isCancellingBankingSync = ref(false);
const isLoadingBankingSync = ref(false);
let bankingSyncPollInterval: ReturnType<typeof setInterval> | null = null;

// Reminder state
const reminderAutoEnabled = ref(false);
const isReminderSettingsLoading = ref(false);
const reminderSettingsError = ref<string | null>(null);
const reminderRunError = ref<string | null>(null);
const reminderRunResult = ref<ReminderRunResponse | null>(null);
const reminderDate = ref(new Date().toLocaleDateString('en-CA'));
const reminderDryRun = ref(true);
const isRunningReminders = ref(false);

// Email logs state
const emailLogs = ref<EmailLog[]>([]);
const emailLogsTotal = ref(0);
const emailLogsOffset = ref(0);
const emailLogsLimit = 20;
const isEmailLogsLoading = ref(false);
const emailLogsError = ref<string | null>(null);

// Banking Sync Functions
function clearBankingSyncPolling(): void {
  if (bankingSyncPollInterval) {
    clearInterval(bankingSyncPollInterval);
    bankingSyncPollInterval = null;
  }
}

function shouldPollBankingSync(status?: BankingSyncStatus | null): boolean {
  return status?.status === 'running' || status?.status === 'waiting_for_2fa';
}

function startBankingSyncPolling(): void {
  if (bankingSyncPollInterval) return;
  bankingSyncPollInterval = setInterval(() => {
    loadBankingSyncStatus();
  }, 5000);
}

async function loadBankingSyncStatus(): Promise<void> {
  isLoadingBankingSync.value = true;
  bankingSyncError.value = null;
  try {
    const status = await api.getBankingSyncStatus();
    bankingSyncStatus.value = status;
    if (shouldPollBankingSync(status)) {
      startBankingSyncPolling();
    } else {
      clearBankingSyncPolling();
    }
  } catch (error) {
    bankingSyncError.value = error instanceof Error ? error.message : 'Status konnte nicht geladen werden';
    clearBankingSyncPolling();
  } finally {
    isLoadingBankingSync.value = false;
  }
}

async function runBankingSync(): Promise<void> {
  isStartingBankingSync.value = true;
  bankingSyncError.value = null;
  try {
    const status = await api.runBankingSync();
    bankingSyncStatus.value = status;
    if (shouldPollBankingSync(status)) {
      startBankingSyncPolling();
    }
  } catch (error) {
    bankingSyncError.value = error instanceof Error ? error.message : 'Sync konnte nicht gestartet werden';
  } finally {
    isStartingBankingSync.value = false;
  }
}

async function cancelBankingSync(): Promise<void> {
  isCancellingBankingSync.value = true;
  bankingSyncError.value = null;
  try {
    const status = await api.cancelBankingSync();
    bankingSyncStatus.value = status;
    if (shouldPollBankingSync(status)) {
      startBankingSyncPolling();
    } else {
      clearBankingSyncPolling();
    }
  } catch (error) {
    bankingSyncError.value = error instanceof Error ? error.message : 'Sync konnte nicht gestoppt werden';
  } finally {
    isCancellingBankingSync.value = false;
  }
}

const bankingSyncStatusLabel = computed(() => {
  switch (bankingSyncStatus.value?.status) {
    case 'running':
      return 'Läuft';
    case 'waiting_for_2fa':
      return 'Wartet auf 2FA';
    case 'success':
      return 'Erfolgreich';
    case 'error':
      return 'Fehlgeschlagen';
    case 'idle':
      return 'Bereit';
    default:
      return 'Unbekannt';
  }
});

const bankingSyncStatusTone = computed(() => {
  switch (bankingSyncStatus.value?.status) {
    case 'running':
      return 'bg-blue-100 text-blue-700';
    case 'waiting_for_2fa':
      return 'bg-amber-100 text-amber-700';
    case 'success':
      return 'bg-green-100 text-green-700';
    case 'error':
      return 'bg-red-100 text-red-700';
    case 'idle':
      return 'bg-gray-100 text-gray-700';
    default:
      return 'bg-gray-100 text-gray-700';
  }
});

const bankingSyncStatusHint = computed(() => {
  if (bankingSyncStatus.value?.status === 'waiting_for_2fa') {
    return 'Bitte in der SecureGo Plus App bestätigen.';
  }
  if (bankingSyncStatus.value?.status === 'error') {
    return bankingSyncStatus.value?.lastError || 'Sync fehlgeschlagen.';
  }
  if (bankingSyncStatus.value?.status === 'success') {
    return 'Letzter Lauf erfolgreich abgeschlossen.';
  }
  return null;
});

const bankingSyncShowLastMessage = computed(() => {
  if (!bankingSyncStatus.value?.lastMessage) return false;
  return bankingSyncStatus.value.status !== 'success';
});

const bankingSyncIsBusy = computed(() => {
  return (
    isStartingBankingSync.value ||
    isCancellingBankingSync.value ||
    shouldPollBankingSync(bankingSyncStatus.value)
  );
});

// Reminder Functions
async function loadReminderSettings() {
  if (!authStore.isAdmin) return;
  isReminderSettingsLoading.value = true;
  reminderSettingsError.value = null;
  try {
    const settings = await api.getReminderSettings();
    reminderAutoEnabled.value = settings.autoEnabled;
  } catch (e) {
    reminderSettingsError.value = e instanceof Error ? e.message : 'Einstellungen konnten nicht geladen werden';
  } finally {
    isReminderSettingsLoading.value = false;
  }
}

async function updateReminderAutoEnabled() {
  if (!authStore.isAdmin) return;
  isReminderSettingsLoading.value = true;
  reminderSettingsError.value = null;
  try {
    const settings = await api.updateReminderSettings({ autoEnabled: reminderAutoEnabled.value });
    reminderAutoEnabled.value = settings.autoEnabled;
  } catch (e) {
    reminderSettingsError.value = e instanceof Error ? e.message : 'Einstellungen konnten nicht gespeichert werden';
  } finally {
    isReminderSettingsLoading.value = false;
  }
}

async function runReminders(stage: 'initial' | 'final') {
  if (!authStore.isAdmin) return;
  isRunningReminders.value = true;
  reminderRunError.value = null;
  reminderRunResult.value = null;
  try {
    const result = await api.runReminders({
      stage,
      date: reminderDate.value,
      dryRun: reminderDryRun.value,
    });
    reminderRunResult.value = result;
  } catch (e) {
    reminderRunError.value = e instanceof Error ? e.message : 'Erinnerung konnte nicht ausgelöst werden';
  } finally {
    isRunningReminders.value = false;
  }
}

// Email Logs Functions
async function loadEmailLogs(reset = false) {
  if (!authStore.isAdmin) return;
  if (isEmailLogsLoading.value) return;

  isEmailLogsLoading.value = true;
  emailLogsError.value = null;
  try {
    if (reset) {
      emailLogsOffset.value = 0;
      emailLogs.value = [];
    }
    const result = await api.getEmailLogs({
      offset: emailLogsOffset.value,
      limit: emailLogsLimit,
    });
    emailLogs.value = [...emailLogs.value, ...result.data];
    emailLogsTotal.value = result.total;
    emailLogsOffset.value = emailLogs.value.length;
  } catch (e) {
    emailLogsError.value = e instanceof Error ? e.message : 'E-Mail-Protokoll konnte nicht geladen werden';
  } finally {
    isEmailLogsLoading.value = false;
  }
}

function formatEmailType(type: string): string {
  switch (type) {
    case 'REMINDER_INITIAL':
      return 'Zahlungserinnerung';
    case 'REMINDER_FINAL':
      return 'Mahnung';
    case 'PASSWORD_RESET':
      return 'Passwort-Reset';
    default:
      return type;
  }
}

function formatDateTime(date: string): string {
  return new Date(date).toLocaleString('de-DE');
}

// Lifecycle
onMounted(() => {
  loadBankingSyncStatus();
  if (authStore.isAdmin) {
    loadReminderSettings();
    loadEmailLogs(true);
  }
});

onUnmounted(() => {
  clearBankingSyncPolling();
});

watch(
  () => authStore.isAdmin,
  (isAdmin) => {
    if (isAdmin) {
      loadReminderSettings();
      loadEmailLogs(true);
    }
  }
);
</script>

<template>
  <div>
    <!-- Header -->
    <div class="mb-8">
      <h1 class="text-2xl font-bold text-gray-900">Automatisierung</h1>
      <p class="text-gray-600 mt-1">Automatische Prozesse und geplante Aufgaben</p>
    </div>

    <!-- Banking Sync Card -->
    <div class="bg-white rounded-xl border p-6 mb-6">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 class="text-lg font-semibold text-gray-900">Banking Sync</h2>
          <p class="text-sm text-gray-600">
            Automatischer CSV-Export und Import aus dem Banking-Portal.
          </p>
        </div>
        <div class="flex items-center gap-3">
          <button
            @click="loadBankingSyncStatus"
            :disabled="isLoadingBankingSync"
            class="text-sm text-gray-600 hover:text-gray-900 underline disabled:opacity-50"
          >
            Aktualisieren
          </button>
          <button
            @click="runBankingSync"
            :disabled="bankingSyncIsBusy || isLoadingBankingSync"
            class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isStartingBankingSync" class="h-4 w-4 animate-spin" />
            <RefreshCw v-else class="h-4 w-4" />
            Jetzt synchronisieren
          </button>
          <button
            v-if="shouldPollBankingSync(bankingSyncStatus)"
            @click="cancelBankingSync"
            :disabled="isCancellingBankingSync || isLoadingBankingSync"
            class="inline-flex items-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isCancellingBankingSync" class="h-4 w-4 animate-spin" />
            <Square v-else class="h-4 w-4" />
            Stoppen
          </button>
        </div>
      </div>

      <div class="mt-4 flex flex-wrap items-center gap-3 text-sm text-gray-600">
        <span class="px-2 py-1 rounded-full text-xs font-medium" :class="bankingSyncStatusTone">
          {{ bankingSyncStatusLabel }}
        </span>
        <span v-if="bankingSyncStatus?.startedAt">
          Start: {{ formatDateTime(bankingSyncStatus.startedAt) }}
        </span>
        <span v-if="bankingSyncStatus?.finishedAt">
          Ende: {{ formatDateTime(bankingSyncStatus.finishedAt) }}
        </span>
        <span v-if="bankingSyncShowLastMessage" class="text-gray-500">
          {{ bankingSyncStatus?.lastMessage }}
        </span>
      </div>

      <div
        v-if="bankingSyncError"
        class="mt-3 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700"
      >
        {{ bankingSyncError }}
      </div>
      <div
        v-else-if="bankingSyncStatusHint"
        class="mt-3 p-3 bg-amber-50 border border-amber-200 rounded-lg text-sm text-amber-800"
      >
        {{ bankingSyncStatusHint }}
      </div>
    </div>

    <!-- Zahlungserinnerungen Card -->
    <div v-if="authStore.isAdmin" class="bg-white rounded-xl border p-6 mb-6">
      <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-4">
        <div>
          <h2 class="text-lg font-semibold text-gray-900">Zahlungserinnerungen</h2>
          <p class="text-sm text-gray-600">
            Erinnerungen und Mahnungen werden an deine Login-E-Mail gesendet.
          </p>
        </div>
        <div class="flex items-center gap-3">
          <span class="text-sm text-gray-600">Automatik</span>
          <label class="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              class="sr-only peer"
              v-model="reminderAutoEnabled"
              :disabled="isReminderSettingsLoading"
              @change="updateReminderAutoEnabled"
            />
            <div
              class="w-11 h-6 bg-gray-200 rounded-full peer peer-checked:bg-primary transition-colors"
            ></div>
            <div
              class="absolute left-1 top-1 w-4 h-4 bg-white rounded-full transition-transform peer-checked:translate-x-5"
            ></div>
          </label>
        </div>
      </div>

      <div class="flex flex-col lg:flex-row lg:items-end gap-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">Datum</label>
          <input
            type="date"
            v-model="reminderDate"
            class="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          />
        </div>
        <label class="inline-flex items-center gap-2 text-sm text-gray-700">
          <input type="checkbox" v-model="reminderDryRun" />
          Nur Vorschau (keine E-Mails, keine Mahngebühren)
        </label>
        <div class="flex flex-col sm:flex-row gap-2">
          <button
            class="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50"
            :disabled="isRunningReminders"
            @click="runReminders('initial')"
          >
            Erinnerung senden
          </button>
          <button
            class="px-4 py-2 rounded-lg bg-red-600 text-white text-sm font-medium hover:bg-red-700 disabled:opacity-50"
            :disabled="isRunningReminders"
            @click="runReminders('final')"
          >
            Mahnung senden
          </button>
        </div>
      </div>

      <div v-if="isReminderSettingsLoading" class="mt-3 text-sm text-gray-500">
        Einstellungen werden aktualisiert...
      </div>
      <div v-if="reminderSettingsError" class="mt-3 text-sm text-red-600">
        {{ reminderSettingsError }}
      </div>

      <div v-if="reminderRunError" class="mt-3 text-sm text-red-600">
        {{ reminderRunError }}
      </div>
      <div v-if="reminderRunResult" class="mt-3 text-sm text-gray-700">
        <p class="font-medium">Ergebnis</p>
        <p>
          Stufe: {{ reminderRunResult.stage }}
          · Offene Beiträge: {{ reminderRunResult.unpaidCount }}
          · Mahngebühren erstellt: {{ reminderRunResult.reminderCreated }}
        </p>
        <p>
          E-Mail gesendet: {{ reminderRunResult.emailSent ? 'Ja' : 'Nein' }}
          · Trockenlauf: {{ reminderRunResult.dryRun ? 'Ja' : 'Nein' }}
        </p>
        <p v-if="reminderRunResult.message" class="text-gray-500">
          {{ reminderRunResult.message }}
        </p>
      </div>
    </div>

    <!-- Email Logs Card -->
    <div v-if="authStore.isAdmin" class="bg-white rounded-xl border p-6">
      <div class="flex items-center justify-between mb-4">
        <div>
          <h2 class="text-lg font-semibold text-gray-900">E-Mail-Protokoll</h2>
          <p class="text-sm text-gray-600">Alle versendeten E-Mails inklusive Inhalt.</p>
        </div>
        <button
          class="text-sm text-primary hover:underline"
          :disabled="isEmailLogsLoading"
          @click="loadEmailLogs(true)"
        >
          Neu laden
        </button>
      </div>

      <div v-if="emailLogsError" class="text-sm text-red-600 mb-3">
        {{ emailLogsError }}
      </div>

      <div v-if="emailLogs.length === 0 && !isEmailLogsLoading" class="text-sm text-gray-500">
        Noch keine E-Mails protokolliert.
      </div>

      <div v-else class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-left text-gray-500 border-b">
              <th class="pb-3 font-medium">Zeitpunkt</th>
              <th class="pb-3 font-medium">Typ</th>
              <th class="pb-3 font-medium">Empfänger</th>
              <th class="pb-3 font-medium">Betreff</th>
              <th class="pb-3 font-medium">Inhalt</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in emailLogs" :key="log.id" class="border-b last:border-0 align-top">
              <td class="py-3 whitespace-nowrap">{{ formatDateTime(log.sentAt) }}</td>
              <td class="py-3 whitespace-nowrap">{{ formatEmailType(log.emailType) }}</td>
              <td class="py-3 whitespace-nowrap">{{ log.toEmail }}</td>
              <td class="py-3">{{ log.subject }}</td>
              <td class="py-3">
                <details class="text-sm">
                  <summary class="cursor-pointer text-primary">Anzeigen</summary>
                  <pre class="mt-2 whitespace-pre-wrap text-gray-700 bg-gray-50 border rounded-lg p-3">{{ log.body || '-' }}</pre>
                </details>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="isEmailLogsLoading" class="mt-3 text-sm text-gray-500">
        E-Mail-Protokoll wird geladen...
      </div>

      <div v-if="emailLogs.length < emailLogsTotal" class="mt-4">
        <button
          class="px-3 py-2 rounded-lg border text-sm font-medium hover:bg-gray-50 disabled:opacity-50"
          :disabled="isEmailLogsLoading"
          @click="loadEmailLogs()"
        >
          Mehr laden
        </button>
      </div>
    </div>
  </div>
</template>
