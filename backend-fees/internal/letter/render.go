package letter

import (
	"bytes"
	"embed"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"codeberg.org/go-pdf/fpdf"
)

//go:embed fonts/*.ttf knirpsenstadt-logo.png
var assets embed.FS

const (
	pt          = 25.4 / 72
	left        = 15.0
	width       = 180.0
	bottom      = 287.0
	association = "Kita Knirpsenstadt e.\u202fV."
)

var (
	ink      = [3]int{20, 20, 15}
	gray     = [3]int{85, 85, 77}
	muted    = [3]int{51, 51, 44}
	hairline = [3]int{154, 154, 144}
	pale     = [3]int{220, 220, 212}
	noteGray = [3]int{122, 122, 112}
)

// Fixed wording is copied from EinstufungPDF.vue; dimensions follow EinstufungPDF.css.
const legalBasis = "Nach § 17 des Kindertagesstättengesetzes haben die Erziehungsberechtigten Beiträge " +
	"zur Inanspruchnahme eines Platzes in der Kindertagesstätte zu entrichten. Dieser monatliche " +
	"Elternbeitrag wird in Verbindung mit der Elternbeitragsordnung des Trägers ermittelt. " +
	"Die Kindertagesstätte „Knirpsenstadt“ in 16341 Panketal, Ahornallee 27 befindet sich in " +
	"freier Trägerschaft des „Knirpsenstadt e.\u202fV. Panketal“."
const calculation = "Berechnet wird nach wirtschaftlicher Leistungsfähigkeit (Nettoeinkommen im Jahr), " +
	"dem Alter des Kindes und der beanspruchten Betreuungszeit. Eine Ermäßigung des Elternbeitrages " +
	"wird auch nach der Anzahl der unterhaltspflichtigen Kinder gewährt " +
	"(jedoch nicht nach dem Brandenburger Entlastungspaket)."
const changeDuty = "Wenn sich das Nettoeinkommen im laufenden Jahr gegenüber dem Vorjahr (bzw. bei " +
	"Selbständigen gegenüber der letzten Festsetzung) um mehr als 10 % verändert, ist dies unter " +
	"Vorlage entsprechender Nachweise unverzüglich anzuzeigen."

// RenderEinstufung returns the PDF bytes of the letter.
func RenderEinstufung(l EinstufungLetter) ([]byte, error) {
	pdf := renderEinstufung(l)
	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, fmt.Errorf("render Einstufung letter: %w", err)
	}
	return out.Bytes(), nil
}

type renderer struct {
	pdf *fpdf.Fpdf
	y   float64
}

type span struct {
	text, style string
	size        float64
	color       [3]int
}

func regular(text string, size float64, color [3]int) span {
	return span{text: text, size: size, color: color}
}
func bold(text string, size float64, color [3]int) span {
	return span{text: text, style: "B", size: size, color: color}
}
func (r *renderer) font(s span) {
	r.pdf.SetFont("Liberation", s.style, s.size)
	r.pdf.SetTextColor(s.color[0], s.color[1], s.color[2])
}

// wrap retains inline styles and splits exceptionally long words without clipping.
func (r *renderer) wrap(w float64, spans ...span) [][]span {
	lines := [][]span{{}}
	used := 0.0
	for _, s := range spans {
		r.font(s)
		words := strings.Split(s.text, " ")
		for i, word := range words {
			if i > 0 {
				word = " " + word
			}
			for _, ch := range word {
				piece := string(ch)
				cw := r.pdf.GetStringWidth(piece)
				// Prefer word boundaries, while retaining a character fallback for long tokens.
				if ch == ' ' && used+r.pdf.GetStringWidth(word) > w {
					lines = append(lines, []span{})
					used = 0
					continue
				}
				if used+cw > w && used > 0 {
					lines = append(lines, []span{})
					used = 0
				}
				idx := len(lines) - 1
				last := len(lines[idx]) - 1
				if last >= 0 && lines[idx][last].style == s.style &&
					lines[idx][last].size == s.size && lines[idx][last].color == s.color {
					lines[idx][last].text += piece
				} else {
					part := s
					part.text = piece
					lines[idx] = append(lines[idx], part)
				}
				used += cw
			}
		}
	}
	return lines
}

// maxJustifyGap is the largest extra space (mm) added per word gap when justifying.
const maxJustifyGap = 1.2

func (r *renderer) draw(x, y, w, lh float64, align string, lines [][]span) {
	for _, line := range lines {
		lineWidth := 0.0
		for _, s := range line {
			r.font(s)
			lineWidth += r.pdf.GetStringWidth(s.text)
		}
		xp := x
		if align == "R" {
			xp += w - lineWidth
		}
		spaces := 0
		for _, s := range line {
			spaces += strings.Count(s.text, " ")
		}
		extra := 0.0
		if align == "J" && spaces > 0 {
			extra = math.Max(0, (w-lineWidth)/float64(spaces))
			// Without hyphenation, short lines before a long word would get
			// wide gaps; leave those ragged instead.
			if extra > maxJustifyGap {
				extra = 0
			}
		}
		for _, s := range line {
			r.font(s)
			if extra == 0 {
				r.pdf.Text(xp, y+lh*.77, s.text)
				xp += r.pdf.GetStringWidth(s.text)
				continue
			}
			for i, word := range strings.Split(s.text, " ") {
				if i > 0 {
					xp += r.pdf.GetStringWidth(" ") + extra
				}
				r.pdf.Text(xp, y+lh*.77, word)
				xp += r.pdf.GetStringWidth(word)
			}
		}
		y += lh
	}
}

// tracked approximates CSS letter spacing by stretching the text horizontally.
// Keeping a single text operation preserves coherent words in PDF text extraction.
func (r *renderer) tracked(x, y, lh, spacing float64, s span) {
	r.font(s)
	w := r.pdf.GetStringWidth(s.text)
	if w == 0 {
		return
	}
	scale := 100 * (w + float64(utf8.RuneCountInString(s.text)-1)*spacing*pt) / w
	r.pdf.TransformBegin()
	r.pdf.TransformScale(scale, 100, x, y+lh*.77)
	r.pdf.Text(x, y+lh*.77, s.text)
	r.pdf.TransformEnd()
}

func (r *renderer) reserve(h float64) {
	if r.y+h > bottom && r.y > 13 {
		r.pdf.AddPage()
		r.y = 13
	}
}
func (r *renderer) text(w, lh float64, s span) {
	for _, line := range r.wrap(w, s) {
		r.reserve(lh)
		r.draw(left, r.y, w, lh, "L", [][]span{line})
		r.y += lh
	}
}

// paragraph approximates CSS justification without automatic language-dependent hyphenation.
func (r *renderer) paragraph(x, y, w, lh float64, lines [][]span) {
	for i, line := range lines {
		align := "J"
		if i == len(lines)-1 {
			align = "L"
		}
		r.draw(x, y+float64(i)*lh, w, lh, align, [][]span{line})
	}
}
func (r *renderer) body(text string) {
	lh := 9 * pt * 1.4
	lines := r.wrap(width, regular(text, 9, ink))
	r.reserve(float64(len(lines)) * lh)
	r.paragraph(left, r.y, width, lh, lines)
	r.y += float64(len(lines)) * lh
}

func (r *renderer) rule(x, y, w, thickness float64, color [3]int) {
	r.pdf.SetDrawColor(color[0], color[1], color[2])
	r.pdf.SetLineWidth(thickness * pt)
	r.pdf.Line(x, y, x+w, y)
}
func (r *renderer) heading(text string) {
	r.y += 3.2
	r.reserve(12)
	lh := 8.4 * pt * 1.36
	r.tracked(left, r.y, lh, 1, bold(strings.ToUpper(text), 8.4, ink))
	r.y += lh
	r.y += .8
	r.rule(left, r.y, width, .4, hairline)
	r.y += 1.6
}
func (r *renderer) callout(quiet bool, spans ...span) {
	lh := 9 * pt * 1.4
	lines := r.wrap(width-6, spans...)
	h := float64(len(lines))*lh + 3.6
	r.y += 3
	r.reserve(h)
	color := ink
	if quiet {
		color = hairline
	}
	r.pdf.SetDrawColor(color[0], color[1], color[2])
	r.pdf.SetLineWidth(.6 * pt)
	r.pdf.Rect(left, r.y, width, h, "D")
	r.draw(left+3, r.y+1.8, width-6, lh, "L", lines)
	r.y += h
}

// loadFonts registers the embedded Liberation Serif as "Liberation".
func loadFonts(pdf *fpdf.Fpdf) {
	for _, font := range []struct{ file, style string }{
		{"Regular", ""}, {"Bold", "B"}, {"Italic", "I"},
	} {
		data, err := assets.ReadFile("fonts/LiberationSerif-" + font.file + ".ttf")
		if err != nil {
			pdf.SetError(err)
			return
		}
		pdf.AddUTF8FontFromBytes("Liberation", font.style, data)
	}
}

func renderEinstufung(l EinstufungLetter) *fpdf.Fpdf {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(left, 13, 15)
	pdf.SetAutoPageBreak(false, 10)
	pdf.SetCellMargin(0)
	pdf.SetCompression(true)
	pdf.SetTitle(fmt.Sprintf("Einstufung %d %s", l.Year, l.ChildName), true)
	pdf.SetAuthor("Kita Knirpsenstadt e. V.", true)
	pdf.SetLang("de")
	loadFonts(pdf)
	if pdf.Error() != nil {
		return pdf
	}
	logo, err := assets.ReadFile("knirpsenstadt-logo.png")
	if err != nil {
		pdf.SetError(err)
		return pdf
	}
	info := pdf.RegisterImageOptionsReader("logo", fpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(logo))
	if pdf.Error() != nil {
		return pdf
	}
	pdf.AddPage()
	r := renderer{pdf: pdf, y: 13}
	logoH := 18 * info.Height() / info.Width()
	headH := math.Max(logoH, 13*pt*1.1+1.2+7.6*pt*1.36+1.5)
	pdf.ImageOptions("logo", left+width-18, r.y+headH-logoH, 18, logoH, false,
		fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	r.y += headH - 1.5 - 7.6*pt*1.36 - 1.2 - 13*pt*1.1
	r.tracked(left, r.y, 13*pt*1.1, .5, regular(association, 13, ink))
	r.y += 13 * pt * 1.1
	r.y += 1.2
	r.tracked(left, r.y, 7.6*pt*1.36, 1.5, regular("ELTERNVEREIN · DER VORSTAND", 7.6, gray))
	r.y += 7.6 * pt * 1.36
	r.y = 13 + headH
	r.rule(left, r.y, width, 1, ink)
	r.y += .9
	r.rule(left, r.y, width, .4, ink)
	r.y += 3
	lh := 8.4 * pt * 1.36
	r.draw(left, r.y, 80, lh, "L", r.wrap(80, regular("Mitgliedsnummer "+l.MemberNumber, 8.4, muted)))
	r.draw(left+80, r.y, 100, lh, "R",
		r.wrap(100, regular("Panketal, den "+FormatDateLong(l.LetterDate), 8.4, muted)))
	r.y += lh + 3
	subject := fmt.Sprintf("Einstufung der Elternbeiträge %d", l.Year)
	if l.Provisional {
		subject = "Vorläufige " + subject
	}
	r.text(width, 12*pt*1.25, bold(subject, 12, ink))
	r.y += .8
	r.text(width, 9.5*pt*1.36,
		regular("für "+l.ChildName+" · gültig ab "+FormatMonthYear(l.ValidFrom), 9.5, muted))
	r.y += 3.2
	r.text(width, 9*pt*1.36, regular("Liebe Eltern,", 9, ink))
	r.y += 1.6
	r.body(l.Introduction)
	r.y += 1.8
	r.heading("Angaben zum Kind")
	r.childData(l)
	r.callout(false, bold("Grundlage der Einstufung: ", 9, ink), regular(l.BasisText, 9, ink))
	r.heading("Monatliche Beiträge")
	r.fees(l.Periods)
	if l.MembershipNote != "" {
		r.y += 1.4
		r.text(width, 7.9*pt*1.35, regular(l.MembershipNote, 7.9, muted))
	}
	r.callout(true, bold("Hinweis: ", 9, ink),
		regular("Bitte gleicht die Beträge für Mitgliedschaft, Betreuung und Essensgeld in ", 9, ink),
		bold("getrennten", 9, ink), regular(" Zahlungen unter Angabe des Namens des Kindes und der "+
			"Mitgliedsnummer aus. Die Beträge gelten fortlaufend ab dem jeweils genannten Monat, "+
			"bis eine neue Einstufung erfolgt.", 9, ink))
	r.fineprint(l.MembershipFee)
	r.y += 3.5
	r.reserve(9*pt*1.4 + 1.8 + 4 + .8 + 8.4*pt*1.36)
	r.text(width, 9*pt*1.4, regular("Mit freundlichen Grüßen", 9, ink))
	r.y += 1.8 + 4
	r.rule(left, r.y, 66, .3, hairline)
	r.y += .8
	r.text(width, 8.4*pt*1.36, regular("Der Vorstand der "+association, 8.4, muted))
	r.footer(l)
	return pdf
}
