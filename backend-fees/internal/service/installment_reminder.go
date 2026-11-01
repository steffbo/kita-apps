package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
	"strings"
	"time"
)

type InstallmentReminderRequest struct {
	InstallmentNumbers []int   `json:"installmentNumbers"`
	ChargeReminderFee  bool    `json:"chargeReminderFee"`
	IncludeQR          *bool   `json:"includeQR" binding:"optional"`
	Subject            *string `json:"subject" binding:"optional"`
	Body               *string `json:"body" binding:"optional"`
}
type InstallmentReminderPreview struct {
	Recipients     []string      `json:"recipients"`
	Subject        string        `json:"subject"`
	Body           string        `json:"body"`
	Deadline       time.Time     `json:"deadline"`
	TotalAmount    float64       `json:"totalAmount"`
	Stage          ReminderStage `json:"stage"`
	QRImageDataURL *string       `json:"qrImageDataUrl,omitempty" binding:"optional"`
	Warnings       []string      `json:"warnings"`
}

func installmentReminderStage(d *domain.InstallmentDetail, today time.Time) (ReminderStage, bool) {
	if d.Plan.Status != "ACTIVE" || d.Summary.Overdue <= 0 {
		return ReminderStageInitial, false
	}
	if len(d.Reminders) == 0 {
		return ReminderStageInitial, true
	}
	log := d.Reminders[0]
	var payload struct {
		Deadline string `json:"deadline"`
	}
	if log.Payload == nil || json.Unmarshal(*log.Payload, &payload) != nil {
		return ReminderStageFinal, false
	}
	deadline, err := time.Parse("2006-01-02", payload.Deadline)
	return ReminderStageFinal, err == nil && deadline.Before(today)
}
func (s *InstallmentService) ReminderCases(ctx context.Context) ([]domain.InstallmentDetail, error) {
	all, err := s.List(ctx, "overdue")
	if err != nil {
		return nil, err
	}
	out := []domain.InstallmentDetail{}
	for _, d := range all {
		if _, ok := installmentReminderStage(&d, util.Today()); ok {
			out = append(out, d)
		}
	}
	return out, nil
}
func (s *InstallmentService) reminderPreview(ctx context.Context, d *domain.InstallmentDetail, req InstallmentReminderRequest) (*InstallmentReminderPreview, error) {
	stage, ok := installmentReminderStage(d, util.Today())
	if !ok {
		return nil, ErrInvalidInput
	}
	if req.ChargeReminderFee && stage != ReminderStageFinal {
		return nil, ErrInvalidInput
	}
	if len(req.InstallmentNumbers) == 0 {
		return nil, ErrInvalidInput
	}
	selected := map[int]bool{}
	var cents int64
	lines := []string{}
	for _, n := range req.InstallmentNumbers {
		if n < 1 || n > len(d.Schedule) || selected[n] {
			return nil, ErrInvalidInput
		}
		selected[n] = true
		r := d.Schedule[n-1]
		if r.Status != "OVERDUE" {
			return nil, ErrInvalidInput
		}
		open := domain.Cents(r.Amount) - domain.Cents(r.Paid)
		cents += open
		lines = append(lines, fmt.Sprintf("Rate %d (%s): %s", n, r.DueDate.Format("02.01.2006"), formatCurrencyEUR(domain.Euros(open))))
	}
	if req.ChargeReminderFee {
		cents += domain.Cents(domain.ReminderFeeAmount)
		lines = append(lines, "Mahngebühr: "+formatCurrencyEUR(domain.ReminderFeeAmount))
	}
	p := &InstallmentReminderPreview{Recipients: []string(d.Plan.Emails), Deadline: defaultReminderDeadline(util.Today()), TotalAmount: domain.Euros(cents), Stage: stage, Warnings: []string{}}
	p.Subject = "Zahlungserinnerung – Ratenzahlung"
	if stage == ReminderStageFinal {
		p.Subject = "Mahnung – Ratenzahlung"
	}
	settings, err := s.reminders.GetPaymentSettings(ctx)
	if err != nil {
		return nil, err
	}
	greeting := "Guten Tag " + d.Plan.DebtorName + ",\n\nbitte überweisen Sie"
	if d.Plan.HouseholdID != nil {
		greeting = "Hallo " + d.Plan.DebtorName + ",\n\nbitte überweist"
	}
	p.Body = fmt.Sprintf("%s den offenen Betrag von %s bis zum %s.\n\n%s\n\nBezug: %s\nEmpfänger: %s\nIBAN: %s\n\nVielen Dank\nKnirpsenstadt e.V.", greeting, formatCurrencyEUR(p.TotalAmount), p.Deadline.Format("02.01.2006"), strings.Join(lines, "\n"), d.Plan.Reference, settings.RecipientName, formatIBANForEmail(settings.IBAN))
	if req.Subject != nil {
		p.Subject = *req.Subject
	}
	if req.Body != nil {
		p.Body = *req.Body
	}
	if strings.TrimSpace(p.Subject) == "" || strings.TrimSpace(p.Body) == "" {
		return nil, ErrInvalidInput
	}
	if len(p.Recipients) == 0 {
		p.Warnings = append(p.Warnings, "Keine E-Mail-Adressen hinterlegt")
	}
	if req.IncludeQR == nil || *req.IncludeQR {
		qr, err := buildPaymentQRCode(settings, p.TotalAmount, "Ratenzahlung "+d.Plan.Reference+" "+d.Plan.DebtorName)
		if err != nil {
			return nil, err
		}
		if qr != nil {
			p.QRImageDataURL = &qr.DataURL
		}
	}
	return p, nil
}
func (s *InstallmentService) PreviewReminder(ctx context.Context, id uuid.UUID, req InstallmentReminderRequest) (*InstallmentReminderPreview, error) {
	d, err := s.Detail(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.reminderPreview(ctx, d, req)
}

// SendReminder serializes sends per plan. A failed SMTP call rolls back both the
// log and the optional charge; as with fee reminders SMTP is not transactional.
func (s *InstallmentService) SendReminder(ctx context.Context, id uuid.UUID, req InstallmentReminderRequest, user uuid.UUID) (*InstallmentReminderPreview, error) {
	var p *InstallmentReminderPreview
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		d, err := s.repo.Get(ctx, id, true)
		if err != nil {
			return err
		}
		d.Summary, d.Schedule = domain.CalculateInstallments(d.Plan, d.Payments, d.Charges, util.Today())
		p, err = s.reminderPreview(ctx, d, req)
		if err != nil {
			return err
		}
		if len(p.Recipients) == 0 || s.reminders.emailSender == nil || !s.reminders.emailSender.IsEnabled() {
			return ErrInvalidInput
		}
		payload, err := json.Marshal(struct {
			Stage              ReminderStage `json:"stage"`
			Deadline           string        `json:"deadline"`
			InstallmentNumbers []int         `json:"installmentNumbers"`
			ChargeReminderFee  bool          `json:"chargeReminderFee"`
		}{p.Stage, p.Deadline.Format("2006-01-02"), req.InstallmentNumbers, req.ChargeReminderFee})
		if err != nil {
			return err
		}
		raw := json.RawMessage(payload)
		typ := domain.EmailLogTypeInstallmentInitial
		if p.Stage == ReminderStageFinal {
			typ = domain.EmailLogTypeInstallmentFinal
		}
		log := domain.EmailLog{ID: uuid.New(), SentAt: util.Now(), ToEmail: strings.Join(p.Recipients, ", "), Subject: p.Subject, Body: &p.Body, EmailType: typ, Payload: &raw, SentBy: &user, HouseholdID: d.Plan.HouseholdID, InstallmentPlanID: &id}
		if user == uuid.Nil {
			log.SentBy = nil
		}
		if err := s.reminders.emailLogRepo.Create(ctx, &log); err != nil {
			return err
		}
		if req.ChargeReminderFee {
			if err := s.repo.SaveCharge(ctx, &domain.InstallmentCharge{ID: uuid.New(), PlanID: id, Amount: domain.ReminderFeeAmount, Label: "Mahngebühr", CreatedAt: util.Now(), EmailLogID: &log.ID}); err != nil {
				return err
			}
		}
		if p.QRImageDataURL != nil {
			png, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(*p.QRImageDataURL, "data:image/png;base64,"))
			if err != nil {
				return err
			}
			return s.reminders.emailSender.SendTextAndHTMLEmailMulti(p.Recipients, p.Subject, p.Body, buildReminderEmailHTML(p.Body, reminderEmailQRCodeCID), reminderEmailQRCodeCID, png)
		}
		return s.reminders.emailSender.SendTextEmailMulti(p.Recipients, p.Subject, p.Body)
	})
	return p, err
}
