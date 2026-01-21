<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { useAuth, useCurrentTimeEntry, useClockIn, useClockOut, useSchedule, useTimeEntries } from '@kita/shared';
import { formatTime, formatDuration, toISODateString } from '@kita/shared/utils';
import { Play, Square, Coffee, Loader2 } from 'lucide-vue-next';

const { user } = useAuth();

// API queries and mutations
const { data: currentEntry, isLoading } = useCurrentTimeEntry();
const clockIn = useClockIn();
const clockOut = useClockOut();

// Get Monday of current week
function getMonday(d: Date): Date {
  const date = new Date(d);
  const day = date.getDay();
  const diff = date.getDate() - day + (day === 0 ? -6 : 1);
  date.setDate(diff);
  date.setHours(0, 0, 0, 0);
  return date;
}

// Get Friday of current week
function getFriday(d: Date): Date {
  const monday = getMonday(d);
  const friday = new Date(monday);
  friday.setDate(monday.getDate() + 4);
  return friday;
}

const today = new Date();
const weekStart = getMonday(today);
const weekEnd = getFriday(today);

const employeeId = computed(() => user.value?.id);

// Fetch week's schedule for current user
const { data: weekSchedule, isLoading: scheduleLoading } = useSchedule({
  startDate: weekStart,
  endDate: weekEnd,
  employeeId,
});

// Fetch week's time entries for current user
const { data: weekTimeEntries, isLoading: timeEntriesLoading } = useTimeEntries({
  startDate: weekStart,
  endDate: weekEnd,
  employeeId,
});

// Calculate minutes from time string (HH:MM)
function parseTimeToMinutes(time: string): number {
  const [h, m] = time.split(':').map(Number);
  return h * 60 + m;
}

// Format minutes as hours (e.g., 450 -> "7:30" or "7.5")
function formatMinutesAsHours(minutes: number): string {
  const hours = Math.floor(minutes / 60);
  const mins = minutes % 60;
  if (mins === 0) return `${hours}`;
  return `${hours}:${mins.toString().padStart(2, '0')}`;
}

// Days of the week (Mon-Fri) with both planned and actual data
const weekDays = computed(() => {
  const days = [];
  const monday = getMonday(today);
  const dayNames = ['Mo', 'Di', 'Mi', 'Do', 'Fr'];
  
  for (let i = 0; i < 5; i++) {
    const date = new Date(monday);
    date.setDate(monday.getDate() + i);
    const dateStr = toISODateString(date);
    
    // Find scheduled entry for this day
    const scheduleEntry = weekSchedule.value?.find(e => e.date === dateStr);
    
    // Find time entry for this day
    const timeEntry = weekTimeEntries.value?.find(e => e.date === dateStr);
    
    // Calculate planned minutes
    let plannedMinutes = 0;
    if (scheduleEntry?.startTime && scheduleEntry?.endTime) {
      plannedMinutes = parseTimeToMinutes(scheduleEntry.endTime) - parseTimeToMinutes(scheduleEntry.startTime);
    }
    
    // Calculate actual minutes (use workedMinutes if available, otherwise calculate)
    let actualMinutes = 0;
    if (timeEntry?.workedMinutes) {
      actualMinutes = timeEntry.workedMinutes;
    } else if (timeEntry?.clockIn && timeEntry?.clockOut) {
      const clockInTime = new Date(timeEntry.clockIn).getTime();
      const clockOutTime = new Date(timeEntry.clockOut).getTime();
      actualMinutes = Math.floor((clockOutTime - clockInTime) / 60000) - (timeEntry.breakMinutes || 0);
    }
    
    const isToday = date.toDateString() === today.toDateString();
    const isFuture = date > today;
    
    days.push({
      name: dayNames[i],
      date: date.getDate(),
      dateStr,
      isToday,
      isFuture,
      scheduleEntry,
      timeEntry,
      plannedMinutes,
      actualMinutes,
    });
  }
  return days;
});

// Calculate totals for the week
const weekTotals = computed(() => {
  let totalPlanned = 0;
  let totalActual = 0;
  
  for (const day of weekDays.value) {
    totalPlanned += day.plannedMinutes;
    totalActual += day.actualMinutes;
  }
  
  return {
    planned: totalPlanned,
    actual: totalActual,
    diff: totalActual - totalPlanned,
  };
});

// Clock state derived from API
const isClockedIn = computed(() => !!currentEntry.value && !currentEntry.value.clockOut);
const clockInTime = computed(() => {
  if (!currentEntry.value?.clockIn) return null;
  return new Date(currentEntry.value.clockIn);
});

// Current time for display
const currentTime = ref(new Date());
const breakMinutes = ref(0);

// Update current time every second
let timeInterval: number;
onMounted(() => {
  timeInterval = setInterval(() => {
    currentTime.value = new Date();
  }, 1000) as unknown as number;
});

onUnmounted(() => {
  clearInterval(timeInterval);
});

// Calculate worked time
const workedTime = computed(() => {
  if (!isClockedIn.value || !clockInTime.value) return 0;
  const diff = currentTime.value.getTime() - clockInTime.value.getTime();
  const totalBreak = (currentEntry.value?.breakMinutes || 0) + breakMinutes.value;
  return Math.max(0, Math.floor(diff / 60000) - totalBreak);
});

// Clock in/out handlers
async function handleClockIn() {
  try {
    await clockIn.mutateAsync({});
    breakMinutes.value = 0;
  } catch (err) {
    console.error('Failed to clock in:', err);
  }
}

async function handleClockOut() {
  try {
    await clockOut.mutateAsync({
      breakMinutes: breakMinutes.value,
    });
    breakMinutes.value = 0;
  } catch (err) {
    console.error('Failed to clock out:', err);
  }
}

function addBreak(minutes: number) {
  breakMinutes.value += minutes;
}

// Loading state
const isProcessing = computed(() => clockIn.isPending.value || clockOut.isPending.value);
const isWeekLoading = computed(() => scheduleLoading.value || timeEntriesLoading.value);
</script>

<template>
  <div class="max-w-2xl mx-auto">
    <!-- Header -->
    <div class="text-center mb-6">
      <h1 class="text-2xl font-bold text-stone-900">
        Hallo, {{ user?.firstName }}!
      </h1>
      <p class="text-stone-600 mt-1 text-sm">
        {{ currentTime.toLocaleDateString('de-DE', { weekday: 'long', day: 'numeric', month: 'long' }) }}
      </p>
    </div>

    <!-- Current Time -->
    <div class="text-center mb-6">
      <div class="text-5xl font-bold text-stone-900 font-mono">
        {{ formatTime(currentTime) }}
      </div>
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-8">
      <Loader2 class="w-6 h-6 animate-spin text-green-600" />
    </div>

    <!-- Status Card -->
    <div v-else class="bg-white rounded-xl border border-stone-200 p-6 mb-6">
      <div v-if="!isClockedIn" class="text-center">
        <p class="text-stone-600 mb-4">Du bist aktuell nicht eingestempelt</p>
        <button
          @click="handleClockIn"
          :disabled="isProcessing"
          class="inline-flex items-center justify-center gap-2 px-6 py-3 text-base font-semibold text-white bg-green-600 rounded-lg hover:bg-green-700 transition-colors shadow-md shadow-green-200 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Loader2 v-if="clockIn.isPending.value" class="w-5 h-5 animate-spin" />
          <Play v-else class="w-5 h-5" />
          Einstempeln
        </button>
      </div>

      <div v-else class="text-center">
        <div class="flex items-center justify-center gap-2 mb-3">
          <div class="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
          <p class="text-green-700 text-sm font-medium">Eingestempelt seit {{ formatTime(clockInTime!) }}</p>
        </div>

        <!-- Worked Time -->
        <div class="text-3xl font-bold text-stone-900 mb-1">
          {{ formatDuration(workedTime) }}
        </div>
        <p class="text-stone-500 text-xs mb-4">Arbeitszeit (ohne Pause)</p>

        <!-- Break buttons -->
        <div class="flex justify-center gap-2 mb-3">
          <button
            @click="addBreak(15)"
            :disabled="isProcessing"
            class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-stone-700 bg-stone-100 rounded-md hover:bg-stone-200 transition-colors disabled:opacity-50"
          >
            <Coffee class="w-3.5 h-3.5" />
            +15 Min.
          </button>
          <button
            @click="addBreak(30)"
            :disabled="isProcessing"
            class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-stone-700 bg-stone-100 rounded-md hover:bg-stone-200 transition-colors disabled:opacity-50"
          >
            <Coffee class="w-3.5 h-3.5" />
            +30 Min.
          </button>
        </div>

        <p v-if="breakMinutes > 0 || currentEntry?.breakMinutes" class="text-xs text-stone-500 mb-4">
          Pause: {{ (currentEntry?.breakMinutes || 0) + breakMinutes }} Min.
        </p>

        <button
          @click="handleClockOut"
          :disabled="isProcessing"
          class="inline-flex items-center justify-center gap-2 px-6 py-3 text-base font-semibold text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors shadow-md shadow-red-200 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Loader2 v-if="clockOut.isPending.value" class="w-5 h-5 animate-spin" />
          <Square v-else class="w-5 h-5" />
          Ausstempeln
        </button>
      </div>
    </div>

    <!-- Week Schedule -->
    <div class="bg-white rounded-xl border border-stone-200 p-6">
      <div class="flex items-center justify-between mb-4">
        <h2 class="font-semibold text-stone-900">Wochenübersicht</h2>
        <div v-if="!isWeekLoading" class="text-sm">
          <span class="text-stone-500">{{ formatMinutesAsHours(weekTotals.actual) }}</span>
          <span class="text-stone-400"> / </span>
          <span class="text-stone-500">{{ formatMinutesAsHours(weekTotals.planned) }} Std.</span>
          <span 
            v-if="weekTotals.diff !== 0"
            :class="weekTotals.diff >= 0 ? 'text-green-600' : 'text-red-600'"
            class="ml-1"
          >
            ({{ weekTotals.diff >= 0 ? '+' : '' }}{{ formatMinutesAsHours(Math.abs(weekTotals.diff)) }})
          </span>
        </div>
      </div>
      
      <!-- Loading -->
      <div v-if="isWeekLoading" class="flex items-center justify-center py-6">
        <Loader2 class="w-5 h-5 animate-spin text-stone-400" />
      </div>
      
      <!-- Week grid -->
      <div v-else class="grid grid-cols-5 gap-2">
        <div
          v-for="day in weekDays"
          :key="day.dateStr"
          :class="[
            'rounded-lg p-3 text-center',
            day.isToday ? 'bg-green-50 ring-2 ring-green-500' : 'bg-stone-50'
          ]"
        >
          <!-- Day header -->
          <div class="mb-2">
            <span :class="['text-xs font-medium', day.isToday ? 'text-green-700' : 'text-stone-500']">
              {{ day.name }}
            </span>
            <div :class="['text-lg font-semibold', day.isToday ? 'text-green-900' : 'text-stone-900']">
              {{ day.date }}
            </div>
          </div>
          
          <!-- Has schedule or time entry -->
          <div v-if="day.scheduleEntry || day.timeEntry" class="text-xs space-y-1">
            <!-- Planned (Soll) -->
            <div class="flex items-center justify-between px-1">
              <span class="text-stone-400">Soll</span>
              <span :class="day.isToday ? 'text-green-800' : 'text-stone-600'">
                {{ day.plannedMinutes > 0 ? formatMinutesAsHours(day.plannedMinutes) : '–' }}
              </span>
            </div>
            <!-- Actual (Ist) -->
            <div class="flex items-center justify-between px-1">
              <span class="text-stone-400">Ist</span>
              <span 
                v-if="day.actualMinutes > 0 || (day.isToday && isClockedIn)"
                :class="[
                  day.isToday ? 'text-green-800 font-medium' : 'text-stone-700',
                  day.actualMinutes > day.plannedMinutes && day.plannedMinutes > 0 ? 'text-blue-600' : '',
                  day.actualMinutes < day.plannedMinutes && !day.isFuture && day.plannedMinutes > 0 ? 'text-amber-600' : ''
                ]"
              >
                {{ day.isToday && isClockedIn ? '...' : formatMinutesAsHours(day.actualMinutes) }}
              </span>
              <span v-else class="text-stone-300">–</span>
            </div>
            <!-- Group name -->
            <div v-if="day.scheduleEntry?.group" class="mt-1 pt-1 border-t border-stone-200 text-[10px] text-stone-500 truncate" :title="day.scheduleEntry.group.name">
              {{ day.scheduleEntry.group.name }}
            </div>
          </div>
          
          <!-- No schedule -->
          <div v-else class="text-xs text-stone-400 py-2">
            Frei
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
