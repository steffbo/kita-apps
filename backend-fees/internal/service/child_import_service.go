package service

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/csvparser"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

// ChildImportService handles CSV import for children
type ChildImportService struct {
	childRepo  repository.ChildRepository
	parentRepo repository.ParentRepository
}

// NewChildImportService creates a new child import service
func NewChildImportService(childRepo repository.ChildRepository, parentRepo repository.ParentRepository) *ChildImportService {
	return &ChildImportService{
		childRepo:  childRepo,
		parentRepo: parentRepo,
	}
}

// ParseResult is returned after initial CSV parsing
type ChildImportParseResult struct {
	Headers           []string   `json:"headers"`
	SampleRows        [][]string `json:"sampleRows"`
	DetectedSeparator string     `json:"detectedSeparator"`
	TotalRows         int        `json:"totalRows"`
}

// ParseCSV parses an uploaded CSV file and returns headers and sample data
func (s *ChildImportService) ParseCSV(reader io.Reader) (*ChildImportParseResult, error) {
	result, err := csvparser.ParseCSV(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	return &ChildImportParseResult{
		Headers:           result.Headers,
		SampleRows:        result.SampleRows,
		DetectedSeparator: result.DetectedSeparator,
		TotalRows:         result.TotalRows,
	}, nil
}

// PreviewRequest contains the mapping and CSV content for preview
type PreviewRequest struct {
	FileContent string         `json:"fileContent"` // Base64 encoded CSV content
	Separator   string         `json:"separator"`
	Mapping     map[string]int `json:"mapping"` // systemField -> csvColumnIndex
	SkipHeader  bool           `json:"skipHeader"`
}

// PreviewRow represents a single row in the preview
type PreviewRow struct {
	Index           int            `json:"index"`
	Child           ChildPreview   `json:"child"`
	Parent1         *ParentPreview `json:"parent1,omitempty"`
	Parent2         *ParentPreview `json:"parent2,omitempty"`
	Warnings        []string       `json:"warnings"`
	IsDuplicate     bool           `json:"isDuplicate"`
	ExistingChildID *string        `json:"existingChildId,omitempty"`
	IsValid         bool           `json:"isValid"`
}

// ChildPreview contains child data for preview
type ChildPreview struct {
	MemberNumber string `json:"memberNumber"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	BirthDate    string `json:"birthDate"`
	EntryDate    string `json:"entryDate"`
	Street       string `json:"street,omitempty"`
	StreetNo     string `json:"streetNo,omitempty"`
	PostalCode   string `json:"postalCode,omitempty"`
	City         string `json:"city,omitempty"`
	LegalHours   *int   `json:"legalHours,omitempty"`
	CareHours    *int   `json:"careHours,omitempty"`
}

// ParentPreview contains parent data for preview
type ParentPreview struct {
	FirstName       string        `json:"firstName"`
	LastName        string        `json:"lastName"`
	Email           string        `json:"email,omitempty"`
	Phone           string        `json:"phone,omitempty"`
	ExistingMatches []ParentMatch `json:"existingMatches,omitempty"`
}

// ParentMatch represents a potential existing parent match
type ParentMatch struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email,omitempty"`
}

// PreviewResult contains all preview rows
type PreviewResult struct {
	Rows       []PreviewRow `json:"rows"`
	ValidCount int          `json:"validCount"`
	ErrorCount int          `json:"errorCount"`
}

// Preview generates a preview of the import data with validation
func (s *ChildImportService) Preview(ctx context.Context, rows [][]string, mapping map[string]int) (*PreviewResult, error) {
	var previewRows []PreviewRow
	validCount := 0
	errorCount := 0

	for idx, row := range rows {
		previewRow := s.processRow(ctx, idx, row, mapping)
		previewRows = append(previewRows, previewRow)

		if previewRow.IsValid && !previewRow.IsDuplicate {
			validCount++
		} else {
			errorCount++
		}
	}

	return &PreviewResult{
		Rows:       previewRows,
		ValidCount: validCount,
		ErrorCount: errorCount,
	}, nil
}

func (s *ChildImportService) processRow(ctx context.Context, index int, row []string, mapping map[string]int) PreviewRow {
	preview := PreviewRow{
		Index:    index,
		Warnings: []string{},
		IsValid:  true,
	}

	// Extract child data
	child := ChildPreview{}

	if idx, ok := mapping["memberNumber"]; ok && idx < len(row) {
		child.MemberNumber = strings.TrimSpace(row[idx])
	}
	if idx, ok := mapping["firstName"]; ok && idx < len(row) {
		child.FirstName = strings.TrimSpace(row[idx])
	}
	if idx, ok := mapping["lastName"]; ok && idx < len(row) {
		child.LastName = strings.TrimSpace(row[idx])
	}
	if idx, ok := mapping["birthDate"]; ok && idx < len(row) {
		dateStr := strings.TrimSpace(row[idx])
		if t, err := csvparser.ParseDate(dateStr); err == nil && !t.IsZero() {
			child.BirthDate = t.Format("2006-01-02")
		} else if dateStr != "" {
			preview.Warnings = append(preview.Warnings, fmt.Sprintf("Ungültiges Geburtsdatum: %s", dateStr))
		}
	}
	if idx, ok := mapping["entryDate"]; ok && idx < len(row) {
		dateStr := strings.TrimSpace(row[idx])
		if t, err := csvparser.ParseDate(dateStr); err == nil && !t.IsZero() {
			child.EntryDate = t.Format("2006-01-02")
		} else if dateStr != "" {
			preview.Warnings = append(preview.Warnings, fmt.Sprintf("Ungültiges Eintrittsdatum: %s", dateStr))
		}
	}
	if idx, ok := mapping["street"]; ok && idx < len(row) {
		child.Street = strings.TrimSpace(row[idx])
	}
	if idx, ok := mapping["streetNo"]; ok && idx < len(row) {
		child.StreetNo = strings.TrimSpace(row[idx])
	}
	if idx, ok := mapping["postalCode"]; ok && idx < len(row) {
		child.PostalCode = strings.TrimSpace(row[idx])
	}
	if idx, ok := mapping["city"]; ok && idx < len(row) {
		child.City = strings.TrimSpace(row[idx])
	}
	if idx, ok := mapping["legalHours"]; ok && idx < len(row) {
		if hours, err := csvparser.ParseInt(row[idx]); err == nil && hours > 0 {
			child.LegalHours = &hours
		}
	}
	if idx, ok := mapping["careHours"]; ok && idx < len(row) {
		if hours, err := csvparser.ParseInt(row[idx]); err == nil && hours > 0 {
			child.CareHours = &hours
		}
	}

	preview.Child = child

	// Validate required fields
	if child.MemberNumber == "" {
		preview.Warnings = append(preview.Warnings, "Mitgliedsnummer fehlt")
		preview.IsValid = false
	}
	if child.FirstName == "" {
		preview.Warnings = append(preview.Warnings, "Vorname fehlt")
		preview.IsValid = false
	}
	if child.LastName == "" {
		preview.Warnings = append(preview.Warnings, "Nachname fehlt")
		preview.IsValid = false
	}
	if child.BirthDate == "" {
		preview.Warnings = append(preview.Warnings, "Geburtsdatum fehlt")
		preview.IsValid = false
	}
	if child.EntryDate == "" {
		preview.Warnings = append(preview.Warnings, "Eintrittsdatum fehlt")
		preview.IsValid = false
	}

	// Check for duplicate member number
	if child.MemberNumber != "" {
		existing, err := s.childRepo.GetByMemberNumber(ctx, child.MemberNumber)
		if err == nil && existing != nil {
			preview.IsDuplicate = true
			id := existing.ID.String()
			preview.ExistingChildID = &id
			preview.Warnings = append(preview.Warnings, fmt.Sprintf("Kind mit Mitgliedsnummer %s existiert bereits", child.MemberNumber))
		}
	}

	// Extract parent 1 data
	parent1 := s.extractParent(row, mapping, "parent1")
	if parent1 != nil {
		// Search for existing parent matches
		matches := s.findParentMatches(ctx, parent1.FirstName, parent1.LastName)
		parent1.ExistingMatches = matches
		preview.Parent1 = parent1
	}

	// Extract parent 2 data
	parent2 := s.extractParent(row, mapping, "parent2")
	if parent2 != nil {
		matches := s.findParentMatches(ctx, parent2.FirstName, parent2.LastName)
		parent2.ExistingMatches = matches
		preview.Parent2 = parent2
	}

	return preview
}

func (s *ChildImportService) extractParent(row []string, mapping map[string]int, prefix string) *ParentPreview {
	parent := &ParentPreview{}
	hasData := false

	if idx, ok := mapping[prefix+"FirstName"]; ok && idx < len(row) {
		parent.FirstName = strings.TrimSpace(row[idx])
		if parent.FirstName != "" {
			hasData = true
		}
	}
	if idx, ok := mapping[prefix+"LastName"]; ok && idx < len(row) {
		parent.LastName = strings.TrimSpace(row[idx])
		if parent.LastName != "" {
			hasData = true
		}
	}
	if idx, ok := mapping[prefix+"Email"]; ok && idx < len(row) {
		parent.Email = strings.TrimSpace(row[idx])
	}
	if idx, ok := mapping[prefix+"Phone"]; ok && idx < len(row) {
		parent.Phone = strings.TrimSpace(row[idx])
	}

	if !hasData {
		return nil
	}

	return parent
}

func (s *ChildImportService) findParentMatches(ctx context.Context, firstName, lastName string) []ParentMatch {
	if firstName == "" && lastName == "" {
		return nil
	}

	// Search by name
	searchTerm := strings.TrimSpace(firstName + " " + lastName)
	parents, _, err := s.parentRepo.List(ctx, searchTerm, 0, 5)
	if err != nil {
		return nil
	}

	var matches []ParentMatch
	for _, p := range parents {
		// Check if names match (case-insensitive)
		if strings.EqualFold(p.FirstName, firstName) && strings.EqualFold(p.LastName, lastName) {
			matches = append(matches, ParentMatch{
				ID:        p.ID.String(),
				FirstName: p.FirstName,
				LastName:  p.LastName,
				Email:     stringOrEmpty(p.Email),
			})
		}
	}

	return matches
}

// ExecuteRequest contains the data to import
type ExecuteRequest struct {
	Rows            []ImportRow      `json:"rows"`
	ParentDecisions []ParentDecision `json:"parentDecisions"`
}

// ImportRow is a row to be imported
type ImportRow struct {
	Index   int            `json:"index"`
	Child   ChildPreview   `json:"child"`
	Parent1 *ParentPreview `json:"parent1,omitempty"`
	Parent2 *ParentPreview `json:"parent2,omitempty"`
}

// ParentDecision indicates how to handle a parent
type ParentDecision struct {
	RowIndex         int    `json:"rowIndex"`
	ParentIndex      int    `json:"parentIndex"` // 1 or 2
	Action           string `json:"action"`      // "create" or "link"
	ExistingParentID string `json:"existingParentId,omitempty"`
}

// ExecuteResult contains the import results
type ExecuteResult struct {
	ChildrenCreated int           `json:"childrenCreated"`
	ParentsCreated  int           `json:"parentsCreated"`
	ParentsLinked   int           `json:"parentsLinked"`
	Errors          []ImportError `json:"errors"`
}

// ImportError describes an error during import
type ImportError struct {
	RowIndex int    `json:"rowIndex"`
	Error    string `json:"error"`
}

// Execute performs the actual import
func (s *ChildImportService) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResult, error) {
	result := &ExecuteResult{
		Errors: []ImportError{},
	}

	// Build parent decision map for quick lookup
	parentDecisionMap := make(map[string]ParentDecision)
	for _, pd := range req.ParentDecisions {
		key := fmt.Sprintf("%d-%d", pd.RowIndex, pd.ParentIndex)
		parentDecisionMap[key] = pd
	}

	for _, row := range req.Rows {
		// Parse dates
		birthDate, err := time.Parse("2006-01-02", row.Child.BirthDate)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				RowIndex: row.Index,
				Error:    "Ungültiges Geburtsdatum",
			})
			continue
		}

		entryDate, err := time.Parse("2006-01-02", row.Child.EntryDate)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				RowIndex: row.Index,
				Error:    "Ungültiges Eintrittsdatum",
			})
			continue
		}

		// Create child
		child := &domain.Child{
			ID:           uuid.New(),
			MemberNumber: row.Child.MemberNumber,
			FirstName:    row.Child.FirstName,
			LastName:     row.Child.LastName,
			BirthDate:    birthDate,
			EntryDate:    entryDate,
			Street:       stringPtr(row.Child.Street),
			StreetNo:     stringPtr(row.Child.StreetNo),
			PostalCode:   stringPtr(row.Child.PostalCode),
			City:         stringPtr(row.Child.City),
			IsActive:     true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if row.Child.LegalHours != nil {
			child.LegalHours = row.Child.LegalHours
		}
		if row.Child.CareHours != nil {
			child.CareHours = row.Child.CareHours
		}

		err = s.childRepo.Create(ctx, child)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				RowIndex: row.Index,
				Error:    fmt.Sprintf("Fehler beim Erstellen: %v", err),
			})
			continue
		}

		result.ChildrenCreated++

		// Handle parent 1
		if row.Parent1 != nil && row.Parent1.FirstName != "" && row.Parent1.LastName != "" {
			parentID, created, err := s.handleParent(ctx, row.Index, 1, row.Parent1, parentDecisionMap)
			if err != nil {
				result.Errors = append(result.Errors, ImportError{
					RowIndex: row.Index,
					Error:    fmt.Sprintf("Fehler bei Elternteil 1: %v", err),
				})
			} else if parentID != uuid.Nil {
				// Link parent to child
				isPrimary := true // First parent is primary
				err = s.childRepo.LinkParent(ctx, child.ID, parentID, isPrimary)
				if err != nil {
					result.Errors = append(result.Errors, ImportError{
						RowIndex: row.Index,
						Error:    fmt.Sprintf("Fehler beim Verknüpfen von Elternteil 1: %v", err),
					})
				} else {
					if created {
						result.ParentsCreated++
					} else {
						result.ParentsLinked++
					}
				}
			}
		}

		// Handle parent 2
		if row.Parent2 != nil && row.Parent2.FirstName != "" && row.Parent2.LastName != "" {
			parentID, created, err := s.handleParent(ctx, row.Index, 2, row.Parent2, parentDecisionMap)
			if err != nil {
				result.Errors = append(result.Errors, ImportError{
					RowIndex: row.Index,
					Error:    fmt.Sprintf("Fehler bei Elternteil 2: %v", err),
				})
			} else if parentID != uuid.Nil {
				// Link parent to child
				isPrimary := false // Second parent is not primary
				err = s.childRepo.LinkParent(ctx, child.ID, parentID, isPrimary)
				if err != nil {
					result.Errors = append(result.Errors, ImportError{
						RowIndex: row.Index,
						Error:    fmt.Sprintf("Fehler beim Verknüpfen von Elternteil 2: %v", err),
					})
				} else {
					if created {
						result.ParentsCreated++
					} else {
						result.ParentsLinked++
					}
				}
			}
		}
	}

	return result, nil
}

func (s *ChildImportService) handleParent(ctx context.Context, rowIndex, parentIndex int, parent *ParentPreview, decisions map[string]ParentDecision) (uuid.UUID, bool, error) {
	key := fmt.Sprintf("%d-%d", rowIndex, parentIndex)
	decision, hasDecision := decisions[key]

	// If user decided to link to existing
	if hasDecision && decision.Action == "link" && decision.ExistingParentID != "" {
		parentID, err := uuid.Parse(decision.ExistingParentID)
		if err != nil {
			return uuid.Nil, false, fmt.Errorf("ungültige Eltern-ID: %v", err)
		}
		return parentID, false, nil
	}

	// Create new parent
	newParent := &domain.Parent{
		ID:        uuid.New(),
		FirstName: parent.FirstName,
		LastName:  parent.LastName,
		Email:     stringPtr(parent.Email),
		Phone:     stringPtr(parent.Phone),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.parentRepo.Create(ctx, newParent)
	if err != nil {
		return uuid.Nil, false, err
	}

	return newParent.ID, true, nil
}

// Note: stringPtr and stringOrEmpty are defined in import_service.go
