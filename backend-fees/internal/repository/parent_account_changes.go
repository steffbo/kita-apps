package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/auth"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

type DataChange struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	EntityType string     `json:"entityType" db:"entity_type"`
	EntityID   uuid.UUID  `json:"entityId" db:"entity_id"`
	ParentID   *uuid.UUID `json:"parentId" db:"parent_id" binding:"optional"`
	UserID     *uuid.UUID `json:"userId" db:"user_id" binding:"optional"`
	Field      string     `json:"field" db:"field"`
	OldValue   *string    `json:"oldValue" db:"old_value" binding:"optional"`
	NewValue   *string    `json:"newValue" db:"new_value" binding:"optional"`
	ChangedAt  time.Time  `json:"changedAt" db:"changed_at"`
	// ImpersonatedBy is the admin who made the change while acting as UserID.
	ImpersonatedBy *uuid.UUID `json:"impersonatedBy" db:"impersonated_by" binding:"optional"`
}

type ContactChange = DataChange

func sameValue(a, b *string) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }

func auditChange(ctx context.Context, tx *sqlx.Tx, entityType string, entityID uuid.UUID,
	actorParentID *uuid.UUID, userID uuid.UUID, field string, oldValue, newValue *string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO fees.data_changes
 (entity_type,entity_id,parent_id,user_id,field,old_value,new_value,changed_at,impersonated_by)
 VALUES ($1,$2,$3,$4,$5,$6,$7,
  $8,$9)`,
		entityType,
		entityID,
		actorParentID,
		userID,
		field,
		oldValue,
		newValue,
		util.Now(),
		auth.ImpersonatorFrom(ctx),
	)
	return err
}

// SaveContact writes changed fields and the linked login email atomically.
func (r *ParentAccountRepository) SaveContact(ctx context.Context, parentID, userID uuid.UUID,
	fields map[string]*string) error {
	return r.saveContact(ctx, parentID, userID, &parentID, fields)
}

// SaveContactAs lets a parent edit another parent of the same household (actor is the editing parent).
func (r *ParentAccountRepository) SaveContactAs(ctx context.Context, parentID, userID,
	actorParentID uuid.UUID, fields map[string]*string) error {
	return r.saveContact(ctx, parentID, userID, &actorParentID, fields)
}

// HasLogin reports whether a user account is linked to the parent.
func (r *ParentAccountRepository) HasLogin(ctx context.Context, parentID uuid.UUID) (bool, error) {
	var ok bool
	err := conn(ctx, r.db).GetContext(ctx, &ok,
		`SELECT EXISTS(SELECT 1 FROM fees.users WHERE parent_id=$1)`, parentID)
	return ok, err
}

func (r *ParentAccountRepository) SaveStaffContact(ctx context.Context, parentID, userID uuid.UUID,
	fields map[string]*string) error {
	return r.saveContact(ctx, parentID, userID, nil, fields)
}

func (r *ParentAccountRepository) saveContact(ctx context.Context, parentID, userID uuid.UUID,
	actorParentID *uuid.UUID, fields map[string]*string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var old struct {
		Email      *string `db:"email"`
		Phone      *string `db:"phone"`
		Street     *string `db:"street"`
		StreetNo   *string `db:"street_no"`
		PostalCode *string `db:"postal_code"`
		City       *string `db:"city"`
	}
	if err = tx.GetContext(ctx, &old,
		`SELECT email,phone,street,street_no,postal_code,city FROM fees.parents WHERE id=$1 FOR UPDATE`,
		parentID); err != nil {
		return err
	}
	previous := map[string]*string{"email": old.Email, "phone": old.Phone, "street": old.Street,
		"street_no": old.StreetNo, "postal_code": old.PostalCode, "city": old.City}
	var linkedID uuid.UUID
	linked := tx.GetContext(ctx, &linkedID, `SELECT id FROM fees.users WHERE parent_id=$1 FOR UPDATE`,
		parentID)
	if linked != nil && !errors.Is(linked, sql.ErrNoRows) {
		return linked
	}
	if linked == nil && fields["email"] == nil {
		return ErrEmailRequired
	}
	for _, field := range []string{"email", "phone", "street", "street_no", "postal_code", "city"} {
		value := fields[field]
		before := previous[field]
		if sameValue(before, value) {
			continue
		}
		if field == "email" && linked == nil {
			if _, err = tx.ExecContext(ctx, `UPDATE fees.users SET email=$2,updated_at=$3 WHERE id=$1`,
				linkedID, *value, util.Now()); err != nil {
				return mapUniqueViolation(err)
			}
		}
		if _, err = tx.ExecContext(ctx, `UPDATE fees.parents SET `+field+`=$2,updated_at=$3 WHERE id=$1`,
			parentID, value, util.Now()); err != nil {
			return err
		}
		if err = auditChange(ctx, tx, "PARENT", parentID, actorParentID, userID, field, before,
			value); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// UpdateLinkedUser changes a PARENT account and its linked contact email in one transaction.
func (r *ParentAccountRepository) UpdateLinkedUser(ctx context.Context, user *domain.User,
	actorID uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var old *string
	if err = tx.GetContext(ctx, &old, `SELECT email FROM fees.parents WHERE id=$1 FOR UPDATE`,
		*user.ParentID); err != nil {
		return err
	}
	if !sameValue(old, &user.Email) {
		if _, err = tx.ExecContext(ctx, `UPDATE fees.parents SET email=$2,updated_at=$3 WHERE id=$1`,
			*user.ParentID, user.Email, util.Now()); err != nil {
			return err
		}
		if err = auditChange(ctx, tx, "PARENT", *user.ParentID, nil, actorID, "email", old,
			&user.Email); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx,
		`UPDATE fees.users SET
	email=$2,first_name=$3,last_name=$4,role=$5,parent_id=$6,is_active=$7,updated_at=$8 WHERE
	id=$1
	`,
		user.ID, user.Email, user.FirstName, user.LastName, user.Role, user.ParentID, user.IsActive,
		util.Now())
	if err != nil {
		return mapUniqueViolation(err)
	}
	return tx.Commit()
}

func (r *ParentAccountRepository) SaveChild(ctx context.Context, child *domain.Child, parentID,
	userID uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var old struct {
		FirstName  string    `db:"first_name"`
		LastName   string    `db:"last_name"`
		BirthDate  time.Time `db:"birth_date"`
		Street     *string   `db:"street"`
		StreetNo   *string   `db:"street_no"`
		PostalCode *string   `db:"postal_code"`
		City       *string   `db:"city"`
	}
	err = tx.GetContext(ctx, &old,
		`SELECT first_name,last_name,birth_date,street,street_no,postal_code,city FROM
	fees.children WHERE id=$1 AND household_id=(SELECT household_id FROM fees.parents WHERE
	id=$2) FOR UPDATE
	`, child.ID, parentID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		`UPDATE fees.children SET
	first_name=$2,last_name=$3,birth_date=$4,street=$5,street_no=$6,postal_code=$7,city=$8,updated_at=$9
	WHERE id=$1
	`, child.ID, child.FirstName, child.LastName, child.BirthDate, child.Street, child.StreetNo, child.PostalCode, child.City, util.Now())
	if err != nil {
		return err
	}
	values := []struct {
		field         string
		before, after *string
	}{
		{"first_name", &old.FirstName, &child.FirstName}, {"last_name", &old.LastName, &child.LastName},
		{"street", old.Street, child.Street}, {"street_no", old.StreetNo, child.StreetNo},
		{"postal_code", old.PostalCode, child.PostalCode}, {"city", old.City, child.City}}
	oldDate, newDate := old.BirthDate.Format("2006-01-02"), child.BirthDate.Format("2006-01-02")
	values = append(values, struct {
		field         string
		before, after *string
	}{"birth_date", &oldDate, &newDate})
	for _, v := range values {
		if !sameValue(v.before, v.after) {
			if err = auditChange(ctx, tx, "CHILD", child.ID, &parentID, userID, v.field, v.before,
				v.after); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (r *ParentAccountRepository) Changes(ctx context.Context, entityType string,
	id uuid.UUID) ([]DataChange, error) {
	out := []DataChange{}
	err := conn(ctx, r.db).SelectContext(ctx, &out,
		`SELECT * FROM fees.data_changes WHERE entity_type=$1 AND entity_id=$2 ORDER BY changed_at
	DESC,id DESC
	`, entityType, id)
	return out, err
}
func (r *ParentAccountRepository) ContactChanges(ctx context.Context, id uuid.UUID) ([]ContactChange, error) {
	return r.Changes(ctx, "PARENT", id)
}

type Activity struct {
	Type            string     `json:"type" db:"type"`
	At              time.Time  `json:"at" db:"at"`
	ParentID        *uuid.UUID `json:"parentId,omitempty" db:"parent_id" binding:"optional"`
	ParentName      *string    `json:"parentName,omitempty" db:"parent_name" binding:"optional"`
	HouseholdID     *uuid.UUID `json:"householdId,omitempty" db:"household_id" binding:"optional"`
	HouseholdName   *string    `json:"householdName,omitempty" db:"household_name" binding:"optional"`
	ChildID         *uuid.UUID `json:"childId,omitempty" db:"child_id" binding:"optional"`
	ChildName       *string    `json:"childName,omitempty" db:"child_name" binding:"optional"`
	TargetName      *string    `json:"targetName,omitempty" db:"target_name" binding:"optional"`
	ReportID        *uuid.UUID `json:"reportId,omitempty" db:"report_id" binding:"optional"`
	Topic           *string    `json:"topic,omitempty" db:"topic" binding:"optional"`
	Message         *string    `json:"message,omitempty" db:"message" binding:"optional"`
	Field           *string    `json:"field,omitempty" db:"field" binding:"optional"`
	OldValue        *string    `json:"oldValue" db:"old_value" binding:"optional"`
	NewValue        *string    `json:"newValue" db:"new_value" binding:"optional"`
	DurationMinutes *int       `json:"durationMinutes,omitempty" db:"duration_minutes" binding:"optional"`
	Occasion        *string    `json:"occasion,omitempty" db:"occasion" binding:"optional"`
	Status          *string    `json:"status,omitempty" db:"status" binding:"optional"`
}

func (r *ParentAccountRepository) Activity(ctx context.Context, limit int) ([]Activity, error) {
	out := []Activity{}
	err := conn(ctx, r.db).SelectContext(ctx, &out, `SELECT * FROM (
 SELECT CASE WHEN d.entity_type='CHILD' THEN 'CHILD_CHANGED' ELSE 'CONTACT_CHANGED' END AS type,
 d.changed_at AS at,d.parent_id,p.first_name||' '||p.last_name AS parent_name,p.household_id,
 h.name AS household_name,CASE WHEN d.entity_type='CHILD' THEN d.entity_id END AS child_id,
 CASE WHEN d.entity_type='CHILD' THEN c.first_name||' '||c.last_name END AS child_name,
 CASE WHEN d.entity_type='PARENT' AND d.entity_id<>d.parent_id THEN tp.first_name||' '||tp.last_name
 END AS target_name,
 NULL::uuid AS report_id,NULL::varchar AS topic,NULL::text AS message,d.field,d.old_value,d.new_value,
 NULL::int AS duration_minutes,NULL::text AS occasion,NULL::varchar AS status
 FROM fees.data_changes d JOIN fees.parents p ON p.id=d.parent_id
 LEFT JOIN fees.households h ON h.id=p.household_id LEFT JOIN fees.children c ON c.id=d.entity_id
 AND d.entity_type='CHILD' LEFT JOIN fees.parents tp ON tp.id=d.entity_id AND d.entity_type='PARENT'
 WHERE d.parent_id IS NOT NULL
 UNION ALL
 SELECT 'PARENT_WORK_SUBMITTED',e.created_at,p.id,p.first_name||' '||p.last_name,e.household_id,h.name,
 NULL::uuid,NULL::text,NULL::text,NULL::uuid,NULL::varchar,NULL::text,NULL::varchar,NULL::text,NULL::text,
 e.duration_minutes,e.occasion,e.status
 FROM fees.parent_work_entries e JOIN fees.households h ON h.id=e.household_id
 JOIN fees.users u ON u.id=e.created_by JOIN fees.parents p ON p.id=u.parent_id WHERE e.source='PARENT'
 UNION ALL
 SELECT 'REPORT_CREATED',r.created_at,r.parent_id,p.first_name||' '||p.last_name,r.household_id,h.name,
 NULL::uuid,NULL::text,NULL::text,r.id,r.topic,left(r.message,200),NULL::varchar,NULL::text,NULL::text,
 NULL::int,
 NULL::text,r.status
 FROM fees.parent_reports r JOIN fees.parents p ON p.id=r.parent_id JOIN fees.households h ON
 h.id=r.household_id
 ) events ORDER BY at DESC LIMIT $1`, limit)
	return out, err
}

var ErrEmailRequired = errors.New("E-Mail wird für die Anmeldung benötigt")

// SaveStaffParent persists the complete staff edit, linked login, and contact audit together.
func (r *ParentAccountRepository) SaveStaffParent(ctx context.Context, parent *domain.Parent,
	actorID uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var old struct {
		Email      *string `db:"email"`
		Phone      *string `db:"phone"`
		Street     *string `db:"street"`
		StreetNo   *string `db:"street_no"`
		PostalCode *string `db:"postal_code"`
		City       *string `db:"city"`
	}
	err = tx.GetContext(ctx, &old,
		`SELECT email,phone,street,street_no,postal_code,city FROM fees.parents WHERE id=$1 FOR UPDATE`,
		parent.ID)
	if err != nil {
		return err
	}
	var linkedID uuid.UUID
	linked := tx.GetContext(ctx, &linkedID, `SELECT id FROM fees.users WHERE parent_id=$1 FOR UPDATE`,
		parent.ID)
	if linked != nil && !errors.Is(linked, sql.ErrNoRows) {
		return linked
	}
	if linked == nil && parent.Email == nil {
		return ErrEmailRequired
	}
	if linked == nil && !sameValue(old.Email, parent.Email) {
		if _, err = tx.ExecContext(ctx, `UPDATE fees.users SET email=$2,updated_at=$3 WHERE id=$1`,
			linkedID, *parent.Email, util.Now()); err != nil {
			return mapUniqueViolation(err)
		}
	}
	_, err = tx.ExecContext(ctx,
		`UPDATE fees.parents SET
	household_id=$2,member_id=$3,first_name=$4,last_name=$5,birth_date=$6,email=$7,
	phone=$8,street=$9,street_no=$10,postal_code=$11,city=$12,
	annual_household_income=$13,income_status=$14,updated_at=$15
	WHERE id=$1
	`, parent.ID, parent.HouseholdID, parent.MemberID, parent.FirstName, parent.LastName, parent.BirthDate, parent.Email, parent.Phone, parent.Street, parent.StreetNo, parent.PostalCode, parent.City, parent.AnnualHouseholdIncome, parent.IncomeStatus, util.Now())
	if err != nil {
		return err
	}
	values := []struct {
		field         string
		before, after *string
	}{{"email", old.Email, parent.Email}, {"phone", old.Phone, parent.Phone}, {"street", old.Street,
		parent.Street}, {"street_no", old.StreetNo, parent.StreetNo}, {"postal_code", old.PostalCode,
		parent.PostalCode}, {"city", old.City, parent.City}}
	for _, v := range values {
		if !sameValue(v.before, v.after) {
			if err = auditChange(ctx, tx, "PARENT", parent.ID, nil, actorID, v.field, v.before,
				v.after); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
