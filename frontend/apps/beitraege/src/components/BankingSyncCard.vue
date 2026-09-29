<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { api } from '@/api';
import type { BankingSyncStatus, ImportError } from '@/api/types';
import { Loader2, RefreshCw, Square } from 'lucide-vue-next';
import ImportErrorList from '@/components/ImportErrorList.vue';
import { formatDateTime } from '@/utils/format';

const emit = defineEmits<{
  (e: 'status-change', status: BankingSyncStatus): void;
  (e: 'sync-finished', status: BankingSyncStatus): void;
}>();

const bankingSyncStatus = ref<BankingSyncStatus | null>(null);
const bankingSyncError = ref<string | null>(null);
const isStartingBankingSync = ref(false);
const isCancellingBankingSync = ref(false);
const isLoadingBankingSync = ref(false);
let bankingSyncPollInterval: ReturnType<typeof setInterval> | null = null;

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
    const previousStatus = bankingSyncStatus.value?.status;
    bankingSyncStatus.value = status;
    emit('status-change', status);
    if (shouldPollBankingSync(status)) {
      startBankingSyncPolling();
    } else {
      clearBankingSyncPolling();
      if (
        status.status === 'success' &&
        (previousStatus === 'running' || previousStatus === 'waiting_for_2fa')
      ) {
        emit('sync-finished', status);
      }
    }
  } catch (error) {
    bankingSyncError.value =
      error instanceof Error ? error.message : 'Status konnte nicht geladen werden';
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
    emit('status-change', status);
    if (shouldPollBankingSync(status)) {
      startBankingSyncPolling();
    }
  } catch (error) {
    bankingSyncError.value =
      error instanceof Error ? error.message : 'Sync konnte nicht gestartet werden';
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
    emit('status-change', status);
    if (shouldPollBankingSync(status)) {
      startBankingSyncPolling();
    } else {
      clearBankingSyncPolling();
    }
  } catch (error) {
    bankingSyncError.value =
      error instanceof Error ? error.message : 'Sync konnte nicht gestoppt werden';
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
      return 'bg-blue-100 dark:bg-blue-950/40 text-blue-700 dark:text-blue-300';
    case 'waiting_for_2fa':
      return 'bg-amber-100 dark:bg-amber-950/40 text-amber-700 dark:text-amber-300';
    case 'success':
      return 'bg-green-100 dark:bg-green-950/40 text-green-700 dark:text-green-300';
    case 'error':
      return 'bg-red-100 dark:bg-red-950/40 text-red-700 dark:text-red-300';
    case 'idle':
      return 'bg-muted text-foreground';
    default:
      return 'bg-muted text-foreground';
  }
});

const bankingSyncStatusHint = computed(() => {
  if (bankingSyncStatus.value?.status === 'waiting_for_2fa') {
    return 'Bitte in der SecureGo Plus App bestätigen.';
  }
  if (bankingSyncStatus.value?.status === 'error') {
    return bankingSyncStatus.value?.lastError || 'Sync fehlgeschlagen.';
  }
  // Success is already shown by the status badge; only problems and 2FA prompts get a hint box.
  return null;
});

// uploadResult is the backend's import response forwarded by banking-sync.
const uploadErrors = computed<ImportError[]>(() => {
  const result = bankingSyncStatus.value?.uploadResult;
  if (!result || typeof result !== 'object' || !('errors' in result)) return [];
  const errors = (result as { errors?: unknown }).errors;
  return Array.isArray(errors) ? (errors as ImportError[]) : [];
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

onMounted(() => {
  loadBankingSyncStatus();
});

onUnmounted(() => {
  clearBankingSyncPolling();
});

defineExpose({ reload: loadBankingSyncStatus });
</script>

<template>
  <div class="bg-card rounded-xl border p-6 mb-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
      <div>
        <h2 class="text-lg font-semibold text-foreground">Banking Sync</h2>
        <p class="text-sm text-muted-foreground">
          Automatischer CSV-Export und Import aus dem Banking-Portal.
        </p>
      </div>
      <div class="flex items-center gap-3">
        <button
          @click="loadBankingSyncStatus"
          :disabled="isLoadingBankingSync"
          class="text-sm text-muted-foreground hover:text-foreground underline disabled:opacity-50"
        >
          Aktualisieren
        </button>
        <button
          @click="runBankingSync"
          :disabled="bankingSyncIsBusy || isLoadingBankingSync"
          class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
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

    <div class="mt-4 flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
      <span class="px-2 py-1 rounded-full text-xs font-medium" :class="bankingSyncStatusTone">
        {{ bankingSyncStatusLabel }}
      </span>
      <span v-if="bankingSyncStatus?.startedAt">
        Start: {{ formatDateTime(bankingSyncStatus.startedAt) }}
      </span>
      <span v-if="bankingSyncStatus?.finishedAt">
        Ende: {{ formatDateTime(bankingSyncStatus.finishedAt) }}
      </span>
      <span v-if="bankingSyncShowLastMessage" class="text-muted-foreground">
        {{ bankingSyncStatus?.lastMessage }}
      </span>
    </div>

    <div
      v-if="bankingSyncError"
      class="mt-3 p-3 bg-red-50 dark:bg-red-950/40 border border-red-200 rounded-lg text-sm text-red-700 dark:text-red-300"
    >
      {{ bankingSyncError }}
    </div>
    <div
      v-else-if="bankingSyncStatusHint"
      class="mt-3 p-3 border rounded-lg text-sm"
      :class="bankingSyncStatus?.status === 'error'
        ? 'bg-red-50 dark:bg-red-950/40 border-red-200 text-red-700 dark:text-red-300'
        : 'bg-amber-50 dark:bg-amber-950/40 border-amber-200 text-amber-800 dark:text-amber-300'"
    >
      {{ bankingSyncStatusHint }}
    </div>
    <ImportErrorList
      v-if="bankingSyncStatus?.status === 'success' && uploadErrors.length > 0"
      class="mt-3"
      title="Sync abgeschlossen, aber Fehler beim Import"
      :errors="uploadErrors"
    />
  </div>
</template>
