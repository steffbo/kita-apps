package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

// UserRepository handles user persistence.
type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User) error
}

// RefreshTokenRepository handles refresh token persistence.
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *domain.RefreshToken) error
	Exists(ctx context.Context, userID uuid.UUID, tokenHash string) (bool, error)
	DeleteByHash(ctx context.Context, tokenHash string) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}

// ChildRepository handles child persistence.
type ChildRepository interface {
	List(ctx context.Context, activeOnly bool, u3Only bool, search string, sortBy string, sortDir string, offset, limit int) ([]domain.Child, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Child, error)
	GetByMemberNumber(ctx context.Context, memberNumber string) (*domain.Child, error)
	Create(ctx context.Context, child *domain.Child) error
	Update(ctx context.Context, child *domain.Child) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetParents(ctx context.Context, childID uuid.UUID) ([]domain.Parent, error)
	LinkParent(ctx context.Context, childID, parentID uuid.UUID, isPrimary bool) error
	UnlinkParent(ctx context.Context, childID, parentID uuid.UUID) error
}

// ParentRepository handles parent persistence.
type ParentRepository interface {
	List(ctx context.Context, search string, sortBy string, sortDir string, offset, limit int) ([]domain.Parent, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Parent, error)
	FindByNameAndEmail(ctx context.Context, firstName, lastName, email string) (*domain.Parent, error)
	Create(ctx context.Context, parent *domain.Parent) error
	Update(ctx context.Context, parent *domain.Parent) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetChildren(ctx context.Context, parentID uuid.UUID) ([]domain.Child, error)
	GetChildrenForParents(ctx context.Context, parentIDs []uuid.UUID) (map[uuid.UUID][]domain.Child, error)
}

// FeeFilter defines filters for fee queries.
type FeeFilter struct {
	Year    *int
	Month   *int
	FeeType string
	ChildID *uuid.UUID
}

// FeeRepository handles fee expectation persistence.
type FeeRepository interface {
	List(ctx context.Context, filter FeeFilter, offset, limit int) ([]domain.FeeExpectation, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.FeeExpectation, error)
	Create(ctx context.Context, fee *domain.FeeExpectation) error
	Update(ctx context.Context, fee *domain.FeeExpectation) error
	Delete(ctx context.Context, id uuid.UUID) error
	Exists(ctx context.Context, childID uuid.UUID, feeType domain.FeeType, year int, month *int) (bool, error)
	FindUnpaid(ctx context.Context, childID uuid.UUID, feeType domain.FeeType, year int, month *int) (*domain.FeeExpectation, error)
	FindOldestUnpaid(ctx context.Context, childID uuid.UUID, feeType domain.FeeType, amount float64) (*domain.FeeExpectation, error)
	FindOldestUnpaidWithReminder(ctx context.Context, childID uuid.UUID, feeType domain.FeeType, combinedAmount float64) ([]domain.FeeExpectation, error)
	GetOverview(ctx context.Context, year int) (*domain.FeeOverview, error)
}

// TransactionRepository handles bank transaction persistence.
type TransactionRepository interface {
	Create(ctx context.Context, tx *domain.BankTransaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.BankTransaction, error)
	Exists(ctx context.Context, bookingDate time.Time, payerIBAN *string, amount float64, description *string) (bool, error)
	ListUnmatched(ctx context.Context, offset, limit int) ([]domain.BankTransaction, int64, error)
	GetBatches(ctx context.Context, offset, limit int) ([]domain.ImportBatch, int64, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteUnmatchedByIBAN(ctx context.Context, iban string) (int64, error)
}

// MatchRepository handles payment match persistence.
type MatchRepository interface {
	Create(ctx context.Context, match *domain.PaymentMatch) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.PaymentMatch, error)
	ExistsForExpectation(ctx context.Context, expectationID uuid.UUID) (bool, error)
	ExistsForTransaction(ctx context.Context, transactionID uuid.UUID) (bool, error)
	GetByExpectation(ctx context.Context, expectationID uuid.UUID) (*domain.PaymentMatch, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// KnownIBANRepository handles known IBAN persistence.
type KnownIBANRepository interface {
	Create(ctx context.Context, iban *domain.KnownIBAN) error
	GetByIBAN(ctx context.Context, iban string) (*domain.KnownIBAN, error)
	IsBlacklisted(ctx context.Context, iban string) (bool, error)
	IsTrusted(ctx context.Context, iban string) (bool, error)
	ListByStatus(ctx context.Context, status domain.KnownIBANStatus, offset, limit int) ([]domain.KnownIBAN, int64, error)
	Delete(ctx context.Context, iban string) error
	UpdateChildLink(ctx context.Context, iban string, childID *uuid.UUID) error
	GetBlacklistedIBANs(ctx context.Context) (map[string]bool, error)
}

// HouseholdRepository handles household persistence.
type HouseholdRepository interface {
	List(ctx context.Context, search string, sortBy string, sortDir string, offset, limit int) ([]domain.Household, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Household, error)
	Create(ctx context.Context, household *domain.Household) error
	Update(ctx context.Context, household *domain.Household) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetParents(ctx context.Context, householdID uuid.UUID) ([]domain.Parent, error)
	GetChildren(ctx context.Context, householdID uuid.UUID) ([]domain.Child, error)
	GetWithMembers(ctx context.Context, id uuid.UUID) (*domain.Household, error)
}

// MemberRepository handles member persistence.
type MemberRepository interface {
	List(ctx context.Context, activeOnly bool, search string, sortBy string, sortDir string, offset, limit int) ([]domain.Member, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Member, error)
	GetByMemberNumber(ctx context.Context, memberNumber string) (*domain.Member, error)
	Create(ctx context.Context, member *domain.Member) error
	Update(ctx context.Context, member *domain.Member) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListActiveAt(ctx context.Context, date time.Time) ([]domain.Member, error)
	GetNextMemberNumber(ctx context.Context) (string, error)
}

// WarningRepository handles transaction warning persistence.
type WarningRepository interface {
	Create(ctx context.Context, warning *domain.TransactionWarning) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.TransactionWarning, error)
	GetByTransactionID(ctx context.Context, transactionID uuid.UUID) (*domain.TransactionWarning, error)
	ListUnresolved(ctx context.Context, offset, limit int) ([]domain.TransactionWarning, int64, error)
	Resolve(ctx context.Context, id uuid.UUID, resolvedBy uuid.UUID, resolutionType domain.ResolutionType, note string) error
	ResolveByTransactionID(ctx context.Context, transactionID uuid.UUID, resolutionType domain.ResolutionType, note string) error
	Delete(ctx context.Context, id uuid.UUID) error
}
