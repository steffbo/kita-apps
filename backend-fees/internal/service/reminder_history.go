package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

const settingReminderHistoryReliableFrom = "reminder_history_reliable_from"

// reminderEmailLogTypes are the log types that count as reminder contact.
var reminderEmailLogTypes = []domain.EmailLogType{
	domain.EmailLogTypeReminderInitial,
	domain.EmailLogTypeReminderFinal,
	domain.EmailLogTypeMembershipReminderInitial,
	domain.EmailLogTypeMembershipReminderFinal,
}

// FeeContact describes the last known reminder contact for a single fee.
type FeeContact struct {
	LastContactAt time.Time
	RunDate       time.Time
	Stage         ReminderStage
}

// ResolveFeeContacts derives per-fee last contact from reminder email logs.
// Logs must be ordered newest first; the newest log containing a fee id wins.
// Logs without resolvable stage/runDate in their payload are ignored.
func ResolveFeeContacts(logs []domain.EmailLog) map[uuid.UUID]FeeContact {
	contacts := make(map[uuid.UUID]FeeContact)
	for _, entry := range logs {
		stageRaw, ok := entry.StageFromPayload()
		if !ok {
			continue
		}
		stage := ReminderStage(strings.ToLower(stageRaw))
		if stage != ReminderStageInitial && stage != ReminderStageFinal {
			continue
		}
		var payload struct {
			RunDate string `json:"runDate"`
		}
		if entry.Payload == nil {
			continue
		}
		if err := json.Unmarshal(*entry.Payload, &payload); err != nil {
			continue
		}
		runDate, err := time.Parse("2006-01-02", payload.RunDate)
		if err != nil {
			continue
		}
		for _, feeID := range entry.FeeIDsFromPayload() {
			if _, exists := contacts[feeID]; exists {
				continue
			}
			contacts[feeID] = FeeContact{
				LastContactAt: entry.SentAt,
				RunDate:       runDate,
				Stage:         stage,
			}
		}
	}
	return contacts
}

// ListFeeContacts loads the last reminder contact per fee for a household.
func (s *ReminderService) ListFeeContacts(ctx context.Context, householdID uuid.UUID) (map[uuid.UUID]FeeContact, error) {
	if s.emailLogRepo == nil {
		return map[uuid.UUID]FeeContact{}, nil
	}
	logs, err := s.emailLogRepo.ListByHouseholdAndTypes(ctx, householdID, reminderEmailLogTypes)
	if err != nil {
		return nil, err
	}
	return ResolveFeeContacts(logs), nil
}

// GetHistoryReliableFrom returns the reliability cutoff date for fee contact
// history (migration 000030). Returns the zero time when unset.
func (s *ReminderService) GetHistoryReliableFrom(ctx context.Context) (time.Time, error) {
	if s.settingsRepo == nil {
		return time.Time{}, nil
	}
	setting, err := s.settingsRepo.Get(ctx, settingReminderHistoryReliableFrom)
	if err != nil {
		if err == repository.ErrNotFound {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(setting.Value))
	if err != nil {
		return time.Time{}, nil
	}
	return parsed, nil
}
