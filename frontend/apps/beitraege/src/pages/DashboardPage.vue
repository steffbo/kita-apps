<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue';
import { api } from '@/api';
import type { FeeOverview, ReminderRunResponse, EmailLog } from '@/api/types';
import {
  Receipt,
  CheckCircle,
  Clock,
  AlertTriangle,
  TrendingUp,
  Loader2,
  Users,
  Link2,
} from 'lucide-vue-next';
import { useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';

const router = useRouter();
const authStore = useAuthStore();
const overview = ref<FeeOverview | null>(null);
const unmatchedTotal = ref(0);
const isLoading = ref(true);
const error = ref<string | null>(null);
const selectedYear = ref(new Date().getFullYear());

const reminderAutoEnabled = ref(false);
const isReminderSettingsLoading = ref(false);
const reminderSettingsError = ref<string | null>(null);
const reminderRunError = ref<string | null>(null);
const reminderRunResult = ref<ReminderRunResponse | null>(null);
const reminderDate = ref(new Date().toLocaleDateString('en-CA'));
const reminderDryRun = ref(true);
const isRunningReminders = ref(false);

const emailLogs = ref<EmailLog[]>([]);
const emailLogsTotal = ref(0);
const emailLogsOffset = ref(0);
const emailLogsLimit = 20;
const isEmailLogsLoading = ref(false);
const emailLogsError = ref<string | null>(null);

const years = computed(() => {
  const currentYear = new Date().getFullYear();
  return [currentYear - 1, currentYear, currentYear + 1];
});

async function loadOverview() {
  isLoading.value = true;
  error.value = null;
  try {
    const [overviewData, unmatchedData] = await Promise.all([
      api.getFeeOverview(selectedYear.value),
      api.getUnmatchedTransactions({ limit: 1 }),
    ]);
    overview.value = overviewData;
    unmatchedTotal.value = unmatchedData.total;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoading.value = false;
  }
}

onMounted(loadOverview);
onMounted(loadReminderSettings);
onMounted(() => {
  if (authStore.isAdmin) {
    loadEmailLogs(true);
  }
});
watch(
  () => authStore.isAdmin,
  (isAdmin) => {
    if (isAdmin) {
      loadEmailLogs(true);
    }
  }
);

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: 'EUR',
  }).format(amount);
}

function getMonthName(month: number): string {
  return new Date(2000, month - 1).toLocaleString('de-DE', { month: 'short' });
}

const stats = computed(() => {
  if (!overview.value) return [];
  return [
    {
      name: 'Offene Beiträge',
      value: overview.value.totalOpen,
      amount: formatCurrency(overview.value.amountOpen),
      icon: Clock,
      color: 'text-blue-600',
      bgColor: 'bg-blue-100',
    },
    {
      name: 'Bezahlte Beiträge',
      value: overview.value.totalPaid,
      amount: formatCurrency(overview.value.amountPaid),
      icon: CheckCircle,
      color: 'text-green-600',
      bgColor: 'bg-green-100',
    },
    {
      name: 'Überfällige Beiträge',
      value: overview.value.totalOverdue,
      amount: formatCurrency(overview.value.amountOverdue),
      icon: AlertTriangle,
      color: 'text-red-600',
      bgColor: 'bg-red-100',
    },
    {
      name: 'Gesamtbetrag',
      value: overview.value.totalOpen + overview.value.totalPaid + overview.value.totalOverdue,
      amount: formatCurrency(
        overview.value.amountOpen + overview.value.amountPaid + overview.value.amountOverdue
      ),
      icon: Receipt,
      color: 'text-primary',
      bgColor: 'bg-primary/10',
    },
  ];
});

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
</script>

<template>
  <div>
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-8">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">Dashboard</h1>
        <p class="text-gray-600 mt-1">Übersicht der Beitragszahlungen</p>
      </div>
      <div class="flex items-center gap-2">
        <label for="year" class="text-sm font-medium text-gray-700">Jahr:</label>
        <select
          id="year"
          v-model="selectedYear"
          @change="loadOverview"
          class="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
        >
          <option v-for="year in years" :key="year" :value="year">{{ year }}</option>
        </select>
      </div>
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4">
      <p class="text-red-600">{{ error }}</p>
      <button @click="loadOverview" class="mt-2 text-sm text-red-700 underline">
        Erneut versuchen
      </button>
    </div>

    <!-- Content -->
    <div v-else-if="overview">
      <!-- Stats Grid -->
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        <div
          v-for="stat in stats"
          :key="stat.name"
          class="bg-white rounded-xl border p-6 hover:shadow-md transition-shadow"
        >
          <div class="flex items-center gap-4">
            <div :class="['p-3 rounded-lg', stat.bgColor]">
              <component :is="stat.icon" :class="['h-6 w-6', stat.color]" />
            </div>
            <div>
              <p class="text-sm text-gray-600">{{ stat.name }}</p>
              <p class="text-2xl font-bold text-gray-900">{{ stat.value }}</p>
              <p :class="['text-sm font-medium', stat.color]">{{ stat.amount }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Warning Cards Grid -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-8">
        <!-- Children with Missing Payments Warning Card -->
        <div
          v-if="overview.childrenWithOpenFees > 0"
          class="bg-amber-50 border border-amber-200 rounded-xl p-6 cursor-pointer hover:bg-amber-100 transition-colors"
          @click="router.push('/kinder?openFees=true')"
        >
          <div class="flex items-center gap-4">
            <div class="p-3 bg-amber-100 rounded-lg">
              <Users class="h-6 w-6 text-amber-600" />
            </div>
            <div class="flex-1">
              <p class="text-sm text-amber-700 font-medium">Fehlende Zahlungen</p>
              <p class="text-lg text-amber-900">
                {{ overview.childrenWithOpenFees }} Kinder haben offene Beiträge
              </p>
              <p v-if="overview" class="text-xs text-amber-700 mt-1">
                Vereinsbeitrag: {{ overview.openMembershipCount }}
                · Essensgeld: {{ overview.openFoodCount }}
                · Platzgeld: {{ overview.openChildcareCount }}
              </p>
            </div>
            <div class="text-amber-600">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
              </svg>
            </div>
          </div>
        </div>

        <!-- Unmatched Transactions Warning Card -->
        <div
          v-if="unmatchedTotal > 0"
          class="bg-orange-50 border border-orange-200 rounded-xl p-6 cursor-pointer hover:bg-orange-100 transition-colors"
          @click="router.push('/import?tab=unmatched')"
        >
          <div class="flex items-center gap-4">
            <div class="p-3 bg-orange-100 rounded-lg">
              <Link2 class="h-6 w-6 text-orange-600" />
            </div>
            <div class="flex-1">
              <p class="text-sm text-orange-700 font-medium">Nicht zugeordnet</p>
              <p class="text-lg text-orange-900">
                {{ unmatchedTotal }} Transaktionen nicht zugeordnet
              </p>
            </div>
            <div class="text-orange-600">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
              </svg>
            </div>
          </div>
        </div>
      </div>

      <!-- Reminder Controls -->
      <div v-if="authStore.isAdmin" class="bg-white rounded-xl border p-6 mb-8">
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

      <!-- Email Logs -->
      <div v-if="authStore.isAdmin" class="bg-white rounded-xl border p-6 mb-8">
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

      <!-- Monthly Overview -->
      <div class="bg-white rounded-xl border p-6">
        <div class="flex items-center gap-2 mb-6">
          <TrendingUp class="h-5 w-5 text-primary" />
          <h2 class="text-lg font-semibold">Monatliche Übersicht</h2>
        </div>

        <div v-if="overview.byMonth.length > 0" class="overflow-x-auto">
          <table class="w-full">
            <thead>
              <tr class="text-left text-sm text-gray-500 border-b">
                <th class="pb-3 font-medium">Monat</th>
                <th class="pb-3 font-medium text-right">Offen</th>
                <th class="pb-3 font-medium text-right">Bezahlt</th>
                <th class="pb-3 font-medium text-right">Offen (€)</th>
                <th class="pb-3 font-medium text-right">Bezahlt (€)</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="month in overview.byMonth"
                :key="month.month"
                class="border-b last:border-0 hover:bg-gray-50"
              >
                <td class="py-3 font-medium">{{ getMonthName(month.month) }} {{ month.year }}</td>
                <td class="py-3 text-right">
                  <span
                    v-if="month.openCount > 0"
                    class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-700"
                  >
                    {{ month.openCount }}
                  </span>
                  <span v-else class="text-gray-400">-</span>
                </td>
                <td class="py-3 text-right">
                  <span
                    v-if="month.paidCount > 0"
                    class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-700"
                  >
                    {{ month.paidCount }}
                  </span>
                  <span v-else class="text-gray-400">-</span>
                </td>
                <td class="py-3 text-right text-blue-600">
                  {{ month.openAmount > 0 ? formatCurrency(month.openAmount) : '-' }}
                </td>
                <td class="py-3 text-right text-green-600">
                  {{ month.paidAmount > 0 ? formatCurrency(month.paidAmount) : '-' }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-else class="text-center py-8 text-gray-500">
          Keine Daten für {{ selectedYear }} vorhanden
        </div>
      </div>
    </div>
  </div>
</template>
