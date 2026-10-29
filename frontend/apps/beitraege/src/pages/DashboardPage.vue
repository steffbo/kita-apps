<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue';
import { api } from '@/api';
import ActivitySummaryCard from '@/components/family/ActivitySummaryCard.vue';
import { useAuthStore } from '@/stores/auth';
import type { FeeOverview, StichtagsmeldungReport, StichtagsmeldungStats, U3ChildDetail, BankingSyncStatus } from '@/api/types';
import {
  Receipt,
  CheckCircle,
  AlertTriangle,
  TrendingUp,
  Loader2,
  Link2,
  Utensils,
  Home,
  Baby,
  Calendar,
  Users2,
  CircleDollarSign,
  RefreshCw,
} from 'lucide-vue-next';
import { useRouter } from 'vue-router';
import { formatCurrency, formatCurrencyWhole, formatDate, formatDateTime, formatMonthName, todayISO } from '@/utils/format';
import { formatCareHours } from '@/utils/child';
import PageHeader from '@/components/ui/PageHeader.vue';
import AppSheet from '@/components/ui/AppSheet.vue';

const router = useRouter();
const authStore = useAuthStore();
const overview = ref<FeeOverview | null>(null);
const monthlyOverview = ref<FeeOverview | null>(null);
const unmatchedTotal = ref(0);
const bankingSyncStatus = ref<BankingSyncStatus | null>(null);
const stichtagStats = ref<StichtagsmeldungStats | null>(null);
const u3Count = ref(0);
const totalChildrenCount = ref(0);
const childrenWithWarnings = ref(0);
const isLoading = ref(true);
const error = ref<string | null>(null);

// Year selector only for monthly overview
const selectedYear = ref(new Date().getFullYear());
const isLoadingMonthly = ref(false);
const monthlyError = ref(false);

// Modal state
const showStichtagModal = ref(false);
const selectedReportDate = ref('');
const report = ref<StichtagsmeldungReport | null>(null);
const u3Children = ref<U3ChildDetail[]>([]);
const isLoadingReport = ref(false);
const isLoadingU3Children = ref(false);

const years = computed(() => {
  const currentYear = new Date().getFullYear();
  return [currentYear - 1, currentYear, currentYear + 1];
});

const ue3Count = computed(() => totalChildrenCount.value - u3Count.value);

const bankingSyncLabel = computed(() => {
  switch (bankingSyncStatus.value?.status) {
    case 'running':
      return 'Läuft';
    case 'waiting_for_2fa':
      return 'Wartet auf 2FA';
    case 'success':
      return 'Erfolgreich';
    case 'error':
      return 'Fehlgeschlagen';
    case 'idle':
      return 'Bereit';
    default:
      return 'Unbekannt';
  }
});

const bankingSyncTone = computed(() => {
  switch (bankingSyncStatus.value?.status) {
    case 'running':
      return 'bg-primary-soft text-primary';
    case 'waiting_for_2fa':
      return 'bg-warning-soft text-warning';
    case 'success':
      return 'bg-success-soft text-success';
    case 'error':
      return 'bg-destructive-soft text-destructive';
    default:
      return 'bg-muted text-foreground';
  }
});

const bankingSyncDate = computed(() => {
  const date = bankingSyncStatus.value?.finishedAt || bankingSyncStatus.value?.startedAt;
  if (!date) return null;
  return formatDateTime(date);
});

// Group U3 children by income bracket
type IncomeBracket = 'upTo20k' | 'from20To35k' | 'from35To55k' | 'maxAccepted' | 'fosterFamily';

const bracketLabels: Record<IncomeBracket, string> = {
  upTo20k: '≤20.000€',
  from20To35k: '>20.000 – ≤35.000€',
  from35To55k: '>35.000 – ≤55.000€',
  maxAccepted: 'Höchstbetrag',
  fosterFamily: 'Pflegefamilie',
};

function getChildBracket(child: U3ChildDetail): IncomeBracket {
  if (child.isFosterFamily) return 'fosterFamily';
  if (child.incomeStatus === 'MAX_ACCEPTED') return 'maxAccepted';
  const income = child.householdIncome ?? 0;
  if (income <= 20000) return 'upTo20k';
  if (income <= 35000) return 'from20To35k';
  return 'from35To55k';
}

const groupedU3Children = computed(() => {
  const groups: Record<IncomeBracket, U3ChildDetail[]> = {
    upTo20k: [],
    from20To35k: [],
    from35To55k: [],
    maxAccepted: [],
    fosterFamily: [],
  };
  for (const child of u3Children.value) {
    groups[getChildBracket(child)].push(child);
  }
  return groups;
});

async function loadDashboard() {
  isLoading.value = true;
  error.value = null;
  try {
    overview.value = await api.getFeeOverview();
    // The year overview loads on its own: the banking sync status below can take seconds, and a
    // late combined response must not overwrite a year the user picked in the meantime.
    loadMonthlyOverview();
    const [unmatchedData, stichtagData, u3Data, totalData, warningsData, syncStatusData] = await Promise.allSettled([
      api.getUnmatchedTransactions({ perPage: 1 }),
      api.getStichtagsmeldungStats(),
      api.getChildren({ activeOnly: true, u3Only: true, perPage: 1 }),
      api.getChildren({ activeOnly: true, perPage: 1 }),
      api.getChildren({ activeOnly: true, hasWarnings: true, perPage: 1 }),
      authStore.isAdmin ? api.getBankingSyncStatus() : Promise.resolve(null),
    ] as const);
    if (unmatchedData.status === 'fulfilled') unmatchedTotal.value = unmatchedData.value.total;
    if (stichtagData.status === 'fulfilled') stichtagStats.value = stichtagData.value;
    if (u3Data.status === 'fulfilled') u3Count.value = u3Data.value.total;
    if (totalData.status === 'fulfilled') totalChildrenCount.value = totalData.value.total;
    if (warningsData.status === 'fulfilled') childrenWithWarnings.value = warningsData.value.total;
    if (syncStatusData.status === 'fulfilled') bankingSyncStatus.value = syncStatusData.value;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoading.value = false;
  }
}

let monthlyRequest = 0;

async function loadMonthlyOverview() {
  // Only the response for the most recently selected year may land, whatever order they arrive in.
  const request = ++monthlyRequest;
  const year = selectedYear.value;
  isLoadingMonthly.value = true;
  monthlyError.value = false;
  try {
    const data = await api.getFeeOverview(year);
    if (request === monthlyRequest) monthlyOverview.value = data;
  } catch (e) {
    // Do not keep showing another year's numbers under the new year's label.
    if (request === monthlyRequest) {
      monthlyOverview.value = null;
      monthlyError.value = true;
    }
  } finally {
    if (request === monthlyRequest) isLoadingMonthly.value = false;
  }
}

async function openStichtagModal() {
  showStichtagModal.value = true;
  if (!selectedReportDate.value && stichtagStats.value) {
    selectedReportDate.value = stichtagStats.value.nextStichtag;
  }
  await loadStichtagReport();
}

async function loadStichtagReport() {
  if (!selectedReportDate.value) return;

  isLoadingReport.value = true;
  isLoadingU3Children.value = true;
  try {
    const [reportData, u3Data] = await Promise.all([
      api.getStichtagsmeldungReport(selectedReportDate.value),
      api.getU3Children(selectedReportDate.value),
    ]);
    report.value = reportData;
    u3Children.value = u3Data;
  } catch (e) {
    report.value = null;
    u3Children.value = [];
  } finally {
    isLoadingReport.value = false;
    isLoadingU3Children.value = false;
  }
}

watch(selectedYear, () => {
  loadMonthlyOverview();
});

onMounted(() => {
  loadDashboard();
});

function formatAge(birthDateStr: string): string {
  const birth = new Date(birthDateStr);
  const now = new Date();
  let years = now.getFullYear() - birth.getFullYear();
  let months = now.getMonth() - birth.getMonth();
  if (months < 0) {
    years--;
    months += 12;
  }
  if (now.getDate() < birth.getDate()) {
    months--;
    if (months < 0) {
      years--;
      months += 12;
    }
  }
  return `${years}J ${months}M`;
}

</script>

<template>
  <div>
    <PageHeader title="Dashboard" />

    <ActivitySummaryCard v-if="authStore.isAdmin" class="mb-6" />

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="note note-danger">
      <AlertTriangle />
      <div class="flex-1">{{ error }}</div>
      <button @click="loadDashboard" class="btn btn-secondary btn-sm">
        Erneut versuchen
      </button>
    </div>

    <!-- Content -->
    <div v-else-if="overview" class="space-y-6">
      <!-- Main Stats Card - Open Fees (current state) -->
      <div class="card p-5">
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <!-- Open Food Fees -->
          <div
            class="card p-4 cursor-pointer hover:bg-accent transition-colors"
            @click="router.push('/beitraege?feeType=FOOD&status=open')"
          >
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-muted p-2">
                <Utensils class="h-5 w-5 text-muted-foreground" />
              </div>
              <div class="flex-1 min-w-0">
                <p class="text-[13px] text-muted-foreground truncate">Offene Essensgelder</p>
                <p class="text-2xl font-semibold tabular-nums">{{ overview.openFoodCount }}</p>
              </div>
            </div>
          </div>

          <!-- Open Childcare Fees -->
          <div
            class="card p-4 cursor-pointer hover:bg-accent transition-colors"
            @click="router.push('/beitraege?feeType=CHILDCARE&status=open')"
          >
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-muted p-2">
                <Home class="h-5 w-5 text-muted-foreground" />
              </div>
              <div class="flex-1 min-w-0">
                <p class="text-[13px] text-muted-foreground truncate">Offene Platzgelder</p>
                <p class="text-2xl font-semibold tabular-nums">{{ overview.openChildcareCount }}</p>
              </div>
            </div>
          </div>

          <!-- Open Membership Fees -->
          <div
            class="card p-4 cursor-pointer hover:bg-accent transition-colors"
            @click="router.push('/beitraege?feeType=MEMBERSHIP&status=open')"
          >
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-muted p-2">
                <Users2 class="h-5 w-5 text-muted-foreground" />
              </div>
              <div class="flex-1 min-w-0">
                <p class="text-[13px] text-muted-foreground truncate">Offene Vereinsbeiträge</p>
                <p class="text-2xl font-semibold tabular-nums">{{ overview.openMembershipCount }}</p>
              </div>
            </div>
          </div>
        </div>

        <!-- Summary row -->
        <div class="mt-4 pt-4 border-t flex items-center justify-between">
          <div class="flex items-center gap-2 text-muted-foreground">
            <Receipt class="h-4 w-4" />
            <span class="text-sm">Offener Gesamtbetrag:</span>
          </div>
          <span class="text-lg font-semibold text-foreground">
            {{ formatCurrency(overview.amountOpen + overview.amountOverdue) }}
          </span>
        </div>
      </div>

      <!-- Secondary Cards Row -->
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <!-- Bank Import Status Card -->
        <div v-if="authStore.isAdmin"
          class="card p-5 cursor-pointer hover:bg-muted/50 transition-colors"
          @click="router.push('/bankabgleich')"
        >
          <div class="flex items-start gap-3">
            <div class="p-2.5 bg-muted rounded-lg">
              <RefreshCw class="h-5 w-5 text-muted-foreground" />
            </div>
            <div class="flex-1 min-w-0">
              <div class="flex items-center justify-between gap-2">
                <p class="text-xs text-muted-foreground font-medium">Bank-Import</p>
                <span
                  :class="[
                    'inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium',
                    bankingSyncTone,
                  ]"
                >
                  {{ bankingSyncLabel }}
                </span>
              </div>
              <p class="text-sm font-semibold text-foreground mt-0.5 truncate">
                {{ bankingSyncDate || 'Noch kein Import' }}
              </p>
              <p class="text-xs text-muted-foreground mt-1">Zum Bankabgleich</p>
            </div>
          </div>
        </div>

        <!-- U3/Ü3 Children Card -->
        <div class="card p-5">
          <div class="flex items-center gap-3">
            <div class="p-2.5 bg-primary-soft rounded-lg">
              <Baby class="h-5 w-5 text-primary" />
            </div>
            <div>
              <p class="text-xs text-muted-foreground font-medium">Aktive Kinder</p>
              <p class="text-lg font-semibold text-foreground">
                {{ u3Count }} <span class="text-muted-foreground font-normal">U3</span>
                · {{ ue3Count }} <span class="text-muted-foreground font-normal">Ü3</span>
              </p>
              <p class="text-xs text-muted-foreground">Gesamt: {{ totalChildrenCount }}</p>
            </div>
          </div>
        </div>

        <!-- Stichtagsmeldung Card (clickable) -->
        <div
          v-if="stichtagStats"
          class="card p-5 cursor-pointer hover:bg-muted/50 transition-colors"
          @click="openStichtagModal"
        >
          <div class="flex items-start gap-3">
            <div class="p-2.5 bg-success-soft rounded-lg">
              <Calendar class="h-5 w-5 text-success" />
            </div>
            <div class="flex-1 min-w-0">
              <div class="flex items-center justify-between gap-2">
                <p class="text-xs text-muted-foreground font-medium">Stichtagsmeldung</p>
                <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-success-soft text-success">
                  {{ stichtagStats.daysUntilStichtag }}d
                </span>
              </div>
              <p class="text-sm font-semibold text-foreground mt-0.5">{{ formatDate(stichtagStats.nextStichtag) }}</p>
              <div class="mt-2 pt-2 border-t text-xs text-muted-foreground space-y-0.5">
                <div class="flex justify-between">
                  <span>≤20k</span>
                  <span class="font-medium text-foreground">{{ stichtagStats.u3IncomeBreakdown.upTo20k }}</span>
                </div>
                <div class="flex justify-between">
                  <span>20–35k</span>
                  <span class="font-medium text-foreground">{{ stichtagStats.u3IncomeBreakdown.from20To35k }}</span>
                </div>
                <div class="flex justify-between">
                  <span>35–55k</span>
                  <span class="font-medium text-foreground">{{ stichtagStats.u3IncomeBreakdown.from35To55k }}</span>
                </div>
                <div class="flex justify-between">
                  <span>Höchstbetrag</span>
                  <span class="font-medium text-foreground">{{ stichtagStats.u3IncomeBreakdown.maxAccepted }}</span>
                </div>
                <div class="flex justify-between text-muted-foreground">
                  <span>Pflegefamilie</span>
                  <span class="font-medium">{{ stichtagStats.u3IncomeBreakdown.fosterFamily }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Missing Income Warning Card -->
        <div
          v-if="childrenWithWarnings > 0"
          class="border bg-destructive-soft border-destructive-line rounded-xl p-5 cursor-pointer hover:bg-destructive-soft transition-colors"
          @click="router.push('/kinder?warnings=true')"
        >
          <div class="flex items-center gap-3">
            <div class="p-2.5 bg-destructive-soft rounded-lg">
              <CircleDollarSign class="h-5 w-5 text-destructive" />
            </div>
            <div class="flex-1">
              <p class="text-xs text-destructive font-medium">Fehlende Daten</p>
              <p class="text-lg font-semibold text-destructive">{{ childrenWithWarnings }} Kinder</p>
            </div>
            <svg class="w-4 h-4 text-destructive" fill="none" stroke="currentColor" viewBox="0 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7" />
            </svg>
          </div>
        </div>

        <!-- Unmatched Transactions Card -->
        <div
          v-if="unmatchedTotal > 0"
          class="border bg-warning-soft border-warning-line rounded-xl p-5 cursor-pointer hover:bg-warning-soft transition-colors"
          @click="router.push('/bankabgleich')"
        >
          <div class="flex items-center gap-3">
            <div class="p-2.5 bg-warning-soft rounded-lg">
              <Link2 class="h-5 w-5 text-warning" />
            </div>
            <div class="flex-1">
              <p class="text-xs text-warning font-medium">Nicht zugeordnet</p>
              <p class="text-lg font-semibold text-warning">{{ unmatchedTotal }} Transaktionen</p>
            </div>
            <svg class="w-4 h-4 text-warning" fill="none" stroke="currentColor" viewBox="0 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7" />
            </svg>
          </div>
        </div>

        <!-- Placeholder if no warnings and no unmatched -->
        <div
          v-if="childrenWithWarnings === 0 && unmatchedTotal === 0"
          class="border bg-muted/50 rounded-xl border-dashed border-border p-5"
        >
          <div class="flex items-center gap-3">
            <div class="p-2.5 bg-muted rounded-lg">
              <CheckCircle class="h-5 w-5 text-muted-foreground" />
            </div>
            <div>
              <p class="text-xs text-muted-foreground font-medium">Alles in Ordnung</p>
              <p class="text-sm text-muted-foreground">Keine Warnungen</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Monthly Overview -->
      <div class="card p-6">
        <div class="flex items-center justify-between mb-4">
          <div class="flex items-center gap-2">
            <TrendingUp class="h-5 w-5 text-primary" />
            <h2 class="text-base font-semibold">Jahresübersicht</h2>
          </div>
          <select
            v-model="selectedYear"
            class="px-3 py-1.5 text-sm border-input bg-card rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          >
            <option v-for="year in years" :key="year" :value="year">{{ year }}</option>
          </select>
        </div>

        <!-- Year summary stats -->
        <div v-if="!isLoadingMonthly && monthlyOverview" class="grid grid-cols-2 gap-4 mb-6">
          <div class="p-4 rounded-lg bg-destructive-soft">
            <div class="flex items-center gap-3">
              <div class="p-2 rounded-lg bg-destructive-soft">
                <AlertTriangle class="h-5 w-5 text-destructive" />
              </div>
              <div>
                <p class="text-xs text-destructive font-medium">Überfällig {{ selectedYear }}</p>
                <p class="text-xl font-semibold text-destructive">{{ monthlyOverview.totalOverdue }}</p>
                <p class="text-xs text-destructive font-medium">{{ formatCurrency(monthlyOverview.amountOverdue) }}</p>
              </div>
            </div>
          </div>
          <div class="p-4 rounded-lg bg-success-soft">
            <div class="flex items-center gap-3">
              <div class="p-2 rounded-lg bg-success-soft">
                <CheckCircle class="h-5 w-5 text-success" />
              </div>
              <div>
                <p class="text-xs text-success font-medium">Bezahlt {{ selectedYear }}</p>
                <p class="text-xl font-semibold text-success">{{ monthlyOverview.totalPaid }}</p>
                <p class="text-xs text-success font-medium">{{ formatCurrency(monthlyOverview.amountPaid) }}</p>
              </div>
            </div>
          </div>
        </div>

        <div v-if="isLoadingMonthly" class="flex items-center justify-center py-8">
          <Loader2 class="h-6 w-6 animate-spin text-primary" />
        </div>

        <div v-else-if="monthlyOverview && monthlyOverview.byMonth.length > 0" class="overflow-x-auto">
          <table class="w-full">
            <thead>
              <tr class="text-left text-sm text-muted-foreground border-b">
                <th class="pb-3 font-medium">Monat</th>
                <th class="pb-3 font-medium text-right">Offen</th>
                <th class="pb-3 font-medium text-right">Bezahlt</th>
                <th class="pb-3 font-medium text-right">Offen (€)</th>
                <th class="pb-3 font-medium text-right">Bezahlt (€)</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="month in monthlyOverview.byMonth"
                :key="month.month"
                class="border-b last:border-0 hover:bg-muted/50"
              >
                <td class="py-3 font-medium">{{ formatMonthName(month.month, 'short') }} {{ month.year }}</td>
                <td class="py-3 text-right">
                  <span
                    v-if="month.openCount > 0"
                    class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-primary-soft text-primary"
                  >
                    {{ month.openCount }}
                  </span>
                  <span v-else class="text-muted-foreground">-</span>
                </td>
                <td class="py-3 text-right">
                  <span
                    v-if="month.paidCount > 0"
                    class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-success-soft text-success"
                  >
                    {{ month.paidCount }}
                  </span>
                  <span v-else class="text-muted-foreground">-</span>
                </td>
                <td class="py-3 text-right text-primary">
                  {{ month.openAmount > 0 ? formatCurrency(month.openAmount) : '-' }}
                </td>
                <td class="py-3 text-right text-success">
                  {{ month.paidAmount > 0 ? formatCurrency(month.paidAmount) : '-' }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-else-if="monthlyError" class="text-center py-8 text-destructive">
          Jahresübersicht {{ selectedYear }} konnte nicht geladen werden.
          <button class="ml-1 underline" @click="loadMonthlyOverview">Erneut versuchen</button>
        </div>
        <div v-else class="text-center py-8 text-muted-foreground">
          Keine Daten für {{ selectedYear }} vorhanden
        </div>
      </div>
    </div>

    <!-- Stichtagsmeldung Modal -->
    <AppSheet v-if="showStichtagModal" title="Stichtagsreport" size="lg"
      @close="showStichtagModal = false">
          <div>
            <div class="mb-6">
              <label for="report-date" class="label">Stichtag</label>
              <div class="flex gap-2">
                <input
                  id="report-date"
                  v-model="selectedReportDate"
                  type="date"
                  class="input flex-1"
                  @change="loadStichtagReport"
                />
                <button
                  @click="selectedReportDate = todayISO(); loadStichtagReport()"
                  class="btn btn-secondary btn-sm whitespace-nowrap"
                >
                  Heute
                </button>
              </div>
            </div>

            <div v-if="isLoadingReport || isLoadingU3Children" class="flex items-center justify-center py-8">
              <Loader2 class="h-6 w-6 animate-spin text-primary" />
            </div>

            <div v-else class="space-y-6">
              <div v-if="report" class="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div class="border rounded-lg border-success-line bg-success-soft p-4">
                  <p class="text-sm text-success font-medium">Kinder angemeldet</p>
                  <p class="text-2xl font-semibold text-success mt-1">{{ report.totalChildrenInKita }}</p>
                  <p class="text-xs text-success mt-1">am {{ formatDate(report.reportDate) }}</p>
                </div>
                <div class="border rounded-lg border-border bg-primary-soft p-4">
                  <p class="text-sm text-primary font-medium">Davon U3</p>
                  <p class="text-2xl font-semibold text-primary mt-1">{{ report.u3ChildrenCount }}</p>
                </div>
                <div class="border rounded-lg border-warning-line bg-warning-soft p-4">
                  <p class="text-sm text-warning font-medium">Davon Ü3</p>
                  <p class="text-2xl font-semibold text-warning mt-1">{{ report.ue3ChildrenCount }}</p>
                </div>
              </div>

              <div v-if="report" class="border rounded-xl border-border">
                <div class="px-4 py-3 border-b bg-muted/50">
                  <h3 class="text-sm font-semibold text-foreground">Allgemeiner Stichtagsreport</h3>
                </div>
                <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 p-4">
                  <div class="border rounded-lg border-border">
                    <div class="px-4 py-3 border-b bg-card">
                      <h4 class="text-sm font-medium text-foreground">Rechtsanspruch</h4>
                    </div>
                    <div v-if="report.legalHoursBreakdown.length === 0" class="px-4 py-6 text-sm text-muted-foreground">
                      Keine Daten vorhanden
                    </div>
                    <div v-else class="overflow-x-auto">
                      <table class="table">
                        <thead>
                          <tr>
                            <th>Stundenmodell</th>
                            <th class="text-right">U3</th>
                            <th class="text-right">Ü3</th>
                            <th class="text-right">Gesamt</th>
                          </tr>
                        </thead>
                        <tbody>
                          <tr v-for="item in report.legalHoursBreakdown" :key="item.legalHours ?? 'unknown'">
                            <td class="text-foreground">{{ formatCareHours(item.legalHours) }}</td>
                            <td class="text-right font-semibold text-primary">{{ item.u3Count }}</td>
                            <td class="text-right font-semibold text-warning">{{ item.ue3Count }}</td>
                            <td class="text-right font-semibold text-foreground">{{ item.count }}</td>
                          </tr>
                        </tbody>
                      </table>
                    </div>
                  </div>
                  <div class="border rounded-lg border-border">
                    <div class="px-4 py-3 border-b bg-card">
                      <h4 class="text-sm font-medium text-foreground">Betreuungszeit</h4>
                    </div>
                    <div v-if="report.careHoursBreakdown.length === 0" class="px-4 py-6 text-sm text-muted-foreground">
                      Keine Daten vorhanden
                    </div>
                    <div v-else class="overflow-x-auto">
                      <table class="table">
                        <thead>
                          <tr>
                            <th>Stundenmodell</th>
                            <th class="text-right">U3</th>
                            <th class="text-right">Ü3</th>
                            <th class="text-right">Gesamt</th>
                          </tr>
                        </thead>
                        <tbody>
                          <tr v-for="item in report.careHoursBreakdown" :key="item.careHours ?? 'unknown'">
                            <td class="text-foreground">{{ formatCareHours(item.careHours) }}</td>
                            <td class="text-right font-semibold text-primary">{{ item.u3Count }}</td>
                            <td class="text-right font-semibold text-warning">{{ item.ue3Count }}</td>
                            <td class="text-right font-semibold text-foreground">{{ item.count }}</td>
                          </tr>
                        </tbody>
                      </table>
                    </div>
                  </div>
                </div>
              </div>

              <div class="border rounded-xl border-success-line bg-success-soft">
                <div class="border px-4 py-3 border-b border-success-line">
                  <h3 class="text-sm font-semibold text-success">U3-Stichtagsmeldung</h3>
                  <p class="text-xs text-success mt-1">Spezifische Prüfung für die offizielle Meldung</p>
                </div>

                <div v-if="u3Children.length === 0" class="px-4 py-8 text-center text-success">
                  Keine U3-Kinder gefunden
                </div>

                <div v-else class="p-4 space-y-6">
                <!-- Group by bracket -->
                <template
                  v-for="bracket in (['upTo20k', 'from20To35k', 'from35To55k', 'maxAccepted', 'fosterFamily'] as const)"
                  :key="bracket"
                >
                  <div>
                    <!-- Bracket header -->
                    <div
                      :class="[
                        'flex items-center justify-between px-3 py-2 rounded-lg mb-2',
                        bracket === 'fosterFamily' || bracket === 'maxAccepted' ? 'bg-muted' : 'bg-success-soft'
                      ]"
                    >
                      <div class="flex items-center gap-2">
                        <span :class="['text-sm font-medium', bracket === 'fosterFamily' || bracket === 'maxAccepted' ? 'text-muted-foreground' : 'text-success']">
                          {{ bracketLabels[bracket] }}
                        </span>
                        <span v-if="bracket === 'fosterFamily' || bracket === 'maxAccepted'" class="text-xs text-muted-foreground">(nicht gezählt)</span>
                      </div>
                      <span :class="['text-sm font-semibold', bracket === 'fosterFamily' || bracket === 'maxAccepted' ? 'text-muted-foreground' : 'text-success']">
                        {{ groupedU3Children[bracket].length }}
                      </span>
                    </div>

                    <!-- Children in bracket -->
                    <div class="space-y-1">
                      <div
                        v-for="child in groupedU3Children[bracket]"
                        :key="child.id"
                        :class="[
                          'flex items-center justify-between px-3 py-2 rounded cursor-pointer transition-colors',
                          bracket === 'fosterFamily' || bracket === 'maxAccepted'
                            ? 'text-muted-foreground hover:bg-muted/50'
                            : 'hover:bg-muted/50'
                        ]"
                        @click="router.push(`/kinder/${child.id}`); showStichtagModal = false"
                      >
                        <div class="flex items-center gap-3">
                          <span :class="['font-medium', bracket === 'fosterFamily' || bracket === 'maxAccepted' ? 'text-muted-foreground' : 'text-foreground']">
                            {{ child.lastName }}, {{ child.firstName }}
                          </span>
                          <span class="text-xs text-muted-foreground">#{{ child.memberNumber }}</span>
                        </div>
                        <div class="flex items-center gap-4">
                          <span :class="['text-sm w-12', bracket === 'fosterFamily' || bracket === 'maxAccepted' ? 'text-muted-foreground' : 'text-muted-foreground']">
                            {{ formatAge(child.birthDate) }}
                          </span>
                          <span
                            v-if="bracket !== 'fosterFamily' && bracket !== 'maxAccepted'"
                            :class="[
                              'text-sm font-medium w-20 text-right',
                              child.householdIncome === null ? 'text-destructive' : 'text-foreground'
                            ]"
                          >
                            {{ formatCurrencyWhole(child.householdIncome) }}
                          </span>
                          <span v-else class="text-sm text-muted-foreground w-20 text-right">—</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </template>
                </div>
              </div>
            </div>
          <div class="mt-4 border-t pt-4">
            <div class="flex items-center justify-between text-sm text-muted-foreground">
              <span>
                <span class="font-medium">{{ u3Children.length }}</span> U3-Kinder ·
                <span class="font-medium">{{ report?.totalChildrenInKita ?? 0 }}</span> Kinder gesamt in Kita
              </span>
              <span>
                <span class="font-medium">{{ u3Children.filter(c => !c.isFosterFamily && c.incomeStatus !== 'MAX_ACCEPTED').length }}</span> für Stichtagsmeldung
              </span>
            </div>
          </div>
          </div>
    </AppSheet>
  </div>
</template>
