<script setup lang="ts">
import { RouterLink } from 'vue-router';
import { useNavigation } from '@/composables/useNavigation';

defineProps<{ large?: boolean }>();
const { groups, isActive } = useNavigation();
</script>

<template>
  <div :class="large ? 'space-y-4' : 'space-y-3'">
    <div v-for="group in groups" :key="group.label || 'main'">
      <p v-if="group.label" class="mb-1 px-2 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
        {{ group.label }}
      </p>
      <RouterLink v-for="item in group.items" :key="item.name" :to="item.to"
        class="group flex items-center gap-2.5 rounded-lg px-2 text-sm hover:bg-accent"
        :class="[large ? 'h-11 text-[15px]' : 'h-8', isActive(item) && 'bg-primary-soft font-medium text-primary-text hover:bg-primary-soft']"
        :aria-current="isActive(item) ? 'page' : undefined">
        <component :is="item.icon" class="h-4 w-4 shrink-0"
          :class="isActive(item) ? 'text-primary-text' : 'text-muted-foreground'" />
        <span class="truncate">{{ item.name }}</span>
        <span v-if="item.count" class="ml-auto text-xs tabular-nums"
          :class="isActive(item) ? 'text-primary-text' : 'text-muted-foreground'">{{ item.count }}</span>
      </RouterLink>
    </div>
  </div>
</template>
