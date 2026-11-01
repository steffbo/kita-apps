<script setup lang="ts">
import { ref, watch } from 'vue';
import { api } from '@/api';
import type { DonationReport } from '@/api/types';
import DonationYearList from '@/components/DonationYearList.vue';

// Donations credited to a household, for staff looking at that household. Hidden when there are none.
const props = defineProps<{ householdId: string }>();

const report = ref<DonationReport | null>(null);
const error = ref('');

watch(() => props.householdId, async (id) => {
  error.value = '';
  try {
    report.value = await api.getDonations(id);
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Spenden konnten nicht geladen werden';
  }
}, { immediate: true });
</script>

<template>
  <p v-if="error" role="alert" class="note note-danger">{{ error }}</p>
  <section v-else-if="report?.count" aria-labelledby="household-donations-heading">
    <h2 id="household-donations-heading" class="mb-2 text-base font-semibold">Spenden des Haushalts</h2>
    <DonationYearList :years="report.years" />
  </section>
</template>
