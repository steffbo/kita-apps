package service

import (
	"strings"
	"testing"
	"time"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

func TestBuildFamilyReminderEmail_InitialStageUsesReminderWording(t *testing.T) {
	subject, body := buildFamilyReminderEmail(
		ReminderStageInitial,
		time.Date(2026, time.April, 5, 0, 0, 0, 0, time.UTC),
		[]string{"Anna"},
		[]reminderItem{{
			ChildName:    "Mia",
			MemberNumber: "4711",
			FeeType:      domain.FeeTypeFood,
			Amount:       45.40,
			Year:         2026,
			Month:        4,
		}},
		nil,
		ReminderPaymentSettings{},
	)

	if subject != "Kita Zahlungserinnerung April 2026" {
		t.Fatalf("unexpected subject: %s", subject)
	}
	if !strings.Contains(body, "ist folgender Beitrag offen") {
		t.Fatalf("expected reminder wording in body, got: %s", body)
	}
	if !strings.Contains(body, "Essensgeld April/2026 — 45,40 EUR") {
		t.Fatalf("expected fee line in body, got: %s", body)
	}
	if !strings.Contains(body, "bis zum 12.04.2026") {
		t.Fatalf("expected default deadline 7 days after run date, got: %s", body)
	}
	if !strings.Contains(body, "wird leider automatisch eine Mahngebühr fällig") {
		t.Fatalf("expected reminder fee warning in body, got: %s", body)
	}
	if strings.Contains(body, "Dies ist eine Mahnung") {
		t.Fatalf("did not expect final warning wording in reminder body, got: %s", body)
	}
}

func TestBuildFamilyReminderEmail_FinalStageUsesDunningWording(t *testing.T) {
	subject, body := buildFamilyReminderEmail(
		ReminderStageFinal,
		time.Date(2026, time.April, 10, 0, 0, 0, 0, time.UTC),
		[]string{"Anna", "Ben"},
		[]reminderItem{{
			ChildName:    "Mia",
			MemberNumber: "4711",
			FeeType:      domain.FeeTypeChildcare,
			Amount:       120.00,
			Year:         2026,
			Month:        4,
		}},
		nil,
		ReminderPaymentSettings{},
	)

	if subject != "Kita Mahnung April 2026" {
		t.Fatalf("unexpected subject: %s", subject)
	}
	if !strings.Contains(body, "ist folgender offener Beitrag vermerkt") {
		t.Fatalf("expected dunning wording in body, got: %s", body)
	}
	if !strings.Contains(body, "Dies ist eine Mahnung") {
		t.Fatalf("expected final warning wording in body, got: %s", body)
	}
	if !strings.Contains(body, "spätestens bis zum 17.04.2026") {
		t.Fatalf("expected default deadline 7 days after run date, got: %s", body)
	}
	if strings.Contains(body, "wird leider automatisch eine Mahngebühr fällig") {
		t.Fatalf("did not expect reminder wording in dunning body, got: %s", body)
	}
}

func TestBuildFamilyReminderEmail_FinalStageIncludesReminderFeeLine(t *testing.T) {
	subject, body := buildFamilyReminderEmail(
		ReminderStageFinal,
		time.Date(2026, time.April, 10, 0, 0, 0, 0, time.UTC),
		[]string{"Anna", "Ben"},
		[]reminderItem{
			{
				ChildName:    "Mia",
				MemberNumber: "4711",
				FeeType:      domain.FeeTypeChildcare,
				Amount:       120.00,
				Year:         2026,
				Month:        4,
			},
			{
				ChildName:    "Mia",
				MemberNumber: "4711",
				FeeType:      domain.FeeTypeReminder,
				Amount:       domain.ReminderFeeAmount,
				BaseFeeType:  feeTypePtr(domain.FeeTypeChildcare),
				BaseYear:     2026,
				BaseMonth:    4,
			},
		},
		nil,
		ReminderPaymentSettings{},
	)

	if subject != "Kita Mahnung April 2026" {
		t.Fatalf("unexpected subject: %s", subject)
	}
	if !strings.Contains(body, "Mia (Mitgliedsnr. 4711): Mahngebühr für Platzgeld April/2026 — 10,00 EUR") {
		t.Fatalf("expected reminder fee line in body, got: %s", body)
	}
}

func TestBuildFamilyReminderEmail_UsesDeadlineOverrideWhenProvided(t *testing.T) {
	override := time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC)

	_, body := buildFamilyReminderEmail(
		ReminderStageInitial,
		time.Date(2026, time.April, 5, 0, 0, 0, 0, time.UTC),
		[]string{"Anna"},
		[]reminderItem{{
			ChildName:    "Mia",
			MemberNumber: "4711",
			FeeType:      domain.FeeTypeFood,
			Amount:       45.40,
			Year:         2026,
			Month:        4,
		}},
		&override,
		ReminderPaymentSettings{},
	)

	if !strings.Contains(body, "bis zum 30.04.2026") {
		t.Fatalf("expected override deadline in body, got: %s", body)
	}
	if strings.Contains(body, "bis zum 12.04.2026") {
		t.Fatalf("did not expect default deadline when override is set, got: %s", body)
	}
}

func feeTypePtr(feeType domain.FeeType) *domain.FeeType {
	return &feeType
}
