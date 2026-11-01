<script setup lang="ts">
// One label/value line of a `<dl class="card divide-y text-sm">`.
defineProps<{ label: string; valueClass?: string }>();
</script>

<template>
  <div class="flex justify-between gap-4 px-4 py-2.5">
    <dt class="text-muted-foreground">{{ label }}</dt>
    <dd :class="valueClass"><slot /></dd>
  </div>
</template>
