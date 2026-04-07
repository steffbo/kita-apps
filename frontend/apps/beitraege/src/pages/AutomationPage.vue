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
const reminderDeadline = ref('');
const reminderDryRun = ref(true);
const isRunningReminders = ref(false);

// Dry-run preview modal state
const showPreviewModal = ref(false);
const previewStage = ref<'initial' | 'final'>('initial');
const expandedPreview = ref<string | null>(null);

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
      deadline: reminderDeadline.value || undefined,
    });
    reminderRunResult.value = result;
    if (result.dryRun && result.previews && result.previews.length > 0) {
      previewStage.value = stage;
      expandedPreview.value = null;
      showPreviewModal.value = true;
    }
  } catch (e) {
    reminderRunError.value = e instanceof Error ? e.message : 'Erinnerung konnte nicht ausgelöst werden';
  } finally {
    isRunningReminders.value = false;
  }
}

async function sendFromModal() {
  showPreviewModal.value = false;
  if (!authStore.isAdmin) return;
  isRunningReminders.value = true;
  reminderRunError.value = null;
  reminderRunResult.value = null;
  try {
    const result = await api.runReminders({
      stage: previewStage.value,
      date: reminderDate.value,
      dryRun: false,
      deadline: reminderDeadline.value || undefined,
    });
    reminderRunResult.value = result;
  } catch (e) {
    reminderRunError.value = e instanceof Error ? e.message : 'Erinnerung konnte nicht ausgelöst werden';
  } finally {
    isRunningReminders.value = false;
  }
}

function togglePreview(householdName: string) {
  expandedPreview.value = expandedPreview.value === householdName ? null : householdName;
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
            Erinnerungen und Mahnungen werden direkt an die Eltern der jeweiligen Familie gesendet.
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
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">
            Frist <span class="font-normal text-gray-400">(optional, Standard: 10. des Monats)</span>
          </label>
          <input
            type="date"
            v-model="reminderDeadline"
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

      <!-- Result after real run -->
      <div v-if="reminderRunResult && !reminderRunResult.dryRun" class="mt-4 p-4 bg-gray-50 border rounded-lg text-sm text-gray-700">
        <p class="font-medium mb-2">Ergebnis</p>
        <p>
          Familien kontaktiert: <span class="font-medium">{{ reminderRunResult.familiesEmailed }}</span>
          · Übersprungen: <span class="font-medium">{{ reminderRunResult.familiesSkippedNoEmail }}</span>
          · Offene Beiträge: <span class="font-medium">{{ reminderRunResult.unpaidCount }}</span>
        </p>
        <p v-if="reminderRunResult.remindersCreated" class="mt-1">
          Mahngebühren erstellt: <span class="font-medium">{{ reminderRunResult.remindersCreated }}</span>
        </p>
        <ul v-if="reminderRunResult.warnings && reminderRunResult.warnings.length > 0" class="mt-2 space-y-1">
          <li
            v-for="warn in reminderRunResult.warnings"
            :key="warn.householdName"
            class="text-amber-700"
          >
            Familie {{ warn.householdName }}: {{ warn.reason }}
          </li>
        </ul>
        <p v-if="reminderRunResult.message" class="mt-1 text-gray-500">
          {{ reminderRunResult.message }}
        </p>
      </div>

      <!-- Result after dry-run (when no previews or empty result) -->
      <div v-if="reminderRunResult && reminderRunResult.dryRun && (!reminderRunResult.previews || reminderRunResult.previews.length === 0)" class="mt-4 p-4 bg-gray-50 border rounded-lg text-sm text-gray-700">
        <p class="font-medium">Vorschau</p>
        <p class="text-gray-500">{{ reminderRunResult.message || 'Keine offenen Beiträge für diesen Zeitraum.' }}</p>
      </div>
    </div>

    <!-- Dry-run preview modal -->
    <div v-if="showPreviewModal" class="fixed inset-0 z-50 flex items-start justify-center pt-16 px-4">
      <div class="absolute inset-0 bg-black/40" @click="showPreviewModal = false"></div>
      <div class="relative bg-white rounded-xl border shadow-xl w-full max-w-2xl max-h-[80vh] flex flex-col">
        <div class="flex items-center justify-between p-5 border-b">
          <div>
            <h3 class="text-base font-semibold text-gray-900">Vorschau E-Mails</h3>
            <p class="text-sm text-gray-500 mt-0.5">
              {{ reminderRunResult?.familiesEmailed }} Familie(n) würden kontaktiert
              <template v-if="reminderRunResult?.familiesSkippedNoEmail">
                · {{ reminderRunResult?.familiesSkippedNoEmail }} übersprungen
              </template>
            </p>
          </div>
          <button @click="showPreviewModal = false" class="text-gray-400 hover:text-gray-600 text-xl leading-none">&times;</button>
        </div>

        <div class="overflow-y-auto p-5 space-y-3 flex-1">
          <!-- Warnings -->
          <div
            v-if="reminderRunResult?.warnings && reminderRunResult.warnings.length > 0"
            class="p-3 bg-amber-50 border border-amber-200 rounded-lg text-sm text-amber-800"
          >
            <p class="font-medium mb-1">Übersprungene Familien</p>
            <ul class="space-y-0.5">
              <li v-for="warn in reminderRunResult.warnings" :key="warn.householdName">
                <span class="font-medium">{{ warn.householdName }}</span>: {{ warn.reason }}
              </li>
            </ul>
          </div>

          <!-- Per-family previews -->
          <div
            v-for="prev in reminderRunResult?.previews"
            :key="prev.householdName"
            class="border rounded-lg overflow-hidden"
          >
            <button
              class="w-full flex items-center justify-between px-4 py-3 text-sm font-medium text-gray-800 hover:bg-gray-50 text-left"
              @click="togglePreview(prev.householdName)"
            >
              <span>{{ prev.householdName }}</span>
              <span class="text-xs text-gray-500">{{ prev.recipients.join(', ') }}</span>
            </button>
            <div v-if="expandedPreview === prev.householdName" class="border-t px-4 py-3 bg-gray-50 text-sm space-y-2">
              <p class="font-medium text-gray-700">Betreff: {{ prev.subject }}</p>
              <pre class="whitespace-pre-wrap text-gray-600 bg-white border rounded p-3 text-xs">{{ prev.body }}</pre>
            </div>
          </div>
        </div>

        <div class="flex justify-end gap-3 p-5 border-t">
          <button
            class="px-4 py-2 rounded-lg border text-sm font-medium hover:bg-gray-50"
            @click="showPreviewModal = false"
          >
            Schließen
          </button>
          <button
            class="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50"
            :disabled="isRunningReminders"
            @click="sendFromModal"
          >
            Jetzt senden
          </button>
        </div>
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
