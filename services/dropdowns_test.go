package services

import (
	"testing"
)

func TestUOMOptions(t *testing.T) {
	if len(UOMOptions) == 0 {
		t.Fatal("UOMOptions should not be empty")
	}

	// Check some expected values
	expected := map[string]bool{
		"Nos": true, "Sqm": true, "Sqft": true, "Kg": true, "Lumpsum": true,
	}
	found := make(map[string]bool)
	for _, opt := range UOMOptions {
		if opt == "" {
			t.Error("UOMOptions contains empty string")
		}
		found[opt] = true
	}
	for k := range expected {
		if !found[k] {
			t.Errorf("expected UOM option %q not found", k)
		}
	}
}

func TestGSTOptions(t *testing.T) {
	if len(GSTOptions) == 0 {
		t.Fatal("GSTOptions should not be empty")
	}

	expected := []int{0, 5, 12, 18, 28}
	if len(GSTOptions) != len(expected) {
		t.Errorf("expected %d GST options, got %d", len(expected), len(GSTOptions))
	}
	for i, v := range expected {
		if GSTOptions[i] != v {
			t.Errorf("GSTOptions[%d] = %d, want %d", i, GSTOptions[i], v)
		}
	}
}

func TestIndianStates(t *testing.T) {
	if len(IndianStates) == 0 {
		t.Fatal("IndianStates should not be empty")
	}

	// India has 28 states + 8 UTs = 36
	if len(IndianStates) < 36 {
		t.Errorf("expected at least 36 states/UTs, got %d", len(IndianStates))
	}

	expected := map[string]bool{
		"Maharashtra": true, "Karnataka": true, "Delhi": true, "Tamil Nadu": true,
	}
	found := make(map[string]bool)
	for _, s := range IndianStates {
		found[s] = true
	}
	for k := range expected {
		if !found[k] {
			t.Errorf("expected state %q not found", k)
		}
	}
}

func TestCountries(t *testing.T) {
	if len(Countries) == 0 {
		t.Fatal("Countries should not be empty")
	}
	if Countries[0] != "India" {
		t.Errorf("expected first country to be 'India', got %q", Countries[0])
	}
}
