<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { api } from '@/api';
import type { Child, FeeExpectation, UpdateChildRequest, Parent, CreateParentRequest, UpdateParentRequest, BankTransaction, IncomeStatus, UpdateHouseholdRequest, ChildcareFeeResult, ChildLedger, FeeCoverage, CoverageStatus } from '@/api/types';
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
  BookOpen,
  History,
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
const selectedTransaction = ref<BankTransaction | null>(null);
const showTransactionModal = ref(false);

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

// Ledger state
const showLedger = ref(false);
const ledger = ref<ChildLedger | null>(null);
const isLoadingLedger = ref(false);
const ledgerYear = ref<number | undefined>(undefined);

// Timeline state
const showTimeline = ref(false);
const timeline = ref<FeeCoverage[] | null>(null);
const isLoadingTimeline = ref(false);
const timelineYear = ref<number>(new Date().getFullYear());

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
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoading.value = false;
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

async function loadLedger() {
  isLoadingLedger.value = true;
  try {
    ledger.value = await api.getChildLedger(childId.value, ledgerYear.value);
  } catch (e) {
    console.error('Failed to load ledger:', e);
    ledger.value = null;
  } finally {
    isLoadingLedger.value = false;
  }
}

function toggleLedger() {
  showLedger.value = !showLedger.value;
  showTimeline.value = false;
  if (showLedger.value && !ledger.value) {
    loadLedger();
  }
}

async function loadTimeline() {
  isLoadingTimeline.value = true;
  try {
    const data = await api.getChildTimeline(childId.value, timelineYear.value);
    console.log('Timeline loaded:', data);
    timeline.value = data;
  } catch (e) {
    console.error('Failed to load timeline:', e);
    timeline.value = null;
  } finally {
    isLoadingTimeline.value = false;
  }
}

function toggleTimeline() {
  showTimeline.value = !showTimeline.value;
  showLedger.value = false;
  if (showTimeline.value && !timeline.value) {
    loadTimeline();
  }
}

// Watch for year filter changes
watch(ledgerYear, () => {
  if (showLedger.value) {
    loadLedger();
  }
});

watch(timelineYear, () => {
  if (showTimeline.value) {
    loadTimeline();
  }
});

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
    const updated = await api.updateChild(childId.value, editForm.value);
    child.value = updated;
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

// Available years for ledger filter (current year and previous 3 years)
const availableYears = computed(() => {
  const currentYear = new Date().getFullYear();
  return [currentYear, currentYear - 1, currentYear - 2, currentYear - 3];
});

function openTransactionModal(fee: FeeExpectation) {
  if (fee.matchedBy?.transaction) {
    selectedTransaction.value = fee.matchedBy.transaction;
    showTransactionModal.value = true;
  }
}

function closeTransactionModal() {
  showTransactionModal.value = false;
  selectedTransaction.value = null;
}

function formatTransactionDate(fee: FeeExpectation): string {
  // Use the transaction's booking date if available, otherwise fall back to paidAt
  if (fee.matchedBy?.transaction?.bookingDate) {
    return formatDate(fee.matchedBy.transaction.bookingDate);
  }
  if (fee.paidAt) {
    return formatDate(fee.paidAt);
  }
  return '';
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

function getTimelineStatusColor(status: CoverageStatus): string {
  switch (status) {
    case 'UNPAID':
      return 'bg-red-50 border-red-200 text-red-700';
    case 'PARTIAL':
      return 'bg-amber-50 border-amber-200 text-amber-700';
    case 'COVERED':
      return 'bg-green-50 border-green-200 text-green-700';
    case 'OVERPAID':
      return 'bg-blue-50 border-blue-200 text-blue-700';
    default:
      return 'bg-gray-50 border-gray-200 text-gray-700';
  }
}

function getTimelineStatusLabel(status: CoverageStatus): string {
  switch (status) {
    case 'UNPAID':
      return 'Unbezahlt';
    case 'PARTIAL':
      return 'Teilbezahlt';
    case 'COVERED':
      return 'Vollständig bezahlt';
    case 'OVERPAID':
      return 'Überbezahlt';
    default:
      return status;
  }
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
  return child.value?.household?.parents || [];
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
          <div class="flex items-center gap-2">
            <button
              @click="toggleTimeline"
              :class="[
                'inline-flex items-center gap-2 px-3 py-1.5 text-sm font-medium rounded-lg transition-colors',
                showTimeline
                  ? 'bg-primary text-white'
                  : 'text-primary border border-primary hover:bg-primary/10'
              ]"
            >
              <History class="h-4 w-4" />
              {{ showTimeline ? 'Übersicht' : 'Zeitstrahl' }}
            </button>
            <button
              @click="toggleLedger"
              :class="[
                'inline-flex items-center gap-2 px-3 py-1.5 text-sm font-medium rounded-lg transition-colors',
                showLedger
                  ? 'bg-primary text-white'
                  : 'text-primary border border-primary hover:bg-primary/10'
              ]"
            >
              <BookOpen class="h-4 w-4" />
              {{ showLedger ? 'Übersicht' : 'Kontobuch' }}
            </button>
          </div>
        </div>

        <!-- Timeline View -->
        <div v-if="showTimeline">
          <!-- Year filter -->
          <div class="flex items-center justify-between mb-4">
            <div class="flex items-center gap-2">
              <label class="text-sm text-gray-600">Jahr:</label>
              <select
                v-model="timelineYear"
                class="px-2 py-1 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              >
                <option v-for="year in availableYears" :key="year" :value="year">{{ year }}</option>
              </select>
            </div>
            <button
              @click="loadTimeline"
              class="p-1.5 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
              title="Aktualisieren"
            >
              <Loader2 v-if="isLoadingTimeline" class="h-4 w-4 animate-spin" />
              <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"/><path d="M16 16h5v5"/></svg>
            </button>
          </div>

          <!-- Timeline Grid -->
          <div v-if="isLoadingTimeline" class="flex items-center justify-center py-8">
            <Loader2 class="h-8 w-8 animate-spin text-primary" />
          </div>
          <div v-else-if="timeline?.length" class="space-y-3">
            <div
              v-for="month in timeline"
              :key="`${month.year}-${month.month}`"
              :class="['rounded-lg border p-4', getTimelineStatusColor(month.status)]"
            >
              <div class="flex items-center justify-between mb-3">
                <div class="flex items-center gap-3">
                  <h3 class="font-semibold">{{ getMonthName(month.month) }} {{ month.year }}</h3>
                  <span
                    :class="[
                      'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium',
                      month.status === 'UNPAID' ? 'bg-red-100 text-red-700' :
                      month.status === 'PARTIAL' ? 'bg-amber-100 text-amber-700' :
                      month.status === 'COVERED' ? 'bg-green-100 text-green-700' :
                      'bg-blue-100 text-blue-700'
                    ]"
                  >
                    <AlertTriangle v-if="month.status === 'UNPAID'" class="h-3 w-3" />
                    <Clock v-else-if="month.status === 'PARTIAL'" class="h-3 w-3" />
                    <CheckCircle v-else-if="month.status === 'COVERED'" class="h-3 w-3" />
                    <Check v-else class="h-3 w-3" />
                    {{ getTimelineStatusLabel(month.status) }}
                  </span>
                </div>
                <div class="text-right">
                  <p class="text-sm">
                    <span class="text-gray-500">Soll:</span> {{ formatCurrency(month.expectedTotal) }}
                  </p>
                  <p class="text-sm">
                    <span class="text-gray-500">Ist:</span> {{ formatCurrency(month.receivedTotal) }}
                  </p>
                  <p class="text-sm font-medium" :class="month.balance !== 0 ? 'text-red-600' : 'text-green-600'">
                    <span class="text-gray-500">Saldo:</span> {{ formatCurrency(month.balance) }}
                  </p>
                </div>
              </div>

              <!-- Transactions -->
              <div v-if="month.transactions?.length > 0" class="mt-3 pt-3 border-t border-current/20">
                <p class="text-xs font-medium mb-2">Zahlungen:</p>
                <div class="space-y-2">
                  <div
                    v-for="tx in month.transactions"
                    :key="tx.transactionId"
                    class="flex items-center justify-between text-sm"
                  >
                    <div class="flex items-center gap-2">
                      <span :class="tx.isForThisMonth ? 'text-gray-600' : 'text-blue-600'">
                        {{ formatDate(tx.bookingDate) }}
                      </span>
                      <span v-if="!tx.isForThisMonth" class="inline-flex items-center gap-1 px-1.5 py-0.5 bg-blue-100 text-blue-700 rounded text-xs">
                        Fremdmonat
                      </span>
                    </div>
                    <div class="text-right">
                      <span class="font-medium">{{ formatCurrency(tx.amount) }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div v-else class="text-center py-8 text-gray-500">
            Keine Daten für {{ timelineYear }}
          </div>
        </div>

        <!-- Ledger View -->
        <div v-if="showLedger">
          <!-- Year filter -->
          <div class="flex items-center justify-between mb-4">
            <div class="flex items-center gap-2">
              <label class="text-sm text-gray-600">Jahr:</label>
              <select
                v-model="ledgerYear"
                class="px-2 py-1 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              >
                <option :value="undefined">Alle</option>
                <option v-for="year in availableYears" :key="year" :value="year">{{ year }}</option>
              </select>
            </div>
            <button
              @click="loadLedger"
              class="p-1.5 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
              title="Aktualisieren"
            >
              <Loader2 v-if="isLoadingLedger" class="h-4 w-4 animate-spin" />
              <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"/><path d="M16 16h5v5"/></svg>
            </button>
          </div>

          <!-- Summary -->
          <div v-if="ledger?.summary" class="grid grid-cols-3 gap-4 mb-4">
            <div class="bg-gray-50 rounded-lg p-3 text-center">
              <p class="text-xs text-gray-500 mb-1">Gesamt Beiträge</p>
              <p class="font-semibold">{{ formatCurrency(ledger.summary.totalFees) }}</p>
              <p class="text-xs text-gray-400">{{ ledger.summary.totalFeesCount }} Posten</p>
            </div>
            <div class="bg-green-50 rounded-lg p-3 text-center">
              <p class="text-xs text-gray-500 mb-1">Bezahlt</p>
              <p class="font-semibold text-green-700">{{ formatCurrency(ledger.summary.totalPaid) }}</p>
              <p class="text-xs text-gray-400">{{ ledger.summary.paidFeesCount }} Posten</p>
            </div>
            <div :class="['rounded-lg p-3 text-center', ledger.summary.totalOpen > 0 ? 'bg-amber-50' : 'bg-gray-50']">
              <p class="text-xs text-gray-500 mb-1">Offen</p>
              <p :class="['font-semibold', ledger.summary.totalOpen > 0 ? 'text-amber-700' : '']">{{ formatCurrency(ledger.summary.totalOpen) }}</p>
              <p class="text-xs text-gray-400">{{ ledger.summary.openFeesCount }} Posten</p>
            </div>
          </div>

          <!-- Ledger table -->
          <div v-if="isLoadingLedger" class="flex items-center justify-center py-8">
            <Loader2 class="h-8 w-8 animate-spin text-primary" />
          </div>
          <div v-else-if="ledger?.entries?.length" class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead>
                <tr class="border-b text-left">
                  <th class="pb-2 font-medium text-gray-500">Datum</th>
                  <th class="pb-2 font-medium text-gray-500">Beschreibung</th>
                  <th class="pb-2 font-medium text-gray-500 text-right">Soll</th>
                  <th class="pb-2 font-medium text-gray-500 text-right">Haben</th>
                  <th class="pb-2 font-medium text-gray-500 text-right">Saldo</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="entry in ledger.entries"
                  :key="entry.id"
                  :class="[
                    'border-b border-gray-100 last:border-0',
                    entry.type === 'payment' ? 'bg-green-50/50' : ''
                  ]"
                >
                  <td class="py-2">{{ formatDate(entry.date) }}</td>
                  <td class="py-2">
                    <span v-if="entry.type === 'fee'">{{ entry.description }}</span>
                    <span v-else class="text-green-700">{{ entry.description }}</span>
                  </td>
                  <td class="py-2 text-right">
                    <span v-if="entry.debit > 0">{{ formatCurrency(entry.debit) }}</span>
                  </td>
                  <td class="py-2 text-right">
                    <span v-if="entry.credit > 0" class="text-green-700">{{ formatCurrency(entry.credit) }}</span>
                  </td>
                  <td class="py-2 text-right font-medium">
                    <span :class="entry.balance > 0 ? 'text-amber-700' : entry.balance < 0 ? 'text-green-700' : ''">
                      {{ formatCurrency(entry.balance) }}
                    </span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-else class="text-center py-8 text-gray-500">
            Keine Einträge im Kontobuch
          </div>
        </div>

        <!-- Standard View (Open/Paid fees) -->
        <div v-else-if="!showTimeline">
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
                  fee.matchedBy?.transaction ? 'hover:bg-green-100 cursor-pointer' : ''
                ]"
                :disabled="!fee.matchedBy?.transaction"
              >
                <div class="flex items-center gap-3">
                  <CheckCircle class="h-5 w-5 text-green-500" />
                  <div>
                    <p class="font-medium">{{ getFeeTypeName(fee.feeType) }}</p>
                    <p class="text-sm text-gray-600">
                      {{ fee.month ? getMonthName(fee.month) + ' ' : '' }}{{ fee.year }}
                      <span v-if="formatTransactionDate(fee)" class="text-green-600">
                        · Bezahlt am {{ formatTransactionDate(fee) }}
                      </span>
                    </p>
                  </div>
                </div>
                <div class="flex items-center gap-2">
                  <p class="font-semibold text-green-700">{{ formatCurrency(fee.amount) }}</p>
                  <CreditCard v-if="fee.matchedBy?.transaction" class="h-4 w-4 text-green-500" />
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
      v-if="showTransactionModal && selectedTransaction"
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

        <div class="space-y-4">
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

        <div class="flex justify-end mt-6">
          <button
            @click="closeTransactionModal"
            class="px-4 py-2 bg-gray-100 text-gray-700 hover:bg-gray-200 rounded-lg transition-colors"
          >
            Schließen
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
            <button @click="closeParentDetailModal" class="p-1 hover:bg-gray-100 rounded">
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
