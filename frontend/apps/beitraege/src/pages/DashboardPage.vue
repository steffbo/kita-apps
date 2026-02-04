<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { api } from '@/api';
import type { FeeOverview } from '@/api/types';
import {
  Receipt,
  CheckCircle,
  Clock,
  AlertTriangle,
  TrendingUp,
  Loader2,
  Users,
  Link2,
} from 'lucide-vue-next';
import { useRouter } from 'vue-router';

const router = useRouter();
const overview = ref<FeeOverview | null>(null);
const unmatchedTotal = ref(0);
const isLoading = ref(true);
const error = ref<string | null>(null);
const selectedYear = ref(new Date().getFullYear());

const years = computed(() => {
  const currentYear = new Date().getFullYear();
  return [currentYear - 1, currentYear, currentYear + 1];
});

async function loadOverview() {
  isLoading.value = true;
  error.value = null;
  try {
    const [overviewData, unmatchedData] = await Promise.all([
      api.getFeeOverview(selectedYear.value),
      api.getUnmatchedTransactions({ limit: 1 }),
    ]);
    overview.value = overviewData;
    unmatchedTotal.value = unmatchedData.total;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoading.value = false;
  }
}

onMounted(loadOverview);

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: 'EUR',
  }).format(amount);
}

function getMonthName(month: number): string {
  return new Date(2000, month - 1).toLocaleString('de-DE', { month: 'short' });
}

const stats = computed(() => {
  if (!overview.value) return [];
  return [
    {
      name: 'Offene Beiträge',
      value: overview.value.totalOpen,
      amount: formatCurrency(overview.value.amountOpen),
      icon: Clock,
      color: 'text-blue-600',
      bgColor: 'bg-blue-100',
    },
    {
      name: 'Bezahlte Beiträge',
      value: overview.value.totalPaid,
      amount: formatCurrency(overview.value.amountPaid),
      icon: CheckCircle,
      color: 'text-green-600',
      bgColor: 'bg-green-100',
    },
    {
      name: 'Überfällige Beiträge',
      value: overview.value.totalOverdue,
      amount: formatCurrency(overview.value.amountOverdue),
      icon: AlertTriangle,
      color: 'text-red-600',
      bgColor: 'bg-red-100',
    },
    {
      name: 'Gesamtbetrag',
      value: overview.value.totalOpen + overview.value.totalPaid + overview.value.totalOverdue,
      amount: formatCurrency(
        overview.value.amountOpen + overview.value.amountPaid + overview.value.amountOverdue
      ),
      icon: Receipt,
      color: 'text-primary',
      bgColor: 'bg-primary/10',
    },
  ];
});
</script>

<template>
  <div>
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-8">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">Dashboard</h1>
        <p class="text-gray-600 mt-1">Übersicht der Beitragszahlungen</p>
      </div>
      <div class="flex items-center gap-2">
        <label for="year" class="text-sm font-medium text-gray-700">Jahr:</label>
        <select
          id="year"
          v-model="selectedYear"
          @change="loadOverview"
          class="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
        >
          <option v-for="year in years" :key="year" :value="year">{{ year }}</option>
        </select>
      </div>
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4">
      <p class="text-red-600">{{ error }}</p>
      <button @click="loadOverview" class="mt-2 text-sm text-red-700 underline">
        Erneut versuchen
      </button>
    </div>

    <!-- Content -->
    <div v-else-if="overview">
      <!-- Stats Grid -->
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        <div
          v-for="stat in stats"
          :key="stat.name"
          class="bg-white rounded-xl border p-6 hover:shadow-md transition-shadow"
        >
          <div class="flex items-center gap-4">
            <div :class="['p-3 rounded-lg', stat.bgColor]">
              <component :is="stat.icon" :class="['h-6 w-6', stat.color]" />
            </div>
            <div>
              <p class="text-sm text-gray-600">{{ stat.name }}</p>
              <p class="text-2xl font-bold text-gray-900">{{ stat.value }}</p>
              <p :class="['text-sm font-medium', stat.color]">{{ stat.amount }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Warning Cards Grid -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-8">
        <!-- Children with Missing Payments Warning Card -->
        <div
          v-if="overview.childrenWithOpenFees > 0"
          class="bg-amber-50 border border-amber-200 rounded-xl p-6 cursor-pointer hover:bg-amber-100 transition-colors"
          @click="router.push('/kinder?openFees=true')"
        >
          <div class="flex items-center gap-4">
            <div class="p-3 bg-amber-100 rounded-lg">
              <Users class="h-6 w-6 text-amber-600" />
            </div>
            <div class="flex-1">
              <p class="text-sm text-amber-700 font-medium">Fehlende Zahlungen</p>
              <p class="text-lg text-amber-900">
                {{ overview.childrenWithOpenFees }} Kinder haben offene Beiträge
              </p>
              <p v-if="overview" class="text-xs text-amber-700 mt-1">
                Vereinsbeitrag: {{ overview.openMembershipCount }}
                · Essensgeld: {{ overview.openFoodCount }}
                · Platzgeld: {{ overview.openChildcareCount }}
              </p>
            </div>
            <div class="text-amber-600">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
              </svg>
            </div>
          </div>
        </div>

        <!-- Unmatched Transactions Warning Card -->
        <div
          v-if="unmatchedTotal > 0"
          class="bg-orange-50 border border-orange-200 rounded-xl p-6 cursor-pointer hover:bg-orange-100 transition-colors"
          @click="router.push('/import?tab=unmatched')"
        >
          <div class="flex items-center gap-4">
            <div class="p-3 bg-orange-100 rounded-lg">
              <Link2 class="h-6 w-6 text-orange-600" />
            </div>
            <div class="flex-1">
              <p class="text-sm text-orange-700 font-medium">Nicht zugeordnet</p>
              <p class="text-lg text-orange-900">
                {{ unmatchedTotal }} Transaktionen nicht zugeordnet
              </p>
            </div>
            <div class="text-orange-600">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
              </svg>
            </div>
          </div>
        </div>
      </div>

      <!-- Monthly Overview -->
      <div class="bg-white rounded-xl border p-6">
        <div class="flex items-center gap-2 mb-6">
          <TrendingUp class="h-5 w-5 text-primary" />
          <h2 class="text-lg font-semibold">Monatliche Übersicht</h2>
        </div>

        <div v-if="overview.byMonth.length > 0" class="overflow-x-auto">
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
                v-for="month in overview.byMonth"
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
  </div>
</template>
