package services

import "testing"

func TestValidateSerials_Valid(t *testing.T) {
	result := ValidateSerials([]string{"SN001", "SN002", "SN003"}, 3, nil)
	if !result.Valid {
		t.Errorf("expected valid, got invalid")
	}
}

func TestValidateSerials_DuplicatesInInput(t *testing.T) {
	result := ValidateSerials([]string{"SN001", "SN002", "SN001"}, 2, nil)
	if result.Valid {
		t.Errorf("expected invalid due to duplicates")
	}
	if len(result.DuplicatesInInput) != 1 || result.DuplicatesInInput[0] != "SN001" {
		t.Errorf("expected SN001 as duplicate, got %v", result.DuplicatesInInput)
	}
}

func TestValidateSerials_DuplicatesInDB(t *testing.T) {
	existing := map[string]string{"SN001": "DC-001"}
	result := ValidateSerials([]string{"SN001", "SN002"}, 2, existing)
	if result.Valid {
		t.Errorf("expected invalid due to DB duplicates")
	}
	if len(result.DuplicatesInDB) != 1 || result.DuplicatesInDB[0].Serial != "SN001" {
		t.Errorf("expected SN001 as DB duplicate, got %v", result.DuplicatesInDB)
	}
}

func TestValidateSerials_CountMismatch(t *testing.T) {
	result := ValidateSerials([]string{"SN001", "SN002"}, 3, nil)
	if result.Valid {
		t.Errorf("expected invalid due to count mismatch")
	}
	if !result.CountMismatch {
		t.Errorf("expected CountMismatch=true")
	}
	if result.Expected != 3 || result.Got != 2 {
		t.Errorf("expected 3/2, got %d/%d", result.Expected, result.Got)
	}
}

func TestValidateSerials_EmptyStringsFiltered(t *testing.T) {
	result := ValidateSerials([]string{"SN001", "", "  ", "SN002"}, 2, nil)
	if !result.Valid {
		t.Errorf("expected valid after filtering empty strings")
	}
}

func TestValidateSerials_AllEmpty(t *testing.T) {
	result := ValidateSerials([]string{"", "  "}, 0, nil)
	if !result.Valid {
		t.Errorf("expected valid for 0 expected with all empty")
	}
}
