package de.knirpsenstadt.service;

import de.knirpsenstadt.api.model.*;
import de.knirpsenstadt.model.Employee;
import de.knirpsenstadt.model.ScheduleEntry;
import de.knirpsenstadt.model.ScheduleEntryType;
import de.knirpsenstadt.model.TimeEntry;
import de.knirpsenstadt.repository.EmployeeRepository;
import de.knirpsenstadt.repository.GroupRepository;
import de.knirpsenstadt.repository.ScheduleEntryRepository;
import de.knirpsenstadt.repository.TimeEntryRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

import java.math.BigDecimal;
import java.time.DayOfWeek;
import java.time.Duration;
import java.time.LocalDate;
import java.time.LocalTime;
import java.time.temporal.TemporalAdjusters;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

@Service
@RequiredArgsConstructor
public class StatisticsService {

    private final EmployeeRepository employeeRepository;
    private final ScheduleEntryRepository scheduleEntryRepository;
    private final TimeEntryRepository timeEntryRepository;
    private final GroupRepository groupRepository;

    /**
     * Get overview statistics for a month
     */
    public OverviewStatistics getOverviewStatistics(LocalDate month) {
        LocalDate startOfMonth = month.withDayOfMonth(1);
        LocalDate endOfMonth = month.withDayOfMonth(month.lengthOfMonth());

        List<Employee> activeEmployees = employeeRepository.findAllActiveOrderByName();
        List<ScheduleEntry> scheduleEntries = scheduleEntryRepository.findByDateBetween(startOfMonth, endOfMonth);
        List<TimeEntry> timeEntries = timeEntryRepository.findByDateBetween(startOfMonth, endOfMonth);

        // Group by employee
        Map<Long, List<ScheduleEntry>> scheduleByEmployee = scheduleEntries.stream()
                .collect(Collectors.groupingBy(e -> e.getEmployee().getId()));
        Map<Long, List<TimeEntry>> timeByEmployee = timeEntries.stream()
                .collect(Collectors.groupingBy(e -> e.getEmployee().getId()));

        float totalScheduledHours = 0;
        float totalWorkedHours = 0;
        float totalOvertimeHours = 0;
        int sickDays = 0;
        int vacationDays = 0;

        List<EmployeeStatisticsSummary> employeeStats = new ArrayList<>();

        for (Employee employee : activeEmployees) {
            List<ScheduleEntry> empSchedule = scheduleByEmployee.getOrDefault(employee.getId(), List.of());
            List<TimeEntry> empTime = timeByEmployee.getOrDefault(employee.getId(), List.of());

            float scheduledHours = calculateScheduledHours(empSchedule);
            float workedHours = calculateWorkedHours(empTime);
            
            // Calculate expected hours for the month based on contract
            float contractedWeeklyHours = employee.getWeeklyHours() != null 
                    ? employee.getWeeklyHours().floatValue() : 0;
            int workingDaysInMonth = countWorkingDays(startOfMonth, endOfMonth);
            float expectedMonthlyHours = (contractedWeeklyHours / 5) * workingDaysInMonth;
            
            float overtimeHours = workedHours - expectedMonthlyHours;

            totalScheduledHours += scheduledHours;
            totalWorkedHours += workedHours;
            totalOvertimeHours += overtimeHours;

            // Count sick and vacation days
            for (ScheduleEntry entry : empSchedule) {
                if (entry.getEntryType() == ScheduleEntryType.SICK) {
                    sickDays++;
                } else if (entry.getEntryType() == ScheduleEntryType.VACATION) {
                    vacationDays++;
                }
            }

            EmployeeStatisticsSummary summary = new EmployeeStatisticsSummary();
            summary.setEmployee(AuthService.toApiEmployee(employee));
            summary.setScheduledHours(scheduledHours);
            summary.setWorkedHours(workedHours);
            summary.setOvertimeHours(overtimeHours);
            summary.setRemainingVacationDays(employee.getRemainingVacationDays() != null 
                    ? employee.getRemainingVacationDays().floatValue() : 0);
            employeeStats.add(summary);
        }

        OverviewStatistics stats = new OverviewStatistics();
        stats.setMonth(month);
        stats.setTotalEmployees(activeEmployees.size());
        stats.setEmployeeStats(employeeStats);
        stats.setTotalScheduledHours(totalScheduledHours);
        stats.setTotalWorkedHours(totalWorkedHours);
        stats.setTotalOvertimeHours(totalOvertimeHours);
        stats.setSickDays(sickDays);
        stats.setVacationDays(vacationDays);

        return stats;
    }

    /**
     * Get weekly statistics
     */
    public WeeklyStatistics getWeeklyStatistics(LocalDate weekStart) {
        LocalDate monday = weekStart.with(TemporalAdjusters.previousOrSame(DayOfWeek.MONDAY));
        LocalDate sunday = monday.plusDays(6);

        List<Employee> activeEmployees = employeeRepository.findAllActiveOrderByName();
        List<ScheduleEntry> scheduleEntries = scheduleEntryRepository.findByDateBetween(monday, sunday);
        List<TimeEntry> timeEntries = timeEntryRepository.findByDateBetween(monday, sunday);

        // Group by employee
        Map<Long, List<ScheduleEntry>> scheduleByEmployee = scheduleEntries.stream()
                .collect(Collectors.groupingBy(e -> e.getEmployee().getId()));
        Map<Long, List<TimeEntry>> timeByEmployee = timeEntries.stream()
                .collect(Collectors.groupingBy(e -> e.getEmployee().getId()));

        // Group by group
        Map<Long, List<ScheduleEntry>> scheduleByGroup = scheduleEntries.stream()
                .filter(e -> e.getGroup() != null)
                .collect(Collectors.groupingBy(e -> e.getGroup().getId()));

        float totalScheduledHours = 0;
        float totalWorkedHours = 0;

        List<EmployeeWeekSummary> byEmployee = new ArrayList<>();

        for (Employee employee : activeEmployees) {
            List<ScheduleEntry> empSchedule = scheduleByEmployee.getOrDefault(employee.getId(), List.of());
            List<TimeEntry> empTime = timeByEmployee.getOrDefault(employee.getId(), List.of());

            float scheduledHours = calculateScheduledHours(empSchedule);
            float workedHours = calculateWorkedHours(empTime);
            int daysWorked = (int) empTime.stream()
                    .filter(t -> t.getClockOut() != null)
                    .map(TimeEntry::getDate)
                    .distinct()
                    .count();

            totalScheduledHours += scheduledHours;
            totalWorkedHours += workedHours;

            EmployeeWeekSummary summary = new EmployeeWeekSummary();
            summary.setEmployee(toApiEmployeeWithWeeklyHours(employee));
            summary.setScheduledHours(scheduledHours);
            summary.setWorkedHours(workedHours);
            summary.setDaysWorked(daysWorked);
            byEmployee.add(summary);
        }

        // Calculate group summaries
        List<GroupWeekSummary> byGroup = new ArrayList<>();
        List<de.knirpsenstadt.model.Group> groups = groupRepository.findAllOrderByName();
        
        for (de.knirpsenstadt.model.Group group : groups) {
            List<ScheduleEntry> groupSchedule = scheduleByGroup.getOrDefault(group.getId(), List.of());
            float groupScheduledHours = calculateScheduledHours(groupSchedule);
            
            // Count staffed days (days with at least one schedule entry)
            int staffedDays = (int) groupSchedule.stream()
                    .map(ScheduleEntry::getDate)
                    .distinct()
                    .count();

            GroupWeekSummary groupSummary = new GroupWeekSummary();
            de.knirpsenstadt.api.model.Group groupDto = new de.knirpsenstadt.api.model.Group();
            groupDto.setId(group.getId());
            groupDto.setName(group.getName());
            groupDto.setColor(group.getColor());
            groupSummary.setGroup(groupDto);
            groupSummary.setTotalScheduledHours(groupScheduledHours);
            groupSummary.setStaffedDays(staffedDays);
            groupSummary.setUnderstaffedDays(5 - staffedDays); // Monday-Friday = 5 days
            byGroup.add(groupSummary);
        }

        WeeklyStatistics stats = new WeeklyStatistics();
        stats.setWeekStart(monday);
        stats.setWeekEnd(sunday);
        stats.setByEmployee(byEmployee);
        stats.setByGroup(byGroup);
        stats.setTotalScheduledHours(totalScheduledHours);
        stats.setTotalWorkedHours(totalWorkedHours);

        return stats;
    }

    /**
     * Get detailed statistics for a single employee
     */
    public EmployeeStatistics getEmployeeStatistics(Long employeeId, LocalDate month) {
        Employee employee = employeeRepository.findById(employeeId)
                .orElseThrow(() -> new RuntimeException("Employee not found"));

        LocalDate startOfMonth = month.withDayOfMonth(1);
        LocalDate endOfMonth = month.withDayOfMonth(month.lengthOfMonth());

        List<ScheduleEntry> scheduleEntries = scheduleEntryRepository
                .findByEmployeeIdAndDateBetween(employeeId, startOfMonth, endOfMonth);
        List<TimeEntry> timeEntries = timeEntryRepository
                .findByEmployeeIdAndDateBetween(employeeId, startOfMonth, endOfMonth);

        float contractedWeeklyHours = employee.getWeeklyHours() != null 
                ? employee.getWeeklyHours().floatValue() : 0;
        int workingDaysInMonth = countWorkingDays(startOfMonth, endOfMonth);
        float expectedMonthlyHours = (contractedWeeklyHours / 5) * workingDaysInMonth;

        float scheduledHours = calculateScheduledHours(scheduleEntries);
        float workedHours = calculateWorkedHours(timeEntries);
        float overtimeHours = workedHours - expectedMonthlyHours;

        int vacationDaysUsed = (int) scheduleEntries.stream()
                .filter(e -> e.getEntryType() == ScheduleEntryType.VACATION)
                .count();
        int sickDays = (int) scheduleEntries.stream()
                .filter(e -> e.getEntryType() == ScheduleEntryType.SICK)
                .count();

        // Build daily breakdown
        Map<LocalDate, List<ScheduleEntry>> scheduleByDate = scheduleEntries.stream()
                .collect(Collectors.groupingBy(ScheduleEntry::getDate));
        Map<LocalDate, List<TimeEntry>> timeByDate = timeEntries.stream()
                .collect(Collectors.groupingBy(TimeEntry::getDate));

        List<DayStatistics> dailyBreakdown = new ArrayList<>();
        LocalDate current = startOfMonth;
        while (!current.isAfter(endOfMonth)) {
            if (current.getDayOfWeek() != DayOfWeek.SATURDAY && current.getDayOfWeek() != DayOfWeek.SUNDAY) {
                List<ScheduleEntry> daySchedule = scheduleByDate.getOrDefault(current, List.of());
                List<TimeEntry> dayTime = timeByDate.getOrDefault(current, List.of());

                DayStatistics dayStats = new DayStatistics();
                dayStats.setDate(current);
                dayStats.setScheduledHours(calculateScheduledHours(daySchedule));
                dayStats.setWorkedHours(calculateWorkedHours(dayTime));
                
                if (!daySchedule.isEmpty()) {
                    dayStats.setEntryType(de.knirpsenstadt.api.model.ScheduleEntryType
                            .fromValue(daySchedule.get(0).getEntryType().name()));
                }
                
                dailyBreakdown.add(dayStats);
            }
            current = current.plusDays(1);
        }

        EmployeeStatistics stats = new EmployeeStatistics();
        stats.setEmployee(AuthService.toApiEmployee(employee));
        stats.setMonth(month);
        stats.setContractedHours(expectedMonthlyHours);
        stats.setScheduledHours(scheduledHours);
        stats.setWorkedHours(workedHours);
        stats.setOvertimeHours(overtimeHours);
        stats.setOvertimeBalance(employee.getOvertimeBalance() != null 
                ? employee.getOvertimeBalance().floatValue() : 0);
        stats.setVacationDaysUsed(vacationDaysUsed);
        stats.setVacationDaysRemaining(employee.getRemainingVacationDays() != null 
                ? employee.getRemainingVacationDays().floatValue() : 0);
        stats.setSickDays(sickDays);
        stats.setDailyBreakdown(dailyBreakdown);

        return stats;
    }

    /**
     * Calculate total scheduled hours from a list of schedule entries
     */
    private float calculateScheduledHours(List<ScheduleEntry> entries) {
        float totalMinutes = 0;
        for (ScheduleEntry entry : entries) {
            if (entry.getStartTime() != null && entry.getEndTime() != null 
                    && entry.getEntryType() == ScheduleEntryType.WORK) {
                long minutes = Duration.between(entry.getStartTime(), entry.getEndTime()).toMinutes();
                int breakMinutes = entry.getBreakMinutes() != null ? entry.getBreakMinutes() : 0;
                totalMinutes += (minutes - breakMinutes);
            }
        }
        return totalMinutes / 60f;
    }

    /**
     * Calculate total worked hours from a list of time entries
     */
    private float calculateWorkedHours(List<TimeEntry> entries) {
        float totalMinutes = 0;
        for (TimeEntry entry : entries) {
            if (entry.getClockIn() != null && entry.getClockOut() != null) {
                long minutes = Duration.between(entry.getClockIn(), entry.getClockOut()).toMinutes();
                int breakMinutes = entry.getBreakMinutes() != null ? entry.getBreakMinutes() : 0;
                totalMinutes += (minutes - breakMinutes);
            }
        }
        return totalMinutes / 60f;
    }

    /**
     * Count working days (Monday-Friday) between two dates
     */
    private int countWorkingDays(LocalDate start, LocalDate end) {
        int count = 0;
        LocalDate current = start;
        while (!current.isAfter(end)) {
            DayOfWeek day = current.getDayOfWeek();
            if (day != DayOfWeek.SATURDAY && day != DayOfWeek.SUNDAY) {
                count++;
            }
            current = current.plusDays(1);
        }
        return count;
    }

    /**
     * Convert employee to API model including weeklyHours
     */
    private de.knirpsenstadt.api.model.Employee toApiEmployeeWithWeeklyHours(Employee entity) {
        de.knirpsenstadt.api.model.Employee dto = AuthService.toApiEmployee(entity);
        // weeklyHours is already set by AuthService.toApiEmployee
        return dto;
    }
}
