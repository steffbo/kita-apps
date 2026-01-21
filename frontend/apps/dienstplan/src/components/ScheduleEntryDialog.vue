<script setup lang="ts">
import { ref, watch, computed } from 'vue';
import { Dialog, Button, Input, Label, Select, type SelectOption } from '@/components/ui';
import type { 
  ScheduleEntry, 
  CreateScheduleEntryRequest, 
  UpdateScheduleEntryRequest,
  Employee,
  Group 
} from '@kita/shared';
import { toISODateString } from '@kita/shared/utils';

const props = defineProps<{
  open: boolean;
  entry?: ScheduleEntry | null;
  employees: Employee[];
  groups: Group[];
  defaultDate?: Date;
  defaultGroupId?: number;
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
  { value: 'SPECIAL_LEAVE', label: 'Sonderurlaub' },
  { value: 'TRAINING', label: 'Fortbildung' },
  { value: 'EVENT', label: 'Veranstaltung' },
];

const employeeOptions = computed<SelectOption[]>(() => {
  return props.employees.map(e => ({
    value: String(e.id),
    label: `${e.firstName} ${e.lastName}`,
  }));
});

const groupOptions = computed<SelectOption[]>(() => 
  props.groups.map(g => ({
    value: String(g.id),
    label: g.name || '',
  }))
);

// Form state
const form = ref({
  employeeId: '',
  date: '',
  startTime: '07:00',
  endTime: '15:00',
  breakMinutes: 30,
  groupId: '',
  entryType: 'WORK' as 'WORK' | 'VACATION' | 'SICK' | 'SPECIAL_LEAVE' | 'TRAINING' | 'EVENT',
  notes: '',
});

// Track if user manually changed the group (don't override manual selections)
const userChangedGroup = ref(false);

// Reset form when dialog opens/entry changes
watch(
  () => [props.open, props.entry, props.defaultDate, props.defaultGroupId],
  () => {
    if (props.open) {
      userChangedGroup.value = false; // Reset on dialog open
      
      if (props.entry) {
        // Use groupId directly, or fall back to group.id if available
        const entryGroupId = props.entry.groupId ?? props.entry.group?.id;
        form.value = {
          employeeId: String(props.entry.employeeId || ''),
          date: props.entry.date || '',
          startTime: props.entry.startTime?.substring(0, 5) || '07:00',
          endTime: props.entry.endTime?.substring(0, 5) || '15:00',
          breakMinutes: props.entry.breakMinutes ?? 30,
          groupId: entryGroupId ? String(entryGroupId) : '',
          entryType: (props.entry.entryType as any) || 'WORK',
          notes: props.entry.notes || '',
        };
      } else {
        form.value = {
          employeeId: '',
          date: props.defaultDate ? toISODateString(props.defaultDate) : '',
          startTime: '07:00',
          endTime: '15:00',
          breakMinutes: 30,
          groupId: props.defaultGroupId ? String(props.defaultGroupId) : '',
          entryType: 'WORK',
          notes: '',
        };
      }
    }
  },
  { immediate: true }
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

// Form validation - groupId is now required for WORK entries
const isFormValid = computed(() => {
  const baseValid = form.value.employeeId && form.value.date && form.value.entryType;
  if (form.value.entryType === 'WORK') {
    return baseValid && form.value.groupId;
  }
  return baseValid;
});

function handleSubmit() {
  if (!isFormValid.value) {
    return;
  }
  
  const data: CreateScheduleEntryRequest = {
    employeeId: parseInt(form.value.employeeId),
    date: form.value.date,
    startTime: form.value.startTime + ':00',
    endTime: form.value.endTime + ':00',
    breakMinutes: form.value.breakMinutes,
    groupId: form.value.groupId ? parseInt(form.value.groupId) : undefined,
    entryType: form.value.entryType,
    notes: form.value.notes || undefined,
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
      <div class="space-y-2">
        <Label for="employee">Mitarbeiter</Label>
        <Select
          v-model="form.employeeId"
          :options="employeeOptions"
          placeholder="Mitarbeiter auswählen"
        />
      </div>

      <div class="grid grid-cols-2 gap-4">
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

      <div class="grid grid-cols-3 gap-4" v-if="form.entryType === 'WORK'">
        <div class="space-y-2">
          <Label for="startTime">Beginn</Label>
          <Input
            id="startTime"
            v-model="form.startTime"
            type="time"
            required
          />
        </div>
        <div class="space-y-2">
          <Label for="endTime">Ende</Label>
          <Input
            id="endTime"
            v-model="form.endTime"
            type="time"
            required
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

      <div class="space-y-2" v-if="form.entryType === 'WORK'">
        <Label for="group">Gruppe</Label>
        <Select
          :model-value="form.groupId"
          @update:model-value="onGroupChange"
          :options="groupOptions"
          placeholder="Gruppe auswählen"
        />
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
