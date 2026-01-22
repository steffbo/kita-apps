package csvparser

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// ParseResult contains the result of parsing a CSV file
type ParseResult struct {
	Headers           []string   `json:"headers"`
	SampleRows        [][]string `json:"sampleRows"`
	AllRows           [][]string `json:"-"` // Not exposed in JSON, used internally
	DetectedSeparator string     `json:"detectedSeparator"`
	TotalRows         int        `json:"totalRows"`
}

// ParseCSV parses a CSV file with automatic encoding and separator detection
func ParseCSV(reader io.Reader) (*ParseResult, error) {
	// Read all content first
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// Detect and convert encoding if needed
	content = convertEncoding(content)

	// Detect separator
	separator := detectSeparator(content)

	// Parse CSV
	csvReader := csv.NewReader(bytes.NewReader(content))
	csvReader.Comma = rune(separator[0])
	csvReader.LazyQuotes = true
	csvReader.FieldsPerRecord = -1 // Allow variable number of fields

	var allRows [][]string
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Skip malformed rows
			continue
		}
		// Trim whitespace from all fields
		for i := range record {
			record[i] = strings.TrimSpace(record[i])
		}
		allRows = append(allRows, record)
	}

	if len(allRows) == 0 {
		return &ParseResult{
			Headers:           []string{},
			SampleRows:        [][]string{},
			AllRows:           [][]string{},
			DetectedSeparator: separator,
			TotalRows:         0,
		}, nil
	}

	// First row is headers
	headers := allRows[0]
	dataRows := allRows[1:]

	// Get sample rows (up to 5)
	sampleCount := 5
	if len(dataRows) < sampleCount {
		sampleCount = len(dataRows)
	}
	sampleRows := dataRows[:sampleCount]

	return &ParseResult{
		Headers:           headers,
		SampleRows:        sampleRows,
		AllRows:           dataRows,
		DetectedSeparator: separator,
		TotalRows:         len(dataRows),
	}, nil
}

// convertEncoding detects if content is not UTF-8 and converts from ISO-8859-1
func convertEncoding(content []byte) []byte {
	if utf8.Valid(content) {
		return content
	}

	// Try to convert from ISO-8859-1 (Latin-1)
	reader := transform.NewReader(bytes.NewReader(content), charmap.ISO8859_1.NewDecoder())
	converted, err := io.ReadAll(reader)
	if err != nil {
		return content // Return original if conversion fails
	}
	return converted
}

// detectSeparator tries to detect the CSV separator by analyzing the content
func detectSeparator(content []byte) string {
	// Read first few lines
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var lines []string
	for i := 0; i < 5 && scanner.Scan(); i++ {
		lines = append(lines, scanner.Text())
	}

	if len(lines) == 0 {
		return ";"
	}

	// Count occurrences of potential separators
	separators := []string{";", ",", "\t"}
	counts := make(map[string][]int)

	for _, sep := range separators {
		counts[sep] = make([]int, len(lines))
		for i, line := range lines {
			counts[sep][i] = strings.Count(line, sep)
		}
	}

	// Find separator with most consistent count across lines
	bestSep := ";"
	bestScore := 0

	for sep, lineCounts := range counts {
		if len(lineCounts) == 0 || lineCounts[0] == 0 {
			continue
		}

		// Check consistency (all lines should have same count)
		consistent := true
		firstCount := lineCounts[0]
		for _, c := range lineCounts {
			if c != firstCount {
				consistent = false
				break
			}
		}

		score := firstCount
		if consistent {
			score += 10 // Bonus for consistency
		}

		if score > bestScore {
			bestScore = score
			bestSep = sep
		}
	}

	return bestSep
}

// ParseDate tries to parse a date string in German or ISO format
func ParseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}

	// German format: DD.MM.YYYY
	germanFormats := []string{
		"02.01.2006",
		"2.1.2006",
		"02.01.06",
		"2.1.06",
	}

	// ISO format: YYYY-MM-DD
	isoFormats := []string{
		"2006-01-02",
		"2006-1-2",
	}

	// Try German formats first (more common in German context)
	for _, format := range germanFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	// Try ISO formats
	for _, format := range isoFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	// Try with time component (sometimes exports include 00:00:00)
	withTimeFormats := []string{
		"02.01.2006 15:04:05",
		"2006-01-02 15:04:05",
		"02.01.2006 15:04",
		"2006-01-02 15:04",
	}

	for _, format := range withTimeFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, &DateParseError{Value: s}
}

// DateParseError indicates a date parsing failure
type DateParseError struct {
	Value string
}

func (e *DateParseError) Error() string {
	return "cannot parse date: " + e.Value
}

// ParseInt tries to parse an integer, handling German number formats
func ParseInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	// Remove thousand separators (both . and ,)
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", "")

	// Remove any non-numeric characters except minus
	re := regexp.MustCompile(`[^\d-]`)
	s = re.ReplaceAllString(s, "")

	if s == "" {
		return 0, nil
	}

	var result int
	_, err := stringToInt(s, &result)
	return result, err
}

func stringToInt(s string, result *int) (int, error) {
	negative := false
	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	}

	val := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			continue
		}
		val = val*10 + int(c-'0')
	}

	if negative {
		val = -val
	}
	*result = val
	return val, nil
}

// FieldMapping defines how CSV columns map to system fields
type FieldMapping struct {
	CSVColumn   int    `json:"csvColumn"`   // Index of CSV column
	SystemField string `json:"systemField"` // Name of system field
}

// SystemFields defines all available system fields for mapping
var ChildFields = []string{
	"memberNumber",
	"firstName",
	"lastName",
	"birthDate",
	"entryDate",
	"street",
	"streetNo",
	"postalCode",
	"city",
	"legalHours",
	"careHours",
}

var ParentFields = []string{
	"parent1FirstName",
	"parent1LastName",
	"parent1Email",
	"parent1Phone",
	"parent2FirstName",
	"parent2LastName",
	"parent2Email",
	"parent2Phone",
}

// RequiredChildFields are mandatory for import
var RequiredChildFields = []string{
	"memberNumber",
	"firstName",
	"lastName",
	"birthDate",
	"entryDate",
}
