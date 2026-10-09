package repository

import (
	"context"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/rs/zerolog/log"
	"time"
)

// GetStichtagsmeldungStats retrieves statistics for the Stichtagsmeldung report.
// It counts active children at the given stichtag date and breaks down U3 children by income.
func (r *PostgresChildRepository) GetStichtagsmeldungStats(ctx context.Context, stichtag time.Time) (*domain.StichtagsmeldungStats, error) {
	report, err := r.GetStichtagsmeldungReport(ctx, stichtag)
	if err != nil {
		return nil, err
	}

	return &domain.StichtagsmeldungStats{
		U3IncomeBreakdown:   report.U3IncomeBreakdown,
		TotalChildrenInKita: report.TotalChildrenInKita,
	}, nil
}

// GetStichtagsmeldungReport retrieves the full Stichtagsmeldung report for a specific date.
func (r *PostgresChildRepository) GetStichtagsmeldungReport(ctx context.Context, stichtag time.Time) (*domain.StichtagsmeldungReport, error) {
	u3Breakdown, totalChildren, u3ChildrenCount, err := r.getStichtagSummary(ctx, stichtag)
	if err != nil {
		return nil, err
	}

	var breakdownRows []struct {
		CareHours *int `db:"care_hours"`
		Count     int  `db:"count"`
		U3Count   int  `db:"u3_count"`
		Ue3Count  int  `db:"ue3_count"`
	}
	var legalBreakdownRows []struct {
		LegalHours *int `db:"legal_hours"`
		Count      int  `db:"count"`
		U3Count    int  `db:"u3_count"`
		Ue3Count   int  `db:"ue3_count"`
	}

	err = r.loadHoursBreakdown(ctx, &breakdownRows, stichtag, "fees.child_care_hours_history", "care_hours")
	if err != nil {
		return nil, err
	}
	err = r.loadHoursBreakdown(ctx, &legalBreakdownRows, stichtag, "fees.child_legal_hours_history", "legal_hours")
	if err != nil {
		return nil, err
	}

	breakdown := make([]domain.CareHoursBreakdown, len(breakdownRows))
	for i, row := range breakdownRows {
		breakdown[i] = domain.CareHoursBreakdown{
			CareHours: row.CareHours,
			Count:     row.Count,
			U3Count:   row.U3Count,
			Ue3Count:  row.Ue3Count,
		}
	}
	legalBreakdown := make([]domain.LegalHoursBreakdown, len(legalBreakdownRows))
	for i, row := range legalBreakdownRows {
		legalBreakdown[i] = domain.LegalHoursBreakdown{
			LegalHours: row.LegalHours,
			Count:      row.Count,
			U3Count:    row.U3Count,
			Ue3Count:   row.Ue3Count,
		}
	}

	return &domain.StichtagsmeldungReport{
		ReportDate:          stichtag,
		U3IncomeBreakdown:   u3Breakdown,
		TotalChildrenInKita: totalChildren,
		U3ChildrenCount:     u3ChildrenCount,
		Ue3ChildrenCount:    totalChildren - u3ChildrenCount,
		CareHoursBreakdown:  breakdown,
		LegalHoursBreakdown: legalBreakdown,
	}, nil
}

// GetU3ChildrenDetails retrieves details of U3 children for the Stichtagsmeldung modal.
func (r *PostgresChildRepository) GetU3ChildrenDetails(ctx context.Context, stichtag time.Time) ([]domain.U3ChildDetail, error) {
	u3Threshold := stichtag.AddDate(-3, 0, 0)

	var children []struct {
		ID              string   `db:"id"`
		MemberNumber    string   `db:"member_number"`
		FirstName       string   `db:"first_name"`
		LastName        string   `db:"last_name"`
		BirthDate       string   `db:"birth_date"`
		HouseholdIncome *float64 `db:"annual_household_income"`
		IncomeStatus    *string  `db:"income_status"`
	}

	err := conn(ctx, r.db).SelectContext(ctx, &children, `
		SELECT
			c.id::text AS id,
			c.member_number,
			c.first_name,
			c.last_name,
			TO_CHAR(c.birth_date, 'YYYY-MM-DD') AS birth_date,
			h.annual_household_income,
			h.income_status
		FROM fees.children c
		LEFT JOIN fees.households h ON c.household_id = h.id
		WHERE c.entry_date <= $1
		  AND (c.exit_date IS NULL OR c.exit_date >= $1)
		  AND c.birth_date > $2
		ORDER BY c.last_name, c.first_name
	`, stichtag, u3Threshold)
	if err != nil {
		log.Error().Err(err).Msg("GetU3ChildrenDetails query failed")
		return nil, err
	}

	result := make([]domain.U3ChildDetail, len(children))
	for i, c := range children {
		isFoster := c.IncomeStatus != nil && *c.IncomeStatus == "FOSTER_FAMILY"
		var income *int
		if c.HouseholdIncome != nil {
			incomeInt := int(*c.HouseholdIncome)
			income = &incomeInt
		}
		result[i] = domain.U3ChildDetail{
			ID:              c.ID,
			MemberNumber:    c.MemberNumber,
			FirstName:       c.FirstName,
			LastName:        c.LastName,
			BirthDate:       c.BirthDate,
			HouseholdIncome: income,
			IncomeStatus:    c.IncomeStatus,
			IsFosterFamily:  isFoster,
		}
	}

	return result, nil
}

func (r *PostgresChildRepository) getStichtagSummary(ctx context.Context, stichtag time.Time) (domain.U3IncomeBreakdown, int, int, error) {
	u3Threshold := stichtag.AddDate(-3, 0, 0)

	var breakdown struct {
		UpTo20k      int `db:"up_to_20k"`
		From20To35k  int `db:"from_20_to_35k"`
		From35To55k  int `db:"from_35_to_55k"`
		MaxAccepted  int `db:"max_accepted"`
		FosterFamily int `db:"foster_family"`
		Total        int `db:"total"`
	}

	err := conn(ctx, r.db).GetContext(ctx, &breakdown, `
		SELECT
			COUNT(*) FILTER (WHERE COALESCE(h.income_status, '') NOT IN ('MAX_ACCEPTED', 'FOSTER_FAMILY') AND COALESCE(h.annual_household_income, 0) <= 20000) AS up_to_20k,
			COUNT(*) FILTER (WHERE COALESCE(h.income_status, '') NOT IN ('MAX_ACCEPTED', 'FOSTER_FAMILY') AND h.annual_household_income > 20000 AND h.annual_household_income <= 35000) AS from_20_to_35k,
			COUNT(*) FILTER (WHERE COALESCE(h.income_status, '') NOT IN ('MAX_ACCEPTED', 'FOSTER_FAMILY') AND h.annual_household_income > 35000 AND h.annual_household_income <= 55000) AS from_35_to_55k,
			COUNT(*) FILTER (WHERE h.income_status = 'MAX_ACCEPTED') AS max_accepted,
			COUNT(*) FILTER (WHERE h.income_status = 'FOSTER_FAMILY') AS foster_family,
			COUNT(*) AS total
		FROM fees.children c
		LEFT JOIN fees.households h ON c.household_id = h.id
		WHERE c.entry_date <= $1
		  AND (c.exit_date IS NULL OR c.exit_date >= $1)
		  AND c.birth_date > $2
	`, stichtag, u3Threshold)
	if err != nil {
		return domain.U3IncomeBreakdown{}, 0, 0, err
	}

	var totalChildren int
	err = conn(ctx, r.db).GetContext(ctx, &totalChildren, `
		SELECT COUNT(*)
		FROM fees.children
		WHERE entry_date <= $1
		  AND (exit_date IS NULL OR exit_date >= $1)
	`, stichtag)
	if err != nil {
		return domain.U3IncomeBreakdown{}, 0, 0, err
	}

	return domain.U3IncomeBreakdown{
		UpTo20k:      breakdown.UpTo20k,
		From20To35k:  breakdown.From20To35k,
		From35To55k:  breakdown.From35To55k,
		MaxAccepted:  breakdown.MaxAccepted,
		FosterFamily: breakdown.FosterFamily,
		Total:        breakdown.Total,
	}, totalChildren, breakdown.Total, nil
}
