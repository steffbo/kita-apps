<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { Dialog, Button, Input, Label, Select, type SelectOption } from '@/components/ui';
import type { Employee, CreateEmployeeRequest, UpdateEmployeeRequest, Group } from '@kita/shared';

const props = defineProps<{
  open: boolean;
  employee?: Employee | null;
  groups: Group[];
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

const groupOptions = computed<SelectOption[]>(() => 
  props.groups.map(g => ({
    value: String(g.id),
    label: g.name || '',
  }))
);

// Form state
const form = ref({
  email: '',
  firstName: '',
  lastName: '',
  role: 'EMPLOYEE' as 'EMPLOYEE' | 'ADMIN',
  weeklyHours: 38,
  vacationDaysPerYear: 30,
  primaryGroupId: '',
});

// Get Springer group ID for default selection
const springerGroupId = computed(() => {
  const springer = props.groups.find(g => g.name === 'Springer');
  return springer ? String(springer.id) : '';
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
          primaryGroupId: props.employee.primaryGroupId ? String(props.employee.primaryGroupId) : springerGroupId.value,
        };
      } else {
        form.value = {
          email: '',
          firstName: '',
          lastName: '',
          role: 'EMPLOYEE',
          weeklyHours: 38,
          vacationDaysPerYear: 30,
          primaryGroupId: springerGroupId.value,
        };
      }
    }
  },
  { immediate: true }
);

function handleSubmit() {
  const data: CreateEmployeeRequest | UpdateEmployeeRequest = {
    email: form.value.email,
    firstName: form.value.firstName,
    lastName: form.value.lastName,
    role: form.value.role,
    weeklyHours: form.value.weeklyHours,
    vacationDaysPerYear: form.value.vacationDaysPerYear,
    primaryGroupId: parseInt(form.value.primaryGroupId),
  };
  emit('save', data);
}
</script>

<template>
  <Dialog
    :open="open"
    @update:open="emit('update:open', $event)"
    :title="isEditing ? 'Mitarbeiter bearbeiten' : 'Neuer Mitarbeiter'"
    :description="isEditing ? 'Bearbeite die Daten des Mitarbeiters.' : 'Lege einen neuen Mitarbeiter an.'"
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
        <Label for="primaryGroup">Stammgruppe</Label>
        <Select
          v-model="form.primaryGroupId"
          :options="groupOptions"
          placeholder="Stammgruppe auswählen"
        />
        <p class="text-xs text-muted-foreground">
          Die Stammgruppe wird automatisch im Dienstplan vorausgewählt.
        </p>
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
