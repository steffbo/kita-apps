<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import { api } from '@/api';
import type { Household } from '@/api/types';
import { Check, Loader2 } from 'lucide-vue-next';
import SearchInput from '@/components/SearchInput.vue';

// Searchable single-select list of households. `modelValue` is the selected household id.
defineProps<{ modelValue: string | null }>();
const emit = defineEmits<{
  'update:modelValue': [id: string];
  select: [household: Household];
  error: [message: string];
}>();

const households = ref<Household[]>([]);
const isLoading = ref(false);
const query = ref('');
let request = 0;
let timer: ReturnType<typeof setTimeout> | undefined;

// Only the latest search may fill the list; the endpoint returns at most 100 households.
async function load(): Promise<void> {
  const current = ++request;
  isLoading.value = true;
  try {
    const result = await api.getHouseholds({ search: query.value.trim() || undefined, perPage: 100 });
    if (current === request) households.value = result.data;
  } catch (e) {
    emit('error', e instanceof Error ? e.message : 'Haushalte konnten nicht geladen werden');
  } finally {
    if (current === request) isLoading.value = false;
  }
}

watch(query, () => {
  clearTimeout(timer);
  timer = setTimeout(() => void load(), 150);
});
onMounted(load);

function pick(household: Household): void {
  emit('update:modelValue', household.id);
  emit('select', household);
}
</script>

<template>
  <div class="space-y-2">
    <SearchInput v-model="query" placeholder="Familie suchen" label="Haushalt suchen" />
    <div v-if="isLoading" class="flex justify-center py-4">
      <Loader2 class="h-5 w-5 animate-spin text-muted-foreground" />
    </div>
    <ul v-else class="card max-h-64 divide-y overflow-y-auto">
      <li v-for="household in households" :key="household.id">
        <button type="button"
          class="flex min-h-12 w-full items-center gap-3 px-4 py-2.5 text-left hover:bg-accent"
          :class="modelValue === household.id && 'bg-primary-soft text-primary-text'"
          :aria-pressed="modelValue === household.id" @click="pick(household)">
          <span class="min-w-0 flex-1 truncate font-medium">{{ household.name }}</span>
          <Check v-if="modelValue === household.id" class="h-4 w-4 shrink-0" />
        </button>
      </li>
      <li v-if="households.length === 0" class="px-4 py-6 text-center text-sm text-muted-foreground">
        Kein Haushalt gefunden
      </li>
    </ul>
  </div>
</template>
