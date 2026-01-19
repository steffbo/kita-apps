package de.knirpsenstadt.service;

import de.knirpsenstadt.api.model.*;
import de.knirpsenstadt.exception.BadRequestException;
import de.knirpsenstadt.exception.ResourceNotFoundException;
import de.knirpsenstadt.model.AssignmentType;
import de.knirpsenstadt.model.Employee;
import de.knirpsenstadt.model.Group;
import de.knirpsenstadt.model.GroupAssignment;
import de.knirpsenstadt.repository.EmployeeRepository;
import de.knirpsenstadt.repository.GroupAssignmentRepository;
import de.knirpsenstadt.repository.GroupRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;
import java.util.stream.Collectors;

@Service
@RequiredArgsConstructor
public class GroupService {

    private final GroupRepository groupRepository;
    private final GroupAssignmentRepository groupAssignmentRepository;
    private final EmployeeRepository employeeRepository;

    public List<de.knirpsenstadt.api.model.Group> getAllGroups() {
        return groupRepository.findAllOrderByName().stream()
                .map(this::toApiGroup)
                .collect(Collectors.toList());
    }

    public GroupWithMembers getGroupWithMembers(Long id) {
        Group group = groupRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Gruppe", id));

        List<GroupAssignment> assignments = groupAssignmentRepository.findByGroupId(id);

        GroupWithMembers result = new GroupWithMembers();
        result.setId(group.getId());
        result.setName(group.getName());
        result.setDescription(group.getDescription());
        result.setColor(group.getColor());

        List<de.knirpsenstadt.api.model.GroupAssignment> dtoAssignments = assignments.stream()
                .map(this::toApiGroupAssignment)
                .collect(Collectors.toList());

        result.setMembers(dtoAssignments);
        return result;
    }

    @Transactional
    public de.knirpsenstadt.api.model.Group createGroup(CreateGroupRequest request) {
        Group group = Group.builder()
                .name(request.getName())
                .description(request.getDescription())
                .color(request.getColor())
                .build();

        Group saved = groupRepository.save(group);
        return toApiGroup(saved);
    }

    @Transactional
    public de.knirpsenstadt.api.model.Group updateGroup(Long id, CreateGroupRequest request) {
        Group group = groupRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Gruppe", id));

        group.setName(request.getName());
        if (request.getDescription() != null) {
            group.setDescription(request.getDescription());
        }
        if (request.getColor() != null) {
            group.setColor(request.getColor());
        }

        Group saved = groupRepository.save(group);
        return toApiGroup(saved);
    }

    @Transactional
    public void deleteGroup(Long id) {
        if (!groupRepository.existsById(id)) {
            throw new ResourceNotFoundException("Gruppe", id);
        }

        groupAssignmentRepository.deleteByGroupId(id);
        groupRepository.deleteById(id);
    }

    @Transactional
    public de.knirpsenstadt.api.model.GroupAssignment addGroupMember(Long groupId, GroupAssignmentRequest request) {
        Group group = groupRepository.findById(groupId)
                .orElseThrow(() -> new ResourceNotFoundException("Gruppe", groupId));

        Employee employee = employeeRepository.findById(request.getEmployeeId())
                .orElseThrow(() -> new ResourceNotFoundException("Mitarbeiter", request.getEmployeeId()));

        AssignmentType assignmentType = AssignmentType.valueOf(request.getAssignmentType().getValue());

        // Check if employee is already a PERMANENT member of another group
        if (assignmentType == AssignmentType.PERMANENT) {
            List<GroupAssignment> existingPrimary = groupAssignmentRepository
                    .findByEmployeeIdAndAssignmentType(employee.getId(), AssignmentType.PERMANENT);
            if (!existingPrimary.isEmpty()) {
                throw new BadRequestException("Mitarbeiter ist bereits Stammpersonal einer anderen Gruppe");
            }
        }

        GroupAssignment assignment = GroupAssignment.builder()
                .group(group)
                .employee(employee)
                .assignmentType(assignmentType)
                .build();

        GroupAssignment saved = groupAssignmentRepository.save(assignment);
        return toApiGroupAssignment(saved);
    }

    @Transactional
    public void removeGroupMember(Long groupId, Long employeeId) {
        groupAssignmentRepository.deleteByGroupIdAndEmployeeId(groupId, employeeId);
    }

    public List<de.knirpsenstadt.api.model.GroupAssignment> getGroupAssignments(Long groupId) {
        if (!groupRepository.existsById(groupId)) {
            throw new ResourceNotFoundException("Gruppe", groupId);
        }

        List<GroupAssignment> assignments = groupAssignmentRepository.findByGroupId(groupId);
        return assignments.stream()
                .map(this::toApiGroupAssignment)
                .collect(Collectors.toList());
    }

    @Transactional
    public List<de.knirpsenstadt.api.model.GroupAssignment> updateGroupAssignments(Long groupId, List<GroupAssignmentRequest> requests) {
        Group group = groupRepository.findById(groupId)
                .orElseThrow(() -> new ResourceNotFoundException("Gruppe", groupId));

        // Delete existing assignments
        groupAssignmentRepository.deleteByGroupId(groupId);

        // Create new assignments
        List<GroupAssignment> newAssignments = requests.stream()
                .map(request -> {
                    Employee employee = employeeRepository.findById(request.getEmployeeId())
                            .orElseThrow(() -> new ResourceNotFoundException("Mitarbeiter", request.getEmployeeId()));

                    return GroupAssignment.builder()
                            .group(group)
                            .employee(employee)
                            .assignmentType(AssignmentType.valueOf(request.getAssignmentType().getValue()))
                            .build();
                })
                .collect(Collectors.toList());

        List<GroupAssignment> saved = groupAssignmentRepository.saveAll(newAssignments);
        return saved.stream()
                .map(this::toApiGroupAssignment)
                .collect(Collectors.toList());
    }

    private de.knirpsenstadt.api.model.Group toApiGroup(Group entity) {
        de.knirpsenstadt.api.model.Group dto = new de.knirpsenstadt.api.model.Group();
        dto.setId(entity.getId());
        dto.setName(entity.getName());
        dto.setDescription(entity.getDescription());
        dto.setColor(entity.getColor());
        return dto;
    }

    private de.knirpsenstadt.api.model.GroupAssignment toApiGroupAssignment(GroupAssignment entity) {
        de.knirpsenstadt.api.model.GroupAssignment dto = new de.knirpsenstadt.api.model.GroupAssignment();
        dto.setId(entity.getId());
        dto.setEmployeeId(entity.getEmployee().getId());
        dto.setGroupId(entity.getGroup().getId());
        dto.setAssignmentType(de.knirpsenstadt.api.model.AssignmentType.fromValue(entity.getAssignmentType().name()));
        dto.setEmployee(AuthService.toApiEmployee(entity.getEmployee()));
        return dto;
    }
}
