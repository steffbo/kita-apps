package service

import (
	"context"
	"strings"
	"time"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/letter"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/util"
)

// StichtagsmeldungService handles Stichtagsmeldung-related business logic.
type StichtagsmeldungService struct {
	childRepo repository.ChildRepository
}

// NewStichtagsmeldungService creates a new Stichtagsmeldung service.
func NewStichtagsmeldungService(childRepo repository.ChildRepository) *StichtagsmeldungService {
	return &StichtagsmeldungService{
		childRepo: childRepo,
	}
}

// GetStats returns the Stichtagsmeldung statistics including the next Stichtag date
// and U3 children income breakdown.
func (s *StichtagsmeldungService) GetStats(ctx context.Context) (*domain.StichtagsmeldungStats, error) {
	now := util.Today()
	nextStichtag := calculateNextStichtag(now)

	stats, err := s.childRepo.GetStichtagsmeldungStats(ctx, nextStichtag)
	if err != nil {
		return nil, err
	}

	stats.NextStichtag = nextStichtag
	stats.DaysUntilStichtag = int(nextStichtag.Sub(now).Hours() / 24)

	return stats, nil
}

// GetReport returns the Stichtagsmeldung report for a specific date.
func (s *StichtagsmeldungService) GetReport(ctx context.Context, reportDate time.Time) (*domain.StichtagsmeldungReport, error) {
	reportDate = time.Date(reportDate.Year(), reportDate.Month(), reportDate.Day(), 0, 0, 0, 0, reportDate.Location())

	report, err := s.childRepo.GetStichtagsmeldungReport(ctx, reportDate)
	if err != nil {
		return nil, err
	}

	report.ReportDate = reportDate
	return report, nil
}

// GetU3ChildrenForDate returns U3 child details for the provided report date.
func (s *StichtagsmeldungService) GetU3ChildrenForDate(ctx context.Context, reportDate *time.Time) ([]domain.U3ChildDetail, error) {
	if reportDate != nil {
		stichtag := time.Date(reportDate.Year(), reportDate.Month(), reportDate.Day(), 0, 0, 0, 0, reportDate.Location())
		return s.childRepo.GetU3ChildrenDetails(ctx, stichtag)
	}

	now := util.Today()
	nextStichtag := calculateNextStichtag(now)
	return s.childRepo.GetU3ChildrenDetails(ctx, nextStichtag)
}

// GetJugendamtReport returns the counts for the quarterly Stichtagsmeldung to the
// Landkreis at the given date, or at the Stichtag the dashboard shows when date is nil.
func (s *StichtagsmeldungService) GetJugendamtReport(ctx context.Context, date *time.Time) (*domain.JugendamtReport, error) {
	today := util.Today()
	stichtag := domain.StichtagForJugendamt(today)
	if date != nil {
		stichtag = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	}

	children, err := s.childRepo.ListEnrolledAt(ctx, stichtag)
	if err != nil {
		return nil, err
	}

	report := &domain.JugendamtReport{
		Stichtag:  stichtag,
		DaysUntil: int(stichtag.Sub(today).Hours() / 24),
		Children:  make([]domain.JugendamtChild, 0, len(children)),
	}
	for _, child := range children {
		report.Add(child)
	}
	return report, nil
}

// GetChildListsPDF prints the children enrolled at the Stichtag of the report
// (see GetJugendamtReport) as two lists: "Krippe" (U3) and "Kita" (Ü3).
func (s *StichtagsmeldungService) GetChildListsPDF(ctx context.Context, date *time.Time) (*LetterFile, error) {
	report, err := s.GetJugendamtReport(ctx, date)
	if err != nil {
		return nil, err
	}
	krippe := letter.ChildList{Title: "Krippe"}
	kita := letter.ChildList{Title: "Kita"}
	for _, child := range report.Children {
		row := letter.ChildListRow{
			LastName:    child.LastName,
			FirstName:   child.FirstName,
			BirthDate:   child.BirthDate,
			Street:      strings.TrimSpace(derefString(child.Street) + " " + derefString(child.StreetNo)),
			PostalCode:  strings.TrimSpace(derefString(child.PostalCode)),
			City:        strings.TrimSpace(derefString(child.City)),
			WeeklyHours: child.CareHours,
		}
		if child.Group == domain.KitaYearGroupU3 {
			krippe.Rows = append(krippe.Rows, row)
		} else {
			kita.Rows = append(kita.Rows, row)
		}
	}
	data, err := letter.RenderChildLists(report.Stichtag, util.Today(), []letter.ChildList{krippe, kita})
	if err != nil {
		return nil, err
	}
	return &LetterFile{Filename: "Kinderlisten_" + report.Stichtag.Format("2006-01-02") + ".pdf", Data: data}, nil
}

// calculateNextStichtag finds the next Stichtag date (15th of Dec/Mar/Jun/Sep).
func calculateNextStichtag(now time.Time) time.Time {
	// Stichtag months: March (3), June (6), September (9), December (12)
	stichtagMonths := []time.Month{time.March, time.June, time.September, time.December}

	year := now.Year()
	day := 15

	for _, month := range stichtagMonths {
		stichtag := time.Date(year, month, day, 0, 0, 0, 0, now.Location())
		if stichtag.After(now) {
			return stichtag
		}
	}

	// If we're past December 15, return March 15 of next year
	return time.Date(year+1, time.March, day, 0, 0, 0, 0, now.Location())
}
