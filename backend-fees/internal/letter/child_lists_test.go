package letter

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func intPtr(v int) *int { return &v }

// Synthetic children; the counts match the 01.09.2026 lists (13 Krippe, 33 Kita).
func sampleChildLists() []ChildList {
	rows := func(n int, birthYear int) []ChildListRow {
		out := make([]ChildListRow, n)
		for i := range out {
			out[i] = ChildListRow{
				LastName: fmt.Sprintf("Nachname %02d", i+1), FirstName: "Vorname",
				BirthDate: date(birthYear, 3, i%28+1), Street: "Musterstraße 1",
				PostalCode: "16341", City: "Panketal", WeeklyHours: intPtr(30 + i%4*5),
			}
		}
		return out
	}
	krippe := rows(13, 2024)
	krippe[0].Street, krippe[0].PostalCode, krippe[0].City, krippe[0].WeeklyHours = "", "", "", nil
	krippe[1].LastName = "Ein sehr langer Doppelname mit Bindestrich-Ergänzung"
	return []ChildList{{Title: "Krippe", Rows: krippe}, {Title: "Kita", Rows: rows(33, 2021)}}
}

func TestRenderChildLists(t *testing.T) {
	stichtag, created := date(2026, 9, 1), date(2026, 10, 31)
	pdf := renderChildLists(stichtag, created, sampleChildLists())
	if err := pdf.Error(); err != nil {
		t.Fatal(err)
	}
	// Krippe fits on one page, Kita needs two; each list starts on its own page.
	if got := pdf.PageNo(); got != 3 {
		t.Fatalf("pages: %d", got)
	}
	empty := renderChildLists(stichtag, created, []ChildList{{Title: "Krippe"}, {Title: "Kita"}})
	if got := empty.PageNo(); got != 2 {
		t.Fatalf("empty lists: %d pages", got)
	}
	out, err := RenderChildLists(stichtag, created, sampleChildLists())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Fatal("missing PDF header")
	}
}

func TestChildListHours(t *testing.T) {
	for _, test := range []struct {
		weekly       *int
		daily, total string
	}{
		{intPtr(45), "9", "45"}, {intPtr(32), "6,4", "32"}, {intPtr(0), "0", "0"}, {nil, "", ""},
	} {
		if got := dailyHours(test.weekly); got != test.daily {
			t.Errorf("daily %v: %q", test.weekly, got)
		}
		if got := weeklyHours(test.weekly); got != test.total {
			t.Errorf("weekly %v: %q", test.weekly, got)
		}
	}
}

func TestFitText(t *testing.T) {
	pdf := renderChildLists(date(2026, 9, 1), date(2026, 10, 31), nil)
	setListFont(pdf, "", 10, ink)
	if got := fitText(pdf, "Panketal", 30); got != "Panketal" {
		t.Fatal(got)
	}
	long := "Ein sehr langer Doppelname mit Bindestrich-Ergänzung"
	got := fitText(pdf, long, 39)
	if got == long || pdf.GetStringWidth(got) > 39 || got[len(got)-len("…"):] != "…" {
		t.Fatalf("not shortened: %q", got)
	}
}

func TestWriteSampleChildLists(t *testing.T) {
	path := os.Getenv("CHILD_LISTS_SAMPLE_OUT")
	if path == "" {
		t.Skip("CHILD_LISTS_SAMPLE_OUT is not set")
	}
	out, err := RenderChildLists(date(2026, 9, 1), date(2026, 10, 31), sampleChildLists())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		t.Fatal(err)
	}
}
