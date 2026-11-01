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

// AssignTransaction gives an unassigned incoming transaction a purpose (currently only donations).
// Fee matches, a plan payment or an earlier purpose make the transaction ineligible.
func (s *ImportService) AssignTransaction(ctx context.Context, transactionID uuid.UUID, kind domain.AssignmentKind, note string, userID uuid.UUID) error {
	note = strings.TrimSpace(note)
	if kind != domain.AssignmentDonation || utf8.RuneCountInString(note) > maxAssignmentNoteLength {
		return ErrInvalidInput
	}
	tx, err := s.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return ErrNotFound
	}
	if tx.Amount <= 0 || tx.HasExclusiveAssignment() {
		return ErrInvalidInput
	}
	switch err := s.transactionRepo.Assign(ctx, transactionID, kind, note, userID); {
	case errors.Is(err, repository.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, repository.ErrDuplicate):
		return ErrInvalidInput
	default:
		return err
	}
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

// GetDonations groups the donation transactions by booking year.
func (s *ImportService) GetDonations(ctx context.Context) (*DonationReport, error) {
	rows, err := s.transactionRepo.ListAssigned(ctx, domain.AssignmentDonation)
	if err != nil {
		return nil, err
	}
	report := &DonationReport{Years: []DonationYear{}}
	var total int64
	var yearTotal int64 // cents of the last group in report.Years
	for _, row := range rows {
		year := row.BookingDate.Year()
		if n := len(report.Years); n == 0 || report.Years[n-1].Year != year {
			report.Years = append(report.Years, DonationYear{Year: year})
			yearTotal = 0
		}
		group := &report.Years[len(report.Years)-1]
		group.Transactions = append(group.Transactions, row)
		group.Count++
		yearTotal += domain.Cents(row.Amount)
		group.Total = domain.Euros(yearTotal)
		total += domain.Cents(row.Amount)
		report.Count++
	}
	report.Total = domain.Euros(total)
	return report, nil
}
