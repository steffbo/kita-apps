<script setup lang="ts">
import { Plus, Calendar } from 'lucide-vue-next';
import { formatDate } from '@kita/shared/utils';

const specialDays = [
  { id: 1, date: '2026-01-01', name: 'Neujahr', dayType: 'HOLIDAY', affectsAll: true },
  { id: 2, date: '2026-04-03', name: 'Karfreitag', dayType: 'HOLIDAY', affectsAll: true },
  { id: 3, date: '2026-04-06', name: 'Ostermontag', dayType: 'HOLIDAY', affectsAll: true },
  { id: 4, date: '2026-07-20', name: 'Sommerschließzeit Beginn', dayType: 'CLOSURE', affectsAll: true },
  { id: 5, date: '2026-08-07', name: 'Sommerschließzeit Ende', dayType: 'CLOSURE', affectsAll: true },
  { id: 6, date: '2026-09-15', name: 'Teamfortbildung', dayType: 'TEAM_DAY', affectsAll: true },
  { id: 7, date: '2026-11-11', name: 'Laternenumzug', dayType: 'EVENT', affectsAll: true },
];

function getTypeLabel(type: string) {
  const labels: Record<string, string> = {
    HOLIDAY: 'Feiertag',
    CLOSURE: 'Schließzeit',
    TEAM_DAY: 'Teamtag',
    EVENT: 'Veranstaltung',
  };
  return labels[type] || type;
}

function getTypeColor(type: string) {
  const colors: Record<string, string> = {
    HOLIDAY: 'bg-red-100 text-red-700',
    CLOSURE: 'bg-amber-100 text-amber-700',
    TEAM_DAY: 'bg-purple-100 text-purple-700',
    EVENT: 'bg-blue-100 text-blue-700',
  };
  return colors[type] || 'bg-stone-100 text-stone-700';
}
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-stone-900">Besondere Tage</h1>
        <p class="text-stone-600">Feiertage, Schließzeiten und Veranstaltungen</p>
      </div>
      <button class="flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-green-600 rounded-md hover:bg-green-700">
        <Plus class="w-4 h-4" />
        Neuer Eintrag
      </button>
    </div>

    <div class="bg-white rounded-lg border border-stone-200 overflow-hidden">
      <table class="w-full">
        <thead>
          <tr class="bg-stone-50 border-b border-stone-200">
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Datum</th>
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Bezeichnung</th>
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Art</th>
            <th class="px-4 py-3 text-right text-sm font-medium text-stone-600">Aktionen</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="day in specialDays"
            :key="day.id"
            class="border-b border-stone-200 last:border-b-0 hover:bg-stone-50"
          >
            <td class="px-4 py-3">
              <div class="flex items-center gap-2">
                <Calendar class="w-4 h-4 text-stone-400" />
                <span class="text-stone-900">{{ formatDate(day.date) }}</span>
              </div>
            </td>
            <td class="px-4 py-3 font-medium text-stone-900">{{ day.name }}</td>
            <td class="px-4 py-3">
              <span
                :class="[
                  'px-2 py-1 text-xs font-medium rounded-full',
                  getTypeColor(day.dayType)
                ]"
              >
                {{ getTypeLabel(day.dayType) }}
              </span>
            </td>
            <td class="px-4 py-3 text-right">
              <button
                v-if="day.dayType !== 'HOLIDAY'"
                class="text-sm text-green-600 hover:text-green-700 font-medium"
              >
                Bearbeiten
              </button>
              <span v-else class="text-sm text-stone-400">Automatisch</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
