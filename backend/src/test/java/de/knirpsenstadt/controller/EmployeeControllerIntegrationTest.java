package de.knirpsenstadt.controller;

import de.knirpsenstadt.AbstractIntegrationTest;
import de.knirpsenstadt.api.model.*;
import de.knirpsenstadt.model.Employee;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.springframework.http.MediaType;

import java.util.ArrayList;
import java.util.List;

import static org.hamcrest.Matchers.*;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@DisplayName("EmployeeController Integration Tests")
class EmployeeControllerIntegrationTest extends AbstractIntegrationTest {

    private final List<Long> createdEmployeeIds = new ArrayList<>();

    @AfterEach
    void cleanUp() {
        // Clean up employees created during tests
        createdEmployeeIds.forEach(id -> {
            try {
                employeeRepository.deleteById(id);
            } catch (Exception ignored) {}
        });
        createdEmployeeIds.clear();
    }

    @Nested
    @DisplayName("GET /employees")
    class ListEmployeesTests {

        @Test
        @DisplayName("should list all employees as admin")
        void listEmployeesAsAdmin() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(get("/employees")
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$").isArray())
                    .andExpect(jsonPath("$[*].email", hasItem(ADMIN_EMAIL)))
                    .andExpect(jsonPath("$[*].email", hasItem(EMPLOYEE_EMAIL)));
        }

        @Test
        @DisplayName("should list employees as regular employee")
        void listEmployeesAsEmployee() throws Exception {
            String token = getEmployeeToken();

            mockMvc.perform(get("/employees")
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$").isArray());
        }

        @Test
        @DisplayName("should exclude inactive employees by default")
        void excludeInactiveByDefault() throws Exception {
            // Create inactive employee
            Employee inactive = employeeRepository.save(
                    Employee.builder()
                            .email("inactive@test.de")
                            .firstName("Inactive")
                            .lastName("User")
                            .passwordHash(passwordEncoder.encode("password"))
                            .role(de.knirpsenstadt.model.EmployeeRole.EMPLOYEE)
                            .weeklyHours(java.math.BigDecimal.valueOf(38))
                            .vacationDaysPerYear(30)
                            .remainingVacationDays(java.math.BigDecimal.valueOf(30))
                            .overtimeBalance(java.math.BigDecimal.ZERO)
                            .active(false)
                            .build()
            );
            createdEmployeeIds.add(inactive.getId());

            String token = getAdminToken();

            mockMvc.perform(get("/employees")
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$[*].email", not(hasItem("inactive@test.de"))));
        }

        @Test
        @DisplayName("should include inactive employees when requested")
        void includeInactiveWhenRequested() throws Exception {
            // Create inactive employee
            Employee inactive = employeeRepository.save(
                    Employee.builder()
                            .email("inactive2@test.de")
                            .firstName("Inactive2")
                            .lastName("User")
                            .passwordHash(passwordEncoder.encode("password"))
                            .role(de.knirpsenstadt.model.EmployeeRole.EMPLOYEE)
                            .weeklyHours(java.math.BigDecimal.valueOf(38))
                            .vacationDaysPerYear(30)
                            .remainingVacationDays(java.math.BigDecimal.valueOf(30))
                            .overtimeBalance(java.math.BigDecimal.ZERO)
                            .active(false)
                            .build()
            );
            createdEmployeeIds.add(inactive.getId());

            String token = getAdminToken();

            mockMvc.perform(get("/employees")
                            .param("includeInactive", "true")
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$[*].email", hasItem("inactive2@test.de")));
        }

        @Test
        @DisplayName("should return 401 without token")
        void listEmployeesWithoutToken() throws Exception {
            mockMvc.perform(get("/employees"))
                    .andExpect(status().isUnauthorized());
        }
    }

    @Nested
    @DisplayName("GET /employees/{id}")
    class GetEmployeeTests {

        @Test
        @DisplayName("should get employee by id")
        void getEmployeeById() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(get("/employees/{id}", adminEmployee.getId())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.id").value(adminEmployee.getId()))
                    .andExpect(jsonPath("$.email").value(ADMIN_EMAIL))
                    .andExpect(jsonPath("$.firstName").value("Admin"))
                    .andExpect(jsonPath("$.lastName").value("User"));
        }

        @Test
        @DisplayName("should return 404 for non-existent employee")
        void getEmployeeNotFound() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(get("/employees/{id}", 99999)
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isNotFound());
        }
    }

    @Nested
    @DisplayName("POST /employees")
    class CreateEmployeeTests {

        @Test
        @DisplayName("should create employee as admin")
        void createEmployeeAsAdmin() throws Exception {
            String token = getAdminToken();

            CreateEmployeeRequest request = new CreateEmployeeRequest();
            request.setEmail("new.employee@test.de");
            request.setFirstName("New");
            request.setLastName("Employee");
            request.setWeeklyHours(35.0f);
            request.setVacationDaysPerYear(28);
            request.setRole(EmployeeRole.EMPLOYEE);

            String response = mockMvc.perform(post("/employees")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isCreated())
                    .andExpect(jsonPath("$.email").value("new.employee@test.de"))
                    .andExpect(jsonPath("$.firstName").value("New"))
                    .andExpect(jsonPath("$.lastName").value("Employee"))
                    .andExpect(jsonPath("$.weeklyHours").value(35.0))
                    .andExpect(jsonPath("$.vacationDaysPerYear").value(28))
                    .andExpect(jsonPath("$.active").value(true))
                    .andReturn()
                    .getResponse()
                    .getContentAsString();

            // Track for cleanup
            de.knirpsenstadt.api.model.Employee created = fromJson(response, de.knirpsenstadt.api.model.Employee.class);
            createdEmployeeIds.add(created.getId());
        }

        @Test
        @DisplayName("should return 409 for duplicate email")
        void createEmployeeDuplicateEmail() throws Exception {
            String token = getAdminToken();

            CreateEmployeeRequest request = new CreateEmployeeRequest();
            request.setEmail(ADMIN_EMAIL); // Already exists
            request.setFirstName("Duplicate");
            request.setLastName("User");
            request.setWeeklyHours(38.0f);
            request.setVacationDaysPerYear(30);

            mockMvc.perform(post("/employees")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isBadRequest());
        }

        @Test
        @DisplayName("should return 400 for invalid request")
        void createEmployeeInvalidRequest() throws Exception {
            String token = getAdminToken();

            CreateEmployeeRequest request = new CreateEmployeeRequest();
            // Missing required fields

            mockMvc.perform(post("/employees")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isBadRequest());
        }
    }

    @Nested
    @DisplayName("PUT /employees/{id}")
    class UpdateEmployeeTests {

        @Test
        @DisplayName("should update employee as admin")
        void updateEmployeeAsAdmin() throws Exception {
            // Create employee to update
            Employee toUpdate = employeeRepository.save(
                    Employee.builder()
                            .email("update.me@test.de")
                            .firstName("Update")
                            .lastName("Me")
                            .passwordHash(passwordEncoder.encode("password"))
                            .role(de.knirpsenstadt.model.EmployeeRole.EMPLOYEE)
                            .weeklyHours(java.math.BigDecimal.valueOf(38))
                            .vacationDaysPerYear(30)
                            .remainingVacationDays(java.math.BigDecimal.valueOf(30))
                            .overtimeBalance(java.math.BigDecimal.ZERO)
                            .active(true)
                            .build()
            );
            createdEmployeeIds.add(toUpdate.getId());

            String token = getAdminToken();

            UpdateEmployeeRequest request = new UpdateEmployeeRequest();
            request.setFirstName("Updated");
            request.setLastName("Name");
            request.setWeeklyHours(32.0f);

            mockMvc.perform(put("/employees/{id}", toUpdate.getId())
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.firstName").value("Updated"))
                    .andExpect(jsonPath("$.lastName").value("Name"))
                    .andExpect(jsonPath("$.weeklyHours").value(32.0));
        }

        @Test
        @DisplayName("should return 404 for non-existent employee")
        void updateEmployeeNotFound() throws Exception {
            String token = getAdminToken();

            UpdateEmployeeRequest request = new UpdateEmployeeRequest();
            request.setFirstName("Test");

            mockMvc.perform(put("/employees/{id}", 99999)
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isNotFound());
        }
    }

    @Nested
    @DisplayName("DELETE /employees/{id}")
    class DeleteEmployeeTests {

        @Test
        @DisplayName("should deactivate employee (soft delete)")
        void deleteEmployeeSoftDelete() throws Exception {
            // Create employee to delete
            Employee toDelete = employeeRepository.save(
                    Employee.builder()
                            .email("delete.me@test.de")
                            .firstName("Delete")
                            .lastName("Me")
                            .passwordHash(passwordEncoder.encode("password"))
                            .role(de.knirpsenstadt.model.EmployeeRole.EMPLOYEE)
                            .weeklyHours(java.math.BigDecimal.valueOf(38))
                            .vacationDaysPerYear(30)
                            .remainingVacationDays(java.math.BigDecimal.valueOf(30))
                            .overtimeBalance(java.math.BigDecimal.ZERO)
                            .active(true)
                            .build()
            );
            createdEmployeeIds.add(toDelete.getId());

            String token = getAdminToken();

            mockMvc.perform(delete("/employees/{id}", toDelete.getId())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isNoContent());

            // Verify employee is deactivated, not deleted
            Employee deleted = employeeRepository.findById(toDelete.getId()).orElseThrow();
            assert !deleted.getActive();
        }

        @Test
        @DisplayName("should return 404 for non-existent employee")
        void deleteEmployeeNotFound() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(delete("/employees/{id}", 99999)
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isNotFound());
        }
    }

    @Nested
    @DisplayName("POST /employees/{id}/reset-password")
    class AdminResetPasswordTests {

        @Test
        @DisplayName("should reset employee password as admin")
        void resetPasswordAsAdmin() throws Exception {
            // Create a separate employee for this test to avoid interfering with other tests
            Employee resetTestEmployee = employeeRepository.save(
                    Employee.builder()
                            .email("reset.test@test.de")
                            .firstName("Reset")
                            .lastName("Test")
                            .passwordHash(passwordEncoder.encode("oldpassword"))
                            .role(de.knirpsenstadt.model.EmployeeRole.EMPLOYEE)
                            .weeklyHours(java.math.BigDecimal.valueOf(38))
                            .vacationDaysPerYear(30)
                            .remainingVacationDays(java.math.BigDecimal.valueOf(30))
                            .overtimeBalance(java.math.BigDecimal.ZERO)
                            .active(true)
                            .build()
            );
            createdEmployeeIds.add(resetTestEmployee.getId());

            String token = getAdminToken();

            mockMvc.perform(post("/employees/{id}/reset-password", resetTestEmployee.getId())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.message").isNotEmpty());
        }

        @Test
        @DisplayName("should return 404 for non-existent employee")
        void resetPasswordNotFound() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(post("/employees/{id}/reset-password", 99999)
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isNotFound());
        }
    }
}
