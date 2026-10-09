package service

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/csvparser"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
	"time"
)

// ExecuteRequest contains the data to import
type ExecuteRequest struct {
	Rows            []ImportRow      `json:"rows"`
	ParentDecisions []ParentDecision `json:"parentDecisions"`
}

// ImportRow is a row to be imported
type ImportRow struct {
	Index           int               `json:"index"`
	Child           ChildPreview      `json:"child"`
	Parent1         *ParentPreview    `json:"parent1,omitempty" binding:"optional"`
	Parent2         *ParentPreview    `json:"parent2,omitempty" binding:"optional"`
	ExistingChildID *string           `json:"existingChildId,omitempty" binding:"optional"` // Set when merging/updating existing child
	MergeParents    bool              `json:"mergeParents,omitempty" binding:"optional"`    // True if only adding parents to existing child
	FieldUpdates    map[string]string `json:"fieldUpdates,omitempty" binding:"optional"`    // Field -> value for updates (from conflict resolution)
}

// ParentDecision indicates how to handle a parent
type ParentDecision struct {
	RowIndex         int    `json:"rowIndex"`
	ParentIndex      int    `json:"parentIndex"` // 1 or 2
	Action           string `json:"action"`      // "create" or "link"
	ExistingParentID string `json:"existingParentId,omitempty" binding:"optional"`
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
		var rowResult *ExecuteResult
		err := s.txm.WithTx(ctx, func(txctx context.Context) error {
			rowResult = s.executeRow(txctx, row, parentDecisionMap)
			if len(rowResult.Errors) > 0 {
				return fmt.Errorf("%s", rowResult.Errors[0].Error)
			}
			return nil
		})
		if rowResult != nil {
			result.ChildrenCreated += rowResult.ChildrenCreated
			result.ChildrenUpdated += rowResult.ChildrenUpdated
			result.ParentsCreated += rowResult.ParentsCreated
			result.ParentsLinked += rowResult.ParentsLinked
			result.Errors = append(result.Errors, rowResult.Errors...)
		}
		if err != nil && (rowResult == nil || len(rowResult.Errors) == 0) {
			result.Errors = append(result.Errors, ImportError{RowIndex: row.Index, Error: err.Error()})
		}
	}

	return result, nil
}

func (s *ChildImportService) executeRow(
	ctx context.Context, row ImportRow, parentDecisionMap map[string]ParentDecision,
) *ExecuteResult {
	result := &ExecuteResult{Errors: []ImportError{}}
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
			return result
		}

		// Get existing child
		existingChild, err := s.childRepo.GetByID(ctx, existingID)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				RowIndex: row.Index,
				Error:    fmt.Sprintf("Kind nicht gefunden: %v", err),
			})
			return result
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
					return result
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
			return result
		}

		entryDate, err := time.Parse("2006-01-02", row.Child.EntryDate)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				RowIndex: row.Index,
				Error:    "Ungültiges Eintrittsdatum",
			})
			return result
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
			CreatedAt:    util.Now(),
			UpdatedAt:    util.Now(),
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
			return result
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
	return result
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
		CreatedAt: util.Now(),
		UpdatedAt: util.Now(),
	}

	err := s.parentRepo.Create(ctx, newParent)
	if err != nil {
		return uuid.Nil, false, err
	}

	return newParent.ID, true, nil
}

// Note: stringPtr and stringOrEmpty are defined in import_service.go
