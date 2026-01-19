package de.knirpsenstadt.controller;

import de.knirpsenstadt.AbstractIntegrationTest;
import de.knirpsenstadt.api.model.*;
import de.knirpsenstadt.model.Group;
import de.knirpsenstadt.repository.GroupAssignmentRepository;
import de.knirpsenstadt.repository.GroupRepository;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.MediaType;

import java.util.ArrayList;
import java.util.List;

import static org.hamcrest.Matchers.*;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@DisplayName("GroupController Integration Tests")
class GroupControllerIntegrationTest extends AbstractIntegrationTest {

    @Autowired
    private GroupRepository groupRepository;

    @Autowired
    private GroupAssignmentRepository groupAssignmentRepository;

    private final List<Long> createdGroupIds = new ArrayList<>();
    private Group testGroup;

    @BeforeEach
    void setUpGroups() {
        // Create a test group
        testGroup = groupRepository.findAll().stream()
                .filter(g -> g.getName().equals("TestGroup"))
                .findFirst()
                .orElseGet(() -> {
                    Group group = Group.builder()
                            .name("TestGroup")
                            .description("Test group for integration tests")
                            .color("#FF5733")
                            .build();
                    return groupRepository.save(group);
                });
    }

    @AfterEach
    void cleanUp() {
        // Clean up groups created during tests
        createdGroupIds.forEach(id -> {
            try {
                groupAssignmentRepository.deleteByGroupId(id);
                groupRepository.deleteById(id);
            } catch (Exception ignored) {}
        });
        createdGroupIds.clear();
    }

    @Nested
    @DisplayName("GET /groups")
    class ListGroupsTests {

        @Test
        @DisplayName("should list all groups")
        void listGroups() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(get("/groups")
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$").isArray())
                    .andExpect(jsonPath("$[*].name", hasItem("TestGroup")));
        }

        @Test
        @DisplayName("should return 401 without token")
        void listGroupsWithoutToken() throws Exception {
            mockMvc.perform(get("/groups"))
                    .andExpect(status().isUnauthorized());
        }
    }

    @Nested
    @DisplayName("GET /groups/{id}")
    class GetGroupTests {

        @Test
        @DisplayName("should get group by id with members")
        void getGroupById() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(get("/groups/{id}", testGroup.getId())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.id").value(testGroup.getId()))
                    .andExpect(jsonPath("$.name").value("TestGroup"))
                    .andExpect(jsonPath("$.description").value("Test group for integration tests"))
                    .andExpect(jsonPath("$.color").value("#FF5733"))
                    .andExpect(jsonPath("$.members").isArray());
        }

        @Test
        @DisplayName("should return 404 for non-existent group")
        void getGroupNotFound() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(get("/groups/{id}", 99999)
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isNotFound());
        }
    }

    @Nested
    @DisplayName("POST /groups")
    class CreateGroupTests {

        @Test
        @DisplayName("should create group as admin")
        void createGroupAsAdmin() throws Exception {
            String token = getAdminToken();

            CreateGroupRequest request = new CreateGroupRequest();
            request.setName("New Group");
            request.setDescription("A new test group");
            request.setColor("#00FF00");

            String response = mockMvc.perform(post("/groups")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isCreated())
                    .andExpect(jsonPath("$.name").value("New Group"))
                    .andExpect(jsonPath("$.description").value("A new test group"))
                    .andExpect(jsonPath("$.color").value("#00FF00"))
                    .andReturn()
                    .getResponse()
                    .getContentAsString();

            // Track for cleanup
            de.knirpsenstadt.api.model.Group created = fromJson(response, de.knirpsenstadt.api.model.Group.class);
            createdGroupIds.add(created.getId());
        }

        @Test
        @DisplayName("should return 400 for invalid request")
        void createGroupInvalidRequest() throws Exception {
            String token = getAdminToken();

            CreateGroupRequest request = new CreateGroupRequest();
            // Missing required name field

            mockMvc.perform(post("/groups")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isBadRequest());
        }
    }

    @Nested
    @DisplayName("PUT /groups/{id}")
    class UpdateGroupTests {

        @Test
        @DisplayName("should update group as admin")
        void updateGroupAsAdmin() throws Exception {
            // Create group to update
            Group toUpdate = groupRepository.save(
                    Group.builder()
                            .name("Update Me")
                            .description("Original description")
                            .color("#AABBCC")
                            .build()
            );
            createdGroupIds.add(toUpdate.getId());

            String token = getAdminToken();

            CreateGroupRequest request = new CreateGroupRequest();
            request.setName("Updated Name");
            request.setDescription("Updated description");
            request.setColor("#DDEEFF");

            mockMvc.perform(put("/groups/{id}", toUpdate.getId())
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.name").value("Updated Name"))
                    .andExpect(jsonPath("$.description").value("Updated description"))
                    .andExpect(jsonPath("$.color").value("#DDEEFF"));
        }

        @Test
        @DisplayName("should return 404 for non-existent group")
        void updateGroupNotFound() throws Exception {
            String token = getAdminToken();

            CreateGroupRequest request = new CreateGroupRequest();
            request.setName("Test");

            mockMvc.perform(put("/groups/{id}", 99999)
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isNotFound());
        }
    }

    @Nested
    @DisplayName("DELETE /groups/{id}")
    class DeleteGroupTests {

        @Test
        @DisplayName("should delete group as admin")
        void deleteGroupAsAdmin() throws Exception {
            // Create group to delete
            Group toDelete = groupRepository.save(
                    Group.builder()
                            .name("Delete Me")
                            .description("Will be deleted")
                            .color("#123456")
                            .build()
            );

            String token = getAdminToken();

            mockMvc.perform(delete("/groups/{id}", toDelete.getId())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isNoContent());

            // Verify group is deleted
            assert !groupRepository.existsById(toDelete.getId());
        }

        @Test
        @DisplayName("should return 404 for non-existent group")
        void deleteGroupNotFound() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(delete("/groups/{id}", 99999)
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isNotFound());
        }
    }

    @Nested
    @DisplayName("GET /groups/{id}/assignments")
    class GetGroupAssignmentsTests {

        @Test
        @DisplayName("should get group assignments")
        void getGroupAssignments() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(get("/groups/{id}/assignments", testGroup.getId())
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$").isArray());
        }

        @Test
        @DisplayName("should return 404 for non-existent group")
        void getAssignmentsGroupNotFound() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(get("/groups/{id}/assignments", 99999)
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isNotFound());
        }
    }

    @Nested
    @DisplayName("PUT /groups/{id}/assignments")
    class UpdateGroupAssignmentsTests {

        @Test
        @DisplayName("should update group assignments")
        void updateGroupAssignments() throws Exception {
            String token = getAdminToken();

            List<GroupAssignmentRequest> requests = new ArrayList<>();
            GroupAssignmentRequest assignment = new GroupAssignmentRequest();
            assignment.setEmployeeId(regularEmployee.getId());
            assignment.setAssignmentType(AssignmentType.PERMANENT);
            requests.add(assignment);

            mockMvc.perform(put("/groups/{id}/assignments", testGroup.getId())
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(requests)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$").isArray())
                    .andExpect(jsonPath("$", hasSize(1)))
                    .andExpect(jsonPath("$[0].employeeId").value(regularEmployee.getId()))
                    .andExpect(jsonPath("$[0].assignmentType").value("PERMANENT"));
        }

        @Test
        @DisplayName("should clear assignments when empty list provided")
        void clearGroupAssignments() throws Exception {
            String token = getAdminToken();

            // First add an assignment
            List<GroupAssignmentRequest> requests = new ArrayList<>();
            GroupAssignmentRequest assignment = new GroupAssignmentRequest();
            assignment.setEmployeeId(regularEmployee.getId());
            assignment.setAssignmentType(AssignmentType.PERMANENT);
            requests.add(assignment);

            mockMvc.perform(put("/groups/{id}/assignments", testGroup.getId())
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(requests)))
                    .andExpect(status().isOk());

            // Then clear assignments
            mockMvc.perform(put("/groups/{id}/assignments", testGroup.getId())
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content("[]"))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$").isArray())
                    .andExpect(jsonPath("$", hasSize(0)));
        }

        @Test
        @DisplayName("should return 404 for non-existent group")
        void updateAssignmentsGroupNotFound() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(put("/groups/{id}/assignments", 99999)
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content("[]"))
                    .andExpect(status().isNotFound());
        }
    }
}
