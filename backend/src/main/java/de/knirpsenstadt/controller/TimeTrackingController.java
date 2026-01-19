package de.knirpsenstadt.controller;

import de.knirpsenstadt.api.TimeTrackingApi;
import de.knirpsenstadt.api.model.*;
import de.knirpsenstadt.security.EmployeePrincipal;
import de.knirpsenstadt.service.TimeTrackingService;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.web.bind.annotation.RestController;

import java.time.LocalDate;
import java.util.List;

@RestController
@RequiredArgsConstructor
public class TimeTrackingController implements TimeTrackingApi {

    private final TimeTrackingService timeTrackingService;

    @Override
    public ResponseEntity<TimeEntry> clockIn(ClockInRequest clockInRequest) {
        TimeEntry entry = timeTrackingService.clockIn(clockInRequest, getCurrentPrincipal());
        return ResponseEntity.ok(entry);
    }

    @Override
    public ResponseEntity<TimeEntry> clockOut(ClockOutRequest clockOutRequest) {
        TimeEntry entry = timeTrackingService.clockOut(clockOutRequest, getCurrentPrincipal());
        return ResponseEntity.ok(entry);
    }

    @Override
    public ResponseEntity<TimeEntry> getCurrentTimeEntry() {
        TimeEntry entry = timeTrackingService.getCurrentTimeEntry(getCurrentPrincipal());
        if (entry == null) {
            return ResponseEntity.noContent().build();
        }
        return ResponseEntity.ok(entry);
    }

    @Override
    public ResponseEntity<List<TimeEntry>> getTimeEntries(LocalDate startDate, LocalDate endDate, Long employeeId) {
        List<TimeEntry> entries;
        if (employeeId != null) {
            entries = timeTrackingService.getTimeEntries(employeeId, startDate, endDate);
        } else {
            entries = timeTrackingService.getMyTimeEntries(startDate, endDate, getCurrentPrincipal());
        }
        return ResponseEntity.ok(entries);
    }

    @Override
    public ResponseEntity<TimeEntry> createTimeEntry(CreateTimeEntryRequest createTimeEntryRequest) {
        TimeEntry entry = timeTrackingService.createTimeEntry(createTimeEntryRequest);
        return ResponseEntity.status(201).body(entry);
    }

    @Override
    public ResponseEntity<TimeEntry> updateTimeEntry(Long id, UpdateTimeEntryRequest updateTimeEntryRequest) {
        TimeEntry entry = timeTrackingService.updateTimeEntry(id, updateTimeEntryRequest);
        return ResponseEntity.ok(entry);
    }

    @Override
    public ResponseEntity<Void> deleteTimeEntry(Long id) {
        timeTrackingService.deleteTimeEntry(id);
        return ResponseEntity.noContent().build();
    }

    @Override
    public ResponseEntity<TimeScheduleComparison> getTimeScheduleComparison(LocalDate startDate, LocalDate endDate, Long employeeId) {
        // TODO: Implement comparison logic
        return ResponseEntity.ok(new TimeScheduleComparison());
    }

    private EmployeePrincipal getCurrentPrincipal() {
        return (EmployeePrincipal) SecurityContextHolder.getContext().getAuthentication().getPrincipal();
    }
}
