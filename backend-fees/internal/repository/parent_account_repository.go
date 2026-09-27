package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

// ParentAccountRepository reads parent-owned data and writes audited contact changes.
type ParentAccountRepository struct{ db *sqlx.DB }

func NewParentAccountRepository(db *sqlx.DB) *ParentAccountRepository {
	return &ParentAccountRepository{db: db}
}

type ParentAccount struct {
	Parent    domain.Parent
	Household *domain.Household
}

func (r *ParentAccountRepository) Account(ctx context.Context, userID uuid.UUID) (*ParentAccount, error) {
	var parentID uuid.UUID
	err := conn(ctx, r.db).GetContext(ctx, &parentID,
		`SELECT parent_id FROM fees.users WHERE id=$1 AND parent_id IS NOT NULL`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var parent domain.Parent
	err = conn(ctx, r.db).GetContext(ctx, &parent, `SELECT * FROM fees.parents WHERE id=$1`, parentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	result := &ParentAccount{Parent: parent}
	if parent.HouseholdID != nil {
		var household domain.Household
		err = conn(ctx, r.db).GetContext(ctx, &household,
			`SELECT * FROM fees.households WHERE id=$1`, *parent.HouseholdID)
		if err != nil {
			return nil, err
		}
		result.Household = &household
	}
	return result, nil
}

func (r *ParentAccountRepository) Parent(ctx context.Context, id uuid.UUID) (*domain.Parent, error) {
	var parent domain.Parent
	err := conn(ctx, r.db).GetContext(ctx, &parent, `SELECT * FROM fees.parents WHERE id=$1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &parent, err
}

func (r *ParentAccountRepository) Children(ctx context.Context, parent domain.Parent) ([]domain.Child,
	error) {
	ids := []uuid.UUID{}
	if parent.HouseholdID != nil {
		err := conn(ctx, r.db).SelectContext(ctx, &ids,
			`SELECT id FROM fees.children WHERE household_id=$1 ORDER BY entry_date,id`, *parent.HouseholdID)
		if err != nil {
			return nil, err
		}
	} else {
		err := conn(ctx, r.db).SelectContext(ctx, &ids,
			`SELECT child_id FROM fees.child_parents WHERE parent_id=$1 ORDER BY child_id`, parent.ID)
		if err != nil {
			return nil, err
		}
	}
	childRepo := NewPostgresChildRepository(r.db)
	result := make([]domain.Child, 0, len(ids))
	for _, id := range ids {
		child, err := childRepo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		result = append(result, *child)
	}
	return result, nil
}

func (r *ParentAccountRepository) OtherParents(ctx context.Context,
	parent domain.Parent) ([]domain.Parent, error) {
	result := []domain.Parent{}
	var err error
	if parent.HouseholdID != nil {
		err = conn(ctx, r.db).SelectContext(ctx, &result,
			`SELECT * FROM fees.parents WHERE household_id=$1 AND id<>$2 ORDER BY last_name,first_name`,
			*parent.HouseholdID, parent.ID)
	} else {
		err = conn(ctx, r.db).SelectContext(ctx, &result, `SELECT DISTINCT p.* FROM fees.parents p
            JOIN fees.child_parents cp ON cp.parent_id=p.id
            WHERE p.id<>$1 AND cp.child_id IN
                (SELECT child_id FROM fees.child_parents WHERE parent_id=$1)
            ORDER BY p.last_name,p.first_name`, parent.ID)
	}
	return result, err
}

type ParentFeeRow struct {
	ID         uuid.UUID `json:"id" db:"id"`
	ChildID    uuid.UUID `json:"childId" db:"child_id"`
	ChildName  string    `json:"childName" db:"child_name"`
	FeeType    string    `json:"feeType" db:"fee_type"`
	Year       int       `json:"year" db:"year"`
	Month      *int      `json:"month" db:"month" binding:"optional"`
	Amount     float64   `json:"amount" db:"amount"`
	DueDate    time.Time `json:"dueDate" db:"due_date"`
	PaidAmount float64   `json:"paidAmount" db:"paid_amount"`
	Status     string    `json:"status" db:"-"`
}

func (r *ParentAccountRepository) Fees(ctx context.Context, parent domain.Parent,
	year int) ([]ParentFeeRow, error) {
	rows := []ParentFeeRow{}
	err := conn(ctx, r.db).SelectContext(ctx, &rows, `SELECT fe.id,fe.child_id,
        c.first_name || ' ' || c.last_name AS child_name,fe.fee_type,fe.year,fe.month,
        fe.amount,fe.due_date,COALESCE(SUM(pm.amount),0) AS paid_amount
        FROM fees.fee_expectations fe JOIN fees.children c ON c.id=fe.child_id
        LEFT JOIN fees.payment_matches pm ON pm.expectation_id=fe.id
        WHERE fe.year=$2 AND (c.household_id=$3 OR ($3::uuid IS NULL AND EXISTS
            (SELECT 1 FROM fees.child_parents cp WHERE cp.child_id=c.id AND cp.parent_id=$1)))
        GROUP BY fe.id,c.first_name,c.last_name ORDER BY fe.due_date DESC,fe.id DESC`,
		parent.ID, year, parent.HouseholdID)
	if err != nil {
		return nil, err
	}
	today := util.Today()
	for i := range rows {
		switch {
		case domain.IsPaid(rows[i].PaidAmount, rows[i].Amount):
			rows[i].Status = string(domain.FeeStatusPaid)
		case rows[i].DueDate.Before(today):
			rows[i].Status = string(domain.FeeStatusOverdue)
		default:
			rows[i].Status = string(domain.FeeStatusOpen)
		}
	}
	return rows, nil
}
