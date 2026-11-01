package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type InstallmentPlan struct {
	ID                uuid.UUID      `json:"id" db:"id"`
	HouseholdID       *uuid.UUID     `json:"householdId" db:"household_id" binding:"optional"`
	DebtorName        string         `json:"debtorName" db:"debtor_name"`
	DebtorAddress     string         `json:"debtorAddress" db:"debtor_address"`
	Emails            pq.StringArray `json:"emails" db:"emails" swaggertype:"array,string"`
	TotalAmount       float64        `json:"totalAmount" db:"total_amount"`
	InstallmentAmount float64        `json:"installmentAmount" db:"installment_amount"`
	IntervalMonths    int            `json:"intervalMonths" db:"interval_months"`
	FirstDueDate      time.Time      `json:"firstDueDate" db:"first_due_date"`
	FinalDueDate      *time.Time     `json:"finalDueDate" db:"final_due_date" binding:"optional"`
	Reference         string         `json:"reference" db:"reference"`
	PayerIBAN         *string        `json:"payerIban" db:"payer_iban" binding:"optional"`
	Note              string         `json:"note" db:"note"`
	Status            string         `json:"status" db:"status"`
	CloseReason       *string        `json:"closeReason" db:"close_reason" binding:"optional"`
	ClosedAt          *time.Time     `json:"closedAt" db:"closed_at" binding:"optional"`
	CreatedAt         time.Time      `json:"createdAt" db:"created_at"`
	UpdatedAt         time.Time      `json:"updatedAt" db:"updated_at"`
	CreatedBy         *uuid.UUID     `json:"createdBy" db:"created_by" binding:"optional"`
}
type InstallmentPayment struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	PlanID        uuid.UUID  `json:"planId" db:"plan_id"`
	TransactionID *uuid.UUID `json:"transactionId" db:"transaction_id" binding:"optional"`
	Amount        float64    `json:"amount" db:"amount"`
	PaidOn        time.Time  `json:"paidOn" db:"paid_on"`
	Source        string     `json:"source" db:"source"`
	Note          string     `json:"note" db:"note"`
	CreatedAt     time.Time  `json:"createdAt" db:"created_at"`
	CreatedBy     *uuid.UUID `json:"createdBy" db:"created_by" binding:"optional"`
}
type InstallmentCharge struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	PlanID     uuid.UUID  `json:"planId" db:"plan_id"`
	Amount     float64    `json:"amount" db:"amount"`
	Label      string     `json:"label" db:"label"`
	CreatedAt  time.Time  `json:"createdAt" db:"created_at"`
	EmailLogID *uuid.UUID `json:"emailLogId" db:"email_log_id" binding:"optional"`
}
type InstallmentContract struct {
	PlanID     uuid.UUID  `json:"planId" db:"plan_id"`
	Filename   string     `json:"filename" db:"filename"`
	SizeBytes  int        `json:"sizeBytes" db:"size_bytes"`
	Content    []byte     `json:"-" db:"content"`
	UploadedAt time.Time  `json:"uploadedAt" db:"uploaded_at"`
	UploadedBy *uuid.UUID `json:"uploadedBy" db:"uploaded_by" binding:"optional"`
}

type InstallmentRate struct {
	Number  int       `json:"number"`
	DueDate time.Time `json:"dueDate"`
	Amount  float64   `json:"amount"`
	Paid    float64   `json:"paid"`
	Status  string    `json:"status"`
}
type InstallmentSummary struct {
	EffectiveTotal float64 `json:"effectiveTotal"`
	Paid           float64 `json:"paid"`
	Rest           float64 `json:"rest"`
	Overdue        float64 `json:"overdue"`
	State          string  `json:"state"`
}
type InstallmentDetail struct {
	Plan      InstallmentPlan      `json:"plan"`
	Summary   InstallmentSummary   `json:"summary"`
	Schedule  []InstallmentRate    `json:"schedule"`
	Payments  []InstallmentPayment `json:"payments"`
	Charges   []InstallmentCharge  `json:"charges"`
	Reminders []EmailLog           `json:"reminders"`
	Contract  *InstallmentContract `json:"contract" binding:"optional"`
}

// CalculateInstallments distributes payments in cents, oldest rate first. Month
// offsets always use the original day, so February does not shift later dates.
func CalculateInstallments(p InstallmentPlan, payments []InstallmentPayment, charges []InstallmentCharge, today time.Time) (InstallmentSummary, []InstallmentRate) {
	total, paid := Cents(p.TotalAmount), int64(0)
	for _, c := range charges {
		total += Cents(c.Amount)
	}
	for _, v := range payments {
		paid += Cents(v.Amount)
	}
	rest := max(int64(0), total-paid)
	summary := InstallmentSummary{EffectiveTotal: Euros(total), Paid: Euros(paid), Rest: Euros(rest), State: "ACTIVE"}
	rates := []InstallmentRate{}
	rate := Cents(p.InstallmentAmount)
	if rate <= 0 || p.IntervalMonths <= 0 {
		return summary, rates
	}
	available := paid
	for remaining, i := total, 0; remaining > 0; i++ {
		amount := min(rate, remaining)
		month := time.Date(p.FirstDueDate.Year(), p.FirstDueDate.Month()+time.Month(i*p.IntervalMonths), 1, 0, 0, 0, 0, time.UTC)
		last := month.AddDate(0, 1, -1).Day()
		due := month.AddDate(0, 0, min(p.FirstDueDate.Day(), last)-1)
		covered := min(available, amount)
		if p.FinalDueDate != nil && covered < amount {
			amount = remaining
			covered = min(available, amount)
			due = *p.FinalDueDate
		}
		status := "OPEN"
		switch {
		case covered == amount:
			status = "PAID"
		case due.Before(today):
			status = "OVERDUE"
			summary.Overdue += Euros(amount - covered)
		case covered > 0:
			status = "PARTIAL"
		}
		rates = append(rates, InstallmentRate{Number: i + 1, DueDate: due, Amount: Euros(amount), Paid: Euros(covered), Status: status})
		remaining -= amount
		available -= covered
	}
	if summary.Overdue > 0 {
		summary.State = "OVERDUE"
	}
	if rest == 0 {
		summary.State = "DONE"
	}
	if p.Status == "CLOSED" {
		summary.State = "CLOSED"
	}
	return summary, rates
}
