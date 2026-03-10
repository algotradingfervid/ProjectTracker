package services

import (
	"testing"
)

func TestDefaultColumnDefs_HasExpectedFields(t *testing.T) {
	types := []string{"bill_from", "dispatch_from", "bill_to", "ship_to", "install_at"}
	for _, addrType := range types {
		t.Run(addrType, func(t *testing.T) {
			cols := ParseColumnDefs(DefaultColumnDefsJSON(addrType))
			if len(cols) == 0 {
				t.Error("expected columns, got none")
			}
			// Every type should have company_name
			found := false
			for _, c := range cols {
				if c.Name == "company_name" {
					found = true
					break
				}
			}
			if !found {
				t.Error("expected company_name column")
			}
		})
	}
}

func TestParseColumnDefs_RoundTrips(t *testing.T) {
	json := DefaultColumnDefsJSON("bill_to")
	cols := ParseColumnDefs(json)
	if len(cols) < 10 {
		t.Errorf("expected at least 10 columns, got %d", len(cols))
	}
	// Check bill_to required fields
	for _, c := range cols {
		if c.Name == "gstin" && !c.Required {
			t.Error("bill_to should require gstin")
		}
	}
}
