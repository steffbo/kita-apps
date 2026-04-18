package service

import (
	"strings"
	"testing"
	"time"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

func TestBuildFamilyMembershipReminderEmail_InitialStageUsesNeutralWording(t *testing.T) {
	subject, body := buildFamilyMembershipReminderEmail(
		ReminderStageInitial,
		time.Date(2026, time.April, 18, 0, 0, 0, 0, time.UTC),
		[]string{"Anna"},
		[]reminderItem{{
			ChildName:    "Mia",
			MemberNumber: "4711",
			FeeType:      domain.FeeTypeMembership,
			Amount:       domain.MembershipFeeAmount,
			Year:         2026,
		}},
		nil,
	)

	if subject != "Kita Zahlungserinnerung Vereinsbeitrag 2026" {
		t.Fatalf("unexpected subject: %s", subject)
	}
	if !strings.Contains(body, "ist folgender Vereinsbeitrag offen") {
		t.Fatalf("expected membership reminder wording in body, got: %s", body)
	}
	if !strings.Contains(body, "Vereinsbeitrag 2026 — 30,00 EUR") {
		t.Fatalf("expected membership fee line in body, got: %s", body)
	}
	if strings.Contains(body, "Mia") {
		t.Fatalf("did not expect child name in membership email, got: %s", body)
	}
	if strings.Contains(body, "Mitgliedsnr.") {
		t.Fatalf("did not expect child member number hint in membership email, got: %s", body)
	}
	if !strings.Contains(body, "bis zum 31.03.2026") {
		t.Fatalf("expected default deadline 31.03.<year>, got: %s", body)
	}
	if strings.Contains(body, "wird leider automatisch eine Mahngebühr fällig") {
		t.Fatalf("did not expect automatic reminder fee warning, got: %s", body)
	}
}

func TestBuildFamilyMembershipReminderEmail_FinalStageUsesDunningWording(t *testing.T) {
	subject, body := buildFamilyMembershipReminderEmail(
		ReminderStageFinal,
		time.Date(2026, time.April, 18, 0, 0, 0, 0, time.UTC),
		[]string{"Anna", "Ben"},
		[]reminderItem{{
			ChildName:    "Mia",
			MemberNumber: "4711",
			FeeType:      domain.FeeTypeMembership,
			Amount:       domain.MembershipFeeAmount,
			Year:         2026,
		}},
		nil,
	)

	if subject != "Kita Mahnung Vereinsbeitrag 2026" {
		t.Fatalf("unexpected subject: %s", subject)
	}
	if !strings.Contains(body, "ist folgender offener Vereinsbeitrag vermerkt") {
		t.Fatalf("expected dunning wording in body, got: %s", body)
	}
	if !strings.Contains(body, "Dies ist eine Mahnung") {
		t.Fatalf("expected final warning wording in body, got: %s", body)
	}
	if !strings.Contains(body, "spätestens bis zum 31.03.2026") {
		t.Fatalf("expected default deadline 31.03.<year> in dunning body, got: %s", body)
	}
	if strings.Contains(body, "Mia") {
		t.Fatalf("did not expect child name in membership dunning email, got: %s", body)
	}
	if strings.Contains(body, "Mitgliedsnr.") {
		t.Fatalf("did not expect child member number hint in membership dunning email, got: %s", body)
	}
}

func TestBuildFamilyMembershipReminderEmail_FinalStageIncludesMembershipReminderFeeLine(t *testing.T) {
	_, body := buildFamilyMembershipReminderEmail(
		ReminderStageFinal,
		time.Date(2026, time.April, 18, 0, 0, 0, 0, time.UTC),
		[]string{"Anna", "Ben"},
		[]reminderItem{
			{
				ChildName:    "Mia",
				MemberNumber: "4711",
				FeeType:      domain.FeeTypeMembership,
				Amount:       domain.MembershipFeeAmount,
				Year:         2026,
			},
			{
				ChildName:    "Mia",
				MemberNumber: "4711",
				FeeType:      domain.FeeTypeReminder,
				Amount:       domain.MembershipReminderFeeAmount,
				BaseFeeType:  feeTypePtr(domain.FeeTypeMembership),
				BaseYear:     2026,
			},
		},
		nil,
	)

	if !strings.Contains(body, "Mahngebühr für Vereinsbeitrag 2026 — 5,00 EUR") {
		t.Fatalf("expected membership reminder fee line in body, got: %s", body)
	}
	if strings.Contains(body, "Mia") {
		t.Fatalf("did not expect child name in multi-item membership email, got: %s", body)
	}
	if strings.Contains(body, "Mitgliedsnr.") {
		t.Fatalf("did not expect child member number hint in multi-item membership email, got: %s", body)
	}
}

func TestBuildFamilyMembershipReminderEmail_UsesDeadlineOverrideWhenProvided(t *testing.T) {
	override := time.Date(2026, time.May, 5, 0, 0, 0, 0, time.UTC)

	_, body := buildFamilyMembershipReminderEmail(
		ReminderStageInitial,
		time.Date(2026, time.April, 18, 0, 0, 0, 0, time.UTC),
		[]string{"Anna"},
		[]reminderItem{{
			ChildName:    "Mia",
			MemberNumber: "4711",
			FeeType:      domain.FeeTypeMembership,
			Amount:       domain.MembershipFeeAmount,
			Year:         2026,
		}},
		&override,
	)

	if !strings.Contains(body, "bis zum 05.05.2026") {
		t.Fatalf("expected override deadline in body, got: %s", body)
	}
	if strings.Contains(body, "31.03.2026") {
		t.Fatalf("did not expect default deadline when override is set, got: %s", body)
	}
}
