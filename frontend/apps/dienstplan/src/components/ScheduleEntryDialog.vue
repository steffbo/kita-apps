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

const employeeOptions = computed<SelectOption[]>(() => 
  props.employees.map(e => ({
    value: String(e.id),
    label: `${e.firstName} ${e.lastName}`,
  }))
);

const groupOptions = computed<SelectOption[]>(() => [
  { value: 'none', label: 'Keine Gruppe (Springer)' },
  ...props.groups.map(g => ({
    value: String(g.id),
    label: g.name || '',
  })),
]);

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

// Reset form when dialog opens/entry changes
watch(
  () => [props.open, props.entry, props.defaultDate, props.defaultGroupId],
  () => {
    if (props.open) {
      if (props.entry) {
        form.value = {
          employeeId: String(props.entry.employeeId || ''),
          date: props.entry.date || '',
          startTime: props.entry.startTime?.substring(0, 5) || '07:00',
          endTime: props.entry.endTime?.substring(0, 5) || '15:00',
          breakMinutes: props.entry.breakMinutes || 30,
          groupId: props.entry.groupId ? String(props.entry.groupId) : '',
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

function handleSubmit() {
  const data: CreateScheduleEntryRequest = {
    employeeId: parseInt(form.value.employeeId),
    date: form.value.date,
    startTime: form.value.startTime + ':00',
    endTime: form.value.endTime + ':00',
    breakMinutes: form.value.breakMinutes,
    groupId: form.value.groupId && form.value.groupId !== 'none' ? parseInt(form.value.groupId) : undefined,
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
    :description="isEditing ? 'Bearbeiten Sie den Dienstplan-Eintrag.' : 'Erstellen Sie einen neuen Dienstplan-Eintrag.'"
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
          v-model="form.groupId"
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
        <Button type="submit">
          {{ isEditing ? 'Speichern' : 'Erstellen' }}
        </Button>
      </div>
    </form>
  </Dialog>
</template>
