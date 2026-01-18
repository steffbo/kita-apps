<script setup lang="ts">
import { ref, computed } from 'vue';
import { useAuth } from '@kita/shared';
import { getWeekStart, getWeekEnd, formatDate, WEEKDAYS_SHORT } from '@kita/shared/utils';
import { ChevronLeft, ChevronRight, Plus } from 'lucide-vue-next';

const { isAdmin } = useAuth();

// Current week state
const currentDate = ref(new Date());
const weekStart = computed(() => getWeekStart(currentDate.value));
const weekEnd = computed(() => getWeekEnd(currentDate.value));

// Navigation
function previousWeek() {
  const newDate = new Date(currentDate.value);
  newDate.setDate(newDate.getDate() - 7);
  currentDate.value = newDate;
}

function nextWeek() {
  const newDate = new Date(currentDate.value);
  newDate.setDate(newDate.getDate() + 7);
  currentDate.value = newDate;
}

function goToToday() {
  currentDate.value = new Date();
}

// Generate week days
const weekDays = computed(() => {
  const days = [];
  const start = new Date(weekStart.value);
  
  for (let i = 0; i < 7; i++) {
    const date = new Date(start);
    date.setDate(date.getDate() + i);
    days.push({
      date,
      dayName: WEEKDAYS_SHORT[i],
      dayNumber: date.getDate(),
      isToday: date.toDateString() === new Date().toDateString(),
      isWeekend: i >= 5,
    });
  }
  
  return days;
});

// Mock groups for now
const groups = ref([
  { id: 1, name: 'Sonnenkinder', color: '#F59E0B' },
  { id: 2, name: 'Mondkinder', color: '#6366F1' },
  { id: 3, name: 'Sternenkinder', color: '#10B981' },
]);

// Mock schedule entries
const scheduleEntries = ref([
  { id: 1, employeeId: 1, employeeName: 'Anna Müller', groupId: 1, date: '2026-01-19', startTime: '07:00', endTime: '14:00', type: 'WORK' },
  { id: 2, employeeId: 2, employeeName: 'Petra Schmidt', groupId: 1, date: '2026-01-19', startTime: '09:00', endTime: '16:00', type: 'WORK' },
  { id: 3, employeeId: 3, employeeName: 'Lisa Weber', groupId: 2, date: '2026-01-19', startTime: '07:30', endTime: '15:30', type: 'WORK' },
]);

function getEntriesForGroupAndDay(groupId: number, date: Date) {
  const dateStr = date.toISOString().split('T')[0];
  return scheduleEntries.value.filter(
    e => e.groupId === groupId && e.date === dateStr
  );
}
</script>

<template>
  <div>
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6">
      <div>
        <h1 class="text-2xl font-bold text-stone-900">Dienstplan</h1>
        <p class="text-stone-600">
          {{ formatDate(weekStart) }} - {{ formatDate(weekEnd) }}
        </p>
      </div>

      <div class="flex items-center gap-2">
        <button
          @click="goToToday"
          class="px-3 py-2 text-sm font-medium text-stone-700 bg-white border border-stone-300 rounded-md hover:bg-stone-50"
        >
          Heute
        </button>
        
        <div class="flex items-center bg-white border border-stone-300 rounded-md">
          <button
            @click="previousWeek"
            class="p-2 hover:bg-stone-50 rounded-l-md"
          >
            <ChevronLeft class="w-4 h-4" />
          </button>
          <button
            @click="nextWeek"
            class="p-2 hover:bg-stone-50 rounded-r-md border-l border-stone-300"
          >
            <ChevronRight class="w-4 h-4" />
          </button>
        </div>

        <button
          v-if="isAdmin"
          class="flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-green-600 rounded-md hover:bg-green-700"
        >
          <Plus class="w-4 h-4" />
          Eintrag
        </button>
      </div>
    </div>

    <!-- Calendar Grid -->
    <div class="bg-white rounded-lg border border-stone-200 overflow-hidden">
      <!-- Week header -->
      <div class="grid grid-cols-8 border-b border-stone-200">
        <div class="px-4 py-3 bg-stone-50 border-r border-stone-200">
          <span class="text-sm font-medium text-stone-600">Gruppe</span>
        </div>
        <div
          v-for="day in weekDays"
          :key="day.date.toISOString()"
          :class="[
            'px-4 py-3 text-center border-r border-stone-200 last:border-r-0',
            day.isWeekend ? 'bg-stone-100' : 'bg-stone-50',
            day.isToday ? 'bg-green-50' : ''
          ]"
        >
          <div class="text-sm font-medium text-stone-600">{{ day.dayName }}</div>
          <div
            :class="[
              'text-lg font-semibold',
              day.isToday ? 'text-green-700' : 'text-stone-900'
            ]"
          >
            {{ day.dayNumber }}
          </div>
        </div>
      </div>

      <!-- Groups rows -->
      <div
        v-for="group in groups"
        :key="group.id"
        class="grid grid-cols-8 border-b border-stone-200 last:border-b-0"
      >
        <!-- Group name -->
        <div class="px-4 py-4 bg-stone-50 border-r border-stone-200 flex items-center gap-2">
          <div
            class="w-3 h-3 rounded-full"
            :style="{ backgroundColor: group.color }"
          />
          <span class="text-sm font-medium text-stone-900">{{ group.name }}</span>
        </div>

        <!-- Day cells -->
        <div
          v-for="day in weekDays"
          :key="`${group.id}-${day.date.toISOString()}`"
          :class="[
            'px-2 py-2 border-r border-stone-200 last:border-r-0 min-h-[100px]',
            day.isWeekend ? 'bg-stone-50' : '',
            day.isToday ? 'bg-green-50/50' : ''
          ]"
        >
          <div class="space-y-1">
            <div
              v-for="entry in getEntriesForGroupAndDay(group.id, day.date)"
              :key="entry.id"
              class="px-2 py-1 rounded text-xs cursor-pointer hover:opacity-80 transition-opacity"
              :style="{ backgroundColor: group.color + '20', borderLeft: `3px solid ${group.color}` }"
            >
              <div class="font-medium text-stone-900 truncate">{{ entry.employeeName }}</div>
              <div class="text-stone-600">{{ entry.startTime }} - {{ entry.endTime }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- Springer row -->
      <div class="grid grid-cols-8 border-b border-stone-200">
        <div class="px-4 py-4 bg-stone-50 border-r border-stone-200 flex items-center gap-2">
          <div class="w-3 h-3 rounded-full bg-stone-400" />
          <span class="text-sm font-medium text-stone-900">Springer</span>
        </div>
        <div
          v-for="day in weekDays"
          :key="`springer-${day.date.toISOString()}`"
          :class="[
            'px-2 py-2 border-r border-stone-200 last:border-r-0 min-h-[80px]',
            day.isWeekend ? 'bg-stone-50' : '',
            day.isToday ? 'bg-green-50/50' : ''
          ]"
        >
          <!-- Springer entries would go here -->
        </div>
      </div>
    </div>

    <!-- Legend -->
    <div class="mt-4 flex flex-wrap gap-4 text-sm text-stone-600">
      <div class="flex items-center gap-2">
        <div class="w-3 h-3 rounded-full bg-green-500" />
        <span>Arbeit</span>
      </div>
      <div class="flex items-center gap-2">
        <div class="w-3 h-3 rounded-full bg-blue-500" />
        <span>Urlaub</span>
      </div>
      <div class="flex items-center gap-2">
        <div class="w-3 h-3 rounded-full bg-red-500" />
        <span>Krank</span>
      </div>
      <div class="flex items-center gap-2">
        <div class="w-3 h-3 rounded-full bg-purple-500" />
        <span>Fortbildung</span>
      </div>
      <div class="flex items-center gap-2">
        <div class="w-3 h-3 rounded-full bg-amber-500" />
        <span>Veranstaltung</span>
      </div>
    </div>
  </div>
</template>
