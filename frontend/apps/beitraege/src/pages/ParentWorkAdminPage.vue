<script setup lang="ts">
import RulesSection from '@/components/parent-work/RulesSection.vue';
import ImportSection from '@/components/parent-work/ImportSection.vue';
import PageHeader from '@/components/ui/PageHeader.vue';
</script>

<template>
  <div class="space-y-10">
    <PageHeader title="Verwaltung" />
    <RulesSection />
    <ImportSection />
  </div>
</template>
