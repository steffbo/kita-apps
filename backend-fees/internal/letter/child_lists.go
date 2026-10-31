package letter

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"codeberg.org/go-pdf/fpdf"
)

// ChildList is one printed list of enrolled children, e.g. "Krippe".
type ChildList struct {
	Title string
	Rows  []ChildListRow
}

// ChildListRow is one child on a list. Missing values stay empty.
type ChildListRow struct {
	LastName, FirstName string
	BirthDate           time.Time
	Street              string // street and house number
	PostalCode, City    string
	WeeklyHours         *int
}

type listColumn struct {
	title string
	width float64
	align string
	value func(i int, row ChildListRow) string
}

// Landscape A4 with 15 mm side margins: the columns add up to 267 mm.
var listColumns = []listColumn{
	{"Nr.", 10, "R", func(i int, _ ChildListRow) string { return strconv.Itoa(i + 1) }},
	{"Name", 42, "L", func(_ int, r ChildListRow) string { return r.LastName }},
	{"Vorname", 42, "L", func(_ int, r ChildListRow) string { return r.FirstName }},
	{"Geburtsdatum", 26, "L", func(_ int, r ChildListRow) string { return FormatDate(r.BirthDate) }},
	{"Straße", 62, "L", func(_ int, r ChildListRow) string { return r.Street }},
	{"PLZ", 16, "L", func(_ int, r ChildListRow) string { return r.PostalCode }},
	{"Ort", 33, "L", func(_ int, r ChildListRow) string { return r.City }},
	{"Std./Tag", 18, "R", func(_ int, r ChildListRow) string { return dailyHours(r.WeeklyHours) }},
	{"Std./Woche", 18, "R", func(_ int, r ChildListRow) string { return weeklyHours(r.WeeklyHours) }},
}

const (
	listLeft      = 15.0
	listWidth     = 267.0
	listTableTop  = 36.0
	listRowHeight = 6.5
	listBodyEnd   = 194.0
)

// dailyHours divides the weekly care hours over five days: 32 → "6,4".
func dailyHours(weekly *int) string {
	if weekly == nil {
		return ""
	}
	return strings.Replace(strconv.FormatFloat(float64(*weekly)/5, 'f', -1, 64), ".", ",", 1)
}

func weeklyHours(weekly *int) string {
	if weekly == nil {
		return ""
	}
	return strconv.Itoa(*weekly)
}

// RenderChildLists prints each list on its own pages (A4 landscape) with the
// association header, list title, Stichtag and page numbers.
func RenderChildLists(stichtag, created time.Time, lists []ChildList) ([]byte, error) {
	pdf := renderChildLists(stichtag, created, lists)
	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, fmt.Errorf("render child lists: %w", err)
	}
	return out.Bytes(), nil
}

func renderChildLists(stichtag, created time.Time, lists []ChildList) *fpdf.Fpdf {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(listLeft, 12, listLeft)
	pdf.SetAutoPageBreak(false, 10)
	pdf.SetCellMargin(1.5)
	pdf.SetCompression(true)
	pdf.SetTitle("Kinderlisten zum Stichtag "+FormatDate(stichtag), true)
	pdf.SetAuthor("Kita Knirpsenstadt e. V.", true)
	pdf.SetLang("de")
	pdf.AliasNbPages("")
	loadFonts(pdf)
	pdf.SetFooterFunc(func() {
		setListFont(pdf, "", 8, gray)
		pdf.SetXY(listLeft, 200)
		pdf.CellFormat(listWidth/2, 4, "Erstellt am "+FormatDate(created), "", 0, "L", false, 0, "")
		pdf.CellFormat(listWidth/2, 4, fmt.Sprintf("Seite %d von {nb}", pdf.PageNo()), "", 0, "R", false, 0, "")
	})
	for _, list := range lists {
		renderChildList(pdf, stichtag, list)
	}
	return pdf
}

func renderChildList(pdf *fpdf.Fpdf, stichtag time.Time, list ChildList) {
	y := listPage(pdf, stichtag, list)
	if len(list.Rows) == 0 {
		setListFont(pdf, "", 10, gray)
		pdf.SetXY(listLeft, y)
		pdf.CellFormat(listWidth, listRowHeight, "Keine Kinder.", "", 0, "L", false, 0, "")
		return
	}
	for i, row := range list.Rows {
		if y+listRowHeight > listBodyEnd {
			y = listPage(pdf, stichtag, list)
		}
		if i%2 == 1 {
			pdf.SetFillColor(244, 244, 240)
			pdf.Rect(listLeft, y, listWidth, listRowHeight, "F")
		}
		setListFont(pdf, "", 10, ink)
		x := listLeft
		for _, col := range listColumns {
			pdf.SetXY(x, y)
			pdf.CellFormat(col.width, listRowHeight, fitText(pdf, col.value(i, row), col.width-3),
				"", 0, col.align, false, 0, "")
			x += col.width
		}
		y += listRowHeight
	}
	pdf.SetDrawColor(hairline[0], hairline[1], hairline[2])
	pdf.SetLineWidth(.4 * pt)
	pdf.Line(listLeft, y, listLeft+listWidth, y)
}

// listPage starts a page with header and column titles and returns the first row position.
func listPage(pdf *fpdf.Fpdf, stichtag time.Time, list ChildList) float64 {
	pdf.AddPage()
	setListFont(pdf, "B", 13, ink)
	pdf.Text(listLeft+1.5, 17, association)
	setListFont(pdf, "", 9, gray)
	pdf.Text(listLeft+1.5, 22, "Elternverein, gemeinnützig anerkannt")
	pdf.Text(listLeft+1.5, 26.5, "Ahornallee 27 · 16341 Panketal")

	setListFont(pdf, "B", 20, ink)
	pdf.SetXY(listLeft, 10)
	pdf.CellFormat(listWidth, 9, list.Title, "", 0, "R", false, 0, "")
	setListFont(pdf, "", 10, muted)
	pdf.SetXY(listLeft, 20)
	children := "Kinder"
	if len(list.Rows) == 1 {
		children = "Kind"
	}
	pdf.CellFormat(listWidth, 5, fmt.Sprintf("Stichtag %s · %d %s", FormatDate(stichtag), len(list.Rows), children),
		"", 0, "R", false, 0, "")

	pdf.SetDrawColor(ink[0], ink[1], ink[2])
	pdf.SetLineWidth(.6 * pt)
	pdf.Line(listLeft, 30, listLeft+listWidth, 30)

	setListFont(pdf, "B", 9, muted)
	x := listLeft
	for _, col := range listColumns {
		pdf.SetXY(x, listTableTop)
		pdf.CellFormat(col.width, 6, col.title, "", 0, col.align, false, 0, "")
		x += col.width
	}
	pdf.SetDrawColor(hairline[0], hairline[1], hairline[2])
	pdf.SetLineWidth(.4 * pt)
	pdf.Line(listLeft, listTableTop+6, listLeft+listWidth, listTableTop+6)
	return listTableTop + 6.5
}

func setListFont(pdf *fpdf.Fpdf, style string, size float64, color [3]int) {
	pdf.SetFont("Liberation", style, size)
	pdf.SetTextColor(color[0], color[1], color[2])
}

// fitText shortens text with an ellipsis so it fits the given width.
func fitText(pdf *fpdf.Fpdf, text string, w float64) string {
	if pdf.GetStringWidth(text) <= w {
		return text
	}
	runes := []rune(text)
	for len(runes) > 0 && pdf.GetStringWidth(string(runes)+"…") > w {
		runes = runes[:len(runes)-1]
	}
	return strings.TrimSpace(string(runes)) + "…"
}
