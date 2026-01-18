package de.knirpsenstadt.repository;

import de.knirpsenstadt.model.Employee;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.stereotype.Repository;

import java.util.List;
import java.util.Optional;

@Repository
public interface EmployeeRepository extends JpaRepository<Employee, Long> {

    Optional<Employee> findByEmail(String email);

    boolean existsByEmail(String email);

    List<Employee> findByActiveTrue();

    @Query("SELECT e FROM Employee e WHERE e.active = true ORDER BY e.lastName, e.firstName")
    List<Employee> findAllActiveOrderByName();

    @Query("SELECT e FROM Employee e ORDER BY e.lastName, e.firstName")
    List<Employee> findAllOrderByName();
}
