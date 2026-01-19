package de.knirpsenstadt.service;

import de.knirpsenstadt.api.model.*;
import de.knirpsenstadt.exception.ResourceNotFoundException;
import de.knirpsenstadt.model.Employee;
import de.knirpsenstadt.model.Group;
import de.knirpsenstadt.model.ScheduleEntryType;
import de.knirpsenstadt.repository.EmployeeRepository;
import de.knirpsenstadt.repository.GroupRepository;
import de.knirpsenstadt.repository.ScheduleEntryRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.DayOfWeek;
import java.time.LocalDate;
import java.time.LocalTime;
import java.time.format.DateTimeFormatter;
import java.time.temporal.TemporalAdjusters;
import java.util.*;
import java.util.stream.Collectors;

@Service
@RequiredArgsConstructor
public class ScheduleService {

    private static final DateTimeFormatter TIME_FORMATTER = DateTimeFormatter.ofPattern("HH:mm:ss");

    private final ScheduleEntryRepository scheduleEntryRepository;
    private final EmployeeRepository employeeRepository;
    private final GroupRepository groupRepository;

    public WeekSchedule getWeekSchedule(LocalDate weekStart, Long groupId) {
        // Ensure weekStart is a Monday
        LocalDate monday = weekStart.with(TemporalAdjusters.previousOrSame(DayOfWeek.MONDAY));
        LocalDate sunday = monday.plusDays(6);

        List<de.knirpsenstadt.model.ScheduleEntry> entries;
        if (groupId != null) {
            entries = scheduleEntryRepository.findByDateBetweenAndGroupId(monday, sunday, groupId);
        } else {
            entries = scheduleEntryRepository.findByDateBetween(monday, sunday);
        }

        WeekSchedule weekSchedule = new WeekSchedule();
        weekSchedule.setWeekStart(monday);
        weekSchedule.setWeekEnd(sunday);

        List<DaySchedule> days = new ArrayList<>();
        for (int i = 0; i < 5; i++) { // Monday to Friday
            LocalDate date = monday.plusDays(i);
            DaySchedule day = new DaySchedule();
            day.setDate(date);

            List<ScheduleEntry> dayEntries = entries.stream()
                    .filter(e -> e.getDate().equals(date))
                    .map(this::toApiScheduleEntry)
                    .collect(Collectors.toList());
            day.setEntries(dayEntries);

            days.add(day);
        }
        weekSchedule.setDays(days);

        return weekSchedule;
    }

    public List<ScheduleEntry> getEmployeeSchedule(Long employeeId, LocalDate from, LocalDate to) {
        if (!employeeRepository.existsById(employeeId)) {
            throw new ResourceNotFoundException("Mitarbeiter", employeeId);
        }

        List<de.knirpsenstadt.model.ScheduleEntry> entries =
                scheduleEntryRepository.findByEmployeeIdAndDateBetween(employeeId, from, to);

        return entries.stream()
                .map(this::toApiScheduleEntry)
                .collect(Collectors.toList());
    }

    public List<ScheduleEntry> getSchedule(LocalDate startDate, LocalDate endDate, Long employeeId, Long groupId) {
        List<de.knirpsenstadt.model.ScheduleEntry> entries;

        if (employeeId != null && groupId != null) {
            entries = scheduleEntryRepository.findByEmployeeIdAndDateBetween(employeeId, startDate, endDate)
                    .stream()
                    .filter(e -> e.getGroup() != null && e.getGroup().getId().equals(groupId))
                    .collect(Collectors.toList());
        } else if (employeeId != null) {
            entries = scheduleEntryRepository.findByEmployeeIdAndDateBetween(employeeId, startDate, endDate);
        } else if (groupId != null) {
            entries = scheduleEntryRepository.findByDateBetweenAndGroupId(startDate, endDate, groupId);
        } else {
            entries = scheduleEntryRepository.findByDateBetween(startDate, endDate);
        }

        return entries.stream()
                .map(this::toApiScheduleEntry)
                .collect(Collectors.toList());
    }

    @Transactional
    public List<ScheduleEntry> bulkCreateScheduleEntries(List<CreateScheduleEntryRequest> requests) {
        return requests.stream()
                .map(this::createScheduleEntry)
                .collect(Collectors.toList());
    }

    @Transactional
    public ScheduleEntry createScheduleEntry(CreateScheduleEntryRequest request) {
        Employee employee = employeeRepository.findById(request.getEmployeeId())
                .orElseThrow(() -> new ResourceNotFoundException("Mitarbeiter", request.getEmployeeId()));

        Group group = null;
        if (request.getGroupId() != null) {
            group = groupRepository.findById(request.getGroupId())
                    .orElseThrow(() -> new ResourceNotFoundException("Gruppe", request.getGroupId()));
        }

        de.knirpsenstadt.model.ScheduleEntry entry = de.knirpsenstadt.model.ScheduleEntry.builder()
                .employee(employee)
                .group(group)
                .date(request.getDate())
                .startTime(request.getStartTime() != null ? LocalTime.parse(request.getStartTime()) : null)
                .endTime(request.getEndTime() != null ? LocalTime.parse(request.getEndTime()) : null)
                .breakMinutes(request.getBreakMinutes() != null ? request.getBreakMinutes() : 0)
                .entryType(request.getEntryType() != null ? ScheduleEntryType.valueOf(request.getEntryType().getValue()) : ScheduleEntryType.WORK)
                .notes(request.getNotes())
                .build();

        de.knirpsenstadt.model.ScheduleEntry saved = scheduleEntryRepository.save(entry);
        return toApiScheduleEntry(saved);
    }

    @Transactional
    public ScheduleEntry updateScheduleEntry(Long id, UpdateScheduleEntryRequest request) {
        de.knirpsenstadt.model.ScheduleEntry entry = scheduleEntryRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Dienstplan-Eintrag", id));

        if (request.getGroupId() != null) {
            Group group = groupRepository.findById(request.getGroupId())
                    .orElseThrow(() -> new ResourceNotFoundException("Gruppe", request.getGroupId()));
            entry.setGroup(group);
        }

        if (request.getDate() != null) {
            entry.setDate(request.getDate());
        }
        if (request.getStartTime() != null) {
            entry.setStartTime(LocalTime.parse(request.getStartTime()));
        }
        if (request.getEndTime() != null) {
            entry.setEndTime(LocalTime.parse(request.getEndTime()));
        }
        if (request.getBreakMinutes() != null) {
            entry.setBreakMinutes(request.getBreakMinutes());
        }
        if (request.getEntryType() != null) {
            entry.setEntryType(ScheduleEntryType.valueOf(request.getEntryType().getValue()));
        }
        if (request.getNotes() != null) {
            entry.setNotes(request.getNotes());
        }

        de.knirpsenstadt.model.ScheduleEntry saved = scheduleEntryRepository.save(entry);
        return toApiScheduleEntry(saved);
    }

    @Transactional
    public void deleteScheduleEntry(Long id) {
        if (!scheduleEntryRepository.existsById(id)) {
            throw new ResourceNotFoundException("Dienstplan-Eintrag", id);
        }
        scheduleEntryRepository.deleteById(id);
    }

    @Transactional
    public List<ScheduleEntry> copyWeekSchedule(LocalDate sourceWeek, LocalDate targetWeek, Long groupId) {
        LocalDate sourceMonday = sourceWeek.with(TemporalAdjusters.previousOrSame(DayOfWeek.MONDAY));
        LocalDate targetMonday = targetWeek.with(TemporalAdjusters.previousOrSame(DayOfWeek.MONDAY));
        long daysDiff = java.time.temporal.ChronoUnit.DAYS.between(sourceMonday, targetMonday);

        List<de.knirpsenstadt.model.ScheduleEntry> sourceEntries;
        if (groupId != null) {
            sourceEntries = scheduleEntryRepository.findByDateBetweenAndGroupId(
                    sourceMonday, sourceMonday.plusDays(6), groupId);
        } else {
            sourceEntries = scheduleEntryRepository.findByDateBetween(
                    sourceMonday, sourceMonday.plusDays(6));
        }

        // Delete existing entries in target week
        List<de.knirpsenstadt.model.ScheduleEntry> existingTarget;
        if (groupId != null) {
            existingTarget = scheduleEntryRepository.findByDateBetweenAndGroupId(
                    targetMonday, targetMonday.plusDays(6), groupId);
        } else {
            existingTarget = scheduleEntryRepository.findByDateBetween(
                    targetMonday, targetMonday.plusDays(6));
        }
        scheduleEntryRepository.deleteAll(existingTarget);

        // Copy entries
        List<de.knirpsenstadt.model.ScheduleEntry> newEntries = sourceEntries.stream()
                .map(source -> de.knirpsenstadt.model.ScheduleEntry.builder()
                        .employee(source.getEmployee())
                        .group(source.getGroup())
                        .date(source.getDate().plusDays(daysDiff))
                        .startTime(source.getStartTime())
                        .endTime(source.getEndTime())
                        .breakMinutes(source.getBreakMinutes())
                        .entryType(source.getEntryType())
                        .notes(source.getNotes())
                        .build())
                .collect(Collectors.toList());

        List<de.knirpsenstadt.model.ScheduleEntry> saved = scheduleEntryRepository.saveAll(newEntries);
        return saved.stream()
                .map(this::toApiScheduleEntry)
                .collect(Collectors.toList());
    }

    private ScheduleEntry toApiScheduleEntry(de.knirpsenstadt.model.ScheduleEntry entity) {
        ScheduleEntry dto = new ScheduleEntry();
        dto.setId(entity.getId());
        dto.setEmployeeId(entity.getEmployee().getId());
        if (entity.getGroup() != null) {
            dto.setGroupId(entity.getGroup().getId());
        }
        dto.setDate(entity.getDate());
        if (entity.getStartTime() != null) {
            dto.setStartTime(entity.getStartTime().format(TIME_FORMATTER));
        }
        if (entity.getEndTime() != null) {
            dto.setEndTime(entity.getEndTime().format(TIME_FORMATTER));
        }
        dto.setBreakMinutes(entity.getBreakMinutes());
        dto.setEntryType(de.knirpsenstadt.api.model.ScheduleEntryType.fromValue(entity.getEntryType().name()));
        dto.setNotes(entity.getNotes());
        dto.setCreatedAt(entity.getCreatedAt());
        dto.setUpdatedAt(entity.getUpdatedAt());

        // Also include employee info
        dto.setEmployee(AuthService.toApiEmployee(entity.getEmployee()));
        if (entity.getGroup() != null) {
            de.knirpsenstadt.api.model.Group g = new de.knirpsenstadt.api.model.Group();
            g.setId(entity.getGroup().getId());
            g.setName(entity.getGroup().getName());
            g.setColor(entity.getGroup().getColor());
            dto.setGroup(g);
        }

        return dto;
    }
}
