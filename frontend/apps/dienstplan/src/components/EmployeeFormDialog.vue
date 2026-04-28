<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { Dialog, Button, Input, Label, Select, type SelectOption } from '@/components/ui';
import type { Employee, CreateEmployeeRequest, UpdateEmployeeRequest, Group, EmployeeContractRequest } from '@kita/shared';

const props = defineProps<{
  open: boolean;
  employee?: Employee | null;
  groups: Group[];
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  'save': [data: { employee: CreateEmployeeRequest | UpdateEmployeeRequest; contract: EmployeeContractRequest }];
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
  nickname: '',
  role: 'EMPLOYEE' as 'EMPLOYEE' | 'ADMIN',
  weeklyHours: 38,
  vacationDaysPerYear: 30,
  primaryGroupId: '',
  contractValidFrom: new Date().toISOString().slice(0, 7),
  workdays: [
    { weekday: 1, active: true, plannedHours: 7 },
    { weekday: 2, active: true, plannedHours: 7 },
    { weekday: 3, active: true, plannedHours: 7 },
    { weekday: 4, active: true, plannedHours: 7 },
    { weekday: 5, active: true, plannedHours: 7 },
  ],
});

// Get Springer group ID for default selection
const springerGroupId = computed(() => {
  const springer = props.groups.find(g => g.name === 'Springer');
  return springer ? String(springer.id) : '';
});

const weekdayLabels = ['Mo', 'Di', 'Mi', 'Do', 'Fr'];

function currentMonthValue() {
  return new Date().toISOString().slice(0, 7);
}

function hoursFromMinutes(minutes?: number) {
  return Math.round(((minutes || 0) / 60) * 100) / 100;
}

function applyDefaultPattern() {
  const hours = Number(form.value.weeklyHours) || 0;
  if (hours === 33) {
    form.value.workdays = [
      { weekday: 1, active: true, plannedHours: 7 },
      { weekday: 2, active: true, plannedHours: 7 },
      { weekday: 3, active: true, plannedHours: 7 },
      { weekday: 4, active: true, plannedHours: 6 },
      { weekday: 5, active: true, plannedHours: 6 },
    ];
    return;
  }
  const daily = Math.round((hours / 5) * 100) / 100;
  form.value.workdays = [1, 2, 3, 4, 5].map(weekday => ({ weekday, active: true, plannedHours: daily }));
}

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
          nickname: props.employee.nickname || '',
          role: (props.employee.role as 'EMPLOYEE' | 'ADMIN') || 'EMPLOYEE',
          weeklyHours: props.employee.weeklyHours || 38,
          vacationDaysPerYear: props.employee.vacationDaysPerYear || 30,
          primaryGroupId: props.employee.primaryGroupId ? String(props.employee.primaryGroupId) : springerGroupId.value,
          contractValidFrom: props.employee.currentContract?.validFrom?.slice(0, 7) || currentMonthValue(),
          workdays: [1, 2, 3, 4, 5].map(weekday => {
            const workday = props.employee?.workPattern?.find(day => day.weekday === weekday);
            return {
              weekday,
              active: !!workday,
              plannedHours: hoursFromMinutes(workday?.plannedMinutes),
            };
          }),
        };
      } else {
        form.value = {
          email: '',
          firstName: '',
          lastName: '',
          nickname: '',
          role: 'EMPLOYEE',
          weeklyHours: 38,
          vacationDaysPerYear: 30,
          primaryGroupId: springerGroupId.value,
          contractValidFrom: currentMonthValue(),
          workdays: [
            { weekday: 1, active: true, plannedHours: 7.6 },
            { weekday: 2, active: true, plannedHours: 7.6 },
            { weekday: 3, active: true, plannedHours: 7.6 },
            { weekday: 4, active: true, plannedHours: 7.6 },
            { weekday: 5, active: true, plannedHours: 7.6 },
          ],
        };
      }
    }
  },
  { immediate: true }
);

function handleSubmit() {
  const employee: CreateEmployeeRequest | UpdateEmployeeRequest = {
    email: form.value.email,
    firstName: form.value.firstName,
    lastName: form.value.lastName,
    nickname: form.value.nickname || undefined,
    role: form.value.role,
    weeklyHours: form.value.weeklyHours,
    vacationDaysPerYear: form.value.vacationDaysPerYear,
    primaryGroupId: parseInt(form.value.primaryGroupId),
  };
  const contract: EmployeeContractRequest = {
    validFrom: `${form.value.contractValidFrom}-01`,
    weeklyHours: Number(form.value.weeklyHours),
    workdays: form.value.workdays
      .filter(day => day.active)
      .map(day => ({
        weekday: day.weekday,
        plannedMinutes: Math.round(Number(day.plannedHours) * 60),
      })),
  };
  emit('save', { employee, contract });
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

      <div class="space-y-2">
        <Label for="nickname">Spitzname</Label>
        <Input
          id="nickname"
          v-model="form.nickname"
          placeholder="Optional"
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
            :max="40"
            step="0.5"
            required
          />
        </div>
      </div>

      <div class="space-y-3 rounded-md border border-stone-200 p-3">
        <div class="grid grid-cols-2 gap-4">
          <div class="space-y-2">
            <Label for="contractValidFrom">Gültig ab</Label>
            <Input
              id="contractValidFrom"
              v-model="form.contractValidFrom"
              type="month"
              required
            />
          </div>
          <div class="flex items-end">
            <Button type="button" variant="outline" class="w-full" @click="applyDefaultPattern">
              Muster berechnen
            </Button>
          </div>
        </div>

        <div class="grid grid-cols-5 gap-2">
          <label
            v-for="(day, index) in form.workdays"
            :key="day.weekday"
            class="rounded-md border border-stone-200 p-2 text-sm"
          >
            <div class="flex items-center gap-2">
              <input type="checkbox" v-model="day.active" class="rounded border-stone-300" />
              <span class="font-medium">{{ weekdayLabels[index] }}</span>
            </div>
            <Input
              v-model="day.plannedHours"
              type="number"
              step="0.25"
              min="0"
              max="10"
              class="mt-2 h-8"
              :disabled="!day.active"
            />
          </label>
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
