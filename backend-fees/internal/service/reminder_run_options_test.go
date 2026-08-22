package service

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestApplyReminderOverrides_NilOptionsKeepsGenerated(t *testing.T) {
	subject, body := applyReminderOverrides("S", "B", nil, uuid.New())
	if subject != "S" || body != "B" {
		t.Fatalf("expected generated values, got subject=%q body=%q", subject, body)
	}
}

func TestApplyReminderOverrides_ReplacesSubjectAndBody(t *testing.T) {
	householdID := uuid.New()
	options := &ReminderRunOptions{
		Overrides: map[uuid.UUID]ReminderOverride{
			householdID: {Subject: "Neuer Betreff", Body: "Neuer Text"},
		},
	}
	subject, body := applyReminderOverrides("S", "B", options, householdID)
	if subject != "Neuer Betreff" || body != "Neuer Text" {
		t.Fatalf("expected overrides applied, got subject=%q body=%q", subject, body)
	}
}

func TestApplyReminderOverrides_EmptyFieldsKeepGenerated(t *testing.T) {
	householdID := uuid.New()
	options := &ReminderRunOptions{
		Overrides: map[uuid.UUID]ReminderOverride{
			householdID: {Subject: "  ", Body: ""},
		},
	}
	subject, body := applyReminderOverrides("S", "B", options, householdID)
	if subject != "S" || body != "B" {
		t.Fatalf("expected generated values kept for blank override fields, got subject=%q body=%q", subject, body)
	}
}

func TestApplyReminderOverrides_OnlyAffectsTargetHousehold(t *testing.T) {
	options := &ReminderRunOptions{
		Overrides: map[uuid.UUID]ReminderOverride{
			uuid.New(): {Subject: "Fremd", Body: strings.Repeat("x", 10)},
		},
	}
	subject, body := applyReminderOverrides("S", "B", options, uuid.New())
	if subject != "S" || body != "B" {
		t.Fatalf("expected other households untouched, got subject=%q body=%q", subject, body)
	}
}

func TestReminderQRDisabled(t *testing.T) {
	cases := []struct {
		name    string
		options *ReminderRunOptions
		want    bool
	}{
		{"nil options", nil, false},
		{"no include flag", &ReminderRunOptions{}, false},
		{"include true", &ReminderRunOptions{IncludeQR: boolPtr(true)}, false},
		{"include false", &ReminderRunOptions{IncludeQR: boolPtr(false)}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := reminderQRDisabled(tc.options); got != tc.want {
				t.Fatalf("reminderQRDisabled = %v, want %v", got, tc.want)
			}
		})
	}
}

func boolPtr(v bool) *bool {
	return &v
}
