import { ref, type Ref } from 'vue';
import { api } from '@/api';
import type { TransactionWarning } from '@/api/types';

export function useImportWarningActions(
  warnings: Ref<TransactionWarning[]>,
  warningsTotal: Ref<number>,
  uploadError: Ref<string | null>,
) {
  // Warning actions state
  const isResolvingWarning = ref<string | null>(null);
  const dismissWarningId = ref<string | null>(null);
  const dismissNote = ref('');

  function showWarningDismiss(warningId: string): void {
    dismissWarningId.value = warningId;
    dismissNote.value = '';
  }

  function cancelWarningDismiss(): void {
    dismissWarningId.value = null;
    dismissNote.value = '';
  }

  async function dismissWarning(warning: TransactionWarning): Promise<void> {
    isResolvingWarning.value = warning.id;
    try {
      await api.dismissWarning(warning.id, dismissNote.value);
      warnings.value = warnings.value.filter(w => w.id !== warning.id);
      warningsTotal.value = Math.max(0, warningsTotal.value - 1);
      dismissWarningId.value = null;
      dismissNote.value = '';
    } catch (error) {
      console.error('Failed to dismiss warning:', error);
      uploadError.value = error instanceof Error ? error.message : 'Warnung konnte nicht verworfen werden';
    } finally {
      isResolvingWarning.value = null;
    }
  }

  async function resolveLateFee(warning: TransactionWarning): Promise<void> {
    isResolvingWarning.value = warning.id;
    try {
      await api.resolveLateFee(warning.id);
      warnings.value = warnings.value.filter(w => w.id !== warning.id);
      warningsTotal.value = Math.max(0, warningsTotal.value - 1);
    } catch (error) {
      console.error('Failed to resolve late fee:', error);
      uploadError.value = error instanceof Error ? error.message : 'Mahngebuhr konnte nicht erstellt werden';
    } finally {
      isResolvingWarning.value = null;
    }
  }

  return {
    isResolvingWarning,
    dismissWarningId,
    dismissNote,
    showWarningDismiss,
    cancelWarningDismiss,
    dismissWarning,
    resolveLateFee,
  };
}
