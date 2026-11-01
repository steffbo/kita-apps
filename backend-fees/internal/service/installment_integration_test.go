package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/handler"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

func TestInstallmentLifecycle(t *testing.T) {
	requireTestDatabase(t)
	cleanupTestData()
	ctx := context.Background()
	repo := repository.NewPostgresInstallmentRepository(testDB)
	txs := repository.NewPostgresTransactionRepository(testDB)
	matches := repository.NewPostgresMatchRepository(testDB)
	sender := &fakeReminderSender{enabled: true}
	rem := newReminderCaseService(sender)
	svc := service.NewInstallmentService(repo, repository.NewPostgresHouseholdRepository(testDB), txs, repository.NewTxManager(testDB), rem)
	tx, err := createTestTransaction(txs, "DE12345678901234567890", 150, util.Today().AddDate(0, -2, 0), "Ratenzahlung")
	if err != nil {
		t.Fatal(err)
	}
	req := service.InstallmentRequest{DebtorName: "Test Schuldner", Emails: []string{"TEST@example.test", "test@example.test"}, TotalAmount: 350, InstallmentAmount: 100, IntervalMonths: 1, FirstDueDate: util.Today().AddDate(0, -3, 0).Format("2006-01-02"), PayerIBAN: tx.PayerIBAN, TransactionID: &tx.ID}
	d, err := svc.Save(ctx, uuid.Nil, req, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	defer testDB.Exec(`DELETE FROM fees.installment_plans WHERE id=$1`, d.Plan.ID)
	if len(d.Payments) != 1 || d.Schedule[0].Status != "PAID" || d.Schedule[1].Paid != 50 || d.Summary.Rest != 200 || len(d.Plan.Emails) != 1 {
		t.Fatalf("detail %+v", d)
	}
	rows, _, err := txs.ListUnmatched(ctx, "", "date", "desc", 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range rows {
		if v.ID == tx.ID {
			t.Fatal("assigned transaction unmatched")
		}
	}
	rows, _, err = txs.ListMatched(ctx, "", "date", "desc", 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, v := range rows {
		if v.ID == tx.ID {
			found = v.InstallmentPayment != nil && v.InstallmentPayment.PlanID == d.Plan.ID
		}
	}
	if !found {
		t.Fatal("matched link missing")
	}
	if _, err := txs.DeleteUnmatchedByIBAN(ctx, *tx.PayerIBAN); err != nil {
		t.Fatal(err)
	}
	if _, err := txs.GetByID(ctx, tx.ID); err != nil {
		t.Fatal("deleted assigned transaction", err)
	}
	// Both service and repository refuse mixed ledger assignments.
	if err := svc.Payment(ctx, d.Plan.ID, service.InstallmentPaymentRequest{TransactionID: &tx.ID}, uuid.New()); err == nil {
		t.Fatal("duplicate assignment accepted")
	}
	if err := matches.Create(ctx, &domain.PaymentMatch{ID: uuid.New(), TransactionID: tx.ID, ExpectationID: uuid.New(), Amount: 10, MatchType: domain.MatchTypeManual, MatchedAt: util.Now()}); err != repository.ErrDuplicate {
		t.Fatal("mixed fee assignment", err)
	}
	c := &domain.InstallmentContract{PlanID: d.Plan.ID, Filename: "Vertrag.pdf", Content: []byte("%PDF-1.7\nexample")}
	if err := svc.SaveContract(ctx, c); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Contract(ctx, d.Plan.ID)
	if err != nil || string(got.Content) != string(c.Content) {
		t.Fatal("contract roundtrip", got, err)
	}
	if err := svc.SaveContract(ctx, &domain.InstallmentContract{PlanID: d.Plan.ID, Content: []byte("not PDF")}); err == nil {
		t.Fatal("invalid PDF accepted")
	}
	if err := svc.DeleteContract(ctx, d.Plan.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Contract(ctx, d.Plan.ID); err != repository.ErrNotFound {
		t.Fatal(err)
	}
	cases, err := svc.ReminderCases(ctx)
	if err != nil || len(cases) != 1 {
		t.Fatal("cases", cases, err)
	}
	numbers := []int{}
	for _, r := range d.Schedule {
		if r.Status == "OVERDUE" {
			numbers = append(numbers, r.Number)
		}
	}
	rr := service.InstallmentReminderRequest{InstallmentNumbers: numbers}
	preview, err := svc.PreviewReminder(ctx, d.Plan.ID, rr)
	if err != nil || preview.Stage != service.ReminderStageInitial || !strings.Contains(preview.Body, "überweisen Sie") {
		t.Fatal(preview, err)
	}
	sender.fail = true
	if _, err := svc.SendReminder(ctx, d.Plan.ID, rr, uuid.Nil); err == nil {
		t.Fatal("SMTP failure accepted")
	}
	d, _ = svc.Detail(ctx, d.Plan.ID)
	if len(d.Reminders) != 0 {
		t.Fatal("failed send logged")
	}
	sender.fail = false
	if _, err := svc.SendReminder(ctx, d.Plan.ID, rr, uuid.Nil); err != nil {
		t.Fatal(err)
	}
	cases, err = svc.ReminderCases(ctx)
	if err != nil || len(cases) != 0 {
		t.Fatal("deadline not respected", err)
	}
	d, err = svc.Detail(ctx, d.Plan.ID)
	if err != nil || len(d.Reminders) != 1 {
		t.Fatal(d, err)
	}
	var payload map[string]interface{}
	json.Unmarshal(*d.Reminders[0].Payload, &payload)
	payload["deadline"] = util.Today().AddDate(0, 0, -1).Format("2006-01-02")
	data, _ := json.Marshal(payload)
	if _, err := testDB.Exec(`UPDATE fees.email_logs SET payload=$1 WHERE id=$2`, string(data), d.Reminders[0].ID); err != nil {
		t.Fatal(err)
	}
	rr.ChargeReminderFee = true
	preview, err = svc.SendReminder(ctx, d.Plan.ID, rr, uuid.Nil)
	if err != nil || preview.Stage != service.ReminderStageFinal {
		t.Fatal(preview, err)
	}
	d, _ = svc.Detail(ctx, d.Plan.ID)
	if len(d.Charges) != 1 || d.Charges[0].Amount != domain.ReminderFeeAmount || d.Summary.EffectiveTotal != 350+domain.ReminderFeeAmount {
		t.Fatal("charge", d)
	}
	// Existing unmatch endpoint uses this repository operation atomically.
	imports := service.NewImportService(txs, repository.NewPostgresFeeRepository(testDB), repository.NewPostgresChildRepository(testDB), matches, repository.NewPostgresKnownIBANRepository(testDB), repository.NewPostgresWarningRepository(testDB), repository.NewTxManager(testDB), repository.NewPostgresFeeScheduleRepository(testDB))
	unmatched, err := imports.UnmatchTransaction(ctx, tx.ID, false)
	var n int64
	if unmatched != nil {
		n = unmatched.MatchesRemoved
	}
	if err != nil || n != 1 {
		t.Fatal(n, err)
	}
	d, _ = svc.Detail(ctx, d.Plan.ID)
	if len(d.Payments) != 0 || d.Summary.Paid != 0 {
		t.Fatal("unmatch", d)
	}
	rows, _, err = txs.ListUnmatched(ctx, "", "date", "desc", 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, v := range rows {
		if v.ID == tx.ID {
			found = v.InstallmentPlanSuggestion != nil
		}
	}
	if !found {
		t.Fatal("suggestion missing")
	}
	req.TransactionID = nil
	req.Emails = nil
	d, err = svc.Save(ctx, d.Plan.ID, req, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	// Expire latest deadline, then reject send without recipients.
	_, err = testDB.Exec(`UPDATE fees.email_logs SET payload=$1 WHERE installment_plan_id=$2`, string(data), d.Plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SendReminder(ctx, d.Plan.ID, rr, uuid.Nil); err == nil {
		t.Fatal("send without recipients")
	}
}

// The HTTP roundtrip exercises multipart parsing, PDF validation and headers.
func TestInstallmentContractHTTP(t *testing.T) {
	requireTestDatabase(t)
	ctx := context.Background()
	repo := repository.NewPostgresInstallmentRepository(testDB)
	svc := service.NewInstallmentService(repo, repository.NewPostgresHouseholdRepository(testDB), repository.NewPostgresTransactionRepository(testDB), repository.NewTxManager(testDB), newReminderCaseService(&fakeReminderSender{enabled: true}))
	d, err := svc.Save(ctx, uuid.Nil, service.InstallmentRequest{DebtorName: "HTTP", TotalAmount: 100, InstallmentAmount: 100, FirstDueDate: "2026-01-31"}, uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}
	defer testDB.Exec(`DELETE FROM fees.installment_plans WHERE id=$1`, d.Plan.ID)
	h := handler.NewInstallmentHandler(svc)
	router := chi.NewRouter()
	router.Put("/{id}/contract", h.UploadContract)
	router.Get("/{id}/contract", h.DownloadContract)
	router.Delete("/{id}/contract", h.DeleteContract)
	for _, tc := range []struct {
		content string
		status  int
	}{{"%PDF-1.7\ncontract", 204}, {"invalid", 400}, {"%PDF" + strings.Repeat("x", 10*1024*1024), 413}} {
		var b bytes.Buffer
		mw := multipart.NewWriter(&b)
		f, _ := mw.CreateFormFile("file", "Vertrag.pdf")
		f.Write([]byte(tc.content))
		mw.Close()
		req := httptest.NewRequest("PUT", "/"+d.Plan.ID.String()+"/contract", &b)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != tc.status {
			t.Fatalf("upload %d: %s", w.Code, w.Body.String())
		}
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/"+d.Plan.ID.String()+"/contract", nil))
	if w.Code != 200 || w.Body.String() != "%PDF-1.7\ncontract" || w.Header().Get("Content-Type") != "application/pdf" || !strings.Contains(w.Header().Get("Content-Disposition"), "Vertrag.pdf") {
		t.Fatalf("download %+v", w)
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("DELETE", "/"+d.Plan.ID.String()+"/contract", nil))
	if w.Code != 204 {
		t.Fatal(w)
	}
}
