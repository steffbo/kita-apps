package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

// FeeService handles fee-related business logic.
type FeeService struct {
	feeRepo         repository.FeeRepository
	childRepo       repository.ChildRepository
	householdRepo   repository.HouseholdRepository
	matchRepo       repository.MatchRepository
	transactionRepo repository.TransactionRepository
	scheduleRepo    repository.FeeScheduleRepository
}

// NewFeeService creates a new fee service.
func NewFeeService(
	feeRepo repository.FeeRepository,
	childRepo repository.ChildRepository,
	householdRepo repository.HouseholdRepository,
	matchRepo repository.MatchRepository,
	transactionRepo repository.TransactionRepository,
	scheduleRepo repository.FeeScheduleRepository,
) *FeeService {
	return &FeeService{
		feeRepo:         feeRepo,
		childRepo:       childRepo,
		householdRepo:   householdRepo,
		matchRepo:       matchRepo,
		transactionRepo: transactionRepo,
		scheduleRepo:    scheduleRepo,
	}
}

// ScheduleAt returns the fee schedule (Elternbeitragsordnung) valid at date.
func (s *FeeService) ScheduleAt(ctx context.Context, date time.Time) (*domain.FeeSchedule, error) {
	return s.scheduleRepo.GetAt(ctx, date)
}

// Schedules returns all fee schedule versions, e.g. to evaluate many months at once.
func (s *FeeService) Schedules(ctx context.Context) (domain.FeeSchedules, error) {
	return s.scheduleRepo.List(ctx)
}

// FeeFilter defines filters for listing fees.
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

// GenerateResult represents the result of fee generation.
type GenerateResult struct {
	Created int `json:"created"`
	Skipped int `json:"skipped"`
}

// ChildcareExpectationSyncResult describes automatic expectation changes after a follow-up classification.
type ChildcareExpectationSyncResult struct {
	Created              int                  `json:"created"`
	Updated              int                  `json:"updated"`
	Skipped              int                  `json:"skipped"`
	DeltaOpen            float64              `json:"deltaOpen"`
	CreditReviewRequired []CreditReviewPeriod `json:"creditReviewRequired"`
}

// CreditReviewPeriod identifies a paid month that would become overpaid after a decrease.
type CreditReviewPeriod struct {
	FeeID         uuid.UUID `json:"feeId"`
	Year          int       `json:"year"`
	Month         int       `json:"month"`
	OldAmount     float64   `json:"oldAmount"`
	NewAmount     float64   `json:"newAmount"`
	MatchedAmount float64   `json:"matchedAmount"`
	CreditAmount  float64   `json:"creditAmount"`
}

// incomeInfo holds income and sibling information for fee calculation.
type incomeInfo struct {
	IsFosterFamily bool
	IsHighestRate  bool
	Income         float64
	SiblingsCount  int
}

// getIncomeInfo retrieves income and sibling information for a child.
// It prefers household data but falls back to parent data if needed.
func (s *FeeService) getIncomeInfo(ctx context.Context, child *domain.Child) incomeInfo {
	info := incomeInfo{SiblingsCount: 1}

	// Try to get income from household first
	if child.HouseholdID != nil && s.householdRepo != nil {
		household, err := s.householdRepo.GetWithMembers(ctx, *child.HouseholdID)
		if err == nil && household != nil {
			if household.IncomeStatus == domain.IncomeStatusFosterFamily {
				info.IsFosterFamily = true
			} else if household.IncomeStatus == domain.IncomeStatusMaxAccepted {
				info.IsHighestRate = true
			} else if household.IncomeStatus == domain.IncomeStatusProvided && household.AnnualHouseholdIncome != nil {
				info.Income = *household.AnnualHouseholdIncome
			}

			// Get sibling count: use override if set, otherwise count active children
			if household.ChildrenCountForFees != nil && *household.ChildrenCountForFees > 0 {
				info.SiblingsCount = *household.ChildrenCountForFees
			} else if len(household.Children) > 0 {
				now := util.Today()
				enrolledCount := 0
				for _, c := range household.Children {
					if c.IsEnrolledAt(now) {
						enrolledCount++
					}
				}
				if enrolledCount > 0 {
					info.SiblingsCount = enrolledCount
				}
			}
		}
	}

	// Fall back to parent income if not set from household
	if !info.IsFosterFamily && !info.IsHighestRate && info.Income == 0 {
		parents, _ := s.childRepo.GetParents(ctx, child.ID)
		for _, parent := range parents {
			if parent.IncomeStatus == domain.IncomeStatusFosterFamily {
				info.IsFosterFamily = true
				break
			}
			if parent.IncomeStatus == domain.IncomeStatusMaxAccepted {
				info.IsHighestRate = true
			}
			if parent.IncomeStatus == domain.IncomeStatusProvided && parent.AnnualHouseholdIncome != nil {
				info.Income = *parent.AnnualHouseholdIncome
			}
		}
	}

	return info
}

// List returns fees matching the filter.
func (s *FeeService) List(ctx context.Context, filter FeeFilter, offset, limit int) ([]domain.FeeExpectation, int64, error) {
	fees, total, err := s.feeRepo.List(ctx, repository.FeeFilter{
		Year:    filter.Year,
		Month:   filter.Month,
		FeeType: filter.FeeType,
		Status:  filter.Status,
		ChildID: filter.ChildID,
		Search:  filter.Search,
		SortBy:  filter.SortBy,
		SortDir: filter.SortDir,
	}, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	// Collect unique child IDs
	childIDs := make(map[uuid.UUID]bool)
	for _, fee := range fees {
		childIDs[fee.ChildID] = true
	}

	// Fetch all children at once
	childMap := make(map[uuid.UUID]*domain.Child)
	for childID := range childIDs {
		child, err := s.childRepo.GetByID(ctx, childID)
		if err == nil {
			childMap[childID] = child
		}
	}

	// Enrich with child data and payment status
	for i := range fees {
		if child, ok := childMap[fees[i].ChildID]; ok {
			fees[i].Child = child
		}
		s.enrichWithPaymentStatus(ctx, &fees[i])
	}

	return fees, total, nil
}

// GetByID returns a fee by ID.
func (s *FeeService) GetByID(ctx context.Context, id uuid.UUID) (*domain.FeeExpectation, error) {
	fee, err := s.feeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	s.enrichWithPaymentStatus(ctx, fee)

	// Get child info
	child, _ := s.childRepo.GetByID(ctx, fee.ChildID)
	fee.Child = child

	return fee, nil
}

// GetOverview returns fee overview statistics.
func (s *FeeService) GetOverview(ctx context.Context, year *int) (*domain.FeeOverview, error) {
	targetYear := util.Now().Year()
	if year != nil {
		targetYear = *year
	}

	overview, err := s.feeRepo.GetOverview(ctx, targetYear)
	if err != nil {
		return nil, err
	}

	return overview, nil
}

// Generate creates fee expectations for the given period.
func (s *FeeService) Generate(ctx context.Context, year int, month *int) (*GenerateResult, error) {
	// Monthly generation must use the billed period rather than today's active flag.
	children, _, err := s.childRepo.List(ctx, month == nil, false, false, false, "", "", "", 0, 1000)
	if err != nil {
		return nil, err
	}

	result := &GenerateResult{}

	// Membership fees are tracked once per household/year.
	if month == nil {
		return s.generateYearlyMembershipFees(ctx, year, children)
	}

	periodStart := time.Date(year, time.Month(*month), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)
	dueDate := time.Date(year, time.Month(*month), 5, 0, 0, 0, 0, time.UTC)
	schedule, err := s.ScheduleAt(ctx, periodStart)
	if err != nil {
		return nil, err
	}

	for _, child := range children {
		if !child.EntryDate.Before(periodEnd) || (child.ExitDate != nil && child.ExitDate.Before(periodStart)) {
			continue
		}

		// Food fee (all children)
		foodAmount := domain.ContributionAmountForMonth(schedule.Config.MonthlyFoodFee, child.EntryDate, year, time.Month(*month))
		created, err := s.createFeeIfNotExists(ctx, child.ID, child.HouseholdID, domain.FeeTypeFood, year, month, foodAmount, dueDate)
		if err != nil {
			return nil, err
		}
		if created {
			result.Created++
		} else {
			result.Skipped++
		}

		// Childcare fee (only U3)
		// If child turns 3 at any point during the month, no childcare fee is charged
		if child.IsUnderThreeForEntireMonth(year, time.Month(*month)) {
			info := s.getIncomeInfo(ctx, &child)

			// Care hours effective in the billed month (falls back to the default when unknown)
			careHours := s.ResolveCareHours(ctx, &child, year, month)

			feeResult := schedule.Config.CalculateChildcareFee(domain.ChildcareFeeInput{
				ChildAgeType:  domain.ChildAgeTypeKrippe,
				NetIncome:     info.Income,
				SiblingsCount: info.SiblingsCount,
				CareHours:     careHours,
				HighestRate:   info.IsHighestRate,
				FosterFamily:  info.IsFosterFamily,
			})
			if feeResult.Fee <= 0 {
				// Beitrag frei: kein Platzgeld erzeugen
				continue
			}
			childcareAmount := domain.ContributionAmountForMonth(feeResult.Fee, child.EntryDate, year, time.Month(*month))
			created, err := s.createFeeIfNotExists(ctx, child.ID, child.HouseholdID, domain.FeeTypeChildcare, year, month, childcareAmount, dueDate)
			if err != nil {
				return nil, err
			}
			if created {
				result.Created++
			} else {
				result.Skipped++
			}
		}
	}

	return result, nil
}

func (s *FeeService) createFeeIfNotExists(ctx context.Context, childID uuid.UUID, householdID *uuid.UUID, feeType domain.FeeType, year int, month *int, amount float64, dueDate time.Time) (bool, error) {
	// Check if fee already exists
	exists, err := s.feeRepo.Exists(ctx, childID, feeType, year, month)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}

	fee := &domain.FeeExpectation{
		ID:          uuid.New(),
		ChildID:     childID,
		HouseholdID: householdID,
		FeeType:     feeType,
		Year:        year,
		Month:       month,
		Amount:      amount,
		DueDate:     dueDate,
		CreatedAt:   time.Now(),
	}

	if err := s.feeRepo.Create(ctx, fee); err != nil {
		return false, err
	}

	return true, nil
}

// SyncChildcareExpectationsFrom updates CHILDCARE expectations from the effective month through year-end.
func (s *FeeService) SyncChildcareExpectationsFrom(ctx context.Context, childID uuid.UUID, householdID *uuid.UUID, effectiveFromMonth time.Time, amount float64, exitDate *time.Time) (*ChildcareExpectationSyncResult, error) {
	result := &ChildcareExpectationSyncResult{}
	effectiveFromMonth = firstDayOfMonth(effectiveFromMonth)
	endMonth := time.December
	if exitDate != nil && exitDate.Year() == effectiveFromMonth.Year() && exitDate.Month() < endMonth {
		endMonth = exitDate.Month()
	}

	for m := effectiveFromMonth.Month(); m <= endMonth; m++ {
		year := effectiveFromMonth.Year()
		month := int(m)
		dueDate := time.Date(year, m, 5, 0, 0, 0, 0, time.UTC)
		fee, err := s.feeRepo.GetByChildFeePeriod(ctx, childID, domain.FeeTypeChildcare, year, month)
		if err != nil {
			return nil, err
		}

		if fee == nil {
			if amount <= 0 {
				result.Skipped++
				continue
			}
			created := &domain.FeeExpectation{
				ID:          uuid.New(),
				ChildID:     childID,
				HouseholdID: householdID,
				FeeType:     domain.FeeTypeChildcare,
				Year:        year,
				Month:       &month,
				Amount:      amount,
				DueDate:     dueDate,
				CreatedAt:   time.Now(),
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

func (s *FeeService) generateYearlyMembershipFees(ctx context.Context, year int, children []domain.Child) (*GenerateResult, error) {
	result := &GenerateResult{}
	dueDate := time.Date(year, 3, 31, 0, 0, 0, 0, time.UTC)
	// The annual membership fee follows the schedule in effect on 1 January.
	schedule, err := s.ScheduleAt(ctx, time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		return nil, err
	}
	membershipFee := schedule.Config.AnnualMembershipFee

	groupedByHousehold := make(map[uuid.UUID][]domain.Child)
	childrenWithoutHousehold := make([]domain.Child, 0)
	for _, child := range children {
		if child.EntryDate.Year() > year {
			continue
		}
		if child.HouseholdID == nil {
			childrenWithoutHousehold = append(childrenWithoutHousehold, child)
			continue
		}
		groupedByHousehold[*child.HouseholdID] = append(groupedByHousehold[*child.HouseholdID], child)
	}

	for householdID, householdChildren := range groupedByHousehold {
		// Use all children in household for existence check, even if not active right now.
		allHouseholdChildren, err := s.childRepo.GetByHouseholdID(ctx, householdID)
		if err != nil {
			return nil, err
		}
		if len(allHouseholdChildren) == 0 {
			allHouseholdChildren = householdChildren
		}

		exists := false
		for _, child := range allHouseholdChildren {
			hasMembershipFee, err := s.feeRepo.Exists(ctx, child.ID, domain.FeeTypeMembership, year, nil)
			if err != nil {
				return nil, err
			}
			if hasMembershipFee {
				exists = true
				break
			}
		}

		if exists {
			result.Skipped++
			if err := s.ensureHouseholdMembershipAssignment(ctx, householdID); err != nil {
				return nil, err
			}
			continue
		}

		representativeChild := pickRepresentativeChildForMembership(householdChildren)
		created, err := s.createFeeIfNotExists(ctx, representativeChild.ID, &householdID, domain.FeeTypeMembership, year, nil, membershipFee, dueDate)
		if err != nil {
			return nil, err
		}
		if created {
			result.Created++
		} else {
			result.Skipped++
		}

		if err := s.ensureHouseholdMembershipAssignment(ctx, householdID); err != nil {
			return nil, err
		}
	}

	// Legacy fallback: if no household linkage exists, keep per-child generation.
	for _, child := range childrenWithoutHousehold {
		created, err := s.createFeeIfNotExists(ctx, child.ID, nil, domain.FeeTypeMembership, year, nil, membershipFee, dueDate)
		if err != nil {
			return nil, err
		}
		if created {
			result.Created++
		} else {
			result.Skipped++
		}
	}

	return result, nil
}

func pickRepresentativeChildForMembership(children []domain.Child) domain.Child {
	if len(children) == 1 {
		return children[0]
	}

	sort.Slice(children, func(i, j int) bool {
		if children[i].EntryDate.Equal(children[j].EntryDate) {
			return children[i].ID.String() < children[j].ID.String()
		}
		return children[i].EntryDate.Before(children[j].EntryDate)
	})

	return children[0]
}

func (s *FeeService) ensureHouseholdMembershipAssignment(ctx context.Context, householdID uuid.UUID) error {
	household, err := s.householdRepo.GetByID(ctx, householdID)
	if err != nil {
		return nil
	}
	if household.MembershipParentID != nil && household.MembershipStatus != "" {
		return nil
	}

	parents, err := s.householdRepo.GetParents(ctx, householdID)
	if err != nil || len(parents) == 0 {
		return nil
	}

	parentID, status := pickHouseholdMembershipParent(parents)
	household.MembershipParentID = &parentID
	household.MembershipStatus = status
	return s.householdRepo.Update(ctx, household)
}

func pickHouseholdMembershipParent(parents []domain.Parent) (uuid.UUID, domain.MembershipAssignmentStatus) {
	candidates := make([]domain.Parent, 0, len(parents))
	for _, parent := range parents {
		if parent.MemberID != nil {
			candidates = append(candidates, parent)
		}
	}

	status := domain.MembershipAssignmentStatusAssumed
	if len(candidates) > 0 {
		parents = candidates
		status = domain.MembershipAssignmentStatusConfirmed
	}

	sort.Slice(parents, func(i, j int) bool {
		if parents[i].CreatedAt.Equal(parents[j].CreatedAt) {
			return parents[i].ID.String() < parents[j].ID.String()
		}
		return parents[i].CreatedAt.Before(parents[j].CreatedAt)
	})

	return parents[0].ID, status
}

// Update updates a fee's amount.
func (s *FeeService) Update(ctx context.Context, id uuid.UUID, amount *float64) (*domain.FeeExpectation, error) {
	fee, err := s.feeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	if amount != nil {
		fee.Amount = *amount
	}

	if err := s.feeRepo.Update(ctx, fee); err != nil {
		return nil, err
	}

	return fee, nil
}

// Delete deletes a fee.
func (s *FeeService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.feeRepo.GetByID(ctx, id)
	if err != nil {
		return ErrNotFound
	}

	return s.feeRepo.Delete(ctx, id)
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// roundToTwoDecimals rounds a float to two decimal places.
func roundToTwoDecimals(val float64) float64 {
	return float64(int(val*100+0.5)) / 100
}

func firstDayOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// CreateFeeInput represents the input for creating a single fee.
type CreateFeeInput struct {
	ChildID            uuid.UUID
	FeeType            domain.FeeType
	Year               int
	Month              *int
	Amount             *float64 // Optional: if nil, use default amount for fee type
	DueDate            *time.Time
	ReconciliationYear *int
}

// Create creates a single fee for a specific child.
// If amount is not provided, it calculates the appropriate amount based on fee type.
func (s *FeeService) Create(ctx context.Context, input CreateFeeInput) (*domain.FeeExpectation, error) {
	// Validate child exists
	child, err := s.childRepo.GetByID(ctx, input.ChildID)
	if err != nil {
		return nil, ErrNotFound
	}

	// Check if fee already exists
	exists, err := s.feeRepo.Exists(ctx, input.ChildID, input.FeeType, input.Year, input.Month)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrAlreadyExists
	}

	// Determine amount
	var amount float64
	if input.Amount != nil {
		amount = *input.Amount
	} else {
		// Calculate default amount based on fee type
		scheduleDate := time.Date(input.Year, 1, 1, 0, 0, 0, 0, time.UTC)
		if input.Month != nil {
			scheduleDate = time.Date(input.Year, time.Month(*input.Month), 1, 0, 0, 0, 0, time.UTC)
		}
		schedule, err := s.ScheduleAt(ctx, scheduleDate)
		if err != nil {
			return nil, err
		}
		switch input.FeeType {
		case domain.FeeTypeFood:
			amount = schedule.Config.MonthlyFoodFee
		case domain.FeeTypeMembership:
			amount = schedule.Config.AnnualMembershipFee
		case domain.FeeTypeReminder:
			amount = domain.ReminderFeeAmount
		case domain.FeeTypeChildcare:
			// Calculate childcare fee based on household income
			amount = s.calculateChildcareFeeForChild(ctx, schedule, child, input.Year, input.Month)
		default:
			return nil, ErrInvalidInput
		}
		if input.Month != nil && (input.FeeType == domain.FeeTypeFood || input.FeeType == domain.FeeTypeChildcare) {
			amount = domain.ContributionAmountForMonth(amount, child.EntryDate, input.Year, time.Month(*input.Month))
		}
	}

	// Determine due date
	var dueDate time.Time
	if input.DueDate != nil {
		dueDate = *input.DueDate
	} else {
		// Default due date: 5th of the month for monthly fees, March 31st for yearly
		if input.Month != nil {
			dueDate = time.Date(input.Year, time.Month(*input.Month), 5, 0, 0, 0, 0, time.UTC)
		} else {
			dueDate = time.Date(input.Year, 3, 31, 0, 0, 0, 0, time.UTC)
		}
	}

	fee := &domain.FeeExpectation{
		ID:                 uuid.New(),
		ChildID:            input.ChildID,
		HouseholdID:        child.HouseholdID,
		FeeType:            input.FeeType,
		Year:               input.Year,
		Month:              input.Month,
		Amount:             amount,
		DueDate:            dueDate,
		CreatedAt:          time.Now(),
		ReconciliationYear: input.ReconciliationYear,
	}

	if err := s.feeRepo.Create(ctx, fee); err != nil {
		return nil, err
	}

	// Return with child info
	return s.GetByID(ctx, fee.ID)
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
