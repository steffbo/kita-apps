<script setup lang="ts">
import PageHeader from '@/components/ui/PageHeader.vue';
import AppSheet from '@/components/ui/AppSheet.vue';
import AppPagination from '@/components/ui/AppPagination.vue';
import EmptyState from '@/components/ui/EmptyState.vue';

import { ref, computed, onMounted, watch } from 'vue';
import { useRouter } from 'vue-router';
import { api } from '@/api';
import type { Einstufung, EinstufungListStatus, MissingEinstufung } from '@/api/types';
import {
  Info,
  Plus,
  Loader2,
  FileText,
  Trash2,
  User,
  Home,
  Search,
  AlertTriangle,
} from 'lucide-vue-next';
import { usePagedList } from '@/composables/usePagedList';
import { formatCareType, formatCurrency, formatDate, formatMonthYear } from '@/utils/format';
import { currentYear, yearOptions } from '@/utils/years';

const router = useRouter();

const selectedYear = ref(currentYear());
const search = ref('');
const status = ref<EinstufungListStatus>('');
const statusOptions: { value: EinstufungListStatus; label: string }[] = [
  { value: '', label: 'Alle' },
  { value: 'current', label: 'Aktuell gültig' },
  { value: 'provisional', label: 'Vorläufig' },
  { value: 'unsent', label: 'Nicht versendet' },
];
const isFiltered = computed(() => !!search.value.trim() || !!status.value);

// U3 children that owe childcare fees without an Einstufung.
const missing = ref<MissingEinstufung[]>([]);
async function loadMissing() {
  try {
    missing.value = await api.getMissingEinstufungen();
  } catch {
    missing.value = [];
  }
}

const {
  items: einstufungen,
  total,
  isLoading,
  error,
  currentPage,
  pageSize,
  totalPages,
  offset,
  load: loadEinstufungen,
  goToPage,
  goToFirstPageAndLoad,
  handleSearchInput,
} = usePagedList<Einstufung>({
  fetchPage: ({ page, perPage }) => api.getEinstufungen({
    year: selectedYear.value,
    search: search.value,
    status: status.value,
    page,
    perPage,
  }),
});

const years = yearOptions(currentYear());

function getChildName(e: Einstufung): string {
  if (e.child) return `${e.child.firstName} ${e.child.lastName}`;
  return '—';
}

function getHouseholdName(e: Einstufung): string {
  if (e.household) return e.household.name;
  return '—';
}

function getRuleBadgeClass(rule: string): string {
  if (rule.includes('beitragsfrei') || rule.includes('Beitragsfrei'))
    return 'badge-success';
  if (rule.includes('Entlastung'))
    return 'badge-primary';
  if (rule.includes('Höchstsatz') || rule.includes('Satzung'))
    return '';
  if (rule.includes('Pflegefamilie'))
    return '';
  return '';
}

// Delete
const showDeleteDialog = ref(false);
const deleteTarget = ref<Einstufung | null>(null);
const isDeleting = ref(false);

function confirmDelete(e: Einstufung) {
  deleteTarget.value = e;
  showDeleteDialog.value = true;
}

async function handleDelete() {
  if (!deleteTarget.value) return;
  isDeleting.value = true;
  try {
    await api.deleteEinstufung(deleteTarget.value.id);
    showDeleteDialog.value = false;
    deleteTarget.value = null;
    loadEinstufungen();
    loadMissing();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Löschen';
  } finally {
    isDeleting.value = false;
  }
}

watch(search, handleSearchInput);
watch([status, selectedYear], goToFirstPageAndLoad);

onMounted(() => {
  loadEinstufungen();
  loadMissing();
});
</script>

<template>
  <div>
    <PageHeader title="Einstufungen">
      <template #actions>
        <button data-write type="button" @click="router.push('/einstufungen/neu')" class="btn btn-primary">
          <Plus class="h-4 w-4" />
          Neue Einstufung
        </button>
      </template>
    </PageHeader>
    <!-- Work list: children without Einstufung -->
    <section
      v-if="missing.length"
      aria-labelledby="missing-heading"
      class="mb-6 rounded-xl border border-warning-line bg-warning-soft p-4">
      <h2 id="missing-heading" class="flex items-center gap-2 text-sm font-semibold text-warning">
        <AlertTriangle class="h-4 w-4" />
        Einstufung fehlt ({{ missing.length }})
      </h2>
      <p class="mt-1 text-xs text-warning">
        Krippenkinder ohne Einstufung für den genannten Monat; ihr Platzgeld wird bis dahin aus den Haushaltsdaten berechnet.
      </p>
      <ul class="mt-3 divide-y divide-warning-line">
        <li
          v-for="m in missing"
          :key="m.childId"
          class="flex flex-wrap items-center justify-between gap-2 py-2 text-sm">
          <div>
            <RouterLink
              :to="`/kinder/${m.childId}`"
              class="font-medium text-foreground hover:text-primary hover:underline">
              {{ m.firstName }} {{ m.lastName }}
            </RouterLink>
            <span class="text-muted-foreground">· {{ m.householdName || 'kein Haushalt' }}</span>
            <span class="block text-xs text-muted-foreground">
              ab {{ formatMonthYear(Number(m.month.slice(0, 4)), Number(m.month.slice(5, 7))) }}
              <template v-if="m.previousUntil">· letzte Einstufung bis {{ formatDate(m.previousUntil) }}</template>
              <template v-else-if="m.entryDate > m.month">· Eintritt {{ formatDate(m.entryDate) }}</template>
            </span>
          </div>
          <RouterLink data-write :to="{ path: '/einstufungen/neu', query: { childId: m.childId } }" class="btn btn-secondary">
            <Plus class="h-4 w-4" />
            Einstufung anlegen
          </RouterLink>
        </li>
      </ul>
    </section>
    <!-- Filters -->
    <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center">
      <div class="relative flex-1">
        <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <input
          v-model="search"
          type="search"
          aria-label="Einstufungen durchsuchen"
          placeholder="Kind, Mitgliedsnr. oder Haushalt"
          class="input pl-10 pr-3" />
      </div>
      <div class="flex items-center gap-2">
        <select v-model="selectedYear" aria-label="Jahr" class="input w-auto">
          <option v-for="y in years" :key="y" :value="y">{{ y }}</option>
        </select>
        <select v-model="status" aria-label="Status" class="input w-auto min-w-0">
          <option v-for="o in statusOptions" :key="o.value" :value="o.value">{{ o.label }}</option>
        </select>
        <span class="ml-auto whitespace-nowrap px-1 text-sm text-muted-foreground sm:ml-0" aria-live="polite">
          {{ total }} gesamt
        </span>
      </div>
    </div>
    <!-- Loading -->
    <div v-if="isLoading" class="flex items-center justify-center py-20">
      <Loader2 class="h-8 w-8 text-primary animate-spin" />
    </div>
    <!-- Error -->
    <div v-else-if="error" class="note note-danger">
      <Info />
      <div class="min-w-0 flex-1">{{ error }}</div>
    </div>
    <!-- Empty state -->
    <EmptyState v-else-if="einstufungen.length === 0" :icon="FileText" text="Keine Einstufungen">
      <p v-if="isFiltered" class="text-muted-foreground mb-4">
        Keine Einstufungen für {{ selectedYear }} passen zum Filter.
      </p>
      <p v-else class="text-muted-foreground mb-4">Für {{ selectedYear }} wurden noch keine Einstufungen erstellt.</p>
      <button data-write type="button" v-if="!isFiltered" @click="router.push('/einstufungen/neu')" class="btn btn-primary">
        <Plus class="h-4 w-4" />
        Erste Einstufung erstellen
      </button>
    </EmptyState>
    <!-- Table -->
    <div v-else class="card overflow-hidden">
      <ul class="divide-y sm:hidden">
        <li v-for="e in einstufungen" :key="e.id" class="space-y-2 px-4 py-3">
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <RouterLink :to="`/kinder/${e.childId}`" class="font-medium hover:underline">
                {{ getChildName(e) }}
              </RouterLink>
              <p class="text-[13px] text-muted-foreground">
                {{ getHouseholdName(e) }} · {{ formatCareType(e.careType) }} · {{ e.careHoursPerWeek }}h
              </p>
            </div>
            <span class="text-right font-semibold tabular-nums">
              {{ formatCurrency(e.monthlyChildcareFee) }}
              <span v-if="e.discountPercent > 0" class="block text-xs text-success">-{{ e.discountPercent }}%</span>
            </span>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <span class="badge" :class="getRuleBadgeClass(e.feeRule)">{{ e.feeRule }}</span>
            <span v-if="e.provisional" class="badge badge-warning">vorläufig</span>
          </div>
          <dl class="space-y-1 text-[13px]">
            <div class="flex justify-between gap-3">
              <dt class="text-muted-foreground">Einkommen</dt>
              <dd class="text-right tabular-nums">
                <span v-if="e.incomeBasis === 'HIGHEST_RATE'">Höchstsatz</span>
                <span v-else-if="e.incomeBasis === 'NOT_REQUIRED'">nicht erforderlich</span>
                <span v-else>{{ formatCurrency(e.annualNetIncome ?? 0) }}</span>
              </dd>
            </div>
            <div class="flex justify-between gap-3">
              <dt class="text-muted-foreground">Gültig ab</dt>
              <dd>{{ formatDate(e.validFrom) }}</dd>
            </div>
            <div class="flex justify-between gap-3">
              <dt class="text-muted-foreground">Versendet</dt>
              <dd>{{ formatDate(e.letterSentAt) }}</dd>
            </div>
          </dl>
          <div class="flex items-center justify-end gap-2">
            <button type="button" class="btn btn-secondary btn-sm" @click="router.push(`/einstufungen/${e.id}`)">
              <FileText />
              Einstufung bearbeiten
            </button>
            <button data-write
              type="button"
              @click.stop="confirmDelete(e)"
              class="btn btn-danger-outline btn-icon btn-sm"
              aria-label="Löschen"
              title="Löschen">
              <Trash2 class="h-4 w-4" />
            </button>
          </div>
        </li>
      </ul>
      <div class="hidden overflow-x-auto sm:block">
        <table class="table">
          <thead class="bg-muted">
            <tr>
              <th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Kind
              </th>
              <th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Bereich
              </th>
              <th
                class="px-4 py-3 !text-right text-xs font-medium text-muted-foreground uppercase tracking-wider
                  tabular-nums">
                Einkommen
              </th>
              <th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Regel
              </th>
              <th
                class="px-4 py-3 !text-right text-xs font-medium text-muted-foreground uppercase tracking-wider
                  tabular-nums">
                Platzgeld
              </th>
              <th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Gültig ab
              </th>
              <th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Versendet
              </th>
              <th
                class="px-4 py-3 !text-right text-xs font-medium text-muted-foreground uppercase tracking-wider
                  tabular-nums">
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr
              v-for="e in einstufungen"
              :key="e.id"
              class="hover:bg-accent cursor-pointer transition-colors"
              @click="router.push(`/einstufungen/${e.id}`)">
              <td class="px-4 py-3 whitespace-nowrap">
                <div class="flex items-start gap-2">
                  <User class="mt-0.5 h-4 w-4 text-muted-foreground" />
                  <div>
                    <RouterLink
                      :to="`/kinder/${e.childId}`"
                      class="text-sm font-medium text-foreground hover:text-primary hover:underline"
                      @click.stop>
                      {{ getChildName(e) }}
                    </RouterLink>
                    <span class="flex items-center gap-1 text-xs text-muted-foreground">
                      <Home class="h-3 w-3" />
                      {{ getHouseholdName(e) }}
                    </span>
                  </div>
                </div>
              </td>
              <td class="px-4 py-3 whitespace-nowrap">
                <span class="text-sm text-muted-foreground">
                  {{ formatCareType(e.careType) }} · {{ e.careHoursPerWeek }}h
                </span>
              </td>
              <td class="px-4 py-3 whitespace-nowrap text-right tabular-nums">
                <span v-if="e.incomeBasis === 'HIGHEST_RATE'" class="text-sm text-muted-foreground italic">
                  Höchstsatz
                </span>
                <span v-else-if="e.incomeBasis === 'NOT_REQUIRED'" class="text-sm text-muted-foreground italic">
                  nicht erforderlich
                </span>
                <span v-else class="text-sm text-foreground">{{ formatCurrency(e.annualNetIncome ?? 0) }}</span>
                <span v-if="e.provisional" class="badge badge-warning ml-2">vorläufig</span>
              </td>
              <td class="px-4 py-3">
                <span class="badge" :class="getRuleBadgeClass(e.feeRule)">{{ e.feeRule }}</span>
              </td>
              <td class="px-4 py-3 whitespace-nowrap text-right tabular-nums">
                <span class="text-sm font-semibold text-foreground">{{ formatCurrency(e.monthlyChildcareFee) }}</span>
                <span v-if="e.discountPercent > 0" class="text-xs text-success ml-1">-{{ e.discountPercent }}%</span>
              </td>
              <td class="px-4 py-3 whitespace-nowrap">
                <span class="text-sm text-muted-foreground">{{ formatDate(e.validFrom) }}</span>
              </td>
              <td class="px-4 py-3 whitespace-nowrap">
                <span class="text-sm text-muted-foreground">{{ formatDate(e.letterSentAt) }}</span>
              </td>
              <td class="px-4 py-3 whitespace-nowrap text-right tabular-nums">
                <button data-write
                  type="button"
                  @click.stop="confirmDelete(e)"
                  class="btn btn-danger-outline btn-icon btn-sm"
                  aria-label="Löschen"
                  title="Löschen">
                  <Trash2 class="h-4 w-4" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <AppPagination
        v-if="total > 0"
        v-model:page-size="pageSize"
        :current-page="currentPage"
        :total-pages="totalPages"
        :total="total"
        :offset="offset"
        @go="goToPage" />
    </div>
    <!-- Delete dialog -->
    <AppSheet v-if="showDeleteDialog" title="Einstufung löschen" @close="showDeleteDialog = false">
      <p class="text-sm text-muted-foreground mb-4">
        Möchtest du die Einstufung für
        <strong>
          {{ deleteTarget?.child ? `${deleteTarget.child.firstName} ${deleteTarget.child.lastName}` : '' }}
        </strong>
        ({{ deleteTarget?.year }}) wirklich löschen?
      </p>
      <template #footer>
        <button type="button" @click="showDeleteDialog = false" class="btn btn-secondary">Abbrechen</button>
        <button type="button" @click="handleDelete" :disabled="isDeleting" class="btn btn-danger">
          <Loader2 v-if="isDeleting" class="h-4 w-4 animate-spin" />
          Löschen
        </button>
      </template>
    </AppSheet>
  </div>
</template>
