<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router';
import { useNavigation } from '@/composables/useNavigation';

// „Aufgaben“ bundles everything that waits for someone to act; each section keeps its URL.
const { taskTabs, isActive } = useNavigation();
</script>

<template>
  <div>
    <h1 class="text-2xl font-semibold tracking-tight">Aufgaben</h1>
    <nav v-if="taskTabs.length > 1" aria-label="Aufgaben-Bereiche"
      class="-mx-4 mb-6 mt-3 flex gap-1 overflow-x-auto border-b px-4 sm:mx-0 sm:px-0">
      <RouterLink v-for="tab in taskTabs" :key="tab.to" :to="tab.to"
        class="-mb-px flex h-11 shrink-0 items-center gap-1.5 border-b-2 px-2.5 text-sm font-medium"
        :class="isActive(tab) ? 'border-primary text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground'"
        :aria-current="isActive(tab) ? 'page' : undefined">
        {{ tab.name }}
        <span v-if="tab.count" class="rounded-full px-1.5 text-xs tabular-nums"
          :class="isActive(tab) ? 'bg-primary-soft text-primary-text' : 'bg-muted'">{{ tab.count }}</span>
      </RouterLink>
    </nav>
    <div v-else class="mb-6" />
    <RouterView />
  </div>
</template>
