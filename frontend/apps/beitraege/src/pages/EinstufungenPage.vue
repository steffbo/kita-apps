<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue';
import { useRouter } from 'vue-router';
import { api } from '@/api';
import type { Einstufung, EinstufungListStatus, MissingEinstufung } from '@/api/types';
import {
  Plus,
  Loader2,
  Calendar,
  FileText,
  ChevronLeft,
  ChevronRight,
  Trash2,
  User,
  Home,
  Search,
  AlertTriangle,
} from 'lucide-vue-next';
import { formatCurrency, formatDate, formatMonthYear } from '@/utils/format';

const router = useRouter();

const einstufungen = ref<Einstufung[]>([]);
const total = ref(0);
const isLoading = ref(true);
const error = ref<string | null>(null);

const selectedYear = ref(new Date().getFullYear());
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
const currentPage = ref(1);
const pageSize = ref(25);

const totalPages = computed(() => Math.ceil(total.value / pageSize.value));

const years = computed(() => {
  const current = new Date().getFullYear();
  return [current - 2, current - 1, current, current + 1];
});

let loadSeq = 0;
async function loadEinstufungen() {
  const seq = ++loadSeq;
  isLoading.value = true;
  error.value = null;
  try {
    const response = await api.getEinstufungen({
      year: selectedYear.value,
      search: search.value,
      status: status.value,
      page: currentPage.value,
      perPage: pageSize.value,
    });
    if (seq !== loadSeq) return; // a newer request superseded this one
    einstufungen.value = response.data;
    total.value = response.total;

    // If the current page ran empty (e.g. after a delete), fall back to the last valid page
    if (response.data.length === 0 && currentPage.value > 1 && response.total > 0) {
      currentPage.value = Math.max(1, Math.ceil(response.total / pageSize.value));
      return;
    }
  } catch (e) {
    if (seq !== loadSeq) return;
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoading.value = false;
  }
}

function formatCareType(type: string): string {
  return type === 'krippe' ? 'Krippe' : 'Kindergarten';
}

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
    return 'bg-green-100 dark:bg-green-950/40 text-green-800 dark:text-green-300';
  if (rule.includes('Entlastung'))
    return 'bg-blue-100 dark:bg-blue-950/40 text-blue-800 dark:text-blue-300';
  if (rule.includes('Höchstsatz') || rule.includes('Satzung'))
    return 'bg-orange-100 dark:bg-orange-950/40 text-orange-800 dark:text-orange-300';
  if (rule.includes('Pflegefamilie'))
    return 'bg-purple-100 dark:bg-purple-950/40 text-purple-800 dark:text-purple-300';
  return 'bg-muted text-foreground';
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

let searchTimer: ReturnType<typeof setTimeout> | undefined;
watch(search, () => {
  clearTimeout(searchTimer);
  searchTimer = setTimeout(reloadFromFirstPage, 300);
});
watch(status, reloadFromFirstPage);

function reloadFromFirstPage() {
  if (currentPage.value !== 1) {
    currentPage.value = 1; // pagination watcher performs the reload
  } else {
    loadEinstufungen();
  }
}

watch([selectedYear], () => {
  if (currentPage.value !== 1) {
    currentPage.value = 1; // pagination watcher performs the reload
  } else {
    loadEinstufungen();
  }
});

watch([currentPage, pageSize], () => {
  loadEinstufungen();
});

onMounted(() => {
  loadEinstufungen();
  loadMissing();
});
</script>

<template>
  <div>
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-foreground">Einstufungen</h1>
        <p class="text-sm text-muted-foreground mt-1">
          Beitragseinstufungen für {{ selectedYear }}
          <span v-if="total > 0">({{ total }} gesamt)</span>
        </p>
      </div>
      <button
        @click="router.push('/einstufungen/neu')"
        class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors"
      >
        <Plus class="h-4 w-4" />
        Neue Einstufung
      </button>
    </div>

    <!-- Work list: children without Einstufung -->
    <section
      v-if="missing.length"
      aria-labelledby="missing-heading"
      class="mb-6 rounded-lg border border-amber-200 bg-amber-50 dark:bg-amber-950/30 p-4"
    >
      <h2 id="missing-heading" class="flex items-center gap-2 text-sm font-semibold text-amber-900 dark:text-amber-200">
        <AlertTriangle class="h-4 w-4" />
        Einstufung fehlt ({{ missing.length }})
      </h2>
      <p class="mt-1 text-xs text-amber-900/80 dark:text-amber-200/80">
        Krippenkinder ohne Einstufung für den genannten Monat; ihr Platzgeld wird bis dahin aus den Haushaltsdaten berechnet.
      </p>
      <ul class="mt-3 divide-y divide-amber-200/70">
        <li v-for="m in missing" :key="m.childId" class="flex flex-wrap items-center justify-between gap-2 py-2 text-sm">
          <div>
            <RouterLink :to="`/kinder/${m.childId}`" class="font-medium text-foreground hover:text-primary hover:underline">
              {{ m.firstName }} {{ m.lastName }}
            </RouterLink>
            <span class="text-muted-foreground"> · {{ m.householdName || 'kein Haushalt' }}</span>
            <span class="block text-xs text-muted-foreground">
              ab {{ formatMonthYear(Number(m.month.slice(0, 4)), Number(m.month.slice(5, 7))) }}
              <template v-if="m.previousUntil"> · letzte Einstufung bis {{ formatDate(m.previousUntil) }}</template>
              <template v-else-if="m.entryDate > m.month"> · Eintritt {{ formatDate(m.entryDate) }}</template>
            </span>
          </div>
          <RouterLink
            :to="{ path: '/einstufungen/neu', query: { childId: m.childId } }"
            class="inline-flex items-center gap-1 rounded-lg border bg-card px-3 py-1.5 text-sm font-medium hover:bg-accent"
          >
            <Plus class="h-4 w-4" />
            Einstufung anlegen
          </RouterLink>
        </li>
      </ul>
    </section>

    <!-- Filters -->
    <div class="mb-4 flex flex-wrap items-center gap-3">
      <Calendar class="h-4 w-4 text-muted-foreground" />
      <select
        v-model="selectedYear"
        aria-label="Jahr"
        class="border rounded-lg px-3 py-1.5 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary"
      >
        <option v-for="y in years" :key="y" :value="y">{{ y }}</option>
      </select>
      <div class="relative">
        <Search class="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <input
          v-model="search"
          type="search"
          aria-label="Einstufungen durchsuchen"
          placeholder="Kind, Mitgliedsnr. oder Haushalt"
          class="w-64 border rounded-lg pl-8 pr-3 py-1.5 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary"
        />
      </div>
      <select
        v-model="status"
        aria-label="Status"
        class="border rounded-lg px-3 py-1.5 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary"
      >
        <option v-for="o in statusOptions" :key="o.value" :value="o.value">{{ o.label }}</option>
      </select>
    </div>

    <!-- Loading -->
    <div v-if="isLoading" class="flex items-center justify-center py-20">
      <Loader2 class="h-8 w-8 text-primary animate-spin" />
    </div>

    <!-- Error -->
    <div v-else-if="error" class="bg-red-50 dark:bg-red-950/40 text-red-700 dark:text-red-300 p-4 rounded-lg">
      {{ error }}
    </div>

    <!-- Empty state -->
    <div v-else-if="einstufungen.length === 0" class="text-center py-20">
      <FileText class="h-12 w-12 text-muted-foreground/60 mx-auto mb-4" />
      <h3 class="text-lg font-medium text-foreground mb-1">Keine Einstufungen</h3>
      <p v-if="isFiltered" class="text-muted-foreground mb-4">Keine Einstufungen für {{ selectedYear }} passen zum Filter.</p>
      <p v-else class="text-muted-foreground mb-4">Für {{ selectedYear }} wurden noch keine Einstufungen erstellt.</p>
      <button
        v-if="!isFiltered"
        @click="router.push('/einstufungen/neu')"
        class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90"
      >
        <Plus class="h-4 w-4" />
        Erste Einstufung erstellen
      </button>
    </div>

    <!-- Table -->
    <div v-else class="bg-card rounded-lg border shadow-sm overflow-x-auto">
      <table class="min-w-full divide-y divide-border">
        <thead class="bg-muted">
          <tr>
            <th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">Kind</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">Bereich</th>
            <th class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider">Einkommen</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">Regel</th>
            <th class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider">Platzgeld</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">Gültig ab</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">Versendet</th>
            <th class="px-4 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr
            v-for="e in einstufungen"
            :key="e.id"
            class="hover:bg-accent cursor-pointer transition-colors"
            @click="router.push(`/einstufungen/${e.id}`)"
          >
            <td class="px-4 py-3 whitespace-nowrap">
              <div class="flex items-start gap-2">
                <User class="mt-0.5 h-4 w-4 text-muted-foreground" />
                <div>
                  <RouterLink
                    :to="`/kinder/${e.childId}`"
                    class="text-sm font-medium text-foreground hover:text-primary hover:underline"
                    @click.stop
                  >
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
              <span class="text-sm text-muted-foreground">{{ formatCareType(e.careType) }} · {{ e.careHoursPerWeek }}h</span>
            </td>
            <td class="px-4 py-3 whitespace-nowrap text-right">
              <span v-if="e.incomeBasis === 'HIGHEST_RATE'" class="text-sm text-muted-foreground italic">Höchstsatz</span>
              <span v-else-if="e.incomeBasis === 'NOT_REQUIRED'" class="text-sm text-muted-foreground italic">nicht erforderlich</span>
              <span v-else class="text-sm text-foreground">{{ formatCurrency(e.annualNetIncome ?? 0) }}</span>
              <span
                v-if="e.provisional"
                class="ml-2 inline-flex items-center px-1.5 py-0.5 rounded-full text-xs font-medium bg-amber-100 text-amber-800 dark:bg-amber-950/60 dark:text-amber-300"
              >vorläufig</span>
            </td>
            <td class="px-4 py-3">
              <span
                class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium"
                :class="getRuleBadgeClass(e.feeRule)"
              >
                {{ e.feeRule }}
              </span>
            </td>
            <td class="px-4 py-3 whitespace-nowrap text-right">
              <span class="text-sm font-semibold text-foreground">{{ formatCurrency(e.monthlyChildcareFee) }}</span>
              <span v-if="e.discountPercent > 0" class="text-xs text-green-600 dark:text-green-300 ml-1">-{{ e.discountPercent }}%</span>
            </td>
            <td class="px-4 py-3 whitespace-nowrap">
              <span class="text-sm text-muted-foreground">{{ formatDate(e.validFrom) }}</span>
            </td>
            <td class="px-4 py-3 whitespace-nowrap">
              <span class="text-sm text-muted-foreground">{{ formatDate(e.letterSentAt) }}</span>
            </td>
            <td class="px-4 py-3 whitespace-nowrap text-right">
              <button
                @click.stop="confirmDelete(e)"
                class="p-1 text-muted-foreground hover:text-red-500 dark:text-red-400 transition-colors"
                title="Löschen"
              >
                <Trash2 class="h-4 w-4" />
              </button>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="flex items-center justify-between px-4 py-3 border-t bg-muted">
        <span class="text-sm text-muted-foreground">
          Seite {{ currentPage }} von {{ totalPages }}
        </span>
        <div class="flex items-center gap-2">
          <button
            :disabled="currentPage <= 1"
            @click="currentPage--"
            class="p-1 rounded hover:bg-accent disabled:opacity-30 disabled:cursor-not-allowed"
          >
            <ChevronLeft class="h-5 w-5" />
          </button>
          <button
            :disabled="currentPage >= totalPages"
            @click="currentPage++"
            class="p-1 rounded hover:bg-accent disabled:opacity-30 disabled:cursor-not-allowed"
          >
            <ChevronRight class="h-5 w-5" />
          </button>
        </div>
      </div>
    </div>

    <!-- Delete dialog -->
    <Teleport to="body">
      <div
        v-if="showDeleteDialog"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
        @click.self="showDeleteDialog = false"
      >
        <div class="bg-card rounded-lg shadow-xl max-w-md w-full mx-4 p-6">
          <h3 class="text-lg font-semibold text-foreground mb-2">Einstufung löschen</h3>
          <p class="text-sm text-muted-foreground mb-4">
            Möchtest du die Einstufung für
            <strong>{{ deleteTarget?.child ? `${deleteTarget.child.firstName} ${deleteTarget.child.lastName}` : '' }}</strong>
            ({{ deleteTarget?.year }}) wirklich löschen?
          </p>
          <div class="flex justify-end gap-2">
            <button
              @click="showDeleteDialog = false"
              class="px-4 py-2 text-sm text-foreground bg-muted rounded-lg hover:bg-accent"
            >
              Abbrechen
            </button>
            <button
              @click="handleDelete"
              :disabled="isDeleting"
              class="px-4 py-2 text-sm text-white bg-red-600 rounded-lg hover:bg-red-700 disabled:opacity-50 inline-flex items-center gap-2"
            >
              <Loader2 v-if="isDeleting" class="h-4 w-4 animate-spin" />
              Löschen
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
