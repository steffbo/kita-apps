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

// buildFamilyReminderEmail builds a parent-facing email for a single family
// (Food/Childcare scope). deadlineOverride sets a custom payment deadline;
// if nil, defaults to 7 days after runDate.
func buildFamilyReminderEmail(
	stage ReminderStage,
	runDate time.Time,
	parentFirstNames []string,
	items []reminderItem,
	deadlineOverride *time.Time,
	paymentSettings ReminderPaymentSettings,
) (string, string) {
	monthName := germanMonthName(int(runDate.Month()))
	year := runDate.Year()
	isFinal := stage == ReminderStageFinal

	var subject string
	if isFinal {
		subject = fmt.Sprintf("Kita Mahnung %s %d", monthName, year)
	} else {
		subject = fmt.Sprintf("Kita Zahlungserinnerung %s %d", monthName, year)
	}

	greeting := "Hallo"
	if len(parentFirstNames) > 0 {
		greeting = "Hallo " + strings.Join(parentFirstNames, " und ")
	}

	// Deadline: use override if provided, otherwise 7 days after the run date
	var dl time.Time
	if deadlineOverride != nil {
		dl = *deadlineOverride
	} else {
		dl = defaultReminderDeadline(runDate)
	}
	deadlineStr := dl.Format("02.01.2006")

	var builder strings.Builder
	builder.WriteString(greeting + ",\n\n")

	if len(items) == 1 {
		item := items[0]
		memberHint := ""
		if item.MemberNumber != "" {
			memberHint = fmt.Sprintf(" (Mitgliedsnr. %s)", item.MemberNumber)
		}
		if isFinal {
			builder.WriteString(fmt.Sprintf("für %s%s ist folgender offener Beitrag vermerkt:\n\n", item.ChildName, memberHint))
		} else {
			builder.WriteString(fmt.Sprintf("für %s%s ist folgender Beitrag offen:\n\n", item.ChildName, memberHint))
		}
		builder.WriteString(reminderLine(item, false) + "\n")
		if isFinal {
			builder.WriteString(fmt.Sprintf("\nBitte überweist den Betrag spätestens bis zum %s auf folgendes Konto:\n\n", deadlineStr))
		} else {
			builder.WriteString(fmt.Sprintf("\nBitte überweist den Betrag bis zum %s auf folgendes Konto:\n\n", deadlineStr))
		}
	} else {
		if isFinal {
			builder.WriteString("für eure Familie sind folgende offene Beiträge vermerkt:\n\n")
		} else {
			builder.WriteString("für eure Familie sind folgende Beiträge offen:\n\n")
		}
		for _, item := range items {
			builder.WriteString("- " + reminderLine(item, true) + "\n")
		}
		if isFinal {
			builder.WriteString(fmt.Sprintf("\nBitte überweist die offenen Beiträge spätestens bis zum %s auf folgendes Konto:\n\n", deadlineStr))
		} else {
			builder.WriteString(fmt.Sprintf("\nBitte überweist die offenen Beiträge bis zum %s auf folgendes Konto:\n\n", deadlineStr))
		}
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
		builder.WriteString(fmt.Sprintf("Falls die Zahlung bis zum %s nicht eingegangen ist, wird leider automatisch eine Mahngebühr fällig.\n\n", deadlineStr))
	}
	builder.WriteString("Vielen Dank!\n\n")
	builder.WriteString("Freundliche Grüße\n")
	builder.WriteString("Knirpsenstadt Beitrag\n\n")
	builder.WriteString("---\n")
	builder.WriteString("Diese E-Mail wurde automatisch erstellt. Fehler sind nicht ausgeschlossen — bei Fragen wendet euch gerne direkt an uns.\n")

	return subject, builder.String()
}

// buildFamilyMembershipReminderEmail builds a parent-facing email for a single
// family (Membership scope). deadlineOverride sets a custom payment deadline;
// if nil, defaults to 7 days after runDate.
func buildFamilyMembershipReminderEmail(
	stage ReminderStage,
	runDate time.Time,
	parentFirstNames []string,
	items []reminderItem,
	deadlineOverride *time.Time,
	paymentSettings ReminderPaymentSettings,
) (string, string) {
	year := runDate.Year()
	isFinal := stage == ReminderStageFinal

	subject := fmt.Sprintf("Kita Zahlungserinnerung Vereinsbeitrag %d", year)
	if isFinal {
		subject = fmt.Sprintf("Kita Mahnung Vereinsbeitrag %d", year)
	}

	greeting := "Hallo"
	if len(parentFirstNames) > 0 {
		greeting = "Hallo " + strings.Join(parentFirstNames, " und ")
	}

	var dl time.Time
	if deadlineOverride != nil {
		dl = *deadlineOverride
	} else {
		dl = defaultReminderDeadline(runDate)
	}
	deadlineStr := dl.Format("02.01.2006")

	var builder strings.Builder
	builder.WriteString(greeting + ",\n\n")

	if len(items) == 1 {
		if isFinal {
			builder.WriteString("für eure Familie ist folgender offener Vereinsbeitrag vermerkt:\n\n")
		} else {
			builder.WriteString("für eure Familie ist folgender Vereinsbeitrag offen:\n\n")
		}
		builder.WriteString(membershipReminderLine(items[0]) + "\n")
		if isFinal {
			builder.WriteString(fmt.Sprintf("\nBitte überweist den Betrag spätestens bis zum %s auf folgendes Konto:\n\n", deadlineStr))
		} else {
			builder.WriteString(fmt.Sprintf("\nBitte überweist den Betrag bis zum %s auf folgendes Konto:\n\n", deadlineStr))
		}
	} else {
		if isFinal {
			builder.WriteString("für eure Familie sind folgende offene Vereinsbeiträge vermerkt:\n\n")
		} else {
			builder.WriteString("für eure Familie sind folgende Vereinsbeiträge offen:\n\n")
		}
		for _, item := range items {
			builder.WriteString("- " + membershipReminderLine(item) + "\n")
		}
		if isFinal {
			builder.WriteString(fmt.Sprintf("\nBitte überweist die offenen Vereinsbeiträge spätestens bis zum %s auf folgendes Konto:\n\n", deadlineStr))
		} else {
			builder.WriteString(fmt.Sprintf("\nBitte überweist die offenen Vereinsbeiträge bis zum %s auf folgendes Konto:\n\n", deadlineStr))
		}
	}

	effectivePaymentSettings := applyLegacyReminderPaymentDefaults(paymentSettings)
	builder.WriteString(fmt.Sprintf("Empfänger: %s\n", effectivePaymentSettings.RecipientName))
	builder.WriteString(fmt.Sprintf("IBAN: %s\n", formatIBANForEmail(effectivePaymentSettings.IBAN)))
	if effectivePaymentSettings.BIC != "" {
		builder.WriteString(fmt.Sprintf("BIC: %s\n", effectivePaymentSettings.BIC))
	}
	builder.WriteString("\n")
	if isFinal {
		builder.WriteString(fmt.Sprintf("Dies ist eine Mahnung. Bitte begleicht die offenen Vereinsbeiträge spätestens bis zum %s.\n\n", deadlineStr))
		builder.WriteString("Falls ihr die Zahlung bereits veranlasst habt, betrachtet diese Nachricht bitte als gegenstandslos.\n\n")
	} else {
		builder.WriteString("Falls ihr die Zahlung bereits veranlasst habt, betrachtet diese Nachricht bitte als gegenstandslos.\n\n")
	}
	builder.WriteString("Vielen Dank!\n\n")
	builder.WriteString("Freundliche Grüße\n")
	builder.WriteString("Knirpsenstadt Beitrag\n\n")
	builder.WriteString("---\n")
	builder.WriteString("Diese E-Mail wurde automatisch erstellt. Fehler sind nicht ausgeschlossen — bei Fragen wendet euch gerne direkt an uns.\n")

	return subject, builder.String()
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

func reminderLine(item reminderItem, includeChild bool) string {
	memberHint := ""
	if item.MemberNumber != "" {
		memberHint = fmt.Sprintf(" (Mitgliedsnr. %s)", item.MemberNumber)
	}
	prefix := ""
	if includeChild {
		prefix = fmt.Sprintf("%s%s: ", item.ChildName, memberHint)
	}

	label := feeTypeLabel(item.FeeType)
	switch item.FeeType {
	case domain.FeeTypeReminder:
		if item.BaseFeeType != nil && item.BaseMonth > 0 {
			return fmt.Sprintf("%sMahngebühr für %s %s/%d — %s",
				prefix,
				feeTypeLabel(*item.BaseFeeType),
				germanMonthName(item.BaseMonth),
				item.BaseYear,
				formatCurrencyEUR(item.Amount),
			)
		}
		if item.BaseFeeType != nil {
			return fmt.Sprintf("%sMahngebühr für %s — %s",
				prefix,
				feeTypeLabel(*item.BaseFeeType),
				formatCurrencyEUR(item.Amount),
			)
		}
		return fmt.Sprintf("%s%s — %s",
			prefix,
			label,
			formatCurrencyEUR(item.Amount),
		)
	default:
		if item.Month > 0 {
			return fmt.Sprintf("%s%s %s/%d — %s",
				prefix,
				label,
				germanMonthName(item.Month),
				item.Year,
				formatCurrencyEUR(item.Amount),
			)
		}
		if item.Year > 0 {
			return fmt.Sprintf("%s%s %d — %s",
				prefix,
				label,
				item.Year,
				formatCurrencyEUR(item.Amount),
			)
		}
		return fmt.Sprintf("%s%s — %s",
			prefix,
			label,
			formatCurrencyEUR(item.Amount),
		)
	}
}

func membershipReminderLine(item reminderItem) string {
	switch item.FeeType {
	case domain.FeeTypeReminder:
		if item.BaseFeeType != nil {
			if item.BaseYear > 0 {
				return fmt.Sprintf("Mahngebühr für %s %d — %s", feeTypeLabel(*item.BaseFeeType), item.BaseYear, formatCurrencyEUR(item.Amount))
			}
			return fmt.Sprintf("Mahngebühr für %s — %s", feeTypeLabel(*item.BaseFeeType), formatCurrencyEUR(item.Amount))
		}
		return fmt.Sprintf("Mahngebühr — %s", formatCurrencyEUR(item.Amount))
	default:
		if item.Year > 0 {
			return fmt.Sprintf("%s %d — %s", feeTypeLabel(item.FeeType), item.Year, formatCurrencyEUR(item.Amount))
		}
		return fmt.Sprintf("%s — %s", feeTypeLabel(item.FeeType), formatCurrencyEUR(item.Amount))
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
		builder.WriteString(fmt.Sprintf("Falls die Zahlung bis zum %s nicht eingeht, werden für die offenen Beiträge Mahngebühren fällig.\n\n", deadlineStr))
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
