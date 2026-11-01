package domain

import "testing"

func TestLooksLikeDonation(t *testing.T) {
	for text, want := range map[string]bool{
		"Spende Fam. Zeck EREF: MOB.148": true,
		"SPENDE 2025":                    true,
		"Weihnachtsspende":               true,
		"Essensgeld 11092":               false,
		"":                               false,
	} {
		tx := BankTransaction{Description: &text}
		if got := tx.LooksLikeDonation(); got != want {
			t.Errorf("%q: got %v", text, got)
		}
	}
	name := "Spender"
	if tx := (BankTransaction{PayerName: &name}); tx.LooksLikeDonation() {
		t.Error("payer name must not count")
	}
}
