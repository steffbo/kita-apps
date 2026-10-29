<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref } from 'vue';
import { X } from 'lucide-vue-next';

// Dialog that slides up from the bottom on phones and sits centered from `sm` up.
// Esc is handled app-wide (utils/dialogEscape) through the „Schließen“ button.
const props = withDefaults(defineProps<{ title: string; hideTitle?: boolean; size?: 'sm' | 'md' | 'lg' }>(), {
  hideTitle: false,
  size: 'md',
});
const emit = defineEmits<{ close: [] }>();

const panel = ref<HTMLElement | null>(null);
let previousFocus: HTMLElement | null = null;
let previousOverflow = '';

onMounted(async () => {
  previousFocus = document.activeElement as HTMLElement | null;
  previousOverflow = document.body.style.overflow;
  document.body.style.overflow = 'hidden';
  await nextTick();
  // A field marked `autofocus` gets the focus; the attribute alone only works on page load.
  (panel.value?.querySelector<HTMLElement>('[autofocus]') ?? panel.value)?.focus();
});

onUnmounted(() => {
  document.body.style.overflow = previousOverflow;
  previousFocus?.focus?.();
});

const widths = { sm: 'sm:max-w-sm', md: 'sm:max-w-lg', lg: 'sm:max-w-2xl' };
</script>

<template>
  <Teleport to="body">
    <div class="fixed inset-0 z-[70] flex items-end justify-center bg-zinc-950/40 dark:bg-black/60 sm:items-center sm:p-4"
      @click.self="emit('close')">
      <div ref="panel" role="dialog" aria-modal="true" :aria-label="props.title" tabindex="-1"
        class="flex max-h-[88dvh] w-full flex-col rounded-t-2xl border-t bg-card text-card-foreground shadow-xl outline-none sm:max-h-[85vh] sm:rounded-xl sm:border"
        :class="widths[props.size]">
        <div class="mx-auto mt-2 h-1 w-9 shrink-0 rounded-full bg-border sm:hidden" aria-hidden="true" />
        <div class="flex shrink-0 items-center gap-2 px-4 pb-1 pt-2 sm:px-5 sm:pt-4">
          <h2 v-if="!props.hideTitle" class="flex-1 text-base font-semibold">{{ props.title }}</h2>
          <span v-else class="flex-1" />
          <button type="button" aria-label="Schließen" data-dialog-close
            class="-mr-2 grid h-10 w-10 place-items-center rounded-lg text-muted-foreground hover:bg-accent hover:text-foreground"
            @click="emit('close')"><X class="h-5 w-5" /></button>
        </div>
        <div class="min-h-0 flex-1 overflow-y-auto px-4 pb-[max(1rem,env(safe-area-inset-bottom))] sm:px-5 sm:pb-5">
          <slot />
        </div>
        <div v-if="$slots.footer"
          class="flex shrink-0 items-center justify-end gap-2 border-t px-4 py-3 pb-[max(0.75rem,env(safe-area-inset-bottom))] sm:px-5 max-sm:[&>.btn]:h-11 max-sm:[&>.btn]:flex-1">
          <slot name="footer" />
        </div>
      </div>
    </div>
  </Teleport>
</template>
