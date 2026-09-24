package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/csvparser"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

// ChildUnmatchedSuggestionsResult represents likely unmatched transactions for a child.
type ChildUnmatchedSuggestionsResult struct {
	ChildID     uuid.UUID                `json:"childId"`
	Scanned     int                      `json:"scanned"`
	Suggestions []domain.MatchSuggestion `json:"suggestions"`
}

func (s *ImportService) matchTransaction(ctx context.Context, tx domain.BankTransaction, children []domain.Child, schedules domain.FeeSchedules) (*domain.MatchSuggestion, *domain.TransactionWarning) {
	suggestion := &domain.MatchSuggestion{
		Transaction:  tx,
		DetectedType: detectFeeType(tx.Amount, feeConfigAt(schedules, tx.BookingDate)),
	}

	matchText := buildMatchText(tx)

	s.matchTrustedIBAN(ctx, tx, children, suggestion)
	s.overrideTrustedIBANForSiblingHint(matchText, children, suggestion)
	if suggestion.Child == nil {
		s.matchChild(matchText, children, suggestion)
	}

	if suggestion.Child != nil && suggestion.DetectedType != nil {
		if warning := s.matchFeeExpectation(ctx, tx, suggestion); warning != nil {
			return nil, warning
		}
	}

	// Boost confidence for name-based matches with fee expectations
	if suggestion.Expectation != nil && (suggestion.MatchedBy == "name" || suggestion.MatchedBy == "parent_name") {
		suggestion.Confidence = min(suggestion.Confidence+confidenceBoostNameMatch, maxConfidenceNameMatch)
	}

	if suggestion.Confidence > 0 {
		return suggestion, nil
	}
	return nil, nil
}

func (s *ImportService) matchTrustedIBAN(ctx context.Context, tx domain.BankTransaction, children []domain.Child, suggestion *domain.MatchSuggestion) {
	if s.knownIBANRepo == nil || tx.PayerIBAN == nil {
		return
	}

	knownIBAN, err := s.knownIBANRepo.GetByIBAN(ctx, *tx.PayerIBAN)
	if err != nil || knownIBAN == nil || knownIBAN.Status != domain.KnownIBANStatusTrusted || knownIBAN.ChildID == nil {
		return
	}

	childID := *knownIBAN.ChildID
	for i := range children {
		if children[i].ID == childID {
			suggestion.Child = &children[i]
			suggestion.MatchedBy = "trusted_iban"
			suggestion.Confidence = trustedIBANConfidence
			return
		}
	}

	if s.childRepo != nil {
		child, err := s.childRepo.GetByID(ctx, childID)
		if err == nil && child != nil {
			suggestion.Child = child
			suggestion.MatchedBy = "trusted_iban"
			suggestion.Confidence = trustedIBANConfidence
		}
	}
}

func (s *ImportService) overrideTrustedIBANForSiblingHint(matchText string, children []domain.Child, suggestion *domain.MatchSuggestion) {
	if suggestion.MatchedBy != "trusted_iban" || suggestion.Child == nil {
		return
	}

	trustedChild := suggestion.Child
	siblings := findSiblings(*trustedChild, children)
	if len(siblings) == 0 {
		return
	}

	memberNumber := csvparser.ExtractMemberNumber(matchText)
	if memberNumber != "" {
		for i := range siblings {
			if siblings[i].MemberNumber != memberNumber {
				continue
			}

			suggestion.Child = &siblings[i]
			suggestion.MatchedBy = "member_number"
			suggestion.Confidence = memberNumberConfidence
			return
		}
	}

	nameMatchedSibling, confidence := csvparser.MatchChildByName(matchText, siblings)
	if nameMatchedSibling == nil || confidence < 0.8 {
		return
	}

	suggestion.Child = nameMatchedSibling
	suggestion.MatchedBy = "name"
	suggestion.Confidence = confidence
}

func findSiblings(child domain.Child, children []domain.Child) []domain.Child {
	if len(child.Parents) == 0 {
		return nil
	}

	parentIDs := make(map[uuid.UUID]struct{}, len(child.Parents))
	for _, parent := range child.Parents {
		parentIDs[parent.ID] = struct{}{}
	}

	siblings := make([]domain.Child, 0, len(children))
	for _, candidate := range children {
		if candidate.ID == child.ID {
			continue
		}

		for _, candidateParent := range candidate.Parents {
			if _, ok := parentIDs[candidateParent.ID]; ok {
				siblings = append(siblings, candidate)
				break
			}
		}
	}

	return siblings
}

// feeConfigAt returns the fee schedule config valid at date, or nil if none.
func feeConfigAt(schedules domain.FeeSchedules, date time.Time) *domain.FeeScheduleConfig {
	schedule, err := schedules.At(date)
	if err != nil {
		return nil
	}
	return &schedule.Config
}

// detectFeeType guesses the fee type from the amount using the fee schedule
// valid at the booking date. Unknown amounts are treated as childcare fees.
func detectFeeType(amount float64, cfg *domain.FeeScheduleConfig) *domain.FeeType {
	feeType := domain.FeeTypeChildcare
	if cfg != nil {
		switch amount {
		case cfg.MonthlyFoodFee, cfg.MonthlyFoodFee + domain.ReminderFeeAmount:
			feeType = domain.FeeTypeFood
		case cfg.AnnualMembershipFee, cfg.AnnualMembershipFee + domain.MembershipReminderFeeAmount:
			feeType = domain.FeeTypeMembership
		}
	}
	return &feeType
}

func (s *ImportService) matchChild(matchText string, children []domain.Child, suggestion *domain.MatchSuggestion) {
	// Try to match by member number first
	memberNumber := csvparser.ExtractMemberNumber(matchText)
	if memberNumber != "" {
		for i := range children {
			if children[i].MemberNumber == memberNumber {
				suggestion.Child = &children[i]
				suggestion.MatchedBy = "member_number"
				suggestion.Confidence = memberNumberConfidence
				return
			}
		}
	}

	// Try by child name
	matchedChild, confidence := csvparser.MatchChildByName(matchText, children)
	if matchedChild != nil {
		suggestion.Child = matchedChild
		suggestion.MatchedBy = "name"
		suggestion.Confidence = confidence
		return
	}

	// Try by parent name
	matchedChild, confidence = csvparser.MatchChildByParentName(matchText, children)
	if matchedChild != nil {
		suggestion.Child = matchedChild
		suggestion.MatchedBy = "parent_name"
		suggestion.Confidence = confidence
	}
}

func (s *ImportService) matchFeeExpectation(ctx context.Context, tx domain.BankTransaction, suggestion *domain.MatchSuggestion) *domain.TransactionWarning {
	childID := suggestion.Child.ID
	feeType := *suggestion.DetectedType

	if feeType == domain.FeeTypeMembership && suggestion.Child.HouseholdID != nil {
		householdChildren, err := s.childRepo.GetByHouseholdID(ctx, *suggestion.Child.HouseholdID)
		if err == nil && len(householdChildren) > 0 {
			count := 0
			for _, householdChild := range householdChildren {
				c, err := s.feeRepo.CountUnpaidByType(ctx, householdChild.ID, feeType, tx.Amount)
				if err == nil {
					count += c
				}
			}

			if count > 1 {
				return &domain.TransactionWarning{
					ID:            uuid.New(),
					TransactionID: tx.ID,
					WarningType:   domain.WarningTypeMultipleOpenFees,
					Message:       fmt.Sprintf("Mehrere offene Vereinsbeiträge (%d) für diesen Haushalt gefunden - manuelle Zuordnung erforderlich", count),
					ActualAmount:  &tx.Amount,
					ChildID:       &childID,
					CreatedAt:     time.Now(),
				}
			}

			if count == 1 {
				for _, householdChild := range householdChildren {
					fee, err := s.feeRepo.FindBestUnpaid(ctx, householdChild.ID, feeType, tx.Amount, tx.BookingDate)
					if err == nil && fee != nil {
						suggestion.Expectation = fee
						return nil
					}
				}
			}

			for _, householdChild := range householdChildren {
				if fees, err := s.feeRepo.FindOldestUnpaidWithReminder(ctx, householdChild.ID, feeType, tx.Amount); err == nil && len(fees) == 2 {
					suggestion.Expectations = fees
					suggestion.Expectation = &fees[0]
					suggestion.MatchedBy = "combined"
					if suggestion.Confidence > 0 {
						suggestion.Confidence = min(suggestion.Confidence+confidenceBoostCombined, maxConfidenceCombined)
					}
					return nil
				}
			}

			return nil
		}
	}

	// Check how many unpaid fees exist with this exact amount
	count, err := s.feeRepo.CountUnpaidByType(ctx, childID, feeType, tx.Amount)
	if err != nil {
		return nil
	}

	// Multiple fees with same amount -> manual review required
	if count > 1 {
		return &domain.TransactionWarning{
			ID:            uuid.New(),
			TransactionID: tx.ID,
			WarningType:   domain.WarningTypeMultipleOpenFees,
			Message:       fmt.Sprintf("Mehrere offene Beiträge (%d) für dieses Kind gefunden - manuelle Zuordnung erforderlich", count),
			ActualAmount:  &tx.Amount,
			ChildID:       &childID,
			CreatedAt:     time.Now(),
		}
	}

	// Exactly one fee -> auto-match
	if count == 1 {
		if fee, err := s.feeRepo.FindBestUnpaid(ctx, childID, feeType, tx.Amount, tx.BookingDate); err == nil && fee != nil {
			suggestion.Expectation = fee
		}
		return nil
	}

	// No exact match -> try combined fee + reminder (e.g., 55.40 = 45.40 + 10.00)
	if fees, err := s.feeRepo.FindOldestUnpaidWithReminder(ctx, childID, feeType, tx.Amount); err == nil && len(fees) == 2 {
		suggestion.Expectations = fees
		suggestion.Expectation = &fees[0]
		suggestion.MatchedBy = "combined"
		if suggestion.Confidence > 0 {
			suggestion.Confidence = min(suggestion.Confidence+confidenceBoostCombined, maxConfidenceCombined)
		}
	}
	return nil
}

func (s *ImportService) enrichChildrenWithParents(ctx context.Context, children []domain.Child) error {
	if len(children) == 0 {
		return nil
	}

	childIDs := make([]uuid.UUID, len(children))
	for i := range children {
		childIDs[i] = children[i].ID
	}

	parentsMap, err := s.childRepo.GetParentsForChildren(ctx, childIDs)
	if err != nil {
		return err
	}

	for i := range children {
		if parents, ok := parentsMap[children[i].ID]; ok {
			children[i].Parents = parents
		}
	}
	return nil
}

func buildMatchText(tx domain.BankTransaction) string {
	var parts []string
	if tx.PayerName != nil && strings.TrimSpace(*tx.PayerName) != "" {
		parts = append(parts, *tx.PayerName)
	}
	if tx.Description != nil && strings.TrimSpace(*tx.Description) != "" {
		parts = append(parts, *tx.Description)
	}
	return strings.Join(parts, " ")
}

// checkForWarning checks if an unmatched transaction from a trusted IBAN should generate a warning.
func (s *ImportService) checkForWarning(ctx context.Context, tx domain.BankTransaction, schedules domain.FeeSchedules) *domain.TransactionWarning {
	if tx.PayerIBAN == nil {
		return nil
	}

	// Check if IBAN is trusted
	knownIBAN, err := s.knownIBANRepo.GetByIBAN(ctx, *tx.PayerIBAN)
	if err != nil || knownIBAN == nil || knownIBAN.Status != domain.KnownIBANStatusTrusted {
		return nil
	}

	warning := &domain.TransactionWarning{
		ID:            uuid.New(),
		TransactionID: tx.ID,
		ActualAmount:  &tx.Amount,
		ChildID:       knownIBAN.ChildID,
		CreatedAt:     time.Now(),
	}

	// If we have a linked child, check what fees are open
	if knownIBAN.ChildID != nil {
		childID := *knownIBAN.ChildID

		// Check for possible bulk payment (amount is multiple of known fee amounts)
		bulkCount := checkBulkPayment(tx.Amount, feeConfigAt(schedules, tx.BookingDate))
		if bulkCount > 1 {
			warning.WarningType = domain.WarningTypePossibleBulk
			warning.Message = fmt.Sprintf("Betrag %.2f EUR könnte eine Sammelzahlung sein (%d Zahlungen)", tx.Amount, bulkCount)
			return warning
		}

		// Check open fees for this child to determine warning type
		filter := repository.FeeFilter{ChildID: &childID}
		fees, _, err := s.feeRepo.List(ctx, filter, 0, 100)
		if err == nil && len(fees) > 0 {
			// Find unpaid fees
			for _, fee := range fees {
				// Check if this fee is already paid
				isPaid, _ := s.matchRepo.ExistsForExpectation(ctx, fee.ID)
				if isPaid {
					continue
				}

				// Compare amounts
				if tx.Amount < fee.Amount {
					warning.WarningType = domain.WarningTypePartialPayment
					warning.ExpectedAmount = &fee.Amount
					warning.Message = fmt.Sprintf("Teilzahlung: %.2f EUR erhalten, %.2f EUR erwartet", tx.Amount, fee.Amount)
					return warning
				} else if tx.Amount > fee.Amount {
					warning.WarningType = domain.WarningTypeOverpayment
					warning.ExpectedAmount = &fee.Amount
					warning.Message = fmt.Sprintf("Überzahlung: %.2f EUR erhalten, %.2f EUR erwartet", tx.Amount, fee.Amount)
					return warning
				}
			}
		}

		// No open fees found for this child
		warning.WarningType = domain.WarningTypeNoMatchingFee
		warning.Message = fmt.Sprintf("Keine offene Beitragsforderung für dieses Kind gefunden (%.2f EUR)", tx.Amount)
		return warning
	}

	// Trusted IBAN without linked child - just note the unexpected payment
	warning.WarningType = domain.WarningTypeUnexpectedAmount
	warning.Message = fmt.Sprintf("Zahlung von %.2f EUR von vertrauter IBAN ohne zugeordnetes Kind", tx.Amount)
	return warning
}

// checkBulkPayment checks if an amount could be a bulk payment of multiple fees.
func checkBulkPayment(amount float64, cfg *domain.FeeScheduleConfig) int {
	if cfg == nil {
		return 1
	}
	foodFee := cfg.MonthlyFoodFee
	membershipFee := cfg.AnnualMembershipFee
	reminderFee := domain.ReminderFeeAmount

	// Check for exact multiples of food fee
	if foodFee > 0 && amount >= foodFee*2 {
		count := int(amount / foodFee)
		if amount == foodFee*float64(count) {
			return count
		}
	}

	// Check for exact multiples of membership fee
	if membershipFee > 0 && amount >= membershipFee*2 {
		count := int(amount / membershipFee)
		if amount == membershipFee*float64(count) {
			return count
		}
	}

	// Check for food + reminder combinations
	combinedFood := foodFee + reminderFee // 55.40
	if amount >= combinedFood*2 {
		count := int(amount / combinedFood)
		if amount == combinedFood*float64(count) {
			return count
		}
	}

	return 1
}

// GetUnmatchedSuggestionsForChild returns likely unmatched transactions for a specific child.
func (s *ImportService) GetUnmatchedSuggestionsForChild(ctx context.Context, childID uuid.UUID, minConfidence float64, limit int) (*ChildUnmatchedSuggestionsResult, error) {
	if limit < 1 {
		limit = 10
	}
	if minConfidence < 0 {
		minConfidence = 0
	}
	if minConfidence > 1 {
		minConfidence = 1
	}

	child, err := s.childRepo.GetByID(ctx, childID)
	if err != nil {
		return nil, ErrNotFound
	}

	children := []domain.Child{*child}
	if err := s.enrichChildrenWithParents(ctx, children); err != nil {
		return nil, err
	}

	scanLimit := 500
	if limit > scanLimit {
		scanLimit = limit
	}

	transactions, _, err := s.transactionRepo.ListUnmatched(ctx, "", "date", "desc", 0, scanLimit)
	if err != nil {
		return nil, err
	}

	schedules, err := s.scheduleRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("load fee schedules: %w", err)
	}

	result := &ChildUnmatchedSuggestionsResult{
		ChildID: childID,
	}

	for _, tx := range transactions {
		result.Scanned++
		suggestion := &domain.MatchSuggestion{
			Transaction:  tx,
			DetectedType: detectFeeType(tx.Amount, feeConfigAt(schedules, tx.BookingDate)),
		}

		s.matchChild(buildMatchText(tx), children, suggestion)
		if suggestion.Child == nil || suggestion.Child.ID != childID {
			continue
		}

		if suggestion.DetectedType != nil {
			_ = s.matchFeeExpectation(ctx, tx, suggestion)
		}

		// Boost confidence for name-based matches with fee expectations
		if suggestion.Expectation != nil && (suggestion.MatchedBy == "name" || suggestion.MatchedBy == "parent_name") {
			suggestion.Confidence = min(suggestion.Confidence+confidenceBoostNameMatch, maxConfidenceNameMatch)
		}

		if suggestion.Confidence < minConfidence {
			continue
		}

		result.Suggestions = append(result.Suggestions, *suggestion)
		if len(result.Suggestions) >= limit {
			break
		}
	}

	return result, nil
}

// GetSuggestionsForTransaction returns match suggestions for a single transaction.
func (s *ImportService) GetSuggestionsForTransaction(ctx context.Context, transactionID uuid.UUID) (*domain.MatchSuggestion, error) {
	// Get the transaction
	tx, err := s.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, ErrNotFound
	}

	children, err := s.loadMatchingChildren(ctx)
	if err != nil {
		return nil, err
	}

	schedules, err := s.scheduleRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("load fee schedules: %w", err)
	}

	// Run matching algorithm
	suggestion, warning := s.matchTransaction(ctx, *tx, children, schedules)
	if suggestion == nil && warning != nil && warning.WarningType == domain.WarningTypeMultipleOpenFees {
		// Provide child/type confidence even when multiple open fees exist,
		// so the manual matching UI can still surface high-confidence candidates.
		fallback := &domain.MatchSuggestion{
			Transaction:  *tx,
			DetectedType: detectFeeType(tx.Amount, feeConfigAt(schedules, tx.BookingDate)),
		}
		s.matchTrustedIBAN(ctx, *tx, children, fallback)
		if fallback.Child == nil {
			s.matchChild(buildMatchText(*tx), children, fallback)
		}
		if fallback.Confidence > 0 {
			return fallback, nil
		}
	}
	if suggestion == nil {
		return &domain.MatchSuggestion{
			Transaction: *tx,
			MatchedBy:   "none",
		}, nil
	}
	return suggestion, nil
}
