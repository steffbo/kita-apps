<script setup lang="ts">
import RulesSection from '@/components/parent-work/RulesSection.vue';
import ImportSection from '@/components/parent-work/ImportSection.vue';
</script>

<template>
  <div class="space-y-10">
    <div>
      <h1 class="text-2xl font-bold text-foreground">Verwaltung</h1>
      <p class="text-sm text-muted-foreground">Regelwerk und Import der Elternstunden</p>
    </div>
    <RulesSection />
    <ImportSection />
  </div>
</template>
