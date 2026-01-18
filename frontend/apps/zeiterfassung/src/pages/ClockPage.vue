<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { useAuth } from '@kita/shared';
import { formatTime, formatDuration } from '@kita/shared/utils';
import { Play, Square, Coffee } from 'lucide-vue-next';

const { user } = useAuth();

// Clock state
const isClockedIn = ref(false);
const clockInTime = ref<Date | null>(null);
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
  return Math.floor(diff / 60000) - breakMinutes.value; // in minutes
});

// Clock in/out handlers
function handleClockIn() {
  isClockedIn.value = true;
  clockInTime.value = new Date();
  breakMinutes.value = 0;
}

function handleClockOut() {
  // Would send to API
  isClockedIn.value = false;
  clockInTime.value = null;
}

function addBreak(minutes: number) {
  breakMinutes.value += minutes;
}
</script>

<template>
  <div class="max-w-2xl mx-auto">
    <!-- Header -->
    <div class="text-center mb-8">
      <h1 class="text-3xl font-bold text-stone-900">
        Hallo, {{ user?.firstName }}!
      </h1>
      <p class="text-stone-600 mt-2">
        {{ currentTime.toLocaleDateString('de-DE', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' }) }}
      </p>
    </div>

    <!-- Current Time -->
    <div class="text-center mb-8">
      <div class="text-6xl font-bold text-stone-900 font-mono">
        {{ formatTime(currentTime) }}
      </div>
    </div>

    <!-- Status Card -->
    <div class="bg-white rounded-xl border border-stone-200 p-8 mb-6">
      <div v-if="!isClockedIn" class="text-center">
        <div class="mb-6">
          <div class="w-16 h-16 mx-auto bg-stone-100 rounded-full flex items-center justify-center mb-4">
            <Play class="w-8 h-8 text-stone-400" />
          </div>
          <p class="text-stone-600">Du bist aktuell nicht eingestempelt</p>
        </div>

        <button
          @click="handleClockIn"
          class="w-full max-w-xs mx-auto flex items-center justify-center gap-3 px-8 py-4 text-lg font-semibold text-white bg-green-600 rounded-xl hover:bg-green-700 transition-colors shadow-lg shadow-green-200"
        >
          <Play class="w-6 h-6" />
          Einstempeln
        </button>
      </div>

      <div v-else class="text-center">
        <div class="mb-6">
          <div class="w-16 h-16 mx-auto bg-green-100 rounded-full flex items-center justify-center mb-4 animate-pulse">
            <div class="w-4 h-4 bg-green-500 rounded-full" />
          </div>
          <p class="text-green-700 font-medium">Eingestempelt seit {{ formatTime(clockInTime!) }}</p>
        </div>

        <!-- Worked Time -->
        <div class="mb-8">
          <div class="text-4xl font-bold text-stone-900 mb-2">
            {{ formatDuration(workedTime) }}
          </div>
          <p class="text-stone-500 text-sm">
            Arbeitszeit (ohne Pause)
          </p>
        </div>

        <!-- Break buttons -->
        <div class="flex justify-center gap-3 mb-6">
          <button
            @click="addBreak(15)"
            class="flex items-center gap-2 px-4 py-2 text-sm font-medium text-stone-700 bg-stone-100 rounded-lg hover:bg-stone-200 transition-colors"
          >
            <Coffee class="w-4 h-4" />
            +15 Min. Pause
          </button>
          <button
            @click="addBreak(30)"
            class="flex items-center gap-2 px-4 py-2 text-sm font-medium text-stone-700 bg-stone-100 rounded-lg hover:bg-stone-200 transition-colors"
          >
            <Coffee class="w-4 h-4" />
            +30 Min. Pause
          </button>
        </div>

        <p v-if="breakMinutes > 0" class="text-sm text-stone-500 mb-6">
          Pause: {{ breakMinutes }} Minuten
        </p>

        <button
          @click="handleClockOut"
          class="w-full max-w-xs mx-auto flex items-center justify-center gap-3 px-8 py-4 text-lg font-semibold text-white bg-red-600 rounded-xl hover:bg-red-700 transition-colors shadow-lg shadow-red-200"
        >
          <Square class="w-6 h-6" />
          Ausstempeln
        </button>
      </div>
    </div>

    <!-- Today's Schedule -->
    <div class="bg-white rounded-xl border border-stone-200 p-6">
      <h2 class="font-semibold text-stone-900 mb-4">Heutiger Dienstplan</h2>
      <div class="flex items-center justify-between p-4 bg-stone-50 rounded-lg">
        <div>
          <p class="font-medium text-stone-900">07:00 - 14:00 Uhr</p>
          <p class="text-sm text-stone-500">Sonnenkinder</p>
        </div>
        <div class="text-right">
          <p class="font-medium text-stone-900">7 Std.</p>
          <p class="text-sm text-stone-500">Soll-Zeit</p>
        </div>
      </div>
    </div>
  </div>
</template>
