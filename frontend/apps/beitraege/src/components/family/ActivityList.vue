<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { RouterLink } from 'vue-router';
import { api } from '@/api';
import type { ParentActivity } from '@/api/types';
import { activitySeenAt, markActivitySeen } from '@/composables/useActivitySeen';
const props = withDefaults(defineProps<{ limit?: number }>(), { limit: 100 });
const activity = ref<ParentActivity[]>([]);
const error = ref('');
const loading = ref(true);
// Captured before marking the page as seen, so this visit still highlights what was new.
const seenAt = activitySeenAt();
// Keys are the database column names stored in fees.data_changes.field.
const fields: Record<string, string> = {
  phone: 'Telefon', email: 'E-Mail', street: 'Straße', street_no: 'Hausnummer',
  postal_code: 'PLZ', city: 'Ort', first_name: 'Vorname', last_name: 'Nachname',
  birth_date: 'Geburtsdatum',
};
const workStatus: Record<string, string> = {
  APPROVED: 'bestätigt', REJECTED: 'abgelehnt', VOIDED: 'zurückgezogen',
};
onMounted(async () => {
  try {
    activity.value = await api.getParentActivity(props.limit);
    markActivitySeen();
  } catch { error.value = 'Aktivitäten konnten nicht geladen werden.'; }
  finally { loading.value = false; }
});
function isNew(item: ParentActivity) {
  return new Date(item.at).getTime() > seenAt;
}
function text(item: ParentActivity) {
  const who = item.parentName ?? 'Ein Elternteil';
  if (item.type === 'CONTACT_CHANGED' || item.type === 'CHILD_CHANGED') {
    const whose = item.childName ?? item.targetName;
    return `${who} hat ${whose ? `bei ${whose} ` : ''}${fields[item.field ?? ''] ?? item.field} geändert: `
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
function done(item: ParentActivity) {
  return item.type === 'REPORT_CREATED' ? item.status === 'DONE'
    : item.type === 'PARENT_WORK_SUBMITTED' && item.status !== 'SUBMITTED';
}
function outcome(item: ParentActivity) {
  if (item.type === 'REPORT_CREATED') return item.status === 'DONE' ? 'erledigt' : 'offen';
  if (item.type === 'PARENT_WORK_SUBMITTED') return workStatus[item.status ?? ''] ?? 'wartet auf Freigabe';
  return '';
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
    <p v-if="error" class="text-sm text-muted-foreground">{{ error }}</p>
    <p v-else-if="loading" class="text-sm text-muted-foreground">Änderungen werden geladen …</p>
    <p v-else-if="!activity.length" class="text-sm text-muted-foreground">Noch keine Aktivitäten.</p>
    <ul class="space-y-3"><li v-for="(item, index) in activity" :key="index"
      class="flex gap-3 border-t pt-3 text-sm first:border-t-0 first:pt-0">
      <span aria-hidden="true" :class="done(item) ? 'text-green-700 dark:text-green-400' : ''">{{
        done(item) ? '✓' : item.type === 'REPORT_CREATED' ? '⚑' :
          item.type === 'PARENT_WORK_SUBMITTED' ? '◷' : '✎' }}</span>
      <div><RouterLink :to="link(item)" class="underline"
        :class="done(item) ? 'text-muted-foreground' : 'text-primary'">{{ text(item) }}</RouterLink>
        <p class="text-xs text-muted-foreground"><span v-if="isNew(item)"
          class="mr-1.5 rounded-full bg-primary px-1.5 py-0.5 font-bold text-primary-foreground">neu</span>{{
            relative(item.at) }}<template v-if="outcome(item)">
          · {{ outcome(item) }}</template></p></div>
    </li></ul>
  </section>
</template>
