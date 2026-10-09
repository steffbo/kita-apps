package service

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/csvparser"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"io"
	"math"
	"strconv"
	"strings"
	"time"
)

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
