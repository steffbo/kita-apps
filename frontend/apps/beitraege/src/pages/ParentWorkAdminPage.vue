<script setup lang="ts">
import RulesSection from '@/components/parent-work/RulesSection.vue';
import ImportSection from '@/components/parent-work/ImportSection.vue';
import PageHeader from '@/components/ui/PageHeader.vue';
import { useAuthStore } from '@/stores/auth';

const auth = useAuthStore();
</script>

<template>
  <div class="space-y-10">
    <PageHeader title="Verwaltung" />
    <RulesSection />
    <ImportSection v-if="!auth.isReadOnly" />
  </div>
</template>
