package service

import (
	"fmt"
	"html"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

// reminderItem is one line of a family reminder mail and of the QR total:
// a selected fee (remaining amount) or a planned Mahngebühr.
type reminderItem struct {
	FeeID        uuid.UUID
	ChildID      uuid.UUID
	ChildName    string
	MemberNumber string
	FeeType      domain.FeeType
	Amount       float64
	Year         int
	Month        int
	DueDate      time.Time
	// Base fee of a Mahngebühr.
	BaseFeeType   *domain.FeeType
	BaseYear      int
	BaseMonth     int
	ReminderForID *uuid.UUID
	// Club member of a membership fee (or its Mahngebühr).
	ClubMember *ReminderCaseMember
}

func collectEmails(parents []domain.Parent) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, p := range parents {
		if p.Email == nil || *p.Email == "" {
			continue
		}
		e := *p.Email
		if !seen[e] {
			seen[e] = true
			result = append(result, e)
		}
	}
	return result
}

func parentFirstNames(parents []domain.Parent) []string {
	names := make([]string, 0, len(parents))
	for _, p := range parents {
		if p.FirstName != "" {
			names = append(names, p.FirstName)
		}
	}
	return names
}

func feeTypeLabel(feeType domain.FeeType) string {
	switch feeType {
	case domain.FeeTypeFood:
		return "Essensgeld"
	case domain.FeeTypeChildcare:
		return "Platzgeld"
	case domain.FeeTypeMembership:
		return "Vereinsbeitrag"
	case domain.FeeTypeReminder:
		return "Mahngebühr"
	default:
		return string(feeType)
	}
}

func defaultReminderDeadline(runDate time.Time) time.Time {
	base := time.Date(runDate.Year(), runDate.Month(), runDate.Day(), 0, 0, 0, 0, time.UTC)
	return base.AddDate(0, 0, 7)
}

func germanMonthName(month int) string {
	switch month {
	case 1:
		return "Januar"
	case 2:
		return "Februar"
	case 3:
		return "Maerz"
	case 4:
		return "April"
	case 5:
		return "Mai"
	case 6:
		return "Juni"
	case 7:
		return "Juli"
	case 8:
		return "August"
	case 9:
		return "September"
	case 10:
		return "Oktober"
	case 11:
		return "November"
	case 12:
		return "Dezember"
	default:
		return ""
	}
}

func formatCurrencyEUR(amount float64) string {
	value := fmt.Sprintf("%.2f", amount)
	value = strings.ReplaceAll(value, ".", ",")
	return value + " EUR"
}

const reminderEmailQRCodeCID = "payment-qr-code"

func buildReminderEmailHTML(textBody string, qrImageCID string) string {
	var builder strings.Builder
	builder.WriteString("<!doctype html><html><body style=\"font-family:Arial,sans-serif;line-height:1.5;color:#111;\">")

	lines := strings.Split(textBody, "\n")
	for idx, line := range lines {
		if idx > 0 {
			builder.WriteString("<br>")
		}
		builder.WriteString(html.EscapeString(line))
	}

	if strings.TrimSpace(qrImageCID) != "" {
		builder.WriteString("<hr style=\"margin:24px 0;border:none;border-top:1px solid #ddd;\">")
		builder.WriteString("<p><strong>QR-Code für die Überweisung:</strong></p>")
		builder.WriteString("<img alt=\"SEPA Zahlungs-QR\" src=\"cid:")
		builder.WriteString(html.EscapeString(qrImageCID))
		builder.WriteString("\" style=\"display:block;max-width:280px;width:100%;height:auto;border:1px solid #ddd;padding:8px;\">")
	}

	builder.WriteString("</body></html>")
	return builder.String()
}

// buildFamilyMixedReminderEmail builds the unified parent-facing email for the
// family-based workflow: one mail for any combination of fee types. deadline
// is always set (server-computed runDate + 7 days).
func buildFamilyMixedReminderEmail(
	stage ReminderStage,
	runDate time.Time,
	deadline time.Time,
	parentFirstNames []string,
	items []reminderItem,
	paymentSettings ReminderPaymentSettings,
) (string, string) {
	isFinal := stage == ReminderStageFinal

	var subject string
	if isFinal {
		subject = "Kita Mahnung: offene Beiträge"
	} else {
		subject = "Kita Zahlungserinnerung: offene Beiträge"
	}

	greeting := "Hallo"
	if len(parentFirstNames) > 0 {
		greeting = "Hallo " + strings.Join(parentFirstNames, " und ")
	}

	deadlineStr := deadline.Format("02.01.2006")

	var builder strings.Builder
	builder.WriteString(greeting + ",\n\n")

	if len(items) == 1 {
		if isFinal {
			builder.WriteString("für eure Familie ist folgender offener Beitrag vermerkt:\n\n")
		} else {
			builder.WriteString("für eure Familie ist folgender Beitrag offen:\n\n")
		}
	} else {
		if isFinal {
			builder.WriteString("für eure Familie sind folgende offene Beiträge vermerkt:\n\n")
		} else {
			builder.WriteString("für eure Familie sind folgende Beiträge offen:\n\n")
		}
	}
	builder.WriteString(familyReminderItemList(items))
	builder.WriteString(fmt.Sprintf("\nGesamtbetrag: %s\n", formatCurrencyEUR(sumReminderItems(items))))

	if isFinal {
		builder.WriteString(fmt.Sprintf("\nBitte überweist den Gesamtbetrag spätestens bis zum %s auf folgendes Konto:\n\n", deadlineStr))
	} else {
		builder.WriteString(fmt.Sprintf("\nBitte überweist den Gesamtbetrag bis zum %s auf folgendes Konto:\n\n", deadlineStr))
	}

	effectivePaymentSettings := applyLegacyReminderPaymentDefaults(paymentSettings)
	builder.WriteString(fmt.Sprintf("Empfänger: %s\n", effectivePaymentSettings.RecipientName))
	builder.WriteString(fmt.Sprintf("IBAN: %s\n", formatIBANForEmail(effectivePaymentSettings.IBAN)))
	if effectivePaymentSettings.BIC != "" {
		builder.WriteString(fmt.Sprintf("BIC: %s\n", effectivePaymentSettings.BIC))
	}
	builder.WriteString("\n")
	if isFinal {
		builder.WriteString(fmt.Sprintf("Dies ist eine Mahnung. Bitte begleicht die offenen Beiträge spätestens bis zum %s.\n\n", deadlineStr))
		builder.WriteString("Falls ihr die Zahlung bereits veranlasst habt, betrachtet diese Nachricht bitte als gegenstandslos.\n\n")
	} else {
		builder.WriteString(fmt.Sprintf("Falls die Zahlung bis zum %s nicht eingeht, können für die offenen Beiträge Mahngebühren erhoben werden.\n\n", deadlineStr))
	}
	builder.WriteString("Vielen Dank!\n\n")
	builder.WriteString("Freundliche Grüße\n")
	builder.WriteString("Knirpsenstadt Beitrag\n\n")
	builder.WriteString("---\n")
	builder.WriteString("Diese E-Mail wurde automatisch erstellt. Fehler sind nicht ausgeschlossen — bei Fragen wendet euch gerne direkt an uns.\n")

	return subject, builder.String()
}

// familyReminderGroup collects the mail lines of one person: a child (food,
// childcare and their Mahngebühren) or a club member (membership fees).
type familyReminderGroup struct {
	header   string
	isMember bool
	sortName string
	items    []reminderItem
}

// familyReminderItemList renders the items grouped per child and per club
// member. Each Mahngebühr is nested under its base fee when that is listed
// too; otherwise it stands alone with the base fee named.
func familyReminderItemList(items []reminderItem) string {
	groups := make(map[string]*familyReminderGroup)
	order := make([]*familyReminderGroup, 0)
	for _, item := range items {
		key := "child:" + item.ChildID.String()
		group := &familyReminderGroup{sortName: item.ChildName}
		if item.MemberNumber != "" {
			group.header = fmt.Sprintf("%s (Mitgliedsnr. %s):", item.ChildName, item.MemberNumber)
		} else {
			group.header = item.ChildName + ":"
		}
		if isMembershipItem(item) && item.ClubMember != nil {
			key = "member:" + item.ClubMember.ID.String()
			group = &familyReminderGroup{
				header:   fmt.Sprintf("Vereinsmitglied %s:", item.ClubMember.Name),
				isMember: true,
				sortName: item.ClubMember.Name,
			}
		}
		if existing, ok := groups[key]; ok {
			group = existing
		} else {
			groups[key] = group
			order = append(order, group)
		}
		group.items = append(group.items, item)
	}

	sort.SliceStable(order, func(i, j int) bool {
		if order[i].isMember != order[j].isMember {
			return !order[i].isMember
		}
		return order[i].sortName < order[j].sortName
	})

	var builder strings.Builder
	for idx, group := range order {
		if idx > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(group.header + "\n")

		listed := make(map[uuid.UUID]bool, len(group.items))
		for _, item := range group.items {
			if item.FeeType != domain.FeeTypeReminder && item.FeeID != uuid.Nil {
				listed[item.FeeID] = true
			}
		}
		nested := make(map[uuid.UUID][]reminderItem)
		bases := make([]reminderItem, 0, len(group.items))
		for _, item := range group.items {
			if item.FeeType == domain.FeeTypeReminder && item.ReminderForID != nil && listed[*item.ReminderForID] {
				nested[*item.ReminderForID] = append(nested[*item.ReminderForID], item)
				continue
			}
			bases = append(bases, item)
		}
		sort.SliceStable(bases, func(i, j int) bool {
			yi, mi := reminderItemPeriod(bases[i])
			yj, mj := reminderItemPeriod(bases[j])
			if yi != yj {
				return yi < yj
			}
			return mi < mj
		})
		for _, item := range bases {
			builder.WriteString(fmt.Sprintf("- %s — %s\n", reminderItemLabel(item), formatCurrencyEUR(item.Amount)))
			for _, reminder := range nested[item.FeeID] {
				builder.WriteString(fmt.Sprintf("  zzgl. Mahngebühr — %s\n", formatCurrencyEUR(reminder.Amount)))
			}
		}
	}
	return builder.String()
}

func isMembershipItem(item reminderItem) bool {
	if item.FeeType == domain.FeeTypeMembership {
		return true
	}
	return item.FeeType == domain.FeeTypeReminder && item.BaseFeeType != nil && *item.BaseFeeType == domain.FeeTypeMembership
}

// reminderItemPeriod is the sort period of an item; a Mahngebühr sorts by
// its base fee.
func reminderItemPeriod(item reminderItem) (int, int) {
	if item.FeeType == domain.FeeTypeReminder && item.BaseFeeType != nil {
		return item.BaseYear, item.BaseMonth
	}
	return item.Year, item.Month
}

// reminderItemLabel names an item without person and amount.
func reminderItemLabel(item reminderItem) string {
	if item.FeeType == domain.FeeTypeReminder {
		if item.BaseFeeType != nil {
			return "Mahngebühr für " + feeRefLabel(*item.BaseFeeType, item.BaseYear, item.BaseMonth)
		}
		return feeTypeLabel(item.FeeType)
	}
	return feeRefLabel(item.FeeType, item.Year, item.Month)
}

// feeRefLabel names a fee type with its period: "Essensgeld September 2026",
// "Vereinsbeitrag 2026".
func feeRefLabel(feeType domain.FeeType, year, month int) string {
	label := feeTypeLabel(feeType)
	if month > 0 {
		return fmt.Sprintf("%s %s %d", label, germanMonthName(month), year)
	}
	if year > 0 {
		return fmt.Sprintf("%s %d", label, year)
	}
	return label
}
