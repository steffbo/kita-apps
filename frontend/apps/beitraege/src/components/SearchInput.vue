<script setup lang="ts">
import { computed, useAttrs } from 'vue';
import { Loader2, Search } from 'lucide-vue-next';

// Einheitliches Suchfeld: Lupe links, optionaler Ladeindikator rechts. `class` gilt dem
// Wrapper (Layout), alle anderen Attribute und Listener (focus, blur, input …) dem Eingabefeld.
defineOptions({ inheritAttrs: false });
withDefaults(defineProps<{ placeholder: string; label?: string; loading?: boolean; size?: 'md' | 'sm' }>(),
  { size: 'md' });
const model = defineModel<string>({ required: true });
const attrs = useAttrs();
const inputAttrs = computed(() => ({ ...attrs, class: undefined }));
</script>

<template>
  <div class="relative" :class="attrs.class">
    <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
    <input v-model="model" v-bind="inputAttrs" type="search" :placeholder="placeholder" :aria-label="label"
      class="w-full rounded-lg border outline-none focus:border-transparent focus:ring-2 focus:ring-primary"
      :class="size === 'sm' ? 'py-1.5 pl-9 pr-3 text-sm' : 'py-2 pl-10 pr-10'" />
    <Loader2 v-if="loading"
      class="absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 animate-spin text-muted-foreground" />
  </div>
</template>
