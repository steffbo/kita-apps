import { ref, computed, watch, onUnmounted, type Ref } from 'vue';
import { api } from '@/api';
import type { ReminderCase, ReminderCasePreview, ReminderCaseSendResult } from '@/api/types';
import { ReminderCaseConflictError } from '@/api/types';
import { todayISO } from '@/utils/format';

export function useAutomationSending(
  selectedCase: Readonly<Ref<ReminderCase | null>>,
  selectedHouseholdId: Ref<string | null>,
  cases: Ref<ReminderCase[]>,
  loadCases: (selectId?: string | null) => Promise<void>,
  openCase: (householdId: string) => void,
) {
  // ── Case detail state ────────────────────────────────────────────────────────
  const selectedFeeIds = ref<string[]>([]);
  const reminderFeeIds = ref<string[]>([]);
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

  let previewRequestedAt: string | null = null;
  let previewTimer: ReturnType<typeof setTimeout> | null = null;
  let previewRequestSeq = 0;

  // A mail that charges at least one Mahngebühr is a Mahnung, otherwise a
  // Zahlungserinnerung.
  const isMahnung = computed(() => reminderFeeIds.value.length > 0);

  // Fees whose Mahngebühr is due by the rules but not ticked.
  const untickedDueFeeCount = computed(() => {
    if (!selectedCase.value) return 0;
    return selectedCase.value.fees.filter(
      (fee) => fee.reminderFeeDue
        && selectedFeeIds.value.includes(fee.feeId)
        && !reminderFeeIds.value.includes(fee.feeId),
    ).length;
  });

  function toggleFee(feeId: string): void {
    const index = selectedFeeIds.value.indexOf(feeId);
    if (index >= 0) {
      selectedFeeIds.value.splice(index, 1);
      const reminderIndex = reminderFeeIds.value.indexOf(feeId);
      if (reminderIndex >= 0) reminderFeeIds.value.splice(reminderIndex, 1);
    } else {
      selectedFeeIds.value.push(feeId);
    }
  }

  function toggleReminderFee(feeId: string): void {
    const index = reminderFeeIds.value.indexOf(feeId);
    if (index >= 0) {
      reminderFeeIds.value.splice(index, 1);
    } else {
      reminderFeeIds.value.push(feeId);
    }
  }

  async function refreshPreview(): Promise<void> {
    const item = selectedCase.value;
    if (!item || selectedFeeIds.value.length === 0) {
      preview.value = null;
      return;
    }
    if (previewTimer) clearTimeout(previewTimer);
    // Correlate responses with requests: only the latest request may update the
    // preview, otherwise a slow older response could overwrite a newer one with
    // wrong amounts or texts.
    const requestSeq = ++previewRequestSeq;
    isPreviewLoading.value = true;
    previewError.value = null;
    previewRequestedAt = new Date().toISOString();
    try {
      const result = await api.previewReminderCase(item.householdId, {
        runDate: todayISO(),
        feeIds: selectedFeeIds.value,
        reminderFeeIds: reminderFeeIds.value,
        includeQR: includeQR.value,
      });
      if (requestSeq !== previewRequestSeq) return;
      const hadEdits = userEdited.value;
      preview.value = result;
      subjectEdit.value = result.subject;
      bodyEdit.value = result.body;
      if (hadEdits) {
        resetNotice.value = true;
      }
      userEdited.value = false;
    } catch (e) {
      if (requestSeq !== previewRequestSeq) return;
      previewError.value = e instanceof Error ? e.message : 'Vorschau konnte nicht geladen werden';
    } finally {
      if (requestSeq === previewRequestSeq) {
        isPreviewLoading.value = false;
      }
    }
  }

  function invalidatePreview(): void {
    if (previewTimer) clearTimeout(previewTimer);
    // Invalidate synchronously on every input change: a response that is still
    // in flight during the debounce window carries the current sequence number
    // and would otherwise overwrite the UI with the previous selection's
    // amounts/text.
    previewRequestSeq++;
    preview.value = null;
    isPreviewLoading.value = false;
    previewError.value = null;
  }

  function schedulePreviewRefresh(): void {
    if (previewTimer) clearTimeout(previewTimer);
    previewTimer = setTimeout(() => {
      void refreshPreview();
    }, 250);
  }

  watch(selectedFeeIds, () => {
    resetNotice.value = false;
    invalidatePreview();
    schedulePreviewRefresh();
  }, { deep: true });

  watch(reminderFeeIds, () => {
    resetNotice.value = false;
    invalidatePreview();
    schedulePreviewRefresh();
  }, { deep: true });

  watch(includeQR, () => {
    invalidatePreview();
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
    // Do not confirm against a preview that is still loading: the shown text
    // may not match the current selection yet.
    if (!preview.value || isPreviewLoading.value) return;
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
        runDate: todayISO(),
        feeIds: selectedFeeIds.value,
        reminderFeeIds: reminderFeeIds.value,
        includeQR: includeQR.value,
        ...(subjectEdit.value.trim() !== '' && subjectEdit.value !== preview.value.subject
          ? { subject: subjectEdit.value } : {}),
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

  onUnmounted(() => {
    if (previewTimer) clearTimeout(previewTimer);
  });
  return {
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
    refreshPreview,
    invalidatePreview,
    onSubjectInput,
    onBodyInput,
    resetEdits,
    openSendConfirmation,
    confirmSend,
  };
}
