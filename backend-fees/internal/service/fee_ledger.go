package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
	"github.com/rs/zerolog/log"
	"sort"
	"time"
)

// ChildcareMonth is the CHILDCARE amount owed for one month.
type ChildcareMonth struct {
	Year   int
	Month  int
	Amount float64
}

// SyncChildcareExpectations aligns the CHILDCARE expectations of already
// generated months with the given amounts. A month counts as generated when the
// child has a CHILDCARE or FOOD expectation for it; later months are left to
// fee generation. Paid months that would become overpaid are not changed but
// reported for a credit review; unpaid expectations without an amount are removed.
func (s *FeeService) SyncChildcareExpectations(ctx context.Context, childID uuid.UUID, householdID *uuid.UUID, months []ChildcareMonth) (*ChildcareExpectationSyncResult, error) {
	result := &ChildcareExpectationSyncResult{CreditReviewRequired: []CreditReviewPeriod{}}
	for _, m := range months {
		month := m.Month
		amount := roundToTwoDecimals(m.Amount)
		dueDate := time.Date(m.Year, time.Month(month), 5, 0, 0, 0, 0, time.UTC)
		fee, err := s.feeRepo.GetByChildFeePeriod(ctx, childID, domain.FeeTypeChildcare, m.Year, month)
		if err != nil {
			return nil, err
		}

		if fee == nil {
			if amount <= 0 {
				continue
			}
			generated, err := s.feeRepo.Exists(ctx, childID, domain.FeeTypeFood, m.Year, &month)
			if err != nil {
				return nil, err
			}
			if !generated {
				continue
			}
			created := &domain.FeeExpectation{
				ID:          uuid.New(),
				ChildID:     childID,
				HouseholdID: householdID,
				FeeType:     domain.FeeTypeChildcare,
				Year:        m.Year,
				Month:       &month,
				Amount:      amount,
				DueDate:     dueDate,
				CreatedAt:   util.Now(),
			}
			if err := s.feeRepo.Create(ctx, created); err != nil {
				return nil, err
			}
			result.Created++
			result.DeltaOpen = roundToTwoDecimals(result.DeltaOpen + amount)
			continue
		}

		matchedAmount, err := s.matchRepo.GetTotalMatchedAmount(ctx, fee.ID)
		if err != nil {
			return nil, err
		}
		oldRemaining := fee.Amount - matchedAmount
		if oldRemaining < 0 {
			oldRemaining = 0
		}

		if domain.Cents(matchedAmount) > domain.Cents(amount)+domain.PaymentToleranceCents {
			result.CreditReviewRequired = append(result.CreditReviewRequired, CreditReviewPeriod{
				FeeID:         fee.ID,
				Year:          fee.Year,
				Month:         month,
				OldAmount:     fee.Amount,
				NewAmount:     amount,
				MatchedAmount: matchedAmount,
				CreditAmount:  roundToTwoDecimals(matchedAmount - amount),
			})
			result.Skipped++
			continue
		}

		if abs64(domain.Cents(fee.Amount)-domain.Cents(amount)) <= domain.PaymentToleranceCents {
			result.Skipped++
			continue
		}

		if amount <= 0 && matchedAmount == 0 {
			if err := s.feeRepo.Delete(ctx, fee.ID); err != nil {
				return nil, err
			}
			result.Removed++
			result.DeltaOpen = roundToTwoDecimals(result.DeltaOpen - oldRemaining)
			continue
		}

		fee.Amount = amount
		fee.DueDate = dueDate
		if err := s.feeRepo.Update(ctx, fee); err != nil {
			return nil, err
		}
		newRemaining := amount - matchedAmount
		if newRemaining < 0 {
			newRemaining = 0
		}
		result.Updated++
		result.DeltaOpen = roundToTwoDecimals(result.DeltaOpen + newRemaining - oldRemaining)
	}

	return result, nil
}

// einstufungChildcareFee is the monthly childcare fee of a U3 child from the
// Einstufung valid in the month, before the entry-month rule. ok is false when
// no Einstufung covers the month; household data then decides.
func (s *FeeService) einstufungChildcareFee(ctx context.Context, schedule *domain.FeeSchedule, child *domain.Child, year int, month time.Month) (fee float64, ok bool) {
	if s.einstufungRepo == nil {
		return 0, false
	}
	e, err := s.einstufungRepo.GetForChildAt(ctx, child.ID, time.Date(year, month, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			log.Warn().Err(err).Str("childId", child.ID.String()).Msg("cannot load Einstufung for fee calculation")
		}
		return 0, false
	}
	var household *domain.Household
	if s.householdRepo != nil {
		household, _ = s.householdRepo.GetByID(ctx, e.HouseholdID)
	}
	m := int(month)
	careHours := s.ResolveCareHours(ctx, child, year, &m)
	return schedule.Config.CalculateChildcareFee(einstufungFeeInput(e, household, domain.ChildAgeTypeKrippe, careHours)).Fee, true
}

// calculateChildcareFeeForChild calculates the childcare fee for a specific child.
func (s *FeeService) calculateChildcareFeeForChild(ctx context.Context, schedule *domain.FeeSchedule, child *domain.Child, year int, month *int) float64 {
	// Only U3 children pay childcare fees
	// If child turns 3 at any point during the month, no childcare fee is charged
	if month != nil {
		if !child.IsUnderThreeForEntireMonth(year, time.Month(*month)) {
			return 0
		}
	} else {
		// No month specified, use current date check
		if !child.IsUnderThree(util.Today()) {
			return 0
		}
	}

	billed := util.Today()
	if month != nil {
		billed = time.Date(year, time.Month(*month), 1, 0, 0, 0, 0, time.UTC)
	}
	if fee, ok := s.einstufungChildcareFee(ctx, schedule, child, billed.Year(), billed.Month()); ok {
		return fee
	}
	return s.householdChildcareFee(ctx, schedule, child, year, month)
}

// householdChildcareFee calculates the childcare fee from household data, for
// months without an Einstufung.
func (s *FeeService) householdChildcareFee(ctx context.Context, schedule *domain.FeeSchedule, child *domain.Child, year int, month *int) float64 {
	info := s.getIncomeInfo(ctx, child)

	// Care hours effective in the billed month (falls back to the default when unknown)
	careHours := s.ResolveCareHours(ctx, child, year, month)

	feeResult := schedule.Config.CalculateChildcareFee(domain.ChildcareFeeInput{
		ChildAgeType:  domain.ChildAgeTypeKrippe,
		NetIncome:     info.Income,
		SiblingsCount: info.SiblingsCount,
		CareHours:     careHours,
		HighestRate:   info.IsHighestRate,
		FosterFamily:  info.IsFosterFamily,
	})

	return feeResult.Fee
}

// DefaultCareHours is used when no care hours are recorded for a child at all.
const DefaultCareHours = 45

// ResolveCareHours determines the weekly care hours that apply for the billed period.
//
// The care hours history is the single source of truth. The value carried on domain.Child
// only reflects the period that is effective today, so it is empty for children that have
// not started yet and must never be used for a different month.
func (s *FeeService) ResolveCareHours(ctx context.Context, child *domain.Child, year int, month *int) int {
	reference := util.Today()
	if month != nil {
		reference = time.Date(year, time.Month(*month), 1, 0, 0, 0, 0, time.UTC)
	}

	history, err := s.childRepo.ListCareHoursHistory(ctx, child.ID)
	if err == nil {
		if hours, ok := careHoursAt(history, reference); ok {
			return hours
		}
	}

	if child.CareHours != nil && *child.CareHours > 0 {
		return *child.CareHours
	}
	return DefaultCareHours
}

// careHoursAt returns the care hours effective at the reference date. If no period covers
// the reference date (e.g. the child starts later), the closest upcoming period is used.
func careHoursAt(history []domain.ChildCareHoursHistory, reference time.Time) (int, bool) {
	ref := time.Date(reference.Year(), reference.Month(), reference.Day(), 0, 0, 0, 0, time.UTC)

	var upcoming *domain.ChildCareHoursHistory
	for i := range history {
		entry := history[i]
		if entry.CareHours == nil || *entry.CareHours <= 0 {
			continue
		}
		from := truncateToDayUTC(entry.EffectiveFrom)
		if from.After(ref) {
			if upcoming == nil || from.Before(truncateToDayUTC(upcoming.EffectiveFrom)) {
				candidate := entry
				upcoming = &candidate
			}
			continue
		}
		if entry.EffectiveUntil != nil && truncateToDayUTC(*entry.EffectiveUntil).Before(ref) {
			continue
		}
		return *entry.CareHours, true
	}

	if upcoming != nil {
		return *upcoming.CareHours, true
	}
	return 0, false
}

func truncateToDayUTC(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// CreateReminder creates a reminder fee (Mahngebühr) for an unpaid fee.
// The reminder fee is 10 EUR and is linked to the original fee.
func (s *FeeService) CreateReminder(ctx context.Context, feeID uuid.UUID) (*domain.FeeExpectation, error) {
	// Get the original fee
	originalFee, err := s.feeRepo.GetByID(ctx, feeID)
	if err != nil {
		return nil, ErrNotFound
	}

	// Check if the original fee is paid
	match, _ := s.matchRepo.GetByExpectation(ctx, feeID)
	if match != nil {
		return nil, ErrInvalidInput // Cannot create reminder for paid fee
	}

	// Create the reminder fee
	now := util.Now()
	reminder := &domain.FeeExpectation{
		ID:            uuid.New(),
		ChildID:       originalFee.ChildID,
		HouseholdID:   originalFee.HouseholdID,
		FeeType:       domain.FeeTypeReminder,
		Year:          now.Year(),
		Month:         nil, // Reminders don't have a specific month
		Amount:        domain.ReminderFeeAmount,
		DueDate:       util.Today().AddDate(0, 0, 14), // Due in 14 days
		CreatedAt:     now,
		ReminderForID: &feeID,
	}

	if err := s.feeRepo.Create(ctx, reminder); err != nil {
		return nil, err
	}

	// Fetch and return with child info
	return s.GetByID(ctx, reminder.ID)
}

// LedgerEntry represents a single entry in the payment ledger.
type LedgerEntry struct {
	ID          uuid.UUID  `json:"id"`
	Date        time.Time  `json:"date"`        // Due date for fees, booking date for payments
	Type        string     `json:"type"`        // "fee" or "payment"
	Description string     `json:"description"` // e.g., "Essensgeld Januar 2024" or "Zahlung DE89..."
	FeeType     string     `json:"feeType,omitempty" binding:"optional"`
	Year        int        `json:"year,omitempty" binding:"optional"`
	Month       *int       `json:"month,omitempty" binding:"optional"`
	Debit       float64    `json:"debit"`   // Amount owed (fees)
	Credit      float64    `json:"credit"`  // Amount paid (payments)
	Balance     float64    `json:"balance"` // Running balance
	IsPaid      bool       `json:"isPaid,omitempty" binding:"optional"`
	PaidAt      *time.Time `json:"paidAt,omitempty" binding:"optional"`

	// Related objects
	Fee         *domain.FeeExpectation  `json:"fee,omitempty" binding:"optional"`
	Transaction *domain.BankTransaction `json:"transaction,omitempty" binding:"optional"`
}

// LedgerSummary provides totals for the ledger.
type LedgerSummary struct {
	TotalFees      float64 `json:"totalFees"`
	TotalPaid      float64 `json:"totalPaid"`
	TotalOpen      float64 `json:"totalOpen"`
	OpenFeesCount  int     `json:"openFeesCount"`
	PaidFeesCount  int     `json:"paidFeesCount"`
	TotalFeesCount int     `json:"totalFeesCount"`
}

// ChildLedger represents the complete payment ledger for a child.
type ChildLedger struct {
	ChildID uuid.UUID     `json:"childId"`
	Child   *domain.Child `json:"child,omitempty" binding:"optional"`
	Entries []LedgerEntry `json:"entries"`
	Summary LedgerSummary `json:"summary"`
}

// GetChildLedger returns the payment ledger for a specific child.
func (s *FeeService) GetChildLedger(ctx context.Context, childID uuid.UUID, year *int) (*ChildLedger, error) {
	// Verify child exists
	child, err := s.childRepo.GetByID(ctx, childID)
	if err != nil {
		return nil, ErrNotFound
	}

	// Get all fees for the child
	filter := repository.FeeFilter{
		ChildID: &childID,
		Year:    year,
	}
	fees, _, err := s.feeRepo.List(ctx, filter, 0, 1000)
	if err != nil {
		return nil, err
	}

	// Build ledger entries
	entries := make([]LedgerEntry, 0, len(fees)*2)
	var totalFees, totalPaid int64 // cents
	var openFeesCount, paidFeesCount int

	for _, fee := range fees {
		matches, _ := s.matchRepo.GetAllByExpectation(ctx, fee.ID)
		var paidAt *time.Time

		var totalMatchedCents int64
		for i := range matches {
			totalMatchedCents += domain.Cents(matches[i].Amount)
		}
		totalMatched := domain.Euros(totalMatchedCents)

		isPaid := domain.IsPaid(totalMatched, fee.Amount)
		if isPaid && len(matches) > 0 {
			paidAt = &matches[0].MatchedAt
			paidFeesCount++
		} else {
			openFeesCount++
		}

		paidCents := totalMatchedCents
		if feeCents := domain.Cents(fee.Amount); paidCents > feeCents {
			paidCents = feeCents
		}
		totalPaid += paidCents

		totalFees += domain.Cents(fee.Amount)

		// Build description
		description := util.FeeTypeToGerman(fee.FeeType)
		if fee.Month != nil {
			description += " " + util.MonthToGerman(*fee.Month)
		}
		description += fmt.Sprintf(" %d", fee.Year)

		// Fee entry
		feeEntry := LedgerEntry{
			ID:          fee.ID,
			Date:        fee.DueDate,
			Type:        "fee",
			Description: description,
			FeeType:     string(fee.FeeType),
			Year:        fee.Year,
			Month:       fee.Month,
			Debit:       fee.Amount,
			Credit:      0,
			IsPaid:      isPaid,
			PaidAt:      paidAt,
			Fee:         &fee,
		}
		entries = append(entries, feeEntry)

		// Add payment entries for each match
		for i := range matches {
			var transaction *domain.BankTransaction
			if s.transactionRepo != nil {
				tx, err := s.transactionRepo.GetByID(ctx, matches[i].TransactionID)
				if err == nil {
					transaction = tx
				}
			}
			if transaction == nil {
				continue
			}

			paymentDesc := "Zahlung"
			if transaction.PayerName != nil && *transaction.PayerName != "" {
				paymentDesc += " von " + *transaction.PayerName
			}
			if transaction.PayerIBAN != nil && *transaction.PayerIBAN != "" {
				iban := *transaction.PayerIBAN
				if len(iban) > 4 {
					paymentDesc += " (..." + iban[len(iban)-4:] + ")"
				}
			}

			paymentEntry := LedgerEntry{
				ID:          matches[i].ID,
				Date:        transaction.BookingDate,
				Type:        "payment",
				Description: paymentDesc,
				Debit:       0,
				Credit:      matches[i].Amount,
				Transaction: transaction,
			}
			entries = append(entries, paymentEntry)
		}
	}

	// Sort entries by date (oldest first)
	sortLedgerEntries(entries)

	// Calculate running balance
	var balance float64
	for i := range entries {
		balance += entries[i].Debit - entries[i].Credit
		entries[i].Balance = balance
	}

	ledger := &ChildLedger{
		ChildID: childID,
		Child:   child,
		Entries: entries,
		Summary: LedgerSummary{
			TotalFees:      domain.Euros(totalFees),
			TotalPaid:      domain.Euros(totalPaid),
			TotalOpen:      domain.Euros(totalFees - totalPaid),
			OpenFeesCount:  openFeesCount,
			PaidFeesCount:  paidFeesCount,
			TotalFeesCount: len(fees),
		},
	}

	return ledger, nil
}

// enrichWithPaymentStatus checks if a fee is paid and loads match details with transaction.
func (s *FeeService) enrichWithPaymentStatus(ctx context.Context, fee *domain.FeeExpectation) {
	matches, _ := s.matchRepo.GetAllByExpectation(ctx, fee.ID)
	if len(matches) == 0 {
		return
	}

	var totalMatchedCents int64
	for i := range matches {
		totalMatchedCents += domain.Cents(matches[i].Amount)
		if s.transactionRepo != nil {
			tx, err := s.transactionRepo.GetByID(ctx, matches[i].TransactionID)
			if err == nil {
				matches[i].Transaction = tx
			}
		}
	}

	fee.MatchedAmount = domain.Euros(totalMatchedCents)
	if domain.IsPaid(fee.MatchedAmount, fee.Amount) {
		fee.IsPaid = true
		paidAt := matches[0].MatchedAt
		fee.PaidAt = &paidAt
	} else {
		fee.IsPaid = false
	}
	remaining := domain.Cents(fee.Amount) - totalMatchedCents
	if remaining < 0 {
		remaining = 0
	}
	fee.Remaining = domain.Euros(remaining)
	fee.PartialMatches = matches
	fee.MatchedBy = &matches[0]
}

// sortLedgerEntries sorts entries by date (oldest first).
func sortLedgerEntries(entries []LedgerEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Date.Before(entries[j].Date)
	})
}
