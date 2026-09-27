package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

type ParentAccountService struct {
	accounts *repository.ParentAccountRepository
	work     *ParentWorkService
}

func NewParentAccountService(accounts *repository.ParentAccountRepository,
	work *ParentWorkService) *ParentAccountService {
	return &ParentAccountService{accounts: accounts, work: work}
}
func mapAccountError(err error) error {
	switch {
	case errors.Is(err, repository.ErrEmailRequired):
		return fmt.Errorf("%w: E-Mail wird für die Anmeldung benötigt", ErrInvalidInput)
	case errors.Is(err, repository.ErrDuplicate):
		return fmt.Errorf("%w: E-Mail-Adresse wird bereits verwendet", ErrConflict)
	case errors.Is(err, repository.ErrNotFound):
		return ErrNotFound
	default:
		return err
	}
}
func (s *ParentAccountService) Account(ctx context.Context, userID uuid.UUID) (*repository.ParentAccount,
	error) {
	account, err := s.accounts.Account(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("%w: Konto ist keinem Elternteil zugeordnet", ErrConflict)
	}
	return account, mapAccountError(err)
}
func (s *ParentAccountService) Overview(ctx context.Context,
	userID uuid.UUID) (*repository.ParentAccount, []domain.Parent, []domain.Child, error) {
	account, err := s.Account(ctx, userID)
	if err != nil {
		return nil, nil, nil, err
	}
	others, err := s.accounts.OtherParents(ctx, account.Parent)
	if err != nil {
		return nil, nil, nil, err
	}
	children, err := s.accounts.Children(ctx, account.Parent)
	return account, others, children, err
}
func (s *ParentAccountService) Fees(ctx context.Context, userID uuid.UUID,
	year int) ([]repository.ParentFeeRow, float64, float64, error) {
	account, err := s.Account(ctx, userID)
	if err != nil {
		return nil, 0, 0, err
	}
	rows, err := s.accounts.Fees(ctx, account.Parent, year)
	if err != nil {
		return nil, 0, 0, err
	}
	var paid, open int64
	for _, v := range rows {
		paid += domain.Cents(v.PaidAmount)
		open += max(0, domain.Cents(v.Amount)-domain.Cents(v.PaidAmount))
	}
	return rows, domain.Euros(open), domain.Euros(paid), nil
}
func (s *ParentAccountService) Work(ctx context.Context, userID uuid.UUID, year int) (*ParentWorkDetail,
	error) {
	account, err := s.Account(ctx, userID)
	if err != nil {
		return nil, err
	}
	if account.Household == nil {
		return nil, fmt.Errorf("%w: Elternteil ist keinem Haushalt zugeordnet", ErrConflict)
	}
	return s.work.Detail(ctx, account.Household.ID, year)
}
func (s *ParentAccountService) Submit(ctx context.Context, userID uuid.UUID,
	v domain.ParentWorkEntry) (*domain.ParentWorkEntry, error) {
	account, err := s.Account(ctx, userID)
	if err != nil {
		return nil, err
	}
	if account.Household == nil {
		return nil, fmt.Errorf("%w: Elternteil ist keinem Haushalt zugeordnet", ErrConflict)
	}
	v.HouseholdID = account.Household.ID
	return s.work.SubmitParentEntry(ctx, v, userID)
}
func (s *ParentAccountService) Withdraw(ctx context.Context, userID,
	id uuid.UUID) (*domain.ParentWorkEntry, error) {
	account, err := s.Account(ctx, userID)
	if err != nil {
		return nil, err
	}
	if account.Household == nil {
		return nil, ErrNotFound
	}
	return s.work.WithdrawParentEntry(ctx, id, account.Household.ID, userID)
}
func cleanContact(v *string) *string {
	if v == nil {
		return nil
	}
	x := strings.TrimSpace(*v)
	if x == "" {
		return nil
	}
	return &x
}
func (s *ParentAccountService) Contact(ctx context.Context, userID uuid.UUID,
	updates map[string]*string) (*domain.Parent, error) {
	account, err := s.Account(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.ParentContact(ctx, userID, account.Parent.ID, updates)
}

// ParentContact changes the contact data of the own parent or another parent of the household.
// Another parent's email stays untouched while it is that parent's login.
func (s *ParentAccountService) ParentContact(ctx context.Context, userID, parentID uuid.UUID,
	updates map[string]*string) (*domain.Parent, error) {
	account, err := s.Account(ctx, userID)
	if err != nil {
		return nil, err
	}
	target := &account.Parent
	if parentID != account.Parent.ID {
		others, err := s.accounts.OtherParents(ctx, account.Parent)
		if err != nil {
			return nil, err
		}
		target = nil
		for i := range others {
			if others[i].ID == parentID {
				target = &others[i]
			}
		}
		if target == nil {
			return nil, ErrNotFound
		}
	}
	p := *target
	fields := map[string]*string{"email": p.Email, "phone": p.Phone, "street": p.Street,
		"street_no": p.StreetNo, "postal_code": p.PostalCode, "city": p.City}
	names := map[string]string{"email": "email", "phone": "phone", "street": "street",
		"streetNo": "street_no", "postalCode": "postal_code", "city": "city"}
	for name, value := range updates {
		key, ok := names[name]
		if !ok {
			return nil, fmt.Errorf("%w: Unbekanntes Kontaktfeld", ErrInvalidInput)
		}
		fields[key] = cleanContact(value)
	}
	email := fields["email"]
	if email != nil {
		parsed, e := mail.ParseAddress(*email)
		if e != nil || parsed.Address != *email || !strings.Contains(*email, "@") {
			return nil, fmt.Errorf("%w: Ungültige E-Mail-Adresse", ErrInvalidInput)
		}
	}
	if p.ID == account.Parent.ID {
		err = s.accounts.SaveContact(ctx, p.ID, userID, fields)
	} else {
		hasLogin, e := s.accounts.HasLogin(ctx, p.ID)
		if e != nil {
			return nil, e
		}
		if hasLogin && !sameContact(p.Email, email) {
			return nil, fmt.Errorf("%w: Die E-Mail-Adresse ist das Login von %s und kann nur dort "+
				"geändert werden", ErrInvalidInput, p.FirstName)
		}
		err = s.accounts.SaveContactAs(ctx, p.ID, userID, account.Parent.ID, fields)
	}
	if err != nil {
		return nil, mapAccountError(err)
	}
	return s.accounts.Parent(ctx, p.ID)
}

func sameContact(a, b *string) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }

// HasLogin reports whether the parent has a linked login.
func (s *ParentAccountService) HasLogin(ctx context.Context, parentID uuid.UUID) (bool, error) {
	return s.accounts.HasLogin(ctx, parentID)
}

type OwnChildInput struct {
	FirstName  string  `json:"firstName"`
	LastName   string  `json:"lastName"`
	BirthDate  string  `json:"birthDate"`
	Street     *string `json:"street" binding:"optional"`
	StreetNo   *string `json:"streetNo" binding:"optional"`
	PostalCode *string `json:"postalCode" binding:"optional"`
	City       *string `json:"city" binding:"optional"`
}

func (s *ParentAccountService) UpdateChild(ctx context.Context, userID, childID uuid.UUID,
	in OwnChildInput) (*domain.Child, error) {
	account, err := s.Account(ctx, userID)
	if err != nil {
		return nil, err
	}
	if account.Household == nil {
		return nil, ErrNotFound
	}
	children, err := s.accounts.Children(ctx, account.Parent)
	if err != nil {
		return nil, err
	}
	var child *domain.Child
	for i := range children {
		if children[i].ID == childID {
			child = &children[i]
			break
		}
	}
	if child == nil {
		return nil, ErrNotFound
	}
	in.FirstName = strings.TrimSpace(in.FirstName)
	in.LastName = strings.TrimSpace(in.LastName)
	birth, err := time.Parse("2006-01-02", in.BirthDate)
	if in.FirstName == "" || in.LastName == "" || err != nil || birth.After(util.Today()) {
		return nil, fmt.Errorf("%w: Pflichtfelder oder Geburtsdatum ungültig", ErrInvalidInput)
	}
	child.FirstName, child.LastName, child.BirthDate = in.FirstName, in.LastName, birth
	child.Street, child.StreetNo, child.PostalCode, child.City = cleanContact(in.Street),
		cleanContact(in.StreetNo), cleanContact(in.PostalCode), cleanContact(in.City)
	if err = s.accounts.SaveChild(ctx, child, account.Parent.ID, userID); err != nil {
		return nil, mapAccountError(err)
	}
	return child, nil
}

type ReportInput struct {
	Topic       string     `json:"topic"`
	ReferenceID *uuid.UUID `json:"referenceId" binding:"optional"`
	Message     string     `json:"message"`
}

func (s *ParentAccountService) CreateReport(ctx context.Context, userID uuid.UUID,
	in ReportInput) (*repository.ParentReport, error) {
	account, err := s.Account(ctx, userID)
	if err != nil {
		return nil, err
	}
	if account.Household == nil {
		return nil, fmt.Errorf("%w: Elternteil ist keinem Haushalt zugeordnet", ErrConflict)
	}
	in.Message = strings.TrimSpace(in.Message)
	if in.Message == "" || utf8.RuneCountInString(in.Message) > 2000 {
		return nil, fmt.Errorf("%w: Nachricht muss 1 bis 2000 Zeichen enthalten", ErrInvalidInput)
	}
	switch in.Topic {
	case "GENERAL", "CHILD", "FEE", "PARENT_WORK", "CONTACT":
	default:
		return nil, fmt.Errorf("%w: Ungültiges Thema", ErrInvalidInput)
	}
	if in.ReferenceID != nil {
		if in.Topic == "GENERAL" || in.Topic == "CONTACT" {
			return nil, fmt.Errorf("%w: Bezug passt nicht zum Thema", ErrInvalidInput)
		}
		ok, err := s.accounts.ReferenceBelongs(ctx, in.Topic, *in.ReferenceID, account.Household.ID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrNotFound
		}
	}
	v := &repository.ParentReport{ParentID: account.Parent.ID, UserID: &userID,
		HouseholdID: account.Household.ID, Topic: in.Topic, ReferenceID: in.ReferenceID,
		Message: in.Message}
	if err = s.accounts.CreateReport(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *ParentAccountService) OwnReports(ctx context.Context,
	userID uuid.UUID) ([]repository.ParentReport, error) {
	account, err := s.Account(ctx, userID)
	if err != nil {
		return nil, err
	}
	if account.Household == nil {
		return []repository.ParentReport{}, nil
	}
	return s.accounts.OwnReports(ctx, account.Household.ID)
}
func (s *ParentAccountService) StaffReports(ctx context.Context,
	status string) ([]repository.ParentReport, error) {
	if status == "" {
		status = "OPEN"
	}
	if status != "OPEN" && status != "DONE" && status != "ALL" {
		return nil, fmt.Errorf("%w: Ungültiger Status", ErrInvalidInput)
	}
	return s.accounts.StaffReports(ctx, status)
}
func (s *ParentAccountService) ResolveReport(ctx context.Context, id,
	userID uuid.UUID, response string) (*repository.ParentReport, error) {
	response = strings.TrimSpace(response)
	if utf8.RuneCountInString(response) > 2000 {
		return nil, fmt.Errorf("%w: Antwort darf höchstens 2000 Zeichen enthalten", ErrInvalidInput)
	}
	var answer *string
	if response != "" {
		answer = &response
	}
	v, err := s.accounts.ResolveReport(ctx, id, userID, answer)
	return v, mapAccountError(err)
}
func (s *ParentAccountService) Changes(ctx context.Context, entityType string,
	id uuid.UUID) ([]repository.DataChange, error) {
	return s.accounts.Changes(ctx, entityType, id)
}
func (s *ParentAccountService) Activity(ctx context.Context, limit int) ([]repository.Activity, error) {
	return s.accounts.Activity(ctx, limit)
}
