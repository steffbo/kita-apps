import { ref, computed, watch } from 'vue';
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

export function useImportTransactions() {
  const activeFilter = ref<StatusFilter>('offen');
  // Transactions state (unified list)
  const unmatchedTransactions = ref<BankTransaction[]>([]);
  const unmatchedTotal = ref(0);
  const matchedTransactions = ref<BankTransaction[]>([]);
  const matchedTotal = ref(0);
  const warnings = ref<TransactionWarning[]>([]);
  const warningsTotal = ref(0);
  const isLoadingTransactions = ref(true);

  // Search, sort and pagination (client-side over the loaded sets)
  const transactionSearch = ref('');
  const sortField = ref<SortField>('date');
  const sortDirection = ref<SortDirection>('desc');
  const page = ref(1);
  const pageSize = 50;

  function toggleSort(field: SortField): void {
    if (sortField.value === field) {
      sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc';
    } else {
      sortField.value = field;
      sortDirection.value = field === 'date' ? 'desc' : 'asc';
    }
  }

  watch([activeFilter, transactionSearch], () => {
    page.value = 1;
  });

  // Unified rows: merge matched, unmatched and open warnings by transaction id
  const transactionRows = computed<TxRow[]>(() => {
    const map = new Map<string, TxRow>();
    for (const tx of unmatchedTransactions.value) {
      map.set(tx.id, { key: tx.id, tx, matched: false, warnings: [] });
    }
    for (const tx of matchedTransactions.value) {
      const existing = map.get(tx.id);
      if (existing) {
        existing.matched = true;
      } else {
        map.set(tx.id, { key: tx.id, tx, matched: true, warnings: [] });
      }
    }
    for (const warning of warnings.value) {
      let row = warning.transactionId ? map.get(warning.transactionId) : undefined;
      if (!row && warning.transaction && !map.has(warning.transaction.id)) {
        const tx = warning.transaction;
        row = { key: tx.id || warning.id, tx, matched: false, warnings: [] };
        map.set(row.key, row);
      }
      if (row) {
        row.warnings.push(warning);
      }
    }
    return [...map.values()];
  });

  const offenCount = computed(() => transactionRows.value.filter(r => !r.matched).length);
  const warnungenCount = computed(() => transactionRows.value.filter(r => r.warnings.length > 0).length);
  const zugeordnetCount = computed(() => matchedTotal.value);

  const filteredRows = computed<TxRow[]>(() => {
    let rows = transactionRows.value;
    if (activeFilter.value === 'offen') {
      rows = rows.filter(r => !r.matched);
    } else if (activeFilter.value === 'warnungen') {
      rows = rows.filter(r => r.warnings.length > 0);
    } else if (activeFilter.value === 'zugeordnet') {
      rows = rows.filter(r => r.matched);
    }

    const search = transactionSearch.value.trim().toLowerCase();
    if (search) {
      rows = rows.filter(
        r =>
          (r.tx.payerName || '').toLowerCase().includes(search) ||
          (r.tx.description || '').toLowerCase().includes(search) ||
          (r.tx.payerIban || '').toLowerCase().includes(search)
      );
    }
    return rows;
  });

  const sortedRows = computed<TxRow[]>(() => {
    const dir = sortDirection.value === 'asc' ? 1 : -1;
    return [...filteredRows.value].sort((a, b) => {
      switch (sortField.value) {
        case 'payer':
          return dir * (a.tx.payerName || '').localeCompare(b.tx.payerName || '');
        case 'description':
          return dir * (a.tx.description || '').localeCompare(b.tx.description || '');
        case 'amount':
          return dir * (a.tx.amount - b.tx.amount);
        default:
          return dir * (new Date(a.tx.bookingDate).getTime() - new Date(b.tx.bookingDate).getTime());
      }
    });
  });

  const totalPages = computed(() => Math.max(1, Math.ceil(sortedRows.value.length / pageSize)));

  const visiblePages = computed<number[]>(() => {
    const total = totalPages.value;
    if (total <= 7) {
      return Array.from({ length: total }, (_, i) => i + 1);
    }
    const start = Math.max(1, Math.min(page.value - 3, total - 6));
    return Array.from({ length: 7 }, (_, i) => start + i);
  });

  const pagedRows = computed<TxRow[]>(() =>
    sortedRows.value.slice((page.value - 1) * pageSize, page.value * pageSize)
  );

  function goToPage(target: number): void {
    page.value = Math.min(Math.max(1, target), totalPages.value);
  }

  return {
    transactionRows,
    activeFilter,
    unmatchedTransactions,
    unmatchedTotal,
    matchedTransactions,
    matchedTotal,
    warnings,
    warningsTotal,
    isLoadingTransactions,
    transactionSearch,
    sortField,
    sortDirection,
    page,
    pageSize,
    toggleSort,
    offenCount,
    warnungenCount,
    zugeordnetCount,
    filteredRows,
    sortedRows,
    totalPages,
    visiblePages,
    pagedRows,
    goToPage,
  };
}
