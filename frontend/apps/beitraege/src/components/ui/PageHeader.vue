<script setup lang="ts">
import { RouterLink, type RouteLocationRaw } from 'vue-router';
import { ChevronLeft } from 'lucide-vue-next';

// Page title with the page's actions on the right (primary action last). No subtitle: counts go into
// filter chips, explanations only where they carry information.
defineProps<{ title: string; back?: RouteLocationRaw; backLabel?: string }>();
</script>

<template>
  <div class="mb-5 flex flex-wrap items-center gap-x-4 gap-y-3 sm:mb-6">
    <div class="flex min-w-0 max-w-full flex-[1_0_auto] items-center gap-1">
      <RouterLink v-if="back" :to="back" :aria-label="backLabel ?? 'Zurück'"
        class="-ml-2 grid h-10 w-10 shrink-0 place-items-center rounded-lg text-muted-foreground hover:bg-accent hover:text-foreground">
        <ChevronLeft class="h-5 w-5" />
      </RouterLink>
      <h1 class="min-w-0 truncate text-2xl font-semibold tracking-tight"><slot name="title">{{ title }}</slot></h1>
      <slot name="meta" />
    </div>
    <div v-if="$slots.actions" class="flex flex-wrap items-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>
