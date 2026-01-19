package de.knirpsenstadt.controller;

import de.knirpsenstadt.api.EmployeesApi;
import de.knirpsenstadt.api.model.*;
import de.knirpsenstadt.service.EmployeeService;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

@RestController
@RequiredArgsConstructor
public class EmployeeController implements EmployeesApi {

    private final EmployeeService employeeService;

    @Override
    public ResponseEntity<List<Employee>> listEmployees(Boolean includeInactive) {
        // Convert includeInactive to activeOnly (inverted logic)
        Boolean activeOnly = includeInactive != null ? !includeInactive : true;
        List<Employee> employees = employeeService.getAllEmployees(activeOnly);
        return ResponseEntity.ok(employees);
    }

    @Override
    public ResponseEntity<Employee> getEmployee(Long id) {
        Employee employee = employeeService.getEmployee(id);
        return ResponseEntity.ok(employee);
    }

    @Override
    public ResponseEntity<Employee> createEmployee(CreateEmployeeRequest createEmployeeRequest) {
        Employee employee = employeeService.createEmployee(createEmployeeRequest);
        return ResponseEntity.status(201).body(employee);
    }

    @Override
    public ResponseEntity<Employee> updateEmployee(Long id, UpdateEmployeeRequest updateEmployeeRequest) {
        Employee employee = employeeService.updateEmployee(id, updateEmployeeRequest);
        return ResponseEntity.ok(employee);
    }

    @Override
    public ResponseEntity<Void> deleteEmployee(Long id) {
        employeeService.deleteEmployee(id);
        return ResponseEntity.noContent().build();
    }

    @Override
    public ResponseEntity<MessageResponse> adminResetPassword(Long id) {
        MessageResponse response = employeeService.adminResetPassword(id);
        return ResponseEntity.ok(response);
    }
}
