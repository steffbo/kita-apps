package de.knirpsenstadt.controller;

import de.knirpsenstadt.AbstractIntegrationTest;
import de.knirpsenstadt.api.model.*;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.springframework.http.MediaType;

import static org.hamcrest.Matchers.*;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@DisplayName("AuthController Integration Tests")
class AuthControllerIntegrationTest extends AbstractIntegrationTest {

    @Nested
    @DisplayName("POST /auth/login")
    class LoginTests {

        @Test
        @DisplayName("should login successfully with valid credentials")
        void loginWithValidCredentials() throws Exception {
            LoginRequest request = new LoginRequest();
            request.setEmail(ADMIN_EMAIL);
            request.setPassword(ADMIN_PASSWORD);

            mockMvc.perform(post("/auth/login")
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.accessToken").isNotEmpty())
                    .andExpect(jsonPath("$.refreshToken").isNotEmpty())
                    .andExpect(jsonPath("$.expiresIn").isNumber())
                    .andExpect(jsonPath("$.user.email").value(ADMIN_EMAIL))
                    .andExpect(jsonPath("$.user.role").value("ADMIN"));
        }

        @Test
        @DisplayName("should return 401 with invalid password")
        void loginWithInvalidPassword() throws Exception {
            LoginRequest request = new LoginRequest();
            request.setEmail(ADMIN_EMAIL);
            request.setPassword("wrongpassword");

            mockMvc.perform(post("/auth/login")
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isUnauthorized());
        }

        @Test
        @DisplayName("should return 401 with non-existent email")
        void loginWithNonExistentEmail() throws Exception {
            LoginRequest request = new LoginRequest();
            request.setEmail("nonexistent@test.de");
            request.setPassword("password");

            mockMvc.perform(post("/auth/login")
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isUnauthorized());
        }

        @Test
        @DisplayName("should return 400 with missing email")
        void loginWithMissingEmail() throws Exception {
            LoginRequest request = new LoginRequest();
            request.setPassword(ADMIN_PASSWORD);

            mockMvc.perform(post("/auth/login")
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isBadRequest());
        }
    }

    @Nested
    @DisplayName("POST /auth/refresh")
    class RefreshTokenTests {

        @Test
        @DisplayName("should refresh token successfully")
        void refreshTokenSuccessfully() throws Exception {
            // First login to get tokens
            LoginRequest loginRequest = new LoginRequest();
            loginRequest.setEmail(ADMIN_EMAIL);
            loginRequest.setPassword(ADMIN_PASSWORD);

            String loginResponse = mockMvc.perform(post("/auth/login")
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(loginRequest)))
                    .andExpect(status().isOk())
                    .andReturn()
                    .getResponse()
                    .getContentAsString();

            AuthResponse authResponse = fromJson(loginResponse, AuthResponse.class);

            // Use refresh token
            RefreshTokenRequest refreshRequest = new RefreshTokenRequest();
            refreshRequest.setRefreshToken(authResponse.getRefreshToken());

            mockMvc.perform(post("/auth/refresh")
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(refreshRequest)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.accessToken").isNotEmpty())
                    .andExpect(jsonPath("$.refreshToken").isNotEmpty())
                    .andExpect(jsonPath("$.user.email").value(ADMIN_EMAIL));
        }

        @Test
        @DisplayName("should return 401 with invalid refresh token")
        void refreshWithInvalidToken() throws Exception {
            RefreshTokenRequest request = new RefreshTokenRequest();
            request.setRefreshToken("invalid-token");

            mockMvc.perform(post("/auth/refresh")
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isUnauthorized());
        }
    }

    @Nested
    @DisplayName("GET /auth/me")
    class GetCurrentUserTests {

        @Test
        @DisplayName("should return current user with valid token")
        void getCurrentUserWithValidToken() throws Exception {
            String token = getAdminToken();

            mockMvc.perform(get("/auth/me")
                            .header("Authorization", bearerToken(token)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.email").value(ADMIN_EMAIL))
                    .andExpect(jsonPath("$.firstName").value("Admin"))
                    .andExpect(jsonPath("$.lastName").value("User"))
                    .andExpect(jsonPath("$.role").value("ADMIN"));
        }

        @Test
        @DisplayName("should return 401 without token")
        void getCurrentUserWithoutToken() throws Exception {
            mockMvc.perform(get("/auth/me"))
                    .andExpect(status().isUnauthorized());
        }

        @Test
        @DisplayName("should return 401 with invalid token")
        void getCurrentUserWithInvalidToken() throws Exception {
            mockMvc.perform(get("/auth/me")
                            .header("Authorization", bearerToken("invalid-token")))
                    .andExpect(status().isUnauthorized());
        }
    }

    @Nested
    @DisplayName("POST /auth/change-password")
    class ChangePasswordTests {

        @Test
        @DisplayName("should change password successfully")
        void changePasswordSuccessfully() throws Exception {
            // Create a temporary user for this test
            var tempUser = employeeRepository.save(
                    de.knirpsenstadt.model.Employee.builder()
                            .email("temp@test.de")
                            .firstName("Temp")
                            .lastName("User")
                            .passwordHash(passwordEncoder.encode("oldpass123"))
                            .role(de.knirpsenstadt.model.EmployeeRole.EMPLOYEE)
                            .weeklyHours(java.math.BigDecimal.valueOf(38))
                            .vacationDaysPerYear(30)
                            .remainingVacationDays(java.math.BigDecimal.valueOf(30))
                            .overtimeBalance(java.math.BigDecimal.ZERO)
                            .active(true)
                            .build()
            );

            String token = getToken("temp@test.de", "oldpass123");

            ChangePasswordRequest request = new ChangePasswordRequest();
            request.setCurrentPassword("oldpass123");
            request.setNewPassword("newpass123");

            mockMvc.perform(post("/auth/change-password")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.message").isNotEmpty());

            // Verify new password works
            LoginRequest loginRequest = new LoginRequest();
            loginRequest.setEmail("temp@test.de");
            loginRequest.setPassword("newpass123");

            mockMvc.perform(post("/auth/login")
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(loginRequest)))
                    .andExpect(status().isOk());

            // Clean up
            employeeRepository.delete(tempUser);
        }

        @Test
        @DisplayName("should return 400 with wrong current password")
        void changePasswordWithWrongCurrentPassword() throws Exception {
            String token = getAdminToken();

            ChangePasswordRequest request = new ChangePasswordRequest();
            request.setCurrentPassword("wrongpassword");
            request.setNewPassword("newpass123");

            mockMvc.perform(post("/auth/change-password")
                            .header("Authorization", bearerToken(token))
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isBadRequest());
        }
    }

    @Nested
    @DisplayName("POST /auth/password-reset/request")
    class RequestPasswordResetTests {

        @Test
        @DisplayName("should always return success message (prevents email enumeration)")
        void requestPasswordResetAlwaysReturnsSuccess() throws Exception {
            PasswordResetRequest request = new PasswordResetRequest();
            request.setEmail(ADMIN_EMAIL);

            mockMvc.perform(post("/auth/password-reset/request")
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.message").isNotEmpty());
        }

        @Test
        @DisplayName("should return success even for non-existent email")
        void requestPasswordResetNonExistentEmail() throws Exception {
            PasswordResetRequest request = new PasswordResetRequest();
            request.setEmail("nonexistent@test.de");

            mockMvc.perform(post("/auth/password-reset/request")
                            .contentType(MediaType.APPLICATION_JSON)
                            .content(toJson(request)))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.message").isNotEmpty());
        }
    }
}
