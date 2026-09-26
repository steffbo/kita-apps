package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

type ParentWorkService struct {
	repo repository.ParentWorkRepository
}

func NewParentWorkService(repo repository.ParentWorkRepository) *ParentWorkService {
	return &ParentWorkService{repo: repo}
}

type ParentWorkOverview struct {
	KitaYear           int                        `json:"kitaYear"`
	Rule               *domain.ParentWorkRule     `json:"rule,omitempty" binding:"optional"`
	Notice             *string                    `json:"notice,omitempty" binding:"optional"`
	Households         []domain.ParentWorkAccount `json:"households"`
	RequiredMinutes    int                        `json:"requiredMinutes"`
	DoneMinutes        int                        `json:"doneMinutes"`
	OpenMinutes        int                        `json:"openMinutes"`
	MissingAmountCents int                        `json:"missingAmountCents"`
}

type ParentWorkHouseholdOption struct {
	ID       uuid.UUID                 `json:"id"`
	Name     string                    `json:"name"`
	Children []domain.ParentWorkChild  `json:"children"`
	Members  []domain.ParentWorkMember `json:"members"`
	Parents  []domain.ParentWorkParent `json:"parents"`
}

type ParentWorkDetail struct {
	domain.ParentWorkAccount
	Entries    []domain.ParentWorkEntry `json:"entries"`
	BoardTerms []domain.BoardTerm       `json:"boardTerms"`
}

func (s *ParentWorkService) Overview(ctx context.Context, year int) (*ParentWorkOverview, error) {
	if year < 1900 || year > 9998 {
		return nil, fmt.Errorf("%w: ungültiges Kita-Jahr", ErrInvalidInput)
	}
	rules, err := s.repo.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	result := &ParentWorkOverview{KitaYear: year, Households: []domain.ParentWorkAccount{}}
	if len(rules) == 0 || rules[0].ValidFrom.After(domain.ParentWorkYearStart(year)) {
		notice := "Für dieses Kita-Jahr gibt es kein Regelwerk; es ist nicht abrechenbar."
		result.Notice = &notice
		from := domain.ParentWorkYearStart(year)
		snap, err := s.repo.Snapshot(ctx, from, domain.ParentWorkYearStart(year+1).AddDate(0, 0, -1))
		if err != nil {
			return nil, err
		}
		zeroRule := &domain.ParentWorkRule{}
		for _, h := range snap.Households {
			var children []domain.ParentWorkChild
			var terms []domain.BoardTerm
			var entries []domain.ParentWorkEntry
			memberIDs := map[uuid.UUID]bool{}
			for _, child := range snap.Children {
				if child.HouseholdID == h.ID {
					children = append(children, child)
				}
			}
			for _, member := range snap.Members {
				if member.HouseholdID == h.ID {
					memberIDs[member.ID] = true
				}
			}
			for _, term := range snap.Terms {
				if memberIDs[term.MemberID] {
					terms = append(terms, term)
				}
			}
			for _, entry := range snap.Entries {
				if entry.HouseholdID == h.ID {
					entries = append(entries, entry)
				}
			}
			row := domain.CalculateParentWork(year, zeroRule, children, terms, nil, entries, 0)
			row.HouseholdID, row.HouseholdName = h.ID, h.Name
			for _, override := range snap.Overrides {
				if override.HouseholdID == h.ID && override.KitaYear == year {
					row.OverrideMinutes = &override.RequiredMinutes
					row.OverrideReason = &override.Reason
				}
			}
			if row.EntryCount > 0 || row.ExemptReason != nil || row.OverrideMinutes != nil {
				result.Households = append(result.Households, row)
				result.DoneMinutes += row.DoneMinutes
			}
		}
		return result, nil
	}
	firstYear := domain.ParentWorkKitaYear(rules[0].ValidFrom)
	from := domain.ParentWorkYearStart(firstYear)
	until := domain.ParentWorkYearStart(year+1).AddDate(0, 0, -1)
	snap, err := s.repo.Snapshot(ctx, from, until)
	if err != nil {
		return nil, err
	}
	children := map[uuid.UUID][]domain.ParentWorkChild{}
	members := map[uuid.UUID][]domain.ParentWorkMember{}
	termsByMember := map[uuid.UUID][]domain.BoardTerm{}
	entries := map[uuid.UUID][]domain.ParentWorkEntry{}
	overrides := map[uuid.UUID]map[int]*domain.ParentWorkOverride{}
	for _, v := range snap.Children {
		children[v.HouseholdID] = append(children[v.HouseholdID], v)
	}
	for _, v := range snap.Members {
		members[v.HouseholdID] = append(members[v.HouseholdID], v)
	}
	for _, v := range snap.Terms {
		termsByMember[v.MemberID] = append(termsByMember[v.MemberID], v)
	}
	for _, v := range snap.Entries {
		entries[v.HouseholdID] = append(entries[v.HouseholdID], v)
	}
	for i := range snap.Overrides {
		v := &snap.Overrides[i]
		if overrides[v.HouseholdID] == nil {
			overrides[v.HouseholdID] = map[int]*domain.ParentWorkOverride{}
		}
		overrides[v.HouseholdID][v.KitaYear] = v
	}
	carry := map[uuid.UUID]int{}
	for y := firstYear; y <= year; y++ {
		var rule *domain.ParentWorkRule
		for i := range rules {
			if !rules[i].ValidFrom.After(domain.ParentWorkYearStart(y)) {
				rule = &rules[i]
			}
		}
		if y == year {
			result.Rule = rule
		}
		for _, h := range snap.Households {
			var terms []domain.BoardTerm
			for _, m := range members[h.ID] {
				terms = append(terms, termsByMember[m.ID]...)
			}
			row := domain.CalculateParentWork(y, rule, children[h.ID], terms,
				overrides[h.ID][y], entries[h.ID], carry[h.ID])
			row.HouseholdID = h.ID
			row.HouseholdName = h.Name
			carry[h.ID] = row.CarryOutMinutes
			if y != year {
				continue
			}
			if row.RequiredMinutes == 0 && row.EntryCount == 0 && row.ExemptReason == nil &&
				row.OverrideMinutes == nil {
				continue
			}
			result.Households = append(result.Households, row)
			result.RequiredMinutes += row.RequiredMinutes
			result.DoneMinutes += row.DoneMinutes
			result.OpenMinutes += row.OpenMinutes
			result.MissingAmountCents += row.MissingAmountCents
		}
	}
	return result, nil
}

func (s *ParentWorkService) Households(
	ctx context.Context, search string,
) ([]ParentWorkHouseholdOption, error) {
	snap, err := s.repo.Snapshot(ctx, domain.ParentWorkYearStart(1900),
		domain.ParentWorkYearStart(9999))
	if err != nil {
		return nil, err
	}
	result := []ParentWorkHouseholdOption{}
	search = strings.ToLower(strings.TrimSpace(search))
	for _, h := range snap.Households {
		v := ParentWorkHouseholdOption{ID: h.ID, Name: h.Name,
			Children: []domain.ParentWorkChild{}, Members: []domain.ParentWorkMember{},
			Parents: []domain.ParentWorkParent{}}
		match := strings.Contains(strings.ToLower(h.Name), search)
		for _, c := range snap.Children {
			if c.HouseholdID == h.ID {
				v.Children = append(v.Children, c)
				match = match || strings.Contains(strings.ToLower(c.Name), search)
			}
		}
		for _, m := range snap.Members {
			if m.HouseholdID == h.ID {
				v.Members = append(v.Members, m)
				match = match || strings.Contains(strings.ToLower(m.Name), search)
			}
		}
		for _, p := range snap.Parents {
			if p.HouseholdID == h.ID {
				v.Parents = append(v.Parents, p)
				match = match || strings.Contains(strings.ToLower(p.Name), search)
			}
		}
		if match {
			result = append(result, v)
		}
	}
	return result, nil
}

func (s *ParentWorkService) Detail(ctx context.Context, id uuid.UUID, year int) (*ParentWorkDetail, error) {
	exists, err := s.repo.HouseholdExists(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}
	overview, err := s.Overview(ctx, year)
	if err != nil {
		return nil, err
	}
	detail := &ParentWorkDetail{Entries: []domain.ParentWorkEntry{}, BoardTerms: []domain.BoardTerm{}}
	for _, row := range overview.Households {
		if row.HouseholdID == id {
			detail.ParentWorkAccount = row
			break
		}
	}
	if detail.HouseholdID == uuid.Nil {
		options, err := s.Households(ctx, "")
		if err != nil {
			return nil, err
		}
		for _, o := range options {
			if o.ID == id {
				detail.HouseholdID = id
				detail.HouseholdName = o.Name
				break
			}
		}
	}
	from := domain.ParentWorkYearStart(year)
	until := domain.ParentWorkYearStart(year+1).AddDate(0, 0, -1)
	snap, err := s.repo.Snapshot(ctx, from, until)
	if err != nil {
		return nil, err
	}
	memberIDs := map[uuid.UUID]bool{}
	for _, m := range snap.Members {
		if m.HouseholdID == id {
			memberIDs[m.ID] = true
		}
	}
	for _, v := range snap.Entries {
		if v.HouseholdID == id {
			detail.Entries = append(detail.Entries, v)
		}
	}
	allTerms, err := s.repo.ListTerms(ctx)
	if err != nil {
		return nil, err
	}
	for _, v := range allTerms {
		if memberIDs[v.MemberID] {
			detail.BoardTerms = append(detail.BoardTerms, v)
		}
	}
	return detail, nil
}

func (s *ParentWorkService) SaveRule(
	ctx context.Context, id uuid.UUID, v domain.ParentWorkRule,
) (*domain.ParentWorkRule, error) {
	if id != uuid.Nil {
		existing, err := s.repo.GetRule(ctx, id)
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		if !existing.ValidFrom.After(util.Today()) {
			return nil, fmt.Errorf("%w: geltende Regelwerke können nicht geändert werden", ErrConflict)
		}
		v.ID = id
		v.CreatedAt = existing.CreatedAt
	}
	if v.ValidFrom.Month() != time.August || v.ValidFrom.Day() != 1 ||
		!v.ValidFrom.After(util.Today()) || v.HoursPerChildMinutes < 0 ||
		v.MissingHourRateCents < 0 || v.MaxCarryOverMinutes < 0 {
		return nil, fmt.Errorf("%w: Gültig ab muss ein zukünftiger 01.08. sein; "+
			"Werte dürfen nicht negativ sein", ErrInvalidInput)
	}
	if err := s.repo.SaveRule(ctx, &v); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, fmt.Errorf("%w: für diesen 01.08. gibt es bereits ein Regelwerk", ErrConflict)
		}
		return nil, err
	}
	return &v, nil
}
func (s *ParentWorkService) Rules(ctx context.Context) ([]domain.ParentWorkRule, error) {
	return s.repo.ListRules(ctx)
}
func (s *ParentWorkService) SaveEntry(ctx context.Context, id uuid.UUID, v domain.ParentWorkEntry,
	userID uuid.UUID) (*domain.ParentWorkEntry, error) {
	if id != uuid.Nil {
		existing, err := s.repo.GetEntry(ctx, id)
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		if existing.Status == "VOIDED" {
			return nil, fmt.Errorf("%w: stornierte Einträge können nicht bearbeitet werden", ErrConflict)
		}
		v.ID = id
		v.CreatedAt = existing.CreatedAt
		v.CreatedBy = existing.CreatedBy
		v.Source = existing.Source
	} else {
		v.Source = "MANUAL"
		v.CreatedBy = &userID
	}
	v.UpdatedBy = &userID
	if v.HouseholdID == uuid.Nil || v.WorkDate.IsZero() || v.DurationMinutes <= 0 ||
		v.DurationMinutes%15 != 0 || strings.TrimSpace(v.Occasion) == "" ||
		!validEntryStatus(v.Status) {
		return nil, fmt.Errorf("%w: Familie, Datum, Anlass, Status und positive "+
			"Viertelstunden sind erforderlich", ErrInvalidInput)
	}
	exists, err := s.repo.HouseholdExists(ctx, v.HouseholdID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}
	v.Occasion = strings.TrimSpace(v.Occasion)
	if err := s.repo.SaveEntry(ctx, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
func validEntryStatus(s string) bool {
	return s == "SUBMITTED" || s == "APPROVED" || s == "REJECTED"
}
func (s *ParentWorkService) VoidEntry(ctx context.Context, id uuid.UUID, reason string,
	userID uuid.UUID) (*domain.ParentWorkEntry, error) {
	v, err := s.repo.GetEntry(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if v.Status == "VOIDED" {
		return nil, fmt.Errorf("%w: Eintrag ist bereits storniert", ErrConflict)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, fmt.Errorf("%w: Stornogrund fehlt", ErrInvalidInput)
	}
	v.Status = "VOIDED"
	v.VoidReason = &reason
	v.UpdatedBy = &userID
	if err := s.repo.SaveEntry(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *ParentWorkService) SaveOverride(ctx context.Context, v domain.ParentWorkOverride,
	userID uuid.UUID) (*domain.ParentWorkOverride, error) {
	if v.KitaYear < 1900 || v.KitaYear > 9998 || v.RequiredMinutes < 0 || strings.TrimSpace(v.Reason) == "" {
		return nil, fmt.Errorf("%w: Kita-Jahr, Soll und Begründung prüfen", ErrInvalidInput)
	}
	exists, err := s.repo.HouseholdExists(ctx, v.HouseholdID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}
	v.Reason = strings.TrimSpace(v.Reason)
	v.CreatedBy = &userID
	if err := s.repo.SaveOverride(ctx, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
func (s *ParentWorkService) DeleteOverride(ctx context.Context, id uuid.UUID, year int) error {
	err := s.repo.DeleteOverride(ctx, id, year)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
func (s *ParentWorkService) Terms(ctx context.Context) ([]domain.BoardTerm, error) {
	return s.repo.ListTerms(ctx)
}
func (s *ParentWorkService) SaveTerm(
	ctx context.Context, id uuid.UUID, v domain.BoardTerm,
) (*domain.BoardTerm, error) {
	if id != uuid.Nil {
		existing, err := s.repo.GetTerm(ctx, id)
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		v.ID = id
		v.CreatedAt = existing.CreatedAt
	}
	if v.MemberID == uuid.Nil || strings.TrimSpace(v.Office) == "" || v.StartDate.IsZero() ||
		(v.EndDate != nil && v.EndDate.Before(v.StartDate)) {
		return nil, fmt.Errorf("%w: Mitglied, Amt und gültiger Zeitraum sind erforderlich", ErrInvalidInput)
	}
	exists, err := s.repo.MemberExists(ctx, v.MemberID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}
	v.Office = strings.TrimSpace(v.Office)
	if err := s.repo.SaveTerm(ctx, &v); err != nil {
		return nil, err
	}
	return s.repo.GetTerm(ctx, v.ID)
}
func (s *ParentWorkService) DeleteTerm(ctx context.Context, id uuid.UUID) error {
	err := s.repo.DeleteTerm(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
