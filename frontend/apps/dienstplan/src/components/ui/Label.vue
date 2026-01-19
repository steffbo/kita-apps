<script setup lang="ts">
import { type HTMLAttributes, computed } from 'vue';
import { Label as LabelPrimitive } from 'radix-vue';
import { cn } from '@/lib/utils';

const props = defineProps<{
  class?: HTMLAttributes['class'];
  for?: string;
}>();

const delegatedProps = computed(() => {
  const { class: _, ...delegated } = props;
  return delegated;
});
</script>

<template>
  <LabelPrimitive
    v-bind="delegatedProps"
    :class="cn('text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70', props.class)"
  >
    <slot />
  </LabelPrimitive>
</template>
