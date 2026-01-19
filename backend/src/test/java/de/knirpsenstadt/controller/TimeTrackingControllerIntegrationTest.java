package de.knirpsenstadt.controller;

import de.knirpsenstadt.AbstractIntegrationTest;
import de.knirpsenstadt.api.model.*;
import de.knirpsenstadt.model.TimeEntry;
import de.knirpsenstadt.model.TimeEntryType;
import de.knirpsenstadt.repository.TimeEntryRepository;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.MediaType;

import java.time.LocalDate;
import java.time.OffsetDateTime;
import java.time.ZoneOffset;
import java.util.ArrayList;
import java.util.List;

import static org.hamcrest.Matchers.*;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@DisplayName("TimeTrackingController Integration Tests")
class TimeTrackingControllerIntegrationTest extends AbstractIntegrationTest {

    @Autowired
    private TimeEntryRepository timeEntryRepository;

    private final List<Long> createdEntryIds = new ArrayList<>();

    @BeforeEach
    void setUpTimeData() {
        // Clean up any active entries for test users before each test
        timeEntryRepository.findActiveByEmployeeId(adminEmployee.getId())
                .ifPresent(entry -> {
                    entry.setClockOut(OffsetDateTime.now());
                    timeEntryRepository.save(entry);
                });
        timeEntryRepository.findActiveByEmployeeId(regularEmployee.getId())
                .ifPresent(entry -> {
                    entry.setClockOut(OffsetDateTime.now());
                    timeEntryRepository.save(entry);
                });
    }

    @AfterEach
    void cleanUp() {
        // Clean up time entries created during tests
        createdEntryIds.forEach(id -> {
            try {
                timeEntryRepository.deleteById(id);
            } catch (Exception ignored) {}
        });
        createdEntryIds.clear();
    }

    @Nested
    @DisplayName("POST /time-tracking/clock-in")
    class ClockInTests {

        @Test
        @DisplayName("should clock in employee")
        void clockIn() throws Exception {
            String token = getEmployeeToken();

            ClockInRequest request = new ClockInRequest();
            request.setNotes("Starting work");

            String response = mockMvc.perform(post("/time-tracking/clock-in")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.employeeId").value(regularEmployee.getId()))
                    .andExpect(jsonPath("$.clockIn").exists())
                    .andExpect(jsonPath("$.clockOut").doesNotExist())
                    .andExpect(jsonPath("$.notes").value("Starting work"))
                    .andReturn()
                    .getResponse()
                    .getContentAsString();

            de.knirpsenstadt.api.model.TimeEntry created = fromJson(response, de.knirpsenstadt.api.model.TimeEntry.class);
            createdEntryIds.add(created.getId());
        }

        @Test
        @DisplayName("should clock in without notes")
        void clockInWithoutNotes() throws Exception {
            String token = getEmployeeToken();

            String response = mockMvc.perform(post("/time-tracking/clock-in")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content("{}"))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.employeeId").value(regularEmployee.getId()))
                    .andExpect(jsonPath("$.clockIn").exists())
                    .andReturn()
                    .getResponse()
                    .getContentAsString();

            de.knirpsenstadt.api.model.TimeEntry created = fromJson(response, de.knirpsenstadt.api.model.TimeEntry.class);
            createdEntryIds.add(created.getId());
        }

        @Test
        @DisplayName("should return 401 without token")
        void clockInWithoutToken() throws Exception {
            mockMvc.perform(post("/time-tracking/clock-in")
                            .contentType(MediaType.APPLICATION_JSON)
                            .content("{}"))
                    .andExpect(status().isUnauthorized());
        }
    }

    @Nested
    @DisplayName("POST /time-tracking/clock-out")
    class ClockOutTests {

        @Test
        @DisplayName("should clock out employee")
        void clockOut() throws Exception {
            // First clock in
            TimeEntry entry = timeEntryRepository.save(
                    TimeEntry.builder()
                            .employee(regularEmployee)
                            .date(LocalDate.now())
                            .clockIn(OffsetDateTime.now().minusHours(8))
                            .entryType(TimeEntryType.WORK)
                            .build()
            );
            createdEntryIds.add(entry.getId());

            String token = getEmployeeToken();

            ClockOutRequest request = new ClockOutRequest();
            request.setBreakMinutes(30);
            request.setNotes("Finished work");

            mockMvc.perform(post("/time-tracking/clock-out")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.id").value(entry.getId()))
                    .andExpect(jsonPath("$.clockOut").exists())
                    .andExpect(jsonPath("$.breakMinutes").value(30))
                    .andExpect(jsonPath("$.notes").value("Finished work"));
        }

        @Test
        @DisplayName("should return 400 when not clocked in")
        void clockOutWhenNotClockedIn() throws Exception {
            String token = getEmployeeToken();

            ClockOutRequest request = new ClockOutRequest();
            request.setBreakMinutes(0);

            mockMvc.perform(post("/time-tracking/clock-out")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isBadRequest());
        }
    }

    @Nested
    @DisplayName("GET /time-tracking/current")
    class GetCurrentTimeEntryTests {

        @Test
        @DisplayName("should get current time entry when clocked in")
        void getCurrentTimeEntry() throws Exception {
            // Clock in
            TimeEntry entry = timeEntryRepository.save(
                    TimeEntry.builder()
                            .employee(regularEmployee)
                            .date(LocalDate.now())
                            .clockIn(OffsetDateTime.now().minusHours(2))
                            .entryType(TimeEntryType.WORK)
                            .build()
            );
            createdEntryIds.add(entry.getId());

            String token = getEmployeeToken();

            mockMvc.perform(get("/time-tracking/current")
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.id").value(entry.getId()))
                    .andExpect(jsonPath("$.clockIn").exists())
                    .andExpect(jsonPath("$.clockOut").doesNotExist());
        }

        @Test
        @DisplayName("should return 204 when not clocked in")
        void getCurrentTimeEntryWhenNotClockedIn() throws Exception {
            String token = getEmployeeToken();

            mockMvc.perform(get("/time-tracking/current")
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isNoContent());
        }
    }

    @Nested
    @DisplayName("GET /time-tracking/entries")
    class GetTimeEntriesTests {

        @Test
        @DisplayName("should get own time entries")
        void getOwnTimeEntries() throws Exception {
            // Create time entry
            TimeEntry entry = timeEntryRepository.save(
                    TimeEntry.builder()
                            .employee(regularEmployee)
                            .date(LocalDate.now())
                            .clockIn(OffsetDateTime.now().minusHours(8))
                            .clockOut(OffsetDateTime.now())
                            .breakMinutes(30)
                            .entryType(TimeEntryType.WORK)
                            .build()
            );
            createdEntryIds.add(entry.getId());

            String token = getEmployeeToken();

            mockMvc.perform(get("/time-tracking/entries")
                            .param("startDate", LocalDate.now().minusDays(1).toString())
                            .param("endDate", LocalDate.now().plusDays(1).toString())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$").isArray())
                    .andExpect(jsonPath("$[?(@.id == " + entry.getId() + ")]").exists());
        }

        @Test
        @DisplayName("should get time entries filtered by employeeId as admin")
        void getTimeEntriesByEmployeeId() throws Exception {
            // Create entries for different employees
            TimeEntry adminEntry = timeEntryRepository.save(
                    TimeEntry.builder()
                            .employee(adminEmployee)
                            .date(LocalDate.now())
                            .clockIn(OffsetDateTime.now().minusHours(8))
                            .clockOut(OffsetDateTime.now())
                            .entryType(TimeEntryType.WORK)
                            .build()
            );
            createdEntryIds.add(adminEntry.getId());

            TimeEntry employeeEntry = timeEntryRepository.save(
                    TimeEntry.builder()
                            .employee(regularEmployee)
                            .date(LocalDate.now())
                            .clockIn(OffsetDateTime.now().minusHours(7))
                            .clockOut(OffsetDateTime.now())
                            .entryType(TimeEntryType.WORK)
                            .build()
            );
            createdEntryIds.add(employeeEntry.getId());

            String token = getAdminToken();

            mockMvc.perform(get("/time-tracking/entries")
                            .param("startDate", LocalDate.now().minusDays(1).toString())
                            .param("endDate", LocalDate.now().plusDays(1).toString())
                            .param("employeeId", regularEmployee.getId().toString())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$").isArray())
                    .andExpect(jsonPath("$[?(@.employeeId == " + regularEmployee.getId() + ")]").exists());
        }

        @Test
        @DisplayName("should return 401 without token")
        void getTimeEntriesWithoutToken() throws Exception {
            mockMvc.perform(get("/time-tracking/entries")
                            .param("startDate", LocalDate.now().toString())
                            .param("endDate", LocalDate.now().plusDays(7).toString()))
                    .andExpect(status().isUnauthorized());
        }
    }

    @Nested
    @DisplayName("POST /time-tracking/entries")
    class CreateTimeEntryTests {

        @Test
        @DisplayName("should create time entry as admin")
        void createTimeEntry() throws Exception {
            String token = getAdminToken();

            OffsetDateTime clockIn = OffsetDateTime.now(ZoneOffset.UTC).minusHours(8);
            OffsetDateTime clockOut = OffsetDateTime.now(ZoneOffset.UTC);

            CreateTimeEntryRequest request = new CreateTimeEntryRequest();
            request.setEmployeeId(regularEmployee.getId());
            request.setDate(LocalDate.now());
            request.setClockIn(clockIn);
            request.setClockOut(clockOut);
            request.setBreakMinutes(30);
            request.setEntryType(de.knirpsenstadt.api.model.TimeEntryType.WORK);
            request.setNotes("Manual entry");
            request.setEditReason("Forgot to clock in");

            String response = mockMvc.perform(post("/time-tracking/entries")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isCreated())
                    .andExpect(jsonPath("$.employeeId").value(regularEmployee.getId()))
                    .andExpect(jsonPath("$.breakMinutes").value(30))
                    .andExpect(jsonPath("$.entryType").value("WORK"))
                    .andExpect(jsonPath("$.notes").value("Manual entry"))
                    .andReturn()
                    .getResponse()
                    .getContentAsString();

            de.knirpsenstadt.api.model.TimeEntry created = fromJson(response, de.knirpsenstadt.api.model.TimeEntry.class);
            createdEntryIds.add(created.getId());
        }

        @Test
        @DisplayName("should return 400 for missing required fields")
        void createTimeEntryMissingFields() throws Exception {
            String token = getAdminToken();

            CreateTimeEntryRequest request = new CreateTimeEntryRequest();
            // Missing required fields

            mockMvc.perform(post("/time-tracking/entries")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isBadRequest());
        }
    }

    @Nested
    @DisplayName("PUT /time-tracking/entries/{id}")
    class UpdateTimeEntryTests {

        @Test
        @DisplayName("should update time entry as admin")
        void updateTimeEntry() throws Exception {
            // Create entry to update
            TimeEntry entry = timeEntryRepository.save(
                    TimeEntry.builder()
                            .employee(regularEmployee)
                            .date(LocalDate.now().minusDays(1))
                            .clockIn(OffsetDateTime.now().minusDays(1).minusHours(8))
                            .clockOut(OffsetDateTime.now().minusDays(1))
                            .breakMinutes(30)
                            .entryType(TimeEntryType.WORK)
                            .build()
            );
            createdEntryIds.add(entry.getId());

            String token = getAdminToken();

            UpdateTimeEntryRequest request = new UpdateTimeEntryRequest();
            request.setBreakMinutes(45);
            request.setNotes("Updated break time");
            request.setEditReason("Correcting break duration");

            mockMvc.perform(put("/time-tracking/entries/{id}", entry.getId())
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.id").value(entry.getId()))
                    .andExpect(jsonPath("$.breakMinutes").value(45))
                    .andExpect(jsonPath("$.notes").value("Updated break time"));
        }

        @Test
        @DisplayName("should return 404 for non-existent entry")
        void updateNonExistentEntry() throws Exception {
            String token = getAdminToken();

            UpdateTimeEntryRequest request = new UpdateTimeEntryRequest();
            request.setBreakMinutes(30);

            mockMvc.perform(put("/time-tracking/entries/{id}", 99999)
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isNotFound());
        }
    }

    @Nested
    @DisplayName("DELETE /time-tracking/entries/{id}")
    class DeleteTimeEntryTests {

        @Test
        @DisplayName("should delete time entry as admin")
        void deleteTimeEntry() throws Exception {
            // Create entry to delete
            TimeEntry entry = timeEntryRepository.save(
                    TimeEntry.builder()
                            .employee(regularEmployee)
                            .date(LocalDate.now().minusDays(2))
                            .clockIn(OffsetDateTime.now().minusDays(2).minusHours(8))
                            .clockOut(OffsetDateTime.now().minusDays(2))
                            .entryType(TimeEntryType.WORK)
                            .build()
            );

            String token = getAdminToken();

            mockMvc.perform(delete("/time-tracking/entries/{id}", entry.getId())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isNoContent());

            // Verify deleted
            assert !timeEntryRepository.existsById(entry.getId());
        }

        @Test
        @DisplayName("should return 404 for non-existent entry")
        void deleteNonExistentEntry() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(delete("/time-tracking/entries/{id}", 99999)
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isNotFound());
        }
    }

    @Nested
    @DisplayName("GET /time-tracking/comparison")
    class GetTimeScheduleComparisonTests {

        @Test
        @DisplayName("should get time schedule comparison")
        void getTimeScheduleComparison() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(get("/time-tracking/comparison")
                            .param("startDate", LocalDate.now().minusDays(7).toString())
                            .param("endDate", LocalDate.now().toString())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk());
        }

        @Test
        @DisplayName("should get comparison filtered by employeeId")
        void getTimeScheduleComparisonByEmployee() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(get("/time-tracking/comparison")
                            .param("startDate", LocalDate.now().minusDays(7).toString())
                            .param("endDate", LocalDate.now().toString())
                            .param("employeeId", regularEmployee.getId().toString())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk());
        }

        @Test
        @DisplayName("should return 401 without token")
        void getComparisonWithoutToken() throws Exception {
            mockMvc.perform(get("/time-tracking/comparison")
                            .param("startDate", LocalDate.now().minusDays(7).toString())
                            .param("endDate", LocalDate.now().toString()))
                    .andExpect(status().isUnauthorized());
        }
    }
}
