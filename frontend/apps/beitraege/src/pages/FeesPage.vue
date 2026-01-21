<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { api } from '@/api';
import type { FeeExpectation, GenerateFeeRequest } from '@/api/types';
import {
  Filter,
  Loader2,
  Plus,
  CheckCircle,
  Clock,
  AlertTriangle,
  Calendar,
} from 'lucide-vue-next';

const fees = ref<FeeExpectation[]>([]);
const total = ref(0);
const isLoading = ref(true);
const error = ref<string | null>(null);

const selectedYear = ref(new Date().getFullYear());
const selectedMonth = ref<number | null>(null);
const selectedType = ref<string>('');

const showGenerateDialog = ref(false);
const generateForm = ref<GenerateFeeRequest>({
  year: new Date().getFullYear(),
  month: new Date().getMonth() + 1,
});
const isGenerating = ref(false);
const generateResult = ref<{ created: number; skipped: number } | null>(null);

const years = computed(() => {
  const currentYear = new Date().getFullYear();
  return [currentYear - 1, currentYear, currentYear + 1];
});

const months = [
  { value: 1, label: 'Januar' },
  { value: 2, label: 'Februar' },
  { value: 3, label: 'März' },
  { value: 4, label: 'April' },
  { value: 5, label: 'Mai' },
  { value: 6, label: 'Juni' },
  { value: 7, label: 'Juli' },
  { value: 8, label: 'August' },
  { value: 9, label: 'September' },
  { value: 10, label: 'Oktober' },
  { value: 11, label: 'November' },
  { value: 12, label: 'Dezember' },
];

const feeTypes = [
  { value: '', label: 'Alle Typen' },
  { value: 'MEMBERSHIP', label: 'Vereinsbeitrag' },
  { value: 'FOOD', label: 'Essensgeld' },
  { value: 'CHILDCARE', label: 'Platzgeld' },
];

async function loadFees() {
  isLoading.value = true;
  error.value = null;
  try {
    const response = await api.getFees({
      year: selectedYear.value,
      month: selectedMonth.value || undefined,
      feeType: selectedType.value || undefined,
      limit: 200,
    });
    fees.value = response.data;
    total.value = response.total;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoading.value = false;
  }
}

onMounted(loadFees);

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('de-DE');
}

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: 'EUR',
  }).format(amount);
}

function getFeeTypeName(type: string): string {
  switch (type) {
    case 'MEMBERSHIP':
      return 'Vereinsbeitrag';
    case 'FOOD':
      return 'Essensgeld';
    case 'CHILDCARE':
      return 'Platzgeld';
    default:
      return type;
  }
}

function getFeeTypeColor(type: string): string {
  switch (type) {
    case 'MEMBERSHIP':
      return 'bg-purple-100 text-purple-700';
    case 'FOOD':
      return 'bg-orange-100 text-orange-700';
    case 'CHILDCARE':
      return 'bg-blue-100 text-blue-700';
    default:
      return 'bg-gray-100 text-gray-700';
  }
}

function getMonthName(month: number): string {
  return months.find(m => m.value === month)?.label || '';
}

function getStatusInfo(fee: FeeExpectation) {
  if (fee.isPaid) {
    return { icon: CheckCircle, color: 'text-green-500', label: 'Bezahlt', bg: 'bg-green-50' };
  }
  const isOverdue = new Date(fee.dueDate) < new Date();
  if (isOverdue) {
    return { icon: AlertTriangle, color: 'text-red-500', label: 'Überfällig', bg: 'bg-red-50' };
  }
  return { icon: Clock, color: 'text-amber-500', label: 'Offen', bg: 'bg-amber-50' };
}

async function handleGenerate() {
  isGenerating.value = true;
  generateResult.value = null;
  try {
    const result = await api.generateFees(generateForm.value);
    generateResult.value = result;
    await loadFees();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Generieren';
  } finally {
    isGenerating.value = false;
  }
}

</script>

<template>
  <div>
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">Beiträge</h1>
        <p class="text-gray-600 mt-1">{{ total }} Beiträge gesamt</p>
      </div>
      <button
        @click="showGenerateDialog = true"
        class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
      >
        <Plus class="h-4 w-4" />
        Beiträge generieren
      </button>
    </div>

    <!-- Filters -->
    <div class="flex flex-wrap gap-4 mb-6 p-4 bg-white rounded-xl border">
      <div class="flex items-center gap-2">
        <Filter class="h-4 w-4 text-gray-400" />
        <span class="text-sm font-medium text-gray-700">Filter:</span>
      </div>
      <select
        v-model="selectedYear"
        @change="loadFees"
        class="px-3 py-1.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
      >
        <option v-for="year in years" :key="year" :value="year">{{ year }}</option>
      </select>
      <select
        v-model="selectedMonth"
        @change="loadFees"
        class="px-3 py-1.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
      >
        <option :value="null">Alle Monate</option>
        <option v-for="month in months" :key="month.value" :value="month.value">
          {{ month.label }}
        </option>
      </select>
      <select
        v-model="selectedType"
        @change="loadFees"
        class="px-3 py-1.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
      >
        <option v-for="type in feeTypes" :key="type.value" :value="type.value">
          {{ type.label }}
        </option>
      </select>
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4">
      <p class="text-red-600">{{ error }}</p>
      <button @click="loadFees" class="mt-2 text-sm text-red-700 underline">
        Erneut versuchen
      </button>
    </div>

    <!-- Fees table -->
    <div v-else class="bg-white rounded-xl border overflow-hidden">
      <table class="w-full">
        <thead class="bg-gray-50">
          <tr class="text-left text-sm text-gray-500">
            <th class="px-4 py-3 font-medium">Mitgl.-Nr.</th>
            <th class="px-4 py-3 font-medium">Kind</th>
            <th class="px-4 py-3 font-medium">Typ</th>
            <th class="px-4 py-3 font-medium">Zeitraum</th>
            <th class="px-4 py-3 font-medium text-right">Betrag</th>
            <th class="px-4 py-3 font-medium">Fällig</th>
            <th class="px-4 py-3 font-medium">Status</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="fee in fees"
            :key="fee.id"
            class="border-t hover:bg-gray-50 transition-colors"
          >
            <td class="px-4 py-3 text-gray-600 font-mono text-sm">
              {{ fee.child?.memberNumber }}
            </td>
            <td class="px-4 py-3">
              <span class="font-medium">
                {{ fee.child?.firstName }} {{ fee.child?.lastName }}
              </span>
            </td>
            <td class="px-4 py-3">
              <span
                :class="[
                  'inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium',
                  getFeeTypeColor(fee.feeType),
                ]"
              >
                {{ getFeeTypeName(fee.feeType) }}
              </span>
            </td>
            <td class="px-4 py-3 text-gray-600">
              {{ fee.month ? getMonthName(fee.month) + ' ' : '' }}{{ fee.year }}
            </td>
            <td class="px-4 py-3 text-right font-medium">
              {{ formatCurrency(fee.amount) }}
            </td>
            <td class="px-4 py-3 text-gray-600">
              {{ formatDate(fee.dueDate) }}
            </td>
            <td class="px-4 py-3">
              <div class="flex items-center gap-1.5">
                <component
                  :is="getStatusInfo(fee).icon"
                  :class="['h-4 w-4', getStatusInfo(fee).color]"
                />
                <span class="text-sm">{{ getStatusInfo(fee).label }}</span>
              </div>
            </td>
          </tr>
          <tr v-if="fees.length === 0">
            <td colspan="7" class="px-4 py-8 text-center text-gray-500">
              Keine Beiträge gefunden
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Generate Dialog -->
    <div
      v-if="showGenerateDialog"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="showGenerateDialog = false"
    >
      <div class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4 p-6">
        <div class="flex items-center gap-3 mb-6">
          <div class="p-2 bg-primary/10 rounded-lg">
            <Calendar class="h-6 w-6 text-primary" />
          </div>
          <div>
            <h2 class="text-xl font-semibold">Beiträge generieren</h2>
            <p class="text-sm text-gray-600">Erstellt fehlende Beiträge für alle aktiven Kinder</p>
          </div>
        </div>

        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Jahr</label>
            <select
              v-model="generateForm.year"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            >
              <option v-for="year in years" :key="year" :value="year">{{ year }}</option>
            </select>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Monat</label>
            <select
              v-model="generateForm.month"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            >
              <option :value="undefined">Nur Jahresbeitrag (Vereinsbeitrag)</option>
              <option v-for="month in months" :key="month.value" :value="month.value">
                {{ month.label }} (Essensgeld + ggf. Platzgeld)
              </option>
            </select>
          </div>

          <div v-if="generateResult" class="p-4 bg-green-50 border border-green-200 rounded-lg">
            <p class="text-green-700 font-medium">Erfolgreich generiert!</p>
            <p class="text-sm text-green-600 mt-1">
              {{ generateResult.created }} Beiträge erstellt, {{ generateResult.skipped }} übersprungen
            </p>
          </div>

          <div class="flex justify-end gap-3 pt-4">
            <button
              @click="showGenerateDialog = false"
              class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
            >
              Schließen
            </button>
            <button
              @click="handleGenerate"
              :disabled="isGenerating"
              class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Loader2 v-if="isGenerating" class="h-4 w-4 animate-spin" />
              <Plus v-else class="h-4 w-4" />
              Generieren
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
