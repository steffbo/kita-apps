import { computed } from 'vue';
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query';
import { apiClient, type Employee, type CreateEmployeeRequest, type UpdateEmployeeRequest, type GroupAssignment } from '../api';

export const employeeKeys = {
  all: ['employees'] as const,
  lists: () => [...employeeKeys.all, 'list'] as const,
  list: (filters: { includeInactive?: boolean }) => [...employeeKeys.lists(), filters] as const,
  details: () => [...employeeKeys.all, 'detail'] as const,
  detail: (id: number) => [...employeeKeys.details(), id] as const,
  assignments: (id: number) => [...employeeKeys.detail(id), 'assignments'] as const,
};

export function useEmployees(includeInactive = false) {
  return useQuery({
    queryKey: employeeKeys.list({ includeInactive }),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/employees', {
        params: { query: { includeInactive } },
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Laden der Mitarbeiter');
      return data as Employee[];
    },
  });
}

export function useEmployee(id: number | (() => number)) {
  const employeeId = computed(() => (typeof id === 'function' ? id() : id));
  
  return useQuery({
    queryKey: computed(() => employeeKeys.detail(employeeId.value)),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/employees/{id}', {
        params: { path: { id: employeeId.value } },
      });
      if (error) throw new Error((error as any)?.message || 'Mitarbeiter nicht gefunden');
      return data as Employee;
    },
    enabled: computed(() => employeeId.value > 0),
  });
}

export function useCreateEmployee() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (employee: CreateEmployeeRequest) => {
      const { data, error } = await apiClient.POST('/employees', {
        body: employee,
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Anlegen des Mitarbeiters');
      return data as Employee;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: employeeKeys.lists() });
    },
  });
}

export function useUpdateEmployee() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id, data: employeeData }: { id: number; data: UpdateEmployeeRequest }) => {
      const { data, error } = await apiClient.PUT('/employees/{id}', {
        params: { path: { id } },
        body: employeeData,
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Aktualisieren des Mitarbeiters');
      return data as Employee;
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: employeeKeys.lists() });
      queryClient.setQueryData(employeeKeys.detail(data!.id!), data);
    },
  });
}

export function useDeleteEmployee() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: number) => {
      const { error } = await apiClient.DELETE('/employees/{id}', {
        params: { path: { id } },
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Deaktivieren des Mitarbeiters');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: employeeKeys.lists() });
    },
  });
}

export function useAdminResetPassword() {
  return useMutation({
    mutationFn: async (id: number) => {
      const { data, error } = await apiClient.POST('/employees/{id}/reset-password', {
        params: { path: { id } },
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Zurücksetzen des Passworts');
      return data;
    },
  });
}

export function useEmployeeAssignments(id: number | (() => number)) {
  const employeeId = computed(() => (typeof id === 'function' ? id() : id));
  
  return useQuery({
    queryKey: computed(() => employeeKeys.assignments(employeeId.value)),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/employees/{id}/assignments', {
        params: { path: { id: employeeId.value } },
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Laden der Gruppenzuordnungen');
      return data as GroupAssignment[];
    },
    enabled: computed(() => employeeId.value > 0),
  });
}
