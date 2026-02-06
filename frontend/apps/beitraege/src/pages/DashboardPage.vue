<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue';
import { api } from '@/api';
import type { FeeOverview, StichtagsmeldungStats, U3ChildDetail } from '@/api/types';
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
  X,
  CircleDollarSign,
} from 'lucide-vue-next';
import { useRouter } from 'vue-router';

const router = useRouter();
const overview = ref<FeeOverview | null>(null);
const monthlyOverview = ref<FeeOverview | null>(null);
const unmatchedTotal = ref(0);
const stichtagStats = ref<StichtagsmeldungStats | null>(null);
const u3Count = ref(0);
const totalChildrenCount = ref(0);
const childrenWithWarnings = ref(0);
const isLoading = ref(true);
const error = ref<string | null>(null);

// Year selector only for monthly overview
const selectedYear = ref(new Date().getFullYear());
const isLoadingMonthly = ref(false);

// Modal state
const showStichtagModal = ref(false);
const u3Children = ref<U3ChildDetail[]>([]);
const isLoadingU3Children = ref(false);

const years = computed(() => {
  const currentYear = new Date().getFullYear();
  return [currentYear - 1, currentYear, currentYear + 1];
});

const ue3Count = computed(() => totalChildrenCount.value - u3Count.value);

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
    const [overviewData, monthlyData, unmatchedData, stichtagData, u3Data, totalData, warningsData] = await Promise.all([
      api.getFeeOverview(), // Current state - no year filter
      api.getFeeOverview(selectedYear.value), // Monthly overview with year
      api.getUnmatchedTransactions({ limit: 1 }),
      api.getStichtagsmeldungStats(),
      api.getChildren({ activeOnly: true, u3Only: true, limit: 1 }),
      api.getChildren({ activeOnly: true, limit: 1 }),
      api.getChildren({ activeOnly: true, hasWarnings: true, limit: 1 }),
    ]);
    overview.value = overviewData;
    monthlyOverview.value = monthlyData;
    unmatchedTotal.value = unmatchedData.total;
    stichtagStats.value = stichtagData;
    u3Count.value = u3Data.total;
    totalChildrenCount.value = totalData.total;
    childrenWithWarnings.value = warningsData.total;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoading.value = false;
  }
}

async function loadMonthlyOverview() {
  isLoadingMonthly.value = true;
  try {
    monthlyOverview.value = await api.getFeeOverview(selectedYear.value);
  } catch (e) {
    // Silently fail for monthly, main dashboard still works
  } finally {
    isLoadingMonthly.value = false;
  }
}

async function openStichtagModal() {
  showStichtagModal.value = true;
  isLoadingU3Children.value = true;
  try {
    u3Children.value = await api.getU3Children();
  } catch (e) {
    // Silently fail
  } finally {
    isLoadingU3Children.value = false;
  }
}

watch(selectedYear, () => {
  loadMonthlyOverview();
});

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && showStichtagModal.value) {
    showStichtagModal.value = false;
  }
}

onMounted(() => {
  loadDashboard();
  document.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown);
});

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: 'EUR',
  }).format(amount);
}

function formatIncome(income: number | null): string {
  if (income === null) return '—';
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: 'EUR',
    maximumFractionDigits: 0,
  }).format(income);
}

function getMonthName(month: number): string {
  return new Date(2000, month - 1).toLocaleString('de-DE', { month: 'short' });
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  });
}

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

function getIncomeStatusLabel(status: string | null): string {
  if (!status) return '';
  const labels: Record<string, string> = {
    CURRENT: 'Aktuell',
    HISTORIC: 'Historisch',
    FOSTER_FAMILY: 'Pflegefamilie',
  };
  return labels[status] || status;
}
</script>

<template>
  <div>
    <!-- Header -->
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-gray-900">Dashboard</h1>
      <p class="text-gray-600 mt-1">Übersicht der Beitragszahlungen</p>
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4">
      <p class="text-red-600">{{ error }}</p>
      <button @click="loadDashboard" class="mt-2 text-sm text-red-700 underline">
        Erneut versuchen
      </button>
    </div>

    <!-- Content -->
    <div v-else-if="overview" class="space-y-6">
      <!-- Main Stats Card - Open Fees (current state) -->
      <div class="bg-white rounded-xl border p-5">
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <!-- Open Food Fees -->
          <div
            class="p-4 rounded-lg bg-orange-50 hover:bg-orange-100 cursor-pointer transition-colors"
            @click="router.push('/beitraege?feeType=FOOD&status=open')"
          >
            <div class="flex items-center gap-3">
              <div class="p-2 rounded-lg bg-orange-100">
                <Utensils class="h-5 w-5 text-orange-600" />
              </div>
              <div class="flex-1 min-w-0">
                <p class="text-xs text-orange-700 font-medium truncate">Offene Essensgelder</p>
                <p class="text-xl font-bold text-orange-900">{{ overview.openFoodCount }}</p>
              </div>
            </div>
          </div>

          <!-- Open Childcare Fees -->
          <div
            class="p-4 rounded-lg bg-blue-50 hover:bg-blue-100 cursor-pointer transition-colors"
            @click="router.push('/beitraege?feeType=CHILDCARE&status=open')"
          >
            <div class="flex items-center gap-3">
              <div class="p-2 rounded-lg bg-blue-100">
                <Home class="h-5 w-5 text-blue-600" />
              </div>
              <div class="flex-1 min-w-0">
                <p class="text-xs text-blue-700 font-medium truncate">Offene Platzgelder</p>
                <p class="text-xl font-bold text-blue-900">{{ overview.openChildcareCount }}</p>
              </div>
            </div>
          </div>

          <!-- Open Membership Fees -->
          <div
            class="p-4 rounded-lg bg-purple-50 hover:bg-purple-100 cursor-pointer transition-colors"
            @click="router.push('/beitraege?feeType=MEMBERSHIP&status=open')"
          >
            <div class="flex items-center gap-3">
              <div class="p-2 rounded-lg bg-purple-100">
                <Users2 class="h-5 w-5 text-purple-600" />
              </div>
              <div class="flex-1 min-w-0">
                <p class="text-xs text-purple-700 font-medium truncate">Offene Vereinsbeiträge</p>
                <p class="text-xl font-bold text-purple-900">{{ overview.openMembershipCount }}</p>
              </div>
            </div>
          </div>
        </div>

        <!-- Summary row -->
        <div class="mt-4 pt-4 border-t flex items-center justify-between">
          <div class="flex items-center gap-2 text-gray-600">
            <Receipt class="h-4 w-4" />
            <span class="text-sm">Offener Gesamtbetrag:</span>
          </div>
          <span class="text-lg font-bold text-gray-900">
            {{ formatCurrency(overview.amountOpen + overview.amountOverdue) }}
          </span>
        </div>
      </div>

      <!-- Secondary Cards Row -->
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <!-- U3/Ü3 Children Card -->
        <div class="bg-white rounded-xl border p-5">
          <div class="flex items-center gap-3">
            <div class="p-2.5 bg-violet-100 rounded-lg">
              <Baby class="h-5 w-5 text-violet-600" />
            </div>
            <div>
              <p class="text-xs text-gray-500 font-medium">Aktive Kinder</p>
              <p class="text-lg font-bold text-gray-900">
                {{ u3Count }} <span class="text-gray-400 font-normal">U3</span>
                · {{ ue3Count }} <span class="text-gray-400 font-normal">Ü3</span>
              </p>
              <p class="text-xs text-gray-500">Gesamt: {{ totalChildrenCount }}</p>
            </div>
          </div>
        </div>

        <!-- Stichtagsmeldung Card (clickable) -->
        <div
          v-if="stichtagStats"
          class="bg-white rounded-xl border p-5 cursor-pointer hover:bg-gray-50 transition-colors"
          @click="openStichtagModal"
        >
          <div class="flex items-start gap-3">
            <div class="p-2.5 bg-emerald-100 rounded-lg">
              <Calendar class="h-5 w-5 text-emerald-600" />
            </div>
            <div class="flex-1 min-w-0">
              <div class="flex items-center justify-between gap-2">
                <p class="text-xs text-gray-500 font-medium">Stichtagsmeldung</p>
                <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-emerald-100 text-emerald-700">
                  {{ stichtagStats.daysUntilStichtag }}d
                </span>
              </div>
              <p class="text-sm font-semibold text-gray-900 mt-0.5">{{ formatDate(stichtagStats.nextStichtag) }}</p>
              <div class="mt-2 pt-2 border-t text-xs text-gray-500 space-y-0.5">
                <div class="flex justify-between">
                  <span>≤20k</span>
                  <span class="font-medium text-gray-700">{{ stichtagStats.u3IncomeBreakdown.upTo20k }}</span>
                </div>
                <div class="flex justify-between">
                  <span>20–35k</span>
                  <span class="font-medium text-gray-700">{{ stichtagStats.u3IncomeBreakdown.from20To35k }}</span>
                </div>
                <div class="flex justify-between">
                  <span>35–55k</span>
                  <span class="font-medium text-gray-700">{{ stichtagStats.u3IncomeBreakdown.from35To55k }}</span>
                </div>
                <div class="flex justify-between">
                  <span>Höchstbetrag</span>
                  <span class="font-medium text-gray-700">{{ stichtagStats.u3IncomeBreakdown.maxAccepted }}</span>
                </div>
                <div class="flex justify-between text-gray-400">
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
          class="bg-red-50 border border-red-200 rounded-xl p-5 cursor-pointer hover:bg-red-100 transition-colors"
          @click="router.push('/kinder?warnings=true')"
        >
          <div class="flex items-center gap-3">
            <div class="p-2.5 bg-red-100 rounded-lg">
              <CircleDollarSign class="h-5 w-5 text-red-600" />
            </div>
            <div class="flex-1">
              <p class="text-xs text-red-700 font-medium">Fehlende Daten</p>
              <p class="text-lg font-bold text-red-900">{{ childrenWithWarnings }} Kinder</p>
            </div>
            <svg class="w-4 h-4 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
            </svg>
          </div>
        </div>

        <!-- Unmatched Transactions Card -->
        <div
          v-if="unmatchedTotal > 0"
          class="bg-amber-50 border border-amber-200 rounded-xl p-5 cursor-pointer hover:bg-amber-100 transition-colors"
          @click="router.push('/import?tab=unmatched')"
        >
          <div class="flex items-center gap-3">
            <div class="p-2.5 bg-amber-100 rounded-lg">
              <Link2 class="h-5 w-5 text-amber-600" />
            </div>
            <div class="flex-1">
              <p class="text-xs text-amber-700 font-medium">Nicht zugeordnet</p>
              <p class="text-lg font-bold text-amber-900">{{ unmatchedTotal }} Transaktionen</p>
            </div>
            <svg class="w-4 h-4 text-amber-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
            </svg>
          </div>
        </div>

        <!-- Placeholder if no warnings and no unmatched -->
        <div
          v-if="childrenWithWarnings === 0 && unmatchedTotal === 0"
          class="bg-gray-50 rounded-xl border border-dashed border-gray-200 p-5"
        >
          <div class="flex items-center gap-3">
            <div class="p-2.5 bg-gray-100 rounded-lg">
              <CheckCircle class="h-5 w-5 text-gray-400" />
            </div>
            <div>
              <p class="text-xs text-gray-400 font-medium">Alles in Ordnung</p>
              <p class="text-sm text-gray-500">Keine Warnungen</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Monthly Overview -->
      <div class="bg-white rounded-xl border p-6">
        <div class="flex items-center justify-between mb-4">
          <div class="flex items-center gap-2">
            <TrendingUp class="h-5 w-5 text-primary" />
            <h2 class="text-lg font-semibold">Jahresübersicht</h2>
          </div>
          <select
            v-model="selectedYear"
            class="px-3 py-1.5 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          >
            <option v-for="year in years" :key="year" :value="year">{{ year }}</option>
          </select>
        </div>

        <!-- Year summary stats -->
        <div v-if="!isLoadingMonthly && monthlyOverview" class="grid grid-cols-2 gap-4 mb-6">
          <div class="p-4 rounded-lg bg-red-50">
            <div class="flex items-center gap-3">
              <div class="p-2 rounded-lg bg-red-100">
                <AlertTriangle class="h-5 w-5 text-red-600" />
              </div>
              <div>
                <p class="text-xs text-red-700 font-medium">Überfällig {{ selectedYear }}</p>
                <p class="text-xl font-bold text-red-900">{{ monthlyOverview.totalOverdue }}</p>
                <p class="text-xs text-red-600 font-medium">{{ formatCurrency(monthlyOverview.amountOverdue) }}</p>
              </div>
            </div>
          </div>
          <div class="p-4 rounded-lg bg-green-50">
            <div class="flex items-center gap-3">
              <div class="p-2 rounded-lg bg-green-100">
                <CheckCircle class="h-5 w-5 text-green-600" />
              </div>
              <div>
                <p class="text-xs text-green-700 font-medium">Bezahlt {{ selectedYear }}</p>
                <p class="text-xl font-bold text-green-900">{{ monthlyOverview.totalPaid }}</p>
                <p class="text-xs text-green-600 font-medium">{{ formatCurrency(monthlyOverview.amountPaid) }}</p>
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
              <tr class="text-left text-sm text-gray-500 border-b">
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
                class="border-b last:border-0 hover:bg-gray-50"
              >
                <td class="py-3 font-medium">{{ getMonthName(month.month) }} {{ month.year }}</td>
                <td class="py-3 text-right">
                  <span
                    v-if="month.openCount > 0"
                    class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-700"
                  >
                    {{ month.openCount }}
                  </span>
                  <span v-else class="text-gray-400">-</span>
                </td>
                <td class="py-3 text-right">
                  <span
                    v-if="month.paidCount > 0"
                    class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-700"
                  >
                    {{ month.paidCount }}
                  </span>
                  <span v-else class="text-gray-400">-</span>
                </td>
                <td class="py-3 text-right text-blue-600">
                  {{ month.openAmount > 0 ? formatCurrency(month.openAmount) : '-' }}
                </td>
                <td class="py-3 text-right text-green-600">
                  {{ month.paidAmount > 0 ? formatCurrency(month.paidAmount) : '-' }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-else class="text-center py-8 text-gray-500">
          Keine Daten für {{ selectedYear }} vorhanden
        </div>
      </div>
    </div>

    <!-- Stichtagsmeldung Modal -->
    <Teleport to="body">
      <div
        v-if="showStichtagModal"
        class="fixed inset-0 z-50 flex items-center justify-center p-4"
      >
        <!-- Backdrop -->
        <div
          class="absolute inset-0 bg-black/50"
          @click="showStichtagModal = false"
        />

        <!-- Modal -->
        <div class="relative bg-white rounded-xl shadow-xl max-w-2xl w-full max-h-[80vh] flex flex-col">
          <!-- Header -->
          <div class="flex items-center justify-between p-6 border-b">
            <div>
              <h2 class="text-lg font-semibold text-gray-900">Stichtagsmeldung - U3-Kinder</h2>
              <p v-if="stichtagStats" class="text-sm text-gray-500 mt-0.5">
                Stichtag: {{ formatDate(stichtagStats.nextStichtag) }}
              </p>
            </div>
            <button
              @click="showStichtagModal = false"
              class="p-2 text-gray-400 hover:text-gray-600 rounded-lg hover:bg-gray-100"
            >
              <X class="h-5 w-5" />
            </button>
          </div>

          <!-- Content -->
          <div class="flex-1 overflow-auto p-6">
            <div v-if="isLoadingU3Children" class="flex items-center justify-center py-8">
              <Loader2 class="h-6 w-6 animate-spin text-primary" />
            </div>

            <div v-else-if="u3Children.length === 0" class="text-center py-8 text-gray-500">
              Keine U3-Kinder gefunden
            </div>

            <div v-else class="space-y-6">
              <!-- Group by bracket -->
              <template v-for="bracket in (['upTo20k', 'from20To35k', 'from35To55k', 'maxAccepted', 'fosterFamily'] as const)" :key="bracket">
                <div>
                  <!-- Bracket header -->
                  <div
                    :class="[
                      'flex items-center justify-between px-3 py-2 rounded-lg mb-2',
                      bracket === 'fosterFamily' || bracket === 'maxAccepted' ? 'bg-gray-100' : 'bg-emerald-50'
                    ]"
                  >
                    <div class="flex items-center gap-2">
                      <span :class="['text-sm font-medium', bracket === 'fosterFamily' || bracket === 'maxAccepted' ? 'text-gray-500' : 'text-emerald-800']">
                        {{ bracketLabels[bracket] }}
                      </span>
                      <span v-if="bracket === 'fosterFamily' || bracket === 'maxAccepted'" class="text-xs text-gray-400">(nicht gezählt)</span>
                    </div>
                    <span :class="['text-sm font-bold', bracket === 'fosterFamily' || bracket === 'maxAccepted' ? 'text-gray-500' : 'text-emerald-800']">
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
                          ? 'text-gray-400 hover:bg-gray-50'
                          : 'hover:bg-gray-50'
                      ]"
                      @click="router.push(`/kinder/${child.id}`); showStichtagModal = false"
                    >
                      <div class="flex items-center gap-3">
                        <span :class="['font-medium', bracket === 'fosterFamily' || bracket === 'maxAccepted' ? 'text-gray-400' : 'text-gray-900']">
                          {{ child.lastName }}, {{ child.firstName }}
                        </span>
                        <span class="text-xs text-gray-400">#{{ child.memberNumber }}</span>
                      </div>
                      <div class="flex items-center gap-4">
                        <span :class="['text-sm w-12', bracket === 'fosterFamily' || bracket === 'maxAccepted' ? 'text-gray-400' : 'text-gray-600']">
                          {{ formatAge(child.birthDate) }}
                        </span>
                        <span
                          v-if="bracket !== 'fosterFamily' && bracket !== 'maxAccepted'"
                          :class="[
                            'text-sm font-medium w-20 text-right',
                            child.householdIncome === null ? 'text-red-600' : 'text-gray-700'
                          ]"
                        >
                          {{ formatIncome(child.householdIncome) }}
                        </span>
                        <span v-else class="text-sm text-gray-400 w-20 text-right">—</span>
                      </div>
                    </div>
                  </div>
                </div>
              </template>
            </div>
          </div>

          <!-- Footer -->
          <div class="p-6 border-t bg-gray-50 rounded-b-xl">
            <div class="flex items-center justify-between text-sm text-gray-600">
              <span>
                <span class="font-medium">{{ u3Children.length }}</span> U3-Kinder ·
                <span class="font-medium">{{ stichtagStats?.totalChildrenInKita ?? 0 }}</span> Kinder gesamt in Kita
              </span>
              <span>
                <span class="font-medium">{{ u3Children.filter(c => !c.isFosterFamily && c.incomeStatus !== 'MAX_ACCEPTED').length }}</span> für Stichtagsmeldung
              </span>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
