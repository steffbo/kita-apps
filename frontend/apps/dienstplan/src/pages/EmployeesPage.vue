<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { Plus, Loader2, Pencil, KeyRound, UserX, UserCheck, Trash2, ArrowUp, ArrowDown, ArrowUpDown } from 'lucide-vue-next';
import { 
  useEmployees, 
  useGroups,
  useCreateEmployee, 
  useUpdateEmployee, 
  useDeleteEmployee,
  usePermanentDeleteEmployee,
  useAdminResetPassword,
  useAuth,
  type Employee,
  type CreateEmployeeRequest,
  type UpdateEmployeeRequest
} from '@kita/shared';
import { Button, Badge } from '@/components/ui';
import EmployeeFormDialog from '@/components/EmployeeFormDialog.vue';

const { isAdmin } = useAuth();

// Show inactive toggle
const showInactive = ref(false);

// Sorting state
type SortField = 'name' | 'role' | 'group' | 'weeklyHours' | 'remainingVacationDays' | 'status';
type SortDirection = 'asc' | 'desc';
const sortField = ref<SortField>('name');
const sortDirection = ref<SortDirection>('asc');

function toggleSort(field: SortField) {
  if (sortField.value === field) {
    sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc';
  } else {
    sortField.value = field;
    sortDirection.value = 'asc';
  }
}

// Queries - fetch all employees when showInactive is true
const { data: allEmployees, isLoading, error, refetch } = useEmployees(true);
const { data: groups } = useGroups();

// Filter and sort employees
const employees = computed(() => {
  if (!allEmployees.value) return [];
  
  let filtered = showInactive.value 
    ? [...allEmployees.value] 
    : allEmployees.value.filter(e => e.active);
  
  // Sort
  filtered.sort((a, b) => {
    let comparison = 0;
    
    switch (sortField.value) {
      case 'name':
        comparison = `${a.lastName} ${a.firstName}`.localeCompare(`${b.lastName} ${b.firstName}`);
        break;
      case 'role':
        comparison = (a.role ?? '').localeCompare(b.role ?? '');
        break;
      case 'group':
        comparison = (a.primaryGroup?.name ?? 'zzz').localeCompare(b.primaryGroup?.name ?? 'zzz');
        break;
      case 'weeklyHours':
        comparison = (a.weeklyHours ?? 0) - (b.weeklyHours ?? 0);
        break;
      case 'remainingVacationDays':
        comparison = (a.remainingVacationDays ?? 0) - (b.remainingVacationDays ?? 0);
        break;
      case 'status':
        comparison = (a.active === b.active) ? 0 : a.active ? -1 : 1;
        break;
    }
    
    return sortDirection.value === 'asc' ? comparison : -comparison;
  });
  
  return filtered;
});

// Mutations
const createEmployee = useCreateEmployee();
const updateEmployee = useUpdateEmployee();
const deleteEmployee = useDeleteEmployee();
const permanentDeleteEmployee = usePermanentDeleteEmployee();
const resetPassword = useAdminResetPassword();

// Dialog state
const dialogOpen = ref(false);
const selectedEmployee = ref<Employee | null>(null);
const showDeleteConfirm = ref(false);
const employeeToDelete = ref<Employee | null>(null);
const deleteMode = ref<'deactivate' | 'permanent'>('deactivate');

function openCreateDialog() {
  selectedEmployee.value = null;
  dialogOpen.value = true;
}

function openEditDialog(employee: Employee) {
  selectedEmployee.value = employee;
  dialogOpen.value = true;
}

function handleRowClick(employee: Employee, event: MouseEvent) {
  // Don't open dialog if clicking on action buttons
  const target = event.target as HTMLElement;
  if (target.closest('button')) return;
  
  if (isAdmin.value) {
    openEditDialog(employee);
  }
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
  deleteMode.value = 'deactivate';
  showDeleteConfirm.value = true;
}

function confirmPermanentDelete(employee: Employee) {
  employeeToDelete.value = employee;
  deleteMode.value = 'permanent';
  showDeleteConfirm.value = true;
}

async function handleDelete() {
  if (!employeeToDelete.value?.id) return;
  
  try {
    if (deleteMode.value === 'permanent') {
      await permanentDeleteEmployee.mutateAsync(employeeToDelete.value.id);
    } else {
      await deleteEmployee.mutateAsync(employeeToDelete.value.id);
    }
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

async function handleActivate(employee: Employee) {
  if (!employee.id) return;

  try {
    await updateEmployee.mutateAsync({
      id: employee.id,
      data: { active: true },
    });
  } catch (err) {
    console.error('Failed to activate employee:', err);
  }
}

// ESC key handler to close modals
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (showDeleteConfirm.value) showDeleteConfirm.value = false;
    else if (dialogOpen.value) dialogOpen.value = false;
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown);
});
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-stone-900">Mitarbeiter</h1>
        <p class="text-stone-600">Verwalte alle Mitarbeiter der Kita</p>
      </div>
      <div class="flex items-center gap-3">
        <label v-if="isAdmin" class="flex items-center gap-2 text-sm text-stone-600 cursor-pointer">
          <input 
            type="checkbox" 
            v-model="showInactive"
            class="rounded border-stone-300 text-primary focus:ring-primary"
          />
          Inaktive anzeigen
        </label>
        <Button v-if="isAdmin" @click="openCreateDialog">
          <Plus class="w-4 h-4 mr-2" />
          Neuer Mitarbeiter
        </Button>
      </div>
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
            <th 
              class="px-4 py-3 text-left text-sm font-medium text-stone-600 cursor-pointer hover:bg-stone-100 select-none"
              @click="toggleSort('name')"
            >
              <div class="flex items-center gap-1">
                Name
                <ArrowUp v-if="sortField === 'name' && sortDirection === 'asc'" class="w-4 h-4" />
                <ArrowDown v-else-if="sortField === 'name' && sortDirection === 'desc'" class="w-4 h-4" />
                <ArrowUpDown v-else class="w-4 h-4 text-stone-400" />
              </div>
            </th>
            <th 
              class="px-4 py-3 text-left text-sm font-medium text-stone-600 cursor-pointer hover:bg-stone-100 select-none"
              @click="toggleSort('role')"
            >
              <div class="flex items-center gap-1">
                Rolle
                <ArrowUp v-if="sortField === 'role' && sortDirection === 'asc'" class="w-4 h-4" />
                <ArrowDown v-else-if="sortField === 'role' && sortDirection === 'desc'" class="w-4 h-4" />
                <ArrowUpDown v-else class="w-4 h-4 text-stone-400" />
              </div>
            </th>
            <th 
              class="px-4 py-3 text-left text-sm font-medium text-stone-600 cursor-pointer hover:bg-stone-100 select-none"
              @click="toggleSort('group')"
            >
              <div class="flex items-center gap-1">
                Stammgruppe
                <ArrowUp v-if="sortField === 'group' && sortDirection === 'asc'" class="w-4 h-4" />
                <ArrowDown v-else-if="sortField === 'group' && sortDirection === 'desc'" class="w-4 h-4" />
                <ArrowUpDown v-else class="w-4 h-4 text-stone-400" />
              </div>
            </th>
            <th 
              class="px-4 py-3 text-left text-sm font-medium text-stone-600 cursor-pointer hover:bg-stone-100 select-none"
              @click="toggleSort('weeklyHours')"
            >
              <div class="flex items-center gap-1">
                Wochenstunden
                <ArrowUp v-if="sortField === 'weeklyHours' && sortDirection === 'asc'" class="w-4 h-4" />
                <ArrowDown v-else-if="sortField === 'weeklyHours' && sortDirection === 'desc'" class="w-4 h-4" />
                <ArrowUpDown v-else class="w-4 h-4 text-stone-400" />
              </div>
            </th>
            <th 
              class="px-4 py-3 text-left text-sm font-medium text-stone-600 cursor-pointer hover:bg-stone-100 select-none"
              @click="toggleSort('remainingVacationDays')"
            >
              <div class="flex items-center gap-1">
                Resturlaub
                <ArrowUp v-if="sortField === 'remainingVacationDays' && sortDirection === 'asc'" class="w-4 h-4" />
                <ArrowDown v-else-if="sortField === 'remainingVacationDays' && sortDirection === 'desc'" class="w-4 h-4" />
                <ArrowUpDown v-else class="w-4 h-4 text-stone-400" />
              </div>
            </th>
            <th 
              class="px-4 py-3 text-left text-sm font-medium text-stone-600 cursor-pointer hover:bg-stone-100 select-none"
              @click="toggleSort('status')"
            >
              <div class="flex items-center gap-1">
                Status
                <ArrowUp v-if="sortField === 'status' && sortDirection === 'asc'" class="w-4 h-4" />
                <ArrowDown v-else-if="sortField === 'status' && sortDirection === 'desc'" class="w-4 h-4" />
                <ArrowUpDown v-else class="w-4 h-4 text-stone-400" />
              </div>
            </th>
            <th v-if="isAdmin" class="px-4 py-3 text-right text-sm font-medium text-stone-600">Aktionen</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="employee in employees"
            :key="employee.id"
            :class="[
              'border-b border-stone-200 last:border-b-0 transition-colors',
              isAdmin ? 'cursor-pointer hover:bg-stone-50' : 'hover:bg-stone-50/50'
            ]"
            @click="handleRowClick(employee, $event)"
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
            <td class="px-4 py-3">
              <Badge
                :class="employee.role === 'ADMIN' ? 'bg-purple-100 text-purple-700' : 'bg-stone-100 text-stone-700'"
                variant="outline"
              >
                {{ employee.role === 'ADMIN' ? 'Leitung' : 'Mitarbeiter' }}
              </Badge>
            </td>
            <td class="px-4 py-3">
              <Badge 
                v-if="employee.primaryGroup?.name"
                variant="outline"
                :style="{
                  backgroundColor: employee.primaryGroup.color ? `${employee.primaryGroup.color}20` : undefined,
                  color: employee.primaryGroup.color || undefined,
                  borderColor: employee.primaryGroup.color ? `${employee.primaryGroup.color}40` : undefined,
                }"
              >
                {{ employee.primaryGroup.name }}
              </Badge>
              <Badge v-else variant="outline" class="bg-stone-100 text-stone-500 border-stone-200">
                Springer
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
              <div class="flex items-center justify-end gap-1">
                <Button 
                  variant="ghost" 
                  size="icon" 
                  class="h-8 w-8"
                  title="Bearbeiten"
                  aria-label="Bearbeiten"
                  @click.stop="openEditDialog(employee)"
                >
                  <Pencil class="w-4 h-4" />
                </Button>
                <Button 
                  variant="ghost" 
                  size="icon" 
                  class="h-8 w-8"
                  title="Passwort zurücksetzen"
                  aria-label="Passwort zurücksetzen"
                  @click.stop="handleResetPassword(employee)"
                >
                  <KeyRound class="w-4 h-4" />
                </Button>
                <Button 
                  v-if="employee.active"
                  variant="ghost" 
                  size="icon" 
                  class="h-8 w-8 text-destructive hover:text-destructive"
                  title="Deaktivieren"
                  aria-label="Deaktivieren"
                  @click.stop="confirmDelete(employee)"
                >
                  <UserX class="w-4 h-4" />
                </Button>
                <Button 
                  v-else
                  variant="ghost" 
                  size="icon" 
                  class="h-8 w-8 text-green-600 hover:text-green-700"
                  title="Aktivieren"
                  aria-label="Aktivieren"
                  @click.stop="handleActivate(employee)"
                >
                  <UserCheck class="w-4 h-4" />
                </Button>
                <Button 
                  variant="ghost" 
                  size="icon" 
                  class="h-8 w-8 text-destructive hover:text-destructive"
                  title="Endgültig löschen"
                  aria-label="Endgültig löschen"
                  @click.stop="confirmPermanentDelete(employee)"
                >
                  <Trash2 class="w-4 h-4" />
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
      :groups="groups ?? []"
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
          <!-- Deactivate mode -->
          <template v-if="deleteMode === 'deactivate'">
            <h3 class="text-lg font-semibold mb-2">Mitarbeiter deaktivieren?</h3>
            <p class="text-stone-600 mb-4">
              Möchtest du <strong>{{ employeeToDelete?.firstName }} {{ employeeToDelete?.lastName }}</strong> wirklich deaktivieren?
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
          </template>
          
          <!-- Permanent delete mode -->
          <template v-else>
            <h3 class="text-lg font-semibold mb-2 text-destructive">Mitarbeiter endgültig löschen?</h3>
            <p class="text-stone-600 mb-2">
              Möchtest du <strong>{{ employeeToDelete?.firstName }} {{ employeeToDelete?.lastName }}</strong> wirklich <strong class="text-destructive">endgültig löschen</strong>?
            </p>
            <p class="text-sm text-destructive bg-destructive/10 rounded p-3 mb-4">
              <strong>Achtung:</strong> Diese Aktion kann nicht rückgängig gemacht werden! 
              Alle zugehörigen Daten (Dienstpläne, Zeiterfassungen, Gruppenzuordnungen) werden ebenfalls gelöscht.
            </p>
            <div class="flex justify-end gap-3">
              <Button variant="outline" @click="showDeleteConfirm = false">
                Abbrechen
              </Button>
              <Button 
                variant="destructive" 
                @click="handleDelete"
                :disabled="permanentDeleteEmployee.isPending.value"
              >
                <Loader2 v-if="permanentDeleteEmployee.isPending.value" class="w-4 h-4 mr-2 animate-spin" />
                Endgültig löschen
              </Button>
            </div>
          </template>
        </div>
      </div>
    </Teleport>
  </div>
</template>
