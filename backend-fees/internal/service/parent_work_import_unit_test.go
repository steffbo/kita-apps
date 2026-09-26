package service

import "testing"

func TestParseParentWorkHours(t *testing.T) {
	tests := []struct {
		input string
		want  int
		valid bool
	}{
		{"1,5", 90, true}, {"1.5", 90, true}, {"2 Std", 120, true},
		{"0,25h", 15, true}, {"0", 0, false}, {"0,2", 0, false}, {"abc", 0, false},
	}
	for _, test := range tests {
		got, err := ParseParentWorkHours(test.input)
		if (err == nil) != test.valid || got != test.want {
			t.Errorf("ParseParentWorkHours(%q) = %d, %v; want %d, valid=%t",
				test.input, got, err, test.want, test.valid)
		}
	}
}

func TestParentWorkNameMatches(t *testing.T) {
	if !parentWorkNameMatches("Änne Groß", normalizeParentWorkName("anne gross")) {
		t.Fatal("name matching should fold umlauts and ß")
	}
	if !parentWorkNameMatches("Erika Muster", normalizeParentWorkName("muster erika")) {
		t.Fatal("name matching should accept surname-first values")
	}
}
