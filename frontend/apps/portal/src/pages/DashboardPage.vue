<script setup lang="ts">
import { computed } from 'vue';
import { RouterLink } from 'vue-router';
import { ArrowRight, CircleDashed } from 'lucide-vue-next';
import { portalModules } from '@/lib/modules';
import { useAuthStore } from '@/stores/auth';

const authStore = useAuthStore();
const visibleModules = computed(() =>
  portalModules.filter((module) => authStore.hasAnyRole(module.roles)),
);
const readyModules = computed(() => visibleModules.value.filter((module) => module.state === 'ready'));
</script>

<template>
  <section class="mx-auto max-w-7xl px-5 py-8 sm:px-8 lg:px-10 lg:py-10">
    <div class="mb-9 grid gap-5 lg:grid-cols-[minmax(0,1fr)_320px] lg:items-end">
      <div>
        <p class="text-sm font-semibold uppercase tracking-[0.18em] text-primary">Portal</p>
        <h1 class="mt-3 text-3xl font-semibold tracking-normal sm:text-4xl">
          Guten Tag{{ authStore.displayName ? `, ${authStore.displayName}` : '' }}.
        </h1>
      </div>
      <div class="rounded-lg border bg-surface px-4 py-3">
        <p class="text-xs font-semibold uppercase tracking-[0.14em] text-muted-foreground">Aktive Bereiche</p>
        <p class="mt-2 text-3xl font-semibold tabular-nums">{{ readyModules.length }}</p>
      </div>
    </div>

    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-[1.1fr_0.9fr]">
      <RouterLink
        v-for="module in visibleModules"
        :key="module.id"
        :to="{ name: module.routeName }"
        class="group rounded-lg border bg-surface p-5 shadow-soft transition hover:-translate-y-0.5 hover:border-primary/40 active:translate-y-px"
      >
        <div class="flex items-start justify-between gap-4">
          <div class="flex min-w-0 items-start gap-4">
            <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg bg-muted text-foreground">
              <component :is="module.icon" class="h-5 w-5" :stroke-width="1.8" />
            </div>
            <div class="min-w-0">
              <h2 class="text-lg font-semibold">{{ module.label }}</h2>
              <p class="mt-1 text-sm leading-6 text-muted-foreground">{{ module.description }}</p>
            </div>
          </div>
          <ArrowRight
            v-if="module.state === 'ready'"
            class="mt-1 h-4 w-4 shrink-0 text-muted-foreground transition group-hover:translate-x-0.5 group-hover:text-primary"
            :stroke-width="1.8"
          />
          <CircleDashed v-else class="mt-1 h-4 w-4 shrink-0 text-muted-foreground" :stroke-width="1.8" />
        </div>
        <div class="mt-5 flex items-center justify-between border-t pt-3">
          <span class="text-xs font-semibold uppercase tracking-[0.14em] text-muted-foreground">Status</span>
          <span
            class="rounded-md px-2 py-1 text-xs font-semibold"
            :class="module.state === 'ready' ? 'bg-emerald-50 text-emerald-700' : 'bg-amber-soft text-amber-foreground'"
          >
            {{ module.state === 'ready' ? 'bereit' : 'später' }}
          </span>
        </div>
      </RouterLink>
    </div>
  </section>
</template>
