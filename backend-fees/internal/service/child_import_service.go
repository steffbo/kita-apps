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
	Index           int             `json:"index"`
	Child           ChildPreview    `json:"child"`
	Parent1         *ParentPreview  `json:"parent1,omitempty"`
	Parent2         *ParentPreview  `json:"parent2,omitempty"`
	Warnings        []string        `json:"warnings"`
	IsDuplicate     bool            `json:"isDuplicate"`
	ExistingChildID *string         `json:"existingChildId,omitempty"`
	ExistingChild   *ChildPreview   `json:"existingChild,omitempty"`
	Action          string          `json:"action"` // "create", "update", "no_change"
	FieldConflicts  []FieldConflict `json:"fieldConflicts,omitempty"`
	IsValid         bool            `json:"isValid"`
}

// FieldConflict represents a conflict between CSV data and existing database data
type FieldConflict struct {
	Field         string `json:"field"`
	FieldLabel    string `json:"fieldLabel"`
	ExistingValue string `json:"existingValue"`
	NewValue      string `json:"newValue"`
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
	AlreadyLinked   bool          `json:"alreadyLinked,omitempty"`  // True if already linked to the existing child
	LinkedParentID  *string       `json:"linkedParentId,omitempty"` // ID of the already linked parent
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
		Action:   "create", // Default action
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

	// Check for existing child by member number first
	var existingChild *domain.Child
	var existingParents []domain.Parent
	if child.MemberNumber != "" {
		existing, err := s.childRepo.GetByMemberNumber(ctx, child.MemberNumber)
		if err == nil && existing != nil {
			existingChild = existing
			preview.IsDuplicate = true
			id := existing.ID.String()
			preview.ExistingChildID = &id

			// Get existing child data for display
			preview.ExistingChild = &ChildPreview{
				MemberNumber: existing.MemberNumber,
				FirstName:    existing.FirstName,
				LastName:     existing.LastName,
				BirthDate:    existing.BirthDate.Format("2006-01-02"),
				EntryDate:    existing.EntryDate.Format("2006-01-02"),
				Street:       stringOrEmpty(existing.Street),
				StreetNo:     stringOrEmpty(existing.StreetNo),
				PostalCode:   stringOrEmpty(existing.PostalCode),
				City:         stringOrEmpty(existing.City),
			}
			if existing.LegalHours != nil {
				preview.ExistingChild.LegalHours = existing.LegalHours
			}
			if existing.CareHours != nil {
				preview.ExistingChild.CareHours = existing.CareHours
			}

			// Get existing parents for this child
			existingParents, _ = s.childRepo.GetParents(ctx, existing.ID)

			// Detect field conflicts
			preview.FieldConflicts = s.detectFieldConflicts(child, existing)

			// Determine action: "update" if there are conflicts, "no_change" otherwise
			if len(preview.FieldConflicts) > 0 {
				preview.Action = "update"
			} else {
				preview.Action = "no_change"
			}

			// For existing children, validation is relaxed - we only need member number
			// Other fields are optional (we'll use existing values if not provided)
			preview.Warnings = append(preview.Warnings, fmt.Sprintf("Kind mit Mitgliedsnummer %s existiert bereits", child.MemberNumber))
		}
	}

	// Validate required fields - only strictly required for NEW children
	if existingChild == nil {
		// For new children, all required fields must be present
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
	} else {
		// For existing children, member number must be present
		if child.MemberNumber == "" {
			preview.Warnings = append(preview.Warnings, "Mitgliedsnummer fehlt")
			preview.IsValid = false
		}
	}

	// Extract parent 1 data
	parent1 := s.extractParent(row, mapping, "parent1")
	if parent1 != nil {
		// Check if this parent is already linked to the existing child
		if existingChild != nil {
			s.checkParentAlreadyLinked(parent1, existingParents)
		}
		// Search for existing parent matches (only if not already linked)
		if !parent1.AlreadyLinked {
			matches := s.findParentMatches(ctx, parent1.FirstName, parent1.LastName)
			parent1.ExistingMatches = matches
		}
		preview.Parent1 = parent1
	}

	// Extract parent 2 data
	parent2 := s.extractParent(row, mapping, "parent2")
	if parent2 != nil {
		// Check if this parent is already linked to the existing child
		if existingChild != nil {
			s.checkParentAlreadyLinked(parent2, existingParents)
		}
		// Search for existing parent matches (only if not already linked)
		if !parent2.AlreadyLinked {
			matches := s.findParentMatches(ctx, parent2.FirstName, parent2.LastName)
			parent2.ExistingMatches = matches
		}
		preview.Parent2 = parent2
	}

	return preview
}

// detectFieldConflicts compares CSV data with existing child data
func (s *ChildImportService) detectFieldConflicts(csvChild ChildPreview, existing *domain.Child) []FieldConflict {
	var conflicts []FieldConflict

	// Compare first name
	if csvChild.FirstName != "" && csvChild.FirstName != existing.FirstName {
		conflicts = append(conflicts, FieldConflict{
			Field:         "firstName",
			FieldLabel:    "Vorname",
			ExistingValue: existing.FirstName,
			NewValue:      csvChild.FirstName,
		})
	}

	// Compare last name
	if csvChild.LastName != "" && csvChild.LastName != existing.LastName {
		conflicts = append(conflicts, FieldConflict{
			Field:         "lastName",
			FieldLabel:    "Nachname",
			ExistingValue: existing.LastName,
			NewValue:      csvChild.LastName,
		})
	}

	// Compare birth date
	existingBirthDate := existing.BirthDate.Format("2006-01-02")
	if csvChild.BirthDate != "" && csvChild.BirthDate != existingBirthDate {
		conflicts = append(conflicts, FieldConflict{
			Field:         "birthDate",
			FieldLabel:    "Geburtsdatum",
			ExistingValue: existingBirthDate,
			NewValue:      csvChild.BirthDate,
		})
	}

	// Compare entry date
	existingEntryDate := existing.EntryDate.Format("2006-01-02")
	if csvChild.EntryDate != "" && csvChild.EntryDate != existingEntryDate {
		conflicts = append(conflicts, FieldConflict{
			Field:         "entryDate",
			FieldLabel:    "Eintrittsdatum",
			ExistingValue: existingEntryDate,
			NewValue:      csvChild.EntryDate,
		})
	}

	// Compare legal hours
	if csvChild.LegalHours != nil {
		existingLegal := 0
		if existing.LegalHours != nil {
			existingLegal = *existing.LegalHours
		}
		if *csvChild.LegalHours != existingLegal {
			conflicts = append(conflicts, FieldConflict{
				Field:         "legalHours",
				FieldLabel:    "Rechtsanspruch",
				ExistingValue: fmt.Sprintf("%d", existingLegal),
				NewValue:      fmt.Sprintf("%d", *csvChild.LegalHours),
			})
		}
	}

	// Compare care hours
	if csvChild.CareHours != nil {
		existingCare := 0
		if existing.CareHours != nil {
			existingCare = *existing.CareHours
		}
		if *csvChild.CareHours != existingCare {
			conflicts = append(conflicts, FieldConflict{
				Field:         "careHours",
				FieldLabel:    "Betreuungszeit",
				ExistingValue: fmt.Sprintf("%d", existingCare),
				NewValue:      fmt.Sprintf("%d", *csvChild.CareHours),
			})
		}
	}

	return conflicts
}

// checkParentAlreadyLinked checks if a parent from CSV is already linked to an existing child
func (s *ChildImportService) checkParentAlreadyLinked(parent *ParentPreview, existingParents []domain.Parent) {
	for _, ep := range existingParents {
		if strings.EqualFold(ep.FirstName, parent.FirstName) && strings.EqualFold(ep.LastName, parent.LastName) {
			parent.AlreadyLinked = true
			id := ep.ID.String()
			parent.LinkedParentID = &id
			return
		}
	}
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
	parents, _, err := s.parentRepo.List(ctx, searchTerm, "name", "asc", 0, 5)
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
	Index           int               `json:"index"`
	Child           ChildPreview      `json:"child"`
	Parent1         *ParentPreview    `json:"parent1,omitempty"`
	Parent2         *ParentPreview    `json:"parent2,omitempty"`
	ExistingChildID *string           `json:"existingChildId,omitempty"` // Set when merging/updating existing child
	MergeParents    bool              `json:"mergeParents,omitempty"`    // True if only adding parents to existing child
	FieldUpdates    map[string]string `json:"fieldUpdates,omitempty"`    // Field -> value for updates (from conflict resolution)
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
	ChildrenUpdated int           `json:"childrenUpdated"`
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
		var childID uuid.UUID
		var isExistingChild bool

		// Check if this is an update/merge for an existing child
		if row.ExistingChildID != nil && *row.ExistingChildID != "" {
			existingID, err := uuid.Parse(*row.ExistingChildID)
			if err != nil {
				result.Errors = append(result.Errors, ImportError{
					RowIndex: row.Index,
					Error:    "Ungültige Kind-ID",
				})
				continue
			}

			// Get existing child
			existingChild, err := s.childRepo.GetByID(ctx, existingID)
			if err != nil {
				result.Errors = append(result.Errors, ImportError{
					RowIndex: row.Index,
					Error:    fmt.Sprintf("Kind nicht gefunden: %v", err),
				})
				continue
			}

			childID = existingID
			isExistingChild = true

			// If not just merging parents, update child fields
			if !row.MergeParents && len(row.FieldUpdates) > 0 {
				// Apply field updates
				updated := false
				for field, value := range row.FieldUpdates {
					switch field {
					case "firstName":
						existingChild.FirstName = value
						updated = true
					case "lastName":
						existingChild.LastName = value
						updated = true
					case "birthDate":
						if t, err := time.Parse("2006-01-02", value); err == nil {
							existingChild.BirthDate = t
							updated = true
						}
					case "entryDate":
						if t, err := time.Parse("2006-01-02", value); err == nil {
							existingChild.EntryDate = t
							updated = true
						}
					case "legalHours":
						if hours, err := csvparser.ParseInt(value); err == nil {
							existingChild.LegalHours = &hours
							updated = true
						}
					case "careHours":
						if hours, err := csvparser.ParseInt(value); err == nil {
							existingChild.CareHours = &hours
							updated = true
						}
					}
				}

				if updated {
					err = s.childRepo.Update(ctx, existingChild)
					if err != nil {
						result.Errors = append(result.Errors, ImportError{
							RowIndex: row.Index,
							Error:    fmt.Sprintf("Fehler beim Aktualisieren: %v", err),
						})
						continue
					}
					result.ChildrenUpdated++
				}
			}
		} else {
			// Create new child
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

			childID = child.ID
			result.ChildrenCreated++
		}

		// Handle parent 1
		if row.Parent1 != nil && row.Parent1.FirstName != "" && row.Parent1.LastName != "" {
			// Skip if already linked
			if row.Parent1.AlreadyLinked {
				// Parent already linked, nothing to do
			} else {
				parentID, created, err := s.handleParent(ctx, row.Index, 1, row.Parent1, parentDecisionMap)
				if err != nil {
					result.Errors = append(result.Errors, ImportError{
						RowIndex: row.Index,
						Error:    fmt.Sprintf("Fehler bei Elternteil 1: %v", err),
					})
				} else if parentID != uuid.Nil {
					// Link parent to child
					isPrimary := !isExistingChild // First parent is primary only for new children
					err = s.childRepo.LinkParent(ctx, childID, parentID, isPrimary)
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
		}

		// Handle parent 2
		if row.Parent2 != nil && row.Parent2.FirstName != "" && row.Parent2.LastName != "" {
			// Skip if already linked
			if row.Parent2.AlreadyLinked {
				// Parent already linked, nothing to do
			} else {
				parentID, created, err := s.handleParent(ctx, row.Index, 2, row.Parent2, parentDecisionMap)
				if err != nil {
					result.Errors = append(result.Errors, ImportError{
						RowIndex: row.Index,
						Error:    fmt.Sprintf("Fehler bei Elternteil 2: %v", err),
					})
				} else if parentID != uuid.Nil {
					// Link parent to child
					isPrimary := false // Second parent is not primary
					err = s.childRepo.LinkParent(ctx, childID, parentID, isPrimary)
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

	// Check if parent with same name and email already exists
	if parent.Email != "" {
		existingParent, err := s.parentRepo.FindByNameAndEmail(ctx, parent.FirstName, parent.LastName, parent.Email)
		if err != nil {
			return uuid.Nil, false, fmt.Errorf("Fehler bei Duplikatprüfung: %v", err)
		}
		if existingParent != nil {
			// Parent already exists, reuse it
			return existingParent.ID, false, nil
		}
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
