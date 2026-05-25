<script setup lang="ts">
import { ClipboardCheck } from 'lucide-vue-next';
</script>

<template>
  <section class="mx-auto max-w-7xl px-5 py-8 sm:px-8 lg:px-10 lg:py-10">
    <div class="mb-8">
      <p class="text-sm font-semibold uppercase tracking-[0.18em] text-primary">Abnahme</p>
      <h1 class="mt-3 text-3xl font-semibold tracking-normal sm:text-4xl">Eingereichte Stunden</h1>
    </div>

    <div class="rounded-lg border bg-surface p-6 shadow-soft">
      <div class="flex min-h-72 flex-col items-center justify-center rounded-lg border border-dashed bg-background px-5 text-center">
        <ClipboardCheck class="mb-4 h-9 w-9 text-primary" :stroke-width="1.8" />
        <p class="text-sm font-semibold">Keine offenen Einreichungen</p>
        <p class="mt-2 max-w-[42ch] text-sm leading-6 text-muted-foreground">
          Sobald Elternstunden eingereicht werden, erscheinen sie hier zur Prüfung.
        </p>
      </div>
    </div>
  </section>
</template>
