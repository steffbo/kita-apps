package de.knirpsenstadt.model;

import jakarta.persistence.*;
import lombok.*;
import org.hibernate.annotations.CreationTimestamp;

import java.time.LocalDate;
import java.time.OffsetDateTime;
import java.time.temporal.ChronoUnit;

@Entity
@Table(name = "time_entries")
@Getter
@Setter
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class TimeEntry {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "employee_id", nullable = false)
    private Employee employee;

    @Column(nullable = false)
    private LocalDate date;

    @Column(name = "clock_in", nullable = false)
    private OffsetDateTime clockIn;

    @Column(name = "clock_out")
    private OffsetDateTime clockOut;

    @Column(name = "break_minutes")
    @Builder.Default
    private Integer breakMinutes = 0;

    @Enumerated(EnumType.STRING)
    @Column(name = "entry_type", nullable = false)
    @Builder.Default
    private TimeEntryType entryType = TimeEntryType.WORK;

    @Column
    private String notes;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "edited_by")
    private Employee editedBy;

    @Column(name = "edited_at")
    private OffsetDateTime editedAt;

    @Column(name = "edit_reason")
    private String editReason;

    @CreationTimestamp
    @Column(name = "created_at", nullable = false, updatable = false)
    private OffsetDateTime createdAt;

    /**
     * Calculate worked minutes for this entry
     */
    public Integer getWorkedMinutes() {
        if (clockIn == null || clockOut == null) {
            return null;
        }
        long totalMinutes = ChronoUnit.MINUTES.between(clockIn, clockOut);
        return Math.max(0, (int) totalMinutes - (breakMinutes != null ? breakMinutes : 0));
    }

    /**
     * Check if currently clocked in (no clock out time)
     */
    public boolean isActive() {
        return clockIn != null && clockOut == null;
    }
}
