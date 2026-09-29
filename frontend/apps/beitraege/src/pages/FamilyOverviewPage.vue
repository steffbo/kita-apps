<script setup lang="ts">
import { onMounted, ref, computed } from 'vue';
import { RouterLink } from 'vue-router';
import { api } from '@/api';
import type { OwnOverview, OwnFees, OwnWork, ParentReport } from '@/api/types';
import { formatCurrency, formatDate, formatHours, todayISO } from '@/utils/format';
import ReportDialog from '@/components/family/ReportDialog.vue';
import ReportList from '@/components/family/ReportList.vue';
const overview = ref<OwnOverview | null>(null);
const fees = ref<OwnFees | null>(null);
const work = ref<OwnWork | null>(null);
const reports = ref<ParentReport[]>([]);
const error = ref('');
const showReport = ref(false);
const berlinDate = todayISO();
const year = Number(berlinDate.slice(0, 4));
const kitaYear = Number(berlinDate.slice(5, 7)) >= 8 ? year : year - 1;
const overdue = computed(() => fees.value?.items.filter(f => f.status === 'OVERDUE').length ?? 0);
const progress = computed(() => work.value?.requiredMinutes
  ? Math.min(100, 100 * work.value.doneMinutes / work.value.requiredMinutes) : 0);
onMounted(async () => {
  try { overview.value = await api.getOwnOverview(); }
  catch (e) { error.value = e instanceof Error ? e.message : 'Familie konnte nicht geladen werden'; return; }
  const [f, w] = await Promise.allSettled([api.getOwnFees(year), api.getOwnWork(kitaYear),
    loadReports()]);
  if (f.status === 'fulfilled') fees.value = f.value;
  if (w.status === 'fulfilled') work.value = w.value;
});
async function loadReports() { reports.value = await api.getOwnReports(); }
// Open reports plus answers from the last 30 days; older finished ones stay on "Meine Daten".
const recentReports = computed(() => reports.value.filter(r => r.status === 'OPEN'
  || Date.now() - new Date(r.resolvedAt ?? r.createdAt).getTime() < 30 * 86400000).slice(0, 5));
</script>
<template>
  <div class="space-y-5">
    <h1 class="text-2xl font-bold">Hallo {{ overview?.parent.firstName || 'zusammen' }}!</h1>
    <p v-if="error" role="alert" class="rounded-xl border border-amber-300 bg-amber-50 p-4
      text-amber-900 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-300">
      {{ error.includes('zugeordnet') ? 'Dein Konto ist noch keinem Elternteil zugeordnet. Bitte wende dich an den Vorstand.' : error }}
    </p>
    <template v-if="overview">
      <section><h2 class="mb-3 text-lg font-bold">Deine Kinder</h2>
        <div class="grid gap-4 sm:grid-cols-2">
          <article v-for="child in overview.children" :key="child.id" class="rounded-2xl border bg-card p-5">
            <h3 class="text-lg font-bold">{{ child.firstName }} {{ child.lastName }}</h3>
            <p class="mt-2 text-sm">Geboren: {{ formatDate(child.birthDate) }}</p>
            <p class="text-sm">Betreuungszeit: {{ child.careHours != null ? `${child.careHours} h/Woche` : 'nicht hinterlegt' }}</p>
            <p class="text-sm">Rechtsanspruch: {{ child.legalHours != null ? `${child.legalHours} h/Woche` : 'nicht hinterlegt' }}</p>
          </article>
        </div>
      </section>
      <div class="grid gap-4 sm:grid-cols-2">
        <section class="rounded-2xl border bg-card p-5">
          <h2 class="text-lg font-bold">Beiträge</h2>
          <p class="mt-2 text-2xl font-bold">{{ formatCurrency(fees?.openTotal ?? 0) }} offen</p>
          <p class="text-sm text-muted-foreground">{{ overdue }} überfällig</p>
          <RouterLink to="/familie/beitraege" class="mt-3 inline-block text-primary underline">Beiträge ansehen</RouterLink>
        </section>
        <section class="rounded-2xl border bg-card p-5">
          <h2 class="text-lg font-bold">Elternstunden {{ kitaYear }}/{{ String(kitaYear + 1).slice(-2) }}</h2>
          <p v-if="work?.exempt" class="mt-2 font-semibold">Befreit</p>
          <template v-else-if="work">
            <p class="mt-2">{{ formatHours(work.doneMinutes) }} von {{ formatHours(work.requiredMinutes) }}</p>
            <div role="progressbar" :aria-valuenow="progress" aria-valuemin="0" aria-valuemax="100"
              class="mt-2 h-3 overflow-hidden rounded-full bg-muted">
              <div class="h-full bg-primary" :style="{ width: `${progress}%` }" />
            </div>
            <p class="mt-2 text-sm">Noch offen: {{ formatHours(work.openMinutes) }}</p>
          </template>
          <RouterLink to="/familie/elternstunden" class="mt-3 inline-block rounded-lg bg-primary px-4 py-2
            text-primary-foreground">Stunden melden</RouterLink>
        </section>
      </div>
      <section class="rounded-2xl border bg-card p-5">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <h2 class="text-lg font-bold">Meldungen deiner Familie</h2>
          <button class="rounded-lg border px-4 py-2 text-primary hover:bg-accent"
            @click="showReport = true">Fehler melden</button>
        </div>
        <p v-if="!recentReports.length" class="mt-3 text-sm text-muted-foreground">
          Keine offenen Meldungen. Stimmt etwas nicht? Dann melde es uns.</p>
        <ReportList v-else class="mt-3" :reports="recentReports" />
        <RouterLink v-if="reports.length > recentReports.length" to="/familie/daten"
          class="mt-3 inline-block text-sm text-primary underline">Alle Meldungen</RouterLink>
      </section>
    </template>
    <ReportDialog v-if="showReport" @close="showReport = false"
      @saved="showReport = false; loadReports()" />
  </div>
</template>
