package de.knirpsenstadt.service;

import de.knirpsenstadt.api.model.*;
import de.knirpsenstadt.exception.BadRequestException;
import de.knirpsenstadt.exception.ResourceNotFoundException;
import de.knirpsenstadt.model.AssignmentType;
import de.knirpsenstadt.model.Employee;
import de.knirpsenstadt.model.EmployeeRole;
import de.knirpsenstadt.model.Group;
import de.knirpsenstadt.model.GroupAssignment;
import de.knirpsenstadt.repository.EmployeeRepository;
import de.knirpsenstadt.repository.GroupAssignmentRepository;
import de.knirpsenstadt.repository.GroupRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.math.BigDecimal;
import java.util.List;
import java.util.Optional;
import java.util.stream.Collectors;

@Service
@RequiredArgsConstructor
public class EmployeeService {

    private final EmployeeRepository employeeRepository;
    private final GroupAssignmentRepository groupAssignmentRepository;
    private final GroupRepository groupRepository;
    private final PasswordEncoder passwordEncoder;

    public List<de.knirpsenstadt.api.model.Employee> getAllEmployees(Boolean activeOnly) {
        List<Employee> employees;
        if (Boolean.TRUE.equals(activeOnly)) {
            employees = employeeRepository.findAllActiveOrderByName();
        } else {
            employees = employeeRepository.findAllOrderByName();
        }
        return employees.stream()
                .map(this::toApiEmployeeWithPrimaryGroup)
                .collect(Collectors.toList());
    }

    public de.knirpsenstadt.api.model.Employee getEmployee(Long id) {
        Employee employee = employeeRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Mitarbeiter", id));
        return toApiEmployeeWithPrimaryGroup(employee);
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

        // Handle primary group assignment (0 or null means no group / Springer)
        if (request.getPrimaryGroupId() != null && request.getPrimaryGroupId() > 0) {
            setPrimaryGroup(saved, request.getPrimaryGroupId());
        }

        // TODO: Send welcome email with temporary password
        System.out.println("Temporary password for " + employee.getEmail() + ": " + tempPassword);

        return toApiEmployeeWithPrimaryGroup(saved);
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
        if (request.getRole() != null) {
            employee.setRole(EmployeeRole.valueOf(request.getRole().getValue()));
        }

        Employee saved = employeeRepository.save(employee);

        // Handle primary group assignment if provided
        if (request.getPrimaryGroupId() != null) {
            if (request.getPrimaryGroupId() == 0) {
                // Remove primary group assignment
                removePrimaryGroup(saved.getId());
            } else {
                setPrimaryGroup(saved, request.getPrimaryGroupId());
            }
        }

        return toApiEmployeeWithPrimaryGroup(saved);
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

    /**
     * Get all group assignments for an employee
     */
    public List<de.knirpsenstadt.api.model.GroupAssignment> getEmployeeAssignments(Long employeeId) {
        if (!employeeRepository.existsById(employeeId)) {
            throw new ResourceNotFoundException("Mitarbeiter", employeeId);
        }

        List<GroupAssignment> assignments = groupAssignmentRepository.findByEmployeeIdWithGroup(employeeId);
        return assignments.stream()
                .map(this::toApiGroupAssignment)
                .collect(Collectors.toList());
    }

    /**
     * Set the primary group (Stammgruppe) for an employee
     */
    private void setPrimaryGroup(Employee employee, Long groupId) {
        Group group = groupRepository.findById(groupId)
                .orElseThrow(() -> new ResourceNotFoundException("Gruppe", groupId));

        // Check existing PERMANENT assignment
        List<GroupAssignment> existingPrimary = groupAssignmentRepository
                .findByEmployeeIdAndAssignmentType(employee.getId(), AssignmentType.PERMANENT);
        
        if (!existingPrimary.isEmpty()) {
            GroupAssignment existing = existingPrimary.get(0);
            
            // If already assigned to the same group, nothing to do
            if (existing.getGroup().getId().equals(groupId)) {
                return;
            }
            
            // Update existing assignment to new group
            existing.setGroup(group);
            groupAssignmentRepository.save(existing);
        } else {
            // Create new PERMANENT assignment
            GroupAssignment assignment = GroupAssignment.builder()
                    .employee(employee)
                    .group(group)
                    .assignmentType(AssignmentType.PERMANENT)
                    .build();
            
            groupAssignmentRepository.save(assignment);
        }
    }

    /**
     * Remove the primary group assignment for an employee
     */
    private void removePrimaryGroup(Long employeeId) {
        List<GroupAssignment> existingPrimary = groupAssignmentRepository
                .findByEmployeeIdAndAssignmentType(employeeId, AssignmentType.PERMANENT);
        
        if (!existingPrimary.isEmpty()) {
            groupAssignmentRepository.deleteAll(existingPrimary);
        }
    }

    /**
     * Get the primary group ID for an employee
     */
    private Optional<GroupAssignment> getPrimaryGroupAssignment(Long employeeId) {
        List<GroupAssignment> assignments = groupAssignmentRepository
                .findByEmployeeIdAndAssignmentType(employeeId, AssignmentType.PERMANENT);
        return assignments.isEmpty() ? Optional.empty() : Optional.of(assignments.get(0));
    }

    /**
     * Convert entity to API model with primary group info
     */
    private de.knirpsenstadt.api.model.Employee toApiEmployeeWithPrimaryGroup(Employee entity) {
        de.knirpsenstadt.api.model.Employee dto = AuthService.toApiEmployee(entity);
        
        // Add primary group info
        getPrimaryGroupAssignment(entity.getId()).ifPresent(assignment -> {
            dto.setPrimaryGroupId(assignment.getGroup().getId());
            
            de.knirpsenstadt.api.model.Group groupDto = new de.knirpsenstadt.api.model.Group();
            groupDto.setId(assignment.getGroup().getId());
            groupDto.setName(assignment.getGroup().getName());
            groupDto.setDescription(assignment.getGroup().getDescription());
            groupDto.setColor(assignment.getGroup().getColor());
            dto.setPrimaryGroup(groupDto);
        });
        
        return dto;
    }

    private de.knirpsenstadt.api.model.GroupAssignment toApiGroupAssignment(GroupAssignment entity) {
        de.knirpsenstadt.api.model.GroupAssignment dto = new de.knirpsenstadt.api.model.GroupAssignment();
        dto.setId(entity.getId());
        dto.setEmployeeId(entity.getEmployee().getId());
        dto.setGroupId(entity.getGroup().getId());
        dto.setAssignmentType(de.knirpsenstadt.api.model.AssignmentType.fromValue(entity.getAssignmentType().name()));
        
        // Include group info
        de.knirpsenstadt.api.model.Group groupDto = new de.knirpsenstadt.api.model.Group();
        groupDto.setId(entity.getGroup().getId());
        groupDto.setName(entity.getGroup().getName());
        groupDto.setDescription(entity.getGroup().getDescription());
        groupDto.setColor(entity.getGroup().getColor());
        
        return dto;
    }
}
