package de.knirpsenstadt.controller;

import de.knirpsenstadt.api.SpecialDaysApi;
import de.knirpsenstadt.api.model.*;
import de.knirpsenstadt.service.SpecialDayService;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

@RestController
@RequiredArgsConstructor
public class SpecialDayController implements SpecialDaysApi {

    private final SpecialDayService specialDayService;

    @Override
    public ResponseEntity<List<SpecialDay>> getSpecialDays(Integer year, Boolean includeHolidays) {
        List<SpecialDay> days = specialDayService.getSpecialDays(year, includeHolidays);
        return ResponseEntity.ok(days);
    }

    @Override
    public ResponseEntity<List<SpecialDay>> getHolidays(Integer year) {
        List<SpecialDay> holidays = specialDayService.getHolidays(year);
        return ResponseEntity.ok(holidays);
    }

    @Override
    public ResponseEntity<SpecialDay> createSpecialDay(CreateSpecialDayRequest createSpecialDayRequest) {
        SpecialDay day = specialDayService.createSpecialDay(createSpecialDayRequest);
        return ResponseEntity.status(201).body(day);
    }

    @Override
    public ResponseEntity<SpecialDay> updateSpecialDay(Long id, CreateSpecialDayRequest createSpecialDayRequest) {
        SpecialDay day = specialDayService.updateSpecialDay(id, createSpecialDayRequest);
        return ResponseEntity.ok(day);
    }

    @Override
    public ResponseEntity<Void> deleteSpecialDay(Long id) {
        specialDayService.deleteSpecialDay(id);
        return ResponseEntity.noContent().build();
    }
}
