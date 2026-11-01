<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import { RouterLink } from 'vue-router';
import { api } from '@/api';
import type { ParentReport } from '@/api/types';
import { formatDate } from '@/utils/format';
import EmptyState from '@/components/ui/EmptyState.vue';
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
    <div class="flex flex-wrap items-center gap-2" role="group" aria-label="Status">
      <button v-for="option in [{ value: 'OPEN', label: 'Offen' }, { value: 'DONE', label: 'Erledigt' },
        { value: 'ALL', label: 'Alle' }]" :key="option.value" type="button" class="chip"
        :aria-pressed="status === option.value" @click="status = option.value as typeof status">
        {{ option.label }}
      </button>
    </div>
    <p v-if="error" role="alert" class="note note-danger">{{ error }}</p>
    <div class="space-y-3">
    <EmptyState v-if="!reports.length" text="Keine Meldungen." />
    <article v-for="report in reports" :key="report.id" class="card p-4">
      <div class="flex items-start justify-between gap-2"><h2 class="text-base font-semibold">{{ topics[report.topic] ?? report.topic }}</h2>
        <span class="badge shrink-0 whitespace-nowrap" :class="report.status === 'OPEN' ? 'badge-warning' : 'badge-success'">
          {{ report.status === 'OPEN' ? 'Offen' : '✓ Erledigt' }}</span></div>
      <p class="mt-2 whitespace-pre-wrap">{{ report.message }}</p>
      <p class="mt-2 text-sm text-muted-foreground">{{ formatDate(report.createdAt) }}<template
        v-if="report.parentName"> · von {{ report.parentName }}</template></p>
      <p v-if="report.response" class="mt-2 rounded-lg bg-muted p-3 text-sm">
        <span class="font-semibold">Antwort:</span> {{ report.response }}</p>
      <label data-write v-if="report.status === 'OPEN'" class="mt-3 block text-sm font-medium">
        Antwort an die Eltern (optional)
        <textarea v-model="answers[report.id]" rows="2" maxlength="2000"
          class="input mt-1" /></label>
      <div class="flex mt-3 flex-wrap items-center gap-4 text-sm"><RouterLink :to="link(report)" class="text-primary underline">
        Bezug ansehen</RouterLink>
        <button data-write v-if="report.status === 'OPEN'" class="btn btn-primary btn-sm"
          @click="resolve(report.id)">{{ answers[report.id]?.trim() ? 'Antworten und erledigen' : 'Erledigt' }}</button></div>
    </article>
    </div>
  </div>
</template>
