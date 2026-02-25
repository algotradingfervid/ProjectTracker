package services

import (
	"testing"

	"projectcreation/collections"
	"projectcreation/testhelpers"

	"github.com/pocketbase/pocketbase/core"
)

// createAddressSettings creates a project_address_settings record with custom required fields.
func createAddressSettings(t *testing.T, app core.App, projectID, addressType string, requiredFields map[string]bool) {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("project_address_settings")
	if err != nil {
		t.Fatalf("project_address_settings collection not found: %v", err)
	}
	record := core.NewRecord(col)
	record.Set("project", projectID)
	record.Set("address_type", addressType)
	for field, required := range requiredFields {
		record.Set(field, required)
	}
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save address settings: %v", err)
	}
}

func TestValidateAddress_BillFrom_AllRequired(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Validation Project")
	collections.MigrateDefaultAddressSettings(app)

	formData := map[string]string{
		"company_name":   "Test Corp",
		"address_line_1": "123 Main St",
		"city":           "Mumbai",
		"state":          "Maharashtra",
		"pin_code":       "400001",
		"country":        "India",
		"gstin":          "27AAPFU0939F1ZV",
	}

	errors := ValidateAddress(app, proj.Id, "bill_from", formData)
	if len(errors) != 0 {
		t.Errorf("expected no errors, got %v", errors)
	}
}

func TestValidateAddress_BillFrom_MissingRequired(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Missing Fields Project")
	collections.MigrateDefaultAddressSettings(app)

	// Missing all required fields
	formData := map[string]string{}

	errors := ValidateAddress(app, proj.Id, "bill_from", formData)
	// bill_from defaults require: company_name, address_line_1, city, state, pin_code, country, gstin
	if len(errors) < 7 {
		t.Errorf("expected at least 7 errors for missing bill_from fields, got %d: %v", len(errors), errors)
	}
	if _, ok := errors["company_name"]; !ok {
		t.Error("expected error for company_name")
	}
	if _, ok := errors["gstin"]; !ok {
		t.Error("expected error for gstin")
	}
}

func TestValidateAddress_ShipTo_CustomRequirements(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Custom Settings")

	// Create custom settings: only require company_name and phone
	createAddressSettings(t, app, proj.Id, "ship_to", map[string]bool{
		"req_company_name": true,
		"req_phone":        true,
	})

	t.Run("all provided", func(t *testing.T) {
		formData := map[string]string{
			"company_name": "Test Corp",
			"phone":        "9876543210",
		}
		errors := ValidateAddress(app, proj.Id, "ship_to", formData)
		if len(errors) != 0 {
			t.Errorf("expected no errors, got %v", errors)
		}
	})

	t.Run("missing phone", func(t *testing.T) {
		formData := map[string]string{
			"company_name": "Test Corp",
		}
		errors := ValidateAddress(app, proj.Id, "ship_to", formData)
		if _, ok := errors["phone"]; !ok {
			t.Error("expected error for missing phone")
		}
	})
}

func TestValidateAddress_NoSettings(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "No Settings")

	// No settings created - should return no errors (nothing is required)
	formData := map[string]string{}
	errors := ValidateAddress(app, proj.Id, "bill_to", formData)
	if len(errors) != 0 {
		t.Errorf("expected no errors when no settings exist, got %v", errors)
	}
}

func TestValidateAddress_InvalidProject(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	formData := map[string]string{}
	errors := ValidateAddress(app, "nonexistent_id", "bill_from", formData)
	if len(errors) != 0 {
		t.Errorf("expected no errors for invalid project, got %v", errors)
	}
}

func TestGetRequiredFields_DefaultSettings(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Defaults Project")
	collections.MigrateDefaultAddressSettings(app)

	t.Run("bill_from", func(t *testing.T) {
		required := GetRequiredFields(app, proj.Id, "bill_from")
		expectedRequired := []string{"company_name", "address_line_1", "city", "state", "pin_code", "country", "gstin"}
		for _, field := range expectedRequired {
			if !required[field] {
				t.Errorf("expected %q to be required for bill_from", field)
			}
		}
		// contact_person should NOT be required for bill_from
		if required["contact_person"] {
			t.Error("contact_person should not be required for bill_from")
		}
	})

	t.Run("ship_to", func(t *testing.T) {
		required := GetRequiredFields(app, proj.Id, "ship_to")
		expectedRequired := []string{"contact_person", "address_line_1", "city", "state", "pin_code", "country", "phone"}
		for _, field := range expectedRequired {
			if !required[field] {
				t.Errorf("expected %q to be required for ship_to", field)
			}
		}
		// gstin should NOT be required for ship_to
		if required["gstin"] {
			t.Error("gstin should not be required for ship_to")
		}
	})

	t.Run("install_at", func(t *testing.T) {
		required := GetRequiredFields(app, proj.Id, "install_at")
		if !required["contact_person"] {
			t.Error("expected contact_person to be required for install_at")
		}
		if !required["phone"] {
			t.Error("expected phone to be required for install_at")
		}
	})
}

func TestGetRequiredFields_CustomSettings(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Custom Project")

	createAddressSettings(t, app, proj.Id, "bill_to", map[string]bool{
		"req_company_name": true,
		"req_email":        true,
		"req_pan":          true,
	})

	required := GetRequiredFields(app, proj.Id, "bill_to")
	if !required["company_name"] {
		t.Error("expected company_name to be required")
	}
	if !required["email"] {
		t.Error("expected email to be required")
	}
	if !required["pan"] {
		t.Error("expected pan to be required")
	}
	if required["phone"] {
		t.Error("phone should not be required")
	}
}

func TestGetRequiredFields_NoSettings(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Empty Settings")

	required := GetRequiredFields(app, proj.Id, "bill_from")
	if len(required) != 0 {
		t.Errorf("expected no required fields when no settings exist, got %v", required)
	}
}
