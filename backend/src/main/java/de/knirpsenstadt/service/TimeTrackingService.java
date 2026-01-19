package de.knirpsenstadt.service;

import de.knirpsenstadt.api.model.ClockInRequest;
import de.knirpsenstadt.api.model.ClockOutRequest;
import de.knirpsenstadt.api.model.CreateTimeEntryRequest;
import de.knirpsenstadt.api.model.TimeEntry;
import de.knirpsenstadt.api.model.UpdateTimeEntryRequest;
import de.knirpsenstadt.exception.BadRequestException;
import de.knirpsenstadt.exception.ResourceNotFoundException;
import de.knirpsenstadt.model.Employee;
import de.knirpsenstadt.model.TimeEntryType;
import de.knirpsenstadt.repository.EmployeeRepository;
import de.knirpsenstadt.repository.TimeEntryRepository;
import de.knirpsenstadt.security.EmployeePrincipal;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.Duration;
import java.time.LocalDate;
import java.time.OffsetDateTime;
import java.util.List;
import java.util.stream.Collectors;

@Service
@RequiredArgsConstructor
public class TimeTrackingService {

    private final TimeEntryRepository timeEntryRepository;
    private final EmployeeRepository employeeRepository;

    @Transactional
    public TimeEntry clockIn(ClockInRequest request, EmployeePrincipal principal) {
        Employee employee = employeeRepository.findById(principal.getId())
                .orElseThrow(() -> new ResourceNotFoundException("Mitarbeiter", principal.getId()));

        // Check if already clocked in
        List<de.knirpsenstadt.model.TimeEntry> openEntries = timeEntryRepository
                .findOpenEntriesByEmployeeId(employee.getId());

        if (!openEntries.isEmpty()) {
            throw new BadRequestException("Sie sind bereits eingestempelt");
        }

        de.knirpsenstadt.model.TimeEntry entry = de.knirpsenstadt.model.TimeEntry.builder()
                .employee(employee)
                .date(LocalDate.now())
                .clockIn(OffsetDateTime.now())
                .entryType(TimeEntryType.WORK)
                .notes(request != null ? request.getNotes() : null)
                .build();

        de.knirpsenstadt.model.TimeEntry saved = timeEntryRepository.save(entry);
        return toApiTimeEntry(saved);
    }

    @Transactional
    public TimeEntry clockOut(ClockOutRequest request, EmployeePrincipal principal) {
        List<de.knirpsenstadt.model.TimeEntry> openEntries = timeEntryRepository
                .findOpenEntriesByEmployeeId(principal.getId());

        if (openEntries.isEmpty()) {
            throw new BadRequestException("Sie sind nicht eingestempelt");
        }

        de.knirpsenstadt.model.TimeEntry entry = openEntries.get(0);
        entry.setClockOut(OffsetDateTime.now());
        if (request != null) {
            if (request.getBreakMinutes() != null) {
                entry.setBreakMinutes(request.getBreakMinutes());
            }
            if (request.getNotes() != null) {
                entry.setNotes(request.getNotes());
            }
        }

        de.knirpsenstadt.model.TimeEntry saved = timeEntryRepository.save(entry);
        return toApiTimeEntry(saved);
    }

    public TimeEntry getCurrentTimeEntry(EmployeePrincipal principal) {
        List<de.knirpsenstadt.model.TimeEntry> openEntries = timeEntryRepository
                .findOpenEntriesByEmployeeId(principal.getId());

        if (openEntries.isEmpty()) {
            return null;
        }

        return toApiTimeEntry(openEntries.get(0));
    }

    public List<TimeEntry> getTimeEntries(Long employeeId, LocalDate from, LocalDate to) {
        if (!employeeRepository.existsById(employeeId)) {
            throw new ResourceNotFoundException("Mitarbeiter", employeeId);
        }

        List<de.knirpsenstadt.model.TimeEntry> entries =
                timeEntryRepository.findByEmployeeIdAndDateBetween(employeeId, from, to);

        return entries.stream()
                .map(this::toApiTimeEntry)
                .collect(Collectors.toList());
    }

    public List<TimeEntry> getMyTimeEntries(LocalDate from, LocalDate to, EmployeePrincipal principal) {
        List<de.knirpsenstadt.model.TimeEntry> entries =
                timeEntryRepository.findByEmployeeIdAndDateBetween(principal.getId(), from, to);

        return entries.stream()
                .map(this::toApiTimeEntry)
                .collect(Collectors.toList());
    }

    @Transactional
    public TimeEntry createTimeEntry(CreateTimeEntryRequest request) {
        Employee employee = employeeRepository.findById(request.getEmployeeId())
                .orElseThrow(() -> new ResourceNotFoundException("Mitarbeiter", request.getEmployeeId()));

        TimeEntryType entryType = TimeEntryType.WORK;
        if (request.getEntryType() != null) {
            entryType = TimeEntryType.valueOf(request.getEntryType().getValue());
        }

        de.knirpsenstadt.model.TimeEntry entry = de.knirpsenstadt.model.TimeEntry.builder()
                .employee(employee)
                .date(request.getDate())
                .clockIn(request.getClockIn())
                .clockOut(request.getClockOut())
                .breakMinutes(request.getBreakMinutes() != null ? request.getBreakMinutes() : 0)
                .entryType(entryType)
                .notes(request.getNotes())
                .build();

        de.knirpsenstadt.model.TimeEntry saved = timeEntryRepository.save(entry);
        return toApiTimeEntry(saved);
    }

    @Transactional
    public TimeEntry updateTimeEntry(Long id, UpdateTimeEntryRequest request) {
        de.knirpsenstadt.model.TimeEntry entry = timeEntryRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Zeiteintrag", id));

        if (request.getClockIn() != null) {
            entry.setClockIn(request.getClockIn());
        }
        if (request.getClockOut() != null) {
            entry.setClockOut(request.getClockOut());
        }
        if (request.getBreakMinutes() != null) {
            entry.setBreakMinutes(request.getBreakMinutes());
        }
        if (request.getEntryType() != null) {
            entry.setEntryType(TimeEntryType.valueOf(request.getEntryType().getValue()));
        }
        if (request.getNotes() != null) {
            entry.setNotes(request.getNotes());
        }

        de.knirpsenstadt.model.TimeEntry saved = timeEntryRepository.save(entry);
        return toApiTimeEntry(saved);
    }

    @Transactional
    public void deleteTimeEntry(Long id) {
        if (!timeEntryRepository.existsById(id)) {
            throw new ResourceNotFoundException("Zeiteintrag", id);
        }
        timeEntryRepository.deleteById(id);
    }

    private TimeEntry toApiTimeEntry(de.knirpsenstadt.model.TimeEntry entity) {
        TimeEntry dto = new TimeEntry();
        dto.setId(entity.getId());
        dto.setEmployeeId(entity.getEmployee().getId());
        dto.setDate(entity.getDate());
        dto.setClockIn(entity.getClockIn());
        dto.setClockOut(entity.getClockOut());
        dto.setBreakMinutes(entity.getBreakMinutes());
        dto.setCreatedAt(entity.getCreatedAt());
        dto.setNotes(entity.getNotes());
        
        // Set entry type
        if (entity.getEntryType() != null) {
            dto.setEntryType(de.knirpsenstadt.api.model.TimeEntryType.fromValue(entity.getEntryType().name()));
        }

        // Calculate worked minutes
        if (entity.getClockIn() != null && entity.getClockOut() != null) {
            long minutes = Duration.between(entity.getClockIn(), entity.getClockOut()).toMinutes();
            int breakMins = entity.getBreakMinutes() != null ? entity.getBreakMinutes() : 0;
            dto.setWorkedMinutes((int) (minutes - breakMins));
        }

        // Include employee info
        dto.setEmployee(AuthService.toApiEmployee(entity.getEmployee()));

        return dto;
    }
}
