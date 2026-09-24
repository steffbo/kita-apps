<script setup lang="ts">
import { ref, computed, watch, onActivated, onUnmounted } from 'vue';
import { api } from '@/api';
import type { EmailLog } from '@/api/types';
import { Eye, Search, ArrowUp, ArrowDown } from 'lucide-vue-next';
import EmailLogModal from '@/components/automation/EmailLogModal.vue';
import { formatDateTime } from '@/utils/format';
import { formatEmailType } from '@/utils/reminders';

// Global log (Versandverlauf); kept alive by the page so filters survive tab switches.
const emailLogs = ref<EmailLog[]>([]);
const emailLogsTotal = ref(0);
const emailLogsPage = ref(1);
const emailLogsPerPage = 20;
const emailLogsSearch = ref('');
const emailLogsTypeFilter = ref('');
const emailLogsSortDir = ref<'asc' | 'desc'>('desc');
const isEmailLogsLoading = ref(false);
const emailLogsError = ref<string | null>(null);
const selectedLog = ref<EmailLog | null>(null);

async function loadEmailLogs(reset = false): Promise<void> {
  if (isEmailLogsLoading.value) return;
  isEmailLogsLoading.value = true;
  emailLogsError.value = null;
  try {
    if (reset) emailLogsPage.value = 1;
    const result = await api.getEmailLogs({
      page: emailLogsPage.value,
      perPage: emailLogsPerPage,
      emailType: emailLogsTypeFilter.value || undefined,
      search: emailLogsSearch.value.trim() || undefined,
      sortDir: emailLogsSortDir.value,
    });
    emailLogs.value = result.data;
    emailLogsTotal.value = result.total;
  } catch (e) {
    emailLogsError.value = e instanceof Error ? e.message : 'Versandverlauf konnte nicht geladen werden';
  } finally {
    isEmailLogsLoading.value = false;
  }
}

const emailLogsTotalPages = computed(() => Math.max(1, Math.ceil(emailLogsTotal.value / emailLogsPerPage)));

function goToEmailLogsPage(target: number): void {
  const page = Math.min(Math.max(1, target), emailLogsTotalPages.value);
  if (page === emailLogsPage.value) return;
  emailLogsPage.value = page;
  loadEmailLogs();
}

let emailLogsSearchTimeout: ReturnType<typeof setTimeout> | null = null;

watch(emailLogsSearch, () => {
  if (emailLogsSearchTimeout) clearTimeout(emailLogsSearchTimeout);
  emailLogsSearchTimeout = setTimeout(() => {
    loadEmailLogs(true);
  }, 300);
});

watch([emailLogsTypeFilter, emailLogsSortDir], () => {
  loadEmailLogs(true);
});

// Runs on first mount and whenever the tab is shown again
onActivated(() => {
  loadEmailLogs(true);
});

onUnmounted(() => {
  if (emailLogsSearchTimeout) clearTimeout(emailLogsSearchTimeout);
});

function toggleEmailLogsSort(): void {
  emailLogsSortDir.value = emailLogsSortDir.value === 'desc' ? 'asc' : 'desc';
}
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-4">
      <p class="text-sm text-gray-600">Alle versendeten E-Mails inklusive Inhalt.</p>
      <button class="text-sm text-primary hover:underline" :disabled="isEmailLogsLoading" @click="loadEmailLogs(true)">
        Neu laden
      </button>
    </div>

    <div v-if="emailLogsError" class="text-sm text-red-600 mb-3">{{ emailLogsError }}</div>

    <div class="flex flex-col sm:flex-row sm:items-center gap-2 mb-4">
      <select
        v-model="emailLogsTypeFilter"
        class="px-3 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
      >
        <option value="">Alle Typen</option>
        <option value="REMINDER_INITIAL">Zahlungserinnerung</option>
        <option value="REMINDER_FINAL">Mahnung</option>
        <option value="MEMBERSHIP_REMINDER_INITIAL">Vereinsbeitrag Erinnerung</option>
        <option value="MEMBERSHIP_REMINDER_FINAL">Vereinsbeitrag Mahnung</option>
        <option value="PASSWORD_RESET">Passwort-Reset</option>
      </select>

      <div class="relative flex-1 min-w-[200px]">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
        <input
          v-model="emailLogsSearch"
          type="text"
          placeholder="Suche nach Empfänger oder Betreff..."
          class="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
        />
      </div>

      <button
        type="button"
        class="inline-flex items-center gap-1.5 px-3 py-2 text-sm border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
        @click="toggleEmailLogsSort"
        :title="emailLogsSortDir === 'desc' ? 'Älteste zuerst' : 'Neueste zuerst'"
      >
        {{ emailLogsSortDir === 'desc' ? 'Neueste zuerst' : 'Älteste zuerst' }}
        <ArrowDown v-if="emailLogsSortDir === 'desc'" class="h-4 w-4" />
        <ArrowUp v-else class="h-4 w-4" />
      </button>
    </div>

    <div v-if="emailLogs.length === 0 && !isEmailLogsLoading" class="text-sm text-gray-500">
      Keine E-Mails für diese Filter gefunden.
    </div>

    <div v-else class="overflow-x-auto">
      <table class="w-full table-fixed text-sm">
        <thead>
          <tr class="text-left text-gray-500 border-b">
            <th class="w-36 pb-3 font-medium">Zeitpunkt</th>
            <th class="w-44 pb-3 font-medium">Typ</th>
            <th class="pb-3 font-medium">Empfänger</th>
            <th class="pb-3 font-medium">Betreff</th>
            <th class="w-32 pb-3 font-medium">Inhalt</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="log in emailLogs" :key="log.id" class="border-b last:border-0 align-top">
            <td class="py-3 whitespace-nowrap">{{ formatDateTime(log.sentAt) }}</td>
            <td class="py-3 whitespace-nowrap">{{ formatEmailType(log.emailType) }}</td>
            <td class="py-3 pr-4 truncate" :title="log.toEmail">{{ log.toEmail }}</td>
            <td class="py-3 pr-4 truncate" :title="log.subject">{{ log.subject }}</td>
            <td class="py-3">
              <button
                type="button"
                class="inline-flex items-center gap-1.5 text-primary hover:underline"
                @click="selectedLog = log"
              >
                <Eye class="h-4 w-4" />
                Anzeigen
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="isEmailLogsLoading" class="mt-3 text-sm text-gray-500">Versandverlauf wird geladen...</div>

    <div
      v-if="emailLogsTotalPages > 1 || emailLogsPage > 1"
      class="flex items-center justify-between mt-4 pt-4 border-t"
    >
      <p class="text-sm text-gray-600">
        Seite {{ emailLogsPage }} von {{ emailLogsTotalPages }} ({{ emailLogsTotal }} Einträge)
      </p>
      <div class="flex items-center gap-2">
        <button
          type="button"
          class="px-3 py-1.5 text-sm border rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="emailLogsPage <= 1 || isEmailLogsLoading"
          @click="goToEmailLogsPage(emailLogsPage - 1)"
        >
          Zurück
        </button>
        <button
          type="button"
          class="px-3 py-1.5 text-sm border rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="emailLogsPage >= emailLogsTotalPages || isEmailLogsLoading"
          @click="goToEmailLogsPage(emailLogsPage + 1)"
        >
          Weiter
        </button>
      </div>
    </div>

    <EmailLogModal v-if="selectedLog" :log="selectedLog" @close="selectedLog = null" />
  </div>
</template>
