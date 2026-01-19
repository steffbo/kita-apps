package de.knirpsenstadt.repository;

import de.knirpsenstadt.model.ScheduleEntry;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.time.LocalDate;
import java.util.List;

@Repository
public interface ScheduleEntryRepository extends JpaRepository<ScheduleEntry, Long> {

    @Query("SELECT se FROM ScheduleEntry se " +
           "JOIN FETCH se.employee " +
           "LEFT JOIN FETCH se.group " +
           "WHERE se.date BETWEEN :startDate AND :endDate " +
           "ORDER BY se.date, se.startTime")
    List<ScheduleEntry> findByDateBetween(
            @Param("startDate") LocalDate startDate,
            @Param("endDate") LocalDate endDate
    );

    @Query("SELECT se FROM ScheduleEntry se " +
           "JOIN FETCH se.employee " +
           "LEFT JOIN FETCH se.group " +
           "WHERE se.employee.id = :employeeId " +
           "AND se.date BETWEEN :startDate AND :endDate " +
           "ORDER BY se.date, se.startTime")
    List<ScheduleEntry> findByEmployeeIdAndDateBetween(
            @Param("employeeId") Long employeeId,
            @Param("startDate") LocalDate startDate,
            @Param("endDate") LocalDate endDate
    );

    @Query("SELECT se FROM ScheduleEntry se " +
           "JOIN FETCH se.employee " +
           "LEFT JOIN FETCH se.group " +
           "WHERE se.group.id = :groupId " +
           "AND se.date BETWEEN :startDate AND :endDate " +
           "ORDER BY se.date, se.startTime")
    List<ScheduleEntry> findByGroupIdAndDateBetween(
            @Param("groupId") Long groupId,
            @Param("startDate") LocalDate startDate,
            @Param("endDate") LocalDate endDate
    );

    @Query("SELECT se FROM ScheduleEntry se " +
           "WHERE se.employee.id = :employeeId " +
           "AND se.date = :date")
    List<ScheduleEntry> findByEmployeeIdAndDate(
            @Param("employeeId") Long employeeId,
            @Param("date") LocalDate date
    );

    @Query("SELECT COUNT(se) FROM ScheduleEntry se " +
           "WHERE se.employee.id = :employeeId " +
           "AND se.date BETWEEN :startDate AND :endDate " +
           "AND se.entryType = 'VACATION'")
    long countVacationDays(
            @Param("employeeId") Long employeeId,
            @Param("startDate") LocalDate startDate,
            @Param("endDate") LocalDate endDate
    );

    @Query("SELECT COUNT(se) FROM ScheduleEntry se " +
           "WHERE se.employee.id = :employeeId " +
           "AND se.date BETWEEN :startDate AND :endDate " +
           "AND se.entryType = 'SICK'")
    long countSickDays(
            @Param("employeeId") Long employeeId,
            @Param("startDate") LocalDate startDate,
            @Param("endDate") LocalDate endDate
    );

    @Query("SELECT se FROM ScheduleEntry se " +
           "JOIN FETCH se.employee " +
           "LEFT JOIN FETCH se.group " +
           "WHERE se.date BETWEEN :startDate AND :endDate " +
           "AND se.group.id = :groupId " +
           "ORDER BY se.date, se.startTime")
    List<ScheduleEntry> findByDateBetweenAndGroupId(
            @Param("startDate") LocalDate startDate,
            @Param("endDate") LocalDate endDate,
            @Param("groupId") Long groupId
    );
}
