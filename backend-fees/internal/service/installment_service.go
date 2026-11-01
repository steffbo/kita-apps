package service

import (
	"context"
	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
	"math"
	"net/mail"
	"strings"
	"time"
)

type InstallmentRequest struct {
	HouseholdID       *uuid.UUID `json:"householdId" binding:"optional"`
	DebtorName        string     `json:"debtorName"`
	DebtorAddress     string     `json:"debtorAddress"`
	Emails            []string   `json:"emails"`
	TotalAmount       float64    `json:"totalAmount"`
	InstallmentAmount float64    `json:"installmentAmount"`
	IntervalMonths    int        `json:"intervalMonths"`
	FirstDueDate      string     `json:"firstDueDate"`
	Reference         string     `json:"reference"`
	PayerIBAN         *string    `json:"payerIban" binding:"optional"`
	Note              string     `json:"note"`
	TransactionID     *uuid.UUID `json:"transactionId" binding:"optional"`
}
type InstallmentPaymentRequest struct {
	TransactionID *uuid.UUID `json:"transactionId" binding:"optional"`
	Amount        float64    `json:"amount"`
	PaidOn        string     `json:"paidOn"`
	Note          string     `json:"note"`
}
type InstallmentService struct {
	repo         repository.InstallmentRepository
	households   repository.HouseholdRepository
	transactions repository.TransactionRepository
	tx           *repository.TxManager
	reminders    *ReminderService
}

func NewInstallmentService(r repository.InstallmentRepository, h repository.HouseholdRepository, t repository.TransactionRepository, tx *repository.TxManager, rem *ReminderService) *InstallmentService {
	return &InstallmentService{r, h, t, tx, rem}
}
func (s *InstallmentService) Detail(ctx context.Context, id uuid.UUID) (*domain.InstallmentDetail, error) {
	d, err := s.repo.Get(ctx, id, false)
	if err != nil {
		return nil, err
	}
	d.Summary, d.Schedule = domain.CalculateInstallments(d.Plan, d.Payments, d.Charges, util.Today())
	return d, nil
}
func (s *InstallmentService) List(ctx context.Context, state string) ([]domain.InstallmentDetail, error) {
	if state == "" {
		state = "active"
	}
	if state != "active" && state != "overdue" && state != "done" && state != "all" {
		return nil, ErrInvalidInput
	}
	plans, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := []domain.InstallmentDetail{}
	for _, p := range plans {
		d, err := s.Detail(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		v := d.Summary.State
		if state == "all" || state == "active" && (v == "ACTIVE" || v == "OVERDUE") || state == "overdue" && v == "OVERDUE" || state == "done" && (v == "DONE" || v == "CLOSED") {
			out = append(out, *d)
		}
	}
	return out, nil
}
func validInstallmentAmount(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && domain.Cents(v) > 0 && v <= 9999999999.99 && math.Abs(v-domain.Euros(domain.Cents(v))) < 0.000001
}
func (s *InstallmentService) Save(ctx context.Context, id uuid.UUID, req InstallmentRequest, user uuid.UUID) (*domain.InstallmentDetail, error) {
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		now := util.Now()
		p := domain.InstallmentPlan{ID: id, Status: "ACTIVE", CreatedAt: now, CreatedBy: &user}
		if id == uuid.Nil {
			p.ID = uuid.New()
		} else {
			d, err := s.repo.Get(ctx, id, true)
			if err != nil {
				return err
			}
			p = d.Plan
			if req.TransactionID != nil {
				return ErrInvalidInput
			}
		}
		due, err := time.Parse("2006-01-02", req.FirstDueDate)
		if err != nil || !validInstallmentAmount(req.TotalAmount) || !validInstallmentAmount(req.InstallmentAmount) {
			return ErrInvalidInput
		}
		interval := req.IntervalMonths
		if interval == 0 {
			interval = 1
		}
		switch interval {
		case 1, 2, 3, 6, 12:
		default:
			return ErrInvalidInput
		}

		name := strings.TrimSpace(req.DebtorName)
		if req.HouseholdID != nil {
			h, err := s.households.GetByID(ctx, *req.HouseholdID)
			if err != nil {
				return err
			}
			name = h.Name
		}
		if name == "" {
			return ErrInvalidInput
		}
		emails := []string{}
		seen := map[string]bool{}
		for _, e := range req.Emails {
			e = strings.ToLower(strings.TrimSpace(e))
			a, err := mail.ParseAddress(e)
			if err != nil || a.Address != e {
				return ErrInvalidInput
			}
			if !seen[e] {
				emails = append(emails, e)
				seen[e] = true
			}
		}
		p.HouseholdID = req.HouseholdID
		p.DebtorName = name
		p.DebtorAddress = req.DebtorAddress
		p.Emails = emails
		p.TotalAmount = req.TotalAmount
		p.InstallmentAmount = req.InstallmentAmount
		p.IntervalMonths = interval
		p.FirstDueDate = due
		p.Reference = req.Reference
		p.Note = req.Note
		p.PayerIBAN = req.PayerIBAN
		p.UpdatedAt = now
		if p.PayerIBAN != nil {
			v := normalizeIBAN(*p.PayerIBAN)
			p.PayerIBAN = &v
		}
		if err := s.repo.SavePlan(ctx, &p); err != nil {
			return err
		}
		id = p.ID
		if req.TransactionID != nil {
			return s.addPayment(ctx, id, InstallmentPaymentRequest{TransactionID: req.TransactionID}, user)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Detail(ctx, id)
}
func (s *InstallmentService) addPayment(ctx context.Context, id uuid.UUID, req InstallmentPaymentRequest, user uuid.UUID) error {
	d, err := s.repo.Get(ctx, id, true)
	if err != nil {
		return err
	}
	if d.Plan.Status != "ACTIVE" {
		return ErrInvalidInput
	}
	p := domain.InstallmentPayment{ID: uuid.New(), PlanID: id, Amount: req.Amount, Source: "MANUAL", Note: req.Note, CreatedAt: util.Now(), CreatedBy: &user, TransactionID: req.TransactionID}
	if req.TransactionID != nil {
		if req.Amount != 0 || req.PaidOn != "" {
			return ErrInvalidInput
		}
		if err := s.repo.LockTransaction(ctx, *req.TransactionID); err != nil {
			return err
		}
		t, err := s.transactions.GetByID(ctx, *req.TransactionID)
		if err != nil {
			return err
		}
		p.Amount = t.Amount
		p.PaidOn = t.BookingDate
		p.Source = "BANK"
		if t.Description != nil {
			p.Note = *t.Description
		}
	} else {
		p.PaidOn, err = time.Parse("2006-01-02", req.PaidOn)
		if err != nil {
			return ErrInvalidInput
		}
	}
	if !validInstallmentAmount(p.Amount) {
		return ErrInvalidInput
	}
	return s.repo.SavePayment(ctx, &p)
}
func (s *InstallmentService) Payment(ctx context.Context, id uuid.UUID, req InstallmentPaymentRequest, user uuid.UUID) error {
	return s.tx.WithTx(ctx, func(ctx context.Context) error { return s.addPayment(ctx, id, req, user) })
}
func (s *InstallmentService) DeletePayment(ctx context.Context, id, payment uuid.UUID) error {
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		if _, err := s.repo.Get(ctx, id, true); err != nil {
			return err
		}
		return s.repo.DeletePayment(ctx, id, payment)
	})
}
func (s *InstallmentService) Terms(ctx context.Context, id uuid.UUID, reason *string, date *string) error {
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		d, err := s.repo.Get(ctx, id, true)
		if err != nil {
			return err
		}
		p := d.Plan
		if reason != nil {
			v := strings.TrimSpace(*reason)
			if v == "" {
				return ErrInvalidInput
			}
			p.Status = "CLOSED"
			p.CloseReason = &v
			now := util.Now()
			p.ClosedAt = &now
		} else {
			p.FinalDueDate = nil
			if date != nil {
				v, err := time.Parse("2006-01-02", *date)
				if err != nil {
					return ErrInvalidInput
				}
				p.FinalDueDate = &v
			}
		}
		p.UpdatedAt = util.Now()
		return s.repo.SavePlan(ctx, &p)
	})
}
func (s *InstallmentService) Contract(ctx context.Context, id uuid.UUID) (*domain.InstallmentContract, error) {
	return s.repo.Contract(ctx, id)
}
func (s *InstallmentService) SaveContract(ctx context.Context, c *domain.InstallmentContract) error {
	if len(c.Content) == 0 || len(c.Content) > 10*1024*1024 || !strings.HasPrefix(string(c.Content), "%PDF") {
		return ErrInvalidInput
	}
	c.SizeBytes = len(c.Content)
	c.UploadedAt = util.Now()
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		if _, err := s.repo.Get(ctx, c.PlanID, true); err != nil {
			return err
		}
		return s.repo.SaveContract(ctx, c)
	})
}
func (s *InstallmentService) DeleteContract(ctx context.Context, id uuid.UUID) error {
	if _, err := s.Detail(ctx, id); err != nil {
		return err
	}
	return s.repo.DeleteContract(ctx, id)
}
