package csvparser

import (
	"testing"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

func TestExtractMemberNumber_WithMissingSpace(t *testing.T) {
	description := "Kinderbetreuungsbeitrag Romy Bachle11089"
	memberNumber := ExtractMemberNumber(description)
	if memberNumber != "11089" {
		t.Fatalf("expected member number 11089, got %q", memberNumber)
	}
}

func TestExtractMemberNumber_WithLineBreak(t *testing.T) {
	description := "Kinderbetreuungsbeitrag Romy\nBächle 11089"
	memberNumber := ExtractMemberNumber(description)
	if memberNumber != "11089" {
		t.Fatalf("expected member number 11089, got %q", memberNumber)
	}
}

func TestMatchChildByName_WithUmlautNormalization(t *testing.T) {
	description := "Kinderbetreuungsbeitrag Romy Bachle11089"
	children := []domain.Child{
		{FirstName: "Romy", LastName: "Bächle"},
	}

	child, confidence := MatchChildByName(description, children)
	if child == nil {
		t.Fatalf("expected to match child, got nil")
	}
	if confidence < 0.8 {
		t.Fatalf("expected confidence >= 0.8, got %.2f", confidence)
	}
}

func TestMatchChildByParentName_WithMojibake(t *testing.T) {
	description := "Vereinsbeitrag 2026 Sarah ThrÃ¤nhardt"
	children := []domain.Child{
		{
			FirstName: "Arthur",
			LastName:  "Thränhardt",
			Parents: []domain.Parent{
				{FirstName: "Sarah", LastName: "Thränhardt"},
			},
		},
	}

	child, confidence := MatchChildByParentName(description, children)
	if child == nil {
		t.Fatalf("expected to match child, got nil")
	}
	if confidence < 0.8 {
		t.Fatalf("expected confidence >= 0.8, got %.2f", confidence)
	}
}

func TestMatchChildByName_EsszettVariants(t *testing.T) {
	// Test that ß, ss, and s are all normalized equivalently
	testCases := []struct {
		name        string
		description string
		childName   string
	}{
		{"ß in DB, s in description", "Essensgeld Frida-Carlotta Quaiser", "Quaißer"},
		{"s in DB, ß in description", "Essensgeld Frida-Carlotta Quaißer", "Quaiser"},
		{"ss in DB, s in description", "Essensgeld Frida-Carlotta Quaiser", "Quaisser"},
		{"s in DB, ss in description", "Essensgeld Frida-Carlotta Quaisser", "Quaiser"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			children := []domain.Child{
				{FirstName: "Frida-Carlotta", LastName: tc.childName},
			}

			child, confidence := MatchChildByName(tc.description, children)
			if child == nil {
				t.Fatalf("expected to match child, got nil")
			}
			if confidence < 0.8 {
				t.Fatalf("expected confidence >= 0.8, got %.2f", confidence)
			}
		})
	}
}

func TestMatchChildByName_LastNameCommaFirstName(t *testing.T) {
	// Test "Nachname, Vorname" format commonly used in bank descriptions
	testCases := []struct {
		name        string
		description string
	}{
		{"comma format", "Essensgeld Mular, Emma"},
		{"normal format", "Essensgeld Emma Mular"},
		{"reverse format", "Essensgeld Mular Emma"},
	}

	children := []domain.Child{
		{FirstName: "Emma", LastName: "Mular"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			child, confidence := MatchChildByName(tc.description, children)
			if child == nil {
				t.Fatalf("expected to match child, got nil")
			}
			if confidence < 0.8 {
				t.Fatalf("expected confidence >= 0.8 for full name match, got %.2f", confidence)
			}
		})
	}
}
