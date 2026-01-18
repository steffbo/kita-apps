<script setup lang="ts">
import { ref } from 'vue';
import { formatDate, formatTime, formatDuration } from '@kita/shared/utils';
import { ChevronLeft, ChevronRight } from 'lucide-vue-next';

const currentMonth = ref(new Date());

// Mock data
const entries = [
  { id: 1, date: '2026-01-16', clockIn: '2026-01-16T07:02:00', clockOut: '2026-01-16T14:05:00', breakMinutes: 30, scheduled: { start: '07:00', end: '14:00' } },
  { id: 2, date: '2026-01-15', clockIn: '2026-01-15T06:58:00', clockOut: '2026-01-15T14:02:00', breakMinutes: 30, scheduled: { start: '07:00', end: '14:00' } },
  { id: 3, date: '2026-01-14', clockIn: '2026-01-14T07:05:00', clockOut: '2026-01-14T14:10:00', breakMinutes: 30, scheduled: { start: '07:00', end: '14:00' } },
  { id: 4, date: '2026-01-13', clockIn: '2026-01-13T09:00:00', clockOut: '2026-01-13T16:00:00', breakMinutes: 30, scheduled: { start: '09:00', end: '16:00' } },
];

function calculateWorked(clockIn: string, clockOut: string, breakMinutes: number) {
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

    <!-- Summary -->
    <div class="grid grid-cols-3 gap-4 mb-6">
      <div class="bg-white rounded-lg border border-stone-200 p-4">
        <p class="text-sm text-stone-500">Soll-Stunden</p>
        <p class="text-2xl font-bold text-stone-900">152 Std.</p>
      </div>
      <div class="bg-white rounded-lg border border-stone-200 p-4">
        <p class="text-sm text-stone-500">Ist-Stunden</p>
        <p class="text-2xl font-bold text-stone-900">48 Std.</p>
      </div>
      <div class="bg-white rounded-lg border border-stone-200 p-4">
        <p class="text-sm text-stone-500">Differenz</p>
        <p class="text-2xl font-bold text-green-600">+2 Std.</p>
      </div>
    </div>

    <!-- Entries Table -->
    <div class="bg-white rounded-lg border border-stone-200 overflow-hidden">
      <table class="w-full">
        <thead>
          <tr class="bg-stone-50 border-b border-stone-200">
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Datum</th>
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Soll</th>
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Kommen</th>
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Gehen</th>
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Pause</th>
            <th class="px-4 py-3 text-right text-sm font-medium text-stone-600">Arbeitszeit</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="entry in entries"
            :key="entry.id"
            class="border-b border-stone-200 last:border-b-0 hover:bg-stone-50"
          >
            <td class="px-4 py-3 font-medium text-stone-900">
              {{ formatDate(entry.date) }}
            </td>
            <td class="px-4 py-3 text-stone-600">
              {{ entry.scheduled.start }} - {{ entry.scheduled.end }}
            </td>
            <td class="px-4 py-3 text-stone-900">
              {{ formatTime(entry.clockIn) }}
            </td>
            <td class="px-4 py-3 text-stone-900">
              {{ formatTime(entry.clockOut) }}
            </td>
            <td class="px-4 py-3 text-stone-600">
              {{ entry.breakMinutes }} Min.
            </td>
            <td class="px-4 py-3 text-right font-medium text-stone-900">
              {{ formatDuration(calculateWorked(entry.clockIn, entry.clockOut, entry.breakMinutes)) }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
