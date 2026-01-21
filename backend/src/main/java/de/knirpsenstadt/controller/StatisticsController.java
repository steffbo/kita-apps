package de.knirpsenstadt.controller;

import de.knirpsenstadt.api.StatisticsApi;
import de.knirpsenstadt.api.model.*;
import de.knirpsenstadt.service.StatisticsService;
import lombok.RequiredArgsConstructor;
import org.springframework.core.io.Resource;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RestController;

import java.time.LocalDate;

@RestController
@RequiredArgsConstructor
public class StatisticsController implements StatisticsApi {

    private final StatisticsService statisticsService;

    @Override
    public ResponseEntity<OverviewStatistics> getOverviewStatistics(LocalDate month) {
        OverviewStatistics stats = statisticsService.getOverviewStatistics(month);
        return ResponseEntity.ok(stats);
    }

    @Override
    public ResponseEntity<EmployeeStatistics> getEmployeeStatistics(Long id, LocalDate month) {
        EmployeeStatistics stats = statisticsService.getEmployeeStatistics(id, month);
        return ResponseEntity.ok(stats);
    }

    @Override
    public ResponseEntity<WeeklyStatistics> getWeeklyStatistics(LocalDate weekStart) {
        WeeklyStatistics stats = statisticsService.getWeeklyStatistics(weekStart);
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
