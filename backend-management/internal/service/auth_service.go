package service

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/knirpsenstadt/kita-apps/backend-management/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/email"
	"github.com/knirpsenstadt/kita-apps/backend-management/internal/repository"
)

type passwordResetToken struct {
	email     string
	expiresAt time.Time
}

// EmailSender defines the interface for sending emails.
type EmailSender interface {
	SendPasswordResetEmail(to, token, baseURL string) error
	IsEnabled() bool
}

// AuthService handles authentication logic.
type AuthService struct {
	employees    repository.EmployeeRepository
	jwtService   *auth.JWTService
	emailService EmailSender
	baseURL      string
	resetExpiry  time.Duration
	resetTokens  map[string]passwordResetToken
	resetMutex   sync.Mutex
}

// NewAuthService creates a new AuthService.
func NewAuthService(employees repository.EmployeeRepository, jwtService *auth.JWTService, emailService *email.Service, baseURL string, resetExpiry time.Duration) *AuthService {
	return &AuthService{
		employees:    employees,
		jwtService:   jwtService,
		emailService: emailService,
		baseURL:      baseURL,
		resetExpiry:  resetExpiry,
		resetTokens:  make(map[string]passwordResetToken),
	}
}

// AuthResult represents the result of a login or token refresh.
type AuthResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	Employee     *domain.Employee
}

// Login authenticates a user and returns tokens.
func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	employee, err := s.employees.GetByEmail(ctx, email)
	if err != nil || employee == nil {
		return nil, NewUnauthorized("Ungültige Anmeldedaten")
	}
	if !employee.Active {
		return nil, NewUnauthorized("Benutzer ist deaktiviert")
	}
	if !auth.CheckPassword(password, employee.PasswordHash) {
		return nil, NewUnauthorized("Ungültige Anmeldedaten")
	}

	pair, err := s.jwtService.GenerateTokenPair(employee.ID, employee.Email, string(employee.Role), employee.FullName())
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
		Employee:     employee,
	}, nil
}

// Refresh exchanges a refresh token for new tokens.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*AuthResult, error) {
	claims, err := s.jwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, NewUnauthorized("Ungültiger Refresh Token")
	}

	employee, err := s.employees.GetByEmail(ctx, claims.Subject)
	if err != nil || employee == nil {
		return nil, NewUnauthorized("Benutzer nicht gefunden")
	}
	if !employee.Active {
		return nil, NewUnauthorized("Benutzer ist deaktiviert")
	}

	pair, err := s.jwtService.GenerateTokenPair(employee.ID, employee.Email, string(employee.Role), employee.FullName())
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
		Employee:     employee,
	}, nil
}

// GetCurrentUser retrieves the current employee.
func (s *AuthService) GetCurrentUser(ctx context.Context, employeeID int64) (*domain.Employee, error) {
	employee, err := s.employees.GetByID(ctx, employeeID)
	if err != nil || employee == nil {
		return nil, NewUnauthorized("Benutzer nicht gefunden")
	}
	return employee, nil
}

// ChangePassword changes the user's password.
func (s *AuthService) ChangePassword(ctx context.Context, employeeID int64, currentPassword, newPassword string) error {
	employee, err := s.employees.GetByID(ctx, employeeID)
	if err != nil || employee == nil {
		return NewUnauthorized("Benutzer nicht gefunden")
	}

	if !auth.CheckPassword(currentPassword, employee.PasswordHash) {
		return NewBadRequest("Aktuelles Passwort ist falsch")
	}

	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}

	return s.employees.UpdatePassword(ctx, employeeID, hash)
}

// RequestPasswordReset creates a password reset token and sends an email.
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) {
	employee, err := s.employees.GetByEmail(ctx, email)
	if err != nil || employee == nil {
		return
	}

	token, err := auth.GenerateRandomToken(32)
	if err != nil {
		return
	}

	s.resetMutex.Lock()
	s.resetTokens[token] = passwordResetToken{
		email:     employee.Email,
		expiresAt: time.Now().Add(s.resetExpiry),
	}
	s.resetMutex.Unlock()

	// Send password reset email
	if s.emailService != nil && s.emailService.IsEnabled() {
		if err := s.emailService.SendPasswordResetEmail(employee.Email, token, s.baseURL); err != nil {
			log.Error().Err(err).Str("email", employee.Email).Msg("failed to send password reset email")
		}
	} else {
		// Fallback to logging when email is not configured
		log.Info().Str("email", employee.Email).Str("token", token).Msg("password reset token generated (email not configured)")
	}
}

// ConfirmPasswordReset confirms a reset token and sets a new password.
func (s *AuthService) ConfirmPasswordReset(ctx context.Context, token, newPassword string) error {
	s.resetMutex.Lock()
	resetToken, ok := s.resetTokens[token]
	s.resetMutex.Unlock()

	if !ok {
		return NewBadRequest("Ungültiger oder abgelaufener Token")
	}

	if time.Now().After(resetToken.expiresAt) {
		s.resetMutex.Lock()
		delete(s.resetTokens, token)
		s.resetMutex.Unlock()
		return NewBadRequest("Token ist abgelaufen")
	}

	employee, err := s.employees.GetByEmail(ctx, resetToken.email)
	if err != nil || employee == nil {
		return NewBadRequest("Benutzer nicht gefunden")
	}

	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}

	if err := s.employees.UpdatePassword(ctx, employee.ID, hash); err != nil {
		return err
	}

	s.resetMutex.Lock()
	delete(s.resetTokens, token)
	s.resetMutex.Unlock()

	return nil
}
