package services

import (
	"testing"
)

func TestShipToTemplateFields(t *testing.T) {
	fields := ShipToTemplateFields()
	if len(fields) == 0 {
		t.Fatal("ShipToTemplateFields() returned empty")
	}

	// Should have 17 fields
	if len(fields) != 17 {
		t.Errorf("expected 17 fields, got %d", len(fields))
	}

	// First field should be company_name
	if fields[0].Key != "company_name" {
		t.Errorf("first field key = %q, want 'company_name'", fields[0].Key)
	}

	// Check always-required fields
	alwaysRequired := map[string]bool{
		"address_line_1": true,
		"city":           true,
		"state":          true,
		"country":        true,
	}
	for _, f := range fields {
		if alwaysRequired[f.Key] && !f.AlwaysRequired {
			t.Errorf("field %q should be AlwaysRequired", f.Key)
		}
		if !alwaysRequired[f.Key] && f.AlwaysRequired {
			t.Errorf("field %q should not be AlwaysRequired", f.Key)
		}
	}

	// All fields should have Key and Label
	for _, f := range fields {
		if f.Key == "" {
			t.Error("found field with empty Key")
		}
		if f.Label == "" {
			t.Errorf("field %q has empty Label", f.Key)
		}
	}
}

func TestInstallAtTemplateFields(t *testing.T) {
	fields := InstallAtTemplateFields()
	if len(fields) == 0 {
		t.Fatal("InstallAtTemplateFields() returned empty")
	}

	// Should have 18 fields (17 ship_to + ship_to_reference)
	if len(fields) != 18 {
		t.Errorf("expected 18 fields, got %d", len(fields))
	}

	// First field should be ship_to_reference
	if fields[0].Key != "ship_to_reference" {
		t.Errorf("first field key = %q, want 'ship_to_reference'", fields[0].Key)
	}
	if !fields[0].AlwaysRequired {
		t.Error("ship_to_reference should be AlwaysRequired")
	}

	// Second field should be company_name (from ShipToTemplateFields)
	if fields[1].Key != "company_name" {
		t.Errorf("second field key = %q, want 'company_name'", fields[1].Key)
	}
}
