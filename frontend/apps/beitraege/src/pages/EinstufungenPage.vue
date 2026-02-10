<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue';
import { useRouter } from 'vue-router';
import { api } from '@/api';
import type { Einstufung } from '@/api/types';
import {
  Plus,
  Loader2,
  Calendar,
  Euro,
  FileText,
  ChevronLeft,
  ChevronRight,
  Trash2,
  User,
  Home,
  Clock,
} from 'lucide-vue-next';

const router = useRouter();

const einstufungen = ref<Einstufung[]>([]);
const total = ref(0);
const isLoading = ref(true);
const error = ref<string | null>(null);

const selectedYear = ref(new Date().getFullYear());
const currentPage = ref(1);
const pageSize = ref(25);

const totalPages = computed(() => Math.ceil(total.value / pageSize.value));

const years = computed(() => {
  const current = new Date().getFullYear();
  return [current - 2, current - 1, current, current + 1];
});

async function loadEinstufungen() {
  isLoading.value = true;
  error.value = null;
  try {
    const response = await api.getEinstufungen({
      year: selectedYear.value,
      page: currentPage.value,
      perPage: pageSize.value,
    });
    einstufungen.value = response.data;
    total.value = response.total;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoading.value = false;
  }
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('de-DE');
}

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR' }).format(amount);
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
    return 'bg-green-100 text-green-800';
  if (rule.includes('Entlastung'))
    return 'bg-blue-100 text-blue-800';
  if (rule.includes('Höchstsatz') || rule.includes('Satzung'))
    return 'bg-orange-100 text-orange-800';
  if (rule.includes('Pflegefamilie'))
    return 'bg-purple-100 text-purple-800';
  return 'bg-gray-100 text-gray-800';
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
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Löschen';
  } finally {
    isDeleting.value = false;
  }
}

watch([selectedYear], () => {
  currentPage.value = 1;
  loadEinstufungen();
});

watch([currentPage, pageSize], () => {
  loadEinstufungen();
});

onMounted(() => loadEinstufungen());
</script>

<template>
  <div>
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">Einstufungen</h1>
        <p class="text-sm text-gray-500 mt-1">
          Beitragseinstufungen für {{ selectedYear }}
          <span v-if="total > 0">({{ total }} gesamt)</span>
        </p>
      </div>
      <button
        @click="router.push('/einstufungen/neu')"
        class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
      >
        <Plus class="h-4 w-4" />
        Neue Einstufung
      </button>
    </div>

    <!-- Year filter -->
    <div class="mb-4 flex items-center gap-3">
      <Calendar class="h-4 w-4 text-gray-400" />
      <select
        v-model="selectedYear"
        class="border rounded-lg px-3 py-1.5 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary"
      >
        <option v-for="y in years" :key="y" :value="y">{{ y }}</option>
      </select>
    </div>

    <!-- Loading -->
    <div v-if="isLoading" class="flex items-center justify-center py-20">
      <Loader2 class="h-8 w-8 text-primary animate-spin" />
    </div>

    <!-- Error -->
    <div v-else-if="error" class="bg-red-50 text-red-700 p-4 rounded-lg">
      {{ error }}
    </div>

    <!-- Empty state -->
    <div v-else-if="einstufungen.length === 0" class="text-center py-20">
      <FileText class="h-12 w-12 text-gray-300 mx-auto mb-4" />
      <h3 class="text-lg font-medium text-gray-900 mb-1">Keine Einstufungen</h3>
      <p class="text-gray-500 mb-4">Für {{ selectedYear }} wurden noch keine Einstufungen erstellt.</p>
      <button
        @click="router.push('/einstufungen/neu')"
        class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90"
      >
        <Plus class="h-4 w-4" />
        Erste Einstufung erstellen
      </button>
    </div>

    <!-- Table -->
    <div v-else class="bg-white rounded-lg border shadow-sm overflow-hidden">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Kind</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Haushalt</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Bereich</th>
            <th class="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Einkommen</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Regel</th>
            <th class="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Platzgeld</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Gültig ab</th>
            <th class="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-200">
          <tr
            v-for="e in einstufungen"
            :key="e.id"
            class="hover:bg-gray-50 cursor-pointer transition-colors"
            @click="router.push(`/einstufungen/${e.id}`)"
          >
            <td class="px-4 py-3 whitespace-nowrap">
              <div class="flex items-center gap-2">
                <User class="h-4 w-4 text-gray-400" />
                <span class="text-sm font-medium text-gray-900">{{ getChildName(e) }}</span>
              </div>
            </td>
            <td class="px-4 py-3 whitespace-nowrap">
              <div class="flex items-center gap-2">
                <Home class="h-4 w-4 text-gray-400" />
                <span class="text-sm text-gray-600">{{ getHouseholdName(e) }}</span>
              </div>
            </td>
            <td class="px-4 py-3 whitespace-nowrap">
              <span class="text-sm text-gray-600">{{ formatCareType(e.careType) }} · {{ e.careHoursPerWeek }}h</span>
            </td>
            <td class="px-4 py-3 whitespace-nowrap text-right">
              <span v-if="e.highestRateVoluntary" class="text-sm text-gray-500 italic">Höchstsatz</span>
              <span v-else class="text-sm text-gray-900">{{ formatCurrency(e.annualNetIncome) }}</span>
            </td>
            <td class="px-4 py-3 whitespace-nowrap">
              <span
                class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium"
                :class="getRuleBadgeClass(e.feeRule)"
              >
                {{ e.feeRule }}
              </span>
            </td>
            <td class="px-4 py-3 whitespace-nowrap text-right">
              <span class="text-sm font-semibold text-gray-900">{{ formatCurrency(e.monthlyChildcareFee) }}</span>
              <span v-if="e.discountPercent > 0" class="text-xs text-green-600 ml-1">-{{ e.discountPercent }}%</span>
            </td>
            <td class="px-4 py-3 whitespace-nowrap">
              <span class="text-sm text-gray-600">{{ formatDate(e.validFrom) }}</span>
            </td>
            <td class="px-4 py-3 whitespace-nowrap text-right">
              <button
                @click.stop="confirmDelete(e)"
                class="p-1 text-gray-400 hover:text-red-500 transition-colors"
                title="Löschen"
              >
                <Trash2 class="h-4 w-4" />
              </button>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="flex items-center justify-between px-4 py-3 border-t bg-gray-50">
        <span class="text-sm text-gray-500">
          Seite {{ currentPage }} von {{ totalPages }}
        </span>
        <div class="flex items-center gap-2">
          <button
            :disabled="currentPage <= 1"
            @click="currentPage--"
            class="p-1 rounded hover:bg-gray-200 disabled:opacity-30 disabled:cursor-not-allowed"
          >
            <ChevronLeft class="h-5 w-5" />
          </button>
          <button
            :disabled="currentPage >= totalPages"
            @click="currentPage++"
            class="p-1 rounded hover:bg-gray-200 disabled:opacity-30 disabled:cursor-not-allowed"
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
        <div class="bg-white rounded-lg shadow-xl max-w-md w-full mx-4 p-6">
          <h3 class="text-lg font-semibold text-gray-900 mb-2">Einstufung löschen</h3>
          <p class="text-sm text-gray-600 mb-4">
            Möchtest du die Einstufung für
            <strong>{{ deleteTarget?.child ? `${deleteTarget.child.firstName} ${deleteTarget.child.lastName}` : '' }}</strong>
            ({{ deleteTarget?.year }}) wirklich löschen?
          </p>
          <div class="flex justify-end gap-2">
            <button
              @click="showDeleteDialog = false"
              class="px-4 py-2 text-sm text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200"
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
