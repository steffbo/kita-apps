<script setup lang="ts">
import { ref } from 'vue';
import { Plus, Loader2 } from 'lucide-vue-next';
import { 
  useEmployees, 
  useCreateEmployee, 
  useUpdateEmployee, 
  useDeleteEmployee,
  useAdminResetPassword,
  useAuth,
  type Employee,
  type CreateEmployeeRequest,
  type UpdateEmployeeRequest
} from '@kita/shared';
import { Button, Badge } from '@/components/ui';
import EmployeeFormDialog from '@/components/EmployeeFormDialog.vue';

const { isAdmin } = useAuth();

// Queries
const { data: employees, isLoading, error, refetch } = useEmployees(false);

// Mutations
const createEmployee = useCreateEmployee();
const updateEmployee = useUpdateEmployee();
const deleteEmployee = useDeleteEmployee();
const resetPassword = useAdminResetPassword();

// Dialog state
const dialogOpen = ref(false);
const selectedEmployee = ref<Employee | null>(null);
const showDeleteConfirm = ref(false);
const employeeToDelete = ref<Employee | null>(null);

function openCreateDialog() {
  selectedEmployee.value = null;
  dialogOpen.value = true;
}

function openEditDialog(employee: Employee) {
  selectedEmployee.value = employee;
  dialogOpen.value = true;
}

async function handleSave(data: CreateEmployeeRequest | UpdateEmployeeRequest) {
  try {
    if (selectedEmployee.value?.id) {
      await updateEmployee.mutateAsync({
        id: selectedEmployee.value.id,
        data: data as UpdateEmployeeRequest,
      });
    } else {
      await createEmployee.mutateAsync(data as CreateEmployeeRequest);
    }
    dialogOpen.value = false;
  } catch (err) {
    console.error('Failed to save employee:', err);
  }
}

function confirmDelete(employee: Employee) {
  employeeToDelete.value = employee;
  showDeleteConfirm.value = true;
}

async function handleDelete() {
  if (!employeeToDelete.value?.id) return;
  
  try {
    await deleteEmployee.mutateAsync(employeeToDelete.value.id);
    showDeleteConfirm.value = false;
    employeeToDelete.value = null;
  } catch (err) {
    console.error('Failed to delete employee:', err);
  }
}

async function handleResetPassword(employee: Employee) {
  if (!employee.id) return;
  
  if (confirm(`Passwort für ${employee.firstName} ${employee.lastName} zurücksetzen?`)) {
    try {
      await resetPassword.mutateAsync(employee.id);
      alert('Passwort-Reset-E-Mail wurde versendet.');
    } catch (err) {
      console.error('Failed to reset password:', err);
    }
  }
}
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-stone-900">Mitarbeiter</h1>
        <p class="text-stone-600">Verwalten Sie alle Mitarbeiter der Kita</p>
      </div>
      <Button v-if="isAdmin" @click="openCreateDialog">
        <Plus class="w-4 h-4 mr-2" />
        Neuer Mitarbeiter
      </Button>
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="w-8 h-8 animate-spin text-primary" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="bg-destructive/10 text-destructive rounded-lg p-4">
      <p>Fehler beim Laden der Mitarbeiter: {{ (error as Error).message }}</p>
      <Button variant="outline" size="sm" class="mt-2" @click="refetch()">
        Erneut versuchen
      </Button>
    </div>

    <!-- Empty state -->
    <div v-else-if="!employees?.length" class="text-center py-12 bg-white rounded-lg border border-stone-200">
      <p class="text-stone-600">Noch keine Mitarbeiter vorhanden.</p>
      <Button v-if="isAdmin" class="mt-4" @click="openCreateDialog">
        <Plus class="w-4 h-4 mr-2" />
        Ersten Mitarbeiter anlegen
      </Button>
    </div>

    <!-- Employee table -->
    <div v-else class="bg-white rounded-lg border border-stone-200 overflow-hidden">
      <table class="w-full">
        <thead>
          <tr class="bg-stone-50 border-b border-stone-200">
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Name</th>
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">E-Mail</th>
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Rolle</th>
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Wochenstunden</th>
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Resturlaub</th>
            <th class="px-4 py-3 text-left text-sm font-medium text-stone-600">Status</th>
            <th v-if="isAdmin" class="px-4 py-3 text-right text-sm font-medium text-stone-600">Aktionen</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="employee in employees"
            :key="employee.id"
            class="border-b border-stone-200 last:border-b-0 hover:bg-stone-50"
          >
            <td class="px-4 py-3">
              <div class="flex items-center gap-3">
                <div class="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
                  <span class="text-sm font-medium text-primary">
                    {{ employee.firstName?.[0] }}{{ employee.lastName?.[0] }}
                  </span>
                </div>
                <span class="font-medium text-stone-900">
                  {{ employee.firstName }} {{ employee.lastName }}
                </span>
              </div>
            </td>
            <td class="px-4 py-3 text-stone-600">{{ employee.email }}</td>
            <td class="px-4 py-3">
              <Badge
                :class="employee.role === 'ADMIN' ? 'bg-purple-100 text-purple-700' : 'bg-stone-100 text-stone-700'"
                variant="outline"
              >
                {{ employee.role === 'ADMIN' ? 'Leitung' : 'Mitarbeiter' }}
              </Badge>
            </td>
            <td class="px-4 py-3 text-stone-600">{{ employee.weeklyHours }} Std.</td>
            <td class="px-4 py-3 text-stone-600">
              {{ employee.remainingVacationDays ?? '-' }} Tage
            </td>
            <td class="px-4 py-3">
              <Badge
                :class="employee.active ? 'bg-green-100 text-green-700' : 'bg-stone-100 text-stone-500'"
                variant="outline"
              >
                {{ employee.active ? 'Aktiv' : 'Inaktiv' }}
              </Badge>
            </td>
            <td v-if="isAdmin" class="px-4 py-3 text-right">
              <div class="flex items-center justify-end gap-2">
                <Button variant="ghost" size="sm" @click="openEditDialog(employee)">
                  Bearbeiten
                </Button>
                <Button variant="ghost" size="sm" @click="handleResetPassword(employee)">
                  Passwort
                </Button>
                <Button 
                  variant="ghost" 
                  size="sm" 
                  class="text-destructive hover:text-destructive"
                  @click="confirmDelete(employee)"
                >
                  Deaktivieren
                </Button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Employee Form Dialog -->
    <EmployeeFormDialog
      v-model:open="dialogOpen"
      :employee="selectedEmployee"
      @save="handleSave"
    />

    <!-- Delete Confirmation Dialog -->
    <Teleport to="body">
      <div
        v-if="showDeleteConfirm"
        class="fixed inset-0 z-50 flex items-center justify-center"
      >
        <div 
          class="fixed inset-0 bg-black/50" 
          @click="showDeleteConfirm = false"
        />
        <div class="relative bg-white rounded-lg p-6 max-w-md w-full mx-4 shadow-xl">
          <h3 class="text-lg font-semibold mb-2">Mitarbeiter deaktivieren?</h3>
          <p class="text-stone-600 mb-4">
            Möchten Sie <strong>{{ employeeToDelete?.firstName }} {{ employeeToDelete?.lastName }}</strong> wirklich deaktivieren?
            Der Mitarbeiter kann später wieder aktiviert werden.
          </p>
          <div class="flex justify-end gap-3">
            <Button variant="outline" @click="showDeleteConfirm = false">
              Abbrechen
            </Button>
            <Button 
              variant="destructive" 
              @click="handleDelete"
              :disabled="deleteEmployee.isPending.value"
            >
              <Loader2 v-if="deleteEmployee.isPending.value" class="w-4 h-4 mr-2 animate-spin" />
              Deaktivieren
            </Button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
