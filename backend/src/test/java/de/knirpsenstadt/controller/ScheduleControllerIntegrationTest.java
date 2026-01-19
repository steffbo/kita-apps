package de.knirpsenstadt.controller;

import de.knirpsenstadt.AbstractIntegrationTest;
import de.knirpsenstadt.api.model.*;
import de.knirpsenstadt.model.Group;
import de.knirpsenstadt.model.ScheduleEntry;
import de.knirpsenstadt.model.ScheduleEntryType;
import de.knirpsenstadt.repository.GroupRepository;
import de.knirpsenstadt.repository.ScheduleEntryRepository;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.MediaType;

import java.time.LocalDate;
import java.time.LocalTime;
import java.util.ArrayList;
import java.util.List;

import static org.hamcrest.Matchers.*;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@DisplayName("ScheduleController Integration Tests")
class ScheduleControllerIntegrationTest extends AbstractIntegrationTest {

    @Autowired
    private ScheduleEntryRepository scheduleEntryRepository;

    @Autowired
    private GroupRepository groupRepository;

    private final List<Long> createdEntryIds = new ArrayList<>();
    private Group testGroup;

    @BeforeEach
    void setUpScheduleData() {
        // Create a test group if needed
        testGroup = groupRepository.findAll().stream()
                .filter(g -> g.getName().equals("ScheduleTestGroup"))
                .findFirst()
                .orElseGet(() -> {
                    Group group = Group.builder()
                            .name("ScheduleTestGroup")
                            .description("Test group for schedule tests")
                            .color("#FF5733")
                            .build();
                    return groupRepository.save(group);
                });
    }

    @AfterEach
    void cleanUp() {
        // Clean up schedule entries created during tests
        createdEntryIds.forEach(id -> {
            try {
                scheduleEntryRepository.deleteById(id);
            } catch (Exception ignored) {}
        });
        createdEntryIds.clear();
    }

    @Nested
    @DisplayName("GET /schedule")
    class GetScheduleTests {

        @Test
        @DisplayName("should get schedule entries for date range")
        void getScheduleForDateRange() throws Exception {
            // Create a schedule entry
            ScheduleEntry entry = scheduleEntryRepository.save(
                    ScheduleEntry.builder()
                            .employee(adminEmployee)
                            .date(LocalDate.now())
                            .startTime(LocalTime.of(8, 0))
                            .endTime(LocalTime.of(16, 0))
                            .breakMinutes(30)
                            .entryType(ScheduleEntryType.WORK)
                            .group(testGroup)
                            .build()
            );
            createdEntryIds.add(entry.getId());

            String token = getAdminToken();

            mockMvc.perform(get("/schedule")
                            .param("startDate", LocalDate.now().minusDays(1).toString())
                            .param("endDate", LocalDate.now().plusDays(1).toString())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$").isArray())
                    .andExpect(jsonPath("$[?(@.id == " + entry.getId() + ")]").exists());
        }

        @Test
        @DisplayName("should filter by employeeId")
        void getScheduleFilteredByEmployee() throws Exception {
            // Create schedule entries for different employees
            ScheduleEntry adminEntry = scheduleEntryRepository.save(
                    ScheduleEntry.builder()
                            .employee(adminEmployee)
                            .date(LocalDate.now())
                            .startTime(LocalTime.of(8, 0))
                            .endTime(LocalTime.of(16, 0))
                            .entryType(ScheduleEntryType.WORK)
                            .build()
            );
            createdEntryIds.add(adminEntry.getId());

            ScheduleEntry regularEntry = scheduleEntryRepository.save(
                    ScheduleEntry.builder()
                            .employee(regularEmployee)
                            .date(LocalDate.now())
                            .startTime(LocalTime.of(9, 0))
                            .endTime(LocalTime.of(17, 0))
                            .entryType(ScheduleEntryType.WORK)
                            .build()
            );
            createdEntryIds.add(regularEntry.getId());

            String token = getAdminToken();

            mockMvc.perform(get("/schedule")
                            .param("startDate", LocalDate.now().minusDays(1).toString())
                            .param("endDate", LocalDate.now().plusDays(1).toString())
                            .param("employeeId", adminEmployee.getId().toString())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$").isArray())
                    .andExpect(jsonPath("$[?(@.employeeId == " + adminEmployee.getId() + ")]").exists())
                    .andExpect(jsonPath("$[?(@.employeeId == " + regularEmployee.getId() + ")]").doesNotExist());
        }

        @Test
        @DisplayName("should filter by groupId")
        void getScheduleFilteredByGroup() throws Exception {
            // Create schedule entry with group
            ScheduleEntry entryWithGroup = scheduleEntryRepository.save(
                    ScheduleEntry.builder()
                            .employee(adminEmployee)
                            .date(LocalDate.now())
                            .startTime(LocalTime.of(8, 0))
                            .endTime(LocalTime.of(16, 0))
                            .entryType(ScheduleEntryType.WORK)
                            .group(testGroup)
                            .build()
            );
            createdEntryIds.add(entryWithGroup.getId());

            String token = getAdminToken();

            mockMvc.perform(get("/schedule")
                            .param("startDate", LocalDate.now().minusDays(1).toString())
                            .param("endDate", LocalDate.now().plusDays(1).toString())
                            .param("groupId", testGroup.getId().toString())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$").isArray())
                    .andExpect(jsonPath("$[?(@.groupId == " + testGroup.getId() + ")]").exists());
        }

        @Test
        @DisplayName("should return 401 without token")
        void getScheduleWithoutToken() throws Exception {
            mockMvc.perform(get("/schedule")
                            .param("startDate", LocalDate.now().toString())
                            .param("endDate", LocalDate.now().plusDays(7).toString()))
                    .andExpect(status().isUnauthorized());
        }
    }

    @Nested
    @DisplayName("GET /schedule/week")
    class GetWeekScheduleTests {

        @Test
        @DisplayName("should get week schedule")
        void getWeekSchedule() throws Exception {
            // Get the Monday of current week
            LocalDate today = LocalDate.now();
            LocalDate weekStart = today.minusDays(today.getDayOfWeek().getValue() - 1);

            String token = getAdminToken();

            mockMvc.perform(get("/schedule/week")
                            .param("weekStart", weekStart.toString())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.weekStart").value(weekStart.toString()))
                    .andExpect(jsonPath("$.weekEnd").value(weekStart.plusDays(6).toString()))
                    .andExpect(jsonPath("$.days").isArray());
        }

        @Test
        @DisplayName("should return 401 without token")
        void getWeekScheduleWithoutToken() throws Exception {
            LocalDate weekStart = LocalDate.now();

            mockMvc.perform(get("/schedule/week")
                            .param("weekStart", weekStart.toString()))
                    .andExpect(status().isUnauthorized());
        }
    }

    @Nested
    @DisplayName("POST /schedule")
    class CreateScheduleEntryTests {

        @Test
        @DisplayName("should create schedule entry")
        void createScheduleEntry() throws Exception {
            String token = getAdminToken();

            CreateScheduleEntryRequest request = new CreateScheduleEntryRequest();
            request.setEmployeeId(regularEmployee.getId());
            request.setDate(LocalDate.now().plusDays(1));
            request.setStartTime("08:00");
            request.setEndTime("16:00");
            request.setBreakMinutes(30);
            request.setGroupId(testGroup.getId());
            request.setEntryType(de.knirpsenstadt.api.model.ScheduleEntryType.WORK);
            request.setNotes("Test schedule entry");

            String response = mockMvc.perform(post("/schedule")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isCreated())
                    .andExpect(jsonPath("$.employeeId").value(regularEmployee.getId()))
                    .andExpect(jsonPath("$.date").value(LocalDate.now().plusDays(1).toString()))
                    .andExpect(jsonPath("$.startTime").value("08:00:00"))
                    .andExpect(jsonPath("$.endTime").value("16:00:00"))
                    .andExpect(jsonPath("$.breakMinutes").value(30))
                    .andExpect(jsonPath("$.entryType").value("WORK"))
                    .andReturn()
                    .getResponse()
                    .getContentAsString();

            // Track for cleanup
            de.knirpsenstadt.api.model.ScheduleEntry created = fromJson(response, de.knirpsenstadt.api.model.ScheduleEntry.class);
            createdEntryIds.add(created.getId());
        }

        @Test
        @DisplayName("should create vacation schedule entry")
        void createVacationEntry() throws Exception {
            String token = getAdminToken();

            CreateScheduleEntryRequest request = new CreateScheduleEntryRequest();
            request.setEmployeeId(regularEmployee.getId());
            request.setDate(LocalDate.now().plusDays(5));
            request.setEntryType(de.knirpsenstadt.api.model.ScheduleEntryType.VACATION);
            request.setNotes("Vacation day");

            String response = mockMvc.perform(post("/schedule")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isCreated())
                    .andExpect(jsonPath("$.entryType").value("VACATION"))
                    .andReturn()
                    .getResponse()
                    .getContentAsString();

            de.knirpsenstadt.api.model.ScheduleEntry created = fromJson(response, de.knirpsenstadt.api.model.ScheduleEntry.class);
            createdEntryIds.add(created.getId());
        }

        @Test
        @DisplayName("should return 400 for missing required fields")
        void createScheduleEntryMissingFields() throws Exception {
            String token = getAdminToken();

            CreateScheduleEntryRequest request = new CreateScheduleEntryRequest();
            // Missing required fields: employeeId, date, entryType

            mockMvc.perform(post("/schedule")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isBadRequest());
        }
    }

    @Nested
    @DisplayName("PUT /schedule/{id}")
    class UpdateScheduleEntryTests {

        @Test
        @DisplayName("should update schedule entry")
        void updateScheduleEntry() throws Exception {
            // Create entry to update
            ScheduleEntry entry = scheduleEntryRepository.save(
                    ScheduleEntry.builder()
                            .employee(adminEmployee)
                            .date(LocalDate.now().plusDays(2))
                            .startTime(LocalTime.of(8, 0))
                            .endTime(LocalTime.of(16, 0))
                            .breakMinutes(30)
                            .entryType(ScheduleEntryType.WORK)
                            .build()
            );
            createdEntryIds.add(entry.getId());

            String token = getAdminToken();

            UpdateScheduleEntryRequest request = new UpdateScheduleEntryRequest();
            request.setStartTime("09:00");
            request.setEndTime("17:00");
            request.setBreakMinutes(45);
            request.setNotes("Updated entry");

            mockMvc.perform(put("/schedule/{id}", entry.getId())
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.id").value(entry.getId()))
                    .andExpect(jsonPath("$.startTime").value("09:00:00"))
                    .andExpect(jsonPath("$.endTime").value("17:00:00"))
                    .andExpect(jsonPath("$.breakMinutes").value(45))
                    .andExpect(jsonPath("$.notes").value("Updated entry"));
        }

        @Test
        @DisplayName("should return 404 for non-existent entry")
        void updateNonExistentEntry() throws Exception {
            String token = getAdminToken();

            UpdateScheduleEntryRequest request = new UpdateScheduleEntryRequest();
            request.setStartTime("09:00");

            mockMvc.perform(put("/schedule/{id}", 99999)
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isNotFound());
        }
    }

    @Nested
    @DisplayName("DELETE /schedule/{id}")
    class DeleteScheduleEntryTests {

        @Test
        @DisplayName("should delete schedule entry")
        void deleteScheduleEntry() throws Exception {
            // Create entry to delete
            ScheduleEntry entry = scheduleEntryRepository.save(
                    ScheduleEntry.builder()
                            .employee(adminEmployee)
                            .date(LocalDate.now().plusDays(3))
                            .startTime(LocalTime.of(8, 0))
                            .endTime(LocalTime.of(16, 0))
                            .entryType(ScheduleEntryType.WORK)
                            .build()
            );

            String token = getAdminToken();

            mockMvc.perform(delete("/schedule/{id}", entry.getId())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isNoContent());

            // Verify deleted
            assert !scheduleEntryRepository.existsById(entry.getId());
        }

        @Test
        @DisplayName("should return 404 for non-existent entry")
        void deleteNonExistentEntry() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(delete("/schedule/{id}", 99999)
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isNotFound());
        }
    }

    @Nested
    @DisplayName("POST /schedule/bulk")
    class BulkCreateScheduleEntriesTests {

        @Test
        @DisplayName("should create multiple schedule entries")
        void bulkCreateScheduleEntries() throws Exception {
            String token = getAdminToken();

            List<CreateScheduleEntryRequest> requests = new ArrayList<>();

            CreateScheduleEntryRequest request1 = new CreateScheduleEntryRequest();
            request1.setEmployeeId(regularEmployee.getId());
            request1.setDate(LocalDate.now().plusDays(10));
            request1.setStartTime("08:00");
            request1.setEndTime("16:00");
            request1.setEntryType(de.knirpsenstadt.api.model.ScheduleEntryType.WORK);
            requests.add(request1);

            CreateScheduleEntryRequest request2 = new CreateScheduleEntryRequest();
            request2.setEmployeeId(regularEmployee.getId());
            request2.setDate(LocalDate.now().plusDays(11));
            request2.setStartTime("08:00");
            request2.setEndTime("16:00");
            request2.setEntryType(de.knirpsenstadt.api.model.ScheduleEntryType.WORK);
            requests.add(request2);

            String response = mockMvc.perform(post("/schedule/bulk")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(requests)))
                    .andExpect(status().isCreated())
                    .andExpect(jsonPath("$").isArray())
                    .andExpect(jsonPath("$", hasSize(2)))
                    .andReturn()
                    .getResponse()
                    .getContentAsString();

            // Track created entries for cleanup
            de.knirpsenstadt.api.model.ScheduleEntry[] created = objectMapper.readValue(response, 
                    de.knirpsenstadt.api.model.ScheduleEntry[].class);
            for (de.knirpsenstadt.api.model.ScheduleEntry entry : created) {
                createdEntryIds.add(entry.getId());
            }
        }

        @Test
        @DisplayName("should return empty array for empty request")
        void bulkCreateEmptyList() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(post("/schedule/bulk")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content("[]"))
                    .andExpect(status().isCreated())
                    .andExpect(jsonPath("$").isArray())
                    .andExpect(jsonPath("$", hasSize(0)));
        }
    }
}
