<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { RouterLink } from 'vue-router';
import { AlertTriangle, Calendar, ChevronLeft, ChevronRight, Loader2 } from 'lucide-vue-next';
import { api } from '@/api';
import type { JugendamtChild, JugendamtReport } from '@/api/types';
import { formatDate } from '@/utils/format';
import PageHeader from '@/components/ui/PageHeader.vue';

type Group = JugendamtChild['group'];
type Filter = 'ALL' | 'U3' | 'UE3' | 'LAST' | 'DEFERRED' | 'SECOND' | 'OTHER';

const FILTER_GROUPS: Record<Filter, Group[] | null> = {
  ALL: null,
  U3: ['U3'],
  UE3: ['OTHER_UE3', 'SECOND_TO_LAST_YEAR', 'LAST_YEAR', 'DEFERRED'],
  LAST: ['LAST_YEAR', 'DEFERRED'],
  DEFERRED: ['DEFERRED'],
  SECOND: ['SECOND_TO_LAST_YEAR'],
  OTHER: ['OTHER_UE3'],
};

const GROUP_LABELS: Record<Group, string> = {
  U3: 'U3',
  OTHER_UE3: 'Ü3',
  SECOND_TO_LAST_YEAR: 'Vorletztes Jahr',
  LAST_YEAR: 'Letztes Jahr',
  DEFERRED: 'Zurückgestellt',
};

const report = ref<JugendamtReport | null>(null);
const loading = ref(true);
const error = ref<string | null>(null);
const filter = ref<Filter>('ALL');

async function load(date?: string) {
  loading.value = true;
  error.value = null;
  try {
    report.value = await api.getJugendamtReport(date);
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Stichtagsmeldung konnte nicht geladen werden';
  } finally {
    loading.value = false;
  }
}

/** Stichtage are the 1st of March, June, September and December. */
function shiftStichtag(date: string, months: number) {
  const [year, month] = date.split('-').map(Number);
  const index = year * 12 + month - 1 + months;
  return `${Math.floor(index / 12)}-${String((index % 12) + 1).padStart(2, '0')}-01`;
}

function step(months: number) {
  if (report.value) load(shiftStichtag(report.value.stichtag, months));
}

const daysLabel = computed(() => {
  const days = report.value?.daysUntil ?? 0;
  if (days === 0) return 'heute';
  const n = Math.abs(days);
  return `${days > 0 ? 'in' : 'vor'} ${n} ${n === 1 ? 'Tag' : 'Tagen'}`;
});

const sections = computed(() => {
  const r = report.value;
  if (!r) return [];
  return [
    { title: 'Belegung', rows: [
      { key: 'ALL' as Filter, label: 'Kinder gesamt', value: r.total },
      { key: 'U3' as Filter, label: 'U3 (Krippe)', value: r.u3 },
      { key: 'UE3' as Filter, label: 'Ü3 (Kindergarten)', value: r.ue3 },
    ] },
    { title: '§ 17a KitaG', rows: [
      { key: 'OTHER' as Filter, label: 'Übrige Ü3', value: r.otherUe3 },
      { key: 'SECOND' as Filter, label: 'Vorletztes Kita-Jahr', value: r.secondToLastYear },
      { key: 'LAST' as Filter, label: 'Letztes Kita-Jahr inkl. Rücksteller', value: r.lastYear },
      { key: 'DEFERRED' as Filter, label: 'Zurückgestellt', value: r.deferred },
    ] },
  ];
});

const children = computed(() => {
  const groups = FILTER_GROUPS[filter.value];
  const all = report.value?.children ?? [];
  return groups ? all.filter(child => groups.includes(child.group)) : all;
});

const filterLabel = computed(() =>
  sections.value.flatMap(s => s.rows).find(row => row.key === filter.value)?.label ?? 'Kinder');

onMounted(() => load());
</script>

<template>
  <div>
    <PageHeader title="Dashboard" />

    <div v-if="error" class="note note-danger mb-4">
      <AlertTriangle />
      <div class="flex-1">{{ error }}</div>
      <button class="btn btn-secondary btn-sm" @click="load(report?.stichtag)">Erneut versuchen</button>
    </div>

    <div v-if="!report && loading" class="flex justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <div v-if="report" class="space-y-6">
      <section class="card p-5" aria-label="Stichtagsmeldung">
        <div class="flex items-start gap-3">
          <div class="rounded-lg bg-success-soft p-2.5">
            <Calendar class="h-5 w-5 text-success" />
          </div>
          <div class="min-w-0 flex-1">
            <p class="text-xs font-medium text-muted-foreground">Stichtagsmeldung Landkreis</p>
            <div class="flex flex-wrap items-center gap-2">
              <p class="text-lg font-semibold" data-testid="stichtag">{{ formatDate(report.stichtag) }}</p>
              <span class="badge">{{ daysLabel }}</span>
            </div>
          </div>
          <div class="flex gap-1">
            <button class="btn btn-ghost btn-icon btn-sm" aria-label="Vorheriger Stichtag" :disabled="loading"
              @click="step(-3)"><ChevronLeft /></button>
            <button class="btn btn-ghost btn-icon btn-sm" aria-label="Nächster Stichtag" :disabled="loading"
              @click="step(3)"><ChevronRight /></button>
          </div>
        </div>

        <div class="mt-4 grid gap-x-8 gap-y-4 border-t pt-4 sm:grid-cols-2">
          <div v-for="section in sections" :key="section.title">
            <p class="mb-1 text-xs font-medium text-muted-foreground">{{ section.title }}</p>
            <button v-for="row in section.rows" :key="row.key" type="button" :aria-pressed="filter === row.key"
              :class="['-mx-2 flex w-[calc(100%+1rem)] items-center justify-between rounded-md px-2 py-1.5 text-sm',
                filter === row.key ? 'bg-primary-soft text-primary-text' : 'hover:bg-accent']"
              @click="filter = row.key">
              <span>{{ row.label }}</span>
              <span class="font-semibold tabular-nums">{{ row.value }}</span>
            </button>
          </div>
        </div>
        <p class="mt-3 text-[13px] text-muted-foreground">
          Zurückgestellt: nach Alter schon schulpflichtig (6 Jahre bis 30.09.), aber noch in der Kita.
        </p>
      </section>

      <section class="card overflow-hidden" aria-label="Kinder zum Stichtag">
        <h2 class="flex items-center gap-2 px-4 pb-2 pt-4 text-base font-semibold">
          {{ filterLabel }} <span class="badge">{{ children.length }}</span>
        </h2>
        <ul class="divide-y border-t sm:hidden">
          <li v-for="child in children" :key="child.id">
            <RouterLink :to="{ name: 'child-detail', params: { id: child.id } }"
              class="flex items-center justify-between gap-3 px-4 py-3 hover:bg-accent">
              <div class="min-w-0">
                <p class="truncate font-medium">{{ child.lastName }}, {{ child.firstName }}</p>
                <p class="text-[13px] text-muted-foreground">{{ formatDate(child.birthDate) }}</p>
              </div>
              <span :class="['badge', child.group === 'DEFERRED' && 'badge-warning']">
                {{ GROUP_LABELS[child.group] }}</span>
            </RouterLink>
          </li>
        </ul>
        <div class="hidden sm:block">
          <table class="table">
            <thead><tr><th>Name</th><th>Geburtsdatum</th><th>Gruppe</th></tr></thead>
            <tbody>
              <tr v-for="child in children" :key="child.id">
                <td><RouterLink :to="{ name: 'child-detail', params: { id: child.id } }" class="font-medium hover:underline">
                  {{ child.lastName }}, {{ child.firstName }}</RouterLink></td>
                <td>{{ formatDate(child.birthDate) }}</td>
                <td><span :class="['badge', child.group === 'DEFERRED' && 'badge-warning']">
                  {{ GROUP_LABELS[child.group] }}</span></td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-if="!children.length" class="px-4 pb-4 text-sm text-muted-foreground">Keine Kinder.</p>
      </section>
    </div>
  </div>
</template>
