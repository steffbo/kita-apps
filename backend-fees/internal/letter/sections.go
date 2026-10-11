package letter

import (
	"math"
	"strconv"
	"strings"
)

func (r *renderer) childData(l EinstufungLetter) {
	rows := [][]span{
		{regular("Name", 9, gray), bold(l.ChildName, 9, ink),
			regular("Geburtsdatum", 9, gray), bold(FormatDate(l.BirthDate), 9, ink)},
		{regular("Besucht die Kita seit", 9, gray), bold(FormatDate(l.EntryDate), 9, ink),
			regular("Mitgliedsnummer", 9, gray), bold(l.MemberNumber, 9, ink)},
		{regular("Einrichtung", 9, gray), bold(association+", Ahornallee 27, 16341 Panketal", 9, ink)},
	}
	widths := []float64{33, 54, 39, 54}
	lh := 9 * pt * 1.36
	// Keep the entire data table together, including unusually long names.
	heights := make([]float64, len(rows))
	wrapped := make([][][][]span, len(rows))
	total := 0.0
	for i, row := range rows {
		heights[i] = lh + 1.6
		for j, s := range row {
			w := widths[j]
			if len(row) == 2 && j == 1 {
				w = width - widths[0]
			}
			if j%2 == 0 {
				w -= 3
			}
			if j == 2 {
				w -= 6
			}
			lines := r.wrap(w, s)
			wrapped[i] = append(wrapped[i], lines)
			heights[i] = math.Max(heights[i], float64(len(lines))*lh+1.6)
		}
		total += heights[i]
	}
	r.reserve(total)
	for i, row := range rows {
		x := left
		for j := range row {
			offset := 0.0
			if j == 2 {
				offset = 6
			}
			r.draw(x+offset, r.y+.8, widths[j]-offset, lh, "L", wrapped[i][j])
			x += widths[j]
		}
		r.y += heights[i]
		if i < len(rows)-1 {
			r.rule(left, r.y, width, .3, pale)
		}
	}
}

var feeWidths = []float64{61.2, 28.8, 21.6, 34.2, 34.2}

func (r *renderer) feeRow(cells [][]span, header bool) {
	lh := 8.8 * pt * 1.36
	wrapped := make([][][]span, len(cells))
	h := lh + 1.8
	for i, cell := range cells {
		wrapped[i] = r.wrap(feeWidths[i]-5, cell...)
		h = math.Max(h, float64(len(wrapped[i]))*lh+1.8)
	}
	r.reserve(h)
	r.rule(left, r.y, width, .4, hairline)
	x := left
	for i := range cells {
		align := "L"
		if i >= 2 {
			align = "R"
		}
		r.draw(x+2.5, r.y+(h-float64(len(wrapped[i]))*lh)/2,
			feeWidths[i]-5, lh, align, wrapped[i])
		x += feeWidths[i]
		r.pdf.SetDrawColor(hairline[0], hairline[1], hairline[2])
		r.pdf.SetLineWidth(.4 * pt)
		r.pdf.Line(x, r.y, x, r.y+h)
	}
	r.y += h
	thickness := .4
	if header {
		thickness = .8
	}
	r.rule(left, r.y, width, thickness, hairline)
}

func (r *renderer) feeHeader() {
	cells := make([][]span, 0, 5)
	for _, label := range []string{"Zeitraum", "Bereich", "Std./Woche", "Platzgeld", "Essensgeld"} {
		cells = append(cells, []span{bold(label, 8.8, ink)})
	}
	r.feeRow(cells, true)
}

func (r *renderer) fees(periods []FeePeriod) {
	r.feeHeader()
	for _, p := range periods {
		style, color := "B", ink
		if p.Previous {
			style, color = "", gray
		}
		cells := make([][]span, 0, 5)
		for _, value := range []string{
			p.Label, p.CareType, strconv.Itoa(p.CareHours), FormatEuro(p.ChildcareFee), FormatEuro(p.FoodFee),
		} {
			cells = append(cells, []span{{text: value, style: style, size: 8.8, color: color}})
		}
		if p.Previous {
			cells[0] = append(cells[0], regular(" (bisher)", 7.6, noteGray))
		}
		h := 0.0
		for i, cell := range cells {
			h = math.Max(h, float64(len(r.wrap(feeWidths[i]-5, cell...)))*8.8*pt*1.36+1.8)
		}
		page := r.pdf.PageNo()
		r.reserve(h)
		if page != r.pdf.PageNo() {
			r.feeHeader()
		}
		r.feeRow(cells, false)
	}
}

func (r *renderer) fineprint(fee float64) {
	payment := "Der monatliche Beitrag wird am 5. eines jeden Monats fällig. Beiträge, die in Verzug " +
		"sind, werden zusätzlich mit einer Mahngebühr von 10,00 € erhoben. Der Vereinsbeitrag"
	if fee > 0 {
		payment += " (derzeit " + FormatEuro(fee) + ")"
	}
	payment += " ist jährlich zu zahlen: bei Vertragsbeginn sofort, ansonsten bis spätestens Ende des " +
		"ersten Quartals. Nach Fristablauf wird ein Mahngeld von 5,00 € erhoben."
	type paragraph struct {
		text    string
		heading bool
		gap     float64
	}
	cols := [][]paragraph{
		{{"Rechtliche Grundlage", true, 1.2}, {legalBasis, false, 1.4}, {calculation, false, 0}},
		{{"Zahlungsbedingungen", true, 1.2}, {payment, false, 2.4},
			{"Änderungspflicht", true, 1.2}, {changeDuty, false, 0}},
	}
	wrapped := make([][][][]span, 2)
	heights := make([]float64, 2)
	for i, col := range cols {
		for _, p := range col {
			s := regular(p.text, 7.9, muted)
			lh := 7.9 * pt * 1.34
			if p.heading {
				s = bold(strings.ToUpper(p.text), 7.6, ink)
				lh = 7.6 * pt * 1.34
			}
			lines := r.wrap(83, s)
			wrapped[i] = append(wrapped[i], lines)
			heights[i] += float64(len(lines))*lh + p.gap
		}
	}
	h := math.Max(heights[0], heights[1])
	r.y += 3.4
	r.reserve(h)
	for i, col := range cols {
		y := r.y
		for j, p := range col {
			lh := 7.9 * pt * 1.34
			if p.heading {
				lh = 7.6 * pt * 1.34
			}
			if p.heading {
				r.tracked(left+float64(i)*97, y, lh, .9, bold(strings.ToUpper(p.text), 7.6, ink))
			} else {
				r.paragraph(left+float64(i)*97, y, 83, lh, wrapped[i][j])
			}
			y += float64(len(wrapped[i][j]))*lh + p.gap
		}
	}
	r.pdf.SetDrawColor(pale[0], pale[1], pale[2])
	r.pdf.SetLineWidth(.3 * pt)
	r.pdf.Line(left+90, r.y, left+90, r.y+h)
	r.y += h
}

// footerLabelWidth leaves room for the widest footer label, "VORSTAND".
const footerLabelWidth = 17

func (r *renderer) footer(l EinstufungLetter) {
	lines := [][2]string{{"Verein", association + " · Ahornallee 27 · 16341 Panketal · " +
		"Vereinsregister VR 4217, Amtsgericht Frankfurt (Oder)"}}
	if len(l.Board) > 0 {
		members := make([]string, 0, len(l.Board))
		for _, b := range l.Board {
			members = append(members, b.Name+" ("+b.Role+")")
		}
		lines = append(lines, [2]string{"Vorstand", strings.Join(members, " · ")})
	}
	if l.Bank.IBAN != "" {
		lines = append(lines, [2]string{"Bank", l.Bank.Holder + " · IBAN " + l.Bank.IBAN +
			" · BIC " + l.Bank.BIC + " · " + l.Bank.Bank})
	}
	lh := 6.9 * pt * 1.4
	wrapped := make([][][]span, len(lines))
	h := 1.4 + 1.2 + 1 + 6.5*pt*1.4
	for i, line := range lines {
		wrapped[i] = r.wrap(width-footerLabelWidth, regular(line[1], 6.9, gray))
		h += float64(len(wrapped[i])) * lh
	}
	r.y += 4
	r.reserve(h)
	r.rule(left, r.y, width, .4, hairline)
	r.y += 1.4
	for i, line := range lines {
		r.tracked(left, r.y, lh, .45, bold(strings.ToUpper(line[0]), 6.4, ink))
		r.draw(left+footerLabelWidth, r.y, width-footerLabelWidth, lh, "L", wrapped[i])
		r.y += float64(len(wrapped[i])) * lh
	}
	r.y += 1.2
	r.rule(left, r.y, width, .3, pale)
	r.y += 1
	r.text(width, 6.5*pt*1.4,
		regular("Rechtlich verbindliche Aussagen für den Verein trifft allein der Vorstand.", 6.5, noteGray))
}
