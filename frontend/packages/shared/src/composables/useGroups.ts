import { computed } from 'vue';
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query';
import { 
  apiClient, 
  type Group, 
  type GroupWithMembers, 
  type CreateGroupRequest, 
  type GroupAssignment,
  type GroupAssignmentRequest 
} from '../api';

export const groupKeys = {
  all: ['groups'] as const,
  lists: () => [...groupKeys.all, 'list'] as const,
  list: () => [...groupKeys.lists()] as const,
  details: () => [...groupKeys.all, 'detail'] as const,
  detail: (id: number) => [...groupKeys.details(), id] as const,
  assignments: (id: number) => [...groupKeys.detail(id), 'assignments'] as const,
};

export function useGroups() {
  return useQuery({
    queryKey: groupKeys.list(),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/groups');
      if (error) throw new Error((error as any)?.message || 'Fehler beim Laden der Gruppen');
      return data as Group[];
    },
  });
}

export function useGroup(id: number | (() => number)) {
  const groupId = computed(() => (typeof id === 'function' ? id() : id));
  
  return useQuery({
    queryKey: computed(() => groupKeys.detail(groupId.value)),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/groups/{id}', {
        params: { path: { id: groupId.value } },
      });
      if (error) throw new Error((error as any)?.message || 'Gruppe nicht gefunden');
      return data as GroupWithMembers;
    },
    enabled: computed(() => groupId.value > 0),
  });
}

export function useGroupAssignments(id: number | (() => number)) {
  const groupId = computed(() => (typeof id === 'function' ? id() : id));
  
  return useQuery({
    queryKey: computed(() => groupKeys.assignments(groupId.value)),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/groups/{id}/assignments', {
        params: { path: { id: groupId.value } },
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Laden der Zuordnungen');
      return data as GroupAssignment[];
    },
    enabled: computed(() => groupId.value > 0),
  });
}

export function useCreateGroup() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (group: CreateGroupRequest) => {
      const { data, error } = await apiClient.POST('/groups', {
        body: group,
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Anlegen der Gruppe');
      return data as Group;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: groupKeys.lists() });
    },
  });
}

export function useUpdateGroup() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id, data: groupData }: { id: number; data: CreateGroupRequest }) => {
      const { data, error } = await apiClient.PUT('/groups/{id}', {
        params: { path: { id } },
        body: groupData,
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Aktualisieren der Gruppe');
      return data as Group;
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: groupKeys.lists() });
      queryClient.invalidateQueries({ queryKey: groupKeys.detail(data!.id!) });
    },
  });
}

export function useDeleteGroup() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: number) => {
      const { error } = await apiClient.DELETE('/groups/{id}', {
        params: { path: { id } },
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Löschen der Gruppe');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: groupKeys.lists() });
    },
  });
}

export function useUpdateGroupAssignments() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id, assignments }: { id: number; assignments: GroupAssignmentRequest[] }) => {
      const { data, error } = await apiClient.PUT('/groups/{id}/assignments', {
        params: { path: { id } },
        body: assignments,
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Aktualisieren der Zuordnungen');
      return data as GroupAssignment[];
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: groupKeys.assignments(variables.id) });
      queryClient.invalidateQueries({ queryKey: groupKeys.detail(variables.id) });
    },
  });
}
