<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { api } from '@/api';
import type { Child, FeeExpectation, UpdateChildRequest, Parent, CreateParentRequest, UpdateParentRequest, BankTransaction, IncomeStatus, UpdateHouseholdRequest, ChildcareFeeResult, MatchSuggestion, PaymentMatch, KnownIBANSummary } from '@/api/types';
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
  X,
  Check,
  Users,
  Plus,
  Link,
  Search,
  Unlink,
  CreditCard,
  Home,
  Euro,
} from 'lucide-vue-next';

const route = useRoute();
const router = useRouter();

const child = ref<Child | null>(null);
const fees = ref<FeeExpectation[]>([]);
const isLoading = ref(true);
const error = ref<string | null>(null);

// Edit dialog state
const showEditDialog = ref(false);
const editForm = ref<UpdateChildRequest>({});
const isEditing = ref(false);
const editError = ref<string | null>(null);

// Delete dialog state
const showDeleteDialog = ref(false);
const isDeleting = ref(false);

// Parent dialog state
const showParentDialog = ref(false);
const parentDialogMode = ref<'create' | 'link'>('create');
const parentForm = ref<CreateParentRequest>({
  firstName: '',
  lastName: '',
});
const isCreatingParent = ref(false);
const parentError = ref<string | null>(null);

// Link parent state
const searchQuery = ref('');
const searchResults = ref<Parent[]>([]);
const isSearching = ref(false);
const selectedParent = ref<Parent | null>(null);
const isLinking = ref(false);

// Unlink parent state
const parentToUnlink = ref<Parent | null>(null);
const showUnlinkDialog = ref(false);
const isUnlinking = ref(false);

// Transaction detail modal state
const selectedFee = ref<FeeExpectation | null>(null);
const selectedTransaction = ref<BankTransaction | null>(null);
const showTransactionModal = ref(false);
const transactionAction = ref<'unmatch' | 'delete' | null>(null);
const isUnmatchingTransaction = ref(false);
const isDeletingTransaction = ref(false);
const transactionActionError = ref<string | null>(null);

// Parent detail modal state
const showParentDetailModal = ref(false);
const selectedParentForDetail = ref<Parent | null>(null);
const isEditingParent = ref(false);
const parentEditForm = ref<UpdateParentRequest>({});
const isSavingParent = ref(false);
const parentDetailError = ref<string | null>(null);

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

// Trusted IBANs
const trustedIbans = ref<KnownIBANSummary[]>([]);
const isLoadingTrustedIbans = ref(false);

// Likely unmatched transactions
const likelyTransactions = ref<MatchSuggestion[]>([]);
const isLoadingLikelyTransactions = ref(false);
const likelyTransactionsError = ref<string | null>(null);
const likelyTransactionsScanned = ref(0);

// Allocation modal state
const showAllocationModal = ref(false);
const allocationSuggestion = ref<MatchSuggestion | null>(null);
const allocationRows = ref<{ fee: FeeExpectation; amount: number }[]>([]);
const allocationError = ref<string | null>(null);
const isAllocating = ref(false);

const childId = computed(() => route.params.id as string);

async function loadChild() {
  isLoading.value = true;
  error.value = null;
  try {
    child.value = await api.getChild(childId.value);
    const feesResponse = await api.getFees({ childId: childId.value, limit: 50 });
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

async function loadChildcareFee() {
  if (!child.value) return;
  
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
  
  isLoadingChildcareFee.value = true;
  try {
    const isFosterFamily = status === 'FOSTER_FAMILY';
    const isHighestRate = status === 'MAX_ACCEPTED';
    const income = household.annualHouseholdIncome || 0;
    
    // Use childrenCountForFees if set, otherwise count all active children in household
    const siblingsCount = household.childrenCountForFees 
      ?? household.children?.filter(c => c.isActive).length 
      ?? 1;
    
    // Get care hours from child
    const careHours = child.value.careHours || 30;
    
    childcareFee.value = await api.calculateChildcareFee({
      income,
      childAgeType: 'krippe',
      siblingsCount,
      careHours,
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

// ESC key handler to close all modals
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (showReminderDialog.value) {
      showReminderDialog.value = false;
    } else if (showAllocationModal.value) {
      closeAllocationModal();
    } else if (showTransactionModal.value) {
      closeTransactionModal();
    } else if (showParentDetailModal.value) {
      closeParentDetailModal();
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

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('de-DE');
}

function formatDateForInput(dateStr: string): string {
  return dateStr.split('T')[0];
}

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: 'EUR',
  }).format(amount);
}

function getFeeTypeName(type: string): string {
  switch (type) {
    case 'MEMBERSHIP':
      return 'Vereinsbeitrag';
    case 'FOOD':
      return 'Essensgeld';
    case 'CHILDCARE':
      return 'Platzgeld';
    case 'REMINDER':
      return 'Mahngebühr';
    default:
      return type;
  }
}

function getMonthName(month: number): string {
  return new Date(2000, month - 1).toLocaleString('de-DE', { month: 'long' });
}

function formatConfidence(confidence: number): string {
  return `${Math.round(confidence * 100)}%`;
}

function formatMatchedBy(reason?: string): string {
  switch (reason) {
    case 'trusted_iban':
      return 'IBAN (bekannt)';
    case 'member_number':
      return 'Mitgliedsnummer';
    case 'name':
      return 'Name';
    case 'parent_name':
      return 'Elternname';
    case 'combined':
      return 'Sammelzahlung';
    default:
      return 'Unbekannt';
  }
}

function formatSuggestionExpectation(suggestion: MatchSuggestion): string {
  const expectation = suggestion.expectation;
  if (!expectation) return '';
  const monthLabel = expectation.month ? `${getMonthName(expectation.month)} ` : '';
  return `${getFeeTypeName(expectation.feeType)} ${monthLabel}${expectation.year}`;
}

function calculateAge(birthDate: string): number {
  const birth = new Date(birthDate);
  const today = new Date();
  let age = today.getFullYear() - birth.getFullYear();
  const m = today.getMonth() - birth.getMonth();
  if (m < 0 || (m === 0 && today.getDate() < birth.getDate())) {
    age--;
  }
  return age;
}

function isUnderThree(birthDate: string): boolean {
  return calculateAge(birthDate) < 3;
}

function openEditDialog() {
  if (!child.value) return;
  editForm.value = {
    firstName: child.value.firstName,
    lastName: child.value.lastName,
    birthDate: formatDateForInput(child.value.birthDate),
    entryDate: formatDateForInput(child.value.entryDate),
    exitDate: child.value.exitDate ? formatDateForInput(child.value.exitDate) : undefined,
    street: child.value.street,
    streetNo: child.value.streetNo,
    postalCode: child.value.postalCode,
    city: child.value.city,
    legalHours: child.value.legalHours,
    legalHoursUntil: child.value.legalHoursUntil ? formatDateForInput(child.value.legalHoursUntil) : undefined,
    careHours: child.value.careHours,
    isActive: child.value.isActive,
  };
  editError.value = null;
  showEditDialog.value = true;
}

async function handleEdit() {
  if (!child.value) return;
  isEditing.value = true;
  editError.value = null;
  try {
    await api.updateChild(childId.value, editForm.value);
    await loadChild();
    showEditDialog.value = false;
  } catch (e) {
    editError.value = e instanceof Error ? e.message : 'Fehler beim Speichern';
  } finally {
    isEditing.value = false;
  }
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

// Parent dialog functions
function openCreateParentDialog() {
  parentDialogMode.value = 'create';
  parentForm.value = {
    firstName: '',
    lastName: '',
    email: '',
    phone: '',
    street: '',
    streetNo: '',
    postalCode: '',
    city: '',
  };
  parentError.value = null;
  showParentDialog.value = true;
}

function openLinkParentDialog() {
  parentDialogMode.value = 'link';
  searchQuery.value = '';
  searchResults.value = [];
  selectedParent.value = null;
  parentError.value = null;
  showParentDialog.value = true;
}

async function handleCreateParent() {
  if (!child.value) return;
  isCreatingParent.value = true;
  parentError.value = null;
  try {
    const newParent = await api.createParent(parentForm.value);
    await api.linkParent(childId.value, newParent.id, child.value.parents?.length === 0);
    await loadChild();
    showParentDialog.value = false;
  } catch (e) {
    parentError.value = e instanceof Error ? e.message : 'Fehler beim Erstellen';
  } finally {
    isCreatingParent.value = false;
  }
}

async function searchParents() {
  if (!searchQuery.value || searchQuery.value.length < 2) {
    searchResults.value = [];
    return;
  }
  isSearching.value = true;
  try {
    const response = await api.getParents({ search: searchQuery.value, limit: 10 });
    // Filter out parents already linked to this child
    const linkedIds = new Set(child.value?.parents?.map(p => p.id) || []);
    searchResults.value = response.data.filter(p => !linkedIds.has(p.id));
  } catch (e) {
    parentError.value = e instanceof Error ? e.message : 'Fehler bei der Suche';
  } finally {
    isSearching.value = false;
  }
}

// Debounce search
let searchTimeout: ReturnType<typeof setTimeout> | null = null;
watch(searchQuery, () => {
  if (searchTimeout) clearTimeout(searchTimeout);
  searchTimeout = setTimeout(searchParents, 300);
});

function selectParent(parent: Parent) {
  selectedParent.value = parent;
}

async function handleLinkParent() {
  if (!selectedParent.value || !child.value) return;
  isLinking.value = true;
  parentError.value = null;
  try {
    await api.linkParent(childId.value, selectedParent.value.id, child.value.parents?.length === 0);
    await loadChild();
    showParentDialog.value = false;
  } catch (e) {
    parentError.value = e instanceof Error ? e.message : 'Fehler beim Verknüpfen';
  } finally {
    isLinking.value = false;
  }
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
const paidFees = computed(() => fees.value.filter(f => f.isPaid));

function getFeeMatches(fee: FeeExpectation): PaymentMatch[] {
  if (fee.partialMatches && fee.partialMatches.length > 0) return fee.partialMatches;
  if (fee.matchedBy) return [fee.matchedBy];
  return [];
}

function openTransactionModal(fee: FeeExpectation) {
  const matches = getFeeMatches(fee);
  if (matches.length === 0) return;
  selectedFee.value = fee;
  selectedTransaction.value = matches[0].transaction ?? null;
  transactionAction.value = null;
  transactionActionError.value = null;
  showTransactionModal.value = true;
}

function closeTransactionModal() {
  showTransactionModal.value = false;
  selectedFee.value = null;
  selectedTransaction.value = null;
  transactionAction.value = null;
  transactionActionError.value = null;
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

function getFeeRemainingAmount(fee: FeeExpectation): number {
  const matched = fee.matchedAmount ?? 0;
  const remaining = fee.amount - matched;
  return remaining > 0 ? remaining : 0;
}

function maskIban(iban?: string): string {
  if (!iban) return '';
  const trimmed = iban.replace(/\s+/g, '');
  if (trimmed.length <= 8) return trimmed;
  return `${trimmed.slice(0, 4)}…${trimmed.slice(-4)}`;
}

const allocationTotal = computed(() =>
  allocationRows.value.reduce((sum, row) => sum + (row.amount || 0), 0)
);

const allocationRemaining = computed(() => {
  const total = allocationSuggestion.value?.transaction.amount ?? 0;
  return total - allocationTotal.value;
});

function clampAllocationAmount(amount: number, fee: FeeExpectation): number {
  const maxFee = getFeeRemainingAmount(fee);
  const maxTx = allocationSuggestion.value?.transaction.amount ?? 0;
  if (amount <= 0) return 0;
  return Math.min(amount, maxFee, maxTx);
}

function assignOnlyToFee(feeId: string): void {
  const row = allocationRows.value.find(item => item.fee.id === feeId);
  if (!row) return;
  const amount = clampAllocationAmount(getFeeRemainingAmount(row.fee), row.fee);
  allocationRows.value = allocationRows.value.map(item => ({
    ...item,
    amount: item.fee.id === feeId ? amount : 0,
  }));
}

function assignRemainingToFee(feeId: string): void {
  const row = allocationRows.value.find(item => item.fee.id === feeId);
  if (!row) return;
  const remaining = allocationRemaining.value + (row.amount || 0);
  row.amount = clampAllocationAmount(remaining, row.fee);
}

function openAllocationModal(suggestion: MatchSuggestion): void {
  allocationSuggestion.value = suggestion;
  const rows = openFees.value.map(fee => ({ fee, amount: 0 }));
  let remaining = suggestion.transaction.amount;

  const applyAllocation = (feeId: string, desiredAmount: number) => {
    const row = rows.find(item => item.fee.id === feeId);
    if (!row || remaining <= 0) {
      return;
    }
    const maxAmount = getFeeRemainingAmount(row.fee);
    const amount = Math.min(desiredAmount, maxAmount, remaining);
    if (amount > 0) {
      row.amount = amount;
      remaining -= amount;
    }
  };

  if (suggestion.expectations && suggestion.expectations.length > 0) {
    for (const expectation of suggestion.expectations) {
      applyAllocation(expectation.id, expectation.amount);
    }
  } else if (suggestion.expectation) {
    applyAllocation(suggestion.expectation.id, suggestion.expectation.amount);
  }

  allocationRows.value = rows;
  allocationError.value = null;
  showAllocationModal.value = true;
}

function closeAllocationModal(): void {
  showAllocationModal.value = false;
  allocationSuggestion.value = null;
  allocationRows.value = [];
  allocationError.value = null;
}

async function confirmAllocation(): Promise<void> {
  if (!allocationSuggestion.value) return;
  const allocations = allocationRows.value
    .filter(row => row.amount > 0)
    .map(row => ({
      expectationId: row.fee.id,
      amount: row.amount,
    }));

  if (allocations.length === 0) {
    allocationError.value = 'Bitte mindestens einen Betrag zuordnen.';
    return;
  }

  if (allocationRemaining.value < -0.01) {
    allocationError.value = 'Die Summe übersteigt den Transaktionsbetrag.';
    return;
  }

  isAllocating.value = true;
  allocationError.value = null;
  try {
    await api.allocateTransaction(allocationSuggestion.value.transaction.id, allocations);
    await loadChild();
    await loadLikelyTransactions();
    closeAllocationModal();
  } catch (e) {
    allocationError.value = e instanceof Error ? e.message : 'Zuordnung fehlgeschlagen';
  } finally {
    isAllocating.value = false;
  }
}

function requestTransactionAction(action: 'unmatch' | 'delete'): void {
  transactionAction.value = action;
  transactionActionError.value = null;
}

function cancelTransactionAction(): void {
  transactionAction.value = null;
  transactionActionError.value = null;
}

async function confirmTransactionAction(): Promise<void> {
  if (!selectedTransaction.value || !transactionAction.value) return;
  const deleteTransaction = transactionAction.value === 'delete';
  if (deleteTransaction) {
    isDeletingTransaction.value = true;
  } else {
    isUnmatchingTransaction.value = true;
  }
  transactionActionError.value = null;
  try {
    await api.unmatchTransaction(selectedTransaction.value.id, { deleteTransaction });
    await loadChild();
    await loadLikelyTransactions();
    closeTransactionModal();
  } catch (e) {
    transactionActionError.value = e instanceof Error ? e.message : 'Aktion fehlgeschlagen';
  } finally {
    if (deleteTransaction) {
      isDeletingTransaction.value = false;
    } else {
      isUnmatchingTransaction.value = false;
    }
  }
}

// Parent detail modal functions
function openParentDetailModal(parent: Parent) {
  selectedParentForDetail.value = parent;
  isEditingParent.value = false;
  parentDetailError.value = null;
  showParentDetailModal.value = true;
}

function closeParentDetailModal() {
  showParentDetailModal.value = false;
  selectedParentForDetail.value = null;
  isEditingParent.value = false;
  parentEditForm.value = {};
  parentDetailError.value = null;
}

function startEditingParent() {
  if (!selectedParentForDetail.value) return;
  parentEditForm.value = {
    firstName: selectedParentForDetail.value.firstName,
    lastName: selectedParentForDetail.value.lastName,
    birthDate: selectedParentForDetail.value.birthDate ? formatDateForInput(selectedParentForDetail.value.birthDate) : undefined,
    email: selectedParentForDetail.value.email,
    phone: selectedParentForDetail.value.phone,
    street: selectedParentForDetail.value.street,
    streetNo: selectedParentForDetail.value.streetNo,
    postalCode: selectedParentForDetail.value.postalCode,
    city: selectedParentForDetail.value.city,
  };
  isEditingParent.value = true;
}

function cancelEditingParent() {
  isEditingParent.value = false;
  parentEditForm.value = {};
  parentDetailError.value = null;
}

async function saveParentEdit() {
  if (!selectedParentForDetail.value) return;
  isSavingParent.value = true;
  parentDetailError.value = null;
  try {
    const updated = await api.updateParent(selectedParentForDetail.value.id, parentEditForm.value);
    selectedParentForDetail.value = updated;
    isEditingParent.value = false;
    // Reload child to update the parent list
    await loadChild();
  } catch (e) {
    parentDetailError.value = e instanceof Error ? e.message : 'Fehler beim Speichern';
  } finally {
    isSavingParent.value = false;
  }
}

function formatIncome(income?: number): string {
  if (income === undefined || income === null) return '-';
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: 'EUR',
    maximumFractionDigits: 0,
  }).format(income);
}

function getIncomeStatusLabel(status?: IncomeStatus): string {
  switch (status) {
    case 'PROVIDED':
      return 'Einkommen angegeben';
    case 'MAX_ACCEPTED':
      return 'Höchstsatz akzeptiert';
    case 'PENDING':
      return 'Dokumente ausstehend';
    case 'NOT_REQUIRED':
      return 'Nicht erforderlich (Kind >3J bei Eintritt)';
    case 'HISTORIC':
      return 'Historisch (Kind jetzt >3J)';
    case 'FOSTER_FAMILY':
      return 'Pflegefamilie (Durchschnittsbeitrag)';
    default:
      return 'Nicht festgelegt';
  }
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

const incomeStatusOptions: { value: IncomeStatus; label: string }[] = [
  { value: '', label: 'Nicht festgelegt' },
  { value: 'PROVIDED', label: 'Einkommen angegeben' },
  { value: 'MAX_ACCEPTED', label: 'Höchstsatz akzeptiert' },
  { value: 'PENDING', label: 'Dokumente ausstehend' },
  { value: 'NOT_REQUIRED', label: 'Nicht erforderlich (Kind >3J bei Eintritt)' },
  { value: 'HISTORIC', label: 'Historisch (Kind jetzt >3J)' },
  { value: 'FOSTER_FAMILY', label: 'Pflegefamilie (Durchschnittsbeitrag)' },
];

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
              @click="openEditDialog"
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
          <div v-if="child.legalHours || child.careHours" class="flex items-start gap-3">
            <Clock class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">Betreuungszeiten</p>
              <p v-if="child.legalHours" class="font-medium">
                Rechtsanspruch: {{ child.legalHours }} Std./Woche
                <span v-if="child.legalHoursUntil" class="text-sm text-gray-500">
                  (bis {{ formatDate(child.legalHoursUntil) }})
                </span>
              </p>
              <p v-if="child.careHours" class="font-medium">
                Betreuungszeit: {{ child.careHours }} Std./Woche
              </p>
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
              </div>
              <div v-if="child.household.incomeStatus === 'PROVIDED' || child.household.incomeStatus === 'HISTORIC'">
                <p class="text-sm text-gray-500">Jahreshaushaltseinkommen</p>
                <p class="font-medium">{{ formatIncome(child.household.annualHouseholdIncome) }}</p>
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
          <div v-if="openFees.length > 0" class="mb-6">
            <h3 class="text-sm font-medium text-gray-500 mb-3 flex items-center gap-2">
              <Clock class="h-4 w-4" />
              Offene Beiträge ({{ openFees.length }})
            </h3>
            <div class="space-y-2">
              <div
                v-for="fee in openFees"
                :key="fee.id"
                :class="[
                  'flex items-center justify-between p-3 rounded-lg',
                  fee.feeType === 'REMINDER' 
                    ? 'bg-red-50 border border-red-200' 
                    : 'bg-amber-50 border border-amber-200'
                ]"
              >
                <div class="flex items-center gap-3">
                  <AlertTriangle
                    v-if="new Date(fee.dueDate) < new Date()"
                    :class="fee.feeType === 'REMINDER' ? 'h-5 w-5 text-red-500' : 'h-5 w-5 text-red-500'"
                  />
                  <Clock v-else :class="fee.feeType === 'REMINDER' ? 'h-5 w-5 text-red-500' : 'h-5 w-5 text-amber-500'" />
                  <div>
                    <p :class="['font-medium', fee.feeType === 'REMINDER' ? 'text-red-700' : '']">{{ getFeeTypeName(fee.feeType) }}</p>
                    <p class="text-sm text-gray-600">
                      {{ fee.month ? getMonthName(fee.month) + ' ' : '' }}{{ fee.year }}
                      · Fällig: {{ formatDate(fee.dueDate) }}
                    </p>
                    <p v-if="fee.matchedAmount && fee.matchedAmount > 0" class="text-xs text-amber-700">
                      Bereits bezahlt: {{ formatCurrency(fee.matchedAmount) }} · Rest: {{ formatCurrency(getFeeRemainingAmount(fee)) }}
                    </p>
                  </div>
                </div>
                <div class="flex items-center gap-2">
                  <p :class="['font-semibold', fee.feeType === 'REMINDER' ? 'text-red-700' : '']">{{ formatCurrency(fee.amount) }}</p>
                  <button
                    v-if="canCreateReminder(fee)"
                    @click="openReminderDialog(fee)"
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
          <div v-if="paidFees.length > 0">
            <h3 class="text-sm font-medium text-gray-500 mb-3 flex items-center gap-2">
              <CheckCircle class="h-4 w-4" />
              Bezahlte Beiträge ({{ paidFees.length }})
            </h3>
            <div class="space-y-2">
              <button
                v-for="fee in paidFees"
                :key="fee.id"
                @click="openTransactionModal(fee)"
                :class="[
                  'w-full flex items-center justify-between p-3 bg-green-50 border border-green-200 rounded-lg text-left transition-colors',
                  getFeeMatches(fee).length > 0 ? 'hover:bg-green-100 cursor-pointer' : ''
                ]"
                :disabled="getFeeMatches(fee).length === 0"
              >
                <div class="flex items-center gap-3">
                  <CheckCircle class="h-5 w-5 text-green-500" />
                  <div>
                    <p class="font-medium">{{ getFeeTypeName(fee.feeType) }}</p>
                    <p class="text-sm text-gray-600">
                      {{ fee.month ? getMonthName(fee.month) + ' ' : '' }}{{ fee.year }}
                      <span v-if="getPaymentSummary(fee)" class="text-green-600">
                        · {{ getPaymentSummary(fee) }}
                      </span>
                    </p>
                  </div>
                </div>
                <div class="flex items-center gap-2">
                  <p class="font-semibold text-green-700">{{ formatCurrency(fee.amount) }}</p>
                  <span
                    v-if="getFeeMatches(fee).length > 1"
                    class="text-xs font-medium text-green-700 bg-green-100 px-2 py-0.5 rounded-full"
                  >
                    {{ getFeeMatches(fee).length }}x
                  </span>
                  <CreditCard v-if="getFeeMatches(fee).length > 0" class="h-4 w-4 text-green-500" />
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

    <!-- Edit Dialog -->
    <div
      v-if="showEditDialog"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="showEditDialog = false"
    >
      <div class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4 p-6 max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between mb-6">
          <h2 class="text-xl font-semibold">Kind bearbeiten</h2>
          <button @click="showEditDialog = false" class="p-1 hover:bg-gray-100 rounded">
            <X class="h-5 w-5" />
          </button>
        </div>

        <form @submit.prevent="handleEdit" class="space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="edit-firstName" class="block text-sm font-medium text-gray-700 mb-1">Vorname</label>
              <input
                id="edit-firstName"
                v-model="editForm.firstName"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div>
              <label for="edit-lastName" class="block text-sm font-medium text-gray-700 mb-1">Nachname</label>
              <input
                id="edit-lastName"
                v-model="editForm.lastName"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="edit-birthDate" class="block text-sm font-medium text-gray-700 mb-1">Geburtsdatum</label>
              <input
                id="edit-birthDate"
                v-model="editForm.birthDate"
                type="date"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div>
              <label for="edit-entryDate" class="block text-sm font-medium text-gray-700 mb-1">Eintrittsdatum</label>
              <input
                id="edit-entryDate"
                v-model="editForm.entryDate"
                type="date"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div>
            <label for="edit-exitDate" class="block text-sm font-medium text-gray-700 mb-1">Austrittsdatum</label>
            <input
              id="edit-exitDate"
              v-model="editForm.exitDate"
              type="date"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
            <p class="text-xs text-gray-500 mt-1">Optional: Datum, an dem das Kind die Kita verlässt</p>
          </div>

          <div class="grid grid-cols-4 gap-4">
            <div class="col-span-3">
              <label for="edit-street" class="block text-sm font-medium text-gray-700 mb-1">Straße</label>
              <input
                id="edit-street"
                v-model="editForm.street"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div>
              <label for="edit-streetNo" class="block text-sm font-medium text-gray-700 mb-1">Hausnr.</label>
              <input
                id="edit-streetNo"
                v-model="editForm.streetNo"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div class="grid grid-cols-3 gap-4">
            <div>
              <label for="edit-postalCode" class="block text-sm font-medium text-gray-700 mb-1">PLZ</label>
              <input
                id="edit-postalCode"
                v-model="editForm.postalCode"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div class="col-span-2">
              <label for="edit-city" class="block text-sm font-medium text-gray-700 mb-1">Ort</label>
              <input
                id="edit-city"
                v-model="editForm.city"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <!-- Care Hours Section -->
          <div class="pt-4 border-t">
            <h3 class="text-sm font-medium text-gray-700 mb-3">Betreuungszeiten</h3>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label for="edit-legalHours" class="block text-sm font-medium text-gray-700 mb-1">Rechtsanspruch (Std./Woche)</label>
                <input
                  id="edit-legalHours"
                  v-model.number="editForm.legalHours"
                  type="number"
                  min="0"
                  max="50"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
                />
              </div>
              <div>
                <label for="edit-legalHoursUntil" class="block text-sm font-medium text-gray-700 mb-1">Rechtsanspruch bis</label>
                <input
                  id="edit-legalHoursUntil"
                  v-model="editForm.legalHoursUntil"
                  type="date"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
                />
              </div>
            </div>
            <div class="mt-4">
              <label for="edit-careHours" class="block text-sm font-medium text-gray-700 mb-1">Betreuungszeit (Std./Woche)</label>
              <input
                id="edit-careHours"
                v-model.number="editForm.careHours"
                type="number"
                min="0"
                max="50"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
              <p class="text-xs text-gray-500 mt-1">Vereinbarte wöchentliche Betreuungszeit mit der Kita</p>
            </div>
          </div>

          <div>
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                v-model="editForm.isActive"
                type="checkbox"
                class="w-4 h-4 text-primary rounded border-gray-300 focus:ring-primary"
              />
              <span class="text-sm text-gray-700">Kind ist aktiv</span>
            </label>
          </div>

          <div v-if="editError" class="p-3 bg-red-50 border border-red-200 rounded-lg">
            <p class="text-sm text-red-600">{{ editError }}</p>
          </div>

          <div class="flex justify-end gap-3 pt-4">
            <button
              type="button"
              @click="showEditDialog = false"
              class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
            >
              Abbrechen
            </button>
            <button
              type="submit"
              :disabled="isEditing"
              class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Loader2 v-if="isEditing" class="h-4 w-4 animate-spin" />
              <Check v-else class="h-4 w-4" />
              Speichern
            </button>
          </div>
        </form>
      </div>
    </div>

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

    <!-- Parent Dialog (Create or Link) -->
    <div
      v-if="showParentDialog"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="showParentDialog = false"
    >
      <div class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4 p-6 max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between mb-6">
          <h2 class="text-xl font-semibold">
            {{ parentDialogMode === 'create' ? 'Elternteil anlegen' : 'Elternteil verknüpfen' }}
          </h2>
          <button @click="showParentDialog = false" class="p-1 hover:bg-gray-100 rounded">
            <X class="h-5 w-5" />
          </button>
        </div>

        <!-- Mode Tabs -->
        <div class="flex gap-2 mb-6 p-1 bg-gray-100 rounded-lg">
          <button
            @click="parentDialogMode = 'create'"
            :class="[
              'flex-1 py-2 px-3 text-sm font-medium rounded-md transition-colors',
              parentDialogMode === 'create'
                ? 'bg-white text-primary shadow-sm'
                : 'text-gray-600 hover:text-gray-900'
            ]"
          >
            Neu anlegen
          </button>
          <button
            @click="parentDialogMode = 'link'"
            :class="[
              'flex-1 py-2 px-3 text-sm font-medium rounded-md transition-colors',
              parentDialogMode === 'link'
                ? 'bg-white text-primary shadow-sm'
                : 'text-gray-600 hover:text-gray-900'
            ]"
          >
            Vorhandenen verknüpfen
          </button>
        </div>

        <!-- Create Form -->
        <form v-if="parentDialogMode === 'create'" @submit.prevent="handleCreateParent" class="space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="parent-firstName" class="block text-sm font-medium text-gray-700 mb-1">Vorname *</label>
              <input
                id="parent-firstName"
                v-model="parentForm.firstName"
                type="text"
                required
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div>
              <label for="parent-lastName" class="block text-sm font-medium text-gray-700 mb-1">Nachname *</label>
              <input
                id="parent-lastName"
                v-model="parentForm.lastName"
                type="text"
                required
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div>
            <label for="parent-email" class="block text-sm font-medium text-gray-700 mb-1">E-Mail</label>
            <input
              id="parent-email"
              v-model="parentForm.email"
              type="email"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>

          <div>
            <label for="parent-phone" class="block text-sm font-medium text-gray-700 mb-1">Telefon</label>
            <input
              id="parent-phone"
              v-model="parentForm.phone"
              type="tel"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>

          <div class="grid grid-cols-4 gap-4">
            <div class="col-span-3">
              <label for="parent-street" class="block text-sm font-medium text-gray-700 mb-1">Straße</label>
              <input
                id="parent-street"
                v-model="parentForm.street"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div>
              <label for="parent-streetNo" class="block text-sm font-medium text-gray-700 mb-1">Hausnr.</label>
              <input
                id="parent-streetNo"
                v-model="parentForm.streetNo"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div class="grid grid-cols-3 gap-4">
            <div>
              <label for="parent-postalCode" class="block text-sm font-medium text-gray-700 mb-1">PLZ</label>
              <input
                id="parent-postalCode"
                v-model="parentForm.postalCode"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div class="col-span-2">
              <label for="parent-city" class="block text-sm font-medium text-gray-700 mb-1">Ort</label>
              <input
                id="parent-city"
                v-model="parentForm.city"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div v-if="parentError" class="p-3 bg-red-50 border border-red-200 rounded-lg">
            <p class="text-sm text-red-600">{{ parentError }}</p>
          </div>

          <div class="flex justify-end gap-3 pt-4">
            <button
              type="button"
              @click="showParentDialog = false"
              class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
            >
              Abbrechen
            </button>
            <button
              type="submit"
              :disabled="isCreatingParent"
              class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Loader2 v-if="isCreatingParent" class="h-4 w-4 animate-spin" />
              <Plus v-else class="h-4 w-4" />
              Anlegen & Verknüpfen
            </button>
          </div>
        </form>

        <!-- Link Form -->
        <div v-else class="space-y-4">
          <div>
            <label for="parent-search" class="block text-sm font-medium text-gray-700 mb-1">Elternteil suchen</label>
            <div class="relative">
              <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
              <input
                id="parent-search"
                v-model="searchQuery"
                type="text"
                placeholder="Name eingeben..."
                class="w-full pl-10 pr-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
              <Loader2 v-if="isSearching" class="absolute right-3 top-1/2 -translate-y-1/2 h-4 w-4 animate-spin text-gray-400" />
            </div>
          </div>

          <!-- Search Results -->
          <div v-if="searchResults.length > 0" class="border rounded-lg divide-y max-h-60 overflow-y-auto">
            <button
              v-for="parent in searchResults"
              :key="parent.id"
              @click="selectParent(parent)"
              :class="[
                'w-full p-3 text-left hover:bg-gray-50 transition-colors',
                selectedParent?.id === parent.id ? 'bg-primary/5 border-l-2 border-l-primary' : ''
              ]"
            >
              <p class="font-medium">{{ parent.firstName }} {{ parent.lastName }}</p>
              <p v-if="parent.email" class="text-sm text-gray-500">{{ parent.email }}</p>
            </button>
          </div>

          <div v-else-if="searchQuery.length >= 2 && !isSearching" class="text-center py-6 text-gray-500 text-sm">
            Keine Eltern gefunden
          </div>

          <div v-else-if="searchQuery.length < 2" class="text-center py-6 text-gray-500 text-sm">
            Mindestens 2 Zeichen eingeben
          </div>

          <!-- Selected Parent Preview -->
          <div v-if="selectedParent" class="p-4 bg-primary/5 border border-primary/20 rounded-lg">
            <p class="text-sm text-gray-500 mb-1">Ausgewählt:</p>
            <p class="font-medium">{{ selectedParent.firstName }} {{ selectedParent.lastName }}</p>
            <p v-if="selectedParent.email" class="text-sm text-gray-600">{{ selectedParent.email }}</p>
          </div>

          <div v-if="parentError" class="p-3 bg-red-50 border border-red-200 rounded-lg">
            <p class="text-sm text-red-600">{{ parentError }}</p>
          </div>

          <div class="flex justify-end gap-3 pt-4">
            <button
              type="button"
              @click="showParentDialog = false"
              class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
            >
              Abbrechen
            </button>
            <button
              @click="handleLinkParent"
              :disabled="!selectedParent || isLinking"
              class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Loader2 v-if="isLinking" class="h-4 w-4 animate-spin" />
              <Link v-else class="h-4 w-4" />
              Verknüpfen
            </button>
          </div>
        </div>
      </div>
    </div>

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

    <!-- Transaction Detail Modal -->
    <div
      v-if="showTransactionModal && selectedFee"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="closeTransactionModal"
    >
      <div class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4 p-6">
        <div class="flex items-center justify-between mb-6">
          <div class="flex items-center gap-3">
            <div class="p-2 bg-green-100 rounded-lg">
              <CreditCard class="h-6 w-6 text-green-600" />
            </div>
            <h2 class="text-xl font-semibold">Transaktionsdetails</h2>
          </div>
          <button @click="closeTransactionModal" class="p-1 hover:bg-gray-100 rounded">
            <X class="h-5 w-5" />
          </button>
        </div>

        <div v-if="selectedFee && getFeeMatches(selectedFee).length > 1" class="mb-6">
          <p class="text-sm font-medium text-gray-600 mb-2">Zahlungen ({{ getFeeMatches(selectedFee).length }})</p>
          <div class="space-y-2">
            <button
              v-for="match in getFeeMatches(selectedFee)"
              :key="match.id"
              @click="selectedTransaction = match.transaction ?? null"
              class="w-full flex items-center justify-between p-2 border rounded-lg text-left hover:bg-gray-50"
            >
              <div>
                <p class="text-sm font-medium">{{ match.transaction?.payerName || 'Unbekannt' }}</p>
                <p class="text-xs text-gray-500">
                  {{ match.transaction?.bookingDate ? formatDate(match.transaction.bookingDate) : 'Kein Datum' }}
                </p>
              </div>
              <div class="text-sm font-semibold text-green-600">
                {{ formatCurrency(match.amount) }}
              </div>
            </button>
          </div>
          <p v-if="!selectedTransaction" class="text-xs text-gray-500 mt-2">
            Wähle eine Zahlung, um die Details anzuzeigen.
          </p>
        </div>

        <div v-if="selectedTransaction" class="space-y-4">
          <div>
            <p class="text-sm text-gray-500">Zahler</p>
            <p class="font-medium">{{ selectedTransaction.payerName || 'Unbekannt' }}</p>
          </div>

          <div>
            <p class="text-sm text-gray-500">Buchungsdatum</p>
            <p class="font-medium">{{ formatDate(selectedTransaction.bookingDate) }}</p>
          </div>

          <div v-if="selectedTransaction.payerIban">
            <p class="text-sm text-gray-500">IBAN</p>
            <p class="font-mono text-sm">{{ selectedTransaction.payerIban }}</p>
          </div>

          <div v-if="selectedTransaction.description">
            <p class="text-sm text-gray-500">Verwendungszweck</p>
            <p class="text-sm text-gray-700 break-words">{{ selectedTransaction.description }}</p>
          </div>

          <div>
            <p class="text-sm text-gray-500">Betrag</p>
            <p class="font-semibold text-green-600 text-lg">{{ formatCurrency(selectedTransaction.amount) }}</p>
          </div>

          <div>
            <p class="text-sm text-gray-500">Importiert am</p>
            <p class="text-sm text-gray-600">{{ formatDate(selectedTransaction.importedAt) }}</p>
          </div>
        </div>

        <div class="mt-6 space-y-3">
          <div
            v-if="transactionAction"
            :class="[
              'p-3 rounded-lg text-sm',
              transactionAction === 'delete' ? 'bg-red-50 text-red-800' : 'bg-amber-50 text-amber-800'
            ]"
          >
            <p class="font-medium">
              {{ transactionAction === 'delete'
                ? 'Transaktion wirklich löschen?'
                : 'Zuordnung wirklich aufheben?' }}
            </p>
            <p class="text-xs mt-1">
              {{ transactionAction === 'delete'
                ? 'Die Transaktion wird gelöscht (inklusive aller Zuordnungen).'
                : 'Der Beitrag wird wieder als offen geführt. Falls mehrere Beiträge mit der Transaktion verknüpft sind, werden alle Zuordnungen aufgehoben.' }}
            </p>
            <div class="flex justify-end gap-2 mt-3">
              <button
                @click="confirmTransactionAction"
                :disabled="isUnmatchingTransaction || isDeletingTransaction"
                :class="[
                  'px-3 py-1.5 text-xs text-white rounded transition-colors disabled:opacity-50',
                  transactionAction === 'delete' ? 'bg-red-600 hover:bg-red-700' : 'bg-amber-600 hover:bg-amber-700'
                ]"
              >
                <Loader2 v-if="isUnmatchingTransaction || isDeletingTransaction" class="h-3 w-3 animate-spin" />
                <span v-else>Ja</span>
              </button>
              <button
                @click="cancelTransactionAction"
                class="px-3 py-1.5 text-xs bg-gray-200 text-gray-700 rounded hover:bg-gray-300"
              >
                Nein
              </button>
            </div>
          </div>

          <p v-if="transactionActionError" class="text-sm text-red-600">
            {{ transactionActionError }}
          </p>

          <div v-if="!transactionAction && selectedTransaction" class="flex flex-col gap-2">
            <button
              @click="requestTransactionAction('unmatch')"
              :disabled="isUnmatchingTransaction || isDeletingTransaction"
              class="inline-flex items-center gap-2 px-3 py-2 text-sm text-amber-700 bg-amber-50 hover:bg-amber-100 rounded-lg transition-colors disabled:opacity-50"
            >
              <Unlink class="h-4 w-4" />
              Zuordnung aufheben
            </button>
            <button
              @click="requestTransactionAction('delete')"
              :disabled="isUnmatchingTransaction || isDeletingTransaction"
              class="inline-flex items-center gap-2 px-3 py-2 text-sm text-red-700 bg-red-50 hover:bg-red-100 rounded-lg transition-colors disabled:opacity-50"
            >
              <Trash2 class="h-4 w-4" />
              Transaktion löschen
            </button>
          </div>

          <div class="flex justify-end">
            <button
              @click="closeTransactionModal"
              class="px-4 py-2 bg-gray-100 text-gray-700 hover:bg-gray-200 rounded-lg transition-colors"
            >
              Schließen
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Allocation Modal -->
    <div
      v-if="showAllocationModal && allocationSuggestion"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="closeAllocationModal"
    >
      <div class="bg-white rounded-xl shadow-xl w-full max-w-2xl mx-4 p-6 max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between mb-6">
          <h2 class="text-xl font-semibold">Transaktion zuordnen</h2>
          <button @click="closeAllocationModal" class="p-1 hover:bg-gray-100 rounded">
            <X class="h-5 w-5" />
          </button>
        </div>

        <div class="p-4 bg-blue-50 rounded-lg mb-6">
          <div class="flex justify-between text-sm">
            <span class="text-gray-500">Zahler</span>
            <span class="font-medium">{{ allocationSuggestion.transaction.payerName || 'Unbekannt' }}</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-gray-500">Datum</span>
            <span class="font-medium">{{ formatDate(allocationSuggestion.transaction.bookingDate) }}</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-gray-500">Betrag</span>
            <span class="font-semibold text-blue-700">{{ formatCurrency(allocationSuggestion.transaction.amount) }}</span>
          </div>
          <div v-if="allocationSuggestion.transaction.description" class="text-xs text-gray-600 mt-2 break-words">
            {{ allocationSuggestion.transaction.description }}
          </div>
        </div>

        <div class="space-y-3">
          <h3 class="text-sm font-medium text-gray-600">Offene Beiträge</h3>
          <div v-if="allocationRows.length === 0" class="text-sm text-gray-500">
            Keine offenen Beiträge vorhanden.
          </div>
          <div v-else class="space-y-2">
            <div
              v-for="row in allocationRows"
              :key="row.fee.id"
              class="flex items-center justify-between gap-4 p-3 border rounded-lg"
            >
              <div>
                <p class="font-medium">{{ getFeeTypeName(row.fee.feeType) }}</p>
                <p class="text-xs text-gray-500">
                  {{ row.fee.month ? getMonthName(row.fee.month) + ' ' : '' }}{{ row.fee.year }}
                </p>
                <p class="text-xs text-gray-500">
                  Rest: {{ formatCurrency(getFeeRemainingAmount(row.fee)) }}
                </p>
              </div>
              <div class="w-40">
                <input
                  v-model.number="row.amount"
                  type="number"
                  min="0"
                  step="0.01"
                  :max="getFeeRemainingAmount(row.fee)"
                  class="w-full px-2 py-1 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
                  placeholder="0,00"
                />
                <div class="flex items-center justify-end gap-2 mt-2">
                  <button
                    type="button"
                    @click="assignRemainingToFee(row.fee.id)"
                    class="text-xs text-gray-500 hover:text-gray-700"
                  >
                    Restbetrag
                  </button>
                  <button
                    type="button"
                    @click="assignOnlyToFee(row.fee.id)"
                    class="text-xs text-primary hover:text-primary/80 font-medium"
                  >
                    Nur diesen Beitrag
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="mt-6 space-y-2">
          <div class="flex justify-between text-sm">
            <span class="text-gray-500">Zugeteilt</span>
            <span class="font-medium">{{ formatCurrency(allocationTotal) }}</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-gray-500">Restbetrag</span>
            <span :class="allocationRemaining < -0.01 ? 'text-red-600 font-medium' : 'font-medium'">
              {{ formatCurrency(allocationRemaining) }}
            </span>
          </div>
          <p v-if="allocationRemaining > 0.01" class="text-xs text-amber-700">
            Ein Restbetrag wird als Überzahlung markiert.
          </p>
          <p v-if="allocationError" class="text-sm text-red-600">
            {{ allocationError }}
          </p>
        </div>

        <div class="flex justify-end gap-3 mt-6">
          <button
            @click="closeAllocationModal"
            class="px-4 py-2 bg-gray-100 text-gray-700 hover:bg-gray-200 rounded-lg transition-colors"
          >
            Abbrechen
          </button>
          <button
            @click="confirmAllocation"
            :disabled="isAllocating || allocationTotal <= 0 || allocationRemaining < -0.01"
            class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isAllocating" class="h-4 w-4 animate-spin" />
            Zuordnen
          </button>
        </div>
      </div>
    </div>

    <!-- Parent Detail Modal -->
    <div
      v-if="showParentDetailModal && selectedParentForDetail"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="closeParentDetailModal"
    >
      <div class="bg-white rounded-xl shadow-xl w-full max-w-lg mx-4 p-6 max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between mb-6">
          <div class="flex items-center gap-3">
            <div class="w-12 h-12 rounded-full bg-primary/10 flex items-center justify-center">
              <User class="h-6 w-6 text-primary" />
            </div>
            <div>
              <h2 class="text-xl font-semibold">
                {{ selectedParentForDetail.firstName }} {{ selectedParentForDetail.lastName }}
              </h2>
              <p class="text-sm text-gray-500">Elternteil</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button
              v-if="!isEditingParent"
              @click="startEditingParent"
              class="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
              title="Bearbeiten"
            >
              <Edit class="h-5 w-5" />
            </button>
              <button @click="closeParentDetailModal" class="p-1 hover:bg-gray-100 rounded" aria-label="Schließen">
                <X class="h-5 w-5" />
              </button>
          </div>
        </div>

        <!-- View Mode -->
        <div v-if="!isEditingParent" class="space-y-4">
          <div v-if="selectedParentForDetail.birthDate">
            <p class="text-sm text-gray-500">Geburtsdatum</p>
            <p class="font-medium">{{ formatDate(selectedParentForDetail.birthDate) }}</p>
          </div>

          <div v-if="selectedParentForDetail.email">
            <p class="text-sm text-gray-500">E-Mail</p>
            <a :href="`mailto:${selectedParentForDetail.email}`" class="font-medium text-primary hover:underline">
              {{ selectedParentForDetail.email }}
            </a>
          </div>

          <div v-if="selectedParentForDetail.phone">
            <p class="text-sm text-gray-500">Telefon</p>
            <a :href="`tel:${selectedParentForDetail.phone}`" class="font-medium text-primary hover:underline">
              {{ selectedParentForDetail.phone }}
            </a>
          </div>

          <div v-if="selectedParentForDetail.street">
            <p class="text-sm text-gray-500">Adresse</p>
            <p class="font-medium">{{ selectedParentForDetail.street }} {{ selectedParentForDetail.streetNo }}</p>
            <p class="text-gray-600">{{ selectedParentForDetail.postalCode }} {{ selectedParentForDetail.city }}</p>
          </div>

          <div class="pt-4 border-t text-sm text-gray-500">
            <p>Erstellt: {{ formatDate(selectedParentForDetail.createdAt) }}</p>
            <p>Aktualisiert: {{ formatDate(selectedParentForDetail.updatedAt) }}</p>
          </div>
        </div>

        <!-- Edit Mode -->
        <form v-else @submit.prevent="saveParentEdit" class="space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="parent-edit-firstName" class="block text-sm font-medium text-gray-700 mb-1">Vorname</label>
              <input
                id="parent-edit-firstName"
                v-model="parentEditForm.firstName"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div>
              <label for="parent-edit-lastName" class="block text-sm font-medium text-gray-700 mb-1">Nachname</label>
              <input
                id="parent-edit-lastName"
                v-model="parentEditForm.lastName"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div>
            <label for="parent-edit-birthDate" class="block text-sm font-medium text-gray-700 mb-1">Geburtsdatum</label>
            <input
              id="parent-edit-birthDate"
              v-model="parentEditForm.birthDate"
              type="date"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>

          <div>
            <label for="parent-edit-email" class="block text-sm font-medium text-gray-700 mb-1">E-Mail</label>
            <input
              id="parent-edit-email"
              v-model="parentEditForm.email"
              type="email"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>

          <div>
            <label for="parent-edit-phone" class="block text-sm font-medium text-gray-700 mb-1">Telefon</label>
            <input
              id="parent-edit-phone"
              v-model="parentEditForm.phone"
              type="tel"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>

          <div class="grid grid-cols-4 gap-4">
            <div class="col-span-3">
              <label for="parent-edit-street" class="block text-sm font-medium text-gray-700 mb-1">Straße</label>
              <input
                id="parent-edit-street"
                v-model="parentEditForm.street"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div>
              <label for="parent-edit-streetNo" class="block text-sm font-medium text-gray-700 mb-1">Hausnr.</label>
              <input
                id="parent-edit-streetNo"
                v-model="parentEditForm.streetNo"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div class="grid grid-cols-3 gap-4">
            <div>
              <label for="parent-edit-postalCode" class="block text-sm font-medium text-gray-700 mb-1">PLZ</label>
              <input
                id="parent-edit-postalCode"
                v-model="parentEditForm.postalCode"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div class="col-span-2">
              <label for="parent-edit-city" class="block text-sm font-medium text-gray-700 mb-1">Ort</label>
              <input
                id="parent-edit-city"
                v-model="parentEditForm.city"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div v-if="parentDetailError" class="p-3 bg-red-50 border border-red-200 rounded-lg">
            <p class="text-sm text-red-600">{{ parentDetailError }}</p>
          </div>

          <div class="flex justify-end gap-3 pt-4">
            <button
              type="button"
              @click="cancelEditingParent"
              class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
            >
              Abbrechen
            </button>
            <button
              type="submit"
              :disabled="isSavingParent"
              class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Loader2 v-if="isSavingParent" class="h-4 w-4 animate-spin" />
              <Check v-else class="h-4 w-4" />
              Speichern
            </button>
          </div>
        </form>
      </div>
    </div>

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
              {{ reminderFee.month ? getMonthName(reminderFee.month) + ' ' : '' }}{{ reminderFee.year }}
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
