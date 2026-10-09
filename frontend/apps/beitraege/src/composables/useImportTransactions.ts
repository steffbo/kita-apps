import { ref, computed, watch, onScopeDispose, type Ref } from 'vue';
import { api } from '@/api';
import type { BankTransaction, TransactionWarning } from '@/api/types';

export type StatusFilter = 'offen' | 'warnungen' | 'zugeordnet' | 'alle';
export type SortField = 'date' | 'payer' | 'description' | 'amount';
type SortDirection = 'asc' | 'desc';

export interface TxRow {
  key: string;
  tx: BankTransaction;
  matched: boolean;
  warnings: TransactionWarning[];
}

export function isPartiallyAllocated(row: TxRow): boolean {
  return (row.tx.matchedAmount ?? 0) > 0.005 && row.tx.amount - (row.tx.matchedAmount ?? 0) > 0.005;
}

export function getTxRemaining(tx: BankTransaction): number {
  const remaining = tx.amount - (tx.matchedAmount ?? 0);
  return remaining > 0 ? remaining : 0;
}

export function useImportTransactions(uploadError: Ref<string | null>) {
  const activeFilter = ref<StatusFilter>('offen');
  const transactions = ref<BankTransaction[]>([]);
  const warnings = ref<TransactionWarning[]>([]);
  const warningsTotal = ref(0);
  const totalRows = ref(0);
  const allTotal = ref(0);
  const offenCount = ref(0);
  const warnungenCount = ref(0);
  const zugeordnetCount = ref(0);
  const isLoadingTransactions = ref(true);
  const transactionSearch = ref('');
  const search = ref('');
  const sortField = ref<SortField>('date');
  const sortDirection = ref<SortDirection>('desc');
  const page = ref(1);
  const pageSize = 50;
  let sequence = 0;
  let searchTimer: ReturnType<typeof setTimeout> | undefined;

  async function loadTransactions(): Promise<void> {
    const requestSequence = ++sequence;
    isLoadingTransactions.value = true;
    try {
      const [result, open, matched, warned, all] = await Promise.all([
        api.getUnmatchedTransactions({
          status: activeFilter.value, search: search.value, sortBy: sortField.value,
          sortDir: sortDirection.value, page: page.value, perPage: pageSize,
        }),
        api.getUnmatchedTransactions({ status: 'offen', perPage: 1 }),
        api.getUnmatchedTransactions({ status: 'zugeordnet', perPage: 1 }),
        api.getUnmatchedTransactions({ status: 'warnungen', perPage: 1 }),
        api.getUnmatchedTransactions({ status: 'alle', perPage: 1 }),
      ]);
      if (requestSequence !== sequence) return;
      const lastPage = Math.max(1, result.totalPages);
      if (page.value > lastPage) {
        page.value = lastPage;
        return;
      }
      const pageWarnings: TransactionWarning[] = [];
      let warningCount = 0;
      if (result.data.length) {
        const params = { transactionIds: result.data.map(tx => tx.id) };
        let warningPage = 1;
        let warningPages = 1;
        do {
          const warningResult = await api.getWarnings(warningPage, 100, params);
          if (requestSequence !== sequence) return;
          pageWarnings.push(...warningResult.data);
          warningCount = warningResult.total;
          warningPages = warningResult.totalPages;
          warningPage++;
        } while (warningPage <= warningPages);
      }
      if (requestSequence !== sequence) return;
      transactions.value = result.data;
      warnings.value = pageWarnings;
      warningsTotal.value = warningCount;
      totalRows.value = result.total;
      allTotal.value = all.total;
      offenCount.value = open.total;
      warnungenCount.value = warned.total;
      zugeordnetCount.value = matched.total;
    } catch (error) {
      if (requestSequence !== sequence) return;
      console.error('Failed to load transactions:', error);
      uploadError.value = error instanceof Error ? error.message : 'Transaktionen konnten nicht geladen werden';
    } finally {
      if (requestSequence === sequence) isLoadingTransactions.value = false;
    }
  }

  function resetPageAndLoad(): void {
    sequence++;
    if (page.value !== 1) page.value = 1;
    else void loadTransactions();
  }

  watch(transactionSearch, value => {
    sequence++;
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => {
      search.value = value.trim();
      resetPageAndLoad();
    }, 300);
  }, { flush: 'sync' });
  watch([activeFilter, sortField, sortDirection, page], () => {
    sequence++;
  }, { flush: 'sync' });
  watch([activeFilter, sortField, sortDirection], resetPageAndLoad);
  watch(page, () => { void loadTransactions(); });
  onScopeDispose(() => {
    sequence++;
    clearTimeout(searchTimer);
  });

  function toggleSort(field: SortField): void {
    if (sortField.value === field) {
      sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc';
    } else {
      sortField.value = field;
      sortDirection.value = field === 'date' ? 'desc' : 'asc';
    }
  }

  const pagedRows = computed<TxRow[]>(() => transactions.value.map(tx => ({
    key: tx.id,
    tx,
    matched: (tx.matches?.length ?? 0) > 0,
    warnings: warnings.value.filter(warning => warning.transactionId === tx.id),
  })));

  const totalPages = computed(() => Math.max(1, Math.ceil(totalRows.value / pageSize)));

  const visiblePages = computed<number[]>(() => {
    const total = totalPages.value;
    if (total <= 7) {
      return Array.from({ length: total }, (_, i) => i + 1);
    }
    const start = Math.max(1, Math.min(page.value - 3, total - 6));
    return Array.from({ length: 7 }, (_, i) => start + i);
  });

  function goToPage(target: number): void {
    page.value = Math.min(Math.max(1, target), totalPages.value);
  }

  return {
    activeFilter, warnings, warningsTotal, isLoadingTransactions,
    transactionSearch, sortField, sortDirection, page, toggleSort,
    offenCount, warnungenCount, zugeordnetCount, totalRows, allTotal,
    totalPages, visiblePages, pagedRows, goToPage, loadTransactions,
  };
}
