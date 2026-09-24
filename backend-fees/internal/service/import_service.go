package service

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/csvparser"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

const (
	// Late payment threshold: 15th day of the month
	latePaymentDayThreshold = 15

	// Confidence thresholds
	autoMatchConfidenceThreshold = 0.95
	memberNumberConfidence       = 0.95
	trustedIBANConfidence        = 0.99
	confidenceBoostCombined      = 0.02
	confidenceBoostNameMatch     = 0.05
	maxConfidenceNameMatch       = 0.93
	maxConfidenceCombined        = 0.99
)

// ImportService handles CSV import and matching logic.
type ImportService struct {
	transactionRepo repository.TransactionRepository
	feeRepo         repository.FeeRepository
	childRepo       repository.ChildRepository
	matchRepo       repository.MatchRepository
	knownIBANRepo   repository.KnownIBANRepository
	warningRepo     repository.WarningRepository
	txm             *repository.TxManager
	scheduleRepo    repository.FeeScheduleRepository
}

// NewImportService creates a new import service.
func NewImportService(
	transactionRepo repository.TransactionRepository,
	feeRepo repository.FeeRepository,
	childRepo repository.ChildRepository,
	matchRepo repository.MatchRepository,
	knownIBANRepo repository.KnownIBANRepository,
	warningRepo repository.WarningRepository,
	txm *repository.TxManager,
	scheduleRepo repository.FeeScheduleRepository,
) *ImportService {
	return &ImportService{
		transactionRepo: transactionRepo,
		feeRepo:         feeRepo,
		childRepo:       childRepo,
		matchRepo:       matchRepo,
		knownIBANRepo:   knownIBANRepo,
		warningRepo:     warningRepo,
		txm:             txm,
		scheduleRepo:    scheduleRepo,
	}
}

// ImportResult represents the result of a CSV import.
type ImportResult struct {
	BatchID     uuid.UUID                   `json:"batchId"`
	FileName    string                      `json:"fileName"`
	TotalRows   int                         `json:"totalRows"`
	Imported    int                         `json:"imported"`
	AutoMatched int                         `json:"autoMatched"`
	Skipped     int                         `json:"skipped"`
	Blacklisted int                         `json:"blacklisted"`
	Warnings    int                         `json:"warnings"`
	Suggestions []domain.MatchSuggestion    `json:"suggestions"`
	WarningList []domain.TransactionWarning `json:"warningList,omitempty" binding:"optional"`
	// Errors lists rows that could not be saved or processed; they are also
	// stored on the import batch.
	Errors []domain.ImportError `json:"errors"`
}

// RescanResult represents the result of rescanning unmatched transactions.
type RescanResult struct {
	Scanned     int                      `json:"scanned"`
	AutoMatched int                      `json:"autoMatched"`
	Suggestions []domain.MatchSuggestion `json:"suggestions"`
	Errors      []domain.ImportError     `json:"errors"`
}

// ProcessCSV processes a CSV file and returns match suggestions.
// Loading the reference data (blacklist, children) or creating the batch
// fails the whole import. Failures of single rows are collected in
// ImportResult.Errors and stored on the batch.
func (s *ImportService) ProcessCSV(ctx context.Context, file io.Reader, fileName string, userID uuid.UUID) (*ImportResult, error) {
	// Parse CSV
	transactions, err := csvparser.ParseBankCSV(file)
	if err != nil {
		return nil, err
	}

	blacklistedIBANs, err := s.knownIBANRepo.GetBlacklistedIBANs(ctx)
	if err != nil {
		return nil, fmt.Errorf("load IBAN blacklist: %w", err)
	}

	children, err := s.loadMatchingChildren(ctx)
	if err != nil {
		return nil, err
	}

	schedules, err := s.scheduleRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("load fee schedules: %w", err)
	}

	batchID := uuid.New()
	result := &ImportResult{
		BatchID:   batchID,
		FileName:  fileName,
		TotalRows: len(transactions),
		Errors:    []domain.ImportError{},
	}

	if err := s.transactionRepo.CreateBatch(ctx, batchID, fileName, userID); err != nil {
		return nil, fmt.Errorf("create import batch: %w", err)
	}

	for _, tx := range transactions {
		// Only process incoming payments
		if tx.Amount <= 0 {
			result.Skipped++
			continue
		}

		// Skip blacklisted IBANs
		if tx.PayerIBAN != nil && blacklistedIBANs[*tx.PayerIBAN] {
			result.Blacklisted++
			continue
		}

		tx.ImportBatchID = &batchID

		exists, err := s.transactionRepo.Exists(ctx, tx.BookingDate, tx.PayerIBAN, tx.Amount, tx.Description)
		if err != nil {
			result.Errors = append(result.Errors, domain.NewImportError(tx, "Duplikatprüfung fehlgeschlagen: "+err.Error()))
			continue
		}
		if exists {
			result.Skipped++
			continue
		}

		if err := s.transactionRepo.Create(ctx, &tx); err != nil {
			if errors.Is(err, repository.ErrDuplicate) {
				// Inserted concurrently since the Exists check (e.g. parallel import).
				result.Skipped++
				continue
			}
			result.Errors = append(result.Errors, domain.NewImportError(tx, "Buchung konnte nicht gespeichert werden: "+err.Error()))
			continue
		}
		result.Imported++

		// Try to match
		suggestion, warning := s.matchTransaction(ctx, tx, children, schedules)
		if suggestion != nil {
			// High confidence with matching fee expectation(s) -> auto-confirm
			if suggestion.Confidence >= autoMatchConfidenceThreshold && (suggestion.Expectation != nil || len(suggestion.Expectations) > 0) {
				err := s.autoConfirmMatch(ctx, suggestion)
				if err == nil {
					result.AutoMatched++
					continue
				}
				result.Errors = append(result.Errors, domain.NewImportError(tx, "Automatische Zuordnung fehlgeschlagen, bleibt als Vorschlag: "+err.Error()))
			}

			result.Suggestions = append(result.Suggestions, *suggestion)
		} else if warning != nil {
			s.saveWarning(ctx, tx, warning, result)
		} else {
			// No match - check if trusted IBAN needs a warning
			if w := s.checkForWarning(ctx, tx, schedules); w != nil {
				s.saveWarning(ctx, tx, w, result)
			}
		}
	}

	if len(result.Errors) > 0 {
		if err := s.transactionRepo.SetBatchErrors(ctx, batchID, result.Errors); err != nil {
			return nil, fmt.Errorf("store import errors: %w", err)
		}
	}

	return result, nil
}

// loadMatchingChildren returns all active children with their parents for matching.
func (s *ImportService) loadMatchingChildren(ctx context.Context) ([]domain.Child, error) {
	const pageSize = 500
	var children []domain.Child
	for offset := 0; ; offset += pageSize {
		page, _, err := s.childRepo.List(ctx, true, false, false, false, "", "", "", offset, pageSize)
		if err != nil {
			return nil, fmt.Errorf("load children: %w", err)
		}
		children = append(children, page...)
		if len(page) < pageSize {
			break
		}
	}
	if err := s.enrichChildrenWithParents(ctx, children); err != nil {
		return nil, fmt.Errorf("load parents: %w", err)
	}
	return children, nil
}

func (s *ImportService) saveWarning(ctx context.Context, tx domain.BankTransaction, warning *domain.TransactionWarning, result *ImportResult) {
	// Add to result list for frontend display during import
	result.WarningList = append(result.WarningList, *warning)

	// Persist to database except for MULTIPLE_OPEN_FEES (computed on-the-fly)
	if s.warningRepo != nil && warning.WarningType != domain.WarningTypeMultipleOpenFees {
		if err := s.warningRepo.Create(ctx, warning); err != nil {
			result.Errors = append(result.Errors, domain.NewImportError(tx, "Warnung konnte nicht gespeichert werden: "+err.Error()))
			return
		}
		result.Warnings++
	}
}

// GetHistory returns import batch history.
func (s *ImportService) GetHistory(ctx context.Context, offset, limit int) ([]domain.ImportBatch, int64, error) {
	return s.transactionRepo.GetBatches(ctx, offset, limit)
}

// Rescan re-scans all unmatched transactions for potential matches.
// High-confidence matches (95%+) are automatically confirmed.
func (s *ImportService) Rescan(ctx context.Context) (*RescanResult, error) {
	result := &RescanResult{Errors: []domain.ImportError{}}

	// Get all unmatched transactions (no search/sort, just get all)
	transactions, _, err := s.transactionRepo.ListUnmatched(ctx, "", "date", "desc", 0, 10000)
	if err != nil {
		return nil, err
	}

	children, err := s.loadMatchingChildren(ctx)
	if err != nil {
		return nil, err
	}

	schedules, err := s.scheduleRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("load fee schedules: %w", err)
	}

	// Re-scan each transaction
	for _, tx := range transactions {
		result.Scanned++
		suggestion, warning := s.matchTransaction(ctx, tx, children, schedules)

		if warning != nil {
			if s.warningRepo != nil {
				if err := s.warningRepo.Create(ctx, warning); err != nil {
					result.Errors = append(result.Errors, domain.NewImportError(tx, "Warnung konnte nicht gespeichert werden: "+err.Error()))
				}
			}
			continue
		}

		if suggestion == nil {
			continue
		}

		// High confidence with matching fee expectation(s) -> auto-confirm
		if suggestion.Confidence >= autoMatchConfidenceThreshold && (suggestion.Expectation != nil || len(suggestion.Expectations) > 0) {
			err := s.autoConfirmMatch(ctx, suggestion)
			if err == nil {
				result.AutoMatched++
				continue
			}
			result.Errors = append(result.Errors, domain.NewImportError(tx, "Automatische Zuordnung fehlgeschlagen, bleibt als Vorschlag: "+err.Error()))
		}

		result.Suggestions = append(result.Suggestions, *suggestion)
	}

	return result, nil
}

func stringPtr(s string) *string {
	return &s
}
