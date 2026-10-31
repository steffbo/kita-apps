<script setup lang="ts">
import { defineAsyncComponent } from 'vue';
import { useAuthStore } from '@/stores/auth';

// `/` shows the fee dashboard for staff and the Stichtag dashboard for the Leitung.
const DashboardPage = defineAsyncComponent(() => import('@/pages/DashboardPage.vue'));
const LeitungDashboardPage = defineAsyncComponent(() => import('@/pages/LeitungDashboardPage.vue'));
const auth = useAuthStore();
</script>

<template>
  <LeitungDashboardPage v-if="auth.isLeitung" />
  <DashboardPage v-else />
</template>
