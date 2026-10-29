<script setup lang="ts">
import type { Component } from 'vue';

defineProps<{ icon?: Component; text: string }>();
</script>

<template>
  <div class="flex flex-col items-center gap-3 px-4 py-10 text-center text-sm text-muted-foreground">
    <span v-if="icon" class="grid h-11 w-11 place-items-center rounded-xl bg-muted">
      <component :is="icon" class="h-5 w-5" />
    </span>
    <p>{{ text }}</p>
    <slot />
  </div>
</template>
