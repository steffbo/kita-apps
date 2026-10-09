import { ref, type Ref } from 'vue';
import { api } from '@/api';
import type { EmailLog, ReminderCase } from '@/api/types';

export function useAutomationChronology(
  selectedCase: Readonly<Ref<ReminderCase | null>>,
  selectedHouseholdId: Ref<string | null>,
) {
  // ── Family chronology ────────────────────────────────────────────────────────
  const chronology = ref<EmailLog[]>([]);
  const isChronologyLoading = ref(false);
  const selectedChronologyLog = ref<EmailLog | null>(null);
  let chronologyRequestSeq = 0;

  async function loadChronology(): Promise<void> {
    const item = selectedCase.value;
    if (!item) return;
    // Correlate responses with the selected household: a slow response for a
    // previously selected family must never render under another family.
    const requestSeq = ++chronologyRequestSeq;
    const householdId = item.householdId;
    isChronologyLoading.value = true;
    try {
      const result = await api.getEmailLogs({ householdId, perPage: 20, sortDir: 'desc' });
      if (requestSeq !== chronologyRequestSeq || selectedHouseholdId.value !== householdId) return;
      chronology.value = result.data;
    } catch {
      if (requestSeq !== chronologyRequestSeq) return;
      chronology.value = [];
    } finally {
      if (requestSeq === chronologyRequestSeq) {
        isChronologyLoading.value = false;
      }
    }
  }

  return {
    chronology,
    isChronologyLoading,
    selectedChronologyLog,
    loadChronology,
  };
}
