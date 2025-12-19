package service

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// FormatNormalizer handles normalization of different data formats
type FormatNormalizer struct {
	dateFormats   []string
	phonePattern  *regexp.Regexp
	numberPattern *regexp.Regexp
}

// NewFormatNormalizer creates a new format normalizer
func NewFormatNormalizer() *FormatNormalizer {
	return &FormatNormalizer{
		dateFormats: []string{
			"2006-01-02",          // ISO: 2024-01-15
			"01/02/2006",          // US: 01/15/2024
			"02/01/2006",          // EU: 15/01/2024
			"2006/01/02",          // Alt ISO
			"02-Jan-2006",         // Text: 15-Jan-2024
			"January 2, 2006",     // Full text
			time.RFC3339,          // With time
			"2006-01-02 15:04:05", // SQL datetime
		},
		phonePattern:  regexp.MustCompile(`[\s\-\(\)\+\.]`),
		numberPattern: regexp.MustCompile(`[\$€£¥₹,\s]`),
	}
}

// NormalizeValue attempts to normalize a value to a standard format
func (fn *FormatNormalizer) NormalizeValue(value string) string {
	if value == "" {
		return ""
	}

	// Try date normalization
	if normalized := fn.normalizeDate(value); normalized != "" {
		return normalized
	}

	// Try phone normalization
	if normalized := fn.normalizePhone(value); normalized != "" {
		return normalized
	}

	// Try number/currency normalization
	if normalized := fn.normalizeNumber(value); normalized != "" {
		return normalized
	}

	// Try name normalization
	if normalized := fn.normalizeName(value); normalized != "" {
		return normalized
	}

	// Return lowercase trimmed as fallback
	return strings.ToLower(strings.TrimSpace(value))
}

// normalizeDate tries to parse and normalize dates to ISO format
func (fn *FormatNormalizer) normalizeDate(value string) string {
	for _, format := range fn.dateFormats {
		if t, err := time.Parse(format, value); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return ""
}

// normalizePhone removes formatting from phone numbers
func (fn *FormatNormalizer) normalizePhone(value string) string {
	// Check if it looks like a phone number
	if !strings.ContainsAny(value, "0123456789") {
		return ""
	}

	// Remove all formatting
	normalized := fn.phonePattern.ReplaceAllString(value, "")

	// Must be 10-15 digits to be a phone
	if len(normalized) >= 10 && len(normalized) <= 15 {
		return normalized
	}

	return ""
}

// normalizeNumber removes currency symbols and formatting
func (fn *FormatNormalizer) normalizeNumber(value string) string {
	// Remove currency symbols and commas
	normalized := fn.numberPattern.ReplaceAllString(value, "")

	// Check if it's a valid number
	if _, err := parseFloat(normalized); err == nil {
		return normalized
	}

	return ""
}

// normalizeName standardizes name formats
func (fn *FormatNormalizer) normalizeName(value string) string {
	// Check if it looks like a name (contains letters and possibly comma/space)
	if !regexp.MustCompile(`[a-zA-Z]`).MatchString(value) {
		return ""
	}

	// Handle "Last, First" format
	if strings.Contains(value, ",") {
		parts := strings.Split(value, ",")
		if len(parts) == 2 {
			// Convert "Smith, John" to "john smith"
			return strings.ToLower(strings.TrimSpace(parts[1]) + " " + strings.TrimSpace(parts[0]))
		}
	}

	// Just lowercase and normalize spaces
	normalized := strings.ToLower(value)
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")
	return strings.TrimSpace(normalized)
}

// DetectFormat identifies the format type of a value
func (fn *FormatNormalizer) DetectFormat(value string) string {
	if fn.normalizeDate(value) != "" {
		return "date"
	}
	if fn.normalizePhone(value) != "" {
		return "phone"
	}
	if fn.normalizeNumber(value) != "" {
		return "number"
	}
	if fn.normalizeName(value) != "" && strings.Contains(value, " ") {
		return "name"
	}
	return "text"
}

// Helper function to parse float
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
