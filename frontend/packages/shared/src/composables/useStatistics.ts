import { computed, type Ref } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import { 
  apiClient, 
  type OverviewStatistics, 
  type EmployeeStatistics,
  type WeeklyStatistics 
} from '../api';
import { toISODateString } from '../utils';

export const statisticsKeys = {
  all: ['statistics'] as const,
  overview: (month: string) => [...statisticsKeys.all, 'overview', month] as const,
  employee: (id: number, month: string) => [...statisticsKeys.all, 'employee', id, month] as const,
  weekly: (weekStart: string) => [...statisticsKeys.all, 'weekly', weekStart] as const,
};

export function useOverviewStatistics(month: Ref<Date> | Date) {
  const monthStr = computed(() => {
    const d = month instanceof Date ? month : month.value;
    // First day of month
    return toISODateString(new Date(d.getFullYear(), d.getMonth(), 1));
  });

  return useQuery({
    queryKey: computed(() => statisticsKeys.overview(monthStr.value)),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/statistics/overview', {
        params: { query: { month: monthStr.value } },
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Laden der Übersicht');
      return data as OverviewStatistics;
    },
  });
}

export function useEmployeeStatistics(params: {
  id: Ref<number> | number;
  month: Ref<Date> | Date;
}) {
  const employeeId = computed(() => (typeof params.id === 'number' ? params.id : params.id.value));
  const monthStr = computed(() => {
    const d = params.month instanceof Date ? params.month : params.month.value;
    return toISODateString(new Date(d.getFullYear(), d.getMonth(), 1));
  });

  return useQuery({
    queryKey: computed(() => statisticsKeys.employee(employeeId.value, monthStr.value)),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/statistics/employee/{id}', {
        params: { 
          path: { id: employeeId.value },
          query: { month: monthStr.value } 
        },
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Laden der Mitarbeiter-Statistik');
      return data as EmployeeStatistics;
    },
    enabled: computed(() => employeeId.value > 0),
  });
}

export function useWeeklyStatistics(weekStart: Ref<Date> | Date) {
  const weekStartStr = computed(() => 
    toISODateString(weekStart instanceof Date ? weekStart : weekStart.value)
  );

  return useQuery({
    queryKey: computed(() => statisticsKeys.weekly(weekStartStr.value)),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/statistics/weekly', {
        params: { query: { weekStart: weekStartStr.value } },
      });
      if (error) throw new Error((error as any)?.message || 'Fehler beim Laden der Wochen-Statistik');
      return data as WeeklyStatistics;
    },
  });
}
