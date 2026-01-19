package de.knirpsenstadt;

import com.fasterxml.jackson.databind.ObjectMapper;
import de.knirpsenstadt.api.model.AuthResponse;
import de.knirpsenstadt.api.model.LoginRequest;
import de.knirpsenstadt.model.Employee;
import de.knirpsenstadt.model.EmployeeRole;
import de.knirpsenstadt.repository.EmployeeRepository;
import org.junit.jupiter.api.BeforeEach;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.http.MediaType;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.test.context.ActiveProfiles;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.MvcResult;
import org.testcontainers.containers.PostgreSQLContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;

import java.math.BigDecimal;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

/**
 * Base class for all integration tests.
 * Provides PostgreSQL Testcontainer, MockMvc, and authentication helpers.
 */
@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT)
@AutoConfigureMockMvc
@Testcontainers
@ActiveProfiles("test")
public abstract class AbstractIntegrationTest {

    @Container
    static PostgreSQLContainer<?> postgres = new PostgreSQLContainer<>("postgres:15-alpine")
            .withDatabaseName("kita_test")
            .withUsername("test")
            .withPassword("test");

    @DynamicPropertySource
    static void configureProperties(DynamicPropertyRegistry registry) {
        registry.add("spring.datasource.url", postgres::getJdbcUrl);
        registry.add("spring.datasource.username", postgres::getUsername);
        registry.add("spring.datasource.password", postgres::getPassword);
    }

    @Autowired
    protected MockMvc mockMvc;

    @Autowired
    protected ObjectMapper objectMapper;

    @Autowired
    protected EmployeeRepository employeeRepository;

    @Autowired
    protected PasswordEncoder passwordEncoder;

    protected static final String ADMIN_EMAIL = "admin@test.de";
    protected static final String ADMIN_PASSWORD = "admin123";
    protected static final String EMPLOYEE_EMAIL = "employee@test.de";
    protected static final String EMPLOYEE_PASSWORD = "employee123";

    protected Employee adminEmployee;
    protected Employee regularEmployee;

    @BeforeEach
    void setUpTestUsers() {
        // Create admin user if not exists
        adminEmployee = employeeRepository.findByEmail(ADMIN_EMAIL)
                .orElseGet(() -> {
                    Employee admin = Employee.builder()
                            .email(ADMIN_EMAIL)
                            .firstName("Admin")
                            .lastName("User")
                            .passwordHash(passwordEncoder.encode(ADMIN_PASSWORD))
                            .role(EmployeeRole.ADMIN)
                            .weeklyHours(BigDecimal.valueOf(38))
                            .vacationDaysPerYear(30)
                            .remainingVacationDays(BigDecimal.valueOf(30))
                            .overtimeBalance(BigDecimal.ZERO)
                            .active(true)
                            .build();
                    return employeeRepository.save(admin);
                });

        // Create regular employee if not exists
        regularEmployee = employeeRepository.findByEmail(EMPLOYEE_EMAIL)
                .orElseGet(() -> {
                    Employee employee = Employee.builder()
                            .email(EMPLOYEE_EMAIL)
                            .firstName("Regular")
                            .lastName("Employee")
                            .passwordHash(passwordEncoder.encode(EMPLOYEE_PASSWORD))
                            .role(EmployeeRole.EMPLOYEE)
                            .weeklyHours(BigDecimal.valueOf(30))
                            .vacationDaysPerYear(26)
                            .remainingVacationDays(BigDecimal.valueOf(26))
                            .overtimeBalance(BigDecimal.ZERO)
                            .active(true)
                            .build();
                    return employeeRepository.save(employee);
                });
    }

    /**
     * Get JWT token for admin user
     */
    protected String getAdminToken() throws Exception {
        return getToken(ADMIN_EMAIL, ADMIN_PASSWORD);
    }

    /**
     * Get JWT token for regular employee
     */
    protected String getEmployeeToken() throws Exception {
        return getToken(EMPLOYEE_EMAIL, EMPLOYEE_PASSWORD);
    }

    /**
     * Get JWT token for a specific user
     */
    protected String getToken(String email, String password) throws Exception {
        LoginRequest loginRequest = new LoginRequest();
        loginRequest.setEmail(email);
        loginRequest.setPassword(password);

        MvcResult result = mockMvc.perform(post("/auth/login")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(loginRequest)))
                .andExpect(status().isOk())
                .andReturn();

        AuthResponse authResponse = objectMapper.readValue(
                result.getResponse().getContentAsString(),
                AuthResponse.class
        );

        return authResponse.getAccessToken();
    }

    /**
     * Create Authorization header value with Bearer token
     */
    protected String bearerToken(String token) {
        return "Bearer " + token;
    }

    /**
     * Helper to convert object to JSON string
     */
    protected String toJson(Object obj) throws Exception {
        return objectMapper.writeValueAsString(obj);
    }

    /**
     * Helper to parse JSON response to object
     */
    protected <T> T fromJson(String json, Class<T> clazz) throws Exception {
        return objectMapper.readValue(json, clazz);
    }
}
