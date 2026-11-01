package service

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

const maxAssignmentNoteLength = 500

func validAssignmentInput(in *domain.AssignmentInput) bool {
	in.Note = strings.TrimSpace(in.Note)
	return in.Kind == domain.AssignmentDonation && utf8.RuneCountInString(in.Note) <= maxAssignmentNoteLength
}

// mapAssignmentError translates repository errors; a missing household or transaction is ErrNotFound.
func mapAssignmentError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, repository.ErrDuplicate):
		return ErrInvalidInput
	default:
		return err
	}
}

// AssignTransaction gives an unassigned incoming transaction a purpose (currently only donations),
// optionally credited to a household. Hidden transactions, fee matches, a plan payment or an earlier
// purpose make it ineligible. Open warnings are settled with the assignment, as with fee matches.
func (s *ImportService) AssignTransaction(ctx context.Context, transactionID uuid.UUID, in domain.AssignmentInput, userID uuid.UUID) error {
	if !validAssignmentInput(&in) {
		return ErrInvalidInput
	}
	return s.txm.WithTx(ctx, func(ctx context.Context) error {
		tx, err := s.transactionRepo.GetByID(ctx, transactionID)
		if err != nil {
			return mapAssignmentError(err)
		}
		if tx.Amount <= 0 || tx.IsHidden || tx.HasExclusiveAssignment() {
			return ErrInvalidInput
		}
		if err := s.transactionRepo.Assign(ctx, transactionID, in, userID); err != nil {
			return mapAssignmentError(err)
		}
		return s.warningRepo.ResolveByTransactionID(ctx, transactionID, domain.ResolutionTypeMatched, "Als Spende zugeordnet")
	})
}

// UpdateAssignment changes note and household of an existing assignment.
func (s *ImportService) UpdateAssignment(ctx context.Context, transactionID uuid.UUID, in domain.AssignmentInput) error {
	if !validAssignmentInput(&in) {
		return ErrInvalidInput
	}
	return mapAssignmentError(s.transactionRepo.UpdateAssignment(ctx, transactionID, in))
}

// DonationYear sums the donations booked in one calendar year.
type DonationYear struct {
	Year         int                      `json:"year"`
	Total        float64                  `json:"total"`
	Count        int                      `json:"count"`
	Transactions []domain.BankTransaction `json:"transactions"`
}

// DonationReport lists all donations grouped by booking year, newest year first.
type DonationReport struct {
	Total float64        `json:"total"`
	Count int            `json:"count"`
	Years []DonationYear `json:"years"`
}

// GetDonations groups the donation transactions by booking year; householdID narrows them to one household.
func (s *ImportService) GetDonations(ctx context.Context, householdID *uuid.UUID) (*DonationReport, error) {
	rows, err := s.transactionRepo.ListAssigned(ctx, domain.AssignmentDonation, householdID)
	if err != nil {
		return nil, err
	}
	return groupDonations(rows), nil
}

// groupDonations expects rows sorted by booking date, newest first. Sums are built in cents.
func groupDonations(rows []domain.BankTransaction) *DonationReport {
	report := &DonationReport{Years: []DonationYear{}}
	var total, yearTotal int64
	for _, row := range rows {
		year := row.BookingDate.Year()
		if n := len(report.Years); n == 0 || report.Years[n-1].Year != year {
			report.Years = append(report.Years, DonationYear{Year: year})
			yearTotal = 0
		}
		cents := domain.Cents(row.Amount)
		total += cents
		yearTotal += cents
		group := &report.Years[len(report.Years)-1]
		group.Transactions = append(group.Transactions, row)
		group.Count++
		group.Total = domain.Euros(yearTotal)
		report.Count++
	}
	report.Total = domain.Euros(total)
	return report
}
