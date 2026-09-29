<script setup lang="ts">
import { computed, ref } from 'vue';
import type { ParentWorkHouseholdOption } from '@/api/types';
import SearchInput from '@/components/SearchInput.vue';

const props = defineProps<{
  households: ParentWorkHouseholdOption[];
  modelValue?: string;
  label: string;
}>();
const emit = defineEmits<{ 'update:modelValue': [value: string] }>();
const search = ref('');
const open = ref(false);
const selected = computed(() => props.households.find(h => h.id === props.modelValue));
const matches = computed(() => {
  const query = search.value.trim().toLocaleLowerCase('de');
  return props.households.filter(h => !query || [h.name, ...h.children.map(c => c.name),
    ...h.members.map(m => m.name), ...h.parents.map(p => p.name)]
    .some(name => name.toLocaleLowerCase('de').includes(query))).slice(0, 30);
});
function select(id: string) {
  emit('update:modelValue', id);
  search.value = '';
  open.value = false;
}
</script>

<template>
  <div class="relative">
    <label class="block text-sm font-medium">{{ label }}
      <SearchInput v-model="search" placeholder="Familie, Kind oder Mitglied suchen" class="mt-1 font-normal"
        @focus="open = true" @input="open = true" @blur="open = false" @keydown.escape="open = false" />
    </label>
    <p v-if="selected" class="mt-1 text-sm text-foreground">Familie: {{ selected.name }}</p>
    <p v-else class="mt-1 text-sm text-amber-700 dark:text-amber-300">Keine Familie ausgewählt</p>
    <div v-if="open"
      class="absolute z-20 mt-1 max-h-48 w-full overflow-y-auto rounded-lg border bg-popover shadow-lg">
      <button v-for="household in matches" :key="household.id" type="button"
        class="block w-full px-3 py-2 text-left text-sm hover:bg-accent"
        @mousedown.prevent @click="select(household.id)">{{ household.name }}</button>
      <p v-if="!matches.length" class="px-3 py-2 text-sm text-muted-foreground">Keine Familie gefunden</p>
    </div>
  </div>
</template>
