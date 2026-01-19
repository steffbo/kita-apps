<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { Dialog, Button, Input, Label, Select, type SelectOption } from '@/components/ui';
import type { Employee, CreateEmployeeRequest, UpdateEmployeeRequest } from '@kita/shared';

const props = defineProps<{
  open: boolean;
  employee?: Employee | null;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  'save': [data: CreateEmployeeRequest | UpdateEmployeeRequest];
}>();

const isEditing = computed(() => !!props.employee?.id);

const roleOptions: SelectOption[] = [
  { value: 'EMPLOYEE', label: 'Mitarbeiter' },
  { value: 'ADMIN', label: 'Leitung' },
];

// Form state
const form = ref({
  email: '',
  firstName: '',
  lastName: '',
  role: 'EMPLOYEE' as 'EMPLOYEE' | 'ADMIN',
  weeklyHours: 38,
  vacationDaysPerYear: 30,
});

// Reset form when dialog opens/employee changes
watch(
  () => [props.open, props.employee],
  () => {
    if (props.open) {
      if (props.employee) {
        form.value = {
          email: props.employee.email || '',
          firstName: props.employee.firstName || '',
          lastName: props.employee.lastName || '',
          role: (props.employee.role as 'EMPLOYEE' | 'ADMIN') || 'EMPLOYEE',
          weeklyHours: props.employee.weeklyHours || 38,
          vacationDaysPerYear: props.employee.vacationDaysPerYear || 30,
        };
      } else {
        form.value = {
          email: '',
          firstName: '',
          lastName: '',
          role: 'EMPLOYEE',
          weeklyHours: 38,
          vacationDaysPerYear: 30,
        };
      }
    }
  },
  { immediate: true }
);

function handleSubmit() {
  emit('save', { ...form.value });
}
</script>

<template>
  <Dialog
    :open="open"
    @update:open="emit('update:open', $event)"
    :title="isEditing ? 'Mitarbeiter bearbeiten' : 'Neuer Mitarbeiter'"
    :description="isEditing ? 'Bearbeiten Sie die Daten des Mitarbeiters.' : 'Legen Sie einen neuen Mitarbeiter an.'"
  >
    <form @submit.prevent="handleSubmit" class="space-y-4">
      <div class="grid grid-cols-2 gap-4">
        <div class="space-y-2">
          <Label for="firstName">Vorname</Label>
          <Input
            id="firstName"
            v-model="form.firstName"
            placeholder="Vorname"
            required
          />
        </div>
        <div class="space-y-2">
          <Label for="lastName">Nachname</Label>
          <Input
            id="lastName"
            v-model="form.lastName"
            placeholder="Nachname"
            required
          />
        </div>
      </div>

      <div class="space-y-2">
        <Label for="email">E-Mail</Label>
        <Input
          id="email"
          v-model="form.email"
          type="email"
          placeholder="email@knirpsenstadt.de"
          required
        />
      </div>

      <div class="grid grid-cols-2 gap-4">
        <div class="space-y-2">
          <Label for="role">Rolle</Label>
          <Select
            v-model="form.role"
            :options="roleOptions"
            placeholder="Rolle auswählen"
          />
        </div>
        <div class="space-y-2">
          <Label for="weeklyHours">Wochenstunden</Label>
          <Input
            id="weeklyHours"
            v-model="form.weeklyHours"
            type="number"
            :min="20"
            :max="38"
            required
          />
        </div>
      </div>

      <div class="space-y-2">
        <Label for="vacationDays">Urlaubstage pro Jahr</Label>
        <Input
          id="vacationDays"
          v-model="form.vacationDaysPerYear"
          type="number"
          :min="20"
          :max="35"
          required
        />
      </div>

      <div class="flex justify-end gap-3 pt-4">
        <Button type="button" variant="outline" @click="emit('update:open', false)">
          Abbrechen
        </Button>
        <Button type="submit">
          {{ isEditing ? 'Speichern' : 'Anlegen' }}
        </Button>
      </div>
    </form>
  </Dialog>
</template>
