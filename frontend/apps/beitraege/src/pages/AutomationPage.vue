<script setup lang="ts">
import AutomationSendPreview from '@/components/automation/AutomationSendPreview.vue';
import AutomationFamilyList from '@/components/automation/AutomationFamilyList.vue';
import AutomationWorklist from '@/components/automation/AutomationWorklist.vue';
import AutomationChronology from '@/components/automation/AutomationChronology.vue';
import { ref } from 'vue';
import { useAutomationWorklist } from '@/composables/useAutomationWorklist';
import { useAutomationSending } from '@/composables/useAutomationSending';
import { useAutomationChronology } from '@/composables/useAutomationChronology';
import { X, ArrowLeft, Settings } from 'lucide-vue-next';
import { useAuthStore } from '@/stores/auth';
import { formatCurrency, formatDate } from '@/utils/format';
import EmailLogTab from '@/components/automation/EmailLogTab.vue';
import EmailLogModal from '@/components/automation/EmailLogModal.vue';
import ReminderSettingsDialog from '@/components/automation/ReminderSettingsDialog.vue';

const authStore = useAuthStore();

// ── Page tabs ────────────────────────────────────────────────────────────────
const activeTab = ref<'worklist' | 'log'>('worklist');
const {
  scope,
  cases,
  isCasesLoading,
  casesError,
  caseSearch,
  selectedHouseholdId,
  filteredCases,
  selectedCase,
  isNestedReminder,
  hasNestedReminderBelow,
  lastContactOf,
  hasBlockedEmail,
  loadCases,
} = useAutomationWorklist();

const {
  chronology,
  isChronologyLoading,
  selectedChronologyLog,
  loadChronology,
} = useAutomationChronology(selectedCase, selectedHouseholdId);

const {
  selectedFeeIds,
  reminderFeeIds,
  includeQR,
  preview,
  isPreviewLoading,
  previewError,
  subjectEdit,
  bodyEdit,
  userEdited,
  resetNotice,
  isSending,
  sendError,
  conflictFeeCount,
  sendResult,
  showConfirmModal,
  isMahnung,
  untickedDueFeeCount,
  toggleFee,
  toggleReminderFee,
  invalidatePreview,
  onSubjectInput,
  onBodyInput,
  resetEdits,
  openSendConfirmation,
  confirmSend,
} = useAutomationSending(
  selectedCase, selectedHouseholdId, cases, loadCases, openCase,
);

const showSettingsDialog = ref(false);

function openCase(householdId: string): void {
  selectedHouseholdId.value = householdId;
  const item = selectedCase.value;
  const actionableIds = (item?.fees ?? [])
    .filter((fee) => fee.status === 'actionable_initial' || fee.status === 'actionable_final' || fee.status === 'history_unknown')
    .map((fee) => fee.feeId);
  selectedFeeIds.value = actionableIds;
  // Mahngebühren are never charged by default, even when due by the rules.
  reminderFeeIds.value = [];
  sendResult.value = null;
  sendError.value = null;
  resetNotice.value = false;
  userEdited.value = false;
  invalidatePreview();
  loadChronology();
}

function closeCase(): void {
  selectedHouseholdId.value = null;
  invalidatePreview();
}

</script>

<template>
  <div>
    <!-- Header -->
    <div class="mb-6 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
      <div>
        <h1 class="text-2xl font-bold text-foreground">Erinnerungen</h1>
        <p class="text-muted-foreground mt-1">Familien mit offenen Beiträgen bearbeiten</p>
      </div>
      <button
        class="inline-flex items-center gap-2 px-3 py-2 rounded-lg border bg-card text-sm font-medium hover:bg-accent"
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
        :class="activeTab === 'worklist' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'"
        @click="activeTab = 'worklist'"
      >
        Arbeitsliste
      </button>
      <button
        class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors"
        :class="activeTab === 'log' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'"
        @click="activeTab = 'log'"
      >
        Versandverlauf
      </button>
    </div>

    <!-- ════════════════════ Worklist ════════════════════ -->
    <div v-if="activeTab === 'worklist'" v-show="authStore.isAdmin">
      <AutomationWorklist
        v-model:scope="scope"
        :cases="cases"
        :isCasesLoading="isCasesLoading"
        :casesError="casesError"
        v-model:caseSearch="caseSearch"
        :filteredCases="filteredCases"
        :loadCases="loadCases"
      />

      <!-- Master-detail -->
      <div :class="selectedCase ? 'lg:grid lg:grid-cols-[minmax(300px,2fr)_minmax(0,3fr)] lg:gap-6' : ''">
        <!-- Family list -->
        <AutomationFamilyList
          :selectedHouseholdId="selectedHouseholdId"
          :filteredCases="filteredCases"
          :selectedCase="selectedCase"
          :lastContactOf="lastContactOf"
          :hasBlockedEmail="hasBlockedEmail"
          :openCase="openCase"
        />

        <!-- Detail panel (desktop) / full view (mobile) -->
        <div v-if="selectedCase" class="mt-6 lg:mt-0">
          <div class="lg:sticky lg:top-6">
            <!-- Mobile back -->
            <button
              class="inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground mb-3 lg:hidden"
              @click="closeCase"
            >
              <ArrowLeft class="h-4 w-4" />
              Zurück zur Liste
            </button>

            <div class="bg-card border rounded-xl p-5">
              <!-- Success banner -->
              <div v-if="sendResult" class="mb-4 p-4 bg-green-50 dark:bg-green-950/40 border border-green-200 rounded-lg text-sm">
                <p class="font-medium text-green-800 dark:text-green-300">E-Mail gesendet an {{ sendResult.sentTo.join(', ') }}</p>
                <p class="text-green-700 dark:text-green-300 mt-1">
                  Frist: {{ formatDate(sendResult.deadline) }}
                  <template v-if="sendResult.createdReminderFees.length > 0">
                    · Mahngebühren erstellt: {{ sendResult.createdReminderFees.length }}
                  </template>
                </p>
              </div>

              <div class="flex items-start justify-between gap-3 mb-4">
                <div>
                  <h2 class="text-lg font-semibold text-foreground">{{ selectedCase.householdName }}</h2>
                  <p class="text-sm text-muted-foreground">
                    Empfänger:
                    <template v-if="selectedCase.recipients.length > 0">{{ selectedCase.recipients.join(', ') }}</template>
                    <template v-else><span class="text-red-600 dark:text-red-300 font-medium">keine gültige E-Mail-Adresse</span></template>
                  </p>
                </div>
                <div class="text-right shrink-0">
                  <p class="text-xs text-muted-foreground">Offen gesamt</p>
                  <p class="font-semibold text-foreground">{{ formatCurrency(selectedCase.totalRemaining) }}</p>
                </div>
              </div>

              <div class="flex flex-wrap items-center gap-2 mb-3">
                <label class="inline-flex items-center gap-2 text-sm text-foreground">
                  <input type="checkbox" v-model="includeQR" />
                  QR-Code
                </label>
              </div>
              <p
                v-if="untickedDueFeeCount > 0"
                class="p-3 bg-amber-50 dark:bg-amber-950/40 border border-amber-300 rounded-lg text-xs text-amber-900 dark:text-amber-300 mb-3"
              >
                Für {{ untickedDueFeeCount === 1 ? 'einen ausgewählten Beitrag' : `${untickedDueFeeCount} ausgewählte Beiträge` }}
                ist laut Regeln eine Mahngebühr fällig (erinnert, Frist abgelaufen). Sie wird nur erhoben, wenn du sie in der Spalte
                „Mahngebühr erheben“ beim Status ankreuzt — sonst geht eine Zahlungserinnerung raus.
              </p>

              <AutomationSendPreview
                :selectedCase="selectedCase"
                :isNestedReminder="isNestedReminder"
                :hasNestedReminderBelow="hasNestedReminderBelow"
                :selectedFeeIds="selectedFeeIds"
                :reminderFeeIds="reminderFeeIds"
                :includeQR="includeQR"
                :preview="preview"
                :isPreviewLoading="isPreviewLoading"
                :previewError="previewError"
                v-model:subjectEdit="subjectEdit"
                v-model:bodyEdit="bodyEdit"
                :userEdited="userEdited"
                :resetNotice="resetNotice"
                :isSending="isSending"
                :sendError="sendError"
                :conflictFeeCount="conflictFeeCount"
                :isMahnung="isMahnung"
                :toggleFee="toggleFee"
                :toggleReminderFee="toggleReminderFee"
                :onSubjectInput="onSubjectInput"
                :onBodyInput="onBodyInput"
                :resetEdits="resetEdits"
                :openSendConfirmation="openSendConfirmation"
              />

      <!-- Family chronology -->
              <AutomationChronology
                :chronology="chronology"
                :isChronologyLoading="isChronologyLoading"
                v-model:selectedChronologyLog="selectedChronologyLog"
              />
            </div>
          </div>
        </div>

        <!-- Desktop empty state -->
        <div v-else class="hidden lg:flex items-center justify-center border border-dashed rounded-xl p-12 text-muted-foreground text-sm">
          Familie aus der Liste auswählen, um den Entwurf zu erstellen.
        </div>
      </div>
    </div>

    <!-- ════════════════════ Versandverlauf ════════════════════ -->
    <KeepAlive>
      <EmailLogTab v-if="activeTab === 'log' && authStore.isAdmin" />
    </KeepAlive>

    <!-- ════════════════════ Modals ════════════════════ -->

    <!-- Send confirmation -->
    <div
      v-if="showConfirmModal && preview"
      class="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4"
      @click.self="showConfirmModal = false"
    >
      <div class="bg-card rounded-xl shadow-xl w-full max-w-lg max-h-[90vh] flex flex-col">
        <div class="flex items-start justify-between gap-4 p-5 border-b">
          <h3 class="text-lg font-semibold text-foreground">
            {{ isMahnung ? 'Mahnung senden?' : 'Erinnerung senden?' }}
          </h3>
          <button type="button" class="rounded-lg p-2 text-muted-foreground hover:bg-accent hover:text-foreground" @click="showConfirmModal = false" aria-label="Schließen">
            <X class="h-5 w-5" />
          </button>
        </div>
        <div class="overflow-y-auto p-5 text-sm space-y-3">
          <dl class="grid grid-cols-2 gap-3">
            <div>
              <dt class="text-muted-foreground">Familie</dt>
              <dd class="font-medium text-foreground">{{ selectedCase?.householdName }}</dd>
            </div>
            <div>
              <dt class="text-muted-foreground">Empfänger</dt>
              <dd class="font-medium text-foreground break-all">{{ preview.recipients.join(', ') }}</dd>
            </div>
            <div>
              <dt class="text-muted-foreground">Beiträge</dt>
              <dd class="font-medium text-foreground">{{ preview.selectedFees.length }}</dd>
            </div>
            <div>
              <dt class="text-muted-foreground">Gesamtbetrag</dt>
              <dd class="font-medium text-foreground">{{ formatCurrency(preview.totalAmount) }}</dd>
            </div>
            <div>
              <dt class="text-muted-foreground">Frist</dt>
              <dd class="font-medium text-foreground">{{ formatDate(preview.deadline) }}</dd>
            </div>
            <div>
              <dt class="text-muted-foreground">QR-Code</dt>
              <dd class="font-medium text-foreground">{{ includeQR ? 'angehängt' : 'deaktiviert' }}</dd>
            </div>
          </dl>
          <div v-if="preview.plannedReminderFees.length > 0" class="p-3 bg-amber-50 dark:bg-amber-950/40 border border-amber-300 rounded-lg">
            <p class="font-medium text-amber-900 dark:text-amber-300 mb-1">Neu entstehende Mahngebühren</p>
            <ul class="space-y-0.5 text-amber-900 dark:text-amber-300">
              <li v-for="planned in preview.plannedReminderFees" :key="planned.baseFeeId" class="flex justify-between gap-3">
                <span>Mahngebühr für {{ planned.baseLabel }}</span>
                <span class="font-medium">{{ formatCurrency(planned.amount) }}</span>
              </li>
            </ul>
          </div>
          <p v-if="preview.warnings && preview.warnings.length > 0" class="text-xs text-amber-700 dark:text-amber-300">
            {{ preview.warnings.length }} Hinweis(e) — siehe Vorschau.
          </p>
        </div>
        <div class="flex justify-end gap-3 p-5 border-t">
          <button class="px-4 py-2 rounded-lg border text-sm font-medium hover:bg-accent" @click="showConfirmModal = false">
            Abbrechen
          </button>
          <button
            class="px-4 py-2 rounded-lg text-white text-sm font-medium disabled:opacity-50"
            :class="isMahnung ? 'bg-amber-600 hover:bg-amber-700' : 'bg-primary hover:bg-primary/90'"
            :disabled="isSending"
            @click="confirmSend"
          >
            {{ isSending ? 'Wird gesendet...' : 'Jetzt senden' }}
          </button>
        </div>
      </div>
    </div>

    <ReminderSettingsDialog v-if="showSettingsDialog" @close="showSettingsDialog = false" />

    <EmailLogModal v-if="selectedChronologyLog" :log="selectedChronologyLog" @close="selectedChronologyLog = null" />
  </div>
</template>
