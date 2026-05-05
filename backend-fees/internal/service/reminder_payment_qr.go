package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/skip2/go-qrcode"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/repository"
)

const (
	settingReminderAutoEnabled      = "reminder_auto_enabled"
	settingReminderPaymentRecipient = "reminder_payment_recipient_name"
	settingReminderPaymentIBAN      = "reminder_payment_iban"
	settingReminderPaymentBIC       = "reminder_payment_bic"
	sepaCreditTransferIdentifier    = "SCT"
	defaultQRCodeImageSize          = 320
	maxSEPAReferenceLengthInRunes   = 140
	defaultPaymentRecipientName     = "Knirpsenstadt e.V."
	defaultPaymentIBAN              = "DE33370205000003321400"
	defaultPaymentBIC               = "BFSWDE33XXX"
)

// ReminderPaymentSettings stores payment target data used for reminder QR generation.
type ReminderPaymentSettings struct {
	RecipientName string
	IBAN          string
	BIC           string
}

type reminderQRCodeData struct {
	Payload string
	PNG     []byte
	DataURL string
}

func (s *ReminderService) GetPaymentSettings(ctx context.Context) (ReminderPaymentSettings, error) {
	if s.settingsRepo == nil {
		return ReminderPaymentSettings{}, nil
	}

	recipient, err := readAppSetting(ctx, s.settingsRepo, settingReminderPaymentRecipient)
	if err != nil {
		return ReminderPaymentSettings{}, err
	}
	iban, err := readAppSetting(ctx, s.settingsRepo, settingReminderPaymentIBAN)
	if err != nil {
		return ReminderPaymentSettings{}, err
	}
	bic, err := readAppSetting(ctx, s.settingsRepo, settingReminderPaymentBIC)
	if err != nil {
		return ReminderPaymentSettings{}, err
	}

	return applyLegacyReminderPaymentDefaults(ReminderPaymentSettings{
		RecipientName: recipient,
		IBAN:          iban,
		BIC:           bic,
	}), nil
}

func (s *ReminderService) SetPaymentSettings(ctx context.Context, settings ReminderPaymentSettings) error {
	if s.settingsRepo == nil {
		return ErrInvalidInput
	}

	normalized := normalizeReminderPaymentSettings(settings)
	entries := []domain.AppSetting{
		{Key: settingReminderPaymentRecipient, Value: normalized.RecipientName},
		{Key: settingReminderPaymentIBAN, Value: normalized.IBAN},
		{Key: settingReminderPaymentBIC, Value: normalized.BIC},
	}

	for _, entry := range entries {
		copy := entry
		if err := s.settingsRepo.Upsert(ctx, &copy); err != nil {
			return err
		}
	}

	return nil
}

func (s *ReminderService) buildReminderQRCode(
	paymentSettings ReminderPaymentSettings,
	runDate time.Time,
	householdName string,
	items []reminderItem,
) (*reminderQRCodeData, error) {
	settings := normalizeReminderPaymentSettings(paymentSettings)
	settings = applyLegacyReminderPaymentDefaults(settings)
	if !settings.HasRequiredValues() {
		return nil, nil
	}

	totalAmount := sumReminderItems(items)
	if totalAmount <= 0 {
		return nil, errors.New("non-positive total amount")
	}

	reference := buildSEPAReference(runDate, householdName, items)
	payload, err := buildSEPAPayload(settings, totalAmount, reference)
	if err != nil {
		return nil, err
	}

	png, err := qrcode.Encode(payload, qrcode.Medium, defaultQRCodeImageSize)
	if err != nil {
		return nil, err
	}

	return &reminderQRCodeData{
		Payload: payload,
		PNG:     png,
		DataURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
	}, nil
}

func readAppSetting(ctx context.Context, repo repository.SettingsRepository, key string) (string, error) {
	entry, err := repo.Get(ctx, key)
	if err != nil {
		if err == repository.ErrNotFound {
			return "", nil
		}
		return "", err
	}
	return entry.Value, nil
}

func normalizeReminderPaymentSettings(settings ReminderPaymentSettings) ReminderPaymentSettings {
	normalized := ReminderPaymentSettings{
		RecipientName: sanitizeSEPAText(settings.RecipientName),
		IBAN:          normalizeIBAN(settings.IBAN),
		BIC:           normalizeBIC(settings.BIC),
	}
	return normalized
}

func applyLegacyReminderPaymentDefaults(settings ReminderPaymentSettings) ReminderPaymentSettings {
	normalized := normalizeReminderPaymentSettings(settings)
	if normalized.RecipientName == "" {
		normalized.RecipientName = defaultPaymentRecipientName
	}
	if normalized.IBAN == "" {
		normalized.IBAN = defaultPaymentIBAN
	}
	if normalized.BIC == "" {
		if strings.TrimSpace(settings.RecipientName) == "" &&
			strings.TrimSpace(settings.IBAN) == "" &&
			strings.TrimSpace(settings.BIC) == "" {
			normalized.BIC = defaultPaymentBIC
		}
	}
	return normalized
}

func (s ReminderPaymentSettings) HasRequiredValues() bool {
	return strings.TrimSpace(s.RecipientName) != "" && strings.TrimSpace(s.IBAN) != ""
}

func normalizeIBAN(iban string) string {
	upper := strings.ToUpper(strings.TrimSpace(iban))
	upper = strings.ReplaceAll(upper, " ", "")
	return upper
}

func normalizeBIC(bic string) string {
	return strings.ToUpper(strings.TrimSpace(bic))
}

func buildSEPAPayload(settings ReminderPaymentSettings, amount float64, reference string) (string, error) {
	if !isLikelyIBAN(settings.IBAN) {
		return "", fmt.Errorf("invalid IBAN format")
	}
	if settings.BIC != "" && !isLikelyBIC(settings.BIC) {
		return "", fmt.Errorf("invalid BIC format")
	}
	if amount <= 0 {
		return "", fmt.Errorf("amount must be positive")
	}

	recipient := sanitizeSEPAText(settings.RecipientName)
	if recipient == "" {
		return "", fmt.Errorf("recipient name is required")
	}

	amountString := fmt.Sprintf("EUR%.2f", amount)
	cleanReference := sanitizeSEPAText(reference)
	cleanReference = truncateRunes(cleanReference, maxSEPAReferenceLengthInRunes)

	lines := []string{
		"BCD",
		"001",
		"1",
		sepaCreditTransferIdentifier,
		settings.BIC,
		recipient,
		settings.IBAN,
		amountString,
		"",
		cleanReference,
		"",
	}

	return strings.Join(lines, "\n"), nil
}

func buildSEPAReference(runDate time.Time, householdName string, items []reminderItem) string {
	purpose := reminderReferencePurpose(items)
	period := reminderReferencePeriod(runDate, purpose)
	family := sanitizeSEPAText(householdName)
	memberNumbers := uniqueMemberNumbers(items)

	parts := []string{purpose, period, family}
	if len(memberNumbers) > 0 {
		parts = append(parts, strings.Join(memberNumbers, "+"))
	}

	reference := sanitizeSEPAText(strings.Join(parts, " "))
	return truncateRunes(reference, maxSEPAReferenceLengthInRunes)
}

func reminderReferencePeriod(runDate time.Time, purpose string) string {
	if purpose == "Vereinsbeitrag" {
		return fmt.Sprintf("%d", runDate.Year())
	}
	return fmt.Sprintf("%s %d", germanMonthName(int(runDate.Month())), runDate.Year())
}

func reminderReferencePurpose(items []reminderItem) string {
	types := make(map[domain.FeeType]struct{})
	for _, item := range items {
		resolved := item.FeeType
		if item.FeeType == domain.FeeTypeReminder && item.BaseFeeType != nil {
			resolved = *item.BaseFeeType
		}
		types[resolved] = struct{}{}
	}

	_, hasFood := types[domain.FeeTypeFood]
	_, hasChildcare := types[domain.FeeTypeChildcare]
	_, hasMembership := types[domain.FeeTypeMembership]

	switch {
	case hasFood && hasChildcare:
		return "Essens- und Platzbeitrag"
	case hasFood:
		return "Essensbeitrag"
	case hasChildcare:
		return "Platzbeitrag"
	case hasMembership:
		return "Vereinsbeitrag"
	default:
		return "Kita Beitrag"
	}
}

func uniqueMemberNumbers(items []reminderItem) []string {
	seen := make(map[string]struct{})
	values := make([]string, 0, len(items))
	for _, item := range items {
		number := strings.TrimSpace(item.MemberNumber)
		if number == "" {
			continue
		}
		if _, ok := seen[number]; ok {
			continue
		}
		seen[number] = struct{}{}
		values = append(values, number)
	}
	sort.Strings(values)
	return values
}

func sumReminderItems(items []reminderItem) float64 {
	total := 0.0
	for _, item := range items {
		total += item.Amount
	}
	return total
}

func sanitizeSEPAText(value string) string {
	replaced := strings.ReplaceAll(value, "\r", " ")
	replaced = strings.ReplaceAll(replaced, "\n", " ")
	replaced = strings.ReplaceAll(replaced, "\t", " ")
	return strings.Join(strings.Fields(strings.TrimSpace(replaced)), " ")
}

func truncateRunes(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(value) <= maxRunes {
		return value
	}

	runes := []rune(value)
	return string(runes[:maxRunes])
}

func isLikelyIBAN(value string) bool {
	if len(value) < 15 || len(value) > 34 {
		return false
	}
	for i, r := range value {
		if i < 2 {
			if r < 'A' || r > 'Z' {
				return false
			}
			continue
		}
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func isLikelyBIC(value string) bool {
	if len(value) != 8 && len(value) != 11 {
		return false
	}
	for _, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func formatIBANForEmail(iban string) string {
	clean := normalizeIBAN(iban)
	if clean == "" {
		return ""
	}

	var chunks []string
	for len(clean) > 4 {
		chunks = append(chunks, clean[:4])
		clean = clean[4:]
	}
	if clean != "" {
		chunks = append(chunks, clean)
	}
	return strings.Join(chunks, " ")
}
