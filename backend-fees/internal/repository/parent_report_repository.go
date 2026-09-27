package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

type ParentReport struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	ParentID    uuid.UUID  `json:"parentId" db:"parent_id"`
	UserID      *uuid.UUID `json:"userId" db:"user_id" binding:"optional"`
	HouseholdID uuid.UUID  `json:"householdId" db:"household_id"`
	Topic       string     `json:"topic" db:"topic"`
	ReferenceID *uuid.UUID `json:"referenceId" db:"reference_id" binding:"optional"`
	Message     string     `json:"message" db:"message"`
	Status      string     `json:"status" db:"status"`
	CreatedAt   time.Time  `json:"createdAt" db:"created_at"`
	ResolvedAt  *time.Time `json:"resolvedAt" db:"resolved_at" binding:"optional"`
	ResolvedBy  *uuid.UUID `json:"resolvedBy" db:"resolved_by" binding:"optional"`
}

func (r *ParentAccountRepository) ReferenceBelongs(ctx context.Context, topic string, id, householdID uuid.UUID) (bool, error) {
	var query string
	switch topic {
	case "CHILD":
		query = `SELECT EXISTS(SELECT 1 FROM fees.children WHERE id=$1 AND household_id=$2)`
	case "FEE":
		query = `SELECT EXISTS(SELECT 1 FROM fees.fee_expectations fe JOIN fees.children c ON c.id=fe.child_id WHERE fe.id=$1 AND c.household_id=$2)`
	case "PARENT_WORK":
		query = `SELECT EXISTS(SELECT 1 FROM fees.parent_work_entries WHERE id=$1 AND household_id=$2)`
	default:
		return false, nil
	}
	var ok bool
	err := conn(ctx, r.db).GetContext(ctx, &ok, query, id, householdID)
	return ok, err
}
func (r *ParentAccountRepository) CreateReport(ctx context.Context, v *ParentReport) error {
	v.ID = uuid.New()
	v.Status = "OPEN"
	v.CreatedAt = util.Now()
	_, err := conn(ctx, r.db).ExecContext(ctx, `INSERT INTO fees.parent_reports
 (id,parent_id,user_id,household_id,topic,reference_id,message,status,created_at)
 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, v.ID, v.ParentID, v.UserID, v.HouseholdID, v.Topic, v.ReferenceID, v.Message, v.Status, v.CreatedAt)
	return err
}
func (r *ParentAccountRepository) OwnReports(ctx context.Context, householdID uuid.UUID) ([]ParentReport, error) {
	out := []ParentReport{}
	err := conn(ctx, r.db).SelectContext(ctx, &out, `SELECT * FROM fees.parent_reports WHERE household_id=$1 ORDER BY created_at DESC,id DESC`, householdID)
	return out, err
}
func (r *ParentAccountRepository) StaffReports(ctx context.Context, status string) ([]ParentReport, error) {
	out := []ParentReport{}
	err := conn(ctx, r.db).SelectContext(ctx, &out, `SELECT * FROM fees.parent_reports WHERE $1='ALL' OR status=$1 ORDER BY created_at DESC,id DESC`, status)
	return out, err
}
func (r *ParentAccountRepository) ResolveReport(ctx context.Context, id, userID uuid.UUID) (*ParentReport, error) {
	var v ParentReport
	err := conn(ctx, r.db).GetContext(ctx, &v, `UPDATE fees.parent_reports SET status='DONE',resolved_at=$2,resolved_by=$3 WHERE id=$1 AND status='OPEN' RETURNING *`, id, util.Now(), userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &v, err
}
