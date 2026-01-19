package de.knirpsenstadt.controller;

import de.knirpsenstadt.api.AuthApi;
import de.knirpsenstadt.api.model.*;
import de.knirpsenstadt.security.EmployeePrincipal;
import de.knirpsenstadt.service.AuthService;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequiredArgsConstructor
public class AuthController implements AuthApi {

    private final AuthService authService;

    @Override
    public ResponseEntity<AuthResponse> login(LoginRequest loginRequest) {
        AuthResponse response = authService.login(loginRequest);
        return ResponseEntity.ok(response);
    }

    @Override
    public ResponseEntity<AuthResponse> refreshToken(RefreshTokenRequest refreshTokenRequest) {
        AuthResponse response = authService.refreshToken(refreshTokenRequest);
        return ResponseEntity.ok(response);
    }

    @Override
    public ResponseEntity<Employee> getCurrentUser() {
        // This will be injected via Spring Security
        return ResponseEntity.ok(authService.getCurrentUser(getCurrentPrincipal()));
    }

    @Override
    public ResponseEntity<MessageResponse> changePassword(ChangePasswordRequest changePasswordRequest) {
        MessageResponse response = authService.changePassword(changePasswordRequest, getCurrentPrincipal());
        return ResponseEntity.ok(response);
    }

    @Override
    public ResponseEntity<MessageResponse> requestPasswordReset(PasswordResetRequest passwordResetRequest) {
        MessageResponse response = authService.requestPasswordReset(passwordResetRequest);
        return ResponseEntity.ok(response);
    }

    @Override
    public ResponseEntity<MessageResponse> confirmPasswordReset(PasswordResetConfirm passwordResetConfirm) {
        MessageResponse response = authService.confirmPasswordReset(passwordResetConfirm);
        return ResponseEntity.ok(response);
    }

    // Helper to get current principal - will be resolved by Spring Security context
    private EmployeePrincipal getCurrentPrincipal() {
        org.springframework.security.core.context.SecurityContext context = 
                org.springframework.security.core.context.SecurityContextHolder.getContext();
        return (EmployeePrincipal) context.getAuthentication().getPrincipal();
    }
}
