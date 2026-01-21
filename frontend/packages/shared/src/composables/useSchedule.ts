import { computed, type Ref } from 'vue';
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query';
import { 
  apiClient, 
  type ScheduleEntry, 
  type CreateScheduleEntryRequest, 
  type UpdateScheduleEntryRequest,
  type WeekSchedule 
} from '../api';
import { toISODateString } from '../utils';

export const scheduleKeys = {
  all: ['schedule'] as const,
  lists: () => [...scheduleKeys.all, 'list'] as const,
  list: (params: { startDate: string; endDate: string; employeeId?: number; groupId?: number }) => 
    [...scheduleKeys.lists(), params] as const,
  week: (weekStart: string) => [...scheduleKeys.all, 'week', weekStart] as const,
  details: () => [...scheduleKeys.all, 'detail'] as const,
  detail: (id: number) => [...scheduleKeys.details(), id] as const,
};

export function useSchedule(params: {
  startDate: Ref<Date> | Date;
  endDate: Ref<Date> | Date;
  employeeId?: Ref<number | undefined> | number;
  groupId?: Ref<number | undefined> | number;
}) {
  const queryParams = computed(() => ({
    startDate: toISODateString(params.startDate instanceof Date ? params.startDate : params.startDate.value),
    endDate: toISODateString(params.endDate instanceof Date ? params.endDate : params.endDate.value),
    employeeId: typeof params.employeeId === 'number' 
      ? params.employeeId 
      : params.employeeId?.value,
    groupId: typeof params.groupId === 'number' 
      ? params.groupId 
      : params.groupId?.value,
  }));

  return useQuery({
    queryKey: computed(() => scheduleKeys.list(queryParams.value)),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/schedule', {
        params: { 
          query: {
            startDate: queryParams.value.startDate,
            endDate: queryParams.value.endDate,
            employeeId: queryParams.value.employeeId,
            groupId: queryParams.value.groupId,
          } 
        },
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Laden des Dienstplans');
      return data as ScheduleEntry[];
    },
  });
}

export function useWeekSchedule(weekStart: Ref<Date> | Date) {
  const weekStartStr = computed(() => 
    toISODateString(weekStart instanceof Date ? weekStart : weekStart.value)
  );

  return useQuery({
    queryKey: computed(() => scheduleKeys.week(weekStartStr.value)),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/schedule/week', {
        params: { query: { weekStart: weekStartStr.value } },
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Laden der Wochenansicht');
      return data as WeekSchedule;
    },
  });
}

export function useCreateScheduleEntry() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (entry: CreateScheduleEntryRequest) => {
      const { data, error } = await apiClient.POST('/schedule', {
        body: entry,
      });
      if (error) {
        const errorMessage = (error as any)?.message || (error as any)?.detail || 'Fehler beim Anlegen des Eintrags';
        throw new Error(errorMessage);
      }
      return data as ScheduleEntry;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: scheduleKeys.all });
    },
  });
}

export function useBulkCreateScheduleEntries() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (entries: CreateScheduleEntryRequest[]) => {
      const { data, error } = await apiClient.POST('/schedule/bulk', {
        body: entries,
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Anlegen der Einträge');
      return data as ScheduleEntry[];
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: scheduleKeys.all });
    },
  });
}

export function useUpdateScheduleEntry() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id, data: entryData }: { id: number; data: UpdateScheduleEntryRequest }) => {
      const { data, error } = await apiClient.PUT('/schedule/{id}', {
        params: { path: { id } },
        body: entryData,
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Aktualisieren des Eintrags');
      return data as ScheduleEntry;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: scheduleKeys.all });
    },
  });
}

export function useDeleteScheduleEntry() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: number) => {
      const { error } = await apiClient.DELETE('/schedule/{id}', {
        params: { path: { id } },
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Löschen des Eintrags');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: scheduleKeys.all });
    },
  });
}
