<script setup lang="ts">
import { computed } from 'vue';
import { formatHours } from '@/utils/format';

// Duration in minutes (multiples of 15): hour and minute quick picks, e.g. 2,5 h = „2“ + „30“.
const props = defineProps<{ label: string }>();
const model = defineModel<number | null>({ default: null });
const hourChoices = [0, 1, 2, 3, 4, 5, 6, 7, 8];
const minuteChoices = [0, 15, 30, 45];
const hours = computed(() => model.value == null ? null : Math.floor(model.value / 60));
const minutes = computed(() => model.value == null ? null : model.value % 60);
function setHours(value: number) {
  if (!Number.isFinite(value) || value < 0) return;
  model.value = Math.floor(value) * 60 + (minutes.value ?? 0);
}
function setMinutes(value: number) {
  model.value = (hours.value ?? 0) * 60 + value;
}
const chip = (active: boolean) => ['h-9 min-w-9 rounded-lg border px-2 text-sm font-medium tabular-nums',
  active ? 'border-primary bg-primary text-primary-foreground' : 'bg-card hover:bg-accent'];
</script>

<template>
  <fieldset>
    <legend class="text-sm font-medium">{{ props.label }}</legend>
    <div role="group" aria-label="Stunden" class="mt-1 flex flex-wrap items-center gap-1.5">
      <span class="w-16 text-sm text-muted-foreground">Stunden</span>
      <button v-for="h in hourChoices" :key="h" type="button" :class="chip(hours === h)"
        :aria-pressed="hours === h" @click="setHours(h)">{{ h }}</button>
      <input type="number" min="0" max="24" step="1" aria-label="Stunden eingeben"
        :value="hours != null && hours > 8 ? hours : ''" class="h-9 w-16 rounded-lg border px-2 text-sm" placeholder="mehr"
        @input="setHours(Number(($event.target as HTMLInputElement).value))" />
    </div>
    <div role="group" aria-label="Minuten" class="mt-2 flex flex-wrap items-center gap-1.5">
      <span class="w-16 text-sm text-muted-foreground">Minuten</span>
      <button v-for="m in minuteChoices" :key="m" type="button" :class="chip(minutes === m)"
        :aria-pressed="minutes === m" @click="setMinutes(m)">{{ m }}</button>
    </div>
    <p class="mt-2 text-sm text-muted-foreground" aria-live="polite">
      {{ model ? `Dauer: ${formatHours(model)}` : 'Noch keine Dauer gewählt.' }}
    </p>
  </fieldset>
</template>
