<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue';
import { api } from '@/api';
import type {
  EmailLog,
  ReminderCase,
  ReminderCaseFee,
  ReminderCasePreview,
  ReminderCaseSendResult,
  ReminderCaseStage,
} from '@/api/types';
import { ReminderCaseConflictError } from '@/api/types';
import { Eye, X, Search, ArrowUp, ArrowDown, ArrowLeft, Settings, Mail, Clock } from 'lucide-vue-next';
import { useAuthStore } from '@/stores/auth';

const authStore = useAuthStore();

// ── Page tabs ────────────────────────────────────────────────────────────────
const activeTab = ref<'worklist' | 'log'>('worklist');

// ── Worklist state ───────────────────────────────────────────────────────────
const scope = ref<'actionable' | 'all'>('actionable');
const cases = ref<ReminderCase[]>([]);
const isCasesLoading = ref(false);
const casesError = ref<string | null>(null);
const caseSearch = ref('');
const selectedHouseholdId = ref<string | null>(null);

const filteredCases = computed(() => {
  const term = caseSearch.value.trim().toLowerCase();
  if (!term) return cases.value;
  return cases.value.filter((item) => item.householdName.toLowerCase().includes(term));
});

const selectedCase = computed(() => {
  if (!selectedHouseholdId.value) return null;
  return cases.value.find((item) => item.householdId === selectedHouseholdId.value) ?? null;
});

function lastContactOf(item: ReminderCase): string | null {
  let latest: string | null = null;
  for (const fee of item.fees) {
    const at = fee.lastContact?.lastContactAt;
    if (at && (!latest || at > latest)) latest = at;
  }
  return latest;
}

function hasBlockedEmail(item: ReminderCase): boolean {
  return !item.recipients || item.recipients.length === 0;
}

async function loadCases(selectId: string | null = null): Promise<void> {
  if (!authStore.isAdmin) return;
  isCasesLoading.value = true;
  casesError.value = null;
  try {
    const result = await api.getReminderCases({ scope: scope.value });
    cases.value = result.cases;
    if (selectId && cases.value.some((item) => item.householdId === selectId)) {
      selectedHouseholdId.value = selectId;
    } else if (selectedHouseholdId.value && !cases.value.some((item) => item.householdId === selectedHouseholdId.value)) {
      selectedHouseholdId.value = null;
    }
  } catch (e) {
    casesError.value = e instanceof Error ? e.message : 'Familien konnten nicht geladen werden';
  } finally {
    isCasesLoading.value = false;
  }
}

function openCase(householdId: string): void {
  selectedHouseholdId.value = householdId;
  const item = selectedCase.value;
  const actionableIds = (item?.fees ?? [])
    .filter((fee) => fee.status === 'actionable_initial' || fee.status === 'actionable_final' || fee.status === 'history_unknown')
    .map((fee) => fee.feeId);
  selectedFeeIds.value = actionableIds;
  stage.value = deriveRecommendedStage(actionableIds, item?.fees ?? []) ?? 'initial';
  sendResult.value = null;
  sendError.value = null;
  resetNotice.value = false;
  userEdited.value = false;
  loadChronology();
}

function closeCase(): void {
  selectedHouseholdId.value = null;
}

// ── Case detail state ────────────────────────────────────────────────────────
const selectedFeeIds = ref<string[]>([]);
const stage = ref<ReminderCaseStage>('initial');
const includeQR = ref(true);
const preview = ref<ReminderCasePreview | null>(null);
const isPreviewLoading = ref(false);
const previewError = ref<string | null>(null);
const subjectEdit = ref('');
const bodyEdit = ref('');
const userEdited = ref(false);
const resetNotice = ref(false);
const isSending = ref(false);
const sendError = ref<string | null>(null);
const conflictFeeCount = ref(0);
const sendResult = ref<ReminderCaseSendResult | null>(null);
const showConfirmModal = ref(false);
const showSettingsDialog = ref(false);

let previewRequestedAt: string | null = null;
let previewTimer: ReturnType<typeof setTimeout> | null = null;

// Derive the recommendation the way the backend does: initial only when every
// selected fee is actionable_initial, final only when every one is
// actionable_final.
function deriveRecommendedStage(selectedIds: string[], fees: ReminderCaseFee[]): ReminderCaseStage | null {
  if (selectedIds.length === 0) return null;
  const byId = new Map<string, ReminderCaseFee>(fees.map((fee) => [fee.feeId, fee]));
  const selected = selectedIds
    .map((id) => byId.get(id))
    .filter((fee): fee is ReminderCaseFee => !!fee);
  if (selected.length === 0) return null;
  const allInitial = selected.every((fee) => fee.status === 'actionable_initial');
  const allFinal = selected.every((fee) => fee.status === 'actionable_final');
  if (allInitial) return 'initial';
  if (allFinal) return 'final';
  return null;
}

const recommendedStage = computed<ReminderCaseStage | null>(() => {
  if (!selectedCase.value) return null;
  return deriveRecommendedStage(selectedFeeIds.value, selectedCase.value.fees);
});

const stageWarning = computed<string | null>(() => {
  const recommended = recommendedStage.value;
  if (!recommended || recommended === stage.value) return null;
  if (stage.value === 'final') {
    return 'Empfehlung: Erinnerung — mindestens ein ausgewählter Beitrag wurde noch nicht erinnert.';
  }
  return 'Empfehlung: Mahnung — alle ausgewählten Beiträge haben eine abgelaufene Frist.';
});

function toggleFee(feeId: string): void {
  const index = selectedFeeIds.value.indexOf(feeId);
  if (index >= 0) {
    selectedFeeIds.value.splice(index, 1);
  } else {
    selectedFeeIds.value.push(feeId);
  }
}

async function refreshPreview(): Promise<void> {
  const item = selectedCase.value;
  if (!item || selectedFeeIds.value.length === 0) {
    preview.value = null;
    return;
  }
  if (previewTimer) clearTimeout(previewTimer);
  isPreviewLoading.value = true;
  previewError.value = null;
  previewRequestedAt = new Date().toISOString();
  try {
    const result = await api.previewReminderCase(item.householdId, {
      stage: stage.value,
      runDate: todayISO(),
      feeIds: selectedFeeIds.value,
      includeQR: includeQR.value,
    });
    const hadEdits = userEdited.value;
    preview.value = result;
    subjectEdit.value = result.subject;
    bodyEdit.value = result.body;
    if (hadEdits) {
      resetNotice.value = true;
    }
    userEdited.value = false;
  } catch (e) {
    previewError.value = e instanceof Error ? e.message : 'Vorschau konnte nicht geladen werden';
  } finally {
    isPreviewLoading.value = false;
  }
}

function schedulePreviewRefresh(): void {
  if (previewTimer) clearTimeout(previewTimer);
  previewTimer = setTimeout(() => {
    void refreshPreview();
  }, 250);
}

watch(selectedFeeIds, () => {
  resetNotice.value = false;
  schedulePreviewRefresh();
}, { deep: true });

watch(stage, () => {
  resetNotice.value = false;
  schedulePreviewRefresh();
});

watch(includeQR, () => {
  schedulePreviewRefresh();
});

function onSubjectInput(): void {
  userEdited.value = true;
  resetNotice.value = false;
}

function onBodyInput(): void {
  userEdited.value = true;
  resetNotice.value = false;
}

function resetEdits(): void {
  if (!preview.value) return;
  subjectEdit.value = preview.value.subject;
  bodyEdit.value = preview.value.body;
  userEdited.value = false;
  resetNotice.value = false;
}

async function openSendConfirmation(): Promise<void> {
  if (!preview.value) return;
  sendError.value = null;
  conflictFeeCount.value = 0;
  showConfirmModal.value = true;
}

async function confirmSend(): Promise<void> {
  const item = selectedCase.value;
  if (!item || !preview.value) return;
  isSending.value = true;
  sendError.value = null;
  conflictFeeCount.value = 0;
  try {
    const result = await api.sendReminderCase(item.householdId, {
      stage: stage.value,
      runDate: todayISO(),
      feeIds: selectedFeeIds.value,
      includeQR: includeQR.value,
      ...(subjectEdit.value.trim() !== '' && subjectEdit.value !== preview.value.subject ? { subject: subjectEdit.value } : {}),
      ...(bodyEdit.value !== preview.value.body ? { body: bodyEdit.value } : {}),
      ...(previewRequestedAt ? { previewedAt: previewRequestedAt } : {}),
    });
    sendResult.value = result;
    showConfirmModal.value = false;
    preview.value = null;
    await loadCases();
    // Open the next actionable family, if any.
    const next = cases.value.find((entry) => entry.householdId !== item.householdId);
    if (next) {
      openCase(next.householdId);
    } else {
      selectedHouseholdId.value = null;
    }
  } catch (e) {
    showConfirmModal.value = false;
    if (e instanceof ReminderCaseConflictError) {
      conflictFeeCount.value = e.feeIds.length;
      sendError.value = `Zustand hat sich geändert (${e.message}). Bitte Vorschau neu prüfen.`;
      await loadCases(item.householdId);
      await refreshPreview();
    } else {
      sendError.value = e instanceof Error ? e.message : 'Versand fehlgeschlagen';
    }
  } finally {
    isSending.value = false;
  }
}

// ── Family chronology ────────────────────────────────────────────────────────
const chronology = ref<EmailLog[]>([]);
const isChronologyLoading = ref(false);
const selectedChronologyLog = ref<EmailLog | null>(null);

async function loadChronology(): Promise<void> {
  const item = selectedCase.value;
  if (!item) return;
  isChronologyLoading.value = true;
  try {
    const result = await api.getEmailLogs({ householdId: item.householdId, perPage: 20, sortDir: 'desc' });
    chronology.value = result.data;
  } catch {
    chronology.value = [];
  } finally {
    isChronologyLoading.value = false;
  }
}

// ── Settings (payment data) ──────────────────────────────────────────────────
const reminderAutoEnabled = ref(false);
const reminderPaymentRecipientName = ref('');
const reminderPaymentIBAN = ref('');
const reminderPaymentBIC = ref('');
const isReminderSettingsLoading = ref(false);
const reminderSettingsError = ref<string | null>(null);

async function loadReminderSettings(): Promise<void> {
  if (!authStore.isAdmin) return;
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
  if (!authStore.isAdmin) return;
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
    showSettingsDialog.value = false;
  } catch (e) {
    reminderSettingsError.value = e instanceof Error ? e.message : 'Einstellungen konnten nicht gespeichert werden';
  } finally {
    isReminderSettingsLoading.value = false;
  }
}

// ── Global log (Versandverlauf) ──────────────────────────────────────────────
const emailLogs = ref<EmailLog[]>([]);
const emailLogsTotal = ref(0);
const emailLogsPage = ref(1);
const emailLogsPerPage = 20;
const emailLogsSearch = ref('');
const emailLogsTypeFilter = ref('');
const emailLogsSortDir = ref<'asc' | 'desc'>('desc');
const isEmailLogsLoading = ref(false);
const emailLogsError = ref<string | null>(null);
const selectedEmailLog = ref<EmailLog | null>(null);

async function loadEmailLogs(reset = false): Promise<void> {
  if (!authStore.isAdmin) return;
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

watch(activeTab, (tab) => {
  if (tab === 'log') loadEmailLogs(true);
});

function toggleEmailLogsSort(): void {
  emailLogsSortDir.value = emailLogsSortDir.value === 'desc' ? 'asc' : 'desc';
}

// ── Formatting helpers ───────────────────────────────────────────────────────
function todayISO(): string {
  return new Date().toLocaleDateString('en-CA');
}

function formatCurrency(value: number): string {
  return value.toLocaleString('de-DE', { style: 'currency', currency: 'EUR' });
}

function formatDate(value: string | undefined): string {
  if (!value) return '—';
  return new Date(value).toLocaleDateString('de-DE');
}

function formatDateTime(value: string): string {
  return new Date(value).toLocaleString('de-DE');
}

function formatPeriod(fee: ReminderCaseFee): string {
  if (fee.month > 0) return `${fee.month}/${fee.year}`;
  return String(fee.year);
}

const feeTypeLabels: Record<string, string> = {
  MEMBERSHIP: 'Vereinsbeitrag',
  FOOD: 'Essensgeld',
  CHILDCARE: 'Platzgeld',
  REMINDER: 'Mahngebühr',
};

function feeTypeLabel(feeType: string): string {
  return feeTypeLabels[feeType] ?? feeType;
}

function feeChipClass(feeType: string): string {
  switch (feeType) {
    case 'MEMBERSHIP':
      return 'bg-purple-100 text-purple-700';
    case 'FOOD':
      return 'bg-orange-100 text-orange-700';
    case 'CHILDCARE':
      return 'bg-blue-100 text-blue-700';
    case 'REMINDER':
      return 'bg-red-100 text-red-700';
    default:
      return 'bg-gray-100 text-gray-700';
  }
}

function feeTypesIn(item: ReminderCase): string[] {
  const types = new Set(item.fees.map((fee) => fee.feeType));
  return Array.from(types);
}

const statusLabels: Record<string, string> = {
  actionable_initial: 'Erinnerung fällig',
  actionable_final: 'Mahnung fällig',
  waiting: 'In Frist',
  never_contacted: 'Nicht fällig',
  history_unknown: 'Historie unbekannt',
};

function statusLabel(status: string): string {
  return statusLabels[status] ?? status;
}

function statusBadgeClass(status: string): string {
  switch (status) {
    case 'actionable_initial':
      return 'bg-amber-100 text-amber-700';
    case 'actionable_final':
      return 'bg-red-100 text-red-700';
    case 'waiting':
      return 'bg-blue-100 text-blue-700';
    case 'history_unknown':
      return 'bg-gray-200 text-gray-700';
    default:
      return 'bg-gray-100 text-gray-600';
  }
}

function formatEmailType(type: string): string {
  switch (type) {
    case 'REMINDER_INITIAL':
      return 'Zahlungserinnerung';
    case 'REMINDER_FINAL':
      return 'Mahnung';
    case 'MEMBERSHIP_REMINDER_INITIAL':
      return 'Vereinsbeitrag Erinnerung';
    case 'MEMBERSHIP_REMINDER_FINAL':
      return 'Vereinsbeitrag Mahnung';
    case 'PASSWORD_RESET':
      return 'Passwort-Reset';
    default:
      return type;
  }
}

// ── Lifecycle ────────────────────────────────────────────────────────────────
onMounted(() => {
  if (authStore.isAdmin) {
    loadCases();
    loadReminderSettings();
  }
});

onUnmounted(() => {
  if (emailLogsSearchTimeout) clearTimeout(emailLogsSearchTimeout);
  if (previewTimer) clearTimeout(previewTimer);
});

watch(
  () => authStore.isAdmin,
  (isAdmin) => {
    if (isAdmin) {
      loadCases();
      loadReminderSettings();
    }
  }
);
</script>

<template>
  <div>
    <!-- Header -->
    <div class="mb-6 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">Erinnerungen</h1>
        <p class="text-gray-600 mt-1">Familien mit offenen Beiträgen bearbeiten</p>
      </div>
      <button
        class="inline-flex items-center gap-2 px-3 py-2 rounded-lg border text-sm font-medium hover:bg-gray-50"
        @click="showSettingsDialog = true"
      >
        <Settings class="h-4 w-4" />
        Zahlungsdaten
      </button>
    </div>

    <!-- Tab switch -->
    <div class="flex gap-1 mb-6 border-b">
      <button
        class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors"
        :class="activeTab === 'worklist' ? 'border-primary text-primary' : 'border-transparent text-gray-500 hover:text-gray-700'"
        @click="activeTab = 'worklist'"
      >
        Arbeitsliste
      </button>
      <button
        class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors"
        :class="activeTab === 'log' ? 'border-primary text-primary' : 'border-transparent text-gray-500 hover:text-gray-700'"
        @click="activeTab = 'log'"
      >
        Versandverlauf
      </button>
    </div>

    <!-- ════════════════════ Worklist ════════════════════ -->
    <div v-if="activeTab === 'worklist'" v-show="authStore.isAdmin">
      <!-- Scope + search -->
      <div class="flex flex-col sm:flex-row sm:items-center gap-2 mb-4">
        <div class="inline-flex rounded-lg border overflow-hidden">
          <button
            class="px-4 py-2 text-sm font-medium transition-colors"
            :class="scope === 'actionable' ? 'bg-primary text-white' : 'bg-white text-gray-600 hover:bg-gray-50'"
            @click="scope = 'actionable'"
          >
            Handlungsbedarf
          </button>
          <button
            class="px-4 py-2 text-sm font-medium transition-colors"
            :class="scope === 'all' ? 'bg-primary text-white' : 'bg-white text-gray-600 hover:bg-gray-50'"
            @click="scope = 'all'"
          >
            Alle offenen
          </button>
        </div>
        <div class="relative flex-1 min-w-[200px]">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
          <input
            v-model="caseSearch"
            type="text"
            placeholder="Familie suchen..."
            class="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          />
        </div>
        <button class="text-sm text-primary hover:underline" :disabled="isCasesLoading" @click="loadCases()">
          Neu laden
        </button>
      </div>

      <div v-if="casesError" class="mb-4 text-sm text-red-600">{{ casesError }}</div>
      <div v-if="isCasesLoading && cases.length === 0" class="text-sm text-gray-500">Familien werden geladen...</div>
      <div v-else-if="filteredCases.length === 0" class="text-sm text-gray-500 py-8 text-center">
        Keine offenen Fälle in dieser Ansicht.
      </div>

      <!-- Master-detail -->
      <div :class="selectedCase ? 'lg:grid lg:grid-cols-[minmax(300px,2fr)_minmax(0,3fr)] lg:gap-6' : ''">
        <!-- Family list -->
        <div :class="selectedCase ? 'hidden lg:block' : ''">
          <ul class="space-y-2">
            <li v-for="item in filteredCases" :key="item.householdId">
              <button
                class="w-full text-left p-4 border rounded-xl transition-colors hover:bg-gray-50"
                :class="item.householdId === selectedHouseholdId ? 'border-primary ring-1 ring-primary' : 'border-gray-200'"
                @click="openCase(item.householdId)"
              >
                <div class="flex items-center justify-between gap-3">
                  <span class="font-medium text-gray-900">{{ item.householdName }}</span>
                  <span class="font-semibold text-gray-900 whitespace-nowrap">{{ formatCurrency(item.totalRemaining) }}</span>
                </div>
                <div class="flex flex-wrap items-center gap-1.5 mt-2">
                  <span
                    v-for="feeType in feeTypesIn(item)"
                    :key="feeType"
                    class="px-2 py-0.5 text-xs rounded-full font-medium"
                    :class="feeChipClass(feeType)"
                  >
                    {{ feeTypeLabel(feeType) }}
                  </span>
                  <span
                    v-if="hasBlockedEmail(item)"
                    class="px-2 py-0.5 text-xs rounded-full font-medium bg-red-600 text-white"
                  >
                    Keine E-Mail
                  </span>
                </div>
                <div class="flex flex-wrap items-center gap-4 mt-2 text-xs text-gray-500">
                  <span class="inline-flex items-center gap-1">
                    <Clock class="h-3.5 w-3.5" />
                    Nächste Aktion: {{ formatDate(item.nextActionAt) }}
                  </span>
                  <span v-if="lastContactOf(item)" class="inline-flex items-center gap-1">
                    <Mail class="h-3.5 w-3.5" />
                    Letzter Kontakt: {{ formatDate(lastContactOf(item) ?? undefined) }}
                  </span>
                </div>
              </button>
            </li>
          </ul>
        </div>

        <!-- Detail panel (desktop) / full view (mobile) -->
        <div v-if="selectedCase" class="mt-6 lg:mt-0">
          <div class="lg:sticky lg:top-6">
            <!-- Mobile back -->
            <button
              class="inline-flex items-center gap-2 text-sm text-gray-600 hover:text-gray-900 mb-3 lg:hidden"
              @click="closeCase"
            >
              <ArrowLeft class="h-4 w-4" />
              Zurück zur Liste
            </button>

            <div class="bg-white border rounded-xl p-5">
              <!-- Success banner -->
              <div v-if="sendResult" class="mb-4 p-4 bg-green-50 border border-green-200 rounded-lg text-sm">
                <p class="font-medium text-green-800">E-Mail gesendet an {{ sendResult.sentTo.join(', ') }}</p>
                <p class="text-green-700 mt-1">
                  Frist: {{ formatDate(sendResult.deadline) }}
                  <template v-if="sendResult.createdReminderFees.length > 0">
                    · Mahngebühren erstellt: {{ sendResult.createdReminderFees.length }}
                  </template>
                </p>
              </div>

              <div class="flex items-start justify-between gap-3 mb-4">
                <div>
                  <h2 class="text-lg font-semibold text-gray-900">{{ selectedCase.householdName }}</h2>
                  <p class="text-sm text-gray-500">
                    Empfänger:
                    <template v-if="selectedCase.recipients.length > 0">{{ selectedCase.recipients.join(', ') }}</template>
                    <template v-else><span class="text-red-600 font-medium">keine gültige E-Mail-Adresse</span></template>
                  </p>
                </div>
                <div class="text-right shrink-0">
                  <p class="text-xs text-gray-500">Offen gesamt</p>
                  <p class="font-semibold text-gray-900">{{ formatCurrency(selectedCase.totalRemaining) }}</p>
                </div>
              </div>

              <!-- Stage selector -->
              <div class="flex flex-wrap items-center gap-2 mb-3">
                <button
                  class="px-4 py-2 text-sm font-medium rounded-lg border transition-colors"
                  :class="stage === 'initial' ? 'bg-primary text-white border-primary' : 'border-gray-300 text-gray-700 hover:bg-gray-50'"
                  @click="stage = 'initial'"
                >
                  Erinnerung
                </button>
                <button
                  class="px-4 py-2 text-sm font-medium rounded-lg border transition-colors"
                  :class="stage === 'final' ? 'bg-amber-600 text-white border-amber-600' : 'border-gray-300 text-gray-700 hover:bg-gray-50'"
                  @click="stage = 'final'"
                >
                  Mahnung
                </button>
                <label class="inline-flex items-center gap-2 text-sm text-gray-700 ml-2">
                  <input type="checkbox" v-model="includeQR" />
                  QR-Code
                </label>
              </div>
              <p v-if="stageWarning" class="text-xs text-amber-700 mb-3">{{ stageWarning }}</p>
              <p v-else-if="recommendedStage" class="text-xs text-gray-500 mb-3">
                Empfehlung: {{ recommendedStage === 'final' ? 'Mahnung' : 'Erinnerung' }}
              </p>

              <!-- Fee selection -->
              <div class="border rounded-lg overflow-hidden mb-4">
                <div class="overflow-x-auto">
                  <table class="w-full text-sm">
                    <thead>
                      <tr class="text-left text-gray-500 border-b bg-gray-50">
                        <th class="w-8 py-2 pl-3"></th>
                        <th class="py-2 pr-3 font-medium">Kind / Beitrag</th>
                        <th class="py-2 pr-3 font-medium">Zeitraum</th>
                        <th class="py-2 pr-3 font-medium">Fällig</th>
                        <th class="py-2 pr-3 font-medium text-right">Soll</th>
                        <th class="py-2 pr-3 font-medium text-right">Offen</th>
                        <th class="py-2 pr-3 font-medium">Status</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr
                        v-for="fee in selectedCase.fees"
                        :key="fee.feeId"
                        class="border-b last:border-0"
                      >
                        <td class="py-2 pl-3">
                          <input
                            type="checkbox"
                            :checked="selectedFeeIds.includes(fee.feeId)"
                            @change="toggleFee(fee.feeId)"
                          />
                        </td>
                        <td class="py-2 pr-3">
                          <div class="flex items-center gap-2">
                            <span>{{ fee.childName }}</span>
                            <span class="px-2 py-0.5 text-xs rounded-full font-medium" :class="feeChipClass(fee.feeType)">
                              {{ feeTypeLabel(fee.feeType) }}
                            </span>
                          </div>
                        </td>
                        <td class="py-2 pr-3 whitespace-nowrap">{{ formatPeriod(fee) }}</td>
                        <td class="py-2 pr-3 whitespace-nowrap">{{ formatDate(fee.dueDate) }}</td>
                        <td class="py-2 pr-3 text-right whitespace-nowrap">{{ formatCurrency(fee.amount) }}</td>
                        <td class="py-2 pr-3 text-right font-medium whitespace-nowrap">{{ formatCurrency(fee.remaining) }}</td>
                        <td class="py-2 pr-3">
                          <span class="px-2 py-0.5 text-xs rounded-full font-medium whitespace-nowrap" :class="statusBadgeClass(fee.status)">
                            {{ statusLabel(fee.status) }}
                          </span>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </div>

              <!-- Warnings -->
              <div
                v-if="preview && preview.warnings && preview.warnings.length > 0"
                class="p-3 bg-amber-50 border border-amber-300 rounded-lg text-sm text-amber-900 mb-4"
              >
                <p class="font-semibold mb-1">Hinweise</p>
                <ul class="space-y-0.5">
                  <li v-for="(warning, index) in preview.warnings" :key="index">· {{ warning }}</li>
                </ul>
              </div>

              <!-- Send error -->
              <div v-if="sendError" class="p-3 bg-red-50 border border-red-300 rounded-lg text-sm text-red-800 mb-4">
                {{ sendError }}
                <span v-if="conflictFeeCount > 0" class="block text-xs mt-1">
                  Betroffene Beiträge: {{ conflictFeeCount }} — bitte Auswahl prüfen.
                </span>
              </div>

              <!-- Preview -->
              <div v-if="selectedFeeIds.length === 0" class="text-sm text-gray-500 mb-4">
                Bitte mindestens einen Beitrag auswählen.
              </div>
              <div v-else-if="isPreviewLoading && !preview" class="text-sm text-gray-500 mb-4">Vorschau wird erstellt...</div>
              <div v-else-if="previewError" class="text-sm text-red-600 mb-4">{{ previewError }}</div>
              <div v-else-if="preview" class="border rounded-lg p-4 mb-4 bg-gray-50">
                <div class="flex flex-wrap items-center justify-between gap-2 mb-3">
                  <p class="text-sm font-medium text-gray-800">
                    Vorschau · Frist: <span class="font-semibold">{{ formatDate(preview.deadline) }}</span>
                    · Gesamtbetrag: <span class="font-semibold">{{ formatCurrency(preview.totalAmount) }}</span>
                  </p>
                  <p v-if="resetNotice" class="text-xs text-amber-700">Manuelle Textänderungen wurden zurückgesetzt.</p>
                </div>

                <!-- Planned reminder fees -->
                <div v-if="preview.plannedReminderFees.length > 0" class="mb-3 p-3 bg-white border rounded-lg text-sm">
                  <p class="font-medium text-gray-800 mb-1">Neu entstehende Mahngebühren</p>
                  <ul class="space-y-0.5 text-gray-700">
                    <li v-for="planned in preview.plannedReminderFees" :key="planned.baseFeeId" class="flex justify-between gap-3">
                      <span>Mahngebühr für {{ planned.baseLabel }}</span>
                      <span class="font-medium">{{ formatCurrency(planned.amount) }}</span>
                    </li>
                  </ul>
                </div>

                <div class="space-y-2">
                  <div>
                    <label class="block text-xs text-gray-500 mb-1">Betreff</label>
                    <input
                      type="text"
                      v-model="subjectEdit"
                      @input="onSubjectInput"
                      class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none bg-white"
                    />
                  </div>
                  <div>
                    <label class="block text-xs text-gray-500 mb-1">Text</label>
                    <textarea
                      v-model="bodyEdit"
                      @input="onBodyInput"
                      rows="14"
                      class="w-full px-3 py-2 border border-gray-300 rounded-lg font-mono text-xs focus:ring-2 focus:ring-primary focus:border-transparent outline-none whitespace-pre-wrap bg-white"
                    ></textarea>
                    <p v-if="userEdited" class="mt-1 text-xs text-amber-700">
                      Text angepasst — Änderungen an Auswahl oder Mahnstufe setzen ihn zurück.
                    </p>
                  </div>
                  <div v-if="includeQR && preview.qrImageDataUrl" class="space-y-2">
                    <p class="text-xs text-gray-500">SEPA-QR-Code</p>
                    <img :src="preview.qrImageDataUrl" alt="SEPA QR-Code" class="w-full max-w-[220px] border rounded bg-white p-2" />
                    <details v-if="preview.qrPayload">
                      <summary class="text-xs text-gray-500 cursor-pointer">Im QR-Code enthalten</summary>
                      <pre class="mt-1 whitespace-pre-wrap break-all font-mono text-xs text-gray-600 bg-white border rounded p-3">{{ preview.qrPayload }}</pre>
                    </details>
                  </div>
                  <p v-else-if="!includeQR" class="text-xs text-gray-500">QR-Code ist deaktiviert und wird nicht angehängt.</p>
                </div>
              </div>

              <!-- Actions -->
              <div class="flex flex-wrap justify-end gap-3">
                <button
                  class="px-4 py-2 rounded-lg border text-sm font-medium hover:bg-gray-50"
                  @click="resetEdits"
                  v-if="preview && userEdited"
                >
                  Text zurücksetzen
                </button>
                <button
                  class="px-4 py-2 bg-primary text-white text-sm font-medium rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
                  :disabled="!preview || isPreviewLoading || isSending || selectedCase.recipients.length === 0"
                  @click="openSendConfirmation"
                >
                  {{ stage === 'final' ? 'Mahnung senden' : 'Erinnerung senden' }}
                </button>
              </div>

              <!-- Family chronology -->
              <div class="mt-6 pt-4 border-t">
                <h3 class="text-sm font-semibold text-gray-800 mb-2">Chronik dieser Familie</h3>
                <div v-if="isChronologyLoading" class="text-xs text-gray-500">Wird geladen...</div>
                <div v-else-if="chronology.length === 0" class="text-xs text-gray-500">Noch keine E-Mails an diese Familie gesendet.</div>
                <ul v-else class="divide-y">
                  <li v-for="log in chronology" :key="log.id" class="py-2 flex items-center justify-between gap-3 text-sm">
                    <div class="min-w-0">
                      <span class="text-gray-900">{{ formatDateTime(log.sentAt) }}</span>
                      <span class="text-gray-500"> · {{ formatEmailType(log.emailType) }}</span>
                      <span class="block truncate text-gray-600">{{ log.subject }}</span>
                    </div>
                    <button class="text-primary hover:underline shrink-0 inline-flex items-center gap-1" @click="selectedChronologyLog = log">
                      <Eye class="h-4 w-4" />
                      Anzeigen
                    </button>
                  </li>
                </ul>
              </div>
            </div>
          </div>
        </div>

        <!-- Desktop empty state -->
        <div v-else class="hidden lg:flex items-center justify-center border border-dashed rounded-xl p-12 text-gray-400 text-sm">
          Familie aus der Liste auswählen, um den Entwurf zu erstellen.
        </div>
      </div>
    </div>

    <!-- ════════════════════ Versandverlauf ════════════════════ -->
    <div v-if="activeTab === 'log'" v-show="authStore.isAdmin">
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
                  @click="selectedEmailLog = log"
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
    </div>

    <!-- ════════════════════ Modals ════════════════════ -->

    <!-- Send confirmation -->
    <div
      v-if="showConfirmModal && preview"
      class="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4"
      @click.self="showConfirmModal = false"
    >
      <div class="bg-white rounded-xl shadow-xl w-full max-w-lg max-h-[90vh] flex flex-col">
        <div class="flex items-start justify-between gap-4 p-5 border-b">
          <h3 class="text-lg font-semibold text-gray-900">
            {{ stage === 'final' ? 'Mahnung senden?' : 'Erinnerung senden?' }}
          </h3>
          <button type="button" class="rounded-lg p-2 text-gray-500 hover:bg-gray-100 hover:text-gray-900" @click="showConfirmModal = false">
            <X class="h-5 w-5" />
          </button>
        </div>
        <div class="overflow-y-auto p-5 text-sm space-y-3">
          <dl class="grid grid-cols-2 gap-3">
            <div>
              <dt class="text-gray-500">Familie</dt>
              <dd class="font-medium text-gray-900">{{ selectedCase?.householdName }}</dd>
            </div>
            <div>
              <dt class="text-gray-500">Empfänger</dt>
              <dd class="font-medium text-gray-900 break-all">{{ preview.recipients.join(', ') }}</dd>
            </div>
            <div>
              <dt class="text-gray-500">Beiträge</dt>
              <dd class="font-medium text-gray-900">{{ preview.selectedFees.length }}</dd>
            </div>
            <div>
              <dt class="text-gray-500">Gesamtbetrag</dt>
              <dd class="font-medium text-gray-900">{{ formatCurrency(preview.totalAmount) }}</dd>
            </div>
            <div>
              <dt class="text-gray-500">Frist</dt>
              <dd class="font-medium text-gray-900">{{ formatDate(preview.deadline) }}</dd>
            </div>
            <div>
              <dt class="text-gray-500">QR-Code</dt>
              <dd class="font-medium text-gray-900">{{ includeQR ? 'angehängt' : 'deaktiviert' }}</dd>
            </div>
          </dl>
          <div v-if="preview.plannedReminderFees.length > 0" class="p-3 bg-amber-50 border border-amber-300 rounded-lg">
            <p class="font-medium text-amber-900 mb-1">Neu entstehende Mahngebühren</p>
            <ul class="space-y-0.5 text-amber-900">
              <li v-for="planned in preview.plannedReminderFees" :key="planned.baseFeeId" class="flex justify-between gap-3">
                <span>Mahngebühr für {{ planned.baseLabel }}</span>
                <span class="font-medium">{{ formatCurrency(planned.amount) }}</span>
              </li>
            </ul>
          </div>
          <p v-if="preview.warnings && preview.warnings.length > 0" class="text-xs text-amber-700">
            {{ preview.warnings.length }} Hinweis(e) — siehe Vorschau.
          </p>
        </div>
        <div class="flex justify-end gap-3 p-5 border-t">
          <button class="px-4 py-2 rounded-lg border text-sm font-medium hover:bg-gray-50" @click="showConfirmModal = false">
            Abbrechen
          </button>
          <button
            class="px-4 py-2 rounded-lg text-white text-sm font-medium disabled:opacity-50"
            :class="stage === 'final' ? 'bg-amber-600 hover:bg-amber-700' : 'bg-primary hover:bg-primary/90'"
            :disabled="isSending"
            @click="confirmSend"
          >
            {{ isSending ? 'Wird gesendet...' : 'Jetzt senden' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Settings dialog -->
    <div
      v-if="showSettingsDialog"
      class="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4"
      @click.self="showSettingsDialog = false"
    >
      <div class="bg-white rounded-xl shadow-xl w-full max-w-lg max-h-[90vh] flex flex-col">
        <div class="flex items-start justify-between gap-4 p-5 border-b">
          <h3 class="text-lg font-semibold text-gray-900">Zahlungsdaten für QR-Code</h3>
          <button type="button" class="rounded-lg p-2 text-gray-500 hover:bg-gray-100 hover:text-gray-900" @click="showSettingsDialog = false">
            <X class="h-5 w-5" />
          </button>
        </div>
        <div class="overflow-y-auto p-5 space-y-3">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Empfänger</label>
            <input
              type="text"
              v-model="reminderPaymentRecipientName"
              placeholder="Knirpsenstadt e.V."
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">IBAN</label>
            <input
              type="text"
              v-model="reminderPaymentIBAN"
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
              placeholder="BFSWDE33XXX"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>
          <p class="text-xs text-gray-500">Wenn Felder leer bleiben, werden die Standard-Zahlungsdaten der Kita verwendet.</p>
          <div v-if="reminderSettingsError" class="text-sm text-red-600">{{ reminderSettingsError }}</div>
        </div>
        <div class="flex justify-end gap-3 p-5 border-t">
          <button class="px-4 py-2 rounded-lg border text-sm font-medium hover:bg-gray-50" @click="showSettingsDialog = false">
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

    <!-- Email detail modal (shared by chronology + global log) -->
    <div
      v-if="selectedEmailLog || selectedChronologyLog"
      class="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4"
      @click.self="selectedEmailLog = null; selectedChronologyLog = null"
    >
      <div class="bg-white rounded-xl shadow-xl w-full max-w-4xl max-h-[90vh] flex flex-col">
        <div class="flex items-start justify-between gap-4 p-5 border-b">
          <div class="min-w-0">
            <h3 class="text-lg font-semibold text-gray-900">Gesendete E-Mail</h3>
            <p class="text-sm text-gray-500 truncate">{{ (selectedChronologyLog ?? selectedEmailLog)?.subject }}</p>
          </div>
          <button
            type="button"
            class="rounded-lg p-2 text-gray-500 hover:bg-gray-100 hover:text-gray-900"
            aria-label="Modal schließen"
            @click="selectedEmailLog = null; selectedChronologyLog = null"
          >
            <X class="h-5 w-5" />
          </button>
        </div>

        <div class="overflow-y-auto p-5">
          <dl class="grid gap-4 text-sm sm:grid-cols-2">
            <div>
              <dt class="font-medium text-gray-500">Zeitpunkt</dt>
              <dd class="mt-1 text-gray-900">{{ formatDateTime((selectedChronologyLog ?? selectedEmailLog)?.sentAt ?? '') }}</dd>
            </div>
            <div>
              <dt class="font-medium text-gray-500">Typ</dt>
              <dd class="mt-1 text-gray-900">{{ formatEmailType((selectedChronologyLog ?? selectedEmailLog)?.emailType ?? '') }}</dd>
            </div>
            <div class="sm:col-span-2">
              <dt class="font-medium text-gray-500">Empfänger</dt>
              <dd class="mt-1 break-all text-gray-900">{{ (selectedChronologyLog ?? selectedEmailLog)?.toEmail }}</dd>
            </div>
            <div class="sm:col-span-2">
              <dt class="font-medium text-gray-500">Betreff</dt>
              <dd class="mt-1 text-gray-900">{{ (selectedChronologyLog ?? selectedEmailLog)?.subject }}</dd>
            </div>
          </dl>

          <pre class="mt-5 max-h-[55vh] overflow-auto whitespace-pre-wrap rounded-lg border bg-gray-50 p-4 text-sm leading-6 text-gray-700">{{ (selectedChronologyLog ?? selectedEmailLog)?.body || '-' }}</pre>
        </div>

        <div class="flex justify-end p-5 border-t">
          <button
            type="button"
            class="px-4 py-2 rounded-lg border text-sm font-medium hover:bg-gray-50"
            @click="selectedEmailLog = null; selectedChronologyLog = null"
          >
            Schließen
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
