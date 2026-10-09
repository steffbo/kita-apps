import { computed, ref, type Ref } from 'vue';

type SortValue = string | number | null | undefined;
type SortDir = 'asc' | 'desc';

export function useTableSort<T>(
  rows: Ref<T[]>,
  accessors: Record<string, (row: T) => SortValue>,
  initial: { key: string; dir?: SortDir },
) {
  const sortKey = ref(initial.key);
  const sortDir = ref<SortDir>(initial.dir ?? 'asc');
  const collator = new Intl.Collator('de', { numeric: true, sensitivity: 'base' });

  function toggle(key: string) {
    if (sortKey.value === key) sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc';
    else { sortKey.value = key; sortDir.value = 'asc'; }
  }

  const sorted = computed(() => {
    const get = accessors[sortKey.value];
    if (!get) return rows.value;
    const factor = sortDir.value === 'asc' ? 1 : -1;
    return [...rows.value].sort((a, b) => {
      const x = get(a), y = get(b);
      const xEmpty = x === null || x === undefined || x === '';
      const yEmpty = y === null || y === undefined || y === '';
      if (xEmpty || yEmpty) return xEmpty === yEmpty ? 0 : xEmpty ? 1 : -1;
      if (typeof x === 'number' && typeof y === 'number') return (x - y) * factor;
      return collator.compare(String(x), String(y)) * factor;
    });
  });

  return { sortKey, sortDir, sorted, toggle };
}
