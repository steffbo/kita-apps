<script setup lang="ts">
import { computed } from 'vue';
import { ArrowDown, ArrowUp, ArrowUpDown } from 'lucide-vue-next';

const props = defineProps<{
  label: string;
  column: string;
  sortKey: string;
  sortDir: 'asc' | 'desc';
  align?: 'left' | 'right';
}>();
const emit = defineEmits<{ sort: [column: string] }>();
const active = computed(() => props.sortKey === props.column);
const ariaSort = computed(() => (active.value ? (props.sortDir === 'asc' ? 'ascending' : 'descending') : 'none'));
</script>

<template>
  <th class="px-4 py-3" :aria-sort="ariaSort" :class="align === 'right' ? 'text-right' : 'text-left'">
    <button type="button" class="inline-flex items-center gap-1 font-medium hover:text-foreground"
      :class="{ 'text-foreground': active }" @click="emit('sort', column)">
      {{ label }}
      <ArrowUp v-if="active && sortDir === 'asc'" class="h-3.5 w-3.5" aria-hidden="true" />
      <ArrowDown v-else-if="active" class="h-3.5 w-3.5" aria-hidden="true" />
      <ArrowUpDown v-else class="h-3.5 w-3.5 opacity-40" aria-hidden="true" />
    </button>
  </th>
</template>
