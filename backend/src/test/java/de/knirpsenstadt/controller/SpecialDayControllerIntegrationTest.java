package de.knirpsenstadt.controller;

import de.knirpsenstadt.AbstractIntegrationTest;
import de.knirpsenstadt.api.model.*;
import de.knirpsenstadt.model.SpecialDay;
import de.knirpsenstadt.model.SpecialDayType;
import de.knirpsenstadt.repository.SpecialDayRepository;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.MediaType;

import java.time.LocalDate;
import java.util.ArrayList;
import java.util.List;

import static org.hamcrest.Matchers.*;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@DisplayName("SpecialDayController Integration Tests")
class SpecialDayControllerIntegrationTest extends AbstractIntegrationTest {

    @Autowired
    private SpecialDayRepository specialDayRepository;

    private final List<Long> createdSpecialDayIds = new ArrayList<>();

    @BeforeEach
    void setUpSpecialDayData() {
        // Clean up any existing test special days
    }

    @AfterEach
    void cleanUp() {
        // Clean up special days created during tests
        createdSpecialDayIds.forEach(id -> {
            try {
                specialDayRepository.deleteById(id);
            } catch (Exception ignored) {}
        });
        createdSpecialDayIds.clear();
    }

    @Nested
    @DisplayName("GET /special-days")
    class GetSpecialDaysTests {

        @Test
        @DisplayName("should get special days for year")
        void getSpecialDaysForYear() throws Exception {
            // Create a special day
            SpecialDay specialDay = specialDayRepository.save(
                    SpecialDay.builder()
                            .date(LocalDate.of(LocalDate.now().getYear(), 6, 15))
                            .name("Team Day")
                            .dayType(SpecialDayType.TEAM_DAY)
                            .affectsAll(true)
                            .notes("Annual team building")
                            .build()
            );
            createdSpecialDayIds.add(specialDay.getId());

            String token = getAdminToken();

            mockMvc.perform(get("/special-days")
                            .param("year", String.valueOf(LocalDate.now().getYear()))
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$").isArray())
                    .andExpect(jsonPath("$[?(@.name == 'Team Day')]").exists());
        }

        @Test
        @DisplayName("should get special days with holidays")
        void getSpecialDaysWithHolidays() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(get("/special-days")
                            .param("year", String.valueOf(LocalDate.now().getYear()))
                            .param("includeHolidays", "true")
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$").isArray());
        }

        @Test
        @DisplayName("should get special days without holidays")
        void getSpecialDaysWithoutHolidays() throws Exception {
            // Create a closure special day
            SpecialDay specialDay = specialDayRepository.save(
                    SpecialDay.builder()
                            .date(LocalDate.of(LocalDate.now().getYear(), 8, 1))
                            .name("Kita Closure")
                            .dayType(SpecialDayType.CLOSURE)
                            .affectsAll(true)
                            .build()
            );
            createdSpecialDayIds.add(specialDay.getId());

            String token = getAdminToken();

            mockMvc.perform(get("/special-days")
                            .param("year", String.valueOf(LocalDate.now().getYear()))
                            .param("includeHolidays", "false")
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$").isArray());
        }

        @Test
        @DisplayName("should return 401 without token")
        void getSpecialDaysWithoutToken() throws Exception {
            mockMvc.perform(get("/special-days")
                            .param("year", String.valueOf(LocalDate.now().getYear())))
                    .andExpect(status().isUnauthorized());
        }
    }

    @Nested
    @DisplayName("GET /special-days/holidays/{year}")
    class GetHolidaysTests {

        @Test
        @DisplayName("should get holidays for year")
        void getHolidaysForYear() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(get("/special-days/holidays/{year}", LocalDate.now().getYear())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$").isArray());
        }

        @Test
        @DisplayName("should get holidays for specific year")
        void getHolidaysFor2024() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(get("/special-days/holidays/{year}", 2024)
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$").isArray());
        }

        @Test
        @DisplayName("should return 401 without token")
        void getHolidaysWithoutToken() throws Exception {
            mockMvc.perform(get("/special-days/holidays/{year}", LocalDate.now().getYear()))
                    .andExpect(status().isUnauthorized());
        }
    }

    @Nested
    @DisplayName("POST /special-days")
    class CreateSpecialDayTests {

        @Test
        @DisplayName("should create special day as admin")
        void createSpecialDay() throws Exception {
            String token = getAdminToken();

            CreateSpecialDayRequest request = new CreateSpecialDayRequest();
            request.setDate(LocalDate.of(LocalDate.now().getYear(), 12, 23));
            request.setName("Christmas Closure");
            request.setDayType(de.knirpsenstadt.api.model.SpecialDayType.CLOSURE);
            request.setAffectsAll(true);
            request.setNotes("Kita closed for Christmas");

            String response = mockMvc.perform(post("/special-days")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isCreated())
                    .andExpect(jsonPath("$.name").value("Christmas Closure"))
                    .andExpect(jsonPath("$.dayType").value("CLOSURE"))
                    .andExpect(jsonPath("$.affectsAll").value(true))
                    .andExpect(jsonPath("$.notes").value("Kita closed for Christmas"))
                    .andReturn()
                    .getResponse()
                    .getContentAsString();

            de.knirpsenstadt.api.model.SpecialDay created = fromJson(response, de.knirpsenstadt.api.model.SpecialDay.class);
            createdSpecialDayIds.add(created.getId());
        }

        @Test
        @DisplayName("should create team day")
        void createTeamDay() throws Exception {
            String token = getAdminToken();

            CreateSpecialDayRequest request = new CreateSpecialDayRequest();
            request.setDate(LocalDate.of(LocalDate.now().getYear(), 9, 15));
            request.setName("Team Building Day");
            request.setDayType(de.knirpsenstadt.api.model.SpecialDayType.TEAM_DAY);
            request.setAffectsAll(true);

            String response = mockMvc.perform(post("/special-days")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isCreated())
                    .andExpect(jsonPath("$.name").value("Team Building Day"))
                    .andExpect(jsonPath("$.dayType").value("TEAM_DAY"))
                    .andReturn()
                    .getResponse()
                    .getContentAsString();

            de.knirpsenstadt.api.model.SpecialDay created = fromJson(response, de.knirpsenstadt.api.model.SpecialDay.class);
            createdSpecialDayIds.add(created.getId());
        }

        @Test
        @DisplayName("should create event")
        void createEvent() throws Exception {
            String token = getAdminToken();

            CreateSpecialDayRequest request = new CreateSpecialDayRequest();
            request.setDate(LocalDate.of(LocalDate.now().getYear(), 7, 4));
            request.setName("Summer Festival");
            request.setDayType(de.knirpsenstadt.api.model.SpecialDayType.EVENT);
            request.setAffectsAll(false);
            request.setNotes("Annual summer festival");

            String response = mockMvc.perform(post("/special-days")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isCreated())
                    .andExpect(jsonPath("$.name").value("Summer Festival"))
                    .andExpect(jsonPath("$.dayType").value("EVENT"))
                    .andExpect(jsonPath("$.affectsAll").value(false))
                    .andReturn()
                    .getResponse()
                    .getContentAsString();

            de.knirpsenstadt.api.model.SpecialDay created = fromJson(response, de.knirpsenstadt.api.model.SpecialDay.class);
            createdSpecialDayIds.add(created.getId());
        }

        @Test
        @DisplayName("should return 400 for missing required fields")
        void createSpecialDayMissingFields() throws Exception {
            String token = getAdminToken();

            CreateSpecialDayRequest request = new CreateSpecialDayRequest();
            // Missing required fields: date, name, dayType

            mockMvc.perform(post("/special-days")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isBadRequest());
        }
    }

    @Nested
    @DisplayName("PUT /special-days/{id}")
    class UpdateSpecialDayTests {

        @Test
        @DisplayName("should update special day as admin")
        void updateSpecialDay() throws Exception {
            // Create special day to update
            SpecialDay specialDay = specialDayRepository.save(
                    SpecialDay.builder()
                            .date(LocalDate.of(LocalDate.now().getYear(), 10, 10))
                            .name("Original Name")
                            .dayType(SpecialDayType.CLOSURE)
                            .affectsAll(true)
                            .build()
            );
            createdSpecialDayIds.add(specialDay.getId());

            String token = getAdminToken();

            CreateSpecialDayRequest request = new CreateSpecialDayRequest();
            request.setDate(LocalDate.of(LocalDate.now().getYear(), 10, 11));
            request.setName("Updated Name");
            request.setDayType(de.knirpsenstadt.api.model.SpecialDayType.TEAM_DAY);
            request.setAffectsAll(false);
            request.setNotes("Updated notes");

            mockMvc.perform(put("/special-days/{id}", specialDay.getId())
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.id").value(specialDay.getId()))
                    .andExpect(jsonPath("$.name").value("Updated Name"))
                    .andExpect(jsonPath("$.dayType").value("TEAM_DAY"))
                    .andExpect(jsonPath("$.affectsAll").value(false))
                    .andExpect(jsonPath("$.notes").value("Updated notes"));
        }

        @Test
        @DisplayName("should return 404 for non-existent special day")
        void updateNonExistentSpecialDay() throws Exception {
            String token = getAdminToken();

            CreateSpecialDayRequest request = new CreateSpecialDayRequest();
            request.setDate(LocalDate.now());
            request.setName("Test");
            request.setDayType(de.knirpsenstadt.api.model.SpecialDayType.CLOSURE);

            mockMvc.perform(put("/special-days/{id}", 99999)
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isNotFound());
        }
    }

    @Nested
    @DisplayName("DELETE /special-days/{id}")
    class DeleteSpecialDayTests {

        @Test
        @DisplayName("should delete special day as admin")
        void deleteSpecialDay() throws Exception {
            // Create special day to delete
            SpecialDay specialDay = specialDayRepository.save(
                    SpecialDay.builder()
                            .date(LocalDate.of(LocalDate.now().getYear(), 11, 11))
                            .name("Delete Me")
                            .dayType(SpecialDayType.EVENT)
                            .affectsAll(true)
                            .build()
            );

            String token = getAdminToken();

            mockMvc.perform(delete("/special-days/{id}", specialDay.getId())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isNoContent());

            // Verify deleted
            assert !specialDayRepository.existsById(specialDay.getId());
        }

        @Test
        @DisplayName("should return 404 for non-existent special day")
        void deleteNonExistentSpecialDay() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(delete("/special-days/{id}", 99999)
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isNotFound());
        }
    }
}
