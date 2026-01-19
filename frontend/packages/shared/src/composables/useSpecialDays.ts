import { computed, type Ref } from 'vue';
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query';
import { 
  apiClient, 
  type SpecialDay, 
  type CreateSpecialDayRequest 
} from '../api';

export const specialDayKeys = {
  all: ['specialDays'] as const,
  lists: () => [...specialDayKeys.all, 'list'] as const,
  list: (params: { year: number; includeHolidays?: boolean }) => 
    [...specialDayKeys.lists(), params] as const,
  holidays: (year: number) => [...specialDayKeys.all, 'holidays', year] as const,
  details: () => [...specialDayKeys.all, 'detail'] as const,
  detail: (id: number) => [...specialDayKeys.details(), id] as const,
};

export function useSpecialDays(params: {
  year: Ref<number> | number;
  includeHolidays?: Ref<boolean> | boolean;
}) {
  const queryParams = computed(() => ({
    year: typeof params.year === 'number' ? params.year : params.year.value,
    includeHolidays: typeof params.includeHolidays === 'boolean' 
      ? params.includeHolidays 
      : params.includeHolidays?.value ?? true,
  }));

  return useQuery({
    queryKey: computed(() => specialDayKeys.list(queryParams.value)),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/special-days', {
        params: { 
          query: {
            year: queryParams.value.year,
            includeHolidays: queryParams.value.includeHolidays,
          } 
        },
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Laden der besonderen Tage');
      return data as SpecialDay[];
    },
  });
}

export function useHolidays(year: Ref<number> | number) {
  const yearValue = computed(() => (typeof year === 'number' ? year : year.value));

  return useQuery({
    queryKey: computed(() => specialDayKeys.holidays(yearValue.value)),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/special-days/holidays/{year}', {
        params: { path: { year: yearValue.value } },
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Laden der Feiertage');
      return data as SpecialDay[];
    },
  });
}

export function useCreateSpecialDay() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (specialDay: CreateSpecialDayRequest) => {
      const { data, error } = await apiClient.POST('/special-days', {
        body: specialDay,
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Anlegen des besonderen Tags');
      return data as SpecialDay;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: specialDayKeys.lists() });
    },
  });
}

export function useUpdateSpecialDay() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id, data: specialDayData }: { id: number; data: CreateSpecialDayRequest }) => {
      const { data, error } = await apiClient.PUT('/special-days/{id}', {
        params: { path: { id } },
        body: specialDayData,
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Aktualisieren des besonderen Tags');
      return data as SpecialDay;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: specialDayKeys.lists() });
    },
  });
}

export function useDeleteSpecialDay() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: number) => {
      const { error } = await apiClient.DELETE('/special-days/{id}', {
        params: { path: { id } },
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Löschen des besonderen Tags');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: specialDayKeys.lists() });
    },
  });
}
