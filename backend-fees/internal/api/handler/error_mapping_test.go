package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

type errorCase struct {
	name        string
	err         error
	wantStatus  int
	wantMessage string // substring; empty = not checked
}

func runErrorCases(t *testing.T, write func(http.ResponseWriter, error), cases []errorCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			write(rec, tc.err)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (%s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantMessage != "" && !strings.Contains(rec.Body.String(), tc.wantMessage) {
				t.Fatalf("body %s does not contain %q", rec.Body.String(), tc.wantMessage)
			}
		})
	}
}

func TestWriteNoFeeScheduleError(t *testing.T) {
	rec := httptest.NewRecorder()
	if writeNoFeeScheduleError(rec, errors.New("boom")) {
		t.Fatal("unrelated error must not be handled")
	}
	if rec.Body.Len() != 0 {
		t.Fatal("nothing may be written for unrelated errors")
	}

	rec = httptest.NewRecorder()
	if !writeNoFeeScheduleError(rec, fmt.Errorf("generate 2024-12: %w", domain.ErrNoFeeSchedule)) {
		t.Fatal("wrapped ErrNoFeeSchedule must be handled")
	}
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "keine Beitragsordnung") {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandleEinstufungError(t *testing.T) {
	runErrorCases(t, handleEinstufungError, []errorCase{
		{name: "no fee schedule", err: fmt.Errorf("x: %w", domain.ErrNoFeeSchedule), wantStatus: http.StatusBadRequest, wantMessage: "keine Beitragsordnung"},
		{name: "not found", err: fmt.Errorf("x: %w", service.ErrNotFound), wantStatus: http.StatusNotFound},
		{name: "invalid input", err: service.ErrInvalidInput, wantStatus: http.StatusBadRequest, wantMessage: "Haushalt"},
		{name: "unexpected", err: errors.New("db down"), wantStatus: http.StatusInternalServerError},
	})
}

func TestWriteFeeScheduleError(t *testing.T) {
	runErrorCases(t, writeFeeScheduleError, []errorCase{
		{name: "invalid input keeps detail", err: fmt.Errorf("%w: validFrom must be the first of a month", service.ErrInvalidInput), wantStatus: http.StatusBadRequest, wantMessage: "validFrom must be the first of a month"},
		{name: "conflict keeps detail", err: fmt.Errorf("%w: version already active", service.ErrConflict), wantStatus: http.StatusConflict, wantMessage: "version already active"},
		{name: "not found", err: service.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "unexpected", err: errors.New("db down"), wantStatus: http.StatusInternalServerError},
	})
}

func TestWriteReminderCaseError(t *testing.T) {
	feeID := uuid.New()
	runErrorCases(t, writeReminderCaseError, []errorCase{
		{name: "invalid input", err: service.ErrInvalidInput, wantStatus: http.StatusBadRequest},
		{name: "wrapped invalid input", err: fmt.Errorf("stage: %w", service.ErrInvalidInput), wantStatus: http.StatusBadRequest},
		{name: "email disabled", err: service.ErrEmailDisabled, wantStatus: http.StatusServiceUnavailable},
		{name: "household not found", err: fmt.Errorf("load: %w", repository.ErrNotFound), wantStatus: http.StatusNotFound},
		{name: "unexpected", err: errors.New("smtp down"), wantStatus: http.StatusInternalServerError},
	})

	rec := httptest.NewRecorder()
	writeReminderCaseError(rec, fmt.Errorf("send: %w", &service.CaseConflictError{Reason: "fees changed", FeeIDs: []uuid.UUID{feeID}}))
	if rec.Code != http.StatusConflict {
		t.Fatalf("conflict status = %d", rec.Code)
	}
	var body ReminderCaseConflictResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Message != "fees changed" || len(body.FeeIDs) != 1 || body.FeeIDs[0] != feeID.String() {
		t.Fatalf("conflict body = %+v", body)
	}
}
