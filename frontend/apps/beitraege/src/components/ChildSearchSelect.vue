<script setup lang="ts">
import EmptyState from '@/components/ui/EmptyState.vue';

import { computed, nextTick, ref, useId } from 'vue';
import type { Child, Household } from '@/api/types';
import SearchInput from '@/components/SearchInput.vue';
import { formatDate, todayISO } from '@/utils/format';

// Kindauswahl per Suche: ab zwei Zeichen werden Treffer aus Vorname, Nachname und
// Mitgliedsnummer angezeigt; mehrere Suchwörter müssen alle vorkommen ("zoe akz").
const MIN_QUERY_LENGTH = 2;
const MAX_RESULTS = 20;

const props = defineProps<{
  children: Child[];
  households?: Household[];
  disabled?: boolean;
}>();
const model = defineModel<string>({ required: true });

const listId = useId();
const search = ref('');
const open = ref(false);
const activeIndex = ref(0);
const searchInput = ref<InstanceType<typeof SearchInput> | null>(null);

const householdNames = computed(() => new Map((props.households ?? []).map((h) => [h.id, h.name])));
const selected = computed(() => props.children.find((c) => c.id === model.value) ?? null);

function hasLeft(child: Child): boolean {
  return !!child.exitDate && child.exitDate.split('T')[0] < todayISO();
}

const tokens = computed(() => search.value.trim().toLocaleLowerCase('de').split(/\s+/).filter(Boolean));
const queryLongEnough = computed(() => tokens.value.join('').length >= MIN_QUERY_LENGTH);

const allMatches = computed(() => {
  if (!queryLongEnough.value) return [];
  return props.children
    .filter((child) => {
      const haystack = `${child.firstName} ${child.lastName} ${child.memberNumber}`.toLocaleLowerCase('de');
      return tokens.value.every((token) => haystack.includes(token));
    })
    .sort((left, right) =>
      Number(hasLeft(left)) - Number(hasLeft(right)) ||
      left.lastName.localeCompare(right.lastName, 'de') ||
      left.firstName.localeCompare(right.firstName, 'de'));
});
const matches = computed(() => allMatches.value.slice(0, MAX_RESULTS));

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
    const child = matches.value[activeIndex.value];
    if (open.value && child) select(child);
  } else if (event.key === 'Escape') {
    open.value = false;
  }
}

function select(child: Child) {
  model.value = child.id;
  search.value = '';
  open.value = false;
}

async function change() {
  model.value = '';
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
          <template v-if="selected.householdId && householdNames.get(selected.householdId)">
            · {{ householdNames.get(selected.householdId) }}
          </template>
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
            <template v-if="child.householdId && householdNames.get(child.householdId)">
              · {{ householdNames.get(child.householdId) }}
            </template>
            <template v-if="hasLeft(child)">· ausgetreten {{ formatDate(child.exitDate) }}</template>
          </span>
        </button>
        <EmptyState v-if="!matches.length" text="Kein Kind gefunden" />
        <p v-else-if="allMatches.length > matches.length" class="px-3 py-2 text-xs text-muted-foreground">
          {{ allMatches.length - matches.length }} weitere Treffer – bitte Suche verfeinern.
        </p>
      </div>
    </template>
  </div>
</template>
