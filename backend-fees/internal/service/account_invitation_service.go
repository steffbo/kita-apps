package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

// InvitationSender is satisfied by the existing SMTP service.
type InvitationSender interface {
	IsEnabled() bool
	SendTextEmail(to, subject, body string) error
}

// AccountInvitationService creates parent accounts and single-use password links.
type AccountInvitationService struct {
	users       repository.UserRepository
	invitations *repository.AccountInvitationRepository
	logs        repository.EmailLogRepository
	sender      InvitationSender
	baseURL     string
}

func NewAccountInvitationService(users repository.UserRepository,
	invitations *repository.AccountInvitationRepository, logs repository.EmailLogRepository,
	sender InvitationSender, baseURL string) *AccountInvitationService {
	return &AccountInvitationService{users, invitations, logs, sender, baseURL}
}

// InvitationResult reports a single item without aborting the batch.
type InvitationResult struct {
	ParentID uuid.UUID `json:"parentId"`
	Email    string    `json:"email,omitempty" binding:"optional"`
	Success  bool      `json:"success"`
	Error    string    `json:"error,omitempty" binding:"optional"`
}

func (s *AccountInvitationService) Candidates(ctx context.Context) ([]repository.InvitationCandidate, error) {
	return s.invitations.Candidates(ctx)
}

func (s *AccountInvitationService) Invite(ctx context.Context, parentIDs []uuid.UUID,
	actorID uuid.UUID) ([]InvitationResult, error) {
	candidates, err := s.Candidates(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]repository.InvitationCandidate, len(candidates))
	for _, c := range candidates {
		byID[c.ParentID] = c
	}
	seen := make(map[uuid.UUID]bool)
	results := make([]InvitationResult, 0, len(parentIDs))
	for _, id := range parentIDs {
		result := InvitationResult{ParentID: id}
		c, found := byID[id]
		if found {
			result.Email = c.Email
		}
		switch {
		case seen[id]:
			result.Error = "Elternteil wurde mehrfach ausgewählt"
		case !found:
			result.Error = "Elternteil ist nicht verfügbar"
		case c.Ambiguous:
			result.Error = "Mehrere Elternteile haben diese E-Mail-Adresse"
		default:
			result.Error = s.inviteOne(ctx, c, actorID)
			result.Success = result.Error == ""
		}
		seen[id] = true
		results = append(results, result)
	}
	return results, nil
}

func (s *AccountInvitationService) inviteOne(ctx context.Context,
	c repository.InvitationCandidate, actorID uuid.UUID) string {
	if s.sender == nil || !s.sender.IsEnabled() {
		return "E-Mail-Versand ist nicht eingerichtet"
	}
	input := UserInput{Email: c.Email, FirstName: &c.FirstName, LastName: &c.LastName,
		Role: domain.UserRolePARENT, IsActive: true}
	if err := input.normalize(); err != nil {
		return err.Error()
	}
	user := &domain.User{
		Email: input.Email, FirstName: input.FirstName, LastName: input.LastName,
		Role: domain.UserRolePARENT, IsActive: true,
	}
	// Reuse the exact matching and ambiguity checks of the single-account path.
	userService := NewUserService(s.users, nil)
	if err := userService.linkParent(ctx, user); err != nil {
		return err.Error()
	}
	if user.ParentID == nil || *user.ParentID != c.ParentID {
		return "Elternteil ist nicht mehr verfügbar"
	}
	if err := s.invitations.CreatePending(ctx, user); err != nil {
		return mapUserRepoError(err).Error()
	}
	if err := s.send(ctx, user, actorID); err != nil {
		return err.Error()
	}
	return ""
}

func (s *AccountInvitationService) Resend(ctx context.Context, id, actorID uuid.UUID) error {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return mapUserRepoError(err)
	}
	if !user.InvitationPending || user.Role != domain.UserRolePARENT || !user.IsActive || user.ParentID == nil {
		return fmt.Errorf("%w: Keine ausstehende Einladung", ErrInvalidInput)
	}
	return s.send(ctx, user, actorID)
}

func (s *AccountInvitationService) send(ctx context.Context, user *domain.User,
	actorID uuid.UUID) error {
	if s.sender == nil || !s.sender.IsEnabled() {
		return fmt.Errorf("%w: E-Mail-Versand ist nicht eingerichtet", ErrInvalidInput)
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	token := hex.EncodeToString(raw)
	expiresAt := util.Now().Add(7 * 24 * time.Hour)
	if err := s.invitations.SaveToken(ctx, user.ID, hashToken(token), expiresAt); err != nil {
		return mapUserRepoError(err)
	}
	link := strings.TrimRight(s.baseURL, "/") + "/passwort-setzen?token=" + token
	subject := "Dein Zugang zur Kita-App der Knirpsenstadt"
	greeting := "Hallo"
	if user.FirstName != nil && strings.TrimSpace(*user.FirstName) != "" {
		greeting += " " + strings.TrimSpace(*user.FirstName)
	}
	body := fmt.Sprintf(`%s,

für dich wurde ein Zugang zur Kita-App der Knirpsenstadt angelegt. Dort siehst du die Beiträge
und Elternstunden eurer Familie und kannst eure Kontaktdaten pflegen.

Bitte setze über diesen Link zuerst dein Passwort:
%s

Der Link ist sieben Tage gültig und funktioniert nur einmal. Danach meldest du dich mit
dieser E-Mail-Adresse (%s) und deinem Passwort an.

Freundliche Grüße
Knirpsenstadt Beitrag`, greeting, link, user.Email)
	if err := s.sender.SendTextEmail(user.Email, subject, body); err != nil {
		return fmt.Errorf("Einladungs-Mail konnte nicht gesendet werden: %w", err)
	}
	masked := strings.Replace(body, link, strings.TrimRight(s.baseURL, "/")+
		"/passwort-setzen?token=[ausgeblendet]", 1)
	if err := s.logs.Create(ctx, &domain.EmailLog{
		ID: uuid.New(), SentAt: util.Now(), ToEmail: user.Email, Subject: subject,
		Body: &masked, EmailType: domain.EmailLogTypeAccountInvitation, SentBy: &actorID,
	}); err != nil {
		return fmt.Errorf("E-Mail-Log konnte nicht gespeichert werden: %w", err)
	}
	return nil
}

func (s *AccountInvitationService) Valid(ctx context.Context, token string) (bool, error) {
	if len(token) != 64 {
		return false, nil
	}
	return s.invitations.Valid(ctx, hashToken(token))
}

func (s *AccountInvitationService) SetPassword(ctx context.Context, token, password string) error {
	if err := validatePassword(password); err != nil {
		return err
	}
	if len(token) != 64 {
		return ErrNotFound
	}
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	err = s.invitations.Consume(ctx, hashToken(token), passwordHash)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
