<script setup lang="ts">
import { Clock, Users, TrendingUp, Calendar } from 'lucide-vue-next';

const stats = [
  { label: 'Gesamtstunden (Monat)', value: '2.480 Std.', icon: Clock, change: '+2.5%' },
  { label: 'Aktive Mitarbeiter', value: '15', icon: Users, change: '0' },
  { label: 'Überstunden gesamt', value: '+48 Std.', icon: TrendingUp, change: '+12 Std.' },
  { label: 'Resturlaub gesamt', value: '187 Tage', icon: Calendar, change: '-23 Tage' },
];

const employeeStats = [
  { name: 'Anna Müller', scheduledHours: 152, workedHours: 158, overtime: 6, vacation: 24 },
  { name: 'Petra Schmidt', scheduledHours: 120, workedHours: 118, overtime: -2, vacation: 28 },
  { name: 'Lisa Weber', scheduledHours: 100, workedHours: 104, overtime: 4, vacation: 22 },
  { name: 'Maria Braun', scheduledHours: 152, workedHours: 160, overtime: 8, vacation: 20 },
];
</script>

<template>
  <div>
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-stone-900">Statistiken</h1>
      <p class="text-stone-600">Übersicht über Arbeitszeiten und Abwesenheiten</p>
    </div>

    <!-- Overview cards -->
    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4 mb-8">
      <div
        v-for="stat in stats"
        :key="stat.label"
        class="bg-white rounded-lg border border-stone-200 p-6"
      >
        <div class="flex items-center justify-between mb-4">
          <component
            :is="stat.icon"
            class="w-8 h-8 text-green-600 p-1.5 bg-green-50 rounded-lg"
          />
          <span
            :class="[
              'text-sm font-medium',
              stat.change.startsWith('+') ? 'text-green-600' : stat.change.startsWith('-') ? 'text-red-600' : 'text-stone-500'
            ]"
          >
            {{ stat.change }}
          </span>
        </div>
        <div class="text-2xl font-bold text-stone-900">{{ stat.value }}</div>
        <div class="text-sm text-stone-500">{{ stat.label }}</div>
      </div>
    </div>

    <!-- Employee table -->
    <div class="bg-white rounded-lg border border-stone-200 overflow-hidden">
      <div class="px-6 py-4 border-b border-stone-200">
        <h2 class="font-semibold text-stone-900">Mitarbeiter-Übersicht (Januar 2026)</h2>
      </div>
      <table class="w-full">
        <thead>
          <tr class="bg-stone-50 border-b border-stone-200">
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Mitarbeiter</th>
            <th class="px-4 py-3 text-right text-sm font-medium text-stone-600">Soll-Stunden</th>
            <th class="px-4 py-3 text-right text-sm font-medium text-stone-600">Ist-Stunden</th>
            <th class="px-4 py-3 text-right text-sm font-medium text-stone-600">Differenz</th>
            <th class="px-4 py-3 text-right text-sm font-medium text-stone-600">Resturlaub</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="employee in employeeStats"
            :key="employee.name"
            class="border-b border-stone-200 last:border-b-0 hover:bg-stone-50"
          >
            <td class="px-4 py-3 font-medium text-stone-900">{{ employee.name }}</td>
            <td class="px-4 py-3 text-right text-stone-600">{{ employee.scheduledHours }} Std.</td>
            <td class="px-4 py-3 text-right text-stone-600">{{ employee.workedHours }} Std.</td>
            <td class="px-4 py-3 text-right">
              <span
                :class="[
                  'font-medium',
                  employee.overtime > 0 ? 'text-green-600' : employee.overtime < 0 ? 'text-red-600' : 'text-stone-600'
                ]"
              >
                {{ employee.overtime > 0 ? '+' : '' }}{{ employee.overtime }} Std.
              </span>
            </td>
            <td class="px-4 py-3 text-right text-stone-600">{{ employee.vacation }} Tage</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
