package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/csvparser"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

// ParentWorkService manages work accounts, entries, rules, and imports.
type ParentWorkService struct {
	repo repository.ParentWorkRepository
	txm  *repository.TxManager
}

// NewParentWorkService creates a service backed by a parent work repository.
func NewParentWorkService(
	repo repository.ParentWorkRepository, txm ...*repository.TxManager,
) *ParentWorkService {
	s := &ParentWorkService{repo: repo}
	if len(txm) > 0 {
		s.txm = txm[0]
	}
	return s
}

// ParentWorkImportParseResult contains the parsed CSV header and rows.
type ParentWorkImportParseResult struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

// ParentWorkImportMapping maps field names to CSV column indexes.
type ParentWorkImportMapping map[string]int

// ParentWorkImportPreviewRequest describes a CSV mapping to preview.
type ParentWorkImportPreviewRequest struct {
	Headers []string                `json:"headers"`
	Rows    [][]string              `json:"rows"`
	Mapping ParentWorkImportMapping `json:"mapping"`
}

// ParentWorkImportRow is a parsed CSV row with matching and validation results.
type ParentWorkImportRow struct {
	Index           int        `json:"index"`
	WorkDate        string     `json:"workDate,omitempty" binding:"optional"`
	DurationMinutes int        `json:"durationMinutes,omitempty" binding:"optional"`
	Occasion        string     `json:"occasion,omitempty" binding:"optional"`
	MemberName      string     `json:"memberName,omitempty" binding:"optional"`
	ChildName       string     `json:"childName,omitempty" binding:"optional"`
	HouseholdID     *uuid.UUID `json:"householdId,omitempty" binding:"optional"`
	HouseholdName   *string    `json:"householdName,omitempty" binding:"optional"`
	MatchedBy       *string    `json:"matchedBy" binding:"optional"`
	Errors          []string   `json:"errors"`
	Duplicate       bool       `json:"duplicate"`
}

// ParentWorkImportExecuteRow is a confirmed row ready to import.
type ParentWorkImportExecuteRow struct {
	HouseholdID     uuid.UUID `json:"householdId"`
	WorkDate        string    `json:"workDate"`
	DurationMinutes int       `json:"durationMinutes"`
	Occasion        string    `json:"occasion"`
	MemberName      *string   `json:"memberName,omitempty" binding:"optional"`
	ChildName       *string   `json:"childName,omitempty" binding:"optional"`
}

// ParentWorkImportExecuteRequest contains confirmed rows for import.
type ParentWorkImportExecuteRequest struct {
	Rows []ParentWorkImportExecuteRow `json:"rows"`
}

// ParentWorkImportExecuteResult reports the number of imported entries.
type ParentWorkImportExecuteResult struct {
	Created int `json:"created"`
}

// ParseParentWorkHours converts decimal hours to positive quarter-hour minutes.
func ParseParentWorkHours(raw string) (int, error) {
	v := strings.TrimSpace(strings.ToLower(raw))
	v = strings.TrimSuffix(v, "std")
	v = strings.TrimSuffix(v, "h")
	v = strings.TrimSpace(strings.ReplaceAll(v, ",", "."))
	hours, err := strconv.ParseFloat(v, 64)
	if err != nil || math.IsNaN(hours) || math.IsInf(hours, 0) || hours <= 0 {
		return 0, fmt.Errorf("ungültige Stundenzahl")
	}
	minutes := hours * 60
	if math.Abs(minutes-math.Round(minutes)) > 0.000001 || int(math.Round(minutes))%15 != 0 {
		return 0, fmt.Errorf("Stunden müssen positive Viertelstunden sein")
	}
	return int(math.Round(minutes)), nil
}

// ParseImport parses a parent work CSV file.
func (s *ParentWorkService) ParseImport(reader io.Reader) (*ParentWorkImportParseResult, error) {
	v, err := csvparser.ParseCSV(reader)
	if err != nil {
		return nil, err
	}
	return &ParentWorkImportParseResult{Headers: v.Headers, Rows: v.AllRows}, nil
}

// PreviewImport validates CSV rows and suggests households.
func (s *ParentWorkService) PreviewImport(
	ctx context.Context, req ParentWorkImportPreviewRequest,
) ([]ParentWorkImportRow, error) {
	for _, field := range []string{"workDate", "hours", "occasion"} {
		if _, ok := req.Mapping[field]; !ok {
			return nil, fmt.Errorf("%w: Spalte für %s fehlt", ErrInvalidInput, field)
		}
	}
	if _, child := req.Mapping["childName"]; !child {
		if _, member := req.Mapping["memberName"]; !member {
			return nil, fmt.Errorf("%w: Mitglied oder Kind muss zugeordnet sein", ErrInvalidInput)
		}
	}
	snap, err := s.repo.Snapshot(ctx, domain.ParentWorkYearStart(1900), domain.ParentWorkYearStart(9999))
	if err != nil {
		return nil, err
	}
	houses := map[uuid.UUID]string{}
	for _, h := range snap.Households {
		houses[h.ID] = h.Name
	}
	seen := map[string]bool{}
	for _, e := range snap.Entries {
		if e.Status != domain.ParentWorkStatusVoided {
			key := importDuplicateKey(e.HouseholdID, e.WorkDate.Format("2006-01-02"),
				e.DurationMinutes, e.Occasion)
			seen[key] = true
		}
	}
	result := make([]ParentWorkImportRow, 0, len(req.Rows))
	for i, row := range req.Rows {
		v := ParentWorkImportRow{Index: i + 1, Errors: []string{}}
		get := func(k string) string {
			n, ok := req.Mapping[k]
			if !ok || n < 0 || n >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[n])
		}
		v.MemberName, v.ChildName, v.Occasion = get("memberName"), get("childName"), get("occasion")
		d, de := csvparser.ParseDate(get("workDate"))
		if de != nil {
			v.Errors = append(v.Errors, "Ungültiges Datum")
		} else {
			v.WorkDate = d.Format("2006-01-02")
		}
		v.DurationMinutes, de = ParseParentWorkHours(get("hours"))
		if de != nil {
			v.Errors = append(v.Errors, "Ungültige Stunden")
		}
		if v.Occasion == "" {
			v.Errors = append(v.Errors, "Anlass fehlt")
		}
		matches := map[uuid.UUID]string{}
		if v.ChildName != "" {
			target := normalizeParentWorkName(v.ChildName)
			for _, c := range snap.Children {
				if parentWorkNameMatches(c.Name, target) || parentWorkNameMatches(c.LastName+", "+c.FirstName, target) {
					matches[c.HouseholdID] = "child"
				}
			}
		}
		if len(matches) == 0 && v.MemberName != "" {
			target := normalizeParentWorkName(v.MemberName)
			for _, m := range snap.Members {
				if parentWorkNameMatches(m.Name, target) {
					matches[m.HouseholdID] = "member"
				}
			}
			for _, p := range snap.Parents {
				if parentWorkNameMatches(p.Name, target) {
					matches[p.HouseholdID] = "member"
				}
			}
		}
		if len(matches) == 1 {
			for id, by := range matches {
				name := houses[id]
				v.HouseholdID = &id
				v.HouseholdName = &name
				v.MatchedBy = &by
			}
		} else if len(matches) > 1 {
			v.Errors = append(v.Errors, "Zuordnung ist mehrdeutig; Familie auswählen")
		} else if v.ChildName != "" || v.MemberName != "" {
			v.Errors = append(v.Errors, "Familie nicht gefunden; Familie auswählen")
		}
		if v.HouseholdID != nil && v.WorkDate != "" && v.DurationMinutes > 0 && v.Occasion != "" {
			key := importDuplicateKey(*v.HouseholdID, v.WorkDate, v.DurationMinutes, v.Occasion)
			v.Duplicate = seen[key]
			seen[key] = true
		}
		result = append(result, v)
	}
	return result, nil
}

func importDuplicateKey(h uuid.UUID, date string, minutes int, occasion string) string {
	return h.String() + "|" + date + "|" + strconv.Itoa(minutes) + "|" +
		strings.ToLower(strings.TrimSpace(occasion))
}

func normalizeParentWorkName(s string) string {
	return csvparser.NormalizeMatchText(s)
}

func parentWorkNameMatches(name, normalizedTarget string) bool {
	name = normalizeParentWorkName(name)
	if name == normalizedTarget {
		return true
	}
	parts := strings.Fields(name)
	return len(parts) == 2 && parts[1]+" "+parts[0] == normalizedTarget
}

// ExecuteImport saves confirmed CSV rows in one transaction.
func (s *ParentWorkService) ExecuteImport(
	ctx context.Context, req ParentWorkImportExecuteRequest, userID uuid.UUID,
) (*ParentWorkImportExecuteResult, error) {
	if len(req.Rows) == 0 {
		return nil, fmt.Errorf("%w: keine Einträge ausgewählt", ErrInvalidInput)
	}
	created := 0
	err := s.txm.WithTx(ctx, func(txctx context.Context) error {
		for i, row := range req.Rows {
			d, e := time.Parse("2006-01-02", row.WorkDate)
			if e != nil {
				return fmt.Errorf("%w: Zeile %d: ungültiges Datum", ErrInvalidInput, i+1)
			}
			v := domain.ParentWorkEntry{
				HouseholdID: row.HouseholdID, WorkDate: d, DurationMinutes: row.DurationMinutes,
				Occasion: strings.TrimSpace(row.Occasion), MemberName: row.MemberName,
				ChildName: row.ChildName, Status: domain.ParentWorkStatusApproved, Source: "IMPORT",
				CreatedBy: &userID, UpdatedBy: &userID,
			}
			if v.HouseholdID == uuid.Nil || v.DurationMinutes <= 0 || v.DurationMinutes%15 != 0 || v.Occasion == "" {
				return fmt.Errorf("%w: Zeile %d: Familie, Datum, Anlass und positive "+
					"Viertelstunden sind erforderlich", ErrInvalidInput, i+1)
			}
			exists, e := s.repo.HouseholdExists(txctx, v.HouseholdID)
			if e != nil {
				return fmt.Errorf("%w: Zeile %d: Familie konnte nicht geprüft werden", ErrInvalidInput, i+1)
			}
			if !exists {
				return fmt.Errorf("%w: Zeile %d: Familie nicht gefunden", ErrInvalidInput, i+1)
			}
			if e = s.repo.SaveEntry(txctx, &v); e != nil {
				return fmt.Errorf("%w: Zeile %d: Eintrag konnte nicht gespeichert werden", ErrInvalidInput, i+1)
			}
			created++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &ParentWorkImportExecuteResult{Created: created}, nil
}

// ParentWorkOverview summarizes all household accounts in a Kita year.
type ParentWorkOverview struct {
	KitaYear           int                                `json:"kitaYear"`
	Rule               *domain.ParentWorkRule             `json:"rule,omitempty" binding:"optional"`
	Notice             *string                            `json:"notice,omitempty" binding:"optional"`
	Households         []domain.ParentWorkAccount         `json:"households"`
	UnassignedChildren []domain.ParentWorkUnassignedChild `json:"unassignedChildren"`
	RequiredMinutes    int                                `json:"requiredMinutes"`
	DoneMinutes        int                                `json:"doneMinutes"`
	OpenMinutes        int                                `json:"openMinutes"`
	MissingAmountCents int                                `json:"missingAmountCents"`
	SubmittedTotal     int                                `json:"submittedTotal"`
}

// ParentWorkHouseholdOption supplies names for household selection.
type ParentWorkHouseholdOption struct {
	ID       uuid.UUID                 `json:"id"`
	Name     string                    `json:"name"`
	Children []domain.ParentWorkChild  `json:"children"`
	Members  []domain.ParentWorkMember `json:"members"`
	Parents  []domain.ParentWorkParent `json:"parents"`
}

// ParentWorkDetail contains one account with its entries and board terms.
type ParentWorkDetail struct {
	domain.ParentWorkAccount
	Entries    []domain.ParentWorkEntry `json:"entries"`
	BoardTerms []domain.BoardTerm       `json:"boardTerms"`
}

// Overview calculates household accounts for a Kita year.
func (s *ParentWorkService) Overview(ctx context.Context, year int) (*ParentWorkOverview, error) {
	if year < 1900 || year > 9998 {
		return nil, fmt.Errorf("%w: ungültiges Kita-Jahr", ErrInvalidInput)
	}
	rules, err := s.repo.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	result := &ParentWorkOverview{KitaYear: year, Households: []domain.ParentWorkAccount{},
		UnassignedChildren: []domain.ParentWorkUnassignedChild{}}
	if len(rules) == 0 || rules[0].ValidFrom.After(domain.ParentWorkYearStart(year)) {
		notice := "Für dieses Kita-Jahr gibt es kein Regelwerk; es ist nicht abrechenbar."
		result.Notice = &notice
		from := domain.ParentWorkYearStart(year)
		snap, err := s.repo.Snapshot(ctx, from, domain.ParentWorkYearStart(year+1).AddDate(0, 0, -1))
		if err != nil {
			return nil, err
		}
		result.UnassignedChildren = unassignedParentWorkChildren(snap.UnassignedChildren, year)
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
				result.SubmittedTotal += row.SubmittedCount
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
	result.UnassignedChildren = unassignedParentWorkChildren(snap.UnassignedChildren, year)
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
			result.SubmittedTotal += row.SubmittedCount
			result.OpenMinutes += row.OpenMinutes
			result.MissingAmountCents += row.MissingAmountCents
		}
	}
	return result, nil
}

func unassignedParentWorkChildren(children []domain.ParentWorkUnassignedChild,
	year int) []domain.ParentWorkUnassignedChild {
	result := []domain.ParentWorkUnassignedChild{}
	from := domain.ParentWorkYearStart(year)
	until := domain.ParentWorkYearStart(year+1).AddDate(0, 0, -1)
	for _, child := range children {
		if !child.EntryDate.After(until) && (child.ExitDate == nil || !child.ExitDate.Before(from)) {
			result = append(result, child)
		}
	}
	return result
}

// Households lists household choices with children and members.
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

// Detail returns one household account and its entries.
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

// ReviewEntry accepts or rejects a submitted entry.
func (s *ParentWorkService) ReviewEntry(ctx context.Context, id uuid.UUID, approve bool,
	reason string, userID uuid.UUID) (*domain.ParentWorkEntry, error) {
	v, err := s.repo.GetEntry(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if v.Status != domain.ParentWorkStatusSubmitted {
		return nil, fmt.Errorf("%w: Nur gemeldete Einträge können geprüft werden", ErrConflict)
	}
	if approve {
		v.Status, v.RejectReason = domain.ParentWorkStatusApproved, nil
	} else {
		reason = strings.TrimSpace(reason)
		if reason == "" {
			return nil, fmt.Errorf("%w: Ablehnungsgrund fehlt", ErrInvalidInput)
		}
		v.Status, v.RejectReason = domain.ParentWorkStatusRejected, &reason
	}
	v.UpdatedBy = &userID
	if err := s.repo.SaveEntry(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

// SubmitParentEntry stores a new parent report without crediting it yet.
func (s *ParentWorkService) SubmitParentEntry(ctx context.Context, v domain.ParentWorkEntry,
	userID uuid.UUID) (*domain.ParentWorkEntry, error) {
	if v.WorkDate.After(util.Today()) {
		return nil, fmt.Errorf("%w: Arbeitsdatum darf nicht in der Zukunft liegen", ErrInvalidInput)
	}
	v.Status, v.Source = domain.ParentWorkStatusSubmitted, "PARENT"
	v.CreatedBy, v.UpdatedBy = &userID, &userID
	if v.HouseholdID == uuid.Nil || v.WorkDate.IsZero() || strings.TrimSpace(v.Occasion) == "" ||
		v.DurationMinutes <= 0 || v.DurationMinutes%15 != 0 {
		return nil, fmt.Errorf("%w: Datum, Anlass und positive Viertelstunden sind erforderlich",
			ErrInvalidInput)
	}
	v.Occasion = strings.TrimSpace(v.Occasion)
	if err := s.repo.SaveEntry(ctx, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// WithdrawParentEntry only voids a submitted entry belonging to this household.
func (s *ParentWorkService) WithdrawParentEntry(ctx context.Context, id, householdID,
	userID uuid.UUID) (*domain.ParentWorkEntry, error) {
	v, err := s.repo.GetEntry(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if v.HouseholdID != householdID || v.Source != "PARENT" {
		return nil, ErrNotFound
	}
	if v.Status != domain.ParentWorkStatusSubmitted {
		return nil, fmt.Errorf("%w: Nur gemeldete Einträge können zurückgezogen werden", ErrConflict)
	}
	reason := "Von Eltern zurückgezogen"
	v.Status, v.VoidReason, v.UpdatedBy = domain.ParentWorkStatusVoided, &reason, &userID
	if err := s.repo.SaveEntry(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

// SaveRule creates or updates a rule version.
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

// Rules lists rule versions.
func (s *ParentWorkService) Rules(ctx context.Context) ([]domain.ParentWorkRule, error) {
	return s.repo.ListRules(ctx)
}

// SaveEntry creates or updates a work entry.
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
		if existing.Status == domain.ParentWorkStatusVoided {
			return nil, fmt.Errorf("%w: stornierte Einträge können nicht bearbeitet werden", ErrConflict)
		}
		v.ID = id
		v.CreatedAt = existing.CreatedAt
		v.CreatedBy = existing.CreatedBy
		v.Source = existing.Source
		if v.Status == "" {
			v.Status = existing.Status
		}
		if existing.Source == "PARENT" && existing.Status == domain.ParentWorkStatusSubmitted &&
			v.Status != domain.ParentWorkStatusSubmitted {
			return nil, fmt.Errorf("%w: Gemeldete Einträge müssen bestätigt oder abgelehnt werden",
				ErrConflict)
		}
	} else {
		v.Source = "MANUAL"
		v.CreatedBy = &userID
		if v.Status == "" {
			v.Status = domain.ParentWorkStatusApproved
		}
	}
	v.UpdatedBy = &userID
	if v.HouseholdID == uuid.Nil {
		return nil, fmt.Errorf("%w: Familie fehlt", ErrInvalidInput)
	}
	if v.WorkDate.IsZero() {
		return nil, fmt.Errorf("%w: Datum fehlt", ErrInvalidInput)
	}
	if strings.TrimSpace(v.Occasion) == "" {
		return nil, fmt.Errorf("%w: Anlass fehlt", ErrInvalidInput)
	}
	if v.DurationMinutes <= 0 || v.DurationMinutes%15 != 0 {
		return nil, fmt.Errorf("%w: Stunden müssen positiv und in Viertelstunden sein", ErrInvalidInput)
	}
	if !validEntryStatus(v.Status) {
		return nil, fmt.Errorf("%w: Ungültiger Status", ErrInvalidInput)
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
	return s == domain.ParentWorkStatusSubmitted || s == domain.ParentWorkStatusApproved ||
		s == domain.ParentWorkStatusRejected
}

// VoidEntry cancels a work entry with a reason.
func (s *ParentWorkService) VoidEntry(ctx context.Context, id uuid.UUID, reason string,
	userID uuid.UUID) (*domain.ParentWorkEntry, error) {
	v, err := s.repo.GetEntry(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if v.Status == domain.ParentWorkStatusVoided {
		return nil, fmt.Errorf("%w: Eintrag ist bereits storniert", ErrConflict)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, fmt.Errorf("%w: Stornogrund fehlt", ErrInvalidInput)
	}
	v.Status = domain.ParentWorkStatusVoided
	v.VoidReason = &reason
	v.UpdatedBy = &userID
	if err := s.repo.SaveEntry(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

// SaveOverride sets a household annual requirement.
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

// DeleteOverride removes a household annual requirement.
func (s *ParentWorkService) DeleteOverride(ctx context.Context, id uuid.UUID, year int) error {
	err := s.repo.DeleteOverride(ctx, id, year)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

// Terms lists board appointments.
func (s *ParentWorkService) Terms(ctx context.Context) ([]domain.BoardTerm, error) {
	return s.repo.ListTerms(ctx)
}

// SaveTerm creates or updates a board appointment.
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

// DeleteTerm removes a board appointment.
func (s *ParentWorkService) DeleteTerm(ctx context.Context, id uuid.UUID) error {
	err := s.repo.DeleteTerm(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
