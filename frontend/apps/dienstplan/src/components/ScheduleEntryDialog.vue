<script setup lang="ts">
import { ref, watch, computed } from 'vue';
import { Dialog, Button, Input, Label, Select, type SelectOption } from '@/components/ui';
import type { 
  ScheduleEntry, 
  CreateScheduleEntryRequest, 
  UpdateScheduleEntryRequest,
  ScheduleEntrySegmentRequest,
  Employee,
  Group 
} from '@kita/shared';
import { useScheduleTimeSuggestion } from '@kita/shared';
import { toISODateString } from '@kita/shared/utils';

type EntryType = 'WORK' | 'VACATION' | 'SICK' | 'CHILD_SICK' | 'RECOVERY_DAY' | 'SPECIAL_LEAVE' | 'TRAINING' | 'EVENT';

const props = defineProps<{
  open: boolean;
  entry?: ScheduleEntry | null;
  employees: Employee[];
  groups: Group[];
  defaultDate?: Date;
  defaultGroupId?: number;
  defaultEmployeeId?: number;
  defaultEntryType?: EntryType;
  absenceMode?: boolean;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  'save': [data: CreateScheduleEntryRequest | UpdateScheduleEntryRequest, id?: number];
  'delete': [id: number];
}>();

const isEditing = computed(() => !!props.entry?.id);

const entryTypeOptions: SelectOption[] = [
  { value: 'WORK', label: 'Arbeit' },
  { value: 'VACATION', label: 'Urlaub' },
  { value: 'SICK', label: 'Krank' },
  { value: 'CHILD_SICK', label: 'Kind krank' },
  { value: 'RECOVERY_DAY', label: 'Erholungstag' },
  { value: 'SPECIAL_LEAVE', label: 'Sonderurlaub' },
  { value: 'TRAINING', label: 'Fortbildung' },
  { value: 'EVENT', label: 'Veranstaltung' },
];

const employeeOptions = computed<SelectOption[]>(() => {
  return props.employees.map(e => ({
    value: String(e.id),
    label: employeeDisplayName(e),
  }));
});

const groupOptions = computed<SelectOption[]>(() => 
  props.groups.map(g => ({
    value: String(g.id),
    label: g.name || '',
  }))
);

const startDefaults = ['06:30', '07:00', '07:30', '08:00'];
const timeSuggestion = useScheduleTimeSuggestion();

// Form state
const form = ref({
  employeeId: '',
  date: '',
  startTime: '07:00',
  endTime: '15:00',
  breakMinutes: 30,
  groupId: '',
  entryType: 'WORK' as EntryType,
  shiftKind: 'EARLY' as 'EARLY' | 'LATE' | 'MANUAL',
  overrideBlockedDay: false,
  plannedMinutes: 0,
  notes: '',
  segments: [] as ScheduleEntrySegmentRequest[],
});

// Track if user manually changed the group (don't override manual selections)
const userChangedGroup = ref(false);

// Reset form when dialog opens/entry changes
watch(
  () => [props.open, props.entry, props.defaultDate, props.defaultGroupId, props.defaultEmployeeId, props.defaultEntryType, props.absenceMode],
  () => {
    if (props.open) {
      userChangedGroup.value = false; // Reset on dialog open
      
      if (props.entry) {
        // Use groupId directly, or fall back to group.id if available
        const entryGroupId = props.entry.groupId ?? props.entry.group?.id;
        const entryType = props.absenceMode && props.entry.entryType === 'WORK'
          ? (props.defaultEntryType || 'VACATION')
          : ((props.entry.entryType as EntryType) || 'WORK');
        form.value = {
          employeeId: String(props.entry.employeeId || ''),
          date: props.entry.date || '',
          startTime: props.entry.startTime?.substring(0, 5) || '07:00',
          endTime: props.entry.endTime?.substring(0, 5) || '15:00',
          breakMinutes: props.entry.breakMinutes ?? 30,
          groupId: entryGroupId ? String(entryGroupId) : '',
          entryType,
          shiftKind: props.absenceMode ? 'MANUAL' : ((props.entry.shiftKind as any) || 'MANUAL'),
          overrideBlockedDay: false,
          plannedMinutes: calculateEntryMinutes(props.entry.startTime, props.entry.endTime, props.entry.breakMinutes),
          notes: props.entry.notes || '',
          segments: (props.entry.segments || []).map(segment => ({
            groupId: segment.groupId || 0,
            startTime: segment.startTime?.substring(0, 5) || '',
            endTime: segment.endTime?.substring(0, 5) || '',
            notes: segment.notes,
            sortOrder: segment.sortOrder,
          })),
        };
      } else {
        form.value = {
          employeeId: props.defaultEmployeeId ? String(props.defaultEmployeeId) : '',
          date: props.defaultDate ? toISODateString(props.defaultDate) : '',
          startTime: '07:00',
          endTime: '15:00',
          breakMinutes: 30,
          groupId: props.defaultGroupId ? String(props.defaultGroupId) : '',
          entryType: props.defaultEntryType || 'WORK',
          shiftKind: 'EARLY',
          overrideBlockedDay: false,
          plannedMinutes: 0,
          notes: '',
          segments: [],
        };
        if (!form.value.groupId && props.defaultEmployeeId) {
          const selectedEmployee = props.employees.find(e => e.id === props.defaultEmployeeId);
          if (selectedEmployee?.primaryGroupId) {
            form.value.groupId = String(selectedEmployee.primaryGroupId);
          }
        }
        void applySuggestion();
      }
    }
  },
  { immediate: true }
);

watch(
  () => [form.value.employeeId, form.value.date, form.value.shiftKind, form.value.startTime],
  () => {
    if (!props.open || isEditing.value || form.value.entryType !== 'WORK') return;
    void applySuggestion();
  }
);

// Auto-fill group when employee is selected (only for new entries)
watch(
  () => form.value.employeeId,
  (newEmployeeId) => {
    // Only auto-fill if:
    // 1. We're creating a new entry (not editing)
    // 2. User hasn't manually selected a group
    // 3. An employee is actually selected
    if (!isEditing.value && !userChangedGroup.value && newEmployeeId) {
      const selectedEmployee = props.employees.find(e => String(e.id) === newEmployeeId);
      if (selectedEmployee?.primaryGroupId) {
        form.value.groupId = String(selectedEmployee.primaryGroupId);
      }
    }
  }
);

// Track when user manually changes the group
function onGroupChange(value: string) {
  userChangedGroup.value = true;
  form.value.groupId = value;
}

async function applySuggestion() {
  if (!form.value.employeeId || !form.value.date || form.value.entryType !== 'WORK' || form.value.shiftKind === 'MANUAL') {
    return;
  }
  try {
    const suggestion = await timeSuggestion.mutateAsync({
      employeeId: parseInt(form.value.employeeId),
      date: form.value.date,
      shiftKind: form.value.shiftKind,
      startTime: form.value.shiftKind === 'EARLY' ? `${form.value.startTime}:00` : undefined,
    });
    if (suggestion?.startTime) form.value.startTime = suggestion.startTime.substring(0, 5);
    if (suggestion?.endTime) form.value.endTime = suggestion.endTime.substring(0, 5);
    form.value.breakMinutes = suggestion?.breakMinutes ?? form.value.breakMinutes;
    form.value.plannedMinutes = suggestion?.plannedMinutes ?? 0;
    form.value.overrideBlockedDay = false;
  } catch (err) {
    console.error('Failed to calculate schedule suggestion:', err);
  }
}

function applyStartDefault(value: string) {
  form.value.shiftKind = 'EARLY';
  form.value.startTime = value;
}

const selectedEmployee = computed(() => props.employees.find(e => String(e.id) === form.value.employeeId));
const selectedGroup = computed(() => props.groups.find(group => String(group.id) === form.value.groupId));
const isCellCreate = computed(() => !isEditing.value && Boolean(props.defaultEmployeeId && props.defaultDate));
const isCompactContext = computed(() => isCellCreate.value || Boolean(props.absenceMode && props.defaultEmployeeId && props.defaultDate));
const isCompactWorkCreate = computed(() => isCellCreate.value && form.value.entryType === 'WORK');
const formattedSelectedDate = computed(() => {
  if (!form.value.date) return '';
  return new Date(`${form.value.date}T00:00:00`).toLocaleDateString('de-DE', {
    weekday: 'short',
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  });
});
const selectedWeekday = computed(() => {
  if (!form.value.date) return 0;
  const day = new Date(`${form.value.date}T00:00:00`).getDay();
  return day === 0 ? 7 : day;
});
const isBlockedDay = computed(() => {
  if (!selectedEmployee.value || !selectedWeekday.value) return false;
  const pattern = selectedEmployee.value.workPattern || selectedEmployee.value.currentContract?.workdays || [];
  return !pattern.some(day => day.weekday === selectedWeekday.value);
});

const calculatedWorkTime = computed(() => {
  if (form.value.entryType !== 'WORK') return '';
  const planned = form.value.plannedMinutes || calculateEntryMinutes(
    `${form.value.startTime}:00`,
    `${form.value.endTime}:00`,
    Number(form.value.breakMinutes) || 0
  );
  return formatMinutes(planned);
});

function employeeDisplayName(employee: Employee): string {
  return employee.nickname || employee.firstName || '';
}

function calculateEntryMinutes(startTime?: string, endTime?: string, breakMinutes = 0): number {
  if (!startTime || !endTime) return 0;
  const start = parseTime(startTime);
  const end = parseTime(endTime);
  return Math.max(0, end - start - breakMinutes);
}

function parseTime(value: string): number {
  const [hours, minutes] = value.split(':').map(Number);
  return (hours || 0) * 60 + (minutes || 0);
}

function formatMinutes(minutes: number): string {
  const hours = Math.floor(minutes / 60);
  const mins = minutes % 60;
  if (!minutes) return '0 Std.';
  if (!mins) return `${hours} Std.`;
  return `${hours} Std. ${mins} Min.`;
}

function addSegment() {
  form.value.segments.push({
    groupId: form.value.groupId ? parseInt(form.value.groupId) : Number(props.groups[0]?.id || 0),
    startTime: form.value.startTime,
    endTime: form.value.endTime,
    notes: '',
    sortOrder: form.value.segments.length + 1,
  });
}

function removeSegment(index: number) {
  form.value.segments.splice(index, 1);
  form.value.segments.forEach((segment, segmentIndex) => {
    segment.sortOrder = segmentIndex + 1;
  });
}

// Form validation - groupId is now required for WORK entries
const isFormValid = computed(() => {
  const baseValid = form.value.employeeId && form.value.date && form.value.entryType;
  if (form.value.entryType === 'WORK') {
    return baseValid && form.value.groupId && (!isBlockedDay.value || form.value.overrideBlockedDay);
  }
  return baseValid;
});

function handleSubmit() {
  if (!isFormValid.value) {
    return;
  }
  
  const isWorkEntry = form.value.entryType === 'WORK';
  const data: CreateScheduleEntryRequest = {
    employeeId: parseInt(form.value.employeeId),
    date: form.value.date,
    startTime: isWorkEntry ? form.value.startTime + ':00' : undefined,
    endTime: isWorkEntry ? form.value.endTime + ':00' : undefined,
    breakMinutes: isWorkEntry ? form.value.breakMinutes : 0,
    groupId: form.value.groupId ? parseInt(form.value.groupId) : undefined,
    entryType: form.value.entryType,
    shiftKind: isWorkEntry ? form.value.shiftKind : 'MANUAL',
    overrideBlockedDay: form.value.overrideBlockedDay,
    notes: form.value.notes || undefined,
    segments: isWorkEntry
      ? form.value.segments.map((segment, index) => ({
          ...segment,
          groupId: Number(segment.groupId),
          startTime: `${segment.startTime}:00`,
          endTime: `${segment.endTime}:00`,
          notes: segment.notes || undefined,
          sortOrder: index + 1,
        }))
      : undefined,
  };
  emit('save', data, props.entry?.id);
}

function handleDelete() {
  if (props.entry?.id) {
    emit('delete', props.entry.id);
  }
}
</script>

<template>
  <Dialog
    :open="open"
    @update:open="emit('update:open', $event)"
    :title="isEditing ? 'Eintrag bearbeiten' : 'Neuer Eintrag'"
    :description="isEditing ? 'Bearbeite den Dienstplan-Eintrag.' : 'Erstelle einen neuen Dienstplan-Eintrag.'"
  >
    <form @submit.prevent="handleSubmit" class="space-y-4">
      <div v-if="isCompactContext" class="rounded-md border border-stone-200 bg-stone-50 px-3 py-2 text-sm">
        <div class="font-medium text-stone-900">
          {{ selectedEmployee ? employeeDisplayName(selectedEmployee) : 'Mitarbeiter' }}
        </div>
        <div class="text-stone-600">
          {{ formattedSelectedDate }}
          <span v-if="form.entryType === 'WORK'"> · Arbeit</span>
          <span v-else> · Abwesenheit</span>
          <span v-if="selectedGroup"> · {{ selectedGroup.name }}</span>
        </div>
      </div>

      <div v-else class="space-y-2">
        <Label for="employee">Mitarbeiter</Label>
        <Select
          v-model="form.employeeId"
          :options="employeeOptions"
          placeholder="Mitarbeiter auswählen"
        />
      </div>

      <div v-if="!isCompactContext" class="grid grid-cols-2 gap-4">
        <div class="space-y-2">
          <Label for="date">Datum</Label>
          <Input
            id="date"
            v-model="form.date"
            type="date"
            required
          />
        </div>
        <div class="space-y-2">
          <Label for="entryType">Typ</Label>
          <Select
            v-model="form.entryType"
            :options="entryTypeOptions"
            placeholder="Typ auswählen"
          />
        </div>
      </div>

      <div v-else-if="form.entryType !== 'WORK'" class="space-y-2">
        <Label for="entryType">Typ</Label>
        <Select
          v-model="form.entryType"
          :options="entryTypeOptions.filter(option => option.value !== 'WORK')"
          placeholder="Typ auswählen"
        />
      </div>

      <div class="space-y-2" v-if="form.entryType === 'WORK'">
        <Label>Dienstart</Label>
        <div class="grid grid-cols-3 gap-2">
          <Button type="button" size="sm" :variant="form.shiftKind === 'EARLY' ? 'default' : 'outline'" @click="form.shiftKind = 'EARLY'; applySuggestion()">
            Früh
          </Button>
          <Button type="button" size="sm" :variant="form.shiftKind === 'LATE' ? 'default' : 'outline'" @click="form.shiftKind = 'LATE'; applySuggestion()">
            Spät
          </Button>
          <Button type="button" size="sm" :variant="form.shiftKind === 'MANUAL' ? 'default' : 'outline'" @click="form.shiftKind = 'MANUAL'">
            Manuell
          </Button>
        </div>
      </div>

      <div v-if="form.entryType === 'WORK' && isBlockedDay" class="rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800">
        Dieser Tag ist im Arbeitsmuster des Mitarbeiters blockiert.
        <label class="mt-2 flex items-center gap-2">
          <input type="checkbox" v-model="form.overrideBlockedDay" class="rounded border-amber-300" />
          Trotzdem einplanen
        </label>
      </div>

      <div class="space-y-2" v-if="form.entryType === 'WORK' && form.shiftKind === 'EARLY'">
        <Label>Schnellstart</Label>
        <div class="flex flex-wrap gap-2">
          <Button
            v-for="value in startDefaults"
            :key="value"
            type="button"
            size="sm"
            :variant="form.startTime === value ? 'default' : 'outline'"
            @click="applyStartDefault(value)"
          >
            {{ value }}
          </Button>
        </div>
      </div>

      <div class="grid grid-cols-3 gap-4" v-if="form.entryType === 'WORK'">
        <div class="space-y-2">
          <Label for="startTime">Beginn</Label>
          <Input
            id="startTime"
            v-model="form.startTime"
            type="time"
            required
            :disabled="form.shiftKind === 'LATE'"
          />
        </div>
        <div class="space-y-2">
          <Label for="endTime">Ende</Label>
          <Input
            id="endTime"
            v-model="form.endTime"
            type="time"
            required
            :disabled="form.shiftKind !== 'MANUAL'"
          />
        </div>
        <div class="space-y-2">
          <Label for="breakMinutes">Pause (Min.)</Label>
          <Input
            id="breakMinutes"
            v-model="form.breakMinutes"
            type="number"
            :min="0"
            :max="120"
          />
        </div>
      </div>

      <div v-if="form.entryType === 'WORK'" class="rounded-md bg-stone-50 px-3 py-2 text-sm text-stone-700">
        Arbeitszeit: <span class="font-medium text-stone-900">{{ calculatedWorkTime }}</span>
        <span class="text-stone-500"> · Pause {{ form.breakMinutes || 0 }} Min.</span>
      </div>

      <div class="space-y-2" v-if="form.entryType === 'WORK' && !isCompactWorkCreate">
        <Label for="group">Gruppe</Label>
        <Select
          :model-value="form.groupId"
          @update:model-value="onGroupChange"
          :options="groupOptions"
          placeholder="Gruppe auswählen"
        />
      </div>

      <div v-if="form.entryType === 'WORK'" class="space-y-2 rounded-md border border-stone-200 p-3">
        <div class="flex items-center justify-between gap-3">
          <div>
            <Label>Springer-Wechsel</Label>
            <p class="text-xs text-stone-500">Optional für Wechsel zwischen Gruppen innerhalb der Schicht.</p>
          </div>
          <Button type="button" size="sm" variant="outline" @click="addSegment">
            Wechsel
          </Button>
        </div>

        <div v-if="form.segments.length" class="space-y-2">
          <div
            v-for="(segment, index) in form.segments"
            :key="index"
            class="grid grid-cols-[1fr_5.8rem_5.8rem_1fr_auto] gap-2 items-end"
          >
            <div class="space-y-1">
              <Label :for="`segment-group-${index}`" class="text-xs">Gruppe</Label>
              <select
                :id="`segment-group-${index}`"
                v-model.number="segment.groupId"
                class="h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm"
              >
                <option v-for="group in groups" :key="group.id" :value="group.id">
                  {{ group.name }}
                </option>
              </select>
            </div>
            <div class="space-y-1">
              <Label :for="`segment-start-${index}`" class="text-xs">Von</Label>
              <Input :id="`segment-start-${index}`" v-model="segment.startTime" type="time" />
            </div>
            <div class="space-y-1">
              <Label :for="`segment-end-${index}`" class="text-xs">Bis</Label>
              <Input :id="`segment-end-${index}`" v-model="segment.endTime" type="time" />
            </div>
            <div class="space-y-1">
              <Label :for="`segment-notes-${index}`" class="text-xs">Notiz</Label>
              <Input :id="`segment-notes-${index}`" v-model="segment.notes" placeholder="optional" />
            </div>
            <Button type="button" size="sm" variant="ghost" @click="removeSegment(index)">
              Entfernen
            </Button>
          </div>
        </div>
      </div>

      <div class="space-y-2">
        <Label for="notes">Notizen</Label>
        <Input
          id="notes"
          v-model="form.notes"
          placeholder="Optionale Notizen"
        />
      </div>

      <div class="flex justify-between gap-3 pt-4">
        <Button 
          v-if="isEditing"
          type="button" 
          variant="destructive" 
          @click="handleDelete"
        >
          Löschen
        </Button>
        <div class="flex-1" />
        <Button type="button" variant="outline" @click="emit('update:open', false)">
          Abbrechen
        </Button>
        <Button type="submit" :disabled="!isFormValid">
          {{ isEditing ? 'Speichern' : 'Erstellen' }}
        </Button>
      </div>
    </form>
  </Dialog>
</template>
