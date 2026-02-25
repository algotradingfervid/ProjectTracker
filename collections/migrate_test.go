package collections_test

import (
	"testing"

	"projectcreation/collections"
	"projectcreation/testhelpers"

	"github.com/pocketbase/pocketbase/core"
)

// addressTypes mirrors the unexported var in migrate_address_settings.go
var testAddressTypes = []string{"bill_from", "ship_from", "bill_to", "ship_to", "install_at"}

func TestMigrateDefaultAddressSettings_CreatesDefaults(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Settings Project")

	if err := collections.MigrateDefaultAddressSettings(app); err != nil {
		t.Fatalf("MigrateDefaultAddressSettings() error: %v", err)
	}

	settingsCol, _ := app.FindCollectionByNameOrId("project_address_settings")

	// Should create 5 settings records (one per address type)
	for _, addrType := range testAddressTypes {
		records, err := app.FindRecordsByFilter(
			settingsCol,
			"project = {:projectId} && address_type = {:addrType}",
			"", 1, 0,
			map[string]any{"projectId": proj.Id, "addrType": addrType},
		)
		if err != nil || len(records) == 0 {
			t.Errorf("expected settings record for type %q, found none", addrType)
		}
	}
}

func TestMigrateDefaultAddressSettings_Idempotent(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	testhelpers.CreateTestProject(t, app, "Idempotent Project")

	// Run twice
	if err := collections.MigrateDefaultAddressSettings(app); err != nil {
		t.Fatalf("first run error: %v", err)
	}
	if err := collections.MigrateDefaultAddressSettings(app); err != nil {
		t.Fatalf("second run error: %v", err)
	}

	// Should still have exactly 5 settings records
	settingsCol, _ := app.FindCollectionByNameOrId("project_address_settings")
	all, err := app.FindAllRecords(settingsCol)
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	if len(all) != 5 {
		t.Errorf("expected 5 settings records, got %d", len(all))
	}
}

func TestMigrateDefaultAddressSettings_MultipleProjects(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	testhelpers.CreateTestProject(t, app, "Project A")
	testhelpers.CreateTestProject(t, app, "Project B")

	if err := collections.MigrateDefaultAddressSettings(app); err != nil {
		t.Fatalf("MigrateDefaultAddressSettings() error: %v", err)
	}

	settingsCol, _ := app.FindCollectionByNameOrId("project_address_settings")
	all, err := app.FindAllRecords(settingsCol)
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	// 2 projects * 5 address types = 10
	if len(all) != 10 {
		t.Errorf("expected 10 settings records for 2 projects, got %d", len(all))
	}
}

func TestMigrateDefaultAddressSettings_NoProjects(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	if err := collections.MigrateDefaultAddressSettings(app); err != nil {
		t.Fatalf("MigrateDefaultAddressSettings() error: %v", err)
	}

	settingsCol, _ := app.FindCollectionByNameOrId("project_address_settings")
	all, err := app.FindAllRecords(settingsCol)
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 settings records, got %d", len(all))
	}
}

func TestMigrateDefaultAddressSettings_BillFromDefaults(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "BillFrom Check")

	if err := collections.MigrateDefaultAddressSettings(app); err != nil {
		t.Fatalf("MigrateDefaultAddressSettings() error: %v", err)
	}

	settingsCol, _ := app.FindCollectionByNameOrId("project_address_settings")
	records, _ := app.FindRecordsByFilter(
		settingsCol,
		"project = {:p} && address_type = 'bill_from'",
		"", 1, 0,
		map[string]any{"p": proj.Id},
	)
	if len(records) == 0 {
		t.Fatal("no bill_from settings found")
	}

	r := records[0]
	// bill_from requires: company_name, address_line_1, city, state, pin_code, country, gstin
	if !r.GetBool("req_company_name") {
		t.Error("bill_from should require company_name")
	}
	if !r.GetBool("req_gstin") {
		t.Error("bill_from should require gstin")
	}
	if !r.GetBool("req_address_line_1") {
		t.Error("bill_from should require address_line_1")
	}
	// bill_from should NOT require contact_person or phone
	if r.GetBool("req_contact_person") {
		t.Error("bill_from should not require contact_person")
	}
	if r.GetBool("req_phone") {
		t.Error("bill_from should not require phone")
	}
}

func TestMigrateDefaultAddressSettings_ShipToDefaults(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "ShipTo Check")

	if err := collections.MigrateDefaultAddressSettings(app); err != nil {
		t.Fatalf("MigrateDefaultAddressSettings() error: %v", err)
	}

	settingsCol, _ := app.FindCollectionByNameOrId("project_address_settings")
	records, _ := app.FindRecordsByFilter(
		settingsCol,
		"project = {:p} && address_type = 'ship_to'",
		"", 1, 0,
		map[string]any{"p": proj.Id},
	)
	if len(records) == 0 {
		t.Fatal("no ship_to settings found")
	}

	r := records[0]
	// ship_to requires: contact_person, address_line_1, city, state, pin_code, country, phone
	if !r.GetBool("req_contact_person") {
		t.Error("ship_to should require contact_person")
	}
	if !r.GetBool("req_phone") {
		t.Error("ship_to should require phone")
	}
	// ship_to should NOT require gstin or company_name
	if r.GetBool("req_gstin") {
		t.Error("ship_to should not require gstin")
	}
	if r.GetBool("req_company_name") {
		t.Error("ship_to should not require company_name")
	}
}

func TestMigrateOrphanBOQs_LinksOrphans(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	// Create an orphan BOQ (no project set)
	boqsCol, _ := app.FindCollectionByNameOrId("boqs")
	orphan := core.NewRecord(boqsCol)
	orphan.Set("title", "Orphan BOQ")
	orphan.Set("reference_number", "REF-001")
	if err := app.Save(orphan); err != nil {
		t.Fatalf("failed to create orphan BOQ: %v", err)
	}

	if err := collections.MigrateOrphanBOQsToProjects(app); err != nil {
		t.Fatalf("MigrateOrphanBOQsToProjects() error: %v", err)
	}

	// Re-fetch the BOQ
	updated, err := app.FindRecordById("boqs", orphan.Id)
	if err != nil {
		t.Fatalf("failed to find BOQ after migration: %v", err)
	}

	projectID := updated.GetString("project")
	if projectID == "" {
		t.Fatal("expected BOQ to have a project after migration")
	}

	// Verify the project was created with matching name
	proj, err := app.FindRecordById("projects", projectID)
	if err != nil {
		t.Fatalf("created project not found: %v", err)
	}
	if proj.GetString("name") != "Orphan BOQ" {
		t.Errorf("project name = %q, want %q", proj.GetString("name"), "Orphan BOQ")
	}
	if proj.GetString("status") != "active" {
		t.Errorf("project status = %q, want %q", proj.GetString("status"), "active")
	}
}

func TestMigrateOrphanBOQs_NoOrphans(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	proj := testhelpers.CreateTestProject(t, app, "Has Project")
	testhelpers.CreateTestBOQ(t, app, proj.Id, "Linked BOQ")

	if err := collections.MigrateOrphanBOQsToProjects(app); err != nil {
		t.Fatalf("MigrateOrphanBOQsToProjects() error: %v", err)
	}

	// Should still have only 1 project
	projectsCol, _ := app.FindCollectionByNameOrId("projects")
	projects, _ := app.FindAllRecords(projectsCol)
	if len(projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(projects))
	}
}

func TestMigrateOrphanBOQs_MultipleOrphans(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	boqsCol, _ := app.FindCollectionByNameOrId("boqs")

	for i, title := range []string{"Orphan A", "Orphan B"} {
		r := core.NewRecord(boqsCol)
		r.Set("title", title)
		if err := app.Save(r); err != nil {
			t.Fatalf("failed to create orphan %d: %v", i, err)
		}
	}

	if err := collections.MigrateOrphanBOQsToProjects(app); err != nil {
		t.Fatalf("MigrateOrphanBOQsToProjects() error: %v", err)
	}

	// Should have created 2 projects
	projectsCol, _ := app.FindCollectionByNameOrId("projects")
	projects, _ := app.FindAllRecords(projectsCol)
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}

	// All BOQs should now have a project
	boqs, _ := app.FindAllRecords(boqsCol)
	for _, boq := range boqs {
		if boq.GetString("project") == "" {
			t.Errorf("BOQ %q still has no project", boq.GetString("title"))
		}
	}
}

func TestMigrateOrphanBOQs_Idempotent(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	boqsCol, _ := app.FindCollectionByNameOrId("boqs")
	orphan := core.NewRecord(boqsCol)
	orphan.Set("title", "Idempotent Orphan")
	if err := app.Save(orphan); err != nil {
		t.Fatalf("failed to create orphan: %v", err)
	}

	// Run twice
	if err := collections.MigrateOrphanBOQsToProjects(app); err != nil {
		t.Fatalf("first run error: %v", err)
	}
	if err := collections.MigrateOrphanBOQsToProjects(app); err != nil {
		t.Fatalf("second run error: %v", err)
	}

	// Should still have exactly 1 project
	projectsCol, _ := app.FindCollectionByNameOrId("projects")
	projects, _ := app.FindAllRecords(projectsCol)
	if len(projects) != 1 {
		t.Errorf("expected 1 project after idempotent runs, got %d", len(projects))
	}
}
