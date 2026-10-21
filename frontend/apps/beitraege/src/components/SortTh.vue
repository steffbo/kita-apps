<script setup lang="ts" generic="K extends string">
import { computed, type Component } from 'vue';
import { ArrowDown, ArrowUp, ArrowUpDown } from 'lucide-vue-next';

// Sortierbarer Tabellenkopf für client- und serverseitig sortierte Tabellen. `icon` steht vor
// der Beschriftung, `cellClass` ersetzt das Standard-Padding (kompakte Tabellen).
const props = withDefaults(defineProps<{
  label: string;
  column: K;
  sortKey: K;
  sortDir: 'asc' | 'desc';
  align?: 'left' | 'right';
  icon?: Component;
  cellClass?: string;
}>(), { align: 'left', icon: undefined, cellClass: 'px-4 py-3' });
const emit = defineEmits<{ sort: [column: K] }>();
const active = computed(() => props.sortKey === props.column);
const ariaSort = computed(() => (active.value ? (props.sortDir === 'asc' ? 'ascending' : 'descending') : 'none'));
</script>

<template>
  <th :class="[cellClass, align === 'right' ? 'text-right' : 'text-left']" :aria-sort="ariaSort">
    <button type="button" class="inline-flex items-center gap-1 font-medium hover:text-foreground"
      :class="{ 'text-foreground': active }" @click="emit('sort', column)">
      <component :is="icon" v-if="icon" class="h-4 w-4" aria-hidden="true" />
      {{ label }}
      <ArrowUp v-if="active && sortDir === 'asc'" class="h-3.5 w-3.5" aria-hidden="true" />
      <ArrowDown v-else-if="active" class="h-3.5 w-3.5" aria-hidden="true" />
      <ArrowUpDown v-else class="h-3.5 w-3.5 opacity-40" aria-hidden="true" />
    </button>
  </th>
</template>
