package services

import (
	"testing"
)

func TestValidateGSTIN(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty is valid", "", true},
		{"whitespace only is valid", "   ", true},
		{"valid GSTIN", "27AAPFU0939F1ZV", true},
		{"valid GSTIN lowercase auto-uppercased", "27aapfu0939f1zv", true},
		{"valid GSTIN with leading/trailing spaces", "  27AAPFU0939F1ZV  ", true},
		{"too short", "27AAPFU0939F1Z", false},
		{"too long", "27AAPFU0939F1ZVX", false},
		{"invalid chars", "27AAPFU0939F1Z!", false},
		{"wrong structure - missing Z", "27AAPFU0939F1AV", false},
		{"all zeros", "000000000000000", false},
		{"first two not digits", "AAAAPFU0939F1ZV", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateGSTIN(tt.input)
			if got != tt.want {
				t.Errorf("ValidateGSTIN(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidatePAN(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty is valid", "", true},
		{"valid PAN", "ABCDE1234F", true},
		{"valid PAN lowercase", "abcde1234f", true},
		{"too short", "ABCDE123", false},
		{"too long", "ABCDE1234FG", false},
		{"wrong format - starts with digits", "12345ABCDF", false},
		{"wrong format - all alpha", "ABCDEFGHIJ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidatePAN(tt.input)
			if got != tt.want {
				t.Errorf("ValidatePAN(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidatePINCode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty is valid", "", true},
		{"valid PIN", "400001", true},
		{"valid PIN with spaces", "  560001  ", true},
		{"starts with zero", "012345", false},
		{"too short", "40001", false},
		{"too long", "4000012", false},
		{"contains letters", "40000A", false},
		{"all zeros", "000000", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidatePINCode(tt.input)
			if got != tt.want {
				t.Errorf("ValidatePINCode(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidatePhone(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty is valid", "", true},
		{"valid phone starting with 9", "9876543210", true},
		{"valid phone starting with 6", "6123456789", true},
		{"valid phone starting with 7", "7123456789", true},
		{"valid phone starting with 8", "8123456789", true},
		{"starts with 5", "5123456789", false},
		{"starts with 0", "0123456789", false},
		{"too short", "987654321", false},
		{"too long", "98765432100", false},
		{"contains letters", "987654321A", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidatePhone(tt.input)
			if got != tt.want {
				t.Errorf("ValidatePhone(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty is valid", "", true},
		{"valid email", "test@example.com", true},
		{"valid email with dots", "first.last@example.co.in", true},
		{"valid email with plus", "user+tag@example.com", true},
		{"missing @", "testexample.com", false},
		{"missing domain", "test@", false},
		{"missing TLD", "test@example", false},
		{"spaces", "test @example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateEmail(tt.input)
			if got != tt.want {
				t.Errorf("ValidateEmail(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateCIN(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty is valid", "", true},
		{"valid CIN", "U74999MH2000PTC123456", true},
		{"valid CIN lowercase", "u74999mh2000ptc123456", true},
		{"too short", "U74999MH2000PTC12345", false},
		{"too long", "U74999MH2000PTC1234567", false},
		{"wrong format", "123456789012345678901", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateCIN(tt.input)
			if got != tt.want {
				t.Errorf("ValidateCIN(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateAddressFormat(t *testing.T) {
	t.Run("all valid fields", func(t *testing.T) {
		fields := map[string]string{
			"gstin":    "27AAPFU0939F1ZV",
			"pan":      "ABCDE1234F",
			"pin_code": "400001",
			"phone":    "9876543210",
			"email":    "test@example.com",
			"cin":      "U74999MH2000PTC123456",
		}
		errors := ValidateAddressFormat(fields)
		if len(errors) != 0 {
			t.Errorf("expected no errors, got %v", errors)
		}
	})

	t.Run("all empty fields are valid", func(t *testing.T) {
		fields := map[string]string{
			"gstin":    "",
			"pan":      "",
			"pin_code": "",
			"phone":    "",
			"email":    "",
			"cin":      "",
		}
		errors := ValidateAddressFormat(fields)
		if len(errors) != 0 {
			t.Errorf("expected no errors for empty fields, got %v", errors)
		}
	})

	t.Run("all invalid fields", func(t *testing.T) {
		fields := map[string]string{
			"gstin":    "INVALID",
			"pan":      "INVALID",
			"pin_code": "INVALID",
			"phone":    "INVALID",
			"email":    "INVALID",
			"cin":      "INVALID",
		}
		errors := ValidateAddressFormat(fields)
		if len(errors) != 6 {
			t.Errorf("expected 6 errors, got %d: %v", len(errors), errors)
		}
	})

	t.Run("partial invalid fields", func(t *testing.T) {
		fields := map[string]string{
			"gstin":    "INVALID",
			"pan":      "ABCDE1234F",
			"pin_code": "400001",
			"phone":    "INVALID",
			"email":    "test@example.com",
		}
		errors := ValidateAddressFormat(fields)
		if len(errors) != 2 {
			t.Errorf("expected 2 errors (gstin, phone), got %d: %v", len(errors), errors)
		}
		if _, ok := errors["gstin"]; !ok {
			t.Error("expected error for gstin")
		}
		if _, ok := errors["phone"]; !ok {
			t.Error("expected error for phone")
		}
	})

	t.Run("nil map", func(t *testing.T) {
		errors := ValidateAddressFormat(map[string]string{})
		if len(errors) != 0 {
			t.Errorf("expected no errors for empty map, got %v", errors)
		}
	})
}
