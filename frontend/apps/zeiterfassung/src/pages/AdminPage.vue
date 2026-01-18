<script setup lang="ts">
import { ref } from 'vue';
import { formatDate, formatTime } from '@kita/shared/utils';
import { Search, Edit, Download } from 'lucide-vue-next';

const searchQuery = ref('');
const selectedEmployee = ref<number | null>(null);

const employees = [
  { id: 1, name: 'Anna Müller' },
  { id: 2, name: 'Petra Schmidt' },
  { id: 3, name: 'Lisa Weber' },
  { id: 4, name: 'Maria Braun' },
];

const entries = [
  { id: 1, employeeName: 'Anna Müller', date: '2026-01-16', clockIn: '2026-01-16T07:02:00', clockOut: '2026-01-16T14:05:00', breakMinutes: 30, edited: false },
  { id: 2, employeeName: 'Petra Schmidt', date: '2026-01-16', clockIn: '2026-01-16T09:00:00', clockOut: '2026-01-16T16:05:00', breakMinutes: 30, edited: true },
  { id: 3, employeeName: 'Lisa Weber', date: '2026-01-16', clockIn: '2026-01-16T07:30:00', clockOut: '2026-01-16T15:35:00', breakMinutes: 30, edited: false },
];
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-stone-900">Verwaltung</h1>
        <p class="text-stone-600">Zeiteinträge aller Mitarbeiter verwalten</p>
      </div>

      <button class="flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-green-600 rounded-md hover:bg-green-700">
        <Download class="w-4 h-4" />
        Export
      </button>
    </div>

    <!-- Filters -->
    <div class="flex gap-4 mb-6">
      <div class="flex-1 relative">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-stone-400" />
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Suchen..."
          class="w-full pl-10 pr-4 py-2 border border-stone-300 rounded-md focus:outline-none focus:ring-2 focus:ring-green-500 focus:border-transparent"
        />
      </div>
      <select
        v-model="selectedEmployee"
        class="px-4 py-2 border border-stone-300 rounded-md focus:outline-none focus:ring-2 focus:ring-green-500 focus:border-transparent"
      >
        <option :value="null">Alle Mitarbeiter</option>
        <option v-for="emp in employees" :key="emp.id" :value="emp.id">
          {{ emp.name }}
        </option>
      </select>
    </div>

    <!-- Entries Table -->
    <div class="bg-white rounded-lg border border-stone-200 overflow-hidden">
      <table class="w-full">
        <thead>
          <tr class="bg-stone-50 border-b border-stone-200">
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Mitarbeiter</th>
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Datum</th>
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Kommen</th>
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Gehen</th>
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Pause</th>
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Status</th>
            <th class="px-4 py-3 text-right text-sm font-medium text-stone-600">Aktionen</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="entry in entries"
            :key="entry.id"
            class="border-b border-stone-200 last:border-b-0 hover:bg-stone-50"
          >
            <td class="px-4 py-3 font-medium text-stone-900">
              {{ entry.employeeName }}
            </td>
            <td class="px-4 py-3 text-stone-600">
              {{ formatDate(entry.date) }}
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
            <td class="px-4 py-3">
              <span
                :class="[
                  'px-2 py-1 text-xs font-medium rounded-full',
                  entry.edited
                    ? 'bg-amber-100 text-amber-700'
                    : 'bg-green-100 text-green-700'
                ]"
              >
                {{ entry.edited ? 'Bearbeitet' : 'Original' }}
              </span>
            </td>
            <td class="px-4 py-3 text-right">
              <button class="p-2 hover:bg-stone-100 rounded-md text-stone-600 hover:text-stone-900">
                <Edit class="w-4 h-4" />
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
