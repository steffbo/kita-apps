<script setup lang="ts">
import { computed, toRefs } from 'vue';
import type { ReminderCase } from '@/api/types';
import { RefreshCw } from 'lucide-vue-next';
import SearchInput from '@/components/SearchInput.vue';

const props = defineProps<{
  scope: 'actionable' | 'all';
  cases: ReminderCase[];
  isCasesLoading: boolean;
  casesError: string | null;
  caseSearch: string;
  filteredCases: ReminderCase[];
  loadCases: (selectId?: string | null) => Promise<void>;
}>();

const {
  cases,
  isCasesLoading,
  casesError,
  filteredCases,
  loadCases,
} = toRefs(props);

const emit = defineEmits<{
  'update:scope': [value: 'actionable' | 'all'];
  'update:caseSearch': [value: string];
}>();
const scope = computed({
  get: () => props.scope,
  set: (value) => emit('update:scope', value),
});
const caseSearch = computed({
  get: () => props.caseSearch,
  set: (value) => emit('update:caseSearch', value),
});
</script>

<template>
  <!-- Scope + search -->
  <div class="flex flex-col sm:flex-row sm:items-center gap-2 mb-4">
    <div class="inline-flex rounded-lg border overflow-hidden">
      <button
        class="px-4 py-2 text-sm font-medium transition-colors"
        :class="scope === 'actionable' ? 'bg-primary text-primary-foreground' : 'bg-card text-muted-foreground hover:bg-accent'"
        @click="scope = 'actionable'"
      >
        Handlungsbedarf
      </button>
      <button
        class="px-4 py-2 text-sm font-medium transition-colors"
        :class="scope === 'all' ? 'bg-primary text-primary-foreground' : 'bg-card text-muted-foreground hover:bg-accent'"
        @click="scope = 'all'"
      >
        Alle offenen
      </button>
    </div>
    <SearchInput v-model="caseSearch" placeholder="Familie suchen..." class="flex-1 min-w-[200px]" />
    <button
      class="inline-flex items-center gap-1.5 text-sm text-primary hover:underline disabled:opacity-50"
      :disabled="isCasesLoading"
      @click="loadCases()"
    >
      <RefreshCw class="h-4 w-4" :class="isCasesLoading ? 'animate-spin' : ''" />
      Neu laden
    </button>
  </div>

  <div v-if="casesError" class="mb-4 text-sm text-red-600 dark:text-red-300">{{ casesError }}</div>
  <div v-if="isCasesLoading && cases.length === 0" class="text-sm text-muted-foreground">Familien werden geladen...</div>
  <div v-else-if="filteredCases.length === 0" class="text-sm text-muted-foreground py-8 text-center">
    Keine offenen Fälle in dieser Ansicht.
  </div>
</template>
