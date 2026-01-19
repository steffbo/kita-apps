package de.knirpsenstadt.service;

import de.knirpsenstadt.api.model.*;
import de.knirpsenstadt.exception.BadRequestException;
import de.knirpsenstadt.exception.UnauthorizedException;
import de.knirpsenstadt.model.Employee;
import de.knirpsenstadt.repository.EmployeeRepository;
import de.knirpsenstadt.security.EmployeePrincipal;
import de.knirpsenstadt.security.JwtService;
import lombok.RequiredArgsConstructor;
import org.springframework.security.authentication.AuthenticationManager;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.Authentication;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.security.SecureRandom;
import java.time.OffsetDateTime;
import java.util.Base64;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

@Service
@RequiredArgsConstructor
public class AuthService {

    private final AuthenticationManager authenticationManager;
    private final JwtService jwtService;
    private final EmployeeRepository employeeRepository;
    private final PasswordEncoder passwordEncoder;

    // Simple in-memory token store (in production, use Redis or DB)
    private final Map<String, PasswordResetToken> resetTokens = new ConcurrentHashMap<>();

    public AuthResponse login(LoginRequest request) {
        Authentication authentication = authenticationManager.authenticate(
                new UsernamePasswordAuthenticationToken(request.getEmail(), request.getPassword())
        );

        EmployeePrincipal principal = (EmployeePrincipal) authentication.getPrincipal();
        Employee employee = employeeRepository.findById(principal.getId())
                .orElseThrow(() -> new UnauthorizedException("Benutzer nicht gefunden"));

        String accessToken = jwtService.generateAccessToken(principal);
        String refreshToken = jwtService.generateRefreshToken(principal);

        AuthResponse response = new AuthResponse();
        response.setAccessToken(accessToken);
        response.setRefreshToken(refreshToken);
        response.setExpiresIn((int) (jwtService.getAccessTokenExpiration() / 1000));
        response.setUser(toApiEmployee(employee));

        return response;
    }

    public AuthResponse refreshToken(RefreshTokenRequest request) {
        String token = request.getRefreshToken();

        try {
            String email = jwtService.extractUsername(token);
            String tokenType = jwtService.extractClaim(token, claims -> claims.get("type", String.class));

            if (!"refresh".equals(tokenType)) {
                throw new UnauthorizedException("Ungültiger Refresh Token");
            }

            Employee employee = employeeRepository.findByEmail(email)
                    .orElseThrow(() -> new UnauthorizedException("Benutzer nicht gefunden"));

            if (!employee.getActive()) {
                throw new UnauthorizedException("Benutzer ist deaktiviert");
            }

            EmployeePrincipal principal = new EmployeePrincipal(employee);

            if (jwtService.isTokenExpired(token)) {
                throw new UnauthorizedException("Refresh Token ist abgelaufen");
            }

            String newAccessToken = jwtService.generateAccessToken(principal);
            String newRefreshToken = jwtService.generateRefreshToken(principal);

            AuthResponse response = new AuthResponse();
            response.setAccessToken(newAccessToken);
            response.setRefreshToken(newRefreshToken);
            response.setExpiresIn((int) (jwtService.getAccessTokenExpiration() / 1000));
            response.setUser(toApiEmployee(employee));

            return response;
        } catch (Exception e) {
            throw new UnauthorizedException("Ungültiger Refresh Token");
        }
    }

    public de.knirpsenstadt.api.model.Employee getCurrentUser(EmployeePrincipal principal) {
        Employee employee = employeeRepository.findById(principal.getId())
                .orElseThrow(() -> new UnauthorizedException("Benutzer nicht gefunden"));
        return toApiEmployee(employee);
    }

    @Transactional
    public MessageResponse changePassword(ChangePasswordRequest request, EmployeePrincipal principal) {
        Employee employee = employeeRepository.findById(principal.getId())
                .orElseThrow(() -> new UnauthorizedException("Benutzer nicht gefunden"));

        if (!passwordEncoder.matches(request.getCurrentPassword(), employee.getPasswordHash())) {
            throw new BadRequestException("Aktuelles Passwort ist falsch");
        }

        employee.setPasswordHash(passwordEncoder.encode(request.getNewPassword()));
        employeeRepository.save(employee);

        MessageResponse response = new MessageResponse();
        response.setMessage("Passwort wurde erfolgreich geändert");
        return response;
    }

    public MessageResponse requestPasswordReset(PasswordResetRequest request) {
        // Always return success message to prevent email enumeration
        MessageResponse response = new MessageResponse();
        response.setMessage("Falls die E-Mail-Adresse existiert, wurde eine Anleitung zum Zurücksetzen gesendet");

        employeeRepository.findByEmail(request.getEmail()).ifPresent(employee -> {
            String token = generateResetToken();
            resetTokens.put(token, new PasswordResetToken(employee.getEmail(), OffsetDateTime.now().plusHours(1)));

            // TODO: Send email with reset link
            // For development, log the token
            System.out.println("Password reset token for " + employee.getEmail() + ": " + token);
        });

        return response;
    }

    @Transactional
    public MessageResponse confirmPasswordReset(PasswordResetConfirm request) {
        PasswordResetToken resetToken = resetTokens.get(request.getToken());

        if (resetToken == null) {
            throw new BadRequestException("Ungültiger oder abgelaufener Token");
        }

        if (resetToken.expiresAt().isBefore(OffsetDateTime.now())) {
            resetTokens.remove(request.getToken());
            throw new BadRequestException("Token ist abgelaufen");
        }

        Employee employee = employeeRepository.findByEmail(resetToken.email())
                .orElseThrow(() -> new BadRequestException("Benutzer nicht gefunden"));

        employee.setPasswordHash(passwordEncoder.encode(request.getNewPassword()));
        employeeRepository.save(employee);

        resetTokens.remove(request.getToken());

        MessageResponse response = new MessageResponse();
        response.setMessage("Passwort wurde erfolgreich zurückgesetzt");
        return response;
    }

    private String generateResetToken() {
        byte[] bytes = new byte[32];
        new SecureRandom().nextBytes(bytes);
        return Base64.getUrlEncoder().withoutPadding().encodeToString(bytes);
    }

    public static de.knirpsenstadt.api.model.Employee toApiEmployee(Employee entity) {
        de.knirpsenstadt.api.model.Employee dto = new de.knirpsenstadt.api.model.Employee();
        dto.setId(entity.getId());
        dto.setEmail(entity.getEmail());
        dto.setFirstName(entity.getFirstName());
        dto.setLastName(entity.getLastName());
        dto.setRole(EmployeeRole.fromValue(entity.getRole().name()));
        dto.setWeeklyHours(entity.getWeeklyHours().floatValue());
        dto.setVacationDaysPerYear(entity.getVacationDaysPerYear());
        dto.setRemainingVacationDays(entity.getRemainingVacationDays().floatValue());
        dto.setOvertimeBalance(entity.getOvertimeBalance().floatValue());
        dto.setActive(entity.getActive());
        dto.setCreatedAt(entity.getCreatedAt());
        dto.setUpdatedAt(entity.getUpdatedAt());
        return dto;
    }

    private record PasswordResetToken(String email, OffsetDateTime expiresAt) {}
}
