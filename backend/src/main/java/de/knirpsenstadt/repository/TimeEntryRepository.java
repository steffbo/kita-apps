package de.knirpsenstadt.repository;

import de.knirpsenstadt.model.TimeEntry;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.time.LocalDate;
import java.util.List;
import java.util.Optional;

@Repository
public interface TimeEntryRepository extends JpaRepository<TimeEntry, Long> {

    @Query("SELECT te FROM TimeEntry te " +
           "JOIN FETCH te.employee " +
           "WHERE te.date BETWEEN :startDate AND :endDate " +
           "ORDER BY te.date, te.clockIn")
    List<TimeEntry> findByDateBetween(
            @Param("startDate") LocalDate startDate,
            @Param("endDate") LocalDate endDate
    );

    @Query("SELECT te FROM TimeEntry te " +
           "JOIN FETCH te.employee " +
           "WHERE te.employee.id = :employeeId " +
           "AND te.date BETWEEN :startDate AND :endDate " +
           "ORDER BY te.date, te.clockIn")
    List<TimeEntry> findByEmployeeIdAndDateBetween(
            @Param("employeeId") Long employeeId,
            @Param("startDate") LocalDate startDate,
            @Param("endDate") LocalDate endDate
    );

    @Query("SELECT te FROM TimeEntry te " +
           "WHERE te.employee.id = :employeeId " +
           "AND te.date = :date")
    List<TimeEntry> findByEmployeeIdAndDate(
            @Param("employeeId") Long employeeId,
            @Param("date") LocalDate date
    );

    /**
     * Find active (clocked in but not out) entry for employee
     */
    @Query("SELECT te FROM TimeEntry te " +
           "WHERE te.employee.id = :employeeId " +
           "AND te.clockOut IS NULL")
    Optional<TimeEntry> findActiveByEmployeeId(@Param("employeeId") Long employeeId);

    /**
     * Check if employee has an active time entry
     */
    @Query("SELECT COUNT(te) > 0 FROM TimeEntry te " +
           "WHERE te.employee.id = :employeeId " +
           "AND te.clockOut IS NULL")
    boolean existsActiveByEmployeeId(@Param("employeeId") Long employeeId);

    /**
     * Find all open (not clocked out) entries for an employee
     */
    @Query("SELECT te FROM TimeEntry te " +
           "WHERE te.employee.id = :employeeId " +
           "AND te.clockOut IS NULL " +
           "ORDER BY te.clockIn DESC")
    List<TimeEntry> findOpenEntriesByEmployeeId(@Param("employeeId") Long employeeId);

    /**
     * Sum worked minutes for employee in date range
     */
    @Query(value = "SELECT COALESCE(SUM(" +
           "  EXTRACT(EPOCH FROM (clock_out - clock_in)) / 60 - COALESCE(break_minutes, 0)" +
           "), 0) FROM time_entries " +
           "WHERE employee_id = :employeeId " +
           "AND date BETWEEN :startDate AND :endDate " +
           "AND clock_out IS NOT NULL", nativeQuery = true)
    long sumWorkedMinutes(
            @Param("employeeId") Long employeeId,
            @Param("startDate") LocalDate startDate,
            @Param("endDate") LocalDate endDate
    );
}
