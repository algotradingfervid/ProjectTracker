package services

import (
	"fmt"
	"strings"
)

// FormatINR formats a float64 amount into Indian Rupee notation.
// It uses the Indian numbering system where, after the rightmost 3 digits,
// digits are grouped in pairs (e.g., ₹1,23,45,678.90).
// The result always includes exactly 2 decimal places.
func FormatINR(amount float64) string {
	negative := false
	if amount < 0 {
		negative = true
		amount = -amount
	}

	// Format with 2 decimal places.
	raw := fmt.Sprintf("%.2f", amount)

	// Split into integer and decimal parts.
	parts := strings.SplitN(raw, ".", 2)
	intPart := parts[0]
	decPart := parts[1]

	// Apply Indian grouping to the integer part.
	formatted := applyIndianGrouping(intPart)

	result := "₹" + formatted + "." + decPart
	if negative {
		result = "-" + result
	}
	return result
}

// applyIndianGrouping inserts commas into an integer string using the
// Indian numbering system: the rightmost 3 digits form the first group,
// then every 2 digits form subsequent groups.
func applyIndianGrouping(s string) string {
	n := len(s)
	if n <= 3 {
		return s
	}

	// The last 3 digits stay together.
	result := s[n-3:]
	remaining := s[:n-3]

	// Group remaining digits in pairs from the right.
	for len(remaining) > 2 {
		result = remaining[len(remaining)-2:] + "," + result
		remaining = remaining[:len(remaining)-2]
	}
	if len(remaining) > 0 {
		result = remaining + "," + result
	}

	return result
}
