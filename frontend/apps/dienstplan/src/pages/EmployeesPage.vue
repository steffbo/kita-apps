<script setup lang="ts">
import { Users, Plus } from 'lucide-vue-next';

// Placeholder data
const employees = [
  { id: 1, firstName: 'Anna', lastName: 'Müller', email: 'anna.mueller@knirpsenstadt.de', role: 'EMPLOYEE', weeklyHours: 38, active: true },
  { id: 2, firstName: 'Petra', lastName: 'Schmidt', email: 'petra.schmidt@knirpsenstadt.de', role: 'EMPLOYEE', weeklyHours: 30, active: true },
  { id: 3, firstName: 'Lisa', lastName: 'Weber', email: 'lisa.weber@knirpsenstadt.de', role: 'EMPLOYEE', weeklyHours: 25, active: true },
  { id: 4, firstName: 'Maria', lastName: 'Braun', email: 'maria.braun@knirpsenstadt.de', role: 'ADMIN', weeklyHours: 38, active: true },
];
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-stone-900">Mitarbeiter</h1>
        <p class="text-stone-600">Verwalten Sie alle Mitarbeiter der Kita</p>
      </div>
      <button class="flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-green-600 rounded-md hover:bg-green-700">
        <Plus class="w-4 h-4" />
        Neuer Mitarbeiter
      </button>
    </div>

    <div class="bg-white rounded-lg border border-stone-200 overflow-hidden">
      <table class="w-full">
        <thead>
          <tr class="bg-stone-50 border-b border-stone-200">
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Name</th>
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">E-Mail</th>
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Rolle</th>
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Wochenstunden</th>
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Status</th>
            <th class="px-4 py-3 text-right text-sm font-medium text-stone-600">Aktionen</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="employee in employees"
            :key="employee.id"
            class="border-b border-stone-200 last:border-b-0 hover:bg-stone-50"
          >
            <td class="px-4 py-3">
              <div class="flex items-center gap-3">
                <div class="w-8 h-8 rounded-full bg-green-100 flex items-center justify-center">
                  <span class="text-sm font-medium text-green-700">
                    {{ employee.firstName[0] }}{{ employee.lastName[0] }}
                  </span>
                </div>
                <span class="font-medium text-stone-900">
                  {{ employee.firstName }} {{ employee.lastName }}
                </span>
              </div>
            </td>
            <td class="px-4 py-3 text-stone-600">{{ employee.email }}</td>
            <td class="px-4 py-3">
              <span
                :class="[
                  'px-2 py-1 text-xs font-medium rounded-full',
                  employee.role === 'ADMIN'
                    ? 'bg-purple-100 text-purple-700'
                    : 'bg-stone-100 text-stone-700'
                ]"
              >
                {{ employee.role === 'ADMIN' ? 'Leitung' : 'Mitarbeiter' }}
              </span>
            </td>
            <td class="px-4 py-3 text-stone-600">{{ employee.weeklyHours }} Std.</td>
            <td class="px-4 py-3">
              <span
                :class="[
                  'px-2 py-1 text-xs font-medium rounded-full',
                  employee.active
                    ? 'bg-green-100 text-green-700'
                    : 'bg-stone-100 text-stone-500'
                ]"
              >
                {{ employee.active ? 'Aktiv' : 'Inaktiv' }}
              </span>
            </td>
            <td class="px-4 py-3 text-right">
              <button class="text-sm text-green-600 hover:text-green-700 font-medium">
                Bearbeiten
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
