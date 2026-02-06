package service

import (
	"context"
	"time"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
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
	now := time.Now()
	nextStichtag := calculateNextStichtag(now)

	stats, err := s.childRepo.GetStichtagsmeldungStats(ctx, nextStichtag)
	if err != nil {
		return nil, err
	}

	stats.NextStichtag = nextStichtag
	stats.DaysUntilStichtag = int(nextStichtag.Sub(now).Hours() / 24)

	return stats, nil
}

// GetU3Children returns details of U3 children for the Stichtagsmeldung modal.
func (s *StichtagsmeldungService) GetU3Children(ctx context.Context) ([]domain.U3ChildDetail, error) {
	now := time.Now()
	nextStichtag := calculateNextStichtag(now)
	return s.childRepo.GetU3ChildrenDetails(ctx, nextStichtag)
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
