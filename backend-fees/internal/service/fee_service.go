package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

// FeeService handles fee-related business logic.
type FeeService struct {
	feeRepo         repository.FeeRepository
	childRepo       repository.ChildRepository
	householdRepo   repository.HouseholdRepository
	matchRepo       repository.MatchRepository
	transactionRepo repository.TransactionRepository
}

// NewFeeService creates a new fee service.
func NewFeeService(
	feeRepo repository.FeeRepository,
	childRepo repository.ChildRepository,
	householdRepo repository.HouseholdRepository,
	matchRepo repository.MatchRepository,
	transactionRepo repository.TransactionRepository,
) *FeeService {
	return &FeeService{
		feeRepo:         feeRepo,
		childRepo:       childRepo,
		householdRepo:   householdRepo,
		matchRepo:       matchRepo,
		transactionRepo: transactionRepo,
	}
}

// FeeFilter defines filters for listing fees.
type FeeFilter struct {
	Year    *int
	Month   *int
	FeeType string
	Status  string
	ChildID *uuid.UUID
	Search  string // Search by member number or child name
}

// GenerateResult represents the result of fee generation.
type GenerateResult struct {
	Created int `json:"created"`
	Skipped int `json:"skipped"`
}

// List returns fees matching the filter.
func (s *FeeService) List(ctx context.Context, filter FeeFilter, offset, limit int) ([]domain.FeeExpectation, int64, error) {
	fees, total, err := s.feeRepo.List(ctx, repository.FeeFilter{
		Year:    filter.Year,
		Month:   filter.Month,
		FeeType: filter.FeeType,
		ChildID: filter.ChildID,
		Search:  filter.Search,
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
		// Check if paid and get match details with transaction
		match, _ := s.matchRepo.GetByExpectation(ctx, fees[i].ID)
		if match != nil {
			fees[i].IsPaid = true
			fees[i].PaidAt = &match.MatchedAt
			// Load transaction data for the match
			if s.transactionRepo != nil {
				tx, err := s.transactionRepo.GetByID(ctx, match.TransactionID)
				if err == nil {
					match.Transaction = tx
				}
			}
			fees[i].MatchedBy = match
		}
	}

	return fees, total, nil
}

// GetByID returns a fee by ID.
func (s *FeeService) GetByID(ctx context.Context, id uuid.UUID) (*domain.FeeExpectation, error) {
	fee, err := s.feeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	// Check if paid and get match details with transaction
	match, _ := s.matchRepo.GetByExpectation(ctx, id)
	if match != nil {
		fee.IsPaid = true
		fee.PaidAt = &match.MatchedAt
		// Load transaction data for the match
		if s.transactionRepo != nil {
			tx, err := s.transactionRepo.GetByID(ctx, match.TransactionID)
			if err == nil {
				match.Transaction = tx
			}
		}
		fee.MatchedBy = match
	}

	// Get child info
	child, _ := s.childRepo.GetByID(ctx, fee.ChildID)
	fee.Child = child

	return fee, nil
}

// GetOverview returns fee overview statistics.
func (s *FeeService) GetOverview(ctx context.Context, year *int) (*domain.FeeOverview, error) {
	targetYear := time.Now().Year()
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
	children, _, err := s.childRepo.List(ctx, true, false, false, "", "", "", 0, 1000)
	if err != nil {
		return nil, err
	}

	result := &GenerateResult{}
	now := time.Now()

	for _, child := range children {
		// Generate monthly fees
		if month != nil {
			dueDate := time.Date(year, time.Month(*month), 5, 0, 0, 0, 0, time.UTC)
			checkDate := time.Date(year, time.Month(*month), 1, 0, 0, 0, 0, time.UTC)

			// Food fee (all children)
			created, err := s.createFeeIfNotExists(ctx, child.ID, domain.FeeTypeFood, year, month, domain.FoodFeeAmount, dueDate)
			if err != nil {
				return nil, err
			}
			if created {
				result.Created++
			} else {
				result.Skipped++
			}

			// Childcare fee (only U3)
			if child.IsUnderThree(checkDate) {
				// Get income and status - prefer household, fall back to parents
				isFosterFamily := false
				isHighestRate := false
				var income float64 = 0
				var siblingsCount int = 1

				// Try to get income from household first
				if child.HouseholdID != nil && s.householdRepo != nil {
					household, err := s.householdRepo.GetByID(ctx, *child.HouseholdID)
					if err == nil && household != nil {
						if household.IncomeStatus == domain.IncomeStatusFosterFamily {
							isFosterFamily = true
						} else if household.IncomeStatus == domain.IncomeStatusMaxAccepted {
							isHighestRate = true
						} else if household.IncomeStatus == domain.IncomeStatusProvided && household.AnnualHouseholdIncome != nil {
							income = *household.AnnualHouseholdIncome
						}
						// TODO: count siblings from household children
					}
				}

				// Fall back to parent income if not set from household
				if !isFosterFamily && !isHighestRate && income == 0 {
					parents, _ := s.childRepo.GetParents(ctx, child.ID)
					for _, parent := range parents {
						if parent.IncomeStatus == domain.IncomeStatusFosterFamily {
							isFosterFamily = true
							break
						}
						if parent.IncomeStatus == domain.IncomeStatusMaxAccepted {
							isHighestRate = true
						}
						if parent.IncomeStatus == domain.IncomeStatusProvided && parent.AnnualHouseholdIncome != nil {
							income = *parent.AnnualHouseholdIncome
						}
					}
				}

				// Get care hours from child, default to 45
				careHours := 45
				if child.CareHours != nil && *child.CareHours > 0 {
					careHours = *child.CareHours
				}

				feeResult := s.CalculateChildcareFee(domain.ChildcareFeeInput{
					ChildAgeType:  domain.ChildAgeTypeKrippe,
					NetIncome:     income,
					SiblingsCount: siblingsCount,
					CareHours:     careHours,
					HighestRate:   isHighestRate,
					FosterFamily:  isFosterFamily,
				})
				created, err := s.createFeeIfNotExists(ctx, child.ID, domain.FeeTypeChildcare, year, month, feeResult.Fee, dueDate)
				if err != nil {
					return nil, err
				}
				if created {
					result.Created++
				} else {
					result.Skipped++
				}
			}
		} else {
			// Generate yearly membership fee
			dueDate := time.Date(year, 3, 31, 0, 0, 0, 0, time.UTC)

			// Only generate if child was enrolled before the year ends
			if child.EntryDate.Year() <= year {
				created, err := s.createFeeIfNotExists(ctx, child.ID, domain.FeeTypeMembership, year, nil, domain.MembershipFeeAmount, dueDate)
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
	}

	_ = now // suppress unused warning

	return result, nil
}

func (s *FeeService) createFeeIfNotExists(ctx context.Context, childID uuid.UUID, feeType domain.FeeType, year int, month *int, amount float64, dueDate time.Time) (bool, error) {
	// Check if fee already exists
	exists, err := s.feeRepo.Exists(ctx, childID, feeType, year, month)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}

	fee := &domain.FeeExpectation{
		ID:        uuid.New(),
		ChildID:   childID,
		FeeType:   feeType,
		Year:      year,
		Month:     month,
		Amount:    amount,
		DueDate:   dueDate,
		CreatedAt: time.Now(),
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

// CalculateChildcareFee calculates the childcare fee (Platzgeld) based on income,
// care hours, number of siblings, and child age type.
func (s *FeeService) CalculateChildcareFee(input domain.ChildcareFeeInput) *domain.ChildcareFeeResult {
	limits := domain.ChildcareFeeLimits
	meta := domain.ChildcareFeeMeta

	// Default values
	if input.SiblingsCount < 1 {
		input.SiblingsCount = 1
	}
	if input.CareHours == 0 {
		input.CareHours = 30
	}

	// Kindergarten (>= 3 years) is free in Brandenburg
	if input.ChildAgeType == domain.ChildAgeTypeKindergarten {
		return &domain.ChildcareFeeResult{
			Fee:             0,
			BaseFee:         0,
			Rule:            "Beitragsfrei (ab 3 Jahren)",
			DiscountFactor:  1.0,
			DiscountPercent: 0,
			ShowEntlastung:  false,
			Notes:           []string{"Die Betreuung im Kindergartenalter ist in Brandenburg beitragsfrei."},
		}
	}

	// Krippe (< 3 years)

	// Foster family: average of all Satzung rates for the care hours (no sibling discount)
	if input.FosterFamily {
		avgFee := calculateAverageSatzungRate(input.CareHours)
		return &domain.ChildcareFeeResult{
			Fee:             roundToTwoDecimals(avgFee),
			BaseFee:         avgFee,
			Rule:            "Pflegefamilie (Durchschnittsbeitrag)",
			DiscountFactor:  1.0,
			DiscountPercent: 0,
			ShowEntlastung:  false,
			Notes:           []string{"Beitrag ist der Durchschnitt aller Sätze für die entsprechende Betreuungszeit."},
		}
	}

	// 7+ children: free
	if input.SiblingsCount >= meta.SiblingsFreeThreshold {
		return &domain.ChildcareFeeResult{
			Fee:             0,
			BaseFee:         0,
			Rule:            "Beitragsfrei (≥ 7 Kinder)",
			DiscountFactor:  1.0,
			DiscountPercent: 0,
			ShowEntlastung:  false,
			Notes:           []string{"Bei 7 oder mehr unterhaltsberechtigten Kindern entfällt der Elternbeitrag."},
		}
	}

	// Highest rate voluntarily chosen (no income check, but sibling discount applies)
	if input.HighestRate {
		lastRow := domain.FeeTableKrippeSatzung[len(domain.FeeTableKrippeSatzung)-1]
		baseFee := findRate(lastRow.Rates[:], input.CareHours)
		discountFactor := getSiblingDiscountFactor(input.SiblingsCount, meta.MaxSiblingsForDiscount)
		fee := baseFee * discountFactor
		discountPercent := int((1 - discountFactor) * 100)

		notes := []string{}
		if input.SiblingsCount > 1 && discountFactor < 1.0 {
			notes = append(notes, "Geschwisterermäßigung berücksichtigt.")
		}

		return &domain.ChildcareFeeResult{
			Fee:             roundToTwoDecimals(fee),
			BaseFee:         baseFee,
			Rule:            "Höchstsatz (Satzung U3)",
			DiscountFactor:  discountFactor,
			DiscountPercent: discountPercent,
			ShowEntlastung:  false,
			Notes:           notes,
		}
	}

	// Income <= 35,000: free
	if input.NetIncome <= limits.MinIncomeFreeU3 {
		return &domain.ChildcareFeeResult{
			Fee:             0,
			BaseFee:         0,
			Rule:            "Beitragsfrei (Einkommen ≤ 35.000 EUR)",
			DiscountFactor:  1.0,
			DiscountPercent: 0,
			ShowEntlastung:  true,
			Notes:           []string{"Gemäß Elternbeitragsentlastungsgesetz."},
		}
	}

	// Entlastung bracket: 35,000.01 - 55,000.00 (no sibling discount)
	if input.NetIncome >= limits.MinIncomeEntlastungU3 && input.NetIncome <= limits.MaxIncomeEntlastungU3 {
		baseFee := findRateInTable(domain.FeeTableKrippeEntlastung, input.NetIncome, input.CareHours)
		return &domain.ChildcareFeeResult{
			Fee:             baseFee,
			BaseFee:         baseFee,
			Rule:            "Reduzierter Beitrag (Entlastung U3)",
			DiscountFactor:  1.0,
			DiscountPercent: 0,
			ShowEntlastung:  true,
			Notes: []string{
				"Kein zusätzlicher Geschwisterrabatt in diesem Einkommensbereich.",
				"Rechtsgrundlage: Elternbeitragsentlastungsgesetz.",
			},
		}
	}

	// Satzung bracket: >= 55,000.01 (sibling discount applies)
	if input.NetIncome >= limits.MinIncomeSatzungU3 {
		baseFee := findRateInTable(domain.FeeTableKrippeSatzung, input.NetIncome, input.CareHours)
		discountFactor := getSiblingDiscountFactor(input.SiblingsCount, meta.MaxSiblingsForDiscount)
		fee := baseFee * discountFactor
		discountPercent := int((1 - discountFactor) * 100)

		notes := []string{}
		if input.SiblingsCount > 1 && discountFactor < 1.0 {
			notes = append(notes, "Geschwisterermäßigung berücksichtigt.")
		}

		return &domain.ChildcareFeeResult{
			Fee:             roundToTwoDecimals(fee),
			BaseFee:         baseFee,
			Rule:            "Regulärer Beitrag (Satzung U3)",
			DiscountFactor:  discountFactor,
			DiscountPercent: discountPercent,
			ShowEntlastung:  false,
			Notes:           notes,
		}
	}

	// Fallback (should not occur, covered by <= 35k)
	return &domain.ChildcareFeeResult{
		Fee:             0,
		BaseFee:         0,
		Rule:            "Beitragsfrei (Einkommen U3 < 35k)",
		DiscountFactor:  1.0,
		DiscountPercent: 0,
		ShowEntlastung:  true,
		Notes:           []string{},
	}
}

// hoursToIndex maps care hours (30, 35, 40, 45, 50, 55) to array index (0-5).
func hoursToIndex(hours int) int {
	idx := (hours - 30) / 5
	if idx < 0 {
		return 0
	}
	if idx > 5 {
		return 5
	}
	return idx
}

// findRate finds the rate for the given hours from a rates array.
func findRate(rates []float64, hours int) float64 {
	idx := hoursToIndex(hours)
	if idx >= 0 && idx < len(rates) {
		return rates[idx]
	}
	return 0
}

// findRateInTable finds the appropriate rate from a fee table based on income and hours.
func findRateInTable(table []domain.FeeTableRow, income float64, hours int) float64 {
	idx := hoursToIndex(hours)

	// Find last bracket where income >= minIncome
	for i := len(table) - 1; i >= 0; i-- {
		if income >= table[i].MinIncome {
			return table[i].Rates[idx]
		}
	}

	return 0
}

// calculateAverageSatzungRate calculates the average of all Satzung rates for the given care hours.
// Used for foster family fee calculation.
func calculateAverageSatzungRate(hours int) float64 {
	idx := hoursToIndex(hours)
	table := domain.FeeTableKrippeSatzung

	var sum float64
	for _, row := range table {
		sum += row.Rates[idx]
	}

	if len(table) == 0 {
		return 0
	}

	return sum / float64(len(table))
}

// getSiblingDiscountFactor returns the discount factor based on number of siblings.
func getSiblingDiscountFactor(siblingsCount, maxForDiscount int) float64 {
	if siblingsCount > maxForDiscount {
		siblingsCount = maxForDiscount
	}
	if factor, ok := domain.SiblingDiscount[siblingsCount]; ok {
		return factor
	}
	return 1.0
}

// roundToTwoDecimals rounds a float to two decimal places.
func roundToTwoDecimals(val float64) float64 {
	return float64(int(val*100+0.5)) / 100
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
	now := time.Now()
	reminder := &domain.FeeExpectation{
		ID:            uuid.New(),
		ChildID:       originalFee.ChildID,
		FeeType:       domain.FeeTypeReminder,
		Year:          now.Year(),
		Month:         nil, // Reminders don't have a specific month
		Amount:        domain.ReminderFeeAmount,
		DueDate:       now.AddDate(0, 0, 14), // Due in 14 days
		CreatedAt:     now,
		ReminderForID: &feeID,
	}

	if err := s.feeRepo.Create(ctx, reminder); err != nil {
		return nil, err
	}

	// Fetch and return with child info
	return s.GetByID(ctx, reminder.ID)
}
