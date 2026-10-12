package service_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/email"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

type recordingLetterSender struct {
	enabled     bool
	to          []string
	subject     string
	attachments []email.Attachment
}

func (s *recordingLetterSender) IsEnabled() bool { return s.enabled }

func (s *recordingLetterSender) SendTextEmailWithAttachments(to []string, subject, _ string,
	attachments []email.Attachment) error {
	s.to, s.subject, s.attachments = to, subject, attachments
	return nil
}

func TestEinstufungLetter_PDFAndSend(t *testing.T) {
	f := newEinstufungFixture(t)
	child, household := f.child(t, "TLET", domain.IncomeStatusUnknown, date(2025, time.March, 1), date(2026, time.January, 1))
	parents := repository.NewPostgresParentRepository(testDB)
	for _, address := range []string{"brief-a@example.test", "brief-b@example.test"} {
		address := address
		if err := parents.Create(f.ctx, &domain.Parent{ID: uuid.New(), HouseholdID: &household.ID,
			FirstName: "Elter", LastName: "TLET", Email: &address, IncomeStatus: domain.IncomeStatusUnknown}); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { testDB.Exec("DELETE FROM fees.email_logs WHERE household_id = $1", household.ID) })

	created, err := f.service.Create(f.ctx, service.CreateEinstufungInput{
		ChildID: child.ID, Year: 2026, IncomeBasis: domain.IncomeBasisHighestRate, ChildrenCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	sender := &recordingLetterSender{}
	letters := service.NewEinstufungLetterService(f.service, f.einstufungRepo,
		repository.NewPostgresParentWorkRepository(testDB), repository.NewPostgresSettingsRepository(testDB),
		repository.NewPostgresEmailLogRepository(testDB), sender)

	file, err := letters.PDF(f.ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(file.Data, []byte("%PDF-")) || file.Filename != "Einstufung_2026_TLET_Test.pdf" {
		t.Fatalf("pdf %q starts with %q", file.Filename, file.Data[:8])
	}

	input := service.SendEinstufungLetterInput{Subject: "Einstufung 2026", Body: "Hallo,\n\nanbei die Einstufung."}
	if _, err := letters.Send(f.ctx, created.ID, uuid.New(), input); !errors.Is(err, service.ErrEmailDisabled) {
		t.Fatalf("disabled sender: err = %v", err)
	}
	sender.enabled = true
	if _, err := letters.Send(f.ctx, created.ID, uuid.New(), service.SendEinstufungLetterInput{Subject: " "}); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("empty mail: err = %v", err)
	}

	before, err := f.einstufungRepo.GetByID(f.ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	result, err := letters.Send(f.ctx, created.ID, uuid.New(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(sender.to) != 2 || len(result.Recipients) != 2 || sender.subject != "Einstufung 2026" {
		t.Fatalf("sent to %v (%v), subject %q", sender.to, result.Recipients, sender.subject)
	}
	if len(sender.attachments) != 1 || sender.attachments[0].ContentType != "application/pdf" ||
		!bytes.HasPrefix(sender.attachments[0].Data, []byte("%PDF-")) {
		t.Fatalf("attachments = %+v", sender.attachments)
	}

	stored, err := f.service.GetByID(f.ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.LetterSentAt == nil || stored.LetterSentAt.Sub(result.SentAt).Abs() > time.Second {
		t.Fatalf("letterSentAt = %v, want %v", stored.LetterSentAt, result.SentAt)
	}
	if !stored.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("sending changed updated_at: %v -> %v", before.UpdatedAt, stored.UpdatedAt)
	}

	logs, err := repository.NewPostgresEmailLogRepository(testDB).ListByHouseholdAndTypes(f.ctx, household.ID,
		[]domain.EmailLogType{domain.EmailLogTypeEinstufung})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].ToEmail != "brief-a@example.test, brief-b@example.test" || logs[0].SentBy == nil {
		t.Fatalf("logs = %+v", logs)
	}
}
