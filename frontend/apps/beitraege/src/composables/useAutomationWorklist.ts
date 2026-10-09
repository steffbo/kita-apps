import { ref, computed, watch, onMounted } from 'vue';
import { api } from '@/api';
import type { ReminderCase, ReminderCaseFee } from '@/api/types';
import { useAuthStore } from '@/stores/auth';

export function useAutomationWorklist() {
  const authStore = useAuthStore();
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

  // The backend lists a Mahngebühr directly after its open base fee; that row
  // joins its base row (no divider, no repeated name).
  function isNestedReminder(fee: ReminderCaseFee): boolean {
    const baseId = fee.reminderForId;
    return !!baseId && !!selectedCase.value?.fees.some((other) => other.feeId === baseId);
  }

  function hasNestedReminderBelow(index: number): boolean {
    const fees = selectedCase.value?.fees ?? [];
    const next = fees[index + 1];
    return !!next && next.reminderForId === fees[index]?.feeId;
  }

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

  let casesRequestSeq = 0;

  async function loadCases(selectId: string | null = null): Promise<void> {
    if (!authStore.isAdmin) return;
    // Correlate responses with requests: a slow response for an outdated scope
    // (e.g. the initial load) must not overwrite a newer scope's result.
    const requestSeq = ++casesRequestSeq;
    isCasesLoading.value = true;
    casesError.value = null;
    try {
      const result = await api.getReminderCases({ scope: scope.value });
      if (requestSeq !== casesRequestSeq) return;
      cases.value = result.cases;
      if (selectId && cases.value.some((item) => item.householdId === selectId)) {
        selectedHouseholdId.value = selectId;
      } else if (
        selectedHouseholdId.value
        && !cases.value.some((item) => item.householdId === selectedHouseholdId.value)
      ) {
        selectedHouseholdId.value = null;
      }
    } catch (e) {
      if (requestSeq !== casesRequestSeq) return;
      casesError.value = e instanceof Error ? e.message : 'Familien konnten nicht geladen werden';
    } finally {
      if (requestSeq === casesRequestSeq) {
        isCasesLoading.value = false;
      }
    }
  }

  watch(scope, () => {
    void loadCases();
  });

  // ── Lifecycle ────────────────────────────────────────────────────────────────
  onMounted(() => {
    if (authStore.isAdmin) {
      loadCases();
    }
  });

  watch(
    () => authStore.isAdmin,
    (isAdmin) => {
      if (isAdmin) {
        loadCases();
      }
    }
  );
  return {
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
  };
}
