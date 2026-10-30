<script setup lang="ts">
import ChildPaymentsCard from '@/components/child/ChildPaymentsCard.vue';
import ChildHouseholdCard from '@/components/child/ChildHouseholdCard.vue';
import ChildInformationCard from '@/components/child/ChildInformationCard.vue';
import { ref, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { api } from '@/api';
import { useChildDetailData } from '@/composables/useChildDetailData';
import ChangeHistory from '@/components/family/ChangeHistory.vue';
import { useAuthStore } from '@/stores/auth';
import type { ChildEinstufungSync, FeeExpectation, MatchSuggestion, Parent, UpdateHouseholdRequest } from '@/api/types';
import { Trash2, Loader2, AlertCircle, Unlink } from 'lucide-vue-next';
import PageHeader from '@/components/ui/PageHeader.vue';
import AppSheet from '@/components/ui/AppSheet.vue';
import ChildNotesCard from '@/components/child/ChildNotesCard.vue';
import ChildEditDialog from '@/components/child/ChildEditDialog.vue';
import ChildEinstufungSyncNotice from '@/components/child/ChildEinstufungSyncNotice.vue';
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

// What the last save of child data did to Einstufungen and generated fees.
const einstufungSync = ref<ChildEinstufungSync | null>(null);

async function onChildSaved(sync: ChildEinstufungSync | null) {
  showEditDialog.value = false;
  einstufungSync.value = sync;
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
  householdEditForm.value = authStore.canAccessFees ? {
    name: child.value.household.name,
    annualHouseholdIncome: child.value.household.annualHouseholdIncome,
    incomeStatus: child.value.household.incomeStatus || '',
    childrenCountForFees: child.value.household.childrenCountForFees,
  } : { name: child.value.household.name };
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
  <div>
    <PageHeader :title="child ? `${child.firstName} ${child.lastName}` : 'Kind'" back="/kinder" />

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="note note-danger">
      <AlertCircle />
      <div><p>{{ error }}</p>
      <button @click="loadChild" class="btn btn-ghost btn-sm mt-1">
        Erneut versuchen
      </button></div>
    </div>

    <!-- Child details -->
    <div v-else-if="child">
      <ChildEinstufungSyncNotice
        v-if="einstufungSync?.einstufungen.length"
        :sync="einstufungSync"
        @close="einstufungSync = null"
      />
      <ChildInformationCard
        :child="child"
        :careHoursHistory="careHoursHistory"
        :legalHoursHistory="legalHoursHistory"
        :upcomingCareHours="upcomingCareHours"
        :upcomingLegalHours="upcomingLegalHours"
        v-model:showEditDialog="showEditDialog"
        v-model:showDeleteDialog="showDeleteDialog"
        :formatHistoryRange="formatHistoryRange"
      />
      <ChildHouseholdCard
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
        />
      <ChildNotesCard v-if="authStore.canAccessFees" :child-id="childId" />

      <!-- Fees section (not for Leitung) -->
      <ChildPaymentsCard
        v-if="authStore.canAccessFees"
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

    <AppSheet v-if="showDeleteDialog" title="Kind löschen?" size="sm" @close="showDeleteDialog = false">
      <p class="text-muted-foreground mb-6">
        Möchtest du <strong>{{ child?.firstName }} {{ child?.lastName }}</strong> wirklich löschen?
        Diese Aktion kann nicht rückgängig gemacht werden.
      </p>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="showDeleteDialog = false">Abbrechen</button>
        <button type="button" class="btn btn-danger" :disabled="isDeleting" @click="handleDelete">
          <Loader2 v-if="isDeleting" class="animate-spin" /><Trash2 v-else />Löschen
        </button>
      </template>
    </AppSheet>

    <ParentFormDialog
      v-if="showParentDialog && child"
      :child="child"
      :initial-mode="parentDialogMode"
      @close="showParentDialog = false"
      @saved="onParentLinked"
    />

    <AppSheet v-if="showUnlinkDialog" title="Verknüpfung aufheben?" size="sm" @close="showUnlinkDialog = false">
      <p class="text-muted-foreground mb-6">
        Möchtest du die Verknüpfung zu <strong>{{ parentToUnlink?.firstName }} {{ parentToUnlink?.lastName }}</strong> aufheben?
        Der Elternteil wird nicht gelöscht, nur die Verknüpfung zu diesem Kind.
      </p>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="showUnlinkDialog = false">Abbrechen</button>
        <button type="button" class="btn btn-primary" :disabled="isUnlinking" @click="handleUnlinkParent">
          <Loader2 v-if="isUnlinking" class="animate-spin" /><Unlink v-else />Aufheben
        </button>
      </template>
    </AppSheet>

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

    <AppSheet v-if="showReminderDialog && reminderFee" title="Mahngebühr erstellen?" size="sm"
      @close="showReminderDialog = false">
      <p class="text-muted-foreground mb-4">
        Möchtest du eine Mahngebühr für den folgenden überfälligen Beitrag erstellen?
      </p>
      <div class="card p-3">
        <p class="font-medium">{{ getFeeTypeName(reminderFee.feeType) }}</p>
        <p class="text-sm text-muted-foreground">
          {{ reminderFee.month ? formatMonthName(reminderFee.month) + ' ' : '' }}{{ reminderFee.year }}
          · {{ formatCurrency(reminderFee.amount) }}
        </p>
        <p class="text-sm text-destructive mt-1">Fällig seit: {{ formatDate(reminderFee.dueDate) }}</p>
      </div>
      <p class="text-sm text-muted-foreground mt-3">
        Es wird eine Mahngebühr von <strong>10,00 EUR</strong> erstellt.
      </p>
      <div v-if="reminderError" class="note note-danger mt-4">{{ reminderError }}</div>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="showReminderDialog = false">Abbrechen</button>
        <button type="button" class="btn btn-primary" :disabled="isCreatingReminder" @click="createReminder">
          <Loader2 v-if="isCreatingReminder" class="animate-spin" /><AlertCircle v-else />Mahngebühr erstellen
        </button>
      </template>
    </AppSheet>
    <ChangeHistory v-if="authStore.isAdmin" kind="child" :id="String(route.params.id)" class="mt-6" />
  </div>
</template>
