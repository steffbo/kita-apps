package de.knirpsenstadt.service;

import de.knirpsenstadt.api.model.*;
import de.knirpsenstadt.exception.ResourceNotFoundException;
import de.knirpsenstadt.model.SpecialDayType;
import de.knirpsenstadt.repository.SpecialDayRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.LocalDate;
import java.util.List;
import java.util.stream.Collectors;

@Service
@RequiredArgsConstructor
public class SpecialDayService {

    private final SpecialDayRepository specialDayRepository;

    public List<SpecialDay> getSpecialDays(Integer year, Boolean includeHolidays) {
        LocalDate start = LocalDate.of(year, 1, 1);
        LocalDate end = LocalDate.of(year, 12, 31);

        List<de.knirpsenstadt.model.SpecialDay> days;
        if (includeHolidays != null && includeHolidays) {
            days = specialDayRepository.findByDateBetween(start, end);
        } else {
            // Exclude holidays
            days = specialDayRepository.findByDateBetween(start, end).stream()
                    .filter(d -> d.getDayType() != SpecialDayType.HOLIDAY)
                    .collect(Collectors.toList());
        }

        return days.stream()
                .map(this::toApiSpecialDay)
                .collect(Collectors.toList());
    }

    public List<SpecialDay> getHolidays(Integer year) {
        int targetYear = year != null ? year : LocalDate.now().getYear();
        LocalDate start = LocalDate.of(targetYear, 1, 1);
        LocalDate end = LocalDate.of(targetYear, 12, 31);

        List<de.knirpsenstadt.model.SpecialDay> holidays = specialDayRepository
                .findByDateBetweenAndType(start, end, SpecialDayType.HOLIDAY);

        return holidays.stream()
                .map(this::toApiSpecialDay)
                .collect(Collectors.toList());
    }

    @Transactional
    public SpecialDay createSpecialDay(CreateSpecialDayRequest request) {
        de.knirpsenstadt.model.SpecialDay day = de.knirpsenstadt.model.SpecialDay.builder()
                .date(request.getDate())
                .name(request.getName())
                .dayType(SpecialDayType.valueOf(request.getDayType().getValue()))
                .affectsAll(request.getAffectsAll() != null ? request.getAffectsAll() : true)
                .notes(request.getNotes())
                .build();

        de.knirpsenstadt.model.SpecialDay saved = specialDayRepository.save(day);
        return toApiSpecialDay(saved);
    }

    @Transactional
    public SpecialDay updateSpecialDay(Long id, CreateSpecialDayRequest request) {
        de.knirpsenstadt.model.SpecialDay day = specialDayRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Besonderer Tag", id));

        day.setDate(request.getDate());
        day.setName(request.getName());
        day.setDayType(SpecialDayType.valueOf(request.getDayType().getValue()));
        if (request.getAffectsAll() != null) {
            day.setAffectsAll(request.getAffectsAll());
        }
        if (request.getNotes() != null) {
            day.setNotes(request.getNotes());
        }

        de.knirpsenstadt.model.SpecialDay saved = specialDayRepository.save(day);
        return toApiSpecialDay(saved);
    }

    @Transactional
    public void deleteSpecialDay(Long id) {
        if (!specialDayRepository.existsById(id)) {
            throw new ResourceNotFoundException("Besonderer Tag", id);
        }
        specialDayRepository.deleteById(id);
    }

    private SpecialDay toApiSpecialDay(de.knirpsenstadt.model.SpecialDay entity) {
        SpecialDay dto = new SpecialDay();
        dto.setId(entity.getId());
        dto.setDate(entity.getDate());
        dto.setName(entity.getName());
        dto.setDayType(de.knirpsenstadt.api.model.SpecialDayType.fromValue(entity.getDayType().name()));
        dto.setAffectsAll(entity.getAffectsAll());
        dto.setNotes(entity.getNotes());
        return dto;
    }
}
