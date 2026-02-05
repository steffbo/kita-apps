package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

// Common repository errors
var (
	ErrNotFound = errors.New("not found")
)

// UserRepository handles user persistence.
type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
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
	List(ctx context.Context, activeOnly bool, u3Only bool, hasWarnings bool, hasOpenFees bool, search string, sortBy string, sortDir string, offset, limit int) ([]domain.Child, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Child, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Child, error)
	GetByMemberNumber(ctx context.Context, memberNumber string) (*domain.Child, error)
	GetNextMemberNumber(ctx context.Context) (string, error)
	Create(ctx context.Context, child *domain.Child) error
	Update(ctx context.Context, child *domain.Child) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetParents(ctx context.Context, childID uuid.UUID) ([]domain.Parent, error)
	GetParentsForChildren(ctx context.Context, childIDs []uuid.UUID) (map[uuid.UUID][]domain.Parent, error)
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
	Status  string
	ChildID *uuid.UUID
	Search  string // Search by member number or child name
	SortBy  string
	SortDir string
}

// FeeRepository handles fee expectation persistence.
type FeeRepository interface {
	List(ctx context.Context, filter FeeFilter, offset, limit int) ([]domain.FeeExpectation, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.FeeExpectation, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.FeeExpectation, error)
	Create(ctx context.Context, fee *domain.FeeExpectation) error
	Update(ctx context.Context, fee *domain.FeeExpectation) error
	Delete(ctx context.Context, id uuid.UUID) error
	Exists(ctx context.Context, childID uuid.UUID, feeType domain.FeeType, year int, month *int) (bool, error)
	FindUnpaid(ctx context.Context, childID uuid.UUID, feeType domain.FeeType, year int, month *int) (*domain.FeeExpectation, error)
	FindOldestUnpaid(ctx context.Context, childID uuid.UUID, feeType domain.FeeType, amount float64) (*domain.FeeExpectation, error)
	// FindBestUnpaid finds the best matching unpaid fee, preferring fees for the payment month.
	// If a fee exists for the same month/year as paymentDate, it is preferred over older fees.
	FindBestUnpaid(ctx context.Context, childID uuid.UUID, feeType domain.FeeType, amount float64, paymentDate time.Time) (*domain.FeeExpectation, error)
	FindOldestUnpaidWithReminder(ctx context.Context, childID uuid.UUID, feeType domain.FeeType, combinedAmount float64) ([]domain.FeeExpectation, error)
	// CountUnpaidByType counts all unpaid fees of a specific type for a child.
	// Used to determine if auto-matching should occur (only when count == 1).
	CountUnpaidByType(ctx context.Context, childID uuid.UUID, feeType domain.FeeType, amount float64) (int, error)
	GetOverview(ctx context.Context, year int) (*domain.FeeOverview, error)
	// GetForChild retrieves all fee expectations for a child, optionally filtered by year.
	GetForChild(ctx context.Context, childID uuid.UUID, year *int) ([]domain.FeeExpectation, error)
	// ListUnpaidByMonthAndTypes returns unpaid fees for a specific year/month and fee types.
	ListUnpaidByMonthAndTypes(ctx context.Context, year int, month int, feeTypes []domain.FeeType) ([]domain.FeeExpectation, error)
	// ListUnpaidWithoutReminderByMonthAndTypes returns unpaid fees without reminders for a specific year/month and fee types.
	ListUnpaidWithoutReminderByMonthAndTypes(ctx context.Context, year int, month int, feeTypes []domain.FeeType) ([]domain.FeeExpectation, error)
}

// SettingsRepository handles app settings persistence.
type SettingsRepository interface {
	Get(ctx context.Context, key string) (*domain.AppSetting, error)
	Upsert(ctx context.Context, setting *domain.AppSetting) error
}

// EmailLogRepository handles email log persistence.
type EmailLogRepository interface {
	Create(ctx context.Context, log *domain.EmailLog) error
	List(ctx context.Context, offset, limit int) ([]domain.EmailLog, int64, error)
}

// TransactionRepository handles bank transaction persistence.
type TransactionRepository interface {
	Create(ctx context.Context, tx *domain.BankTransaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.BankTransaction, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.BankTransaction, error)
	Exists(ctx context.Context, bookingDate time.Time, payerIBAN *string, amount float64, description *string) (bool, error)
	ListUnmatched(ctx context.Context, search, sortBy, sortDir string, offset, limit int) ([]domain.BankTransaction, int64, error)
	ListMatched(ctx context.Context, search, sortBy, sortDir string, offset, limit int) ([]domain.BankTransaction, int64, error)
	GetBatches(ctx context.Context, offset, limit int) ([]domain.ImportBatch, int64, error)
	CreateBatch(ctx context.Context, id uuid.UUID, fileName string, importedBy uuid.UUID) error
	Hide(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
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
	GetAllByExpectation(ctx context.Context, expectationID uuid.UUID) ([]domain.PaymentMatch, error)
	GetTotalMatchedAmount(ctx context.Context, expectationID uuid.UUID) (float64, error)
	GetByTransactionIDs(ctx context.Context, transactionIDs []uuid.UUID) (map[uuid.UUID][]domain.PaymentMatch, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByTransactionID(ctx context.Context, transactionID uuid.UUID) (int64, error)
}

// KnownIBANRepository handles known IBAN persistence.
type KnownIBANRepository interface {
	Create(ctx context.Context, iban *domain.KnownIBAN) error
	GetByIBAN(ctx context.Context, iban string) (*domain.KnownIBAN, error)
	IsBlacklisted(ctx context.Context, iban string) (bool, error)
	IsTrusted(ctx context.Context, iban string) (bool, error)
	ListByStatus(ctx context.Context, status domain.KnownIBANStatus, offset, limit int) ([]domain.KnownIBAN, int64, error)
	ListTrustedByChildWithCounts(ctx context.Context, childID uuid.UUID) ([]domain.KnownIBANSummary, error)
	Delete(ctx context.Context, iban string) error
	UpdateChildLink(ctx context.Context, iban string, childID *uuid.UUID) error
	GetBlacklistedIBANs(ctx context.Context) (map[string]bool, error)
}

// HouseholdRepository handles household persistence.
type HouseholdRepository interface {
	List(ctx context.Context, search string, sortBy string, sortDir string, offset, limit int) ([]domain.Household, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Household, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Household, error)
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
