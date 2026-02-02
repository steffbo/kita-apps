package banking

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/banking/encrypt"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/banking/fints"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/rs/zerolog/log"
)

// SyncResult represents the result of a synchronization operation.
type SyncResult struct {
	TransactionsFetched  int
	TransactionsImported int
	TransactionsSkipped  int
	Errors               []string
	LastSyncAt           time.Time
}

// Service handles banking synchronization operations.
type Service struct {
	configRepo      repository.BankingConfigRepository
	transactionRepo repository.TransactionRepository
	childRepo       repository.ChildRepository
	feeRepo         repository.FeeRepository
	matchRepo       repository.MatchRepository
	knownIBANRepo   repository.KnownIBANRepository
	warningRepo     repository.WarningRepository
	encryptor       *encrypt.Encryptor
}

// NewService creates a new banking service.
func NewService(
	configRepo repository.BankingConfigRepository,
	transactionRepo repository.TransactionRepository,
	childRepo repository.ChildRepository,
	feeRepo repository.FeeRepository,
	matchRepo repository.MatchRepository,
	knownIBANRepo repository.KnownIBANRepository,
	warningRepo repository.WarningRepository,
	encryptor *encrypt.Encryptor,
) *Service {
	return &Service{
		configRepo:      configRepo,
		transactionRepo: transactionRepo,
		childRepo:       childRepo,
		feeRepo:         feeRepo,
		matchRepo:       matchRepo,
		knownIBANRepo:   knownIBANRepo,
		warningRepo:     warningRepo,
		encryptor:       encryptor,
	}
}

// Sync performs a synchronization with the bank.
// It fetches transactions since the last sync and imports them.
func (s *Service) Sync(ctx context.Context) (*SyncResult, error) {
	result := &SyncResult{
		LastSyncAt: time.Now(),
	}

	// Get banking configuration
	config, err := s.configRepo.Get(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to get banking config: %w", err)
	}

	if !config.IsConfigured() {
		return result, fmt.Errorf("banking not configured")
	}

	if !config.SyncEnabled {
		return result, fmt.Errorf("sync disabled")
	}

	// Decrypt PIN
	pin, err := s.encryptor.Decrypt(config.EncryptedPIN)
	if err != nil {
		return result, fmt.Errorf("failed to decrypt PIN: %w", err)
	}

	// Create FinTS client
	fintsConfig := fints.Config{
		BankCode:      config.BankBLZ,
		UserID:        config.UserID,
		PIN:           pin,
		FinTSURL:      config.FinTSURL,
		AccountNumber: config.AccountNumber,
	}
	client := fints.NewClient(fintsConfig)

	// Determine sync start date
	syncStart := time.Now().AddDate(0, 0, -90) // Default: last 90 days
	if config.LastSyncAt != nil {
		// Only fetch since last sync, but at least 1 day back to catch late-booked transactions
		syncStart = config.LastSyncAt.AddDate(0, 0, -1)
	}

	// Fetch transactions from bank
	log.Info().Time("since", syncStart).Msg("Fetching transactions from bank")
	transactions, err := client.FetchTransactions(ctx, syncStart)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	result.TransactionsFetched = len(transactions)
	log.Info().Int("count", len(transactions)).Msg("Fetched transactions from bank")

	// Get blacklisted IBANs for filtering
	blacklistedIBANs, err := s.knownIBANRepo.GetBlacklistedIBANs(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get blacklisted IBANs, continuing without blacklist")
		blacklistedIBANs = make(map[string]bool)
	}

	// Get all children for matching
	children, _, _ := s.childRepo.List(ctx, true, false, false, false, "", "", "", 0, 1000)
	s.enrichChildrenWithParents(ctx, children)

	// Process each transaction
	for _, tx := range transactions {
		// Only process incoming payments
		if tx.Amount <= 0 {
			result.TransactionsSkipped++
			continue
		}

		// Skip blacklisted IBANs
		if tx.PayerIBAN != nil && blacklistedIBANs[*tx.PayerIBAN] {
			result.TransactionsSkipped++
			continue
		}

		// Check for duplicates
		exists, _ := s.transactionRepo.Exists(ctx, tx.BookingDate, tx.PayerIBAN, tx.Amount, tx.Description)
		if exists {
			result.TransactionsSkipped++
			continue
		}

		// Save transaction
		tx.ID = uuid.New()
		tx.ImportedAt = time.Now()
		if err := s.transactionRepo.Create(ctx, &tx); err != nil {
			log.Error().Err(err).Interface("transaction", tx).Msg("Failed to create transaction")
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to save transaction: %v", err))
			continue
		}

		result.TransactionsImported++

		// Attempt to match the transaction
		s.matchTransaction(ctx, tx, children)
	}

	// Update last sync timestamp
	if err := s.configRepo.UpdateLastSync(ctx, result.LastSyncAt); err != nil {
		log.Error().Err(err).Msg("Failed to update last sync timestamp")
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to update sync timestamp: %v", err))
	}

	return result, nil
}

// TestConnection tests the connection to the bank without importing data.
func (s *Service) TestConnection(ctx context.Context) error {
	config, err := s.configRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get banking config: %w", err)
	}

	if !config.IsConfigured() {
		return fmt.Errorf("banking not configured")
	}

	// Decrypt PIN
	pin, err := s.encryptor.Decrypt(config.EncryptedPIN)
	if err != nil {
		return fmt.Errorf("failed to decrypt PIN: %w", err)
	}

	// Create FinTS client and test connection
	fintsConfig := fints.Config{
		BankCode:      config.BankBLZ,
		UserID:        config.UserID,
		PIN:           pin,
		FinTSURL:      config.FinTSURL,
		AccountNumber: config.AccountNumber,
	}
	client := fints.NewClient(fintsConfig)

	return client.TestConnection(ctx)
}

// GetStatus returns the current banking sync status.
func (s *Service) GetStatus(ctx context.Context) (*domain.SyncStatus, error) {
	config, err := s.configRepo.Get(ctx)
	if err != nil {
		return &domain.SyncStatus{
			IsConfigured: false,
		}, nil
	}

	// Count total transactions
	var totalCount int64
	// Use a simple query to count transactions - we could add this to the repo interface
	// For now, return 0

	return &domain.SyncStatus{
		LastSyncAt:        config.LastSyncAt,
		TransactionsCount: totalCount,
		IsConfigured:      config.IsConfigured(),
		SyncEnabled:       config.SyncEnabled,
	}, nil
}

// matchTransaction attempts to match a transaction to children and fees.
func (s *Service) matchTransaction(ctx context.Context, tx domain.BankTransaction, children []domain.Child) {
	// This is a simplified matching logic
	// In production, you'd use the same matching logic from the ImportService

	// Try to extract member number from description
	if tx.Description == nil {
		return
	}

	// Check for warnings if IBAN is trusted but no match found
	if tx.PayerIBAN != nil {
		knownIBAN, err := s.knownIBANRepo.GetByIBAN(ctx, *tx.PayerIBAN)
		if err == nil && knownIBAN != nil && knownIBAN.Status == domain.KnownIBANStatusTrusted && knownIBAN.ChildID != nil {
			// Trusted IBAN but no match - create warning
			warning := &domain.TransactionWarning{
				ID:            uuid.New(),
				TransactionID: tx.ID,
				WarningType:   domain.WarningTypeNoMatchingFee,
				Message:       fmt.Sprintf("Trusted IBAN payment of %.2f EUR - no matching fee found", tx.Amount),
				ActualAmount:  &tx.Amount,
				ChildID:       knownIBAN.ChildID,
				CreatedAt:     time.Now(),
			}
			s.warningRepo.Create(ctx, warning)
		}
	}
}

// enrichChildrenWithParents loads parents for all children.
func (s *Service) enrichChildrenWithParents(ctx context.Context, children []domain.Child) {
	if len(children) == 0 {
		return
	}

	childIDs := make([]uuid.UUID, len(children))
	for i := range children {
		childIDs[i] = children[i].ID
	}

	parentsMap, err := s.childRepo.GetParentsForChildren(ctx, childIDs)
	if err != nil {
		return
	}

	for i := range children {
		if parents, ok := parentsMap[children[i].ID]; ok {
			children[i].Parents = parents
		}
	}
}
