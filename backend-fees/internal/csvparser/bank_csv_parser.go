package csvparser

import (
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

// Column indices for the bank CSV format
const (
	colBookingDate = 0  // Buchungstag
	colValueDate   = 1  // Valutadatum
	colPayerName   = 11 // Name Zahlungsbeteiligter
	colPayerIBAN   = 12 // IBAN Zahlungsbeteiligter
	colDescription = 4  // Verwendungszweck
	colAmount      = 14 // Betrag
	colCurrency    = 15 // Währung
)

// memberNumberRegex matches 5-digit member numbers (e.g., "11072")
var memberNumberRegex = regexp.MustCompile(`\b(\d{5})\b`)

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

	return transactions, nil
}

func parseRow(record []string) (*domain.BankTransaction, error) {
	if len(record) < 16 {
		return nil, fmt.Errorf("insufficient columns: got %d, need at least 16", len(record))
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

// ExtractMemberNumber extracts a 5-digit member number from a string.
func ExtractMemberNumber(text string) string {
	matches := memberNumberRegex.FindStringSubmatch(text)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// MatchChildByName attempts to find a child by matching their name in the description.
// Returns the matched child and a confidence score (0-1).
func MatchChildByName(description string, children []domain.Child) (*domain.Child, float64) {
	if description == "" {
		return nil, 0
	}

	descLower := strings.ToLower(description)
	var bestMatch *domain.Child
	var bestScore float64

	for i := range children {
		child := &children[i]
		score := calculateNameMatchScore(descLower, child)
		if score > bestScore && score >= 0.5 {
			bestScore = score
			bestMatch = child
		}
	}

	return bestMatch, bestScore
}

// calculateNameMatchScore calculates how well a child's name matches the description.
func calculateNameMatchScore(descLower string, child *domain.Child) float64 {
	firstName := strings.ToLower(child.FirstName)
	lastName := strings.ToLower(child.LastName)
	fullName := firstName + " " + lastName
	fullNameReverse := lastName + " " + firstName

	// Full name match (highest confidence)
	if strings.Contains(descLower, fullName) || strings.Contains(descLower, fullNameReverse) {
		return 0.85
	}

	// Last name match (medium confidence)
	if strings.Contains(descLower, lastName) && len(lastName) >= 3 {
		// Boost if first name initial is also present
		if len(firstName) > 0 && strings.Contains(descLower, string(firstName[0])+".") {
			return 0.75
		}
		return 0.6
	}

	// First name match (lower confidence, names can be common)
	if strings.Contains(descLower, firstName) && len(firstName) >= 4 {
		return 0.4
	}

	return 0
}
