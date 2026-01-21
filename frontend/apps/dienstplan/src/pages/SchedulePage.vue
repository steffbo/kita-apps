<script setup lang="ts">
import { ref, computed } from 'vue';
import { ChevronLeft, ChevronRight, Plus, Loader2 } from 'lucide-vue-next';
import { 
  useAuth,
  useWeekSchedule,
  useGroups,
  useEmployees,
  useCreateScheduleEntry,
  useUpdateScheduleEntry,
  useDeleteScheduleEntry,
  type ScheduleEntry,
  type CreateScheduleEntryRequest,
  type UpdateScheduleEntryRequest
} from '@kita/shared';
import { getWeekStart, getWeekEnd, formatDate, WEEKDAYS_SHORT, toISODateString } from '@kita/shared/utils';
import { Button } from '@/components/ui';
import ScheduleEntryDialog from '@/components/ScheduleEntryDialog.vue';

const { isAdmin } = useAuth();

// Current week state
const currentDate = ref(new Date());
const weekStart = computed(() => getWeekStart(currentDate.value));
const weekEnd = computed(() => getWeekEnd(currentDate.value));

// API queries
const { data: weekSchedule, isLoading: scheduleLoading, error: scheduleError, refetch: refetchSchedule } = useWeekSchedule(weekStart);
const { data: groups, isLoading: groupsLoading } = useGroups();
const { data: employees } = useEmployees(false);

// Mutations
const createEntry = useCreateScheduleEntry();
const updateEntry = useUpdateScheduleEntry();
const deleteEntry = useDeleteScheduleEntry();

// Dialog state
const dialogOpen = ref(false);
const selectedEntry = ref<ScheduleEntry | null>(null);
const defaultDate = ref<Date | undefined>();
const defaultGroupId = ref<number | undefined>();

// Display settings
const showWeekends = ref(false);

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
const allWeekDays = computed(() => {
  const days = [];
  const start = new Date(weekStart.value);
  
  for (let i = 0; i < 7; i++) {
    const date = new Date(start);
    date.setDate(date.getDate() + i);
    
    // Find special day info from schedule data
    const daySchedule = weekSchedule.value?.days?.find(
      d => d.date === toISODateString(date)
    );
    
    days.push({
      date,
      dateStr: toISODateString(date),
      dayName: WEEKDAYS_SHORT[i],
      dayNumber: date.getDate(),
      isToday: date.toDateString() === new Date().toDateString(),
      isWeekend: i >= 5,
      isHoliday: daySchedule?.isHoliday || false,
      holidayName: daySchedule?.holidayName,
    });
  }
  
  return days;
});

// Filtered week days based on showWeekends
const weekDays = computed(() => {
  if (showWeekends.value) {
    return allWeekDays.value;
  }
  return allWeekDays.value.filter(d => !d.isWeekend);
});

// Dynamic grid columns based on whether weekends are shown
const gridColsClass = computed(() => {
  return showWeekends.value ? 'grid-cols-8' : 'grid-cols-6';
});

// Get entries for a group on a specific day
function getEntriesForGroupAndDay(groupId: number, dateStr: string): ScheduleEntry[] {
  const daySchedule = weekSchedule.value?.days?.find(d => d.date === dateStr);
  if (!daySchedule) return [];
  
  // Check byGroup first for performance
  if (daySchedule.byGroup && daySchedule.byGroup[String(groupId)]) {
    return daySchedule.byGroup[String(groupId)] || [];
  }
  
  // Fallback: filter from all entries
  return (daySchedule.entries || []).filter(e => e.groupId === groupId);
}

// Get color for entry type
function getEntryColor(entryType: string, groupColor?: string): string {
  switch (entryType) {
    case 'VACATION': return '#3B82F6'; // blue
    case 'SICK': return '#EF4444'; // red
    case 'TRAINING': return '#8B5CF6'; // purple
    case 'EVENT': return '#F59E0B'; // amber
    case 'SPECIAL_LEAVE': return '#EC4899'; // pink
    default: return groupColor || '#10B981'; // green or group color
  }
}

// Dialog handlers
function openCreateDialog(date: Date, groupId?: number) {
  selectedEntry.value = null;
  defaultDate.value = date;
  defaultGroupId.value = groupId;
  dialogOpen.value = true;
}

function openEditDialog(entry: ScheduleEntry) {
  selectedEntry.value = entry;
  defaultDate.value = undefined;
  defaultGroupId.value = undefined;
  dialogOpen.value = true;
}

async function handleSave(data: CreateScheduleEntryRequest | UpdateScheduleEntryRequest, id?: number) {
  try {
    if (id) {
      await updateEntry.mutateAsync({ id, data: data as UpdateScheduleEntryRequest });
    } else {
      await createEntry.mutateAsync(data as CreateScheduleEntryRequest);
    }
    dialogOpen.value = false;
  } catch (err) {
    console.error('Failed to save entry:', err);
  }
}

async function handleDelete(id: number) {
  try {
    await deleteEntry.mutateAsync(id);
    dialogOpen.value = false;
  } catch (err) {
    console.error('Failed to delete entry:', err);
  }
}

const isLoading = computed(() => scheduleLoading.value || groupsLoading.value);

// Calculate weekly hours per employee
const employeeWeeklyHours = computed(() => {
  if (!employees.value || !weekSchedule.value) return [];
  
  return employees.value.map(emp => {
    // Get all entries for this employee this week
    const entries = weekSchedule.value?.days?.flatMap(day => 
      (day.entries || []).filter(e => e.employeeId === emp.id && e.entryType === 'WORK')
    ) || [];
    
    // Calculate planned minutes
    let plannedMinutes = 0;
    for (const entry of entries) {
      if (entry.startTime && entry.endTime) {
        const start = parseTime(entry.startTime);
        const end = parseTime(entry.endTime);
        const breakMins = entry.breakMinutes || 0;
        plannedMinutes += (end - start) - breakMins;
      }
    }
    
    const plannedHours = plannedMinutes / 60;
    const contractedHours = emp.weeklyHours || 0;
    const remainingHours = contractedHours - plannedHours;
    
    return {
      employee: emp,
      contractedHours,
      plannedHours,
      remainingHours,
    };
  }).filter(e => e.contractedHours > 0); // Only show employees with contracted hours
});

// Parse time string "HH:mm:ss" or "HH:mm" to minutes since midnight
function parseTime(timeStr: string): number {
  const parts = timeStr.split(':');
  return parseInt(parts[0]) * 60 + parseInt(parts[1]);
}

// Get status color class for remaining hours
function getHoursStatusClass(remaining: number): string {
  if (remaining < -2) return 'text-red-600 font-medium'; // Over-scheduled
  if (remaining < 0) return 'text-amber-600'; // Slightly over
  if (remaining > 2) return 'text-amber-600'; // Under-scheduled
  return 'text-green-600'; // Good
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
        <Button variant="outline" size="sm" @click="goToToday">
          Heute
        </Button>
        
        <div class="flex items-center">
          <Button variant="outline" size="icon" @click="previousWeek" class="rounded-r-none">
            <ChevronLeft class="w-4 h-4" />
          </Button>
          <Button variant="outline" size="icon" @click="nextWeek" class="rounded-l-none border-l-0">
            <ChevronRight class="w-4 h-4" />
          </Button>
        </div>

        <Button 
          variant="outline" 
          size="sm" 
          @click="showWeekends = !showWeekends"
          :class="showWeekends ? 'bg-stone-100' : ''"
        >
          {{ showWeekends ? 'Mo-So' : 'Mo-Fr' }}
        </Button>

        <Button v-if="isAdmin" @click="openCreateDialog(new Date())">
          <Plus class="w-4 h-4 mr-2" />
          Eintrag
        </Button>
      </div>
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="w-8 h-8 animate-spin text-primary" />
    </div>

    <!-- Error state -->
    <div v-else-if="scheduleError" class="bg-destructive/10 text-destructive rounded-lg p-4">
      <p>Fehler beim Laden des Dienstplans: {{ (scheduleError as Error).message }}</p>
      <Button variant="outline" size="sm" class="mt-2" @click="refetchSchedule()">
        Erneut versuchen
      </Button>
    </div>

    <!-- Calendar Grid -->
    <div v-else class="bg-white rounded-lg border border-stone-200 overflow-hidden">
      <!-- Week header -->
      <div :class="['grid border-b border-stone-200', gridColsClass]">
        <div class="px-4 py-3 bg-stone-50 border-r border-stone-200">
          <span class="text-sm font-medium text-stone-600">Gruppe</span>
        </div>
        <div
          v-for="day in weekDays"
          :key="day.dateStr"
          :class="[
            'px-4 py-3 text-center border-r border-stone-200 last:border-r-0',
            day.isWeekend ? 'bg-stone-100' : 'bg-stone-50',
            day.isToday ? 'bg-primary/10' : '',
            day.isHoliday ? 'bg-red-50' : ''
          ]"
        >
          <div class="text-sm font-medium text-stone-600">{{ day.dayName }}</div>
          <div
            :class="[
              'text-lg font-semibold',
              day.isToday ? 'text-primary' : 'text-stone-900',
              day.isHoliday ? 'text-red-600' : ''
            ]"
          >
            {{ day.dayNumber }}
          </div>
          <div v-if="day.isHoliday" class="text-xs text-red-600 truncate">
            {{ day.holidayName }}
          </div>
        </div>
      </div>

      <!-- Groups rows -->
      <div
        v-for="group in groups"
        :key="group.id"
        :class="['grid border-b border-stone-200 last:border-b-0', gridColsClass]"
      >
        <!-- Group name -->
        <div class="px-4 py-4 bg-stone-50 border-r border-stone-200 flex items-center gap-2">
          <div
            class="w-3 h-3 rounded-full"
            :style="{ backgroundColor: group.color || '#10B981' }"
          />
          <span class="text-sm font-medium text-stone-900">{{ group.name }}</span>
        </div>

        <!-- Day cells -->
        <div
          v-for="day in weekDays"
          :key="`${group.id}-${day.dateStr}`"
          :class="[
            'px-2 py-2 border-r border-stone-200 last:border-r-0 min-h-[100px] cursor-pointer hover:bg-stone-50/50',
            day.isWeekend ? 'bg-stone-50' : '',
            day.isToday ? 'bg-primary/5' : '',
            day.isHoliday ? 'bg-red-50/50' : ''
          ]"
          @click="isAdmin && openCreateDialog(day.date, group.id)"
        >
          <div class="space-y-1">
            <div
              v-for="entry in getEntriesForGroupAndDay(group.id!, day.dateStr)"
              :key="entry.id"
              class="px-2 py-1 rounded text-xs cursor-pointer hover:opacity-80 transition-opacity"
              :style="{ 
                backgroundColor: getEntryColor(entry.entryType || 'WORK', group.color) + '20', 
                borderLeft: `3px solid ${getEntryColor(entry.entryType || 'WORK', group.color)}` 
              }"
              @click.stop="openEditDialog(entry)"
            >
              <div class="font-medium text-stone-900 truncate">
                {{ entry.employee?.firstName }} {{ entry.employee?.lastName }}
              </div>
              <div class="text-stone-600" v-if="entry.entryType === 'WORK'">
                {{ entry.startTime?.substring(0, 5) }} - {{ entry.endTime?.substring(0, 5) }}
              </div>
              <div class="text-stone-600" v-else>
                {{ entry.entryType === 'VACATION' ? 'Urlaub' : 
                   entry.entryType === 'SICK' ? 'Krank' : 
                   entry.entryType === 'TRAINING' ? 'Fortbildung' :
                   entry.entryType === 'EVENT' ? 'Veranstaltung' : entry.entryType }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Weekly Hours Summary Table -->
    <div v-if="employeeWeeklyHours.length > 0" class="mt-6 bg-white rounded-lg border border-stone-200 overflow-hidden">
      <div class="px-4 py-3 bg-stone-50 border-b border-stone-200">
        <h2 class="text-sm font-semibold text-stone-900">Wochenstunden-Übersicht</h2>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-stone-200">
              <th class="px-4 py-2 text-left font-medium text-stone-600">Mitarbeiter</th>
              <th class="px-4 py-2 text-right font-medium text-stone-600">Vertrag</th>
              <th class="px-4 py-2 text-right font-medium text-stone-600">Geplant</th>
              <th class="px-4 py-2 text-right font-medium text-stone-600">Offen</th>
            </tr>
          </thead>
          <tbody>
            <tr 
              v-for="row in employeeWeeklyHours" 
              :key="row.employee.id"
              class="border-b border-stone-100 last:border-b-0 hover:bg-stone-50"
            >
              <td class="px-4 py-2">
                <div class="flex items-center gap-2">
                  <div 
                    class="w-2 h-2 rounded-full"
                    :style="{ backgroundColor: row.employee.primaryGroup?.color || '#9CA3AF' }"
                  />
                  <span class="text-stone-900">{{ row.employee.firstName }} {{ row.employee.lastName }}</span>
                </div>
              </td>
              <td class="px-4 py-2 text-right text-stone-600">
                {{ row.contractedHours.toFixed(1) }} Std.
              </td>
              <td class="px-4 py-2 text-right text-stone-600">
                {{ row.plannedHours.toFixed(1) }} Std.
              </td>
              <td class="px-4 py-2 text-right" :class="getHoursStatusClass(row.remainingHours)">
                {{ row.remainingHours >= 0 ? '+' : '' }}{{ row.remainingHours.toFixed(1) }} Std.
              </td>
            </tr>
          </tbody>
        </table>
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

    <!-- Schedule Entry Dialog -->
    <ScheduleEntryDialog
      v-model:open="dialogOpen"
      :entry="selectedEntry"
      :employees="employees || []"
      :groups="groups || []"
      :default-date="defaultDate"
      :default-group-id="defaultGroupId"
      @save="handleSave"
      @delete="handleDelete"
    />
  </div>
</template>
