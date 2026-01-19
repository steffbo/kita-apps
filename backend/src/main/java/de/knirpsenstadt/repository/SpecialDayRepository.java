package de.knirpsenstadt.repository;

import de.knirpsenstadt.model.SpecialDay;
import de.knirpsenstadt.model.SpecialDayType;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.time.LocalDate;
import java.util.List;
import java.util.Optional;

@Repository
public interface SpecialDayRepository extends JpaRepository<SpecialDay, Long> {

    @Query("SELECT sd FROM SpecialDay sd " +
           "WHERE EXTRACT(YEAR FROM sd.date) = :year " +
           "ORDER BY sd.date")
    List<SpecialDay> findByYear(@Param("year") int year);

    @Query("SELECT sd FROM SpecialDay sd " +
           "WHERE EXTRACT(YEAR FROM sd.date) = :year " +
           "AND sd.dayType = :dayType " +
           "ORDER BY sd.date")
    List<SpecialDay> findByYearAndDayType(
            @Param("year") int year,
            @Param("dayType") SpecialDayType dayType
    );

    List<SpecialDay> findByDateBetween(LocalDate startDate, LocalDate endDate);

    Optional<SpecialDay> findByDateAndDayType(LocalDate date, SpecialDayType dayType);

    @Query("SELECT sd FROM SpecialDay sd " +
           "WHERE sd.date = :date " +
           "AND sd.dayType = 'HOLIDAY'")
    Optional<SpecialDay> findHolidayByDate(@Param("date") LocalDate date);

    @Query("SELECT COUNT(sd) > 0 FROM SpecialDay sd " +
           "WHERE sd.date = :date " +
           "AND sd.dayType = 'HOLIDAY'")
    boolean isHoliday(@Param("date") LocalDate date);

    @Query("SELECT COUNT(sd) > 0 FROM SpecialDay sd " +
           "WHERE sd.date = :date " +
           "AND sd.dayType = 'CLOSURE'")
    boolean isClosure(@Param("date") LocalDate date);

    @Query("SELECT sd FROM SpecialDay sd " +
           "WHERE sd.date BETWEEN :startDate AND :endDate " +
           "AND sd.dayType = :dayType " +
           "ORDER BY sd.date")
    List<SpecialDay> findByDateBetweenAndType(
            @Param("startDate") LocalDate startDate,
            @Param("endDate") LocalDate endDate,
            @Param("dayType") SpecialDayType dayType
    );
}
