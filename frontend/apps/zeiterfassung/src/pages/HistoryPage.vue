<script setup lang="ts">
import { ref, computed } from 'vue';
import { useTimeEntries, useTimeScheduleComparison } from '@kita/shared';
import { formatDate, formatTime, formatDuration } from '@kita/shared/utils';
import { ChevronLeft, ChevronRight, Loader2 } from 'lucide-vue-next';

const currentMonth = ref(new Date());

// Compute start and end of month for API query
const startDate = computed(() => {
  const d = new Date(currentMonth.value);
  d.setDate(1);
  d.setHours(0, 0, 0, 0);
  return d;
});

const endDate = computed(() => {
  const d = new Date(currentMonth.value);
  d.setMonth(d.getMonth() + 1);
  d.setDate(0); // Last day of current month
  d.setHours(23, 59, 59, 999);
  return d;
});

// Fetch time entries and comparison data
const { data: entries, isLoading: entriesLoading } = useTimeEntries({
  startDate,
  endDate,
});

const { data: comparison, isLoading: comparisonLoading } = useTimeScheduleComparison({
  startDate,
  endDate,
});

const isLoading = computed(() => entriesLoading.value || comparisonLoading.value);

// Sort entries by date descending (newest first)
const sortedEntries = computed(() => {
  if (!entries.value) return [];
  return [...entries.value].sort((a, b) => {
    const dateA = new Date(a.date || 0).getTime();
    const dateB = new Date(b.date || 0).getTime();
    return dateB - dateA;
  });
});

function calculateWorked(clockIn: string | undefined, clockOut: string | undefined, breakMinutes: number) {
  if (!clockIn || !clockOut) return 0;
  const start = new Date(clockIn);
  const end = new Date(clockOut);
  const diff = (end.getTime() - start.getTime()) / 60000;
  return diff - breakMinutes;
}

function previousMonth() {
  const d = new Date(currentMonth.value);
  d.setMonth(d.getMonth() - 1);
  currentMonth.value = d;
}

function nextMonth() {
  const d = new Date(currentMonth.value);
  d.setMonth(d.getMonth() + 1);
  currentMonth.value = d;
}

// Format hours for display (e.g., "152 Std.")
function formatHours(minutes: number | undefined): string {
  if (minutes === undefined || minutes === null) return '0 Std.';
  const hours = Math.round(minutes / 60 * 10) / 10;
  return `${hours} Std.`;
}

// Calculate difference and format with sign
function formatDifference(actual: number | undefined, scheduled: number | undefined): { text: string; color: string } {
  const actualMinutes = actual || 0;
  const scheduledMinutes = scheduled || 0;
  const diff = actualMinutes - scheduledMinutes;
  const hours = Math.round(diff / 60 * 10) / 10;
  
  if (hours >= 0) {
    return { text: `+${hours} Std.`, color: 'text-green-600' };
  } else {
    return { text: `${hours} Std.`, color: 'text-red-600' };
  }
}

// Check if entry was edited
function isEdited(entry: { editedBy?: number }): boolean {
  return entry.editedBy !== undefined && entry.editedBy !== null;
}
</script>

<template>
  <div>
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-stone-900">Zeitübersicht</h1>
        <p class="text-stone-600">Deine erfassten Arbeitszeiten</p>
      </div>
      
      <div class="flex items-center gap-2">
        <button
          @click="previousMonth"
          class="p-2 hover:bg-stone-100 rounded-md"
        >
          <ChevronLeft class="w-5 h-5" />
        </button>
        <span class="font-medium text-stone-900 min-w-[140px] text-center">
          {{ currentMonth.toLocaleDateString('de-DE', { month: 'long', year: 'numeric' }) }}
        </span>
        <button
          @click="nextMonth"
          class="p-2 hover:bg-stone-100 rounded-md"
        >
          <ChevronRight class="w-5 h-5" />
        </button>
      </div>
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="w-8 h-8 animate-spin text-green-600" />
    </div>

    <template v-else>
      <!-- Summary -->
      <div class="grid grid-cols-3 gap-4 mb-6">
        <div class="bg-white rounded-lg border border-stone-200 p-4">
          <p class="text-sm text-stone-500">Soll-Stunden</p>
          <p class="text-2xl font-bold text-stone-900">
            {{ formatHours(comparison?.summary?.totalScheduledMinutes) }}
          </p>
        </div>
        <div class="bg-white rounded-lg border border-stone-200 p-4">
          <p class="text-sm text-stone-500">Ist-Stunden</p>
          <p class="text-2xl font-bold text-stone-900">
            {{ formatHours(comparison?.summary?.totalActualMinutes) }}
          </p>
        </div>
        <div class="bg-white rounded-lg border border-stone-200 p-4">
          <p class="text-sm text-stone-500">Differenz</p>
          <p class="text-2xl font-bold" :class="formatDifference(comparison?.summary?.totalActualMinutes, comparison?.summary?.totalScheduledMinutes).color">
            {{ formatDifference(comparison?.summary?.totalActualMinutes, comparison?.summary?.totalScheduledMinutes).text }}
          </p>
        </div>
      </div>

      <!-- Empty state -->
      <div v-if="!sortedEntries.length" class="bg-white rounded-lg border border-stone-200 p-8 text-center">
        <p class="text-stone-500">Keine Zeiteinträge für diesen Monat vorhanden.</p>
      </div>

      <!-- Entries Table -->
      <div v-else class="bg-white rounded-lg border border-stone-200 overflow-hidden">
        <table class="w-full">
          <thead>
            <tr class="bg-stone-50 border-b border-stone-200">
              <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Datum</th>
              <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Kommen</th>
              <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Gehen</th>
              <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Pause</th>
              <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Status</th>
              <th class="px-4 py-3 text-right text-sm font-medium text-stone-600">Arbeitszeit</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="entry in sortedEntries"
              :key="entry.id"
              class="border-b border-stone-200 last:border-b-0 hover:bg-stone-50"
            >
              <td class="px-4 py-3 font-medium text-stone-900">
                {{ entry.date ? formatDate(entry.date) : '–' }}
              </td>
              <td class="px-4 py-3 text-stone-900">
                {{ entry.clockIn ? formatTime(entry.clockIn) : '–' }}
              </td>
              <td class="px-4 py-3 text-stone-900">
                <template v-if="entry.clockOut">
                  {{ formatTime(entry.clockOut) }}
                </template>
                <span v-else class="text-amber-600 text-sm">Aktiv</span>
              </td>
              <td class="px-4 py-3 text-stone-600">
                {{ entry.breakMinutes || 0 }} Min.
              </td>
              <td class="px-4 py-3">
                <span
                  :class="[
                    'px-2 py-1 text-xs font-medium rounded-full',
                    isEdited(entry)
                      ? 'bg-amber-100 text-amber-700'
                      : 'bg-green-100 text-green-700'
                  ]"
                >
                  {{ isEdited(entry) ? 'Bearbeitet' : 'Original' }}
                </span>
              </td>
              <td class="px-4 py-3 text-right font-medium text-stone-900">
                <template v-if="entry.clockOut">
                  {{ formatDuration(calculateWorked(entry.clockIn, entry.clockOut, entry.breakMinutes || 0)) }}
                </template>
                <span v-else class="text-stone-400">–</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
  </div>
</template>
