import { computed, type Ref } from 'vue';
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query';
import { 
  apiClient, 
  type TimeEntry, 
  type CreateTimeEntryRequest, 
  type UpdateTimeEntryRequest,
  type ClockInRequest,
  type ClockOutRequest,
  type TimeScheduleComparison 
} from '../api';
import { toISODateString } from '../utils';

export const timeTrackingKeys = {
  all: ['timeTracking'] as const,
  current: () => [...timeTrackingKeys.all, 'current'] as const,
  entries: () => [...timeTrackingKeys.all, 'entries'] as const,
  entriesList: (params: { startDate: string; endDate: string; employeeId?: number }) => 
    [...timeTrackingKeys.entries(), params] as const,
  comparison: () => [...timeTrackingKeys.all, 'comparison'] as const,
  comparisonRange: (params: { startDate: string; endDate: string; employeeId?: number }) => 
    [...timeTrackingKeys.comparison(), params] as const,
};

export function useCurrentTimeEntry() {
  return useQuery({
    queryKey: timeTrackingKeys.current(),
    queryFn: async () => {
      const { data, error, response } = await apiClient.GET('/time-tracking/current');
      // 204 means not clocked in
      if (response.status === 204) return null;
      if (error) throw new Error((error as any)?.message || 'Fehler beim Laden des aktuellen Status');
      return data as TimeEntry;
    },
    refetchInterval: 60000, // Refetch every minute
  });
}

export function useTimeEntries(params: {
  startDate: Ref<Date> | Date;
  endDate: Ref<Date> | Date;
  employeeId?: Ref<number | undefined> | number;
}) {
  const queryParams = computed(() => ({
    startDate: toISODateString(params.startDate instanceof Date ? params.startDate : params.startDate.value),
    endDate: toISODateString(params.endDate instanceof Date ? params.endDate : params.endDate.value),
    employeeId: typeof params.employeeId === 'number' 
      ? params.employeeId 
      : params.employeeId?.value,
  }));

  return useQuery({
    queryKey: computed(() => timeTrackingKeys.entriesList(queryParams.value)),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/time-tracking/entries', {
        params: { 
          query: {
            startDate: queryParams.value.startDate,
            endDate: queryParams.value.endDate,
            employeeId: queryParams.value.employeeId,
          } 
        },
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Laden der Zeiteinträge');
      return data as TimeEntry[];
    },
  });
}

export function useTimeScheduleComparison(params: {
  startDate: Ref<Date> | Date;
  endDate: Ref<Date> | Date;
  employeeId?: Ref<number | undefined> | number;
}) {
  const queryParams = computed(() => ({
    startDate: toISODateString(params.startDate instanceof Date ? params.startDate : params.startDate.value),
    endDate: toISODateString(params.endDate instanceof Date ? params.endDate : params.endDate.value),
    employeeId: typeof params.employeeId === 'number' 
      ? params.employeeId 
      : params.employeeId?.value,
  }));

  return useQuery({
    queryKey: computed(() => timeTrackingKeys.comparisonRange(queryParams.value)),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/time-tracking/comparison', {
        params: { 
          query: {
            startDate: queryParams.value.startDate,
            endDate: queryParams.value.endDate,
            employeeId: queryParams.value.employeeId,
          } 
        },
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Laden des Vergleichs');
      return data as TimeScheduleComparison;
    },
  });
}

export function useClockIn() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request?: ClockInRequest) => {
      const { data, error } = await apiClient.POST('/time-tracking/clock-in', {
        body: request,
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Einstempeln');
      return data as TimeEntry;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: timeTrackingKeys.current() });
      queryClient.invalidateQueries({ queryKey: timeTrackingKeys.entries() });
    },
  });
}

export function useClockOut() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request?: ClockOutRequest) => {
      const { data, error } = await apiClient.POST('/time-tracking/clock-out', {
        body: request,
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Ausstempeln');
      return data as TimeEntry;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: timeTrackingKeys.current() });
      queryClient.invalidateQueries({ queryKey: timeTrackingKeys.entries() });
    },
  });
}

export function useCreateTimeEntry() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (entry: CreateTimeEntryRequest) => {
      const { data, error } = await apiClient.POST('/time-tracking/entries', {
        body: entry,
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Anlegen des Eintrags');
      return data as TimeEntry;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: timeTrackingKeys.entries() });
    },
  });
}

export function useUpdateTimeEntry() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id, data: entryData }: { id: number; data: UpdateTimeEntryRequest }) => {
      const { data, error } = await apiClient.PUT('/time-tracking/entries/{id}', {
        params: { path: { id } },
        body: entryData,
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Aktualisieren des Eintrags');
      return data as TimeEntry;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: timeTrackingKeys.entries() });
      queryClient.invalidateQueries({ queryKey: timeTrackingKeys.current() });
    },
  });
}

export function useDeleteTimeEntry() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: number) => {
      const { error } = await apiClient.DELETE('/time-tracking/entries/{id}', {
        params: { path: { id } },
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Löschen des Eintrags');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: timeTrackingKeys.entries() });
    },
  });
}
