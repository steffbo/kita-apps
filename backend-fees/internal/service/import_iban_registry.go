package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

// markIBANAsTrusted marks the IBAN from a transaction as trusted.
func (s *ImportService) markIBANAsTrusted(ctx context.Context, transactionID uuid.UUID, childID *uuid.UUID) error {
	tx, err := s.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return err
	}
	if tx.PayerIBAN == nil {
		return nil
	}

	// Check if already known
	existing, err := s.knownIBANRepo.GetByIBAN(ctx, *tx.PayerIBAN)
	if err != nil {
		return err
	}
	if existing != nil {
		if existing.Status == domain.KnownIBANStatusBlacklisted {
			return nil
		}
		if existing.ChildID == nil && childID != nil {
			return s.knownIBANRepo.UpdateChildLink(ctx, existing.IBAN, childID)
		}
		return nil
	}

	knownIBAN := &domain.KnownIBAN{
		IBAN:                  *tx.PayerIBAN,
		PayerName:             tx.PayerName,
		Status:                domain.KnownIBANStatusTrusted,
		ChildID:               childID,
		Reason:                stringPtr("Automatically marked as trusted after successful match"),
		OriginalTransactionID: &tx.ID,
		OriginalDescription:   tx.Description,
		OriginalAmount:        &tx.Amount,
	}

	return s.knownIBANRepo.Create(ctx, knownIBAN)
}

// GetBlacklist returns all blacklisted IBANs.
func (s *ImportService) GetBlacklist(ctx context.Context, offset, limit int) ([]domain.KnownIBAN, int64, error) {
	return s.knownIBANRepo.ListByStatus(ctx, domain.KnownIBANStatusBlacklisted, offset, limit)
}

// RemoveFromBlacklist removes an IBAN from the blacklist.
func (s *ImportService) RemoveFromBlacklist(ctx context.Context, iban string) error {
	existing, err := s.knownIBANRepo.GetByIBAN(ctx, iban)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrNotFound
	}
	if existing.Status != domain.KnownIBANStatusBlacklisted {
		return ErrInvalidInput
	}

	return s.knownIBANRepo.Delete(ctx, iban)
}

// LinkIBANToChild links a trusted IBAN to a specific child.
func (s *ImportService) LinkIBANToChild(ctx context.Context, iban string, childID uuid.UUID) error {
	// Verify the IBAN exists and is trusted
	existing, err := s.knownIBANRepo.GetByIBAN(ctx, iban)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrNotFound
	}
	if existing.Status != domain.KnownIBANStatusTrusted {
		return ErrInvalidInput
	}

	// Verify child exists
	_, err = s.childRepo.GetByID(ctx, childID)
	if err != nil {
		return ErrNotFound
	}

	return s.knownIBANRepo.UpdateChildLink(ctx, iban, &childID)
}

// UnlinkIBANFromChild removes the child link from a trusted IBAN.
func (s *ImportService) UnlinkIBANFromChild(ctx context.Context, iban string) error {
	return s.knownIBANRepo.UpdateChildLink(ctx, iban, nil)
}

// GetTrustedIBANs returns all trusted IBANs.
func (s *ImportService) GetTrustedIBANs(ctx context.Context, offset, limit int) ([]domain.KnownIBAN, int64, error) {
	return s.knownIBANRepo.ListByStatus(ctx, domain.KnownIBANStatusTrusted, offset, limit)
}

// GetTrustedIBANsForChild returns trusted IBANs for a child with usage counts.
func (s *ImportService) GetTrustedIBANsForChild(ctx context.Context, childID uuid.UUID) ([]domain.KnownIBANSummary, error) {
	if s.knownIBANRepo == nil {
		return []domain.KnownIBANSummary{}, nil
	}
	return s.knownIBANRepo.ListTrustedByChildWithCounts(ctx, childID)
}
