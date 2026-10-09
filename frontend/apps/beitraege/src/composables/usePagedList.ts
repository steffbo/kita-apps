import { computed, ref, watch, type Ref } from 'vue';

interface PagedListOptions<T> {
  fetchPage: (pagination: { page: number; perPage: number }) => Promise<{ data: T[]; total: number }>;
  onLoaded?: (items: T[]) => void;
}

export function usePagedList<T extends { id: string }>(options: PagedListOptions<T>) {
  const items = ref<T[]>([]) as Ref<T[]>;
  const total = ref(0);
  const isLoading = ref(true);
  const error = ref<string | null>(null);
  const currentPage = ref(1);
  const pageSize = ref(25);
  const selectedIds = ref<Set<string>>(new Set());
  const totalPages = computed(() => Math.ceil(total.value / pageSize.value));
  const offset = computed(() => (currentPage.value - 1) * pageSize.value);

  let loadSeq = 0;
  async function load() {
    const seq = ++loadSeq;
    isLoading.value = true;
    error.value = null;
    try {
      const response = await options.fetchPage({ page: currentPage.value, perPage: pageSize.value });
      if (seq !== loadSeq) return; // a newer request superseded this one
      items.value = response.data;
      total.value = response.total;

      // If the current page ran empty (e.g. after deletes), fall back to the last valid page
      if (response.data.length === 0 && currentPage.value > 1 && response.total > 0) {
        currentPage.value = Math.max(1, Math.ceil(response.total / pageSize.value));
        return;
      }

      // Clear selection if items no longer exist
      const currentIds = new Set(response.data.map(item => item.id));
      selectedIds.value = new Set([...selectedIds.value].filter(id => currentIds.has(id)));
      options.onLoaded?.(response.data);
    } catch (e) {
      if (seq !== loadSeq) return;
      error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
    } finally {
      if (seq === loadSeq) isLoading.value = false;
    }
  }

  function goToFirstPageAndLoad() {
    if (currentPage.value !== 1) {
      currentPage.value = 1; // pagination watcher performs the reload
    } else {
      load();
    }
  }

  let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null;
  function handleSearchInput() {
    if (searchDebounceTimer) {
      clearTimeout(searchDebounceTimer);
    }
    searchDebounceTimer = setTimeout(() => goToFirstPageAndLoad(), 150);
  }

  // Filters/search use explicit handlers so each interaction triggers exactly one load
  watch([currentPage, pageSize], () => {
    load();
  });

  return {
    items,
    total,
    isLoading,
    error,
    currentPage,
    pageSize,
    selectedIds,
    totalPages,
    offset,
    load,
    goToFirstPageAndLoad,
    handleSearchInput,
  };
}
