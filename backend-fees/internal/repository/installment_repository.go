package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

type PostgresInstallmentRepository struct{ db *sqlx.DB }

func NewPostgresInstallmentRepository(db *sqlx.DB) *PostgresInstallmentRepository {
	return &PostgresInstallmentRepository{db: db}
}

func (r *PostgresInstallmentRepository) List(ctx context.Context) ([]domain.InstallmentPlan, error) {
	v := []domain.InstallmentPlan{}
	err := conn(ctx, r.db).SelectContext(ctx, &v, `SELECT * FROM fees.installment_plans ORDER BY created_at DESC, id`)
	return v, err
}

func (r *PostgresInstallmentRepository) Get(ctx context.Context, id uuid.UUID, lock bool) (*domain.InstallmentDetail, error) {
	d := &domain.InstallmentDetail{Payments: []domain.InstallmentPayment{}, Charges: []domain.InstallmentCharge{}, Reminders: []domain.EmailLog{}}
	q := `SELECT * FROM fees.installment_plans WHERE id=$1`
	if lock {
		q += ` FOR UPDATE`
	}
	err := conn(ctx, r.db).GetContext(ctx, &d.Plan, q, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	for _, x := range []struct {
		dest  interface{}
		query string
	}{
		{&d.Payments, `SELECT * FROM fees.installment_payments WHERE plan_id=$1 ORDER BY paid_on,created_at,id`},
		{&d.Charges, `SELECT * FROM fees.installment_charges WHERE plan_id=$1 ORDER BY created_at,id`},
		{&d.Reminders, `SELECT ` + emailLogColumns + ` FROM fees.email_logs WHERE installment_plan_id=$1 ORDER BY sent_at DESC,id`},
	} {
		if err := conn(ctx, r.db).SelectContext(ctx, x.dest, x.query, id); err != nil {
			return nil, err
		}
	}
	c := &domain.InstallmentContract{}
	err = conn(ctx, r.db).GetContext(ctx, c, `SELECT plan_id,filename,size_bytes,uploaded_at,uploaded_by FROM fees.installment_contracts WHERE plan_id=$1`, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		d.Contract = c
	}

	return d, nil
}

func (r *PostgresInstallmentRepository) SavePlan(ctx context.Context, v *domain.InstallmentPlan) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `INSERT INTO fees.installment_plans (id,household_id,debtor_name,debtor_address,emails,total_amount,installment_amount,interval_months,first_due_date,final_due_date,reference,payer_iban,note,status,close_reason,closed_at,created_at,updated_at,created_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19) ON CONFLICT (id) DO UPDATE SET household_id=EXCLUDED.household_id,debtor_name=EXCLUDED.debtor_name,debtor_address=EXCLUDED.debtor_address,emails=EXCLUDED.emails,total_amount=EXCLUDED.total_amount,installment_amount=EXCLUDED.installment_amount,interval_months=EXCLUDED.interval_months,first_due_date=EXCLUDED.first_due_date,final_due_date=EXCLUDED.final_due_date,reference=EXCLUDED.reference,payer_iban=EXCLUDED.payer_iban,note=EXCLUDED.note,status=EXCLUDED.status,close_reason=EXCLUDED.close_reason,closed_at=EXCLUDED.closed_at,updated_at=EXCLUDED.updated_at`, v.ID, v.HouseholdID, v.DebtorName, v.DebtorAddress, v.Emails, v.TotalAmount, v.InstallmentAmount, v.IntervalMonths, v.FirstDueDate, v.FinalDueDate, v.Reference, v.PayerIBAN, v.Note, v.Status, v.CloseReason, v.ClosedAt, v.CreatedAt, v.UpdatedAt, v.CreatedBy)
	return err
}

func (r *PostgresInstallmentRepository) SavePayment(ctx context.Context, v *domain.InstallmentPayment) error {
	if v.TransactionID != nil {
		var occupied bool
		if err := conn(ctx, r.db).GetContext(ctx, &occupied, `SELECT `+transactionOccupiedSQL, v.TransactionID); err != nil {
			return err
		}
		if occupied {
			return ErrDuplicate
		}
	}
	_, err := conn(ctx, r.db).ExecContext(ctx, `INSERT INTO fees.installment_payments (id,plan_id,transaction_id,amount,paid_on,source,note,created_at,created_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, v.ID, v.PlanID, v.TransactionID, v.Amount, v.PaidOn, v.Source, v.Note, v.CreatedAt, v.CreatedBy)
	return err
}

func (r *PostgresInstallmentRepository) SaveCharge(ctx context.Context, v *domain.InstallmentCharge) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `INSERT INTO fees.installment_charges (id,plan_id,amount,label,created_at,email_log_id) VALUES ($1,$2,$3,$4,$5,$6)`, v.ID, v.PlanID, v.Amount, v.Label, v.CreatedAt, v.EmailLogID)
	return err
}

func (r *PostgresInstallmentRepository) SaveContract(ctx context.Context, v *domain.InstallmentContract) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `INSERT INTO fees.installment_contracts (plan_id,filename,size_bytes,content,uploaded_at,uploaded_by) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (plan_id) DO UPDATE SET filename=EXCLUDED.filename,size_bytes=EXCLUDED.size_bytes,content=EXCLUDED.content,uploaded_at=EXCLUDED.uploaded_at,uploaded_by=EXCLUDED.uploaded_by`, v.PlanID, v.Filename, v.SizeBytes, v.Content, v.UploadedAt, v.UploadedBy)
	return err
}

func (r *PostgresInstallmentRepository) Contract(ctx context.Context, id uuid.UUID) (*domain.InstallmentContract, error) {
	v := &domain.InstallmentContract{}
	err := conn(ctx, r.db).GetContext(ctx, v, `SELECT * FROM fees.installment_contracts WHERE plan_id=$1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return v, err
}

func (r *PostgresInstallmentRepository) DeleteContract(ctx context.Context, id uuid.UUID) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `DELETE FROM fees.installment_contracts WHERE plan_id=$1`, id)
	return err
}

func (r *PostgresInstallmentRepository) DeletePayment(ctx context.Context, id, payment uuid.UUID) error {
	result, err := conn(ctx, r.db).ExecContext(ctx, `DELETE FROM fees.installment_payments WHERE plan_id=$1 AND id=$2`, id, payment)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return ErrNotFound
	}
	return err
}

func (r *PostgresInstallmentRepository) LockTransaction(ctx context.Context, id uuid.UUID) error {
	var found uuid.UUID
	err := conn(ctx, r.db).GetContext(ctx, &found, `SELECT id FROM fees.bank_transactions WHERE id=$1 FOR UPDATE`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
