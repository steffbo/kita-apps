package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/request"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

// CalculateChildcareFee handles GET /childcare-fee/calculate
// @Summary Calculate childcare fee
// @Description Calculate the monthly childcare fee based on income, child age, and care hours
// @Tags Fees
// @Produce json
// @Security BearerAuth
// @Param childAgeType query string false "Child age type" default(krippe) Enums(krippe, kindergarten)
// @Param income query number true "Annual net household income"
// @Param siblingsCount query int false "Number of siblings" default(1)
// @Param careHours query int false "Weekly care hours" default(30) Enums(30, 35, 40, 45, 50, 55)
// @Param highestRate query bool false "Apply highest rate" default(false)
// @Param fosterFamily query bool false "Foster family (uses average rate)" default(false)
// @Param date query string false "Reference date (YYYY-MM-DD) selecting the fee schedule; default today"
// @Success 200 {object} domain.ChildcareFeeResult "Calculated fee"
// @Failure 400 {object} response.ErrorBody "Invalid income value or date"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Router /childcare-fee/calculate [get]
func (h *FeeHandler) CalculateChildcareFee(w http.ResponseWriter, r *http.Request) {
	// Parse child age type (default: krippe)
	childAgeType := domain.ChildAgeType(request.GetQueryString(r, "childAgeType", "krippe"))
	if childAgeType != domain.ChildAgeTypeKrippe && childAgeType != domain.ChildAgeTypeKindergarten {
		childAgeType = domain.ChildAgeTypeKrippe
	}

	// Parse income
	incomeStr := request.GetQueryString(r, "income", "0")
	income, err := strconv.ParseFloat(incomeStr, 64)
	if err != nil {
		response.BadRequest(w, "invalid income value")
		return
	}

	// Parse siblings count (default: 1)
	siblingsCountStr := request.GetQueryString(r, "siblingsCount", "1")
	siblingsCount, err := strconv.Atoi(siblingsCountStr)
	if err != nil || siblingsCount < 1 {
		siblingsCount = 1
	}

	// Parse care hours (default: 30, valid: 30, 35, 40, 45, 50, 55)
	careHoursStr := request.GetQueryString(r, "careHours", "30")
	careHours, err := strconv.Atoi(careHoursStr)
	if err != nil {
		careHours = 30
	}
	// Validate care hours - must be one of 30, 35, 40, 45, 50, 55
	validHours := map[int]bool{30: true, 35: true, 40: true, 45: true, 50: true, 55: true}
	if !validHours[careHours] {
		// Round to nearest valid hour
		if careHours < 30 {
			careHours = 30
		} else if careHours > 55 {
			careHours = 55
		} else {
			careHours = ((careHours + 2) / 5) * 5
		}
	}

	// Parse highest rate flag (default: false)
	highestRateStr := request.GetQueryString(r, "highestRate", "false")
	highestRate := highestRateStr == "true" || highestRateStr == "1"

	// Parse foster family flag (default: false)
	fosterFamilyStr := request.GetQueryString(r, "fosterFamily", "false")
	fosterFamily := fosterFamilyStr == "true" || fosterFamilyStr == "1"

	input := domain.ChildcareFeeInput{
		ChildAgeType:  childAgeType,
		NetIncome:     income,
		SiblingsCount: siblingsCount,
		CareHours:     careHours,
		HighestRate:   highestRate,
		FosterFamily:  fosterFamily,
	}

	date := util.Today()
	if dateStr := request.GetQueryString(r, "date", ""); dateStr != "" {
		parsed, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			response.BadRequest(w, "invalid date format (expected YYYY-MM-DD)")
			return
		}
		date = parsed
	}
	schedule, err := h.feeService.ScheduleAt(r.Context(), date)
	if err != nil {
		if !writeNoFeeScheduleError(w, err) {
			response.InternalError(w, "failed to load fee schedule: "+err.Error())
		}
		return
	}

	response.Success(w, schedule.Config.CalculateChildcareFee(input))
}

// writeNoFeeScheduleError answers 400 when no fee regulation covers the requested
// date (e.g. before the first version) and reports whether it did.
func writeNoFeeScheduleError(w http.ResponseWriter, err error) bool {
	if !errors.Is(err, domain.ErrNoFeeSchedule) {
		return false
	}
	response.BadRequest(w, "Für diesen Zeitraum ist keine Beitragsordnung hinterlegt (siehe Beitragsordnung, erste Version gilt ab ihrem Startdatum)")
	return true
}
