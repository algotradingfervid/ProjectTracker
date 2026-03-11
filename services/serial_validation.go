package services

import "strings"

// SerialConflict represents a serial number that already exists in another DC.
type SerialConflict struct {
	Serial     string
	ExistingDC string
}

// SerialValidationResult contains the outcome of serial number validation.
type SerialValidationResult struct {
	Valid             bool
	DuplicatesInInput []string
	DuplicatesInDB    []SerialConflict
	CountMismatch     bool
	Expected, Got     int
}

// ValidateSerials checks serial numbers for duplicates and count mismatches.
func ValidateSerials(input []string, expectedQty int, existingSerials map[string]string) SerialValidationResult {
	result := SerialValidationResult{Valid: true}

	seen := make(map[string]bool)
	var cleaned []string
	for _, s := range input {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if seen[s] {
			result.DuplicatesInInput = append(result.DuplicatesInInput, s)
			result.Valid = false
		} else {
			seen[s] = true
			cleaned = append(cleaned, s)
		}
	}

	for _, s := range cleaned {
		if dcNum, exists := existingSerials[s]; exists {
			result.DuplicatesInDB = append(result.DuplicatesInDB, SerialConflict{Serial: s, ExistingDC: dcNum})
			result.Valid = false
		}
	}

	uniqueCount := len(seen)
	if uniqueCount != expectedQty {
		result.CountMismatch = true
		result.Expected = expectedQty
		result.Got = uniqueCount
		result.Valid = false
	}

	return result
}
