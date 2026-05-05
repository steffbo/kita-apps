package service

import (
	"strings"
	"testing"
	"time"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

func TestBuildSEPAPayload_FormatsAmountAndReference(t *testing.T) {
	settings := ReminderPaymentSettings{
		RecipientName: "Knirpsenstadt e.V.",
		IBAN:          "DE33370205000003321400",
		BIC:           "BFSWDE33XXX",
	}

	payload, err := buildSEPAPayload(settings, 123.45, "Essensbeitrag April 2026 Schmidt 12345")
	if err != nil {
		t.Fatalf("expected payload build to succeed: %v", err)
	}

	lines := strings.Split(payload, "\n")
	if len(lines) != 11 {
		t.Fatalf("expected 11 lines in EPC payload, got %d", len(lines))
	}
	if lines[0] != "BCD" {
		t.Fatalf("expected BCD marker, got %q", lines[0])
	}
	if lines[7] != "EUR123.45" {
		t.Fatalf("expected amount line EUR123.45, got %q", lines[7])
	}
	if lines[9] != "Essensbeitrag April 2026 Schmidt 12345" {
		t.Fatalf("unexpected reference line: %q", lines[9])
	}
}

func TestBuildSEPAReference_IncludesPurposeFamilyAndMemberNumbers(t *testing.T) {
	runDate := time.Date(2026, time.April, 5, 0, 0, 0, 0, time.UTC)
	items := []reminderItem{
		{FeeType: domain.FeeTypeFood, MemberNumber: "12345"},
		{FeeType: domain.FeeTypeFood, MemberNumber: "12346"},
		{FeeType: domain.FeeTypeFood, MemberNumber: "12345"}, // duplicate should be deduplicated
	}

	reference := buildSEPAReference(runDate, "Schmidt", items)

	if !strings.Contains(reference, "Essensbeitrag") {
		t.Fatalf("expected purpose in reference, got: %s", reference)
	}
	if !strings.Contains(reference, "April 2026") {
		t.Fatalf("expected month/year in reference, got: %s", reference)
	}
	if !strings.Contains(reference, "Schmidt") {
		t.Fatalf("expected household name in reference, got: %s", reference)
	}
	if !strings.Contains(reference, "12345+12346") {
		t.Fatalf("expected combined member numbers in reference, got: %s", reference)
	}
}

func TestBuildSEPAReference_OmitsMonthForMembershipFee(t *testing.T) {
	runDate := time.Date(2026, time.May, 5, 0, 0, 0, 0, time.UTC)
	items := []reminderItem{
		{FeeType: domain.FeeTypeMembership, MemberNumber: "11038"},
	}

	reference := buildSEPAReference(runDate, "Wrana", items)

	if reference != "Vereinsbeitrag 2026 Wrana 11038" {
		t.Fatalf("unexpected membership reference: %s", reference)
	}
	if strings.Contains(reference, "Mai") {
		t.Fatalf("did not expect month in membership reference, got: %s", reference)
	}
}

func TestBuildReminderQRCode_UsesLegacyDefaultsWhenSettingsMissing(t *testing.T) {
	service := &ReminderService{}
	runDate := time.Date(2026, time.April, 5, 0, 0, 0, 0, time.UTC)

	data, err := service.buildReminderQRCode(ReminderPaymentSettings{}, runDate, "Schmidt", []reminderItem{{Amount: 45.40}})
	if err != nil {
		t.Fatalf("expected no error for missing settings, got: %v", err)
	}
	if data == nil {
		t.Fatalf("expected QR data with legacy defaults")
	}
	if !strings.HasPrefix(data.Payload, "BCD\n001\n1\nSCT\n") {
		t.Fatalf("expected valid EPC payload prefix, got: %s", data.Payload)
	}
}
