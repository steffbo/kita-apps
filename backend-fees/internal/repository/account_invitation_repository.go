package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

// InvitationCandidate is one invitable email address. Temporarily (2026-09-30, to test the
// invitation on production) it also includes parents whose linked active PARENT account
// already exists; UserID is then set and inviting sends a new set-password link.
type InvitationCandidate struct {
	ParentID  uuid.UUID `json:"parentId" db:"parent_id"`
	FirstName string    `json:"firstName" db:"first_name"`
	LastName  string    `json:"lastName" db:"last_name"`
	Email     string    `json:"email" db:"email"`
	Ambiguous bool      `json:"ambiguous" db:"ambiguous"`
	// Children lists the parent's active children ("Vorname Nachname").
	Children pq.StringArray `json:"children" db:"children" swaggertype:"array,string"`
	// UserID is the existing account of the parent (temporary, see above).
	UserID *uuid.UUID `json:"userId,omitempty" db:"user_id" binding:"optional"`
}

// AccountInvitationRepository stores the one active invitation for each user.
type AccountInvitationRepository struct{ db *sqlx.DB }

func NewAccountInvitationRepository(db *sqlx.DB) *AccountInvitationRepository {
	return &AccountInvitationRepository{db: db}
}

// Candidates lists parents who have at least one active child at today (active flag set and
// no exit date before today) and no account, or only their own linked active PARENT account.
func (r *AccountInvitationRepository) Candidates(ctx context.Context,
	today time.Time) ([]InvitationCandidate, error) {
	result := []InvitationCandidate{}
	err := r.db.SelectContext(ctx, &result, `
        WITH active_children AS (
            SELECT cp.parent_id, c.first_name || ' ' || c.last_name AS name, c.first_name, c.last_name
            FROM fees.child_parents cp JOIN fees.children c ON c.id = cp.child_id
            WHERE c.is_active AND (c.exit_date IS NULL OR c.exit_date >= $1)
        ), parent_emails AS (
            SELECT p.id AS parent_id, p.first_name, p.last_name, p.email,
                COUNT(*) OVER (PARTITION BY LOWER(p.email)) > 1 AS ambiguous
            FROM fees.parents p
            WHERE NULLIF(TRIM(p.email), '') IS NOT NULL
        ), possible AS (
            SELECT p.*,
                acc.id AS user_id,
                ROW_NUMBER() OVER (PARTITION BY LOWER(p.email) ORDER BY p.parent_id) AS row_no
            FROM parent_emails p
            LEFT JOIN fees.users acc ON acc.parent_id = p.parent_id
                AND acc.role = 'PARENT' AND acc.is_active
            WHERE NOT EXISTS (SELECT 1 FROM fees.users u
                    WHERE (u.parent_id = p.parent_id OR LOWER(u.email) = LOWER(p.email))
                      AND u.id IS DISTINCT FROM acc.id)
              AND EXISTS (SELECT 1 FROM active_children a WHERE a.parent_id = p.parent_id)
        )
        SELECT p.parent_id, p.first_name, p.last_name, p.email, p.ambiguous, p.user_id,
            ARRAY(SELECT a.name FROM active_children a WHERE a.parent_id = p.parent_id
                ORDER BY a.last_name, a.first_name) AS children
        FROM possible p WHERE p.row_no = 1 ORDER BY p.last_name, p.first_name, p.parent_id`, today)
	return result, err
}

func (r *AccountInvitationRepository) CreatePending(ctx context.Context, user *domain.User) error {
	user.InvitationPending = true
	user.PasswordHash = ""
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO fees.users (id, email, password_hash, first_name, last_name, role,
            parent_id, is_active, invitation_pending)
        VALUES ($1,$2,'',$3,$4,'PARENT',$5,TRUE,TRUE)`,
		user.ID, user.Email, user.FirstName, user.LastName, user.ParentID)
	return mapUniqueViolation(err)
}

// Reopen marks an existing active PARENT account of the parent as invitation pending, so a
// set-password link can be sent (temporary, see InvitationCandidate). Login is blocked until
// the new password is set.
func (r *AccountInvitationRepository) Reopen(ctx context.Context, userID, parentID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
        UPDATE fees.users SET invitation_pending = TRUE, updated_at = NOW()
        WHERE id = $1 AND parent_id = $2 AND role = 'PARENT' AND is_active`, userID, parentID)
	if err != nil {
		return err
	}
	return requireRow(result)
}

// Unreopen reverts Reopen when the mail could not be sent and the account still has a password.
func (r *AccountInvitationRepository) Unreopen(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
        UPDATE fees.users SET invitation_pending = FALSE, updated_at = NOW()
        WHERE id = $1 AND invitation_pending AND password_hash <> ''`, userID)
	return err
}

func (r *AccountInvitationRepository) SaveToken(ctx context.Context, userID uuid.UUID,
	tokenHash string, expiresAt time.Time) error {
	result, err := r.db.ExecContext(ctx, `
        INSERT INTO fees.account_invitations (user_id, token_hash, expires_at)
        SELECT id, $2, $3 FROM fees.users
        WHERE id = $1 AND invitation_pending AND is_active AND role = 'PARENT'
        ON CONFLICT (user_id) DO UPDATE SET token_hash = EXCLUDED.token_hash,
            expires_at = EXCLUDED.expires_at`, userID, tokenHash, expiresAt)
	if err != nil {
		return err
	}
	return requireRow(result)
}

func (r *AccountInvitationRepository) Consume(ctx context.Context, tokenHash, passwordHash string) error {
	var id uuid.UUID
	err := r.db.GetContext(ctx, &id, `
        WITH claimed AS (
            DELETE FROM fees.account_invitations
            WHERE token_hash = $1 AND expires_at > NOW()
            RETURNING user_id
        )
        UPDATE fees.users u SET password_hash = $2, invitation_pending = FALSE,
            updated_at = NOW()
        FROM claimed WHERE u.id = claimed.user_id AND u.invitation_pending
            AND u.is_active AND u.role = 'PARENT'
        RETURNING u.id`, tokenHash, passwordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
