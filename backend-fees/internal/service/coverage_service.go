package service

import (
	"context"
	"sort"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

// CoverageService handles fee coverage calculations and timeline generation.
type CoverageService struct {
	feeRepo         repository.FeeRepository
	childRepo       repository.ChildRepository
	transactionRepo repository.TransactionRepository
	matchRepo       repository.MatchRepository
}

// NewCoverageService creates a new coverage service.
func NewCoverageService(
	feeRepo repository.FeeRepository,
	childRepo repository.ChildRepository,
	transactionRepo repository.TransactionRepository,
	matchRepo repository.MatchRepository,
) *CoverageService {
	return &CoverageService{
		feeRepo:         feeRepo,
		childRepo:       childRepo,
		transactionRepo: transactionRepo,
		matchRepo:       matchRepo,
	}
}

// GetChildTimeline returns a month-by-month coverage timeline for a child.
// Uses temporal proximity: transactions in March cover March's fees.
func (s *CoverageService) GetChildTimeline(ctx context.Context, childID uuid.UUID, year int) ([]domain.FeeCoverage, error) {
	// Get all fees for the child in the year
	fees, err := s.feeRepo.GetForChild(ctx, childID, &year)
	if err != nil {
		return nil, err
	}

	// Get all matched transactions for this child
	matchedTxs, err := s.getMatchedTransactionsForChild(ctx, childID, fees)
	if err != nil {
		return nil, err
	}

	// Group fees by month
	feesByMonth := make(map[int][]domain.FeeExpectation)
	for _, fee := range fees {
		month := 0
		if fee.Month != nil {
			month = *fee.Month
		}
		feesByMonth[month] = append(feesByMonth[month], fee)
	}

	// Calculate coverage for each month
	var timeline []domain.FeeCoverage
	for month := 1; month <= 12; month++ {
		monthFees := feesByMonth[month]
		if len(monthFees) == 0 {
			continue // Skip months with no fees
		}

		coverage := s.calculateMonthCoverage(childID, year, month, monthFees, matchedTxs)
		timeline = append(timeline, coverage)
	}

	// Sort by month
	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].Month < timeline[j].Month
	})

	return timeline, nil
}

// getMatchedTransactionsForChild retrieves all transactions matched to a child's fees.
func (s *CoverageService) getMatchedTransactionsForChild(
	ctx context.Context,
	childID uuid.UUID,
	fees []domain.FeeExpectation,
) ([]domain.BankTransaction, error) {
	// Get all matches for these fees
	var matchedTxs []domain.BankTransaction
	seenTxs := make(map[uuid.UUID]bool)

	for _, fee := range fees {
		matches, err := s.matchRepo.GetAllByExpectation(ctx, fee.ID)
		if err != nil {
			continue
		}

		for _, match := range matches {
			if seenTxs[match.TransactionID] {
				continue
			}
			seenTxs[match.TransactionID] = true

			// Get transaction details
			tx, err := s.transactionRepo.GetByID(ctx, match.TransactionID)
			if err != nil {
				continue
			}
			matchedTxs = append(matchedTxs, *tx)
		}
	}

	return matchedTxs, nil
}

// calculateMonthCoverage calculates coverage for a specific month using temporal matching.
// A transaction covers the month's fees if it was received in that month (not based on fee month).
func (s *CoverageService) calculateMonthCoverage(
	childID uuid.UUID,
	year, month int,
	fees []domain.FeeExpectation,
	allTransactions []domain.BankTransaction,
) domain.FeeCoverage {
	coverage := domain.FeeCoverage{
		ChildID:      childID,
		Year:         year,
		Month:        month,
		Transactions: []domain.CoveredTransaction{},
	}

	// Calculate expected total for this month
	for _, fee := range fees {
		coverage.ExpectedTotal += fee.Amount
	}

	// Find transactions that arrived IN this month (temporal matching)
	var monthTransactions []domain.BankTransaction
	var otherTransactions []domain.BankTransaction

	for _, tx := range allTransactions {
		txMonth := int(tx.BookingDate.Month())
		txYear := tx.BookingDate.Year()

		// Transaction covers this month if it arrived in this month
		if txYear == year && txMonth == month {
			monthTransactions = append(monthTransactions, tx)
		} else {
			// Transaction from a different month - could be used for shortfall
			otherTransactions = append(otherTransactions, tx)
		}
	}

	// Sort by date (oldest first for FIFO matching)
	sort.Slice(monthTransactions, func(i, j int) bool {
		return monthTransactions[i].BookingDate.Before(monthTransactions[j].BookingDate)
	})
	sort.Slice(otherTransactions, func(i, j int) bool {
		return otherTransactions[i].BookingDate.Before(otherTransactions[j].BookingDate)
	})

	// Apply transactions that arrived in this month first
	remaining := coverage.ExpectedTotal
	for _, tx := range monthTransactions {
		if remaining <= 0 {
			break
		}

		applied := tx.Amount
		if applied > remaining {
			applied = remaining // Cap at what's needed
		}

		coverage.ReceivedTotal += applied
		remaining -= applied

		desc := ""
		if tx.Description != nil {
			desc = *tx.Description
		}

		coverage.Transactions = append(coverage.Transactions, domain.CoveredTransaction{
			TransactionID:  tx.ID,
			Amount:         applied,
			BookingDate:    tx.BookingDate,
			Description:    &desc,
			IsForThisMonth: true,
		})
	}

	// If still short after using this month's transactions, use older transactions
	// This handles cases where someone pays for multiple months at once
	for _, tx := range otherTransactions {
		if remaining <= 0 {
			break
		}

		applied := tx.Amount
		if applied > remaining {
			applied = remaining
		}

		coverage.ReceivedTotal += applied
		remaining -= applied

		desc := ""
		if tx.Description != nil {
			desc = *tx.Description
		}

		coverage.Transactions = append(coverage.Transactions, domain.CoveredTransaction{
			TransactionID:  tx.ID,
			Amount:         applied,
			BookingDate:    tx.BookingDate,
			Description:    &desc,
			IsForThisMonth: false, // Mark as from different month
		})
	}

	// Calculate final balance and status
	coverage.Balance = coverage.ExpectedTotal - coverage.ReceivedTotal

	switch {
	case coverage.ReceivedTotal == 0:
		coverage.Status = domain.CoverageStatusUnpaid
	case coverage.ReceivedTotal < coverage.ExpectedTotal:
		coverage.Status = domain.CoverageStatusPartial
	case coverage.ReceivedTotal == coverage.ExpectedTotal:
		coverage.Status = domain.CoverageStatusCovered
	default:
		coverage.Status = domain.CoverageStatusOverpaid
	}

	return coverage
}

// ChildBalance represents a child's overall financial balance.
type ChildBalance struct {
	ChildID       uuid.UUID `json:"childId"`
	FirstName     string    `json:"firstName"`
	LastName      string    `json:"lastName"`
	TotalExpected float64   `json:"totalExpected"`
	TotalReceived float64   `json:"totalReceived"`
	Balance       float64   `json:"balance"` // positive = owes money, negative = credit
	OpenMonths    int       `json:"openMonths"`
}
