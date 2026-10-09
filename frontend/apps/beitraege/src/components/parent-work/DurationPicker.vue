<script setup lang="ts">
import { computed, ref, useId, watch } from 'vue';
import { Minus, Plus } from 'lucide-vue-next';
import { formatDecimalNumber } from '@/utils/format';

// Duration in minutes (multiples of 15). One text field that accepts „2,5“, „2:30“, „2h30“ or „150 min“,
// ±15-minute steppers and quick picks for the usual values (2,5 h is one click).
const props = defineProps<{ label: string }>();
const model = defineModel<number | null>({ default: null });
const id = useId();
const presets = [30, 60, 90, 120, 150, 180, 240];
const text = ref(format(model.value));
const touched = ref(false);

function format(minutes: number | null): string {
  return minutes ? formatDecimalNumber(minutes / 60, 2) : '';
}
function parse(raw: string): number | null {
  const s = raw.trim().toLowerCase().replace(/\s+/g, ' ');
  let m: RegExpMatchArray | null;
  let minutes: number;
  if ((m = s.match(/^(\d+):(\d{1,2})$/))) minutes = Number(m[1]) * 60 + Number(m[2]);
  else if ((m = s.match(/^(\d+) ?(?:h|std\.?) ?(\d+) ?(?:m|min\.?)?$/))) minutes = Number(m[1]) * 60 + Number(m[2]);
  else if ((m = s.match(/^(\d+) ?(?:m|min\.?|minuten)$/))) minutes = Number(m[1]);
  else if ((m = s.match(/^(\d+(?:[.,]\d+)?) ?(?:h|std\.?|stunden?)?$/))) {
    minutes = Math.round(Number(m[1].replace(',', '.')) * 60);
  } else return null;
  return minutes > 0 && minutes % 15 === 0 && minutes <= 24 * 60 ? minutes : null;
}
const invalid = computed(() => text.value.trim() !== '' && parse(text.value) === null);
const spelled = computed(() => {
  if (!model.value) return '';
  const h = Math.floor(model.value / 60), m = model.value % 60;
  return [h ? `${h} Std.` : '', m ? `${m} Min.` : ''].filter(Boolean).join(' ');
});

watch(model, (value) => { if (parse(text.value) !== value) text.value = format(value); });
function onInput(value: string) {
  text.value = value;
  model.value = parse(value);
}
function onBlur() {
  touched.value = true;
  if (model.value) text.value = format(model.value);
}
function step(delta: number) {
  const next = model.value ? model.value + delta : delta > 0 ? 60 : null;
  if (next && next >= 15 && next <= 24 * 60) model.value = next;
}
</script>

<template>
  <div>
    <label :for="id" class="block text-sm font-medium">{{ props.label }}</label>
    <div class="mt-1 flex items-center gap-3">
      <div class="flex h-10 items-stretch overflow-hidden rounded-lg border bg-card focus-within:ring-2
        focus-within:ring-ring" :class="{ 'border-red-500': touched && invalid }">
        <button type="button" class="px-2.5 text-muted-foreground hover:bg-accent disabled:opacity-40"
          aria-label="15 Minuten weniger" :disabled="!model || model <= 15" @click="step(-15)">
          <Minus class="h-4 w-4" />
        </button>
        <input :id="id" :value="text" inputmode="decimal" autocomplete="off" placeholder="z. B. 2,5"
          :aria-invalid="touched && invalid" :aria-describedby="`${id}-hint`"
          class="w-20 border-x bg-transparent text-center tabular-nums outline-none"
          @input="onInput(($event.target as HTMLInputElement).value)" @blur="onBlur" />
        <span class="flex items-center border-r px-2 text-sm text-muted-foreground">Std.</span>
        <button type="button" class="px-2.5 text-muted-foreground hover:bg-accent"
          aria-label="15 Minuten mehr" @click="step(15)">
          <Plus class="h-4 w-4" />
        </button>
      </div>
      <span :id="`${id}-hint`" class="text-sm" :class="touched && invalid ? 'text-red-700 dark:text-red-300'
        : 'text-muted-foreground'" aria-live="polite">
        {{ touched && invalid ? 'In Viertelstunden, z. B. 2,5 oder 2:30' : spelled }}
      </span>
    </div>
    <div role="group" aria-label="Schnellwahl" class="mt-2 flex flex-wrap gap-1.5">
      <button v-for="p in presets" :key="p" type="button" :aria-pressed="model === p"
        :aria-label="`${format(p)} Std.`"
        class="min-w-11 rounded-full border px-3 py-1 text-sm tabular-nums transition-colors"
        :class="model === p ? 'border-primary bg-primary text-primary-foreground'
          : 'text-muted-foreground hover:border-primary hover:text-foreground'"
        @click="model = p">{{ format(p) }}</button>
    </div>
  </div>
</template>
