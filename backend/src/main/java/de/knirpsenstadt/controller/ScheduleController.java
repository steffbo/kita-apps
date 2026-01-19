package de.knirpsenstadt.controller;

import de.knirpsenstadt.api.ScheduleApi;
import de.knirpsenstadt.api.model.*;
import de.knirpsenstadt.service.ScheduleService;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RestController;

import java.time.LocalDate;
import java.util.List;

@RestController
@RequiredArgsConstructor
public class ScheduleController implements ScheduleApi {

    private final ScheduleService scheduleService;

    @Override
    public ResponseEntity<List<ScheduleEntry>> getSchedule(LocalDate startDate, LocalDate endDate, Long employeeId, Long groupId) {
        List<ScheduleEntry> entries = scheduleService.getSchedule(startDate, endDate, employeeId, groupId);
        return ResponseEntity.ok(entries);
    }

    @Override
    public ResponseEntity<WeekSchedule> getWeekSchedule(LocalDate weekStart) {
        WeekSchedule schedule = scheduleService.getWeekSchedule(weekStart, null);
        return ResponseEntity.ok(schedule);
    }

    @Override
    public ResponseEntity<ScheduleEntry> createScheduleEntry(CreateScheduleEntryRequest createScheduleEntryRequest) {
        ScheduleEntry entry = scheduleService.createScheduleEntry(createScheduleEntryRequest);
        return ResponseEntity.status(201).body(entry);
    }

    @Override
    public ResponseEntity<ScheduleEntry> updateScheduleEntry(Long id, UpdateScheduleEntryRequest updateScheduleEntryRequest) {
        ScheduleEntry entry = scheduleService.updateScheduleEntry(id, updateScheduleEntryRequest);
        return ResponseEntity.ok(entry);
    }

    @Override
    public ResponseEntity<Void> deleteScheduleEntry(Long id) {
        scheduleService.deleteScheduleEntry(id);
        return ResponseEntity.noContent().build();
    }

    @Override
    public ResponseEntity<List<ScheduleEntry>> bulkCreateScheduleEntries(List<CreateScheduleEntryRequest> createScheduleEntryRequest) {
        List<ScheduleEntry> entries = scheduleService.bulkCreateScheduleEntries(createScheduleEntryRequest);
        return ResponseEntity.status(201).body(entries);
    }
}
