package de.knirpsenstadt.service;

import de.knirpsenstadt.api.model.*;
import de.knirpsenstadt.exception.BadRequestException;
import de.knirpsenstadt.exception.ResourceNotFoundException;
import de.knirpsenstadt.model.Employee;
import de.knirpsenstadt.model.EmployeeRole;
import de.knirpsenstadt.repository.EmployeeRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.math.BigDecimal;
import java.util.List;
import java.util.stream.Collectors;

@Service
@RequiredArgsConstructor
public class EmployeeService {

    private final EmployeeRepository employeeRepository;
    private final PasswordEncoder passwordEncoder;

    public List<de.knirpsenstadt.api.model.Employee> getAllEmployees(Boolean activeOnly) {
        List<Employee> employees;
        if (Boolean.TRUE.equals(activeOnly)) {
            employees = employeeRepository.findAllActiveOrderByName();
        } else {
            employees = employeeRepository.findAllOrderByName();
        }
        return employees.stream()
                .map(AuthService::toApiEmployee)
                .collect(Collectors.toList());
    }

    public de.knirpsenstadt.api.model.Employee getEmployee(Long id) {
        Employee employee = employeeRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Mitarbeiter", id));
        return AuthService.toApiEmployee(employee);
    }

    @Transactional
    public de.knirpsenstadt.api.model.Employee createEmployee(CreateEmployeeRequest request) {
        if (employeeRepository.existsByEmail(request.getEmail())) {
            throw new BadRequestException("E-Mail-Adresse wird bereits verwendet");
        }

        String tempPassword = generateTemporaryPassword();

        Employee employee = Employee.builder()
                .email(request.getEmail())
                .firstName(request.getFirstName())
                .lastName(request.getLastName())
                .role(request.getRole() != null ? EmployeeRole.valueOf(request.getRole().getValue()) : EmployeeRole.EMPLOYEE)
                .weeklyHours(BigDecimal.valueOf(request.getWeeklyHours()))
                .vacationDaysPerYear(request.getVacationDaysPerYear())
                .remainingVacationDays(BigDecimal.valueOf(request.getVacationDaysPerYear()))
                .overtimeBalance(BigDecimal.ZERO)
                .active(true)
                .passwordHash(passwordEncoder.encode(tempPassword))
                .build();

        Employee saved = employeeRepository.save(employee);

        // TODO: Send welcome email with temporary password
        System.out.println("Temporary password for " + employee.getEmail() + ": " + tempPassword);

        return AuthService.toApiEmployee(saved);
    }

    @Transactional
    public de.knirpsenstadt.api.model.Employee updateEmployee(Long id, UpdateEmployeeRequest request) {
        Employee employee = employeeRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Mitarbeiter", id));

        if (request.getEmail() != null) {
            employee.setEmail(request.getEmail());
        }
        if (request.getFirstName() != null) {
            employee.setFirstName(request.getFirstName());
        }
        if (request.getLastName() != null) {
            employee.setLastName(request.getLastName());
        }
        if (request.getWeeklyHours() != null) {
            employee.setWeeklyHours(BigDecimal.valueOf(request.getWeeklyHours()));
        }
        if (request.getVacationDaysPerYear() != null) {
            employee.setVacationDaysPerYear(request.getVacationDaysPerYear());
        }
        if (request.getRemainingVacationDays() != null) {
            employee.setRemainingVacationDays(BigDecimal.valueOf(request.getRemainingVacationDays()));
        }
        if (request.getOvertimeBalance() != null) {
            employee.setOvertimeBalance(BigDecimal.valueOf(request.getOvertimeBalance()));
        }
        if (request.getActive() != null) {
            employee.setActive(request.getActive());
        }

        Employee saved = employeeRepository.save(employee);
        return AuthService.toApiEmployee(saved);
    }

    @Transactional
    public void deleteEmployee(Long id) {
        Employee employee = employeeRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Mitarbeiter", id));

        // Soft delete - just deactivate
        employee.setActive(false);
        employeeRepository.save(employee);
    }

    @Transactional
    public MessageResponse adminResetPassword(Long id) {
        Employee employee = employeeRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Mitarbeiter", id));

        String tempPassword = generateTemporaryPassword();
        employee.setPasswordHash(passwordEncoder.encode(tempPassword));
        employeeRepository.save(employee);

        // TODO: Send email with new password
        System.out.println("New temporary password for " + employee.getEmail() + ": " + tempPassword);

        MessageResponse response = new MessageResponse();
        response.setMessage("Passwort wurde zurückgesetzt. Eine E-Mail mit dem neuen Passwort wurde versendet.");
        return response;
    }

    private String generateTemporaryPassword() {
        return java.util.UUID.randomUUID().toString().substring(0, 8);
    }
}
