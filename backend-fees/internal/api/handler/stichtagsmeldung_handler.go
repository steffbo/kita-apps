package handler

import (
	"net/http"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// StichtagsmeldungHandler handles Stichtagsmeldung-related requests.
type StichtagsmeldungHandler struct {
	stichtagService *service.StichtagsmeldungService
}

// NewStichtagsmeldungHandler creates a new Stichtagsmeldung handler.
func NewStichtagsmeldungHandler(stichtagService *service.StichtagsmeldungService) *StichtagsmeldungHandler {
	return &StichtagsmeldungHandler{
		stichtagService: stichtagService,
	}
}

// StichtagsmeldungStatsResponse represents the Stichtagsmeldung stats response.
// @Description Stichtagsmeldung statistics for quarterly reporting
type StichtagsmeldungStatsResponse struct {
	NextStichtag        string                    `json:"nextStichtag" example:"2026-03-15"`
	DaysUntilStichtag   int                       `json:"daysUntilStichtag" example:"37"`
	U3IncomeBreakdown   U3IncomeBreakdownResponse `json:"u3IncomeBreakdown"`
	TotalChildrenInKita int                       `json:"totalChildrenInKita" example:"45"`
} //@name StichtagsmeldungStats

// U3IncomeBreakdownResponse represents U3 children grouped by income.
// @Description U3 children income breakdown (foster families excluded)
type U3IncomeBreakdownResponse struct {
	UpTo20k     int `json:"upTo20k" example:"5"`
	From20To35k int `json:"from20To35k" example:"8"`
	From35To55k int `json:"from35To55k" example:"12"`
	Total       int `json:"total" example:"25"`
} //@name U3IncomeBreakdown

// GetStats handles GET /stichtagsmeldung/stats
// @Summary Get Stichtagsmeldung statistics
// @Description Get statistics for quarterly Stichtagsmeldung reporting including next Stichtag date and U3 income breakdown
// @Tags Stichtagsmeldung
// @Produce json
// @Security BearerAuth
// @Success 200 {object} StichtagsmeldungStatsResponse "Stichtagsmeldung statistics"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /stichtagsmeldung/stats [get]
func (h *StichtagsmeldungHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.stichtagService.GetStats(r.Context())
	if err != nil {
		response.InternalError(w, "failed to get Stichtagsmeldung stats")
		return
	}

	resp := StichtagsmeldungStatsResponse{
		NextStichtag:      stats.NextStichtag.Format("2006-01-02"),
		DaysUntilStichtag: stats.DaysUntilStichtag,
		U3IncomeBreakdown: U3IncomeBreakdownResponse{
			UpTo20k:     stats.U3IncomeBreakdown.UpTo20k,
			From20To35k: stats.U3IncomeBreakdown.From20To35k,
			From35To55k: stats.U3IncomeBreakdown.From35To55k,
			Total:       stats.U3IncomeBreakdown.Total,
		},
		TotalChildrenInKita: stats.TotalChildrenInKita,
	}

	response.Success(w, resp)
}
