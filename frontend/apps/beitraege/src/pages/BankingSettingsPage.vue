<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { api } from '@/api';
import type { BankingConfig, SyncStatus, SyncResult } from '@/api/types';
import {
  Building2,
  Save,
  Loader2,
  RefreshCw,
  CheckCircle,
  AlertTriangle,
  Trash2,
  TestTube,
} from 'lucide-vue-next';

const isLoading = ref(false);
const isSaving = ref(false);
const isTesting = ref(false);
const isSyncing = ref(false);
const error = ref<string | null>(null);
const successMessage = ref<string | null>(null);

const config = ref<BankingConfig>({
  bankName: 'SozialBank',
  bankBlz: '',
  userId: '',
  accountNumber: '',
  pin: '',
  fintsUrl: 'https://fints.sozialbank.com/fints',
  syncEnabled: true,
  isConfigured: false,
});

const syncStatus = ref<SyncStatus | null>(null);

onMounted(async () => {
  await loadConfig();
  await loadSyncStatus();
});

async function loadConfig() {
  try {
    const data = await api.getBankingConfig();
    config.value = { ...config.value, ...data, pin: '' };
    error.value = null;
  } catch (e) {
    // Config might not exist yet, that's ok
    if ((e as Error).message?.includes('not found')) {
      error.value = null;
    } else {
      error.value = 'Fehler beim Laden der Konfiguration';
    }
  }
}

async function loadSyncStatus() {
  try {
    syncStatus.value = await api.getSyncStatus();
  } catch (e) {
    console.error('Failed to load sync status:', e);
  }
}

async function saveConfig() {
  isSaving.value = true;
  error.value = null;
  successMessage.value = null;

  try {
    await api.saveBankingConfig(config.value);
    successMessage.value = 'Konfiguration erfolgreich gespeichert';
    await loadConfig();
    await loadSyncStatus();
  } catch (e) {
    error.value = (e as Error).message || 'Fehler beim Speichern';
  } finally {
    isSaving.value = false;
  }
}

async function testConnection() {
  isTesting.value = true;
  error.value = null;
  successMessage.value = null;

  try {
    const result = await api.testBankConnection();
    successMessage.value = result.message;
  } catch (e) {
    error.value = (e as Error).message || 'Verbindungstest fehlgeschlagen';
  } finally {
    isTesting.value = false;
  }
}

async function triggerSync() {
  isSyncing.value = true;
  error.value = null;
  successMessage.value = null;

  try {
    const result = await api.triggerSync();
    if (result.success) {
      successMessage.value = `Synchronisation erfolgreich: ${result.transactionsImported} neue Transaktionen importiert`;
    } else {
      error.value = result.errors?.join(', ') || 'Synchronisation fehlgeschlagen';
    }
    await loadSyncStatus();
  } catch (e) {
    error.value = (e as Error).message || 'Synchronisation fehlgeschlagen';
  } finally {
    isSyncing.value = false;
  }
}

async function deleteConfig() {
  if (!confirm('Möchtest du die Bank-Konfiguration wirklich löschen?')) {
    return;
  }

  try {
    await api.deleteBankingConfig();
    config.value = {
      bankName: 'SozialBank',
      bankBlz: '',
      userId: '',
      accountNumber: '',
      pin: '',
      fintsUrl: 'https://fints.sozialbank.com/fints',
      syncEnabled: true,
      isConfigured: false,
    };
    syncStatus.value = null;
    successMessage.value = 'Konfiguration gelöscht';
  } catch (e) {
    error.value = (e as Error).message || 'Fehler beim Löschen';
  }
}

function formatDate(dateString?: string): string {
  if (!dateString) return '-';
  const date = new Date(dateString);
  return date.toLocaleString('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}
</script>

<template>
  <div class="container mx-auto py-6 px-4">
    <!-- Header -->
    <div class="mb-6">
      <div class="flex items-center gap-2 mb-2">
        <Building2 class="w-6 h-6 text-blue-600" />
        <h1 class="text-2xl font-bold text-gray-900">Bank-Synchronisation</h1>
      </div>
      <p class="text-gray-600">
        Automatische Synchronisation mit der SozialBank via FinTS
      </p>
    </div>

    <!-- Alerts -->
    <div v-if="error" class="mb-4 p-4 bg-red-50 border border-red-200 rounded-lg">
      <div class="flex items-center gap-2">
        <AlertTriangle class="w-5 h-5 text-red-600" />
        <span class="text-red-800">{{ error }}</span>
      </div>
    </div>

    <div v-if="successMessage" class="mb-4 p-4 bg-green-50 border border-green-200 rounded-lg">
      <div class="flex items-center gap-2">
        <CheckCircle class="w-5 h-5 text-green-600" />
        <span class="text-green-800">{{ successMessage }}</span>
      </div>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Configuration Form -->
      <div class="lg:col-span-2 space-y-6">
        <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <h2 class="text-lg font-semibold text-gray-900 mb-4">Bank-Zugangsdaten</h2>

          <div class="space-y-4">
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">
                  Bankname
                </label>
                <input
                  v-model="config.bankName"
                  type="text"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  placeholder="SozialBank"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">
                  Bankleitzahl (BLZ)
                </label>
                <input
                  v-model="config.bankBlz"
                  type="text"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  placeholder="12345678"
                  maxlength="8"
                />
              </div>
            </div>

            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">
                User ID / NetKey <span class="text-red-500">*</span>
                <span class="text-sm text-gray-500 ml-1">(Online-Banking Benutzername)</span>
              </label>
              <input
                v-model="config.userId"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                placeholder="Ihr NetKey"
              />
            </div>

            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">
                Kontonummer <span class="text-sm text-gray-500">(optional)</span>
              </label>
              <input
                v-model="config.accountNumber"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                placeholder="1234567890"
              />
            </div>

            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">
                PIN / Passwort
                <span v-if="config.isConfigured" class="text-sm text-gray-500 ml-2">
                  (leer lassen um aktuelle PIN beizubehalten)
                </span>
              </label>
              <input
                v-model="config.pin"
                type="password"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                placeholder="••••••"
              />
            </div>

            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">
                FinTS URL
              </label>
              <input
                v-model="config.fintsUrl"
                type="url"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                placeholder="https://fints.sozialbank.com/fints"
              />
            </div>

            <div class="flex items-center">
              <input
                v-model="config.syncEnabled"
                type="checkbox"
                id="syncEnabled"
                class="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              />
              <label for="syncEnabled" class="ml-2 block text-sm text-gray-700">
                Automatische Synchronisation aktivieren
              </label>
            </div>
          </div>

          <div class="mt-6 flex gap-3">
            <button
              @click="saveConfig"
              :disabled="isSaving"
              class="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Loader2 v-if="isSaving" class="w-4 h-4 animate-spin" />
              <Save v-else class="w-4 h-4" />
              {{ isSaving ? 'Speichern...' : 'Speichern' }}
            </button>

            <button
              v-if="config.isConfigured"
              @click="testConnection"
              :disabled="isTesting"
              class="flex items-center gap-2 px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Loader2 v-if="isTesting" class="w-4 h-4 animate-spin" />
              <TestTube v-else class="w-4 h-4" />
              {{ isTesting ? 'Teste...' : 'Verbindung testen' }}
            </button>

            <button
              v-if="config.isConfigured"
              @click="deleteConfig"
              class="flex items-center gap-2 px-4 py-2 bg-red-50 text-red-700 rounded-lg hover:bg-red-100 ml-auto"
            >
              <Trash2 class="w-4 h-4" />
              Löschen
            </button>
          </div>
        </div>
      </div>

      <!-- Status Panel -->
      <div class="space-y-6">
        <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <h2 class="text-lg font-semibold text-gray-900 mb-4">Synchronisations-Status</h2>

          <div v-if="!syncStatus" class="text-gray-500 text-center py-4">
            Keine Konfiguration vorhanden
          </div>

          <div v-else class="space-y-4">
            <div class="flex items-center justify-between">
              <span class="text-sm text-gray-600">Status:</span>
              <span
                :class="{
                  'px-2 py-1 rounded-full text-xs font-medium': true,
                  'bg-green-100 text-green-800': syncStatus.isConfigured && syncStatus.syncEnabled,
                  'bg-yellow-100 text-yellow-800': syncStatus.isConfigured && !syncStatus.syncEnabled,
                  'bg-gray-100 text-gray-800': !syncStatus.isConfigured,
                }"
              >
                {{ syncStatus.isConfigured && syncStatus.syncEnabled ? 'Aktiv' : syncStatus.isConfigured ? 'Pausiert' : 'Nicht konfiguriert' }}
              </span>
            </div>

            <div v-if="syncStatus.lastSyncAt" class="flex items-center justify-between">
              <span class="text-sm text-gray-600">Letzte Synchronisation:</span>
              <span class="text-sm font-medium">{{ formatDate(syncStatus.lastSyncAt) }}</span>
            </div>

            <div v-if="syncStatus.lastSyncError" class="p-3 bg-red-50 border border-red-200 rounded-lg">
              <div class="flex items-center gap-2">
                <AlertTriangle class="w-4 h-4 text-red-600" />
                <span class="text-sm text-red-800">{{ syncStatus.lastSyncError }}</span>
              </div>
            </div>

            <hr class="border-gray-200" />

            <button
              v-if="syncStatus.isConfigured"
              @click="triggerSync"
              :disabled="isSyncing"
              class="w-full flex items-center justify-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Loader2 v-if="isSyncing" class="w-4 h-4 animate-spin" />
              <RefreshCw v-else class="w-4 h-4" />
              {{ isSyncing ? 'Synchronisiere...' : 'Jetzt synchronisieren' }}
            </button>
          </div>
        </div>

        <div class="bg-blue-50 border border-blue-200 rounded-lg p-4">
          <h3 class="text-sm font-medium text-blue-900 mb-2">Hinweise</h3>
          <ul class="text-sm text-blue-800 space-y-1 list-disc list-inside">
            <li>Die PIN wird verschlüsselt gespeichert</li>
            <li>Bei regelmäßigem Abruf ist keine 2FA nötig</li>
            <li>Verwende einen Cron-Job für täglichen Sync</li>
          </ul>
        </div>
      </div>
    </div>
  </div>
</template>
