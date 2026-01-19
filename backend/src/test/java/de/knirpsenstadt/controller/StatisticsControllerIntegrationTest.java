package de.knirpsenstadt.controller;

import de.knirpsenstadt.AbstractIntegrationTest;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;

import java.time.LocalDate;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@DisplayName("StatisticsController Integration Tests")
class StatisticsControllerIntegrationTest extends AbstractIntegrationTest {

    @Nested
    @DisplayName("GET /statistics/overview")
    class GetOverviewStatisticsTests {

        @Test
        @DisplayName("should get overview statistics for month")
        void getOverviewStatistics() throws Exception {
            String token = getAdminToken();
            
            // First day of current month
            LocalDate monthStart = LocalDate.now().withDayOfMonth(1);

            mockMvc.perform(get("/statistics/overview")
                            .param("month", monthStart.toString())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.month").value(monthStart.toString()))
                    .andExpect(jsonPath("$.totalEmployees").isNumber())
                    .andExpect(jsonPath("$.totalScheduledHours").isNumber())
                    .andExpect(jsonPath("$.totalWorkedHours").isNumber())
                    .andExpect(jsonPath("$.totalOvertimeHours").isNumber())
                    .andExpect(jsonPath("$.sickDays").isNumber())
                    .andExpect(jsonPath("$.vacationDays").isNumber())
                    .andExpect(jsonPath("$.employeeStats").isArray());
        }

        @Test
        @DisplayName("should get overview statistics for specific month")
        void getOverviewStatisticsForSpecificMonth() throws Exception {
            String token = getAdminToken();
            
            LocalDate specificMonth = LocalDate.of(2024, 1, 1);

            mockMvc.perform(get("/statistics/overview")
                            .param("month", specificMonth.toString())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.month").value(specificMonth.toString()));
        }

        @Test
        @DisplayName("should return 401 without token")
        void getOverviewStatisticsWithoutToken() throws Exception {
            LocalDate monthStart = LocalDate.now().withDayOfMonth(1);

            mockMvc.perform(get("/statistics/overview")
                            .param("month", monthStart.toString()))
                    .andExpect(status().isUnauthorized());
        }

        @Test
        @DisplayName("employee should be denied access to overview statistics")
        void employeeCannotViewOverviewStatistics() throws Exception {
            String token = getEmployeeToken();
            
            LocalDate monthStart = LocalDate.now().withDayOfMonth(1);

            mockMvc.perform(get("/statistics/overview")
                            .param("month", monthStart.toString())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isForbidden());
        }
    }

    @Nested
    @DisplayName("GET /statistics/employee/{id}")
    class GetEmployeeStatisticsTests {

        @Test
        @DisplayName("should get employee statistics")
        void getEmployeeStatistics() throws Exception {
            String token = getAdminToken();
            
            LocalDate monthStart = LocalDate.now().withDayOfMonth(1);

            mockMvc.perform(get("/statistics/employee/{id}", regularEmployee.getId())
                            .param("month", monthStart.toString())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.month").value(monthStart.toString()))
                    .andExpect(jsonPath("$.contractedHours").isNumber())
                    .andExpect(jsonPath("$.scheduledHours").isNumber())
                    .andExpect(jsonPath("$.workedHours").isNumber())
                    .andExpect(jsonPath("$.overtimeHours").isNumber())
                    .andExpect(jsonPath("$.overtimeBalance").isNumber())
                    .andExpect(jsonPath("$.vacationDaysUsed").isNumber())
                    .andExpect(jsonPath("$.vacationDaysRemaining").isNumber())
                    .andExpect(jsonPath("$.sickDays").isNumber())
                    .andExpect(jsonPath("$.dailyBreakdown").isArray());
        }

        @Test
        @DisplayName("should get own statistics as employee")
        void getOwnStatisticsAsEmployee() throws Exception {
            String token = getEmployeeToken();
            
            LocalDate monthStart = LocalDate.now().withDayOfMonth(1);

            mockMvc.perform(get("/statistics/employee/{id}", regularEmployee.getId())
                            .param("month", monthStart.toString())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.month").value(monthStart.toString()));
        }

        @Test
        @DisplayName("admin can get any employee statistics")
        void adminCanGetAnyEmployeeStatistics() throws Exception {
            String token = getAdminToken();
            
            LocalDate monthStart = LocalDate.now().withDayOfMonth(1);

            mockMvc.perform(get("/statistics/employee/{id}", adminEmployee.getId())
                            .param("month", monthStart.toString())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk());
        }

        @Test
        @DisplayName("should return 401 without token")
        void getEmployeeStatisticsWithoutToken() throws Exception {
            LocalDate monthStart = LocalDate.now().withDayOfMonth(1);

            mockMvc.perform(get("/statistics/employee/{id}", regularEmployee.getId())
                            .param("month", monthStart.toString()))
                    .andExpect(status().isUnauthorized());
        }
    }

    @Nested
    @DisplayName("GET /statistics/weekly")
    class GetWeeklyStatisticsTests {

        @Test
        @DisplayName("should get weekly statistics")
        void getWeeklyStatistics() throws Exception {
            String token = getAdminToken();
            
            // Get Monday of current week
            LocalDate today = LocalDate.now();
            LocalDate weekStart = today.minusDays(today.getDayOfWeek().getValue() - 1);

            mockMvc.perform(get("/statistics/weekly")
                            .param("weekStart", weekStart.toString())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.weekStart").value(weekStart.toString()))
                    .andExpect(jsonPath("$.weekEnd").value(weekStart.plusDays(6).toString()))
                    .andExpect(jsonPath("$.byEmployee").isArray())
                    .andExpect(jsonPath("$.byGroup").isArray())
                    .andExpect(jsonPath("$.totalScheduledHours").isNumber())
                    .andExpect(jsonPath("$.totalWorkedHours").isNumber());
        }

        @Test
        @DisplayName("should get weekly statistics for specific week")
        void getWeeklyStatisticsForSpecificWeek() throws Exception {
            String token = getAdminToken();
            
            LocalDate weekStart = LocalDate.of(2024, 1, 8); // A Monday

            mockMvc.perform(get("/statistics/weekly")
                            .param("weekStart", weekStart.toString())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.weekStart").value(weekStart.toString()));
        }

        @Test
        @DisplayName("should return 401 without token")
        void getWeeklyStatisticsWithoutToken() throws Exception {
            LocalDate weekStart = LocalDate.now();

            mockMvc.perform(get("/statistics/weekly")
                            .param("weekStart", weekStart.toString()))
                    .andExpect(status().isUnauthorized());
        }

        @Test
        @DisplayName("employee should be able to view weekly statistics")
        void employeeCanViewWeeklyStatistics() throws Exception {
            String token = getEmployeeToken();
            
            LocalDate today = LocalDate.now();
            LocalDate weekStart = today.minusDays(today.getDayOfWeek().getValue() - 1);

            mockMvc.perform(get("/statistics/weekly")
                            .param("weekStart", weekStart.toString())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk());
        }
    }

    @Nested
    @DisplayName("GET /export/timesheet")
    class ExportTimesheetTests {

        @Test
        @DisplayName("should export timesheet as PDF")
        void exportTimesheetAsPdf() throws Exception {
            String token = getAdminToken();
            
            LocalDate monthStart = LocalDate.now().withDayOfMonth(1);

            // The endpoint returns noContent for now (TODO in implementation)
            mockMvc.perform(get("/export/timesheet")
                            .param("month", monthStart.toString())
                            .param("format", "pdf")
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isNoContent());
        }

        @Test
        @DisplayName("should export timesheet as XLSX")
        void exportTimesheetAsXlsx() throws Exception {
            String token = getAdminToken();
            
            LocalDate monthStart = LocalDate.now().withDayOfMonth(1);

            mockMvc.perform(get("/export/timesheet")
                            .param("month", monthStart.toString())
                            .param("format", "xlsx")
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isNoContent());
        }

        @Test
        @DisplayName("should export timesheet for specific employee")
        void exportTimesheetForEmployee() throws Exception {
            String token = getAdminToken();
            
            LocalDate monthStart = LocalDate.now().withDayOfMonth(1);

            mockMvc.perform(get("/export/timesheet")
                            .param("month", monthStart.toString())
                            .param("format", "pdf")
                            .param("employeeId", regularEmployee.getId().toString())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isNoContent());
        }

        @Test
        @DisplayName("should return 401 without token")
        void exportTimesheetWithoutToken() throws Exception {
            LocalDate monthStart = LocalDate.now().withDayOfMonth(1);

            mockMvc.perform(get("/export/timesheet")
                            .param("month", monthStart.toString())
                            .param("format", "pdf"))
                    .andExpect(status().isUnauthorized());
        }
    }

    @Nested
    @DisplayName("GET /export/schedule")
    class ExportScheduleTests {

        @Test
        @DisplayName("should export schedule as PDF")
        void exportScheduleAsPdf() throws Exception {
            String token = getAdminToken();
            
            LocalDate today = LocalDate.now();
            LocalDate weekStart = today.minusDays(today.getDayOfWeek().getValue() - 1);

            // The endpoint returns noContent for now (TODO in implementation)
            mockMvc.perform(get("/export/schedule")
                            .param("weekStart", weekStart.toString())
                            .param("format", "pdf")
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isNoContent());
        }

        @Test
        @DisplayName("should export schedule as XLSX")
        void exportScheduleAsXlsx() throws Exception {
            String token = getAdminToken();
            
            LocalDate today = LocalDate.now();
            LocalDate weekStart = today.minusDays(today.getDayOfWeek().getValue() - 1);

            mockMvc.perform(get("/export/schedule")
                            .param("weekStart", weekStart.toString())
                            .param("format", "xlsx")
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isNoContent());
        }

        @Test
        @DisplayName("should return 401 without token")
        void exportScheduleWithoutToken() throws Exception {
            LocalDate weekStart = LocalDate.now();

            mockMvc.perform(get("/export/schedule")
                            .param("weekStart", weekStart.toString())
                            .param("format", "pdf"))
                    .andExpect(status().isUnauthorized());
        }
    }
}
