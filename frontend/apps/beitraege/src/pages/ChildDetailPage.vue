<script setup lang="ts">
import ChildPaymentsCard from '@/components/child/ChildPaymentsCard.vue';
import ChildHouseholdCard from '@/components/child/ChildHouseholdCard.vue';
import ChildInformationCard from '@/components/child/ChildInformationCard.vue';
import { ref, onMounted, onUnmounted, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { api } from '@/api';
import { useChildDetailData } from '@/composables/useChildDetailData';
import ChangeHistory from '@/components/family/ChangeHistory.vue';
import { useAuthStore } from '@/stores/auth';
import type { FeeExpectation, MatchSuggestion, Parent, UpdateHouseholdRequest } from '@/api/types';
import { ArrowLeft, Trash2, Loader2, AlertCircle, Unlink } from 'lucide-vue-next';
import ChildNotesCard from '@/components/child/ChildNotesCard.vue';
import ChildEditDialog from '@/components/child/ChildEditDialog.vue';
import ParentFormDialog from '@/components/child/ParentFormDialog.vue';
import TransactionDetailModal from '@/components/child/TransactionDetailModal.vue';
import AllocationModal from '@/components/child/AllocationModal.vue';
import ParentDetailModal from '@/components/child/ParentDetailModal.vue';
import { formatCurrency, formatDate, formatMonthName, todayISO } from '@/utils/format';
import { getFeeMatches, getFeeTypeName } from '@/utils/fees';

const authStore = useAuthStore();
const route = useRoute();
const router = useRouter();
const childId = computed(() => route.params.id as string);
const {
  child,
  fees,
  isLoading,
  error,
  careHoursHistory,
  legalHoursHistory,
  childcareFee,
  isLoadingChildcareFee,
  childcareFeeCareHours,
  trustedIbans,
  likelyTransactions,
  isLoadingLikelyTransactions,
  likelyTransactionsError,
  likelyTransactionsScanned,
  upcomingCareHours,
  upcomingLegalHours,
  loadChild,
  loadLikelyTransactions,
} = useChildDetailData(childId);

// Dialogs rendered as sub-components (each resets its own state on open)
const showEditDialog = ref(false);
const showParentDialog = ref(false);
const parentDialogMode = ref<'create' | 'link'>('create');
const selectedFee = ref<FeeExpectation | null>(null);
const allocationSuggestion = ref<MatchSuggestion | null>(null);
const selectedParentForDetail = ref<Parent | null>(null);

// Delete dialog state
const showDeleteDialog = ref(false);
const isDeleting = ref(false);

// Unlink parent state
const parentToUnlink = ref<Parent | null>(null);
const showUnlinkDialog = ref(false);
const isUnlinking = ref(false);

// Household editing state
const isEditingHousehold = ref(false);
const householdEditForm = ref<UpdateHouseholdRequest>({});
const isSavingHousehold = ref(false);
const householdError = ref<string | null>(null);

// Reminder dialog state
const showReminderDialog = ref(false);
const reminderFee = ref<FeeExpectation | null>(null);
const isCreatingReminder = ref(false);
const reminderError = ref<string | null>(null);

// ESC key handler to close all modals (note dialogs close themselves)
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (showReminderDialog.value) {
      showReminderDialog.value = false;
    } else if (allocationSuggestion.value) {
      allocationSuggestion.value = null;
    } else if (selectedFee.value) {
      selectedFee.value = null;
    } else if (selectedParentForDetail.value) {
      selectedParentForDetail.value = null;
    } else if (showUnlinkDialog.value) {
      showUnlinkDialog.value = false;
    } else if (showParentDialog.value) {
      showParentDialog.value = false;
    } else if (showDeleteDialog.value) {
      showDeleteDialog.value = false;
    } else if (showEditDialog.value) {
      showEditDialog.value = false;
    }
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown);
});

function formatHistoryRange(entry: { effectiveFrom: string; effectiveUntil?: string | null }): string {
  const from = formatDate(entry.effectiveFrom);
  if (!entry.effectiveUntil) return `ab ${from}`;
  return `${from} bis ${formatDate(entry.effectiveUntil)}`;
}

function formatSuggestionExpectation(suggestion: MatchSuggestion): string {
  const expectation = suggestion.expectation;
  if (!expectation) return '';
  const monthLabel = expectation.month ? `${formatMonthName(expectation.month)} ` : '';
  return `${getFeeTypeName(expectation.feeType)} ${monthLabel}${expectation.year}`;
}

async function onChildSaved() {
  showEditDialog.value = false;
  await loadChild();
}

async function handleDelete() {
  isDeleting.value = true;
  try {
    await api.deleteChild(childId.value);
    router.push('/kinder');
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Löschen';
    showDeleteDialog.value = false;
  } finally {
    isDeleting.value = false;
  }
}

function openCreateParentDialog() {
  parentDialogMode.value = 'create';
  showParentDialog.value = true;
}

function openLinkParentDialog() {
  parentDialogMode.value = 'link';
  showParentDialog.value = true;
}

async function onParentLinked() {
  showParentDialog.value = false;
  await loadChild();
}

function confirmUnlinkParent(parent: Parent) {
  parentToUnlink.value = parent;
  showUnlinkDialog.value = true;
}

async function handleUnlinkParent() {
  if (!parentToUnlink.value) return;
  isUnlinking.value = true;
  try {
    await api.unlinkParent(childId.value, parentToUnlink.value.id);
    await loadChild();
    showUnlinkDialog.value = false;
    parentToUnlink.value = null;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Entfernen';
  } finally {
    isUnlinking.value = false;
  }
}

const openFees = computed(() => fees.value.filter(f => !f.isPaid));

type FeeGroup = {
  fee: FeeExpectation;
  reminders: FeeExpectation[];
};

const feeGroups = computed<FeeGroup[]>(() => {
  const baseFees = fees.value.filter(f => f.feeType !== 'REMINDER');
  const baseIds = new Set(baseFees.map(f => f.id));
  const remindersByBase = new Map<string, FeeExpectation[]>();
  const orphanReminders: FeeExpectation[] = [];

  for (const fee of fees.value) {
    if (fee.feeType !== 'REMINDER') continue;
    if (fee.reminderForId && baseIds.has(fee.reminderForId)) {
      const list = remindersByBase.get(fee.reminderForId) ?? [];
      list.push(fee);
      remindersByBase.set(fee.reminderForId, list);
    } else {
      orphanReminders.push(fee);
    }
  }

  const groups: FeeGroup[] = baseFees.map(fee => ({
    fee,
    reminders: remindersByBase.get(fee.id) ?? [],
  }));

  for (const reminder of orphanReminders) {
    groups.push({ fee: reminder, reminders: [] });
  }

  return groups;
});

const openFeeGroups = computed(() =>
  feeGroups.value.filter(group => {
    const hasOpenReminders = group.reminders.some(rem => !rem.isPaid);
    return !group.fee.isPaid || hasOpenReminders;
  })
);

const paidFeeGroups = computed(() =>
  feeGroups.value.filter(group => {
    if (!group.fee.isPaid) return false;
    return group.reminders.every(rem => rem.isPaid);
  })
);

function openTransactionModal(fee: FeeExpectation) {
  if (getFeeMatches(fee).length === 0) return;
  selectedFee.value = fee;
}

function openAllocationModal(suggestion: MatchSuggestion): void {
  allocationSuggestion.value = suggestion;
}

// After unmatching, deleting or allocating a transaction
async function onTransactionChanged() {
  selectedFee.value = null;
  allocationSuggestion.value = null;
  await loadChild();
  await loadLikelyTransactions();
}

function openParentDetailModal(parent: Parent) {
  selectedParentForDetail.value = parent;
}

function getPaymentSummary(fee: FeeExpectation): string {
  const matches = getFeeMatches(fee);
  if (matches.length === 0) return '';
  if (matches.length === 1) {
    const txDate = matches[0].transaction?.bookingDate;
    if (txDate) {
      return `Bezahlt am ${formatDate(txDate)}`;
    }
    if (fee.paidAt) {
      return `Bezahlt am ${formatDate(fee.paidAt)}`;
    }
    return 'Bezahlt';
  }
  return `Bezahlt mit ${matches.length} Zahlungen`;
}

function getReminderCounts(reminders: FeeExpectation[]) {
  const total = reminders.length;
  const paid = reminders.filter(r => r.isPaid).length;
  return { total, paid, open: total - paid };
}

function getReminderSummary(reminders: FeeExpectation[]): string {
  if (reminders.length === 0) return '';
  const { total, paid, open } = getReminderCounts(reminders);
  const label = total === 1 ? 'Mahngebühr' : 'Mahngebühren';
  if (open === 0) return `${total} ${label} bezahlt`;
  if (paid === 0) return `${total} ${label} offen`;
  return `${paid} bezahlt · ${open} offen`;
}

// Siblings computed property (other children in the same household)
const siblings = computed(() => {
  if (!child.value?.household?.children) return [];
  return child.value.household.children.filter(c => c.id !== childId.value);
});

// Household parents computed property
const householdParents = computed(() => {
  return child.value?.parents || [];
});

// Household functions
function startEditingHousehold() {
  if (!child.value?.household) return;
  householdEditForm.value = {
    name: child.value.household.name,
    annualHouseholdIncome: child.value.household.annualHouseholdIncome,
    incomeStatus: child.value.household.incomeStatus || '',
    childrenCountForFees: child.value.household.childrenCountForFees,
  };
  householdError.value = null;
  isEditingHousehold.value = true;
}

function cancelEditingHousehold() {
  isEditingHousehold.value = false;
  householdEditForm.value = {};
  householdError.value = null;
}

async function saveHouseholdEdit() {
  if (!child.value?.household) return;
  isSavingHousehold.value = true;
  householdError.value = null;
  try {
    await api.updateHousehold(child.value.household.id, householdEditForm.value);
    isEditingHousehold.value = false;
    // Reload child to get updated household
    await loadChild();
  } catch (e) {
    householdError.value = e instanceof Error ? e.message : 'Fehler beim Speichern';
  } finally {
    isSavingHousehold.value = false;
  }
}

// Reminder functions
function canCreateReminder(fee: FeeExpectation): boolean {
  // Can create reminder if: past due date, not paid, not already a REMINDER type, no existing reminder
  const isPastDue = fee.dueDate.slice(0, 10) < todayISO();
  const isUnpaid = !fee.isPaid;
  const isNotReminder = fee.feeType !== 'REMINDER';
  const hasNoReminder = !fees.value.some(f => f.reminderForId === fee.id);
  return isPastDue && isUnpaid && isNotReminder && hasNoReminder;
}

function openReminderDialog(fee: FeeExpectation) {
  reminderFee.value = fee;
  reminderError.value = null;
  showReminderDialog.value = true;
}

async function createReminder() {
  if (!reminderFee.value) return;
  isCreatingReminder.value = true;
  reminderError.value = null;
  try {
    await api.createReminder(reminderFee.value.id);
    await loadChild(); // Reload to get the new reminder fee
    showReminderDialog.value = false;
    reminderFee.value = null;
  } catch (e) {
    reminderError.value = e instanceof Error ? e.message : 'Fehler beim Erstellen der Mahngebühr';
  } finally {
    isCreatingReminder.value = false;
  }
}
</script>

<template>
  <ChangeHistory v-if="authStore.isAdmin" kind="child" :id="String(route.params.id)" />
  <div>
    <!-- Back button -->
    <button
      @click="router.push('/kinder')"
      class="flex items-center gap-2 text-muted-foreground hover:text-foreground mb-6"
    >
      <ArrowLeft class="h-4 w-4" />
      Zurück zur Übersicht
    </button>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="bg-red-50 dark:bg-red-950/40 border border-red-200 rounded-lg p-4">
      <p class="text-red-600 dark:text-red-300">{{ error }}</p>
      <button @click="loadChild" class="mt-2 text-sm text-red-700 dark:text-red-300 underline">
        Erneut versuchen
      </button>
    </div>

    <!-- Child details -->
    <div v-else-if="child">
      <ChildInformationCard
        :child="child"
        :careHoursHistory="careHoursHistory"
        :legalHoursHistory="legalHoursHistory"
        :upcomingCareHours="upcomingCareHours"
        :upcomingLegalHours="upcomingLegalHours"
        v-model:showEditDialog="showEditDialog"
        v-model:showDeleteDialog="showDeleteDialog"
        :formatHistoryRange="formatHistoryRange"
      /><ChildHouseholdCard
          :child="child"
          :childcareFee="childcareFee"
          :isLoadingChildcareFee="isLoadingChildcareFee"
          :childcareFeeCareHours="childcareFeeCareHours"
          :trustedIbans="trustedIbans"
          :isEditingHousehold="isEditingHousehold"
          :householdEditForm="householdEditForm"
          :isSavingHousehold="isSavingHousehold"
          :householdError="householdError"
          :openCreateParentDialog="openCreateParentDialog"
          :openLinkParentDialog="openLinkParentDialog"
          :confirmUnlinkParent="confirmUnlinkParent"
          :openParentDetailModal="openParentDetailModal"
          :siblings="siblings"
          :householdParents="householdParents"
          :startEditingHousehold="startEditingHousehold"
          :cancelEditingHousehold="cancelEditingHousehold"
          :saveHouseholdEdit="saveHouseholdEdit"
        /><ChildNotesCard :child-id="childId" />

      <!-- Fees section -->
      <ChildPaymentsCard
        :fees="fees"
        :likelyTransactions="likelyTransactions"
        :isLoadingLikelyTransactions="isLoadingLikelyTransactions"
        :likelyTransactionsError="likelyTransactionsError"
        :likelyTransactionsScanned="likelyTransactionsScanned"
        :formatSuggestionExpectation="formatSuggestionExpectation"
        :openFeeGroups="openFeeGroups"
        :paidFeeGroups="paidFeeGroups"
        :openTransactionModal="openTransactionModal"
        :openAllocationModal="openAllocationModal"
        :getPaymentSummary="getPaymentSummary"
        :getReminderSummary="getReminderSummary"
        :canCreateReminder="canCreateReminder"
        :openReminderDialog="openReminderDialog"
      />
    </div>

    <ChildEditDialog
      v-if="showEditDialog && child"
      :child="child"
      :care-hours-history="careHoursHistory"
      :legal-hours-history="legalHoursHistory"
      @close="showEditDialog = false"
      @saved="onChildSaved"
    />

    <!-- Delete Confirmation Dialog -->
    <div
      v-if="showDeleteDialog"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="showDeleteDialog = false"
    >
      <div class="bg-card rounded-xl shadow-xl w-full max-w-sm mx-4 p-6">
        <div class="flex items-center gap-3 mb-4">
          <div class="p-2 bg-red-100 dark:bg-red-950/40 rounded-lg">
            <Trash2 class="h-6 w-6 text-red-600 dark:text-red-300" />
          </div>
          <h2 class="text-xl font-semibold">Kind löschen?</h2>
        </div>

        <p class="text-muted-foreground mb-6">
          Möchtest du <strong>{{ child?.firstName }} {{ child?.lastName }}</strong> wirklich löschen?
          Diese Aktion kann nicht rückgängig gemacht werden.
        </p>

        <div class="flex justify-end gap-3">
          <button
            @click="showDeleteDialog = false"
            class="px-4 py-2 text-foreground hover:bg-accent rounded-lg transition-colors"
          >
            Abbrechen
          </button>
          <button
            @click="handleDelete"
            :disabled="isDeleting"
            class="inline-flex items-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isDeleting" class="h-4 w-4 animate-spin" />
            <Trash2 v-else class="h-4 w-4" />
            Löschen
          </button>
        </div>
      </div>
    </div>

    <ParentFormDialog
      v-if="showParentDialog && child"
      :child="child"
      :initial-mode="parentDialogMode"
      @close="showParentDialog = false"
      @saved="onParentLinked"
    />

    <!-- Unlink Parent Confirmation Dialog -->
    <div
      v-if="showUnlinkDialog"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="showUnlinkDialog = false"
    >
      <div class="bg-card rounded-xl shadow-xl w-full max-w-sm mx-4 p-6">
        <div class="flex items-center gap-3 mb-4">
          <div class="p-2 bg-amber-100 dark:bg-amber-950/40 rounded-lg">
            <Unlink class="h-6 w-6 text-amber-600 dark:text-amber-300" />
          </div>
          <h2 class="text-xl font-semibold">Verknüpfung aufheben?</h2>
        </div>

        <p class="text-muted-foreground mb-6">
          Möchtest du die Verknüpfung zu <strong>{{ parentToUnlink?.firstName }} {{ parentToUnlink?.lastName }}</strong> aufheben?
          Der Elternteil wird nicht gelöscht, nur die Verknüpfung zu diesem Kind.
        </p>

        <div class="flex justify-end gap-3">
          <button
            @click="showUnlinkDialog = false"
            class="px-4 py-2 text-foreground hover:bg-accent rounded-lg transition-colors"
          >
            Abbrechen
          </button>
          <button
            @click="handleUnlinkParent"
            :disabled="isUnlinking"
            class="inline-flex items-center gap-2 px-4 py-2 bg-amber-600 text-white rounded-lg hover:bg-amber-700 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isUnlinking" class="h-4 w-4 animate-spin" />
            <Unlink v-else class="h-4 w-4" />
            Aufheben
          </button>
        </div>
      </div>
    </div>

    <TransactionDetailModal
      v-if="selectedFee"
      :fee="selectedFee"
      @close="selectedFee = null"
      @changed="onTransactionChanged"
    />

    <AllocationModal
      v-if="allocationSuggestion"
      :suggestion="allocationSuggestion"
      :open-fees="openFees"
      @close="allocationSuggestion = null"
      @allocated="onTransactionChanged"
    />

    <ParentDetailModal
      v-if="selectedParentForDetail"
      :parent="selectedParentForDetail"
      @close="selectedParentForDetail = null"
      @saved="loadChild"
    />

    <!-- Reminder Confirmation Dialog -->
    <div
      v-if="showReminderDialog && reminderFee"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="showReminderDialog = false"
    >
      <div class="bg-card rounded-xl shadow-xl w-full max-w-sm mx-4 p-6">
        <div class="flex items-center gap-3 mb-4">
          <div class="p-2 bg-amber-100 dark:bg-amber-950/40 rounded-lg">
            <AlertCircle class="h-6 w-6 text-amber-600 dark:text-amber-300" />
          </div>
          <h2 class="text-xl font-semibold">Mahngebühr erstellen?</h2>
        </div>

        <div class="mb-6">
          <p class="text-muted-foreground mb-4">
            Möchtest du eine Mahngebühr für den folgenden überfälligen Beitrag erstellen?
          </p>
          <div class="p-3 bg-amber-50 dark:bg-amber-950/40 border border-amber-200 rounded-lg">
            <p class="font-medium">{{ getFeeTypeName(reminderFee.feeType) }}</p>
            <p class="text-sm text-muted-foreground">
              {{ reminderFee.month ? formatMonthName(reminderFee.month) + ' ' : '' }}{{ reminderFee.year }}
              · {{ formatCurrency(reminderFee.amount) }}
            </p>
            <p class="text-sm text-red-600 dark:text-red-300 mt-1">
              Fällig seit: {{ formatDate(reminderFee.dueDate) }}
            </p>
          </div>
          <p class="text-sm text-muted-foreground mt-3">
            Es wird eine Mahngebühr von <strong>10,00 EUR</strong> erstellt.
          </p>
        </div>

        <div v-if="reminderError" class="p-3 bg-red-50 dark:bg-red-950/40 border border-red-200 rounded-lg mb-4">
          <p class="text-sm text-red-600 dark:text-red-300">{{ reminderError }}</p>
        </div>

        <div class="flex justify-end gap-3">
          <button
            @click="showReminderDialog = false"
            class="px-4 py-2 text-foreground hover:bg-accent rounded-lg transition-colors"
          >
            Abbrechen
          </button>
          <button
            @click="createReminder"
            :disabled="isCreatingReminder"
            class="inline-flex items-center gap-2 px-4 py-2 bg-amber-600 text-white rounded-lg hover:bg-amber-700 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isCreatingReminder" class="h-4 w-4 animate-spin" />
            <AlertCircle v-else class="h-4 w-4" />
            Mahngebühr erstellen
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
