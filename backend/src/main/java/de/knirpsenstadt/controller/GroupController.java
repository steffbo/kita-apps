package de.knirpsenstadt.controller;

import de.knirpsenstadt.api.GroupsApi;
import de.knirpsenstadt.api.model.*;
import de.knirpsenstadt.service.GroupService;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

@RestController
@RequiredArgsConstructor
public class GroupController implements GroupsApi {

    private final GroupService groupService;

    @Override
    public ResponseEntity<List<Group>> listGroups() {
        List<Group> groups = groupService.getAllGroups();
        return ResponseEntity.ok(groups);
    }

    @Override
    public ResponseEntity<GroupWithMembers> getGroup(Long id) {
        GroupWithMembers group = groupService.getGroupWithMembers(id);
        return ResponseEntity.ok(group);
    }

    @Override
    public ResponseEntity<Group> createGroup(CreateGroupRequest createGroupRequest) {
        Group group = groupService.createGroup(createGroupRequest);
        return ResponseEntity.status(201).body(group);
    }

    @Override
    public ResponseEntity<Group> updateGroup(Long id, CreateGroupRequest createGroupRequest) {
        Group group = groupService.updateGroup(id, createGroupRequest);
        return ResponseEntity.ok(group);
    }

    @Override
    public ResponseEntity<Void> deleteGroup(Long id) {
        groupService.deleteGroup(id);
        return ResponseEntity.noContent().build();
    }

    @Override
    public ResponseEntity<List<GroupAssignment>> getGroupAssignments(Long id) {
        List<GroupAssignment> assignments = groupService.getGroupAssignments(id);
        return ResponseEntity.ok(assignments);
    }

    @Override
    public ResponseEntity<List<GroupAssignment>> updateGroupAssignments(Long id, List<GroupAssignmentRequest> groupAssignmentRequest) {
        List<GroupAssignment> assignments = groupService.updateGroupAssignments(id, groupAssignmentRequest);
        return ResponseEntity.ok(assignments);
    }
}
