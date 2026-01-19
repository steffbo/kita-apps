<script setup lang="ts">
import { computed } from 'vue';
import { cn } from '@/lib/utils';

const props = defineProps<{
  class?: string;
  variant?: 'default' | 'destructive' | 'outline' | 'secondary';
}>();

const variantClasses = computed(() => {
  switch (props.variant) {
    case 'destructive':
      return 'border-transparent bg-destructive text-destructive-foreground hover:bg-destructive/80';
    case 'outline':
      return 'text-foreground';
    case 'secondary':
      return 'border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/80';
    default:
      return 'border-transparent bg-primary text-primary-foreground hover:bg-primary/80';
  }
});
</script>

<template>
  <div
    :class="cn(
      'inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2',
      variantClasses,
      props.class
    )"
  >
    <slot />
  </div>
</template>
