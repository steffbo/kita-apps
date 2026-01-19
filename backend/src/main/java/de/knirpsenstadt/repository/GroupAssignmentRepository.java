package de.knirpsenstadt.repository;

import de.knirpsenstadt.model.GroupAssignment;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.util.List;

@Repository
public interface GroupAssignmentRepository extends JpaRepository<GroupAssignment, Long> {

    List<GroupAssignment> findByGroupId(Long groupId);

    List<GroupAssignment> findByEmployeeId(Long employeeId);

    @Query("SELECT ga FROM GroupAssignment ga " +
           "JOIN FETCH ga.employee " +
           "JOIN FETCH ga.group " +
           "WHERE ga.group.id = :groupId")
    List<GroupAssignment> findByGroupIdWithEmployee(@Param("groupId") Long groupId);

    @Query("SELECT ga FROM GroupAssignment ga " +
           "JOIN FETCH ga.group " +
           "WHERE ga.employee.id = :employeeId")
    List<GroupAssignment> findByEmployeeIdWithGroup(@Param("employeeId") Long employeeId);

    void deleteByGroupIdAndEmployeeId(Long groupId, Long employeeId);

    void deleteByGroupId(Long groupId);

    boolean existsByGroupIdAndEmployeeId(Long groupId, Long employeeId);

    List<GroupAssignment> findByEmployeeIdAndAssignmentType(Long employeeId, de.knirpsenstadt.model.AssignmentType assignmentType);
}
