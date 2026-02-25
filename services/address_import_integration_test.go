package services

import (
	"testing"

	"projectcreation/collections"
	"projectcreation/testhelpers"
)

func TestCommitAddressImport_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Import Project")
	collections.MigrateDefaultAddressSettings(app)

	rows := []map[string]string{
		{
			"company_name":   "Test Corp A",
			"contact_person": "John Doe",
			"address_line_1": "123 Main St",
			"city":           "Mumbai",
			"state":          "Maharashtra",
			"pin_code":       "400001",
			"country":        "India",
			"phone":          "9876543210",
		},
		{
			"company_name":   "Test Corp B",
			"contact_person": "Jane Doe",
			"address_line_1": "456 Second St",
			"city":           "Delhi",
			"state":          "Delhi",
			"pin_code":       "110001",
			"country":        "India",
			"phone":          "9876543211",
		},
	}

	result, err := CommitAddressImport(app, proj.Id, "ship_to", rows)
	if err != nil {
		t.Fatalf("CommitAddressImport() error: %v", err)
	}
	if result.TotalRows != 2 {
		t.Errorf("TotalRows = %d, want 2", result.TotalRows)
	}
	if result.Imported != 2 {
		t.Errorf("Imported = %d, want 2", result.Imported)
	}
	if result.Failed != 0 {
		t.Errorf("Failed = %d, want 0", result.Failed)
	}

	// Verify records in DB
	col, _ := app.FindCollectionByNameOrId("addresses")
	records, _ := app.FindRecordsByFilter(col,
		"project = {:p} && address_type = 'ship_to'", "", 0, 0,
		map[string]any{"p": proj.Id},
	)
	if len(records) != 2 {
		t.Errorf("expected 2 addresses in DB, got %d", len(records))
	}
}

func TestCommitAddressImport_Empty(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Empty Import")
	collections.MigrateDefaultAddressSettings(app)

	result, err := CommitAddressImport(app, proj.Id, "ship_to", []map[string]string{})
	if err != nil {
		t.Fatalf("CommitAddressImport() error: %v", err)
	}
	if result.TotalRows != 0 {
		t.Errorf("TotalRows = %d, want 0", result.TotalRows)
	}
	if result.Imported != 0 {
		t.Errorf("Imported = %d, want 0", result.Imported)
	}
}

func TestCommitAddressImport_ValidationErrors(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Invalid Import")
	collections.MigrateDefaultAddressSettings(app)

	// Missing required fields for ship_to (contact_person, address_line_1, etc.)
	rows := []map[string]string{
		{
			"company_name": "Corp A",
			// Missing: contact_person, address_line_1, city, state, pin_code, country, phone
		},
	}

	result, err := CommitAddressImport(app, proj.Id, "ship_to", rows)
	if err != nil {
		t.Fatalf("CommitAddressImport() error: %v", err)
	}
	if result.Imported != 0 {
		t.Errorf("Imported = %d, want 0 (should fail validation)", result.Imported)
	}
	if result.Failed == 0 {
		t.Error("expected failed rows due to validation")
	}
	if !result.RolledBack {
		t.Error("expected RolledBack = true")
	}
	if len(result.Errors) == 0 {
		t.Error("expected validation errors")
	}
}

func TestCommitAddressImport_InstallAt_WithShipToRef(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "InstallAt Import")
	collections.MigrateDefaultAddressSettings(app)

	// Create a ship_to address first
	testhelpers.CreateTestAddress(t, app, proj.Id, "ship_to", "Ship Corp")

	rows := []map[string]string{
		{
			"ship_to_reference": "Ship Corp",
			"contact_person":    "John Doe",
			"address_line_1":    "789 Install Rd",
			"city":              "Bangalore",
			"state":             "Karnataka",
			"pin_code":          "560001",
			"country":           "India",
			"phone":             "9876543210",
		},
	}

	result, err := CommitAddressImport(app, proj.Id, "install_at", rows)
	if err != nil {
		t.Fatalf("CommitAddressImport() error: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("Imported = %d, want 1", result.Imported)
	}

	// Verify the install_at record has ship_to_parent set
	col, _ := app.FindCollectionByNameOrId("addresses")
	records, _ := app.FindRecordsByFilter(col,
		"project = {:p} && address_type = 'install_at'", "", 0, 0,
		map[string]any{"p": proj.Id},
	)
	if len(records) != 1 {
		t.Fatalf("expected 1 install_at address, got %d", len(records))
	}
	if records[0].GetString("ship_to_parent") == "" {
		t.Error("expected ship_to_parent to be set")
	}
}

func TestCommitAddressImport_InstallAt_InvalidRef(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Bad Ref Import")
	collections.MigrateDefaultAddressSettings(app)

	// No ship_to addresses exist, so reference will fail
	rows := []map[string]string{
		{
			"ship_to_reference": "Nonexistent Corp",
			"contact_person":    "John Doe",
			"address_line_1":    "789 Install Rd",
			"city":              "Bangalore",
			"state":             "Karnataka",
			"pin_code":          "560001",
			"country":           "India",
			"phone":             "9876543210",
		},
	}

	result, err := CommitAddressImport(app, proj.Id, "install_at", rows)
	if err != nil {
		t.Fatalf("CommitAddressImport() error: %v", err)
	}
	if result.Imported != 0 {
		t.Errorf("Imported = %d, want 0", result.Imported)
	}
	if len(result.Errors) == 0 {
		t.Error("expected errors for invalid ship_to_reference")
	}
}

func TestCommitAddressImport_LargeBatch(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Large Import")

	// Create settings that don't require many fields (to simplify the test)
	createAddressSettings(t, app, proj.Id, "ship_to", map[string]bool{})

	// Generate 150 rows (exceeds importBatchSize of 100, tests chunking)
	rows := make([]map[string]string, 150)
	for i := range rows {
		rows[i] = map[string]string{
			"address_line_1": "123 Main St",
			"city":           "Mumbai",
			"state":          "Maharashtra",
			"country":        "India",
		}
	}

	result, err := CommitAddressImport(app, proj.Id, "ship_to", rows)
	if err != nil {
		t.Fatalf("CommitAddressImport() error: %v", err)
	}
	if result.TotalRows != 150 {
		t.Errorf("TotalRows = %d, want 150", result.TotalRows)
	}
	if result.Imported != 150 {
		t.Errorf("Imported = %d, want 150", result.Imported)
	}
}
