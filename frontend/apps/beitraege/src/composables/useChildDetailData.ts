import { ref, computed, onMounted, watch, type Ref } from 'vue';
import { api } from '@/api';
import type {
  CareHoursHistoryEntry,
  Child,
  ChildcareFeeResult,
  FeeExpectation,
  KnownIBANSummary,
  LegalHoursHistoryEntry,
  MatchSuggestion,
} from '@/api/types';
import { isUnderThree } from '@/utils/child';

export type EffectiveCareHours = {
  hours: number;
  effectiveFrom: string | null;
  isUpcoming: boolean;
};

export function useChildDetailData(childId: Readonly<Ref<string>>) {
  const child = ref<Child | null>(null);
  const fees = ref<FeeExpectation[]>([]);
  const isLoading = ref(true);
  const error = ref<string | null>(null);

  // Care hours history
  const careHoursHistory = ref<CareHoursHistoryEntry[]>([]);
  const legalHoursHistory = ref<LegalHoursHistoryEntry[]>([]);

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
    return nextUpcomingPeriod(
      careHoursHistory.value.filter((entry) => entry.careHours !== undefined && entry.careHours !== null),
    );
  });

  const upcomingLegalHours = computed(() => {
    if (child.value?.legalHours !== undefined && child.value?.legalHours !== null) return null;
    return nextUpcomingPeriod(
      legalHoursHistory.value.filter((entry) => entry.legalHours !== undefined && entry.legalHours !== null),
    );
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
      likelyTransactionsError.value = e instanceof Error
        ? e.message : 'Transaktionen konnten nicht geladen werden';
    } finally {
      isLoadingLikelyTransactions.value = false;
    }
  }

  onMounted(loadChild);
  watch(childId, () => {
    loadChild();
  });
  return {
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
    isLoadingTrustedIbans,
    likelyTransactions,
    isLoadingLikelyTransactions,
    likelyTransactionsError,
    likelyTransactionsScanned,
    upcomingCareHours,
    upcomingLegalHours,
    loadChild,
    loadChildcareFee,
    loadTrustedIbans,
    loadLikelyTransactions,
  };
}
