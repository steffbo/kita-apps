<script setup lang="ts">
import { ref, computed } from 'vue';
import { ChevronLeft, ChevronRight, Loader2, TrendingUp, Clock, Calendar, Users, Target } from 'lucide-vue-next';
import { 
  useOverviewStatistics,
  useWeeklyStatistics,
  useAuth,
} from '@kita/shared';
import { formatDate, getWeekStart } from '@kita/shared/utils';
import { Button } from '@/components/ui';

// Auth available for future admin-only features
useAuth();

// Current month
const currentMonth = ref(new Date());
const currentWeek = ref(new Date());

// Queries
const { data: overview, isLoading: overviewLoading, error: overviewError } = useOverviewStatistics(currentMonth);
const { data: weekly, isLoading: weeklyLoading } = useWeeklyStatistics(computed(() => getWeekStart(currentWeek.value)));

// Navigation
function previousMonth() {
  const newDate = new Date(currentMonth.value);
  newDate.setMonth(newDate.getMonth() - 1);
  currentMonth.value = newDate;
}

function nextMonth() {
  const newDate = new Date(currentMonth.value);
  newDate.setMonth(newDate.getMonth() + 1);
  currentMonth.value = newDate;
}

function previousWeek() {
  const newDate = new Date(currentWeek.value);
  newDate.setDate(newDate.getDate() - 7);
  currentWeek.value = newDate;
}

function nextWeek() {
  const newDate = new Date(currentWeek.value);
  newDate.setDate(newDate.getDate() + 7);
  currentWeek.value = newDate;
}

// Format helpers
function formatHours(hours: number | undefined): string {
  if (hours === undefined) return '-';
  return `${hours.toFixed(1)} Std.`;
}

function formatMonthYear(date: Date): string {
  return date.toLocaleDateString('de-DE', { month: 'long', year: 'numeric' });
}

function getOvertimeClass(hours: number | undefined): string {
  if (!hours) return 'text-stone-600';
  return hours > 0 ? 'text-green-600' : hours < 0 ? 'text-red-600' : 'text-stone-600';
}

// Capacity calculation helpers
function getCapacityPercent(scheduled: number | undefined, contracted: number | undefined): number {
  if (!scheduled || !contracted || contracted === 0) return 0;
  return Math.min(Math.round((scheduled / contracted) * 100), 150);
}

function getCapacityBarClass(percent: number): string {
  if (percent < 80) return 'bg-amber-500'; // Under-scheduled
  if (percent <= 105) return 'bg-green-500'; // Good
  if (percent <= 120) return 'bg-blue-500'; // Slight overtime
  return 'bg-red-500'; // Over-scheduled
}

function getDifferenceText(scheduled: number | undefined, contracted: number | undefined): string {
  if (scheduled === undefined || contracted === undefined) return '-';
  const diff = scheduled - contracted;
  if (diff === 0) return 'Genau richtig';
  return diff > 0 ? `+${diff.toFixed(1)} Std.` : `${diff.toFixed(1)} Std.`;
}

function getDifferenceClass(scheduled: number | undefined, contracted: number | undefined): string {
  if (scheduled === undefined || contracted === undefined) return 'text-stone-500';
  const diff = scheduled - contracted;
  if (diff > 2) return 'text-red-600 font-medium';
  if (diff > 0) return 'text-blue-600';
  if (diff < -2) return 'text-amber-600 font-medium';
  if (diff < 0) return 'text-amber-600';
  return 'text-green-600 font-medium';
}

const isLoading = computed(() => overviewLoading.value);
</script>

<template>
  <div>
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-stone-900">Statistiken</h1>
      <p class="text-stone-600">Übersicht über Arbeitszeiten und Abwesenheiten</p>
    </div>

    <!-- Month Navigation -->
    <div class="flex items-center gap-4 mb-6">
      <div class="flex items-center gap-2">
        <Button variant="outline" size="icon" @click="previousMonth">
          <ChevronLeft class="w-4 h-4" />
        </Button>
        <span class="font-semibold text-lg min-w-[180px] text-center">
          {{ formatMonthYear(currentMonth) }}
        </span>
        <Button variant="outline" size="icon" @click="nextMonth">
          <ChevronRight class="w-4 h-4" />
        </Button>
      </div>
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="w-8 h-8 animate-spin text-primary" />
    </div>

    <!-- Error state -->
    <div v-else-if="overviewError" class="bg-destructive/10 text-destructive rounded-lg p-4">
      <p>Fehler beim Laden: {{ (overviewError as Error).message }}</p>
    </div>

    <!-- Content -->
    <div v-else class="space-y-6">
      <!-- Summary Cards -->
      <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <div class="bg-white rounded-lg border border-stone-200 p-6">
          <div class="flex items-center gap-3">
            <div class="p-2 bg-primary/10 rounded-lg">
              <Users class="w-5 h-5 text-primary" />
            </div>
            <div>
              <p class="text-sm text-stone-500">Mitarbeiter</p>
              <p class="text-2xl font-bold text-stone-900">{{ overview?.totalEmployees ?? '-' }}</p>
            </div>
          </div>
        </div>

        <div class="bg-white rounded-lg border border-stone-200 p-6">
          <div class="flex items-center gap-3">
            <div class="p-2 bg-blue-100 rounded-lg">
              <Clock class="w-5 h-5 text-blue-600" />
            </div>
            <div>
              <p class="text-sm text-stone-500">Geplante Stunden</p>
              <p class="text-2xl font-bold text-stone-900">{{ formatHours(overview?.totalScheduledHours) }}</p>
            </div>
          </div>
        </div>

        <div class="bg-white rounded-lg border border-stone-200 p-6">
          <div class="flex items-center gap-3">
            <div class="p-2 bg-green-100 rounded-lg">
              <TrendingUp class="w-5 h-5 text-green-600" />
            </div>
            <div>
              <p class="text-sm text-stone-500">Gearbeitete Stunden</p>
              <p class="text-2xl font-bold text-stone-900">{{ formatHours(overview?.totalWorkedHours) }}</p>
            </div>
          </div>
        </div>

        <div class="bg-white rounded-lg border border-stone-200 p-6">
          <div class="flex items-center gap-3">
            <div class="p-2 bg-amber-100 rounded-lg">
              <Calendar class="w-5 h-5 text-amber-600" />
            </div>
            <div>
              <p class="text-sm text-stone-500">Urlaub / Krank</p>
              <p class="text-2xl font-bold text-stone-900">
                {{ overview?.vacationDays ?? 0 }} / {{ overview?.sickDays ?? 0 }} Tage
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Employee Statistics Table -->
      <div class="bg-white rounded-lg border border-stone-200 overflow-hidden">
        <div class="px-6 py-4 border-b border-stone-200">
          <h2 class="text-lg font-semibold text-stone-900">Mitarbeiter-Übersicht</h2>
        </div>
        <div class="overflow-x-auto">
          <table class="w-full">
            <thead>
              <tr class="bg-stone-50 border-b border-stone-200">
                <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Mitarbeiter</th>
                <th class="px-4 py-3 text-right text-sm font-medium text-stone-600">Geplant</th>
                <th class="px-4 py-3 text-right text-sm font-medium text-stone-600">Gearbeitet</th>
                <th class="px-4 py-3 text-right text-sm font-medium text-stone-600">Überstunden</th>
                <th class="px-4 py-3 text-right text-sm font-medium text-stone-600">Resturlaub</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="stat in overview?.employeeStats"
                :key="stat.employee?.id"
                class="border-b border-stone-200 last:border-b-0 hover:bg-stone-50"
              >
                <td class="px-4 py-3">
                  <div class="flex items-center gap-3">
                    <div class="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
                      <span class="text-sm font-medium text-primary">
                        {{ stat.employee?.firstName?.[0] }}{{ stat.employee?.lastName?.[0] }}
                      </span>
                    </div>
                    <div>
                      <span class="font-medium text-stone-900">
                        {{ stat.employee?.firstName }} {{ stat.employee?.lastName }}
                      </span>
                      <div class="text-xs text-stone-500">{{ stat.employee?.weeklyHours }} Std./Woche</div>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3 text-right text-stone-600">
                  {{ formatHours(stat.scheduledHours) }}
                </td>
                <td class="px-4 py-3 text-right text-stone-600">
                  {{ formatHours(stat.workedHours) }}
                </td>
                <td class="px-4 py-3 text-right" :class="getOvertimeClass(stat.overtimeHours)">
                  <span v-if="stat.overtimeHours !== undefined && stat.overtimeHours > 0">+</span>{{ formatHours(stat.overtimeHours) }}
                </td>
                <td class="px-4 py-3 text-right text-stone-600">
                  {{ stat.remainingVacationDays ?? '-' }} Tage
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="!overview?.employeeStats?.length" class="p-6 text-center text-stone-500">
          Keine Statistiken für diesen Monat verfügbar.
        </div>
      </div>

      <!-- Weekly Capacity View -->
      <div class="bg-white rounded-lg border border-stone-200 overflow-hidden">
        <div class="px-6 py-4 border-b border-stone-200 flex items-center justify-between">
          <div class="flex items-center gap-2">
            <Target class="w-5 h-5 text-primary" />
            <h2 class="text-lg font-semibold text-stone-900">Wochen-Kapazität</h2>
          </div>
          <div class="flex items-center gap-2">
            <Button variant="outline" size="icon" @click="previousWeek">
              <ChevronLeft class="w-4 h-4" />
            </Button>
            <span class="text-sm text-stone-600 min-w-[200px] text-center">
              {{ formatDate(getWeekStart(currentWeek)) }} - {{ formatDate(new Date(getWeekStart(currentWeek).getTime() + 6 * 24 * 60 * 60 * 1000)) }}
            </span>
            <Button variant="outline" size="icon" @click="nextWeek">
              <ChevronRight class="w-4 h-4" />
            </Button>
          </div>
        </div>
        
        <div v-if="weeklyLoading" class="p-6 text-center">
          <Loader2 class="w-6 h-6 animate-spin text-primary mx-auto" />
        </div>
        
        <div v-else-if="weekly" class="p-6 space-y-6">
          <!-- Summary -->
          <div class="grid gap-4 md:grid-cols-3">
            <div class="p-4 bg-stone-50 rounded-lg">
              <p class="text-sm text-stone-500">Vertrags-Stunden (Summe)</p>
              <p class="text-xl font-bold text-stone-900">
                {{ formatHours(weekly.byEmployee?.reduce((sum, e) => sum + (e.employee?.weeklyHours || 0), 0)) }}
              </p>
            </div>
            <div class="p-4 bg-blue-50 rounded-lg">
              <p class="text-sm text-stone-500">Geplante Stunden</p>
              <p class="text-xl font-bold text-blue-700">{{ formatHours(weekly.totalScheduledHours) }}</p>
            </div>
            <div class="p-4 bg-green-50 rounded-lg">
              <p class="text-sm text-stone-500">Gearbeitete Stunden</p>
              <p class="text-xl font-bold text-green-700">{{ formatHours(weekly.totalWorkedHours) }}</p>
            </div>
          </div>

          <!-- Capacity per employee -->
          <div>
            <h3 class="font-medium text-stone-700 mb-3">Kapazitätsauslastung pro Mitarbeiter</h3>
            <div class="space-y-3">
              <div
                v-for="emp in weekly.byEmployee"
                :key="emp.employee?.id"
                class="p-4 bg-stone-50 rounded-lg"
              >
                <div class="flex items-center justify-between mb-2">
                  <div class="flex items-center gap-3">
                    <div class="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
                      <span class="text-xs font-medium text-primary">
                        {{ emp.employee?.firstName?.[0] }}{{ emp.employee?.lastName?.[0] }}
                      </span>
                    </div>
                    <div>
                      <span class="font-medium text-stone-900">
                        {{ emp.employee?.firstName }} {{ emp.employee?.lastName }}
                      </span>
                      <div class="text-xs text-stone-500">
                        Vertrag: {{ emp.employee?.weeklyHours ?? '-' }} Std./Woche
                      </div>
                    </div>
                  </div>
                  <div class="text-right">
                    <div class="text-sm font-medium text-stone-700">
                      {{ formatHours(emp.scheduledHours) }} geplant
                    </div>
                    <div class="text-xs" :class="getDifferenceClass(emp.scheduledHours, emp.employee?.weeklyHours)">
                      {{ getDifferenceText(emp.scheduledHours, emp.employee?.weeklyHours) }}
                    </div>
                  </div>
                </div>
                
                <!-- Progress bar -->
                <div class="h-3 bg-stone-200 rounded-full overflow-hidden">
                  <div
                    class="h-full rounded-full transition-all"
                    :class="getCapacityBarClass(getCapacityPercent(emp.scheduledHours, emp.employee?.weeklyHours))"
                    :style="{ width: `${Math.min(getCapacityPercent(emp.scheduledHours, emp.employee?.weeklyHours), 100)}%` }"
                  />
                </div>
                <div class="flex justify-between text-xs text-stone-500 mt-1">
                  <span>0%</span>
                  <span>{{ getCapacityPercent(emp.scheduledHours, emp.employee?.weeklyHours) }}%</span>
                  <span>100%</span>
                </div>
                
                <!-- Worked vs Scheduled if there's actual data -->
                <div v-if="emp.workedHours !== undefined && emp.workedHours > 0" class="mt-2 pt-2 border-t border-stone-200">
                  <div class="flex justify-between text-sm">
                    <span class="text-stone-600">Tatsächlich gearbeitet:</span>
                    <span class="font-medium text-green-700">{{ formatHours(emp.workedHours) }}</span>
                  </div>
                </div>
              </div>
            </div>
            
            <div v-if="!weekly.byEmployee?.length" class="text-center text-stone-500 py-4">
              Keine Daten für diese Woche verfügbar.
            </div>
          </div>

          <!-- Legend -->
          <div class="flex flex-wrap gap-4 pt-4 border-t border-stone-200">
            <div class="flex items-center gap-2 text-xs text-stone-600">
              <div class="w-3 h-3 rounded-full bg-amber-500"></div>
              <span>Unterplant (&lt;80%)</span>
            </div>
            <div class="flex items-center gap-2 text-xs text-stone-600">
              <div class="w-3 h-3 rounded-full bg-green-500"></div>
              <span>Optimal (80-105%)</span>
            </div>
            <div class="flex items-center gap-2 text-xs text-stone-600">
              <div class="w-3 h-3 rounded-full bg-blue-500"></div>
              <span>Leichte Überstunden (105-120%)</span>
            </div>
            <div class="flex items-center gap-2 text-xs text-stone-600">
              <div class="w-3 h-3 rounded-full bg-red-500"></div>
              <span>Überplant (&gt;120%)</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
