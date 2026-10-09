package service

import (
	"context"
	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
	"time"
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

	var baseIDs []uuid.UUID
	for _, fee := range fees {
		if fee.ReminderForID != nil {
			baseIDs = append(baseIDs, *fee.ReminderForID)
		}
	}
	baseFees := map[uuid.UUID]*domain.FeeExpectation{}
	if len(baseIDs) > 0 {
		if baseFees, err = s.feeRepo.GetByIDs(ctx, baseIDs); err != nil {
			return nil, 0, err
		}
	}

	// Enrich with child data, base fee of reminders and payment status
	for i := range fees {
		if child, ok := childMap[fees[i].ChildID]; ok {
			fees[i].Child = child
		}
		if fees[i].ReminderForID != nil {
			if base, ok := baseFees[*fees[i].ReminderForID]; ok {
				fees[i].ReminderFor = &domain.FeeRef{FeeType: base.FeeType, Year: base.Year, Month: base.Month}
			}
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

	// Membership fees are tracked once per club member (or household without members) and year.
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
		CreatedAt:   util.Now(),
	}

	if err := s.feeRepo.Create(ctx, fee); err != nil {
		return false, err
	}

	return true, nil
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
		CreatedAt:          util.Now(),
		ReconciliationYear: input.ReconciliationYear,
	}

	if err := s.feeRepo.Create(ctx, fee); err != nil {
		return nil, err
	}

	// Return with child info
	return s.GetByID(ctx, fee.ID)
}
