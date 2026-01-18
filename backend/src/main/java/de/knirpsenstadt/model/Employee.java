package de.knirpsenstadt.model;

import jakarta.persistence.*;
import lombok.*;
import org.hibernate.annotations.CreationTimestamp;
import org.hibernate.annotations.UpdateTimestamp;

import java.math.BigDecimal;
import java.time.OffsetDateTime;

@Entity
@Table(name = "employees")
@Getter
@Setter
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class Employee {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(nullable = false, unique = true)
    private String email;

    @Column(name = "password_hash", nullable = false)
    private String passwordHash;

    @Column(name = "first_name", nullable = false)
    private String firstName;

    @Column(name = "last_name", nullable = false)
    private String lastName;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false)
    @Builder.Default
    private EmployeeRole role = EmployeeRole.EMPLOYEE;

    @Column(name = "weekly_hours", nullable = false, precision = 4, scale = 2)
    private BigDecimal weeklyHours;

    @Column(name = "vacation_days_per_year", nullable = false)
    @Builder.Default
    private Integer vacationDaysPerYear = 30;

    @Column(name = "remaining_vacation_days", nullable = false, precision = 5, scale = 2)
    @Builder.Default
    private BigDecimal remainingVacationDays = BigDecimal.valueOf(30);

    @Column(name = "overtime_balance", nullable = false, precision = 6, scale = 2)
    @Builder.Default
    private BigDecimal overtimeBalance = BigDecimal.ZERO;

    @Column(nullable = false)
    @Builder.Default
    private Boolean active = true;

    @CreationTimestamp
    @Column(name = "created_at", nullable = false, updatable = false)
    private OffsetDateTime createdAt;

    @UpdateTimestamp
    @Column(name = "updated_at", nullable = false)
    private OffsetDateTime updatedAt;

    public String getFullName() {
        return firstName + " " + lastName;
    }
}
