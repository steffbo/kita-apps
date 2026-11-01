<script setup lang="ts">
import EmptyState from '@/components/ui/EmptyState.vue';

import { computed, nextTick, onUnmounted, ref, useId, watch } from 'vue';
import { api } from '@/api';
import type { Child } from '@/api/types';
import SearchInput from '@/components/SearchInput.vue';
import { formatDate, todayISO } from '@/utils/format';

// Kindauswahl per Suche: ab zwei Zeichen sucht der Server in Vorname, Nachname und
// Mitgliedsnummer; mehrere Suchwörter müssen alle vorkommen ("zoe akz").
const MIN_QUERY_LENGTH = 2;
const MAX_RESULTS = 20;
const FETCHED_RESULTS = 100;

defineProps<{
  selected: Child | null;
  disabled?: boolean;
}>();
const emit = defineEmits<{ select: [child: Child | null] }>();

const listId = useId();
const search = ref('');
const open = ref(false);
const activeIndex = ref(0);
const results = ref<Child[]>([]);
const total = ref(0);
const isSearching = ref(false);
const searchError = ref<string | null>(null);
const searchInput = ref<InstanceType<typeof SearchInput> | null>(null);

function hasLeft(child: Child): boolean {
  return !!child.exitDate && child.exitDate.split('T')[0] < todayISO();
}

const query = computed(() => search.value.trim());
const queryLongEnough = computed(() => query.value.replace(/\s+/g, '').length >= MIN_QUERY_LENGTH);

const matches = computed(() => [...results.value]
  .sort((left, right) =>
    Number(hasLeft(left)) - Number(hasLeft(right)) ||
    left.lastName.localeCompare(right.lastName, 'de') ||
    left.firstName.localeCompare(right.firstName, 'de'))
  .slice(0, MAX_RESULTS));

let request = 0;
let timer: ReturnType<typeof setTimeout> | undefined;
// Enter while the search is still running picks the first result when it arrives.
let selectOnResults = false;

// Old results vanish at once, so Enter cannot pick a child from the previous search.
watch(query, (value) => {
  clearTimeout(timer);
  const current = ++request;
  selectOnResults = false;
  results.value = [];
  total.value = 0;
  searchError.value = null;
  isSearching.value = queryLongEnough.value;
  if (!queryLongEnough.value) return;
  timer = setTimeout(async () => {
    try {
      const response = await api.getChildren({ search: value, perPage: FETCHED_RESULTS });
      if (current !== request) return;
      results.value = response.data;
      total.value = response.total;
      if (selectOnResults && matches.value[0]) select(matches.value[0]);
    } catch (e) {
      if (current === request) searchError.value = e instanceof Error ? e.message : 'Suche fehlgeschlagen';
    } finally {
      if (current === request) isSearching.value = false;
    }
  }, 150);
});
onUnmounted(() => {
  clearTimeout(timer);
  request++;
});

function optionId(index: number) {
  return `${listId}-option-${index}`;
}

function onInput() {
  open.value = true;
  activeIndex.value = 0;
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault();
    open.value = true;
    if (!matches.value.length) return;
    const step = event.key === 'ArrowDown' ? 1 : -1;
    activeIndex.value = (activeIndex.value + step + matches.value.length) % matches.value.length;
  } else if (event.key === 'Enter') {
    // Enter wählt den markierten Treffer, statt das Formular abzuschicken.
    event.preventDefault();
    if (isSearching.value) selectOnResults = true;
    const child = matches.value[activeIndex.value];
    if (open.value && child) select(child);
  } else if (event.key === 'Escape') {
    open.value = false;
  }
}

function select(child: Child) {
  emit('select', child);
  search.value = '';
  open.value = false;
}

async function change() {
  emit('select', null);
  await nextTick();
  searchInput.value?.$el.querySelector('input')?.focus();
}
</script>

<template>
  <div class="relative">
    <div
      v-if="selected"
      class="flex items-center justify-between gap-3 rounded-lg border px-3 py-2 text-sm"
      :class="disabled ? 'bg-muted text-muted-foreground' : 'bg-card'">
      <div class="min-w-0">
        <p class="truncate font-medium text-foreground">
          {{ selected.firstName }} {{ selected.lastName }}
          <span class="font-normal text-muted-foreground">({{ selected.memberNumber }})</span>
        </p>
        <p class="truncate text-xs text-muted-foreground">
          geb. {{ formatDate(selected.birthDate) }} · Eintritt {{ formatDate(selected.entryDate) }}
          <template v-if="selected.household"> · {{ selected.household.name }}</template>
        </p>
      </div>
      <button v-if="!disabled" type="button" class="btn btn-secondary shrink-0" @click="change">Ändern</button>
    </div>
    <template v-else>
      <SearchInput
        ref="searchInput"
        v-model="search"
        placeholder="Name oder Mitgliedsnummer"
        label="Kind suchen"
        :disabled="disabled"
        :loading="isSearching"
        role="combobox"
        autocomplete="off"
        :aria-expanded="open && queryLongEnough"
        :aria-controls="listId"
        :aria-activedescendant="open && matches.length ? optionId(activeIndex) : undefined"
        @focus="open = true"
        @input="onInput"
        @blur="open = false"
        @keydown="onKeydown" />
      <p v-if="!queryLongEnough" class="mt-1 text-xs text-muted-foreground">
        Mindestens {{ MIN_QUERY_LENGTH }} Zeichen eingeben.
      </p>
      <div
        v-if="open && queryLongEnough"
        :id="listId"
        role="listbox"
        class="absolute z-20 mt-1 max-h-72 w-full overflow-y-auto rounded-lg border bg-popover shadow-lg">
        <button
          v-for="(child, index) in matches"
          :id="optionId(index)"
          :key="child.id"
          type="button"
          role="option"
          :aria-selected="index === activeIndex"
          class="block w-full px-3 py-2 text-left text-sm"
          :class="index === activeIndex ? 'bg-accent' : 'hover:bg-accent'"
          @mousedown.prevent
          @mouseenter="activeIndex = index"
          @click="select(child)">
          <span class="font-medium" :class="hasLeft(child) ? 'text-muted-foreground' : 'text-foreground'">
            {{ child.firstName }} {{ child.lastName }}
          </span>
          <span class="text-muted-foreground">({{ child.memberNumber }})</span>
          <span class="block text-xs text-muted-foreground">
            geb. {{ formatDate(child.birthDate) }}
            <template v-if="child.household"> · {{ child.household.name }}</template>
            <template v-if="hasLeft(child)">· ausgetreten {{ formatDate(child.exitDate) }}</template>
          </span>
        </button>
        <p v-if="searchError" role="alert" class="px-3 py-2 text-sm text-destructive">{{ searchError }}</p>
        <p v-else-if="isSearching" class="px-3 py-2 text-sm text-muted-foreground">Suche …</p>
        <EmptyState v-else-if="!matches.length" text="Kein Kind gefunden" />
        <p v-else-if="total > matches.length" class="px-3 py-2 text-xs text-muted-foreground">
          {{ total - matches.length }} weitere Treffer – bitte Suche verfeinern.
        </p>
      </div>
    </template>
  </div>
</template>
