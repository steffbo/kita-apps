package de.knirpsenstadt.controller;

import de.knirpsenstadt.api.StatisticsApi;
import de.knirpsenstadt.api.model.*;
import lombok.RequiredArgsConstructor;
import org.springframework.core.io.Resource;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RestController;

import java.time.LocalDate;
import java.util.ArrayList;

@RestController
@RequiredArgsConstructor
public class StatisticsController implements StatisticsApi {

    @Override
    public ResponseEntity<OverviewStatistics> getOverviewStatistics(LocalDate month) {
        // TODO: Implement statistics calculation
        OverviewStatistics stats = new OverviewStatistics();
        stats.setMonth(month);
        stats.setTotalEmployees(0);
        stats.setTotalScheduledHours(0f);
        stats.setTotalWorkedHours(0f);
        stats.setTotalOvertimeHours(0f);
        stats.setSickDays(0);
        stats.setVacationDays(0);
        stats.setEmployeeStats(new ArrayList<>());
        return ResponseEntity.ok(stats);
    }

    @Override
    public ResponseEntity<EmployeeStatistics> getEmployeeStatistics(Long id, LocalDate month) {
        // TODO: Implement employee statistics
        EmployeeStatistics stats = new EmployeeStatistics();
        stats.setMonth(month);
        stats.setContractedHours(0f);
        stats.setScheduledHours(0f);
        stats.setWorkedHours(0f);
        stats.setOvertimeHours(0f);
        stats.setOvertimeBalance(0f);
        stats.setVacationDaysUsed(0);
        stats.setVacationDaysRemaining(0f);
        stats.setSickDays(0);
        stats.setDailyBreakdown(new ArrayList<>());
        return ResponseEntity.ok(stats);
    }

    @Override
    public ResponseEntity<WeeklyStatistics> getWeeklyStatistics(LocalDate weekStart) {
        // TODO: Implement weekly statistics
        WeeklyStatistics stats = new WeeklyStatistics();
        stats.setWeekStart(weekStart);
        stats.setWeekEnd(weekStart.plusDays(6));
        stats.setByEmployee(new ArrayList<>());
        stats.setByGroup(new ArrayList<>());
        stats.setTotalScheduledHours(0f);
        stats.setTotalWorkedHours(0f);
        return ResponseEntity.ok(stats);
    }

    @Override
    public ResponseEntity<Resource> exportTimesheet(LocalDate month, String format, Long employeeId) {
        // TODO: Implement export
        return ResponseEntity.noContent().build();
    }

    @Override
    public ResponseEntity<Resource> exportSchedule(LocalDate weekStart, String format) {
        // TODO: Implement export
        return ResponseEntity.noContent().build();
    }
}
