<script setup lang="ts">
import { ref, computed } from 'vue';
import { BarChart3, CalendarX, ChevronLeft, ChevronRight, Plus, Loader2 } from 'lucide-vue-next';
import { 
  useAuth,
  useWeekSchedule,
  useGroups,
  useEmployees,
  useCreateScheduleEntry,
  useUpdateScheduleEntry,
  useDeleteScheduleEntry,
  useUpdateEmployee,
  useCreateEmployeeContract,
  useUpdateEmployeeContract,
  type Employee,
  type EmployeeContractRequest,
  type CreateEmployeeRequest,
  type UpdateEmployeeRequest,
  type ScheduleEntry,
  type SpecialDay,
  type CreateScheduleEntryRequest,
  type UpdateScheduleEntryRequest
} from '@kita/shared';
import { getWeekStart, getWeekEnd, formatDate, WEEKDAYS_SHORT, toISODateString } from '@kita/shared/utils';
import { Button, Dialog } from '@/components/ui';
import ScheduleEntryDialog from '@/components/ScheduleEntryDialog.vue';
import EmployeeFormDialog from '@/components/EmployeeFormDialog.vue';

const { isAdmin } = useAuth();
type EntryType = 'WORK' | 'VACATION' | 'SICK' | 'CHILD_SICK' | 'RECOVERY_DAY' | 'SPECIAL_LEAVE' | 'TRAINING' | 'EVENT';

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
const updateEmployee = useUpdateEmployee();
const createEmployeeContract = useCreateEmployeeContract();
const updateEmployeeContract = useUpdateEmployeeContract();

// Dialog state
const dialogOpen = ref(false);
const selectedEntry = ref<ScheduleEntry | null>(null);
const defaultDate = ref<Date | undefined>();
const defaultGroupId = ref<number | undefined>();
const defaultEmployeeId = ref<number | undefined>();
const defaultEntryType = ref<EntryType>('WORK');
const dialogAbsenceMode = ref(false);
const employeeDialogOpen = ref(false);
const selectedEmployee = ref<Employee | null>(null);
const staffingDialogOpen = ref(false);

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
    
    const dateStr = toISODateString(date);
    const specialDay = getSpecialDayForDate(dateStr);
    const specialDayName = specialDay?.name || undefined;
    const specialDayType = specialDay?.dayType;

    // Fallback for older responses while backend data is refreshing.
    const daySchedule = weekSchedule.value?.days?.find(
      d => d.date === dateStr
    );
    const isClosedDay = isClosedSpecialDay(specialDay) || (daySchedule?.isHoliday || false);
    
    days.push({
      date,
      dateStr,
      dayName: WEEKDAYS_SHORT[i],
      dayNumber: date.getDate(),
      isToday: date.toDateString() === new Date().toDateString(),
      isPast: date < new Date(new Date().toDateString()),
      isWeekend: i >= 5,
      isHoliday: isClosedDay,
      holidayName: specialDayName || daySchedule?.holidayName,
      specialDay,
      specialDayName: specialDayName || daySchedule?.holidayName,
      specialDayType,
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

const activeEmployees = computed(() => (employees.value || []).filter(emp => emp.active !== false));
const sortedActiveEmployees = computed(() => {
  return [...activeEmployees.value].sort((a, b) => {
    const groupA = a.primaryGroup?.name || 'Springer';
    const groupB = b.primaryGroup?.name || 'Springer';
    const groupCompare = groupA.localeCompare(groupB);
    if (groupCompare !== 0) return groupCompare;
    return employeeDisplayName(a).localeCompare(employeeDisplayName(b));
  });
});
const timelineStart = 6 * 60;
const timelineEnd = 17 * 60;
const timelineStep = 30;
const timelineTicks = Array.from({ length: 12 }, (_, index) => timelineStart + index * 60);
const cellTimelineStart = 6 * 60 + 30;
const cellTimelineEnd = 16 * 60 + 30;
const cellHourTicks = Array.from({ length: 10 }, (_, index) => (index + 7) * 60);
const cellMajorMarkers = [
  { time: 9 * 60, label: '9', title: '09:00 Frühstück' },
  { time: 12 * 60, label: '12', title: '12:00 Mittag' },
  { time: 15 * 60, label: '15', title: '15:00 Nachmittag' },
];

function getSpecialDayForDate(dateStr: string): SpecialDay | undefined {
  const specialDays = weekSchedule.value?.specialDays || [];
  const matches = specialDays.filter(day => {
    if (!day.date) return false;
    const endDate = day.endDate || day.date;
    return day.date <= dateStr && dateStr <= endDate;
  });

  return matches.sort((a, b) => specialDayPriority(a.dayType) - specialDayPriority(b.dayType))[0];
}

function specialDayPriority(dayType?: string): number {
  switch (dayType) {
    case 'HOLIDAY': return 0;
    case 'CLOSURE': return 1;
    case 'TEAM_DAY': return 2;
    case 'EVENT': return 3;
    default: return 4;
  }
}

function isClosedSpecialDay(day?: SpecialDay): boolean {
  return day?.dayType === 'HOLIDAY' || day?.dayType === 'CLOSURE';
}

function getSpecialDayBackgroundClass(dayType?: string, target: 'header' | 'cell' = 'header'): string {
  const subtle = target === 'cell';
  switch (dayType) {
    case 'HOLIDAY': return subtle ? 'bg-red-50/50' : 'bg-red-50';
    case 'CLOSURE': return subtle ? 'bg-orange-50/60' : 'bg-orange-50';
    case 'TEAM_DAY': return subtle ? 'bg-purple-50/60' : 'bg-purple-50';
    case 'EVENT': return subtle ? 'bg-amber-50/60' : 'bg-amber-50';
    default: return '';
  }
}

function getSpecialDayTextClass(dayType?: string): string {
  switch (dayType) {
    case 'HOLIDAY': return 'text-red-600';
    case 'CLOSURE': return 'text-orange-700';
    case 'TEAM_DAY': return 'text-purple-700';
    case 'EVENT': return 'text-amber-700';
    default: return 'text-stone-900';
  }
}

// Get entries for an employee on a specific day
function getEntriesForEmployeeAndDay(employeeId: number, dateStr: string): ScheduleEntry[] {
  const daySchedule = weekSchedule.value?.days?.find(d => d.date === dateStr);
  if (!daySchedule) return [];
  return (daySchedule.entries || []).filter(e => e.employeeId === employeeId);
}

// Get color for entry type
function getEntryColor(entryType: string, groupColor?: string): string {
  switch (entryType) {
    case 'VACATION': return '#3B82F6'; // blue
    case 'SICK': return '#EF4444'; // red
    case 'CHILD_SICK': return '#F97316'; // orange
    case 'RECOVERY_DAY': return '#14B8A6'; // teal
    case 'TRAINING': return '#8B5CF6'; // purple
    case 'EVENT': return '#F59E0B'; // amber
    case 'SPECIAL_LEAVE': return '#EC4899'; // pink
    default: return groupColor || '#10B981'; // green or group color
  }
}

function getEntryTypeLabel(entryType: string): string {
  switch (entryType) {
    case 'VACATION': return 'Urlaub';
    case 'SICK': return 'Krank';
    case 'CHILD_SICK': return 'Kind krank';
    case 'RECOVERY_DAY': return 'Erholungstag';
    case 'SPECIAL_LEAVE': return 'Sonderurlaub';
    case 'TRAINING': return 'Fortbildung';
    case 'EVENT': return 'Veranstaltung';
    default: return entryType;
  }
}

// Dialog handlers
function openCreateDialog(date: Date, groupId?: number) {
  selectedEntry.value = null;
  defaultDate.value = date;
  defaultGroupId.value = groupId;
  defaultEmployeeId.value = undefined;
  defaultEntryType.value = 'WORK';
  dialogAbsenceMode.value = false;
  dialogOpen.value = true;
}

function openCreateEmployeeDialog(
  date: Date,
  employeeId: number,
  groupId?: number,
  entryType: EntryType = 'WORK',
) {
  const employee = employees.value?.find(e => e.id === employeeId);
  selectedEntry.value = null;
  defaultDate.value = date;
  defaultEmployeeId.value = employeeId;
  defaultGroupId.value = groupId || employee?.primaryGroupId;
  defaultEntryType.value = entryType;
  dialogAbsenceMode.value = entryType !== 'WORK';
  dialogOpen.value = true;
}

function openAbsenceDialog(date: Date, employee: Employee) {
  if (!employee.id) return;

  const dateStr = toISODateString(date);
  const existingEntries = getEntriesForEmployeeAndDay(employee.id, dateStr);
  const existingAbsence = existingEntries.find(entry => entry.entryType !== 'WORK');
  const existingWork = existingEntries.find(entry => entry.entryType === 'WORK');
  const entry = existingAbsence || existingWork;

  if (entry) {
    selectedEntry.value = entry;
    defaultDate.value = date;
    defaultEmployeeId.value = employee.id;
    defaultGroupId.value = entry.groupId || entry.group?.id || employee.primaryGroupId;
    defaultEntryType.value = entry.entryType === 'WORK' ? 'VACATION' : (entry.entryType as EntryType);
    dialogAbsenceMode.value = true;
    dialogOpen.value = true;
    return;
  }

  openCreateEmployeeDialog(date, employee.id, employee.primaryGroupId, 'VACATION');
}

function openEditDialog(entry: ScheduleEntry) {
  selectedEntry.value = entry;
  defaultDate.value = undefined;
  defaultGroupId.value = undefined;
  defaultEmployeeId.value = undefined;
  defaultEntryType.value = 'WORK';
  dialogAbsenceMode.value = false;
  dialogOpen.value = true;
}

function openEmployeeDialog(employee: Employee) {
  selectedEmployee.value = employee;
  employeeDialogOpen.value = true;
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

async function handleEmployeeSave(data: { employee: CreateEmployeeRequest | UpdateEmployeeRequest; contract: EmployeeContractRequest }) {
  if (!selectedEmployee.value?.id) return;

  try {
    await updateEmployee.mutateAsync({
      id: selectedEmployee.value.id,
      data: data.employee as UpdateEmployeeRequest,
    });

    if (selectedEmployee.value.currentContract?.id && selectedEmployee.value.currentContract.validFrom === data.contract.validFrom) {
      await updateEmployeeContract.mutateAsync({
        employeeId: selectedEmployee.value.id,
        contractId: selectedEmployee.value.currentContract.id,
        data: data.contract,
      });
    } else {
      await createEmployeeContract.mutateAsync({
        employeeId: selectedEmployee.value.id,
        data: data.contract,
      });
    }

    employeeDialogOpen.value = false;
  } catch (err) {
    console.error('Failed to save employee:', err);
  }
}

const isLoading = computed(() => scheduleLoading.value || groupsLoading.value);

function isEmployeeBlocked(employee: any, date: Date): boolean {
  const day = date.getDay() === 0 ? 7 : date.getDay();
  const pattern = employee.workPattern || employee.currentContract?.workdays || [];
  return !pattern.some((item: any) => item.weekday === day);
}

function employeeDisplayName(employee: any): string {
  return employee.nickname || employee.firstName || '';
}

function getEmployeeTargetMinutesForDate(employee: any, date: Date, isHoliday = false): number {
  if (date.getDay() === 0 || date.getDay() === 6 || isHoliday) return 0;

  const pattern = employee.workPattern || employee.currentContract?.workdays || [];
  if (pattern.length > 0) {
    const weekday = date.getDay() === 0 ? 7 : date.getDay();
    const workday = pattern.find((item: any) => item.weekday === weekday);
    return workday?.plannedMinutes || 0;
  }

  return Math.round(((employee.weeklyHours || 0) * 60) / 5);
}

function getEmployeeTargetMinutes(employee: any): number {
  const pattern = employee.workPattern || employee.currentContract?.workdays || [];
  if (pattern.length > 0) {
    return allWeekDays.value.reduce((sum, day) => {
      if (day.isWeekend || day.isHoliday) return sum;
      const weekday = day.date.getDay() === 0 ? 7 : day.date.getDay();
      const workday = pattern.find((item: any) => item.weekday === weekday);
      return sum + (workday?.plannedMinutes || 0);
    }, 0);
  }

  const weekdayCount = allWeekDays.value.filter(day => !day.isWeekend).length;
  const openWeekdayCount = allWeekDays.value.filter(day => !day.isWeekend && !day.isHoliday).length;
  if (!weekdayCount) return 0;
  return Math.round(((employee.weeklyHours || 0) * 60 / weekdayCount) * openWeekdayCount);
}

const staffingDays = computed(() => {
  if (!groups.value || !weekSchedule.value) return [];

  return weekDays.value.map(day => {
    const daySchedule = weekSchedule.value?.days?.find(item => item.date === day.dateStr);
    const entries = (daySchedule?.entries || []).filter(entry =>
      entry.entryType === 'WORK' && entry.startTime && entry.endTime && entry.groupId
    );

    return {
      day,
      groups: [...groups.value!].sort((a, b) => (a.name || '').localeCompare(b.name || '')).map(group => {
        const groupEntries = entries.filter(entry => entry.groupId === group.id);
        return {
          group,
          segments: buildCoverageSegments(groupEntries),
        };
      }),
    };
  });
});

// Calculate weekly hours per employee
const employeeWeeklyHours = computed(() => {
  if (!employees.value || !weekSchedule.value) return [];
  
  return sortedActiveEmployees.value.map(emp => {
    let plannedMinutes = 0;

    for (const day of allWeekDays.value) {
      const daySchedule = weekSchedule.value?.days?.find(item => item.date === day.dateStr);
      const entries = (daySchedule?.entries || []).filter(entry => entry.employeeId === emp.id);
      const workEntries = entries.filter(entry => entry.entryType === 'WORK');
      const hasExcusedAbsence = entries.some(entry => entry.entryType !== 'WORK');

      let dayPlannedMinutes = 0;
      for (const entry of workEntries) {
        if (entry.startTime && entry.endTime) {
          const start = parseTime(entry.startTime);
          const end = parseTime(entry.endTime);
          const breakMins = entry.breakMinutes || 0;
          dayPlannedMinutes += (end - start) - breakMins;
        }
      }

      if (hasExcusedAbsence) {
        dayPlannedMinutes = Math.max(
          dayPlannedMinutes,
          getEmployeeTargetMinutesForDate(emp, day.date, day.isHoliday)
        );
      }

      plannedMinutes += dayPlannedMinutes;
    }
    
    const plannedHours = plannedMinutes / 60;
    const contractedHours = getEmployeeTargetMinutes(emp) / 60;
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

function getWorkEntriesForEmployeeAndDay(employeeId: number, dateStr: string): ScheduleEntry[] {
  return getEntriesForEmployeeAndDay(employeeId, dateStr).filter(entry =>
    entry.entryType === 'WORK' && Boolean(entry.startTime) && Boolean(entry.endTime)
  );
}

function getAbsenceEntriesForEmployeeAndDay(employeeId: number, dateStr: string): ScheduleEntry[] {
  return getEntriesForEmployeeAndDay(employeeId, dateStr).filter(entry => entry.entryType !== 'WORK');
}

function getPrimaryEntryForEmployeeAndDay(employeeId: number, dateStr: string): ScheduleEntry | undefined {
  const entries = getEntriesForEmployeeAndDay(employeeId, dateStr);
  return entries.find(entry => entry.entryType === 'WORK') || entries[0];
}

function getGroupsForCell(employee: Employee) {
  const allGroups = groups.value || [];
  return [...allGroups].sort((a, b) => {
    if (a.id === employee.primaryGroupId) return -1;
    if (b.id === employee.primaryGroupId) return 1;
    return (a.name || '').localeCompare(b.name || '');
  });
}

function compactEntryStyle(entry: ScheduleEntry) {
  if (!entry.startTime || !entry.endTime) {
    return { left: '0%', width: '100%', backgroundColor: getEntryColor(entry.entryType || 'WORK', entry.group?.color) };
  }

  const start = Math.max(parseTime(entry.startTime), cellTimelineStart);
  const end = Math.min(parseTime(entry.endTime), cellTimelineEnd);
  const width = Math.max(3, ((end - start) / (cellTimelineEnd - cellTimelineStart)) * 100);
  return {
    left: `${((start - cellTimelineStart) / (cellTimelineEnd - cellTimelineStart)) * 100}%`,
    width: `${width}%`,
    backgroundColor: getEntryColor(entry.entryType || 'WORK', entry.group?.color),
  };
}

function markerStyle(minutes: number) {
  return {
    left: `${((minutes - cellTimelineStart) / (cellTimelineEnd - cellTimelineStart)) * 100}%`,
  };
}

function compactEntryTitle(entry: ScheduleEntry, employee: Employee): string {
  const groupName = entry.group?.name || employee.primaryGroup?.name || 'Springer';
  const start = entry.startTime?.substring(0, 5) || '';
  const end = entry.endTime?.substring(0, 5) || '';
  return `${groupName}: ${start} - ${end}`;
}

async function handleCellGroupClick(date: Date, employee: Employee, groupId?: number) {
  if (!employee.id || !groupId) return;

  const existingEntry = getPrimaryEntryForEmployeeAndDay(employee.id, toISODateString(date));
  if (!existingEntry?.id) {
    openCreateEmployeeDialog(date, employee.id, groupId);
    return;
  }

  try {
    await updateEntry.mutateAsync({
      id: existingEntry.id,
      data: { groupId },
    });
  } catch (err) {
    console.error('Failed to update entry group:', err);
  }
}

function formatHour(minutes: number): string {
  const hour = Math.floor(minutes / 60);
  return `${hour}:00`;
}

function formatTimeLabel(minutes: number): string {
  const hour = Math.floor(minutes / 60);
  const minute = minutes % 60;
  return `${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}`;
}

function buildCoverageSegments(
  entries: ScheduleEntry[],
  rangeStart = timelineStart,
  rangeEnd = timelineEnd,
  step = timelineStep,
) {
  const slots = [];
  for (let start = rangeStart; start < rangeEnd; start += step) {
    const end = start + step;
    const count = entries.filter(entry => {
      if (!entry.startTime || !entry.endTime) return false;
      const entryStart = parseTime(entry.startTime);
      const entryEnd = parseTime(entry.endTime);
      return entryStart < end && entryEnd > start;
    }).length;
    slots.push({ start, end, count });
  }

  const segments: Array<{ start: number; end: number; count: number }> = [];
  for (const slot of slots) {
    const last = segments[segments.length - 1];
    if (slot.count > 0 && last?.count === slot.count && last.end === slot.start) {
      last.end = slot.end;
    } else if (slot.count > 0) {
      segments.push({ ...slot });
    }
  }
  return segments;
}

function segmentStyle(segment: { start: number; end: number; count: number }, color?: string) {
  const start = Math.max(segment.start, timelineStart);
  const end = Math.min(segment.end, timelineEnd);
  return {
    left: `${((start - timelineStart) / (timelineEnd - timelineStart)) * 100}%`,
    width: `${((end - start) / (timelineEnd - timelineStart)) * 100}%`,
    backgroundColor: color || '#10B981',
    opacity: String(Math.min(0.35 + segment.count * 0.18, 0.95)),
  };
}

function getCellCoverageSegments(dateStr: string, groupId?: number) {
  if (!groupId) return [];
  const daySchedule = weekSchedule.value?.days?.find(day => day.date === dateStr);
  const entries = (daySchedule?.entries || []).filter(entry =>
    entry.entryType === 'WORK' && entry.startTime && entry.endTime && entry.groupId === groupId
  );
  return buildCoverageSegments(entries, cellTimelineStart, cellTimelineEnd, timelineStep);
}

function cellCoverageSegmentStyle(segment: { start: number; end: number; count: number }, color?: string) {
  const start = Math.max(segment.start, cellTimelineStart);
  const end = Math.min(segment.end, cellTimelineEnd);
  return {
    left: `${((start - cellTimelineStart) / (cellTimelineEnd - cellTimelineStart)) * 100}%`,
    width: `${((end - start) / (cellTimelineEnd - cellTimelineStart)) * 100}%`,
    backgroundColor: color || '#10B981',
    opacity: String(Math.min(0.18 + segment.count * 0.16, 0.9)),
  };
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

        <Button
          variant="outline"
          size="sm"
          :disabled="staffingDays.length === 0"
          @click="staffingDialogOpen = true"
        >
          <BarChart3 class="w-4 h-4 mr-2" />
          Besetzung
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
        <div class="px-4 py-3 bg-stone-100 border-r border-stone-200">
          <span class="text-sm font-medium text-stone-600">Mitarbeiter</span>
        </div>
        <div
          v-for="day in weekDays"
          :key="day.dateStr"
          :class="[
            'px-4 py-3 text-center border-r border-stone-200 last:border-r-0',
            day.isWeekend ? 'bg-stone-100' : 'bg-stone-50',
            day.isPast ? 'bg-stone-100 text-stone-400 opacity-60' : '',
            day.isToday ? 'bg-primary/10' : '',
            getSpecialDayBackgroundClass(day.specialDayType, 'header')
          ]"
        >
          <div class="text-sm font-medium text-stone-600">{{ day.dayName }}</div>
          <div
            :class="[
              'text-lg font-semibold',
              day.isToday ? 'text-primary' : 'text-stone-900',
              getSpecialDayTextClass(day.specialDayType)
            ]"
          >
            {{ day.dayNumber }}
          </div>
          <div v-if="day.specialDayName" :class="['text-xs truncate', getSpecialDayTextClass(day.specialDayType)]">
            {{ day.specialDayName }}
          </div>
        </div>
      </div>

      <!-- Employee rows -->
      <div
        v-for="employee in sortedActiveEmployees"
        :key="employee.id"
        :class="['grid border-b border-stone-200 last:border-b-0', gridColsClass]"
      >
        <!-- Employee name -->
        <div
          :class="[
            'px-3 py-1.5 bg-stone-100 border-r border-stone-200 flex items-center gap-2',
            isAdmin ? 'cursor-pointer hover:bg-stone-200/70' : ''
          ]"
          @click="isAdmin && openEmployeeDialog(employee)"
        >
          <div
            class="h-2.5 w-2.5 shrink-0 rounded-full"
            :style="{ backgroundColor: employee.primaryGroup?.color || '#10B981' }"
          />
          <div class="flex min-w-0 items-baseline gap-1.5">
            <span class="truncate text-sm font-medium text-stone-900">{{ employeeDisplayName(employee) }}</span>
            <span class="shrink-0 text-xs text-stone-500">{{ employee.weeklyHours }} Std.</span>
            <span class="truncate text-xs text-stone-500">· {{ employee.primaryGroup?.name || 'Springer' }}</span>
          </div>
        </div>

        <!-- Day cells -->
        <div
          v-for="day in weekDays"
          :key="`${employee.id}-${day.dateStr}`"
          :class="[
            'group/cell relative px-2 py-1 border-r border-stone-200 last:border-r-0 min-h-[38px]',
            isEmployeeBlocked(employee, day.date) ? 'bg-stone-100/70 cursor-not-allowed' : 'cursor-pointer hover:bg-stone-50/50',
            day.isWeekend ? 'bg-stone-50' : '',
            day.isPast ? 'bg-stone-100/60 opacity-60 grayscale-[35%]' : '',
            day.isToday ? 'bg-primary/5' : '',
            getSpecialDayBackgroundClass(day.specialDayType, 'cell')
          ]"
          @click="isAdmin && openCreateEmployeeDialog(day.date, employee.id!)"
        >
          <div class="flex h-full min-h-[30px] flex-col justify-center gap-1">
            <div v-if="isEmployeeBlocked(employee, day.date)" class="text-xs text-stone-400">
              blockiert
            </div>
            <template v-else>
              <div v-if="getWorkEntriesForEmployeeAndDay(employee.id!, day.dateStr).length" class="space-y-1">
                <button
                  v-for="entry in getWorkEntriesForEmployeeAndDay(employee.id!, day.dateStr)"
                  :key="entry.id"
                  type="button"
                  class="relative block h-4 w-full overflow-hidden rounded-full border border-stone-200 bg-stone-100 transition-opacity hover:opacity-80"
                  :title="compactEntryTitle(entry, employee)"
                  @click.stop="openEditDialog(entry)"
                >
                  <span
                    v-for="segment in getCellCoverageSegments(day.dateStr, entry.groupId || entry.group?.id || employee.primaryGroupId)"
                    :key="`${segment.start}-${segment.end}-${segment.count}`"
                    class="absolute inset-y-0"
                    :style="cellCoverageSegmentStyle(segment, entry.group?.color || employee.primaryGroup?.color)"
                    :title="`${entry.group?.name || employee.primaryGroup?.name || 'Springer'}: ${segment.count} MA (${formatTimeLabel(segment.start)}-${formatTimeLabel(segment.end)})`"
                  />
                  <span
                    class="absolute bottom-[3px] h-1 rounded-full"
                    :style="compactEntryStyle(entry)"
                  />
                  <span
                    v-for="tick in cellHourTicks"
                    :key="tick"
                    class="absolute inset-y-0 w-px bg-stone-500/35"
                    :style="markerStyle(tick)"
                  />
                  <span
                    v-for="marker in cellMajorMarkers"
                    :key="marker.time"
                    class="absolute inset-y-0 w-px bg-stone-600/60"
                    :style="markerStyle(marker.time)"
                    :title="marker.title"
                  >
                    <span class="absolute top-0 left-1/2 -translate-x-1/2 text-[8px] font-semibold leading-none text-stone-600">
                      {{ marker.label }}
                    </span>
                  </span>
                </button>
              </div>
              <div v-else class="relative h-4 overflow-hidden rounded-full border border-dashed border-stone-200 bg-stone-50/80">
                <span
                  v-for="tick in cellHourTicks"
                  :key="tick"
                  class="absolute inset-y-0 w-px bg-stone-300"
                  :style="markerStyle(tick)"
                />
                <span
                  v-for="marker in cellMajorMarkers"
                  :key="marker.time"
                  class="absolute inset-y-0 w-px bg-stone-400"
                  :style="markerStyle(marker.time)"
                  :title="marker.title"
                >
                  <span class="absolute top-0 left-1/2 -translate-x-1/2 text-[8px] font-semibold leading-none text-stone-400">
                    {{ marker.label }}
                  </span>
                </span>
              </div>

              <div v-if="getAbsenceEntriesForEmployeeAndDay(employee.id!, day.dateStr).length" class="flex flex-wrap gap-1">
                <button
                  v-for="entry in getAbsenceEntriesForEmployeeAndDay(employee.id!, day.dateStr)"
                  :key="entry.id"
                  type="button"
                  class="rounded px-1.5 py-0.5 text-[11px] font-medium leading-none text-white hover:opacity-80"
                  :style="{ backgroundColor: getEntryColor(entry.entryType || 'VACATION') }"
                  @click.stop="openEditDialog(entry)"
                >
                  {{ getEntryTypeLabel(entry.entryType || '') }}
                </button>
              </div>
            </template>
          </div>

          <div
            v-if="isAdmin && !isEmployeeBlocked(employee, day.date)"
            class="absolute bottom-1 right-1 flex items-center gap-1 rounded bg-white/90 px-1 py-0.5 opacity-0 shadow-sm ring-1 ring-stone-200 transition-opacity group-hover/cell:opacity-100 focus-within:opacity-100"
            @click.stop
          >
            <button
              v-for="group in getGroupsForCell(employee)"
              :key="group.id"
              type="button"
              class="h-3.5 w-3.5 rounded-full ring-1 ring-white transition-transform hover:scale-125 focus:outline-none focus:ring-2 focus:ring-stone-900"
              :style="{ backgroundColor: group.color || '#10B981' }"
              :title="`Arbeit in ${group.name}`"
              @click.stop="handleCellGroupClick(day.date, employee, group.id)"
            />
            <button
              type="button"
              class="ml-0.5 rounded p-0.5 text-stone-500 hover:bg-stone-100 hover:text-stone-900 focus:outline-none focus:ring-2 focus:ring-stone-900"
              title="Abwesenheit eintragen"
              @click.stop="openAbsenceDialog(day.date, employee)"
            >
              <CalendarX class="h-3.5 w-3.5" />
            </button>
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
        <div class="w-3 h-3 rounded-full bg-orange-500" />
        <span>Kind krank</span>
      </div>
      <div class="flex items-center gap-2">
        <div class="w-3 h-3 rounded-full bg-teal-500" />
        <span>Erholungstag</span>
      </div>
      <div class="flex items-center gap-2">
        <div class="w-3 h-3 rounded-full bg-pink-500" />
        <span>Sonderurlaub</span>
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
      :default-employee-id="defaultEmployeeId"
      :default-entry-type="defaultEntryType"
      :absence-mode="dialogAbsenceMode"
      @save="handleSave"
      @delete="handleDelete"
    />

    <EmployeeFormDialog
      v-model:open="employeeDialogOpen"
      :employee="selectedEmployee"
      :groups="groups || []"
      @save="handleEmployeeSave"
    />

    <Dialog
      v-model:open="staffingDialogOpen"
      title="Besetzung nach Uhrzeit"
      :description="`${formatDate(weekStart)} - ${formatDate(weekEnd)}`"
      content-class="max-h-[85vh] max-w-[min(1120px,calc(100vw-2rem))] grid-rows-[auto_minmax(0,1fr)] overflow-hidden"
    >
      <div class="min-h-0 overflow-auto pr-1">
        <div class="divide-y divide-stone-200 border-t border-stone-200">
          <div v-for="day in staffingDays" :key="day.day.dateStr" class="py-4">
            <div class="mb-3 flex items-center justify-between">
              <div>
                <div class="text-sm font-semibold text-stone-900">
                  {{ day.day.dayName }} {{ day.day.dayNumber }}
                </div>
                <div v-if="day.day.specialDayName" :class="['text-xs', getSpecialDayTextClass(day.day.specialDayType)]">
                  {{ day.day.specialDayName }}
                </div>
              </div>
            </div>

            <div class="min-w-[780px]">
              <div class="ml-28 grid grid-cols-11 text-[11px] text-stone-400">
                <span v-for="tick in timelineTicks.slice(0, -1)" :key="tick">{{ formatHour(tick) }}</span>
              </div>

              <div class="mt-1 space-y-2">
                <div v-for="row in day.groups" :key="row.group.id" class="grid grid-cols-[7rem_1fr] items-center gap-4">
                  <div class="flex min-w-0 items-center gap-2 text-xs">
                    <span class="h-2.5 w-2.5 rounded-full" :style="{ backgroundColor: row.group.color || '#10B981' }" />
                    <span class="truncate font-medium text-stone-700">{{ row.group.name }}</span>
                  </div>

                  <div class="relative h-7 rounded border border-stone-200 bg-stone-50">
                    <div class="absolute inset-y-0 left-0 right-0 grid grid-cols-11">
                      <div v-for="tick in timelineTicks.slice(0, -1)" :key="tick" class="border-r border-stone-200 last:border-r-0" />
                    </div>
                    <div
                      v-for="segment in row.segments"
                      :key="`${segment.start}-${segment.end}-${segment.count}`"
                      class="absolute top-1 bottom-1 rounded-sm px-1 text-center text-[11px] font-semibold leading-5 text-white"
                      :style="segmentStyle(segment, row.group.color)"
                      :title="`${row.group.name}: ${segment.count} Mitarbeiter (${formatTimeLabel(segment.start)}-${formatTimeLabel(segment.end)})`"
                    >
                      {{ segment.count }}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Dialog>
  </div>
</template>
