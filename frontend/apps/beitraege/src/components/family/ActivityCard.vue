<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { RouterLink } from 'vue-router';
import { api } from '@/api';
import type { ParentActivity } from '@/api/types';
const activity = ref<ParentActivity[]>([]);
const error = ref('');
const fields: Record<string, string> = {
  phone: 'Telefon', email: 'E-Mail', street: 'Straße', streetNo: 'Hausnummer',
  postalCode: 'PLZ', city: 'Ort', firstName: 'Vorname', lastName: 'Nachname',
  birthDate: 'Geburtsdatum',
};
onMounted(async () => {
  try { activity.value = await api.getParentActivity(); }
  catch { error.value = 'Aktivitäten konnten nicht geladen werden.'; }
});
function text(item: ParentActivity) {
  const who = item.parentName ?? 'Ein Elternteil';
  if (item.type === 'CONTACT_CHANGED' || item.type === 'CHILD_CHANGED') {
    return `${who} hat ${fields[item.field ?? ''] ?? item.field} geändert: `
      + `${item.oldValue || '—'} → ${item.newValue || '—'}`;
  }
  if (item.type === 'PARENT_WORK_SUBMITTED') {
    return `${who} hat ${((item.durationMinutes ?? 0) / 60).toLocaleString('de-DE')} h gemeldet`
      + ` (${item.occasion ?? 'Elternstunden'})`;
  }
  return `${who} hat einen Fehler gemeldet: ${item.message ?? ''}`;
}
function link(item: ParentActivity) {
  if (item.childId) return `/kinder/${item.childId}`;
  if (item.type === 'PARENT_WORK_SUBMITTED' && item.householdId) {
    return `/elternstunden/familien/${item.householdId}`;
  }
  if (item.type === 'REPORT_CREATED') return '/meldungen';
  return `/eltern/${item.parentId}`;
}
function relative(date: string) {
  const minutes = Math.max(0, Math.floor((Date.now() - new Date(date).getTime()) / 60000));
  if (minutes < 1) return 'gerade eben';
  if (minutes < 60) return `vor ${minutes} Min.`;
  if (minutes < 1440) return `vor ${Math.floor(minutes / 60)} Std.`;
  const days = Math.floor(minutes / 1440);
  return days === 1 ? 'vor 1 Tag' : `vor ${days} Tagen`;
}
</script>
<template>
  <section class="rounded-2xl border bg-card p-5">
    <h2 class="text-lg font-bold">Letzte Änderungen von Eltern</h2>
    <p v-if="error" class="mt-3 text-sm text-muted-foreground">{{ error }}</p>
    <p v-else-if="!activity.length" class="mt-3 text-sm text-muted-foreground">Noch keine Aktivitäten.</p>
    <ul class="mt-3 space-y-3"><li v-for="(item, index) in activity" :key="index"
      class="flex gap-3 border-t pt-3 text-sm">
      <span aria-hidden="true">{{ item.type === 'REPORT_CREATED' ? '⚑' :
        item.type === 'PARENT_WORK_SUBMITTED' ? '◷' : '✎' }}</span>
      <div><RouterLink :to="link(item)" class="text-primary underline">{{ text(item) }}</RouterLink>
        <p class="text-xs text-muted-foreground">{{ relative(item.at) }}</p></div>
    </li></ul>
  </section>
</template>
