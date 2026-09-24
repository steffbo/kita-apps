<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { api } from '@/api';
import type {
  CareHoursHistoryEntry,
  Child,
  ChildcareFeeResult,
  FeeExpectation,
  KnownIBANSummary,
  LegalHoursHistoryEntry,
  MatchSuggestion,
  Parent,
  UpdateHouseholdRequest,
} from '@/api/types';
import {
  ArrowLeft,
  Edit,
  Trash2,
  Loader2,
  User,
  Calendar,
  MapPin,
  Receipt,
  CheckCircle,
  Clock,
  AlertTriangle,
  AlertCircle,
  Check,
  Users,
  Plus,
  Link,
  Unlink,
  CreditCard,
  Home,
  Euro,
} from 'lucide-vue-next';
import ChildNotesCard from '@/components/child/ChildNotesCard.vue';
import ChildEditDialog from '@/components/child/ChildEditDialog.vue';
import ParentFormDialog from '@/components/child/ParentFormDialog.vue';
import TransactionDetailModal from '@/components/child/TransactionDetailModal.vue';
import AllocationModal from '@/components/child/AllocationModal.vue';
import ParentDetailModal from '@/components/child/ParentDetailModal.vue';
import { formatCurrency, formatCurrencyWhole, formatDate, formatMonthName } from '@/utils/format';
import {
  formatConfidence,
  formatMatchedBy,
  getFeeMatches,
  getFeeRemainingAmount,
  getFeeTypeName,
  getTxRemainingAmount,
  maskIban,
} from '@/utils/fees';
import {
  calculateAge,
  formatCareHours,
  getIncomeStatusLabel,
  incomeStatusOptions,
  isUnderThree,
} from '@/utils/child';

const route = useRoute();
const router = useRouter();

const child = ref<Child | null>(null);
const fees = ref<FeeExpectation[]>([]);
const isLoading = ref(true);
const error = ref<string | null>(null);

type EffectiveCareHours = {
  hours: number;
  effectiveFrom: string | null;
  isUpcoming: boolean;
};

// Dialogs rendered as sub-components (each resets its own state on open)
const showEditDialog = ref(false);
const showParentDialog = ref(false);
const parentDialogMode = ref<'create' | 'link'>('create');
const selectedFee = ref<FeeExpectation | null>(null);
const allocationSuggestion = ref<MatchSuggestion | null>(null);
const selectedParentForDetail = ref<Parent | null>(null);

// Care hours history
const careHoursHistory = ref<CareHoursHistoryEntry[]>([]);
const legalHoursHistory = ref<LegalHoursHistoryEntry[]>([]);

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

// Childcare fee calculation state
const childcareFee = ref<ChildcareFeeResult | null>(null);
const isLoadingChildcareFee = ref(false);
const childcareFeeCareHours = ref<EffectiveCareHours | null>(null);

// Trusted IBANs
const trustedIbans = ref<KnownIBANSummary[]>([]);
const isLoadingTrustedIbans = ref(false);

// Likely unmatched transactions
const likelyTransactions = ref<MatchSuggestion[]>([]);
const isLoadingLikelyTransactions = ref(false);
const likelyTransactionsError = ref<string | null>(null);
const likelyTransactionsScanned = ref(0);

const childId = computed(() => route.params.id as string);

async function loadChild() {
  isLoading.value = true;
  error.value = null;
  try {
    child.value = await api.getChild(childId.value);
    careHoursHistory.value = await api.getCareHoursHistory(childId.value);
    legalHoursHistory.value = await api.getLegalHoursHistory(childId.value);
    const feesResponse = await api.getFees({ childId: childId.value, perPage: 50 });
    fees.value = feesResponse.data;
    // Load childcare fee if applicable
    await loadChildcareFee();
    await loadTrustedIbans();
    await loadLikelyTransactions();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoading.value = false;
  }
}

async function loadTrustedIbans() {
  isLoadingTrustedIbans.value = true;
  try {
    const result = await api.getChildTrustedIBANs(childId.value);
    trustedIbans.value = result ?? [];
  } catch (e) {
    trustedIbans.value = [];
  } finally {
    isLoadingTrustedIbans.value = false;
  }
}

// Resolves the care hours the childcare fee must be based on.
// `child.careHours` only reflects the period that is effective today, so for children
// that have not started yet it is null. In that case the next upcoming care hours entry
// is used instead of silently assuming a default value.
// Returns the period that starts closest in the future, if any.
function nextUpcomingPeriod<T extends { effectiveFrom: string }>(entries: T[]): T | null {
  const now = Date.now();
  const upcoming = entries
    .filter((entry) => new Date(entry.effectiveFrom).getTime() > now)
    .sort((a, b) => new Date(a.effectiveFrom).getTime() - new Date(b.effectiveFrom).getTime());
  return upcoming[0] ?? null;
}

// The backend only reports the hours effective today, so children that start later have no
// current value. These computeds surface the already recorded upcoming period instead.
const upcomingCareHours = computed(() => {
  if (child.value?.careHours !== undefined && child.value?.careHours !== null) return null;
  return nextUpcomingPeriod(careHoursHistory.value.filter((entry) => entry.careHours !== undefined && entry.careHours !== null));
});

const upcomingLegalHours = computed(() => {
  if (child.value?.legalHours !== undefined && child.value?.legalHours !== null) return null;
  return nextUpcomingPeriod(legalHoursHistory.value.filter((entry) => entry.legalHours !== undefined && entry.legalHours !== null));
});

function resolveEffectiveCareHours(): EffectiveCareHours | null {
  if (!child.value) return null;

  const current = child.value.careHours;
  if (current !== undefined && current !== null) {
    return { hours: current, effectiveFrom: null, isUpcoming: false };
  }

  const upcoming = upcomingCareHours.value;
  if (upcoming && upcoming.careHours !== undefined && upcoming.careHours !== null) {
    return { hours: upcoming.careHours, effectiveFrom: upcoming.effectiveFrom, isUpcoming: true };
  }

  return null;
}

async function loadChildcareFee() {
  if (!child.value) return;

  childcareFeeCareHours.value = null;

  // Only calculate for U3 children (under 3 years)
  if (!isUnderThree(child.value.birthDate)) {
    childcareFee.value = null;
    return;
  }
  
  const household = child.value.household;
  if (!household) {
    childcareFee.value = null;
    return;
  }
  
  // Check income status - need income to calculate (except for MAX_ACCEPTED and FOSTER_FAMILY)
  const status = household.incomeStatus;
  if (!status || status === 'PENDING' || status === 'NOT_REQUIRED' || status === 'HISTORIC') {
    childcareFee.value = null;
    return;
  }

  // Without known care hours the fee is not calculable; never fall back to a default.
  const effectiveCareHours = resolveEffectiveCareHours();
  childcareFeeCareHours.value = effectiveCareHours;
  if (!effectiveCareHours) {
    childcareFee.value = null;
    return;
  }

  isLoadingChildcareFee.value = true;
  try {
    const isFosterFamily = status === 'FOSTER_FAMILY';
    const isHighestRate = status === 'MAX_ACCEPTED';
    const income = household.annualHouseholdIncome || 0;
    
    // Use childrenCountForFees if set, otherwise count all active children in household
    const siblingsCount = household.childrenCountForFees 
      ?? household.children?.filter(c => c.isActive).length 
      ?? 1;
    
    childcareFee.value = await api.calculateChildcareFee({
      income,
      childAgeType: 'krippe',
      siblingsCount,
      careHours: effectiveCareHours.hours,
      highestRate: isHighestRate,
      fosterFamily: isFosterFamily,
    });
  } catch (e) {
    console.error('Failed to calculate childcare fee:', e);
    childcareFee.value = null;
  } finally {
    isLoadingChildcareFee.value = false;
  }
}

async function loadLikelyTransactions() {
  isLoadingLikelyTransactions.value = true;
  likelyTransactionsError.value = null;
  try {
    const result = await api.getChildUnmatchedSuggestions(childId.value, {
      minConfidence: 0.6,
      limit: 10,
    });
    likelyTransactions.value = result.suggestions ?? [];
    likelyTransactionsScanned.value = result.scanned ?? 0;
  } catch (e) {
    likelyTransactionsError.value = e instanceof Error ? e.message : 'Transaktionen konnten nicht geladen werden';
  } finally {
    isLoadingLikelyTransactions.value = false;
  }
}

onMounted(loadChild);

// Reload when navigating between children (e.g., clicking sibling links)
watch(childId, () => {
  loadChild();
});

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
  const isPastDue = new Date(fee.dueDate) < new Date();
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
    <!-- Back button -->
    <button
      @click="router.push('/kinder')"
      class="flex items-center gap-2 text-gray-600 hover:text-gray-900 mb-6"
    >
      <ArrowLeft class="h-4 w-4" />
      Zurück zur Übersicht
    </button>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4">
      <p class="text-red-600">{{ error }}</p>
      <button @click="loadChild" class="mt-2 text-sm text-red-700 underline">
        Erneut versuchen
      </button>
    </div>

    <!-- Child details -->
    <div v-else-if="child">
      <!-- Header -->
      <div class="bg-white rounded-xl border p-6 mb-6">
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-4">
            <div class="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center">
              <User class="h-8 w-8 text-primary" />
            </div>
            <div>
              <h1 class="text-2xl font-bold text-gray-900">
                {{ child.firstName }} {{ child.lastName }}
              </h1>
              <p class="text-gray-600 font-mono">Mitglieds-Nr. {{ child.memberNumber }}</p>
              <div class="flex items-center gap-2 mt-2">
                <span
                  :class="[
                    'inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium',
                    child.isActive ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600',
                  ]"
                >
                  {{ child.isActive ? 'Aktiv' : 'Inaktiv' }}
                </span>
                <span
                  v-if="isUnderThree(child.birthDate)"
                  class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-amber-100 text-amber-700"
                >
                  U3
                </span>
              </div>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button
              @click="showEditDialog = true"
              class="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
              title="Bearbeiten"
            >
              <Edit class="h-5 w-5" />
            </button>
            <button
              @click="showDeleteDialog = true"
              class="p-2 text-red-500 hover:text-red-700 hover:bg-red-50 rounded-lg transition-colors"
              title="Löschen"
            >
              <Trash2 class="h-5 w-5" />
            </button>
          </div>
        </div>

        <!-- Info grid -->
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 mt-6 pt-6 border-t">
          <div class="flex items-start gap-3">
            <Calendar class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">Geburtsdatum</p>
              <p class="font-medium">{{ formatDate(child.birthDate) }}</p>
              <p class="text-sm text-gray-500">{{ calculateAge(child.birthDate) }} Jahre alt</p>
            </div>
          </div>
          <div class="flex items-start gap-3">
            <Calendar class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">Eintrittsdatum</p>
              <p class="font-medium">{{ formatDate(child.entryDate) }}</p>
            </div>
          </div>
          <div v-if="child.exitDate" class="flex items-start gap-3">
            <Calendar class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">Austrittsdatum</p>
              <p class="font-medium">{{ formatDate(child.exitDate) }}</p>
            </div>
          </div>
          <div v-if="child.street" class="flex items-start gap-3">
            <MapPin class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">Adresse</p>
              <p class="font-medium">{{ child.street }} {{ child.streetNo }}</p>
              <p class="text-sm text-gray-500">{{ child.postalCode }} {{ child.city }}</p>
            </div>
          </div>
          <div v-if="child.legalHours || child.careHours || legalHoursHistory.length > 0 || careHoursHistory.length > 0" class="flex items-start gap-3">
            <Clock class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">Betreuungszeiten</p>
              <p class="font-medium">
                Rechtsanspruch:
                <template v-if="upcomingLegalHours">
                  ab {{ formatDate(upcomingLegalHours.effectiveFrom) }}: {{ formatCareHours(upcomingLegalHours.legalHours) }}
                </template>
                <template v-else>
                  {{ formatCareHours(child.legalHours) }}
                  <span v-if="child.legalHoursUntil" class="text-sm text-gray-500">
                    (bis {{ formatDate(child.legalHoursUntil) }})
                  </span>
                </template>
              </p>
              <p class="font-medium">
                Betreuungszeit:
                <template v-if="upcomingCareHours">
                  ab {{ formatDate(upcomingCareHours.effectiveFrom) }}: {{ formatCareHours(upcomingCareHours.careHours) }}
                </template>
                <template v-else>{{ formatCareHours(child.careHours) }}</template>
              </p>
              <div v-if="legalHoursHistory.length > 0" class="mt-2 space-y-1">
                <p class="text-xs uppercase tracking-wide text-gray-400">Historie Rechtsanspruch</p>
                <div
                  v-for="entry in legalHoursHistory"
                  :key="entry.id"
                  class="flex items-center justify-between text-sm text-gray-600"
                >
                  <span>{{ formatHistoryRange(entry) }}</span>
                  <span class="font-medium text-gray-700">{{ formatCareHours(entry.legalHours) }}</span>
                </div>
              </div>
              <div v-if="careHoursHistory.length > 0" class="mt-2 space-y-1">
                <p class="text-xs uppercase tracking-wide text-gray-400">Historie Betreuungszeit</p>
                <div
                  v-for="entry in careHoursHistory"
                  :key="entry.id"
                  class="flex items-center justify-between text-sm text-gray-600"
                >
                  <span>{{ formatHistoryRange(entry) }}</span>
                  <span class="font-medium text-gray-700">{{ formatCareHours(entry.careHours) }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

      </div>

      <!-- Household & Income Section -->
      <div class="bg-white rounded-xl border p-6 mb-6">
        <div class="flex items-center justify-between mb-4">
          <div class="flex items-center gap-2">
            <Home class="h-5 w-5 text-primary" />
            <h2 class="text-lg font-semibold">Haushalt & Einkommen</h2>
          </div>
          <div class="flex items-center gap-2">
            <button
              @click="openLinkParentDialog"
              class="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium text-primary hover:bg-primary/10 rounded-md transition-colors"
              title="Vorhandenen Elternteil verknüpfen"
            >
              <Link class="h-3 w-3" />
              Verknüpfen
            </button>
            <button
              @click="openCreateParentDialog"
              class="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium bg-primary text-white hover:bg-primary/90 rounded-md transition-colors"
            >
              <Plus class="h-3 w-3" />
              Elternteil
            </button>
            <button
              v-if="child.household && !isEditingHousehold"
              @click="startEditingHousehold"
              class="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
              title="Bearbeiten"
            >
              <Edit class="h-4 w-4" />
            </button>
          </div>
        </div>

        <!-- Has Household -->
        <div v-if="child.household">
          <!-- View Mode -->
          <div v-if="!isEditingHousehold" class="space-y-4">
            <!-- Household Name -->
            <div>
              <p class="text-sm text-gray-500">Haushaltsname</p>
              <p class="font-medium">{{ child.household.name }}</p>
            </div>

            <!-- Income Status -->
            <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              <div>
                <p class="text-sm text-gray-500">Einkommensstatus</p>
                <p class="font-medium">{{ getIncomeStatusLabel(child.household.incomeStatus) }}</p>
                <router-link
                  v-if="!child.household.incomeStatus || child.household.incomeStatus === 'PENDING'"
                  :to="`/einstufungen/neu?childId=${child.id}`"
                  class="inline-flex items-center gap-1 mt-1 text-xs font-medium text-primary hover:text-primary/80 transition-colors"
                >
                  Einstufung erstellen →
                </router-link>
                <router-link
                  v-else
                  :to="`/einstufungen/neu?childId=${child.id}&followUp=1`"
                  class="inline-flex items-center gap-1 mt-1 text-xs font-medium text-primary hover:text-primary/80 transition-colors"
                >
                  Folgeeinstufung erstellen →
                </router-link>
              </div>
              <div v-if="child.household.incomeStatus === 'PROVIDED' || child.household.incomeStatus === 'HISTORIC'">
                <p class="text-sm text-gray-500">Jahreshaushaltseinkommen</p>
                <p class="font-medium">{{ formatCurrencyWhole(child.household.annualHouseholdIncome) }}</p>
              </div>
              <div v-if="child.household.childrenCountForFees">
                <p class="text-sm text-gray-500">Kinder (Beitragsberechnung)</p>
                <p class="font-medium">{{ child.household.childrenCountForFees }}</p>
              </div>
            </div>

            <div v-if="trustedIbans.length > 0" class="pt-3 border-t">
              <p class="text-sm text-gray-500 mb-2">Bekannte IBANs</p>
              <div class="flex flex-wrap gap-2">
                <span
                  v-for="iban in trustedIbans"
                  :key="iban.iban"
                  :title="iban.payerName ? `${iban.payerName} · ${iban.iban}` : iban.iban"
                  class="inline-flex items-center gap-1 px-2 py-1 bg-gray-50 border border-gray-200 rounded-full text-xs text-gray-700"
                >
                  <span class="font-mono">{{ maskIban(iban.iban) }}</span>
                  <span v-if="iban.transactionCount > 0" class="text-gray-500">· {{ iban.transactionCount }} Zahlungen</span>
                </span>
              </div>
            </div>

            <!-- Platzgeld (Childcare Fee) for U3 children -->
            <div v-if="isUnderThree(child.birthDate)" class="pt-4 border-t">
              <div class="flex items-start gap-3">
                <Euro class="h-5 w-5 text-primary mt-0.5" />
                <div class="flex-1">
                  <p class="text-sm text-gray-500">Monatliches Platzgeld</p>
                  <div v-if="isLoadingChildcareFee" class="flex items-center gap-2">
                    <Loader2 class="h-4 w-4 animate-spin text-gray-400" />
                    <span class="text-gray-400 text-sm">Berechne...</span>
                  </div>
                  <div v-else-if="childcareFee">
                    <p class="font-semibold text-lg text-primary">{{ formatCurrency(childcareFee.fee) }}</p>
                    <p class="text-sm text-gray-500">{{ childcareFee.rule }}</p>
                    <p v-if="childcareFeeCareHours" class="text-sm text-gray-500">
                      Basis: {{ formatCareHours(childcareFeeCareHours.hours) }}<span v-if="childcareFeeCareHours.effectiveFrom"> (ab {{ formatDate(childcareFeeCareHours.effectiveFrom) }})</span>
                    </p>
                    <p v-if="childcareFee.discountPercent > 0" class="text-sm text-green-600">
                      Geschwisterrabatt: {{ childcareFee.discountPercent }}%
                    </p>
                    <p v-if="childcareFee.notes && childcareFee.notes.length > 0" class="text-xs text-gray-400 mt-1">
                      {{ childcareFee.notes.join(' · ') }}
                    </p>
                  </div>
                  <div v-else>
                    <p class="text-gray-400 text-sm italic">
                      <span v-if="!child.household.incomeStatus || child.household.incomeStatus === 'PENDING'">
                        Einkommen noch nicht angegeben
                      </span>
                      <span v-else-if="child.household.incomeStatus === 'NOT_REQUIRED' || child.household.incomeStatus === 'HISTORIC'">
                        Nicht zutreffend
                      </span>
                      <span v-else-if="!childcareFeeCareHours">
                        Betreuungszeit nicht hinterlegt
                      </span>
                      <span v-else>
                        Kann nicht berechnet werden
                      </span>
                    </p>
                  </div>
                </div>
              </div>
            </div>

            <!-- Family Members -->
            <div v-if="householdParents.length > 0 || siblings.length > 0" class="pt-4 border-t">
              <p class="text-sm text-gray-500 mb-3">Familienmitglieder</p>
              
              <!-- Parents in Household -->
              <div v-if="householdParents.length > 0" class="mb-3">
                <p class="text-xs text-gray-400 uppercase tracking-wide mb-2">Eltern</p>
                <div class="flex flex-wrap gap-2">
                  <div
                    v-for="parent in householdParents"
                    :key="parent.id"
                    class="inline-flex items-center bg-blue-50 border border-blue-200 rounded-lg text-sm"
                  >
                    <button
                      @click="openParentDetailModal(parent)"
                      class="inline-flex items-center gap-2 px-3 py-1.5 hover:bg-blue-100 rounded-l-lg transition-colors"
                    >
                      <User class="h-4 w-4 text-blue-500" />
                      <span>{{ parent.firstName }} {{ parent.lastName }}</span>
                    </button>
                    <button
                      @click="confirmUnlinkParent(parent)"
                      class="p-1.5 text-blue-400 hover:text-red-500 hover:bg-red-50 rounded-r-lg border-l border-blue-200 transition-colors"
                      title="Verknüpfung aufheben"
                      aria-label="Verknüpfung aufheben"
                    >
                      <Unlink class="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
              </div>

              <!-- Siblings in Household -->
              <div v-if="siblings.length > 0">
                <p class="text-xs text-gray-400 uppercase tracking-wide mb-2">Geschwister</p>
                <div class="flex flex-wrap gap-2">
                  <router-link
                    v-for="sibling in siblings"
                    :key="sibling.id"
                    :to="`/kinder/${sibling.id}`"
                    class="inline-flex items-center gap-2 px-3 py-1.5 bg-amber-50 hover:bg-amber-100 border border-amber-200 rounded-lg text-sm transition-colors"
                  >
                    <User class="h-4 w-4 text-amber-500" />
                    <span>{{ sibling.firstName }} {{ sibling.lastName }}</span>
                  </router-link>
                </div>
              </div>
            </div>
          </div>

          <!-- Edit Mode -->
          <form v-else @submit.prevent="saveHouseholdEdit" class="space-y-4">
            <div>
              <label for="household-name" class="block text-sm font-medium text-gray-700 mb-1">Haushaltsname</label>
              <input
                id="household-name"
                v-model="householdEditForm.name"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>

            <div>
              <label for="household-incomeStatus" class="block text-sm font-medium text-gray-700 mb-1">Einkommensstatus</label>
              <select
                id="household-incomeStatus"
                v-model="householdEditForm.incomeStatus"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none bg-white"
              >
                <option v-for="option in incomeStatusOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </div>

            <div v-if="householdEditForm.incomeStatus === 'PROVIDED' || householdEditForm.incomeStatus === 'HISTORIC'">
              <label for="household-income" class="block text-sm font-medium text-gray-700 mb-1">Jahreshaushaltseinkommen</label>
              <input
                id="household-income"
                v-model.number="householdEditForm.annualHouseholdIncome"
                type="number"
                min="0"
                step="any"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>

            <div>
              <label for="household-childrenCount" class="block text-sm font-medium text-gray-700 mb-1">Anzahl Kinder (für Beitragsberechnung)</label>
              <input
                id="household-childrenCount"
                v-model.number="householdEditForm.childrenCountForFees"
                type="number"
                min="1"
                max="10"
                placeholder="Automatisch"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
              <p class="text-xs text-gray-500 mt-1">Leer lassen für automatische Zählung der U3-Kinder im Haushalt</p>
            </div>

            <div v-if="householdError" class="p-3 bg-red-50 border border-red-200 rounded-lg">
              <p class="text-sm text-red-600">{{ householdError }}</p>
            </div>

            <div class="flex justify-end gap-3 pt-2">
              <button
                type="button"
                @click="cancelEditingHousehold"
                class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
              >
                Abbrechen
              </button>
              <button
                type="submit"
                :disabled="isSavingHousehold"
                class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                <Loader2 v-if="isSavingHousehold" class="h-4 w-4 animate-spin" />
                <Check v-else class="h-4 w-4" />
                Speichern
              </button>
            </div>
          </form>
        </div>

        <!-- No Household -->
        <div v-else class="text-center py-6 bg-gray-50 rounded-lg border border-dashed">
          <Users class="h-8 w-8 text-gray-400 mx-auto mb-2" />
          <p class="text-gray-500 text-sm mb-1">Noch keine Eltern zugeordnet</p>
          <p class="text-gray-400 text-xs mb-4">Ein Haushalt wird automatisch erstellt, wenn der erste Elternteil verknüpft wird.</p>
          <div class="flex items-center justify-center gap-2">
            <button
              @click="openLinkParentDialog"
              class="inline-flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-primary border border-primary hover:bg-primary/10 rounded-lg transition-colors"
            >
              <Link class="h-4 w-4" />
              Verknüpfen
            </button>
            <button
              @click="openCreateParentDialog"
              class="inline-flex items-center gap-1 px-3 py-1.5 text-sm font-medium bg-primary text-white hover:bg-primary/90 rounded-lg transition-colors"
            >
              <Plus class="h-4 w-4" />
              Neu anlegen
            </button>
          </div>
        </div>
      </div>

      <ChildNotesCard :child-id="childId" />

      <!-- Fees section -->
      <div class="bg-white rounded-xl border p-6">
        <div class="flex items-center justify-between mb-6">
          <div class="flex items-center gap-2">
            <Receipt class="h-5 w-5 text-primary" />
            <h2 class="text-lg font-semibold">Beiträge</h2>
          </div>
        </div>

        <!-- Standard View (Open/Paid fees) -->
        <div>
          <!-- Likely unmatched transactions -->
          <div class="mb-6">
            <h3 class="text-sm font-medium text-gray-500 mb-3 flex items-center gap-2">
              <Receipt class="h-4 w-4" />
              Wahrscheinlich zugehörige Transaktionen
            </h3>

            <div v-if="isLoadingLikelyTransactions" class="flex items-center gap-2 text-sm text-gray-500">
              <Loader2 class="h-4 w-4 animate-spin" />
              Lade Vorschläge...
            </div>
            <div v-else-if="likelyTransactionsError" class="text-sm text-red-600">
              {{ likelyTransactionsError }}
            </div>
            <div v-else-if="likelyTransactions.length === 0" class="text-sm text-gray-500">
              Keine offenen Transaktionen mit hoher Wahrscheinlichkeit gefunden.
            </div>
            <div v-else class="space-y-2">
              <div
                v-for="suggestion in likelyTransactions"
                :key="suggestion.transaction.id"
                class="flex items-start justify-between gap-4 p-3 bg-blue-50 border border-blue-200 rounded-lg"
              >
                <div class="space-y-1">
                  <p class="font-medium text-blue-900">
                    {{ suggestion.transaction.payerName || 'Unbekannt' }}
                    <span class="text-xs text-blue-600 ml-2">· {{ formatDate(suggestion.transaction.bookingDate) }}</span>
                  </p>
                  <p v-if="suggestion.transaction.description" class="text-sm text-blue-800 break-words">
                    {{ suggestion.transaction.description }}
                  </p>
                  <div class="text-xs text-blue-700 flex items-center gap-2">
                    <span>Konfidenz: {{ formatConfidence(suggestion.confidence) }}</span>
                    <span>· Match: {{ formatMatchedBy(suggestion.matchedBy) }}</span>
                    <span v-if="formatSuggestionExpectation(suggestion)">
                      · Vorschlag: {{ formatSuggestionExpectation(suggestion) }}
                    </span>
                  </div>
                </div>
                <div class="text-right space-y-2">
                  <p class="font-semibold text-blue-900">{{ formatCurrency(suggestion.transaction.amount) }}</p>
                  <p
                    v-if="(suggestion.transaction.matchedAmount ?? 0) > 0"
                    class="text-xs text-amber-700"
                  >
                    Bereits zugeordnet: {{ formatCurrency(suggestion.transaction.matchedAmount ?? 0) }}
                    · Rest: {{ formatCurrency(getTxRemainingAmount(suggestion.transaction)) }}
                  </p>
                  <button
                    @click="openAllocationModal(suggestion)"
                    class="inline-flex items-center gap-1 px-2 py-1 text-xs text-blue-700 bg-white hover:bg-blue-100 border border-blue-200 rounded transition-colors"
                  >
                    Zuordnen
                  </button>
                </div>
              </div>
              <p v-if="likelyTransactionsScanned > 0" class="text-xs text-gray-400">
                {{ likelyTransactions.length }} Treffer aus {{ likelyTransactionsScanned }} offenen Transaktionen.
              </p>
            </div>
          </div>

          <!-- Open fees -->
          <div v-if="openFeeGroups.length > 0" class="mb-6">
            <h3 class="text-sm font-medium text-gray-500 mb-3 flex items-center gap-2">
              <Clock class="h-4 w-4" />
              Offene Beiträge ({{ openFeeGroups.length }})
            </h3>
            <div class="space-y-2">
              <div
                v-for="group in openFeeGroups"
                :key="group.fee.id"
                :class="[
                  'flex items-center justify-between p-3 rounded-lg',
                  group.fee.feeType === 'REMINDER' 
                    ? 'bg-red-50 border border-red-200' 
                    : 'bg-amber-50 border border-amber-200'
                ]"
              >
                <div class="flex items-center gap-3">
                  <AlertTriangle
                    v-if="new Date(group.fee.dueDate) < new Date()"
                    :class="group.fee.feeType === 'REMINDER' ? 'h-5 w-5 text-red-500' : 'h-5 w-5 text-red-500'"
                  />
                  <Clock v-else :class="group.fee.feeType === 'REMINDER' ? 'h-5 w-5 text-red-500' : 'h-5 w-5 text-amber-500'" />
                  <div>
                    <p :class="['font-medium', group.fee.feeType === 'REMINDER' ? 'text-red-700' : '']">{{ getFeeTypeName(group.fee.feeType) }}</p>
                    <p class="text-sm text-gray-600">
                      {{ group.fee.month ? formatMonthName(group.fee.month) + ' ' : '' }}{{ group.fee.year }}
                      · Fällig: {{ formatDate(group.fee.dueDate) }}
                    </p>
                    <p v-if="group.fee.matchedAmount && group.fee.matchedAmount > 0" class="text-xs text-amber-700">
                      Bereits bezahlt: {{ formatCurrency(group.fee.matchedAmount) }} · Rest: {{ formatCurrency(getFeeRemainingAmount(group.fee)) }}
                    </p>
                    <p
                      v-if="group.reminders.length > 0"
                      :class="[
                        'text-xs',
                        group.reminders.some(rem => !rem.isPaid) ? 'text-red-700' : 'text-green-700'
                      ]"
                    >
                      Mahngebühren: {{ getReminderSummary(group.reminders) }}
                    </p>
                    <p
                      v-if="group.fee.isPaid && group.reminders.some(rem => !rem.isPaid)"
                      class="text-xs text-amber-700"
                    >
                      Beitrag bezahlt · Mahngebühren offen
                    </p>
                  </div>
                </div>
                <div class="flex items-center gap-2">
                  <p :class="['font-semibold', group.fee.feeType === 'REMINDER' ? 'text-red-700' : '']">{{ formatCurrency(group.fee.amount) }}</p>
                  <button
                    v-if="canCreateReminder(group.fee)"
                    @click="openReminderDialog(group.fee)"
                    class="p-1.5 text-amber-600 hover:text-amber-800 hover:bg-amber-100 rounded-lg transition-colors"
                    title="Mahngebühr erstellen"
                  >
                    <AlertCircle class="h-4 w-4" />
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- Paid fees -->
          <div v-if="paidFeeGroups.length > 0">
            <h3 class="text-sm font-medium text-gray-500 mb-3 flex items-center gap-2">
              <CheckCircle class="h-4 w-4" />
              Bezahlte Beiträge ({{ paidFeeGroups.length }})
            </h3>
            <div class="space-y-2">
              <button
                v-for="group in paidFeeGroups"
                :key="group.fee.id"
                @click="openTransactionModal(group.fee)"
                :class="[
                  'w-full flex items-center justify-between p-3 bg-green-50 border border-green-200 rounded-lg text-left transition-colors',
                  getFeeMatches(group.fee).length > 0 ? 'hover:bg-green-100 cursor-pointer' : ''
                ]"
                :disabled="getFeeMatches(group.fee).length === 0"
              >
                <div class="flex items-center gap-3">
                  <CheckCircle class="h-5 w-5 text-green-500" />
                  <div>
                    <p class="font-medium">{{ getFeeTypeName(group.fee.feeType) }}</p>
                    <p class="text-sm text-gray-600">
                      {{ group.fee.month ? formatMonthName(group.fee.month) + ' ' : '' }}{{ group.fee.year }}
                      <span v-if="getPaymentSummary(group.fee)" class="text-green-600">
                        · {{ getPaymentSummary(group.fee) }}
                      </span>
                    </p>
                    <p v-if="group.reminders.length > 0" class="text-xs text-green-700">
                      Mahngebühren: {{ getReminderSummary(group.reminders) }}
                    </p>
                  </div>
                </div>
                <div class="flex items-center gap-2">
                  <p class="font-semibold text-green-700">{{ formatCurrency(group.fee.amount) }}</p>
                  <span
                    v-if="getFeeMatches(group.fee).length > 1"
                    class="text-xs font-medium text-green-700 bg-green-100 px-2 py-0.5 rounded-full"
                  >
                    {{ getFeeMatches(group.fee).length }}x
                  </span>
                  <CreditCard v-if="getFeeMatches(group.fee).length > 0" class="h-4 w-4 text-green-500" />
                </div>
              </button>
            </div>
          </div>

          <div v-if="fees.length === 0" class="text-center py-8 text-gray-500">
            Keine Beiträge vorhanden
          </div>
        </div>
      </div>
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
      <div class="bg-white rounded-xl shadow-xl w-full max-w-sm mx-4 p-6">
        <div class="flex items-center gap-3 mb-4">
          <div class="p-2 bg-red-100 rounded-lg">
            <Trash2 class="h-6 w-6 text-red-600" />
          </div>
          <h2 class="text-xl font-semibold">Kind löschen?</h2>
        </div>

        <p class="text-gray-600 mb-6">
          Möchtest du <strong>{{ child?.firstName }} {{ child?.lastName }}</strong> wirklich löschen?
          Diese Aktion kann nicht rückgängig gemacht werden.
        </p>

        <div class="flex justify-end gap-3">
          <button
            @click="showDeleteDialog = false"
            class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
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
      <div class="bg-white rounded-xl shadow-xl w-full max-w-sm mx-4 p-6">
        <div class="flex items-center gap-3 mb-4">
          <div class="p-2 bg-amber-100 rounded-lg">
            <Unlink class="h-6 w-6 text-amber-600" />
          </div>
          <h2 class="text-xl font-semibold">Verknüpfung aufheben?</h2>
        </div>

        <p class="text-gray-600 mb-6">
          Möchtest du die Verknüpfung zu <strong>{{ parentToUnlink?.firstName }} {{ parentToUnlink?.lastName }}</strong> aufheben?
          Der Elternteil wird nicht gelöscht, nur die Verknüpfung zu diesem Kind.
        </p>

        <div class="flex justify-end gap-3">
          <button
            @click="showUnlinkDialog = false"
            class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
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
      <div class="bg-white rounded-xl shadow-xl w-full max-w-sm mx-4 p-6">
        <div class="flex items-center gap-3 mb-4">
          <div class="p-2 bg-amber-100 rounded-lg">
            <AlertCircle class="h-6 w-6 text-amber-600" />
          </div>
          <h2 class="text-xl font-semibold">Mahngebühr erstellen?</h2>
        </div>

        <div class="mb-6">
          <p class="text-gray-600 mb-4">
            Möchtest du eine Mahngebühr für den folgenden überfälligen Beitrag erstellen?
          </p>
          <div class="p-3 bg-amber-50 border border-amber-200 rounded-lg">
            <p class="font-medium">{{ getFeeTypeName(reminderFee.feeType) }}</p>
            <p class="text-sm text-gray-600">
              {{ reminderFee.month ? formatMonthName(reminderFee.month) + ' ' : '' }}{{ reminderFee.year }}
              · {{ formatCurrency(reminderFee.amount) }}
            </p>
            <p class="text-sm text-red-600 mt-1">
              Fällig seit: {{ formatDate(reminderFee.dueDate) }}
            </p>
          </div>
          <p class="text-sm text-gray-500 mt-3">
            Es wird eine Mahngebühr von <strong>10,00 EUR</strong> erstellt.
          </p>
        </div>

        <div v-if="reminderError" class="p-3 bg-red-50 border border-red-200 rounded-lg mb-4">
          <p class="text-sm text-red-600">{{ reminderError }}</p>
        </div>

        <div class="flex justify-end gap-3">
          <button
            @click="showReminderDialog = false"
            class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
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
