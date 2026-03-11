package services

import (
	"testing"
)

func TestValidateDCSerials_ValidIssuance(t *testing.T) {
	// Test that ValidateSerials succeeds with correct count and no duplicates
	existing := map[string]string{
		"SN-001": "DC-001",
		"SN-002": "DC-001",
	}

	result := ValidateSerials([]string{"SN-100", "SN-101", "SN-102"}, 3, existing)
	if !result.Valid {
		t.Errorf("expected valid result, got invalid: duplicatesInInput=%v, duplicatesInDB=%v, countMismatch=%v",
			result.DuplicatesInInput, result.DuplicatesInDB, result.CountMismatch)
	}
}

func TestValidateDCSerials_MissingSerials(t *testing.T) {
	// Test that validation fails when serial count doesn't match expected
	existing := map[string]string{}

	result := ValidateSerials([]string{"SN-100"}, 3, existing)
	if result.Valid {
		t.Error("expected invalid result for missing serials")
	}
	if !result.CountMismatch {
		t.Error("expected CountMismatch to be true")
	}
	if result.Expected != 3 || result.Got != 1 {
		t.Errorf("expected Expected=3, Got=1; got Expected=%d, Got=%d", result.Expected, result.Got)
	}
}

func TestValidateDCSerials_DuplicateSerials(t *testing.T) {
	// Test that validation rejects serials already in use by another DC
	existing := map[string]string{
		"SN-001": "DC-001",
		"SN-002": "DC-002",
	}

	result := ValidateSerials([]string{"SN-001", "SN-003"}, 2, existing)
	if result.Valid {
		t.Error("expected invalid result for duplicate serials")
	}
	if len(result.DuplicatesInDB) != 1 {
		t.Errorf("expected 1 DB duplicate, got %d", len(result.DuplicatesInDB))
	}
	if result.DuplicatesInDB[0].Serial != "SN-001" {
		t.Errorf("expected duplicate serial SN-001, got %s", result.DuplicatesInDB[0].Serial)
	}
	if result.DuplicatesInDB[0].ExistingDC != "DC-001" {
		t.Errorf("expected existing DC DC-001, got %s", result.DuplicatesInDB[0].ExistingDC)
	}
}

func TestValidateDCSerials_DuplicatesInInput(t *testing.T) {
	// Test that validation rejects duplicate serials within the input
	existing := map[string]string{}

	result := ValidateSerials([]string{"SN-001", "SN-002", "SN-001"}, 2, existing)
	if result.Valid {
		t.Error("expected invalid result for input duplicates")
	}
	if len(result.DuplicatesInInput) != 1 {
		t.Errorf("expected 1 input duplicate, got %d", len(result.DuplicatesInInput))
	}
}

func TestValidateDCSerials_EmptySerials(t *testing.T) {
	// Test that zero expected with zero serials is valid
	existing := map[string]string{}

	result := ValidateSerials([]string{}, 0, existing)
	if !result.Valid {
		t.Error("expected valid result for empty serials with zero expected")
	}
}
