package handler

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
	"io"
	"mime"
	"net/http"
	"path/filepath"
)

type InstallmentHandler struct{ svc *service.InstallmentService }

func NewInstallmentHandler(s *service.InstallmentService) *InstallmentHandler {
	return &InstallmentHandler{s}
}

type installmentCloseRequest struct {
	Reason string `json:"reason"`
}
type installmentFinalRequest struct {
	DueDate string `json:"dueDate"`
}

func installmentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound), errors.Is(err, service.ErrNotFound):
		response.NotFound(w, "Ratenzahlung nicht gefunden")
	case errors.Is(err, repository.ErrDuplicate):
		response.Conflict(w, "Buchung bereits zugeordnet")
	case errors.Is(err, service.ErrInvalidInput):
		response.BadRequest(w, "Ungültige Angaben")
	default:
		response.InternalError(w, "Ratenzahlung konnte nicht verarbeitet werden")
	}
}

// List installment plans.
// @Summary List installment plans
// @Tags Installment plans
// @Security BearerAuth
// @Produce json
// @Param state query string false "active|overdue|done|all"
// @Success 200 {array} domain.InstallmentDetail
// @Failure 400,401,403,404,409,500 {object} response.ErrorBody
// @Router /installment-plans [get]
func (h *InstallmentHandler) List(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.List(r.Context(), r.URL.Query().Get("state"))
	if err != nil {
		installmentError(w, err)
		return
	}
	response.JSON(w, 200, v)
}

// Create installment plans.
// @Summary Create installment plans
// @Tags Installment plans
// @Security BearerAuth
// @Produce json
// @Accept json
// @Param request body service.InstallmentRequest true "Request"
// @Success 201 {object} domain.InstallmentDetail
// @Failure 400,401,403,404,409,500 {object} response.ErrorBody
// @Router /installment-plans [post]
func (h *InstallmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req service.InstallmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Ungültiges JSON")
		return
	}
	v, err := h.svc.Save(r.Context(), uuid.Nil, req, parentWorkUser(r))
	if err != nil {
		installmentError(w, err)
		return
	}
	response.JSON(w, 201, v)
}

// Detail installment plans.
// @Summary Detail installment plans
// @Tags Installment plans
// @Security BearerAuth
// @Produce json
// @Param id path string true "Plan UUID"
// @Success 200 {object} domain.InstallmentDetail
// @Failure 400,401,403,404,409,500 {object} response.ErrorBody
// @Router /installment-plans/{id} [get]
func (h *InstallmentHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	v, err := h.svc.Detail(r.Context(), id)
	if err != nil {
		installmentError(w, err)
		return
	}
	response.JSON(w, 200, v)
}

// Update installment plans.
// @Summary Update installment plans
// @Tags Installment plans
// @Security BearerAuth
// @Produce json
// @Param id path string true "Plan UUID"
// @Accept json
// @Param request body service.InstallmentRequest true "Request"
// @Success 200 {object} domain.InstallmentDetail
// @Failure 400,401,403,404,409,500 {object} response.ErrorBody
// @Router /installment-plans/{id} [put]
func (h *InstallmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	var req service.InstallmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Ungültiges JSON")
		return
	}
	v, err := h.svc.Save(r.Context(), id, req, parentWorkUser(r))
	if err != nil {
		installmentError(w, err)
		return
	}
	response.JSON(w, 200, v)
}

// Close installment plans.
// @Summary Close installment plans
// @Tags Installment plans
// @Security BearerAuth
// @Produce json
// @Param id path string true "Plan UUID"
// @Accept json
// @Param request body installmentCloseRequest true "Request"
// @Success 204 "No content"
// @Failure 400,401,403,404,409,500 {object} response.ErrorBody
// @Router /installment-plans/{id}/close [post]
func (h *InstallmentHandler) Close(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	var req installmentCloseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Ungültiges JSON")
		return
	}
	err := h.svc.Terms(r.Context(), id, &req.Reason, nil)
	if err != nil {
		installmentError(w, err)
		return
	}
	response.NoContent(w)
}

// Payment installment plans.
// @Summary Payment installment plans
// @Tags Installment plans
// @Security BearerAuth
// @Produce json
// @Param id path string true "Plan UUID"
// @Accept json
// @Param request body service.InstallmentPaymentRequest true "Request"
// @Success 204 "No content"
// @Failure 400,401,403,404,409,500 {object} response.ErrorBody
// @Router /installment-plans/{id}/payments [post]
func (h *InstallmentHandler) Payment(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	var req service.InstallmentPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Ungültiges JSON")
		return
	}
	err := h.svc.Payment(r.Context(), id, req, parentWorkUser(r))
	if err != nil {
		installmentError(w, err)
		return
	}
	response.NoContent(w)
}

// DeletePayment installment plans.
// @Summary DeletePayment installment plans
// @Tags Installment plans
// @Security BearerAuth
// @Produce json
// @Param id path string true "Plan UUID"
// @Param paymentId path string true "Payment UUID"
// @Success 204 "No content"
// @Failure 400,401,403,404,409,500 {object} response.ErrorBody
// @Router /installment-plans/{id}/payments/{paymentId} [delete]
func (h *InstallmentHandler) DeletePayment(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	payment, ok := parseUUIDParam(w, r, "paymentId")
	if !ok {
		return
	}
	err := h.svc.DeletePayment(r.Context(), id, payment)
	if err != nil {
		installmentError(w, err)
		return
	}
	response.NoContent(w)
}

// Final installment plans.
// @Summary Final installment plans
// @Tags Installment plans
// @Security BearerAuth
// @Produce json
// @Param id path string true "Plan UUID"
// @Accept json
// @Param request body installmentFinalRequest true "Request"
// @Success 204 "No content"
// @Failure 400,401,403,404,409,500 {object} response.ErrorBody
// @Router /installment-plans/{id}/final-installment [put]
func (h *InstallmentHandler) Final(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	var req installmentFinalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Ungültiges JSON")
		return
	}
	err := h.svc.Terms(r.Context(), id, nil, &req.DueDate)
	if err != nil {
		installmentError(w, err)
		return
	}
	response.NoContent(w)
}

// DeleteFinal installment plans.
// @Summary DeleteFinal installment plans
// @Tags Installment plans
// @Security BearerAuth
// @Produce json
// @Param id path string true "Plan UUID"
// @Success 204 "No content"
// @Failure 400,401,403,404,409,500 {object} response.ErrorBody
// @Router /installment-plans/{id}/final-installment [delete]
func (h *InstallmentHandler) DeleteFinal(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	err := h.svc.Terms(r.Context(), id, nil, nil)
	if err != nil {
		installmentError(w, err)
		return
	}
	response.NoContent(w)
}

// ReminderCases installment plans.
// @Summary ReminderCases installment plans
// @Tags Installment plans
// @Security BearerAuth
// @Produce json
// @Success 200 {array} domain.InstallmentDetail
// @Failure 400,401,403,404,409,500 {object} response.ErrorBody
// @Router /installment-plans/reminder-cases [get]
func (h *InstallmentHandler) ReminderCases(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.ReminderCases(r.Context())
	if err != nil {
		installmentError(w, err)
		return
	}
	response.JSON(w, 200, v)
}

// Preview installment plans.
// @Summary Preview installment plans
// @Tags Installment plans
// @Security BearerAuth
// @Produce json
// @Param id path string true "Plan UUID"
// @Accept json
// @Param request body service.InstallmentReminderRequest true "Request"
// @Success 200 {object} service.InstallmentReminderPreview
// @Failure 400,401,403,404,409,500 {object} response.ErrorBody
// @Router /installment-plans/{id}/reminder/preview [post]
func (h *InstallmentHandler) Preview(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	var req service.InstallmentReminderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Ungültiges JSON")
		return
	}
	v, err := h.svc.PreviewReminder(r.Context(), id, req)
	if err != nil {
		installmentError(w, err)
		return
	}
	response.JSON(w, 200, v)
}

// Send installment plans.
// @Summary Send installment plans
// @Tags Installment plans
// @Security BearerAuth
// @Produce json
// @Param id path string true "Plan UUID"
// @Accept json
// @Param request body service.InstallmentReminderRequest true "Request"
// @Success 200 {object} service.InstallmentReminderPreview
// @Failure 400,401,403,404,409,500 {object} response.ErrorBody
// @Router /installment-plans/{id}/reminder/send [post]
func (h *InstallmentHandler) Send(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	var req service.InstallmentReminderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Ungültiges JSON")
		return
	}
	v, err := h.svc.SendReminder(r.Context(), id, req, parentWorkUser(r))
	if err != nil {
		installmentError(w, err)
		return
	}
	response.JSON(w, 200, v)
}

// DeleteContract installment plans.
// @Summary DeleteContract installment plans
// @Tags Installment plans
// @Security BearerAuth
// @Produce json
// @Param id path string true "Plan UUID"
// @Success 204 "No content"
// @Failure 400,401,403,404,409,500 {object} response.ErrorBody
// @Router /installment-plans/{id}/contract [delete]
func (h *InstallmentHandler) DeleteContract(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	err := h.svc.DeleteContract(r.Context(), id)
	if err != nil {
		installmentError(w, err)
		return
	}
	response.NoContent(w)
}

// UploadContract replaces the signed PDF.
// @Summary Upload installment contract
// @Tags Installment plans
// @Security BearerAuth
// @Accept multipart/form-data
// @Param id path string true "Plan UUID"
// @Param file formData file true "PDF (max 10 MB)"
// @Success 204 "No content"
// @Failure 400,401,403,404,413,500 {object} response.ErrorBody
// @Router /installment-plans/{id}/contract [put]
func (h *InstallmentHandler) UploadContract(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	if !parseUpload(w, r) {
		return
	}
	defer r.MultipartForm.RemoveAll()
	f, header, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "PDF fehlt")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 10*1024*1024+1))
	if err != nil {
		response.BadRequest(w, "PDF nicht lesbar")
		return
	}
	if len(data) > 10*1024*1024 {
		response.TooLarge(w, "PDF maximal 10 MB")
		return
	}
	user := parentWorkUser(r)
	err = h.svc.SaveContract(r.Context(), &domain.InstallmentContract{PlanID: id, Filename: filepath.Base(header.Filename), Content: data, UploadedBy: &user})
	if err != nil {
		installmentError(w, err)
		return
	}
	response.NoContent(w)
}

// DownloadContract returns the original PDF.
// @Summary Download installment contract
// @Tags Installment plans
// @Security BearerAuth
// @Produce application/pdf
// @Param id path string true "Plan UUID"
// @Success 200 {file} binary
// @Failure 401,403,404,500 {object} response.ErrorBody
// @Router /installment-plans/{id}/contract [get]
func (h *InstallmentHandler) DownloadContract(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	c, err := h.svc.Contract(r.Context(), id)
	if err != nil {
		installmentError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": c.Filename}))
	w.Write(c.Content)
}
