<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import { RouterLink } from 'vue-router';
import { api } from '@/api';
import type { ParentReport } from '@/api/types';
import { formatDate } from '@/utils/format';
const status = ref<'OPEN' | 'DONE' | 'ALL'>('OPEN');
const reports = ref<ParentReport[]>([]);
const error = ref('');
const feeChildren = ref<Record<string, string>>({});
const answers = ref<Record<string, string>>({});
async function load() {
  try {
    reports.value = await api.getParentReports(status.value);
    const feeIds = [...new Set(reports.value.filter(r => r.topic === 'FEE' && r.referenceId)
      .map(r => r.referenceId!))];
    const fees = await Promise.all(feeIds.map(id => api.getFee(id).catch(() => null)));
    for (const fee of fees) if (fee) feeChildren.value[fee.id] = fee.childId;
  }
  catch (e) { error.value = e instanceof Error ? e.message : 'Meldungen konnten nicht geladen werden'; }
}
onMounted(load); watch(status, load);
async function resolve(id: string) {
  error.value = '';
  try { await api.resolveParentReport(id, answers.value[id] ?? ''); delete answers.value[id]; await load(); }
  catch (e) { error.value = e instanceof Error ? e.message : 'Erledigen fehlgeschlagen'; }
}
function link(report: ParentReport) {
  if (report.topic === 'FEE' && report.referenceId && feeChildren.value[report.referenceId]) {
    return `/kinder/${feeChildren.value[report.referenceId]}`;
  }
  if (report.topic === 'CHILD' && report.referenceId) return `/kinder/${report.referenceId}`;
  if (report.topic === 'PARENT_WORK') return `/elternstunden/familien/${report.householdId}`;
  return `/eltern/${report.parentId}`;
}
const topics: Record<string, string> = {
  GENERAL: 'Allgemein', CHILD: 'Kind', FEE: 'Beitrag',
  PARENT_WORK: 'Elternstunden', CONTACT: 'Kontaktdaten',
};
</script>
<template>
  <div class="space-y-5">
    <h1 class="text-2xl font-bold">Meldungen</h1>
    <label class="block max-w-44 text-sm font-medium">Status
      <select v-model="status" class="mt-1 w-full rounded-lg border px-3 py-2">
        <option value="OPEN">Offen</option><option value="DONE">Erledigt</option>
        <option value="ALL">Alle</option>
      </select>
    </label>
    <p v-if="error" role="alert" class="text-red-700 dark:text-red-300">{{ error }}</p>
    <p v-if="!reports.length" class="text-muted-foreground">Keine Meldungen.</p>
    <article v-for="report in reports" :key="report.id" class="rounded-2xl border bg-card p-4">
      <div class="flex items-start justify-between gap-2"><h2 class="font-bold">{{ topics[report.topic] ?? report.topic }}</h2>
        <span class="shrink-0 whitespace-nowrap rounded-full px-2.5 py-0.5 text-xs font-semibold" :class="
          report.status === 'OPEN'
            ? 'bg-amber-100 text-amber-800 dark:bg-amber-950/40 dark:text-amber-300'
            : 'bg-green-100 text-green-800 dark:bg-green-950/40 dark:text-green-300'">
          {{ report.status === 'OPEN' ? 'Offen' : '✓ Erledigt' }}</span></div>
      <p class="mt-2 whitespace-pre-wrap">{{ report.message }}</p>
      <p class="mt-2 text-sm text-muted-foreground">{{ formatDate(report.createdAt) }}<template
        v-if="report.parentName"> · von {{ report.parentName }}</template></p>
      <p v-if="report.response" class="mt-2 rounded-lg bg-muted p-3 text-sm">
        <span class="font-semibold">Antwort:</span> {{ report.response }}</p>
      <label v-if="report.status === 'OPEN'" class="mt-3 block text-sm font-medium">
        Antwort an die Eltern (optional)
        <textarea v-model="answers[report.id]" rows="2" maxlength="2000"
          class="mt-1 w-full rounded-lg border px-3 py-2" /></label>
      <div class="mt-3 flex flex-wrap items-center gap-4 text-sm"><RouterLink :to="link(report)" class="text-primary underline">
        Bezug ansehen</RouterLink>
        <button v-if="report.status === 'OPEN'" class="rounded-lg bg-primary px-3 py-1.5 text-primary-foreground"
          @click="resolve(report.id)">{{ answers[report.id]?.trim() ? 'Antworten und erledigen' : 'Erledigt' }}</button></div>
    </article>
  </div>
</template>
