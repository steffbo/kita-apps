<script setup lang="ts">
import { computed } from 'vue';
import { ChevronLeft, ChevronRight } from 'lucide-vue-next';

// Footer of a paged list: range, page size and page buttons. The page numbers hide on phones.
const props = withDefaults(defineProps<{
  currentPage: number;
  totalPages: number;
  total: number;
  offset: number;
  pageSizeOptions?: number[];
}>(), { pageSizeOptions: () => [10, 25, 50, 100] });
const pageSize = defineModel<number>('pageSize', { required: true });
const emit = defineEmits<{ go: [page: number] }>();

const pages = computed(() => {
  const result: (number | '...')[] = [];
  const total = props.totalPages;
  const current = props.currentPage;
  if (total <= 7) {
    for (let i = 1; i <= total; i++) result.push(i);
    return result;
  }
  result.push(1);
  if (current > 3) result.push('...');
  for (let i = Math.max(2, current - 1); i <= Math.min(total - 1, current + 1); i++) result.push(i);
  if (current < total - 2) result.push('...');
  result.push(total);
  return result;
});
</script>

<template>
  <div class="flex items-center justify-between gap-3 border-t px-4 py-2.5 text-sm">
    <div class="flex items-center gap-3">
      <span class="tabular-nums text-muted-foreground">
        {{ total ? offset + 1 : 0 }}–{{ Math.min(offset + pageSize, total) }} von {{ total }}
      </span>
      <select v-model="pageSize" aria-label="Einträge pro Seite"
        class="hidden h-8 rounded-lg border border-input bg-card px-2 text-sm sm:block">
        <option v-for="size in pageSizeOptions" :key="size" :value="size">{{ size }} pro Seite</option>
      </select>
    </div>
    <div class="flex items-center gap-1">
      <button type="button" class="btn btn-ghost btn-icon btn-sm" aria-label="Vorherige Seite"
        :disabled="currentPage <= 1" @click="emit('go', currentPage - 1)"><ChevronLeft /></button>
      <template v-for="(page, index) in pages" :key="`${page}-${index}`">
        <span v-if="page === '...'" class="hidden px-1 text-muted-foreground sm:inline">…</span>
        <button v-else type="button" class="btn btn-sm hidden min-w-8 px-2 tabular-nums sm:inline-flex"
          :class="page === currentPage ? 'bg-primary-soft text-primary-text' : 'btn-ghost'"
          :aria-current="page === currentPage ? 'page' : undefined" @click="emit('go', page)">{{ page }}</button>
      </template>
      <span class="px-1 tabular-nums text-muted-foreground sm:hidden">{{ currentPage }} / {{ totalPages || 1 }}</span>
      <button type="button" class="btn btn-ghost btn-icon btn-sm" aria-label="Nächste Seite"
        :disabled="currentPage >= totalPages" @click="emit('go', currentPage + 1)"><ChevronRight /></button>
    </div>
  </div>
</template>
