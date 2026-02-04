package csvparser

import (
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

const estimatedFullNameParts = 4

// Column indices for the bank CSV format (BFS/SozialBank export)
const (
	colBookingDate = 4  // Buchungstag
	colValueDate   = 5  // Valutadatum
	colPayerName   = 6  // Name Zahlungsbeteiligter
	colPayerIBAN   = 7  // IBAN Zahlungsbeteiligter
	colDescription = 10 // Verwendungszweck
	colAmount      = 11 // Betrag
	colCurrency    = 12 // Waehrung
)

// memberNumberRegex matches 5-digit member numbers (e.g., "11072")
var memberNumberRegex = regexp.MustCompile(`\b(\d{5})\b`)
var whitespaceRegex = regexp.MustCompile(`\s+`)
var letterDigitRegex = regexp.MustCompile(`([\p{L}])(\d)`)
var digitLetterRegex = regexp.MustCompile(`(\d)([\p{L}])`)
var nonNameCharRegex = regexp.MustCompile(`[^\p{L}\d]+`)
var germanFoldReplacer = strings.NewReplacer(
	"ã¤", "a",
	"ã¶", "o",
	"ã¼", "u",
	"ãÿ", "s",
	"ä", "a",
	"ö", "o",
	"ü", "u",
	"ß", "s",
	"ae", "a",
	"oe", "o",
	"ue", "u",
	"ss", "s",
)

// ParseBankCSV parses a German bank CSV file (semicolon-delimited, ISO-8859-1 encoded).
func ParseBankCSV(file io.Reader) ([]domain.BankTransaction, error) {
	// Wrap reader to convert from ISO-8859-1 to UTF-8
	utf8Reader := transform.NewReader(file, charmap.ISO8859_1.NewDecoder())

	reader := csv.NewReader(utf8Reader)
	reader.Comma = ';'
	reader.LazyQuotes = true

	// Skip header row
	_, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	var transactions []domain.BankTransaction

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Skip malformed rows
			continue
		}

		tx, err := parseRow(record)
		if err != nil {
			// Skip rows that can't be parsed
			continue
		}

		transactions = append(transactions, *tx)
	}

	// Sort by booking date ascending (oldest first).
	// This ensures correct matching when a family has multiple unpaid fees:
	// e.g., a December payment should match the December fee before
	// a January payment matches the January fee.
	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].BookingDate.Before(transactions[j].BookingDate)
	})

	return transactions, nil
}

func parseRow(record []string) (*domain.BankTransaction, error) {
	if len(record) < 13 {
		return nil, fmt.Errorf("insufficient columns: got %d, need at least 13", len(record))
	}

	// Parse booking date (DD.MM.YYYY)
	bookingDate, err := parseGermanDate(record[colBookingDate])
	if err != nil {
		return nil, fmt.Errorf("invalid booking date: %w", err)
	}

	// Parse value date (DD.MM.YYYY)
	valueDate, err := parseGermanDate(record[colValueDate])
	if err != nil {
		valueDate = bookingDate // Fallback to booking date
	}

	// Parse amount (German format: "1.234,56")
	amount, err := parseGermanAmount(record[colAmount])
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	// Get optional fields
	payerName := stringPtr(strings.TrimSpace(record[colPayerName]))
	payerIBAN := stringPtr(strings.TrimSpace(record[colPayerIBAN]))
	description := stringPtr(strings.TrimSpace(record[colDescription]))

	currency := strings.TrimSpace(record[colCurrency])
	if currency == "" {
		currency = "EUR"
	}

	return &domain.BankTransaction{
		ID:          uuid.New(),
		BookingDate: bookingDate,
		ValueDate:   valueDate,
		PayerName:   payerName,
		PayerIBAN:   payerIBAN,
		Description: description,
		Amount:      amount,
		Currency:    currency,
		ImportedAt:  time.Now(),
	}, nil
}

// parseGermanDate parses a date in DD.MM.YYYY format.
func parseGermanDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}

	return time.Parse("02.01.2006", s)
}

// parseGermanAmount parses a German-formatted amount (comma as decimal, dot as thousand separator).
func parseGermanAmount(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty amount")
	}

	// Remove thousand separators (dots)
	s = strings.ReplaceAll(s, ".", "")
	// Replace decimal comma with dot
	s = strings.ReplaceAll(s, ",", ".")

	return strconv.ParseFloat(s, 64)
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func normalizeMatchText(text string) string {
	if text == "" {
		return ""
	}

	normalized := strings.TrimSpace(text)
	if normalized == "" {
		return ""
	}

	normalized = strings.ToLower(normalized)
	normalized = strings.ReplaceAll(normalized, "\u00a0", " ")
	normalized = whitespaceRegex.ReplaceAllString(normalized, " ")
	normalized = letterDigitRegex.ReplaceAllString(normalized, "$1 $2")
	normalized = digitLetterRegex.ReplaceAllString(normalized, "$1 $2")

	// Fold German umlauts to their base letters and common ASCII variants.
	normalized = germanFoldReplacer.Replace(normalized)

	return normalized
}

// ExtractMemberNumber extracts a 5-digit member number from a string.
func ExtractMemberNumber(text string) string {
	normalized := normalizeMatchText(text)
	matches := memberNumberRegex.FindStringSubmatch(normalized)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// MatchChildByName attempts to find a child by matching their name in the description.
// Returns the matched child and a confidence score (0-1).
func MatchChildByName(description string, children []domain.Child) (*domain.Child, float64) {
	normalizedDescription := normalizeMatchText(description)
	if normalizedDescription == "" {
		return nil, 0
	}

	var bestMatch *domain.Child
	var bestScore float64

	for i := range children {
		child := &children[i]
		score := calculatePersonNameMatchScore(normalizedDescription, child.FirstName, child.LastName)
		if score > bestScore && score >= 0.5 {
			bestScore = score
			bestMatch = child
		}
	}

	return bestMatch, bestScore
}

// MatchChildByParentName attempts to match parents' names in the description and returns the related child.
func MatchChildByParentName(description string, children []domain.Child) (*domain.Child, float64) {
	normalizedDescription := normalizeMatchText(description)
	if normalizedDescription == "" {
		return nil, 0
	}

	var bestMatch *domain.Child
	var bestScore float64

	for i := range children {
		child := &children[i]
		for _, parent := range child.Parents {
			score := calculatePersonNameMatchScore(normalizedDescription, parent.FirstName, parent.LastName)
			if score > bestScore && score >= 0.5 {
				bestScore = score
				bestMatch = child
			}
		}
	}

	return bestMatch, bestScore
}

// calculatePersonNameMatchScore calculates how well a person's name matches the description.
func calculatePersonNameMatchScore(descNormalized string, firstNameRaw string, lastNameRaw string) float64 {
	rawFirstName := strings.TrimSpace(firstNameRaw)
	rawLastName := strings.TrimSpace(lastNameRaw)

	firstName := normalizeMatchText(firstNameRaw)
	lastName := normalizeMatchText(lastNameRaw)

	// Check various full name patterns using strings.Builder for efficiency
	fullNamePatterns := make([]string, 0, estimatedFullNameParts)

	var sb strings.Builder
	sb.Grow(len(firstName) + len(lastName) + 2)

	sb.WriteString(firstName)
	sb.WriteString(" ")
	sb.WriteString(lastName)
	fullNamePatterns = append(fullNamePatterns, sb.String())

	sb.Reset()
	sb.Grow(len(lastName) + len(firstName) + 2)
	sb.WriteString(lastName)
	sb.WriteString(" ")
	sb.WriteString(firstName)
	fullNamePatterns = append(fullNamePatterns, sb.String())

	sb.Reset()
	sb.Grow(len(lastName) + len(firstName) + 3)
	sb.WriteString(lastName)
	sb.WriteString(", ")
	sb.WriteString(firstName)
	fullNamePatterns = append(fullNamePatterns, sb.String())

	sb.Reset()
	sb.Grow(len(lastName) + len(firstName) + 4)
	sb.WriteString(lastName)
	sb.WriteString(" , ")
	sb.WriteString(firstName)
	fullNamePatterns = append(fullNamePatterns, sb.String())

	for _, pattern := range fullNamePatterns {
		if strings.Contains(descNormalized, pattern) {
			return 0.85
		}
	}

	// Compact match (no separators)
	compactDesc := nonNameCharRegex.ReplaceAllString(descNormalized, "")
	compactFirst := nonNameCharRegex.ReplaceAllString(firstName, "")
	compactLast := nonNameCharRegex.ReplaceAllString(lastName, "")
	if compactFirst != "" && compactLast != "" {
		if strings.Contains(compactDesc, compactFirst+compactLast) || strings.Contains(compactDesc, compactLast+compactFirst) {
			return 0.85
		}
	}

	// Both names present but not adjacent (still high confidence)
	if strings.Contains(descNormalized, firstName) && strings.Contains(descNormalized, lastName) {
		return 0.80
	}

	// Last name match (medium confidence)
	if strings.Contains(descNormalized, lastName) && len(rawLastName) >= 3 {
		// Boost if first name initial is also present
		if len(rawFirstName) > 0 && strings.Contains(descNormalized, string(firstName[0])+".") {
			return 0.75
		}
		return 0.6
	}

	// First name match (lower confidence, names can be common)
	if strings.Contains(descNormalized, firstName) && len(rawFirstName) >= 4 {
		return 0.4
	}

	return 0
}
