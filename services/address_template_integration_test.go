package services

import (
	"testing"

	"projectcreation/collections"
	"projectcreation/testhelpers"

	"github.com/xuri/excelize/v2"
)

func TestGenerateAddressTemplate_ShipTo(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Template Project")
	collections.MigrateDefaultAddressSettings(app)

	result, err := GenerateAddressTemplate(app, proj.Id, "ship_to")
	if err != nil {
		t.Fatalf("GenerateAddressTemplate() error: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("GenerateAddressTemplate() returned empty bytes")
	}

	// Verify valid Excel
	f, err := excelize.OpenReader(bytesReader(result))
	if err != nil {
		t.Fatalf("result is not valid Excel: %v", err)
	}
	defer f.Close()

	// Check main sheet name
	sheets := f.GetSheetList()
	if sheets[0] != "Addresses" {
		t.Errorf("expected first sheet 'Addresses', got %q", sheets[0])
	}

	// Check header row has expected columns
	fields := ShipToTemplateFields()
	for i, field := range fields {
		col, _ := excelize.ColumnNumberToName(i + 1)
		cell := col + "1"
		val, _ := f.GetCellValue("Addresses", cell)
		if val == "" {
			t.Errorf("expected header at %s for field %q, got empty", cell, field.Label)
		}
	}
}

func TestGenerateAddressTemplate_InstallAt(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "InstallAt Template")
	collections.MigrateDefaultAddressSettings(app)

	result, err := GenerateAddressTemplate(app, proj.Id, "install_at")
	if err != nil {
		t.Fatalf("GenerateAddressTemplate() error: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("GenerateAddressTemplate() returned empty bytes")
	}

	f, err := excelize.OpenReader(bytesReader(result))
	if err != nil {
		t.Fatalf("result is not valid Excel: %v", err)
	}
	defer f.Close()

	// First column should be "Ship To Reference *" (always required)
	a1, _ := f.GetCellValue("Addresses", "A1")
	if a1 != "Ship To Reference *" {
		t.Errorf("expected first column 'Ship To Reference *', got %q", a1)
	}

	// install_at has more columns than ship_to (ship_to_reference prepended)
	installFields := InstallAtTemplateFields()
	shipToFields := ShipToTemplateFields()
	if len(installFields) != len(shipToFields)+1 {
		t.Errorf("install_at should have 1 more field than ship_to: %d vs %d",
			len(installFields), len(shipToFields))
	}
}

func TestGenerateAddressTemplate_RequiredFieldsMarked(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Required Markers")
	collections.MigrateDefaultAddressSettings(app)

	result, err := GenerateAddressTemplate(app, proj.Id, "ship_to")
	if err != nil {
		t.Fatalf("GenerateAddressTemplate() error: %v", err)
	}

	f, err := excelize.OpenReader(bytesReader(result))
	if err != nil {
		t.Fatalf("invalid Excel: %v", err)
	}
	defer f.Close()

	// address_line_1 is AlwaysRequired, should have " *" suffix
	// Find which column it is
	fields := ShipToTemplateFields()
	for i, field := range fields {
		if field.Key == "address_line_1" {
			col, _ := excelize.ColumnNumberToName(i + 1)
			val, _ := f.GetCellValue("Addresses", col+"1")
			if val != "Address Line 1 *" {
				t.Errorf("expected 'Address Line 1 *', got %q", val)
			}
			break
		}
	}

	// contact_person is required by default for ship_to settings
	for i, field := range fields {
		if field.Key == "contact_person" {
			col, _ := excelize.ColumnNumberToName(i + 1)
			val, _ := f.GetCellValue("Addresses", col+"1")
			if val != "Contact Person *" {
				t.Errorf("expected 'Contact Person *', got %q", val)
			}
			break
		}
	}
}

func TestGenerateAddressTemplate_HasInstructionsSheet(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Instructions Test")
	collections.MigrateDefaultAddressSettings(app)

	result, err := GenerateAddressTemplate(app, proj.Id, "ship_to")
	if err != nil {
		t.Fatalf("GenerateAddressTemplate() error: %v", err)
	}

	f, err := excelize.OpenReader(bytesReader(result))
	if err != nil {
		t.Fatalf("invalid Excel: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	found := false
	for _, s := range sheets {
		if s == "Instructions" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'Instructions' sheet to exist")
	}

	// Instructions sheet should have a title
	title, _ := f.GetCellValue("Instructions", "A1")
	if title == "" {
		t.Error("Instructions sheet A1 should have a title")
	}
}

func TestGenerateAddressTemplate_NoSettings(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "No Settings Template")

	// No MigrateDefaultAddressSettings - should still generate a template
	// with only AlwaysRequired fields marked
	result, err := GenerateAddressTemplate(app, proj.Id, "ship_to")
	if err != nil {
		t.Fatalf("GenerateAddressTemplate() error: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("expected non-empty template even without settings")
	}

	f, err := excelize.OpenReader(bytesReader(result))
	if err != nil {
		t.Fatalf("invalid Excel: %v", err)
	}
	defer f.Close()

	// address_line_1 is AlwaysRequired, should still have " *"
	fields := ShipToTemplateFields()
	for i, field := range fields {
		if field.Key == "address_line_1" {
			col, _ := excelize.ColumnNumberToName(i + 1)
			val, _ := f.GetCellValue("Addresses", col+"1")
			if val != "Address Line 1 *" {
				t.Errorf("expected 'Address Line 1 *', got %q", val)
			}
			break
		}
	}

	// contact_person should NOT be marked required without settings
	for i, field := range fields {
		if field.Key == "contact_person" {
			col, _ := excelize.ColumnNumberToName(i + 1)
			val, _ := f.GetCellValue("Addresses", col+"1")
			if val != "Contact Person" {
				t.Errorf("expected 'Contact Person' (no asterisk without settings), got %q", val)
			}
			break
		}
	}
}
