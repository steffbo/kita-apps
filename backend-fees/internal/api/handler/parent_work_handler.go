package handler

import "net/http"

// ParentWorkPing handles GET /parent-work/ping while the module is being built.
// @Summary Check parent-work route access
// @Tags Parent Work
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string "Route available"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 403 {object} response.ErrorBody "Insufficient permissions"
// @Router /parent-work/ping [get]
func ParentWorkPing(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
