package collections_test

import (
	"testing"

	"projectcreation/collections"
	"projectcreation/testhelpers"
)

func TestSeed_CreatesData(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	if err := collections.Seed(app); err != nil {
		t.Fatalf("Seed() error: %v", err)
	}

	// Verify project was created
	projectsCol, _ := app.FindCollectionByNameOrId("projects")
	projects, err := app.FindAllRecords(projectsCol)
	if err != nil {
		t.Fatalf("query projects error: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	if projects[0].GetString("name") != "Interior Fit-Out — Block A" {
		t.Errorf("project name = %q, want %q", projects[0].GetString("name"), "Interior Fit-Out — Block A")
	}

	// Verify BOQ was created and linked to project
	boqsCol, _ := app.FindCollectionByNameOrId("boqs")
	boqs, _ := app.FindAllRecords(boqsCol)
	if len(boqs) != 1 {
		t.Fatalf("expected 1 BOQ, got %d", len(boqs))
	}
	if boqs[0].GetString("project") != projects[0].Id {
		t.Errorf("BOQ project = %q, want %q", boqs[0].GetString("project"), projects[0].Id)
	}

	// Verify 3 main items
	mainItemsCol, _ := app.FindCollectionByNameOrId("main_boq_items")
	mainItems, _ := app.FindAllRecords(mainItemsCol)
	if len(mainItems) != 3 {
		t.Errorf("expected 3 main items, got %d", len(mainItems))
	}

	// Verify sub items exist
	subItemsCol, _ := app.FindCollectionByNameOrId("sub_items")
	subItems, _ := app.FindAllRecords(subItemsCol)
	if len(subItems) == 0 {
		t.Error("expected sub items to be created")
	}

	// Verify sub_sub_items exist (wall work and electrical have them)
	subSubItemsCol, _ := app.FindCollectionByNameOrId("sub_sub_items")
	subSubItems, _ := app.FindAllRecords(subSubItemsCol)
	if len(subSubItems) == 0 {
		t.Error("expected sub-sub items to be created")
	}
}

func TestSeed_Idempotent(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	if err := collections.Seed(app); err != nil {
		t.Fatalf("first Seed() error: %v", err)
	}
	if err := collections.Seed(app); err != nil {
		t.Fatalf("second Seed() error: %v", err)
	}

	// Should still have exactly 1 BOQ
	boqsCol, _ := app.FindCollectionByNameOrId("boqs")
	boqs, _ := app.FindAllRecords(boqsCol)
	if len(boqs) != 1 {
		t.Errorf("expected 1 BOQ after idempotent seed, got %d", len(boqs))
	}

	// Should still have exactly 1 project
	projectsCol, _ := app.FindCollectionByNameOrId("projects")
	projects, _ := app.FindAllRecords(projectsCol)
	if len(projects) != 1 {
		t.Errorf("expected 1 project after idempotent seed, got %d", len(projects))
	}
}

func TestSeed_MainItemDetails(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	if err := collections.Seed(app); err != nil {
		t.Fatalf("Seed() error: %v", err)
	}

	mainItemsCol, _ := app.FindCollectionByNameOrId("main_boq_items")
	items, _ := app.FindRecordsByFilter(
		mainItemsCol,
		"description = {:d}",
		"", 1, 0,
		map[string]any{"d": "Wall Partition & Finishing Work"},
	)
	if len(items) == 0 {
		t.Fatal("wall partition main item not found")
	}

	item := items[0]
	if item.GetInt("qty") != 450 {
		t.Errorf("qty = %v, want 450", item.GetInt("qty"))
	}
	if item.GetString("uom") != "sqft" {
		t.Errorf("uom = %q, want %q", item.GetString("uom"), "sqft")
	}
	if item.GetInt("gst_percent") != 18 {
		t.Errorf("gst_percent = %v, want 18", item.GetInt("gst_percent"))
	}
}

func TestSeed_SubItemHierarchy(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	if err := collections.Seed(app); err != nil {
		t.Fatalf("Seed() error: %v", err)
	}

	// Find "Installation Labour" sub item (should have sub-sub items)
	subItemsCol, _ := app.FindCollectionByNameOrId("sub_items")
	labourItems, _ := app.FindRecordsByFilter(
		subItemsCol,
		"description = {:d}",
		"", 1, 0,
		map[string]any{"d": "Installation Labour"},
	)
	if len(labourItems) == 0 {
		t.Fatal("Installation Labour sub item not found")
	}

	// Check it has sub-sub items
	subSubItemsCol, _ := app.FindCollectionByNameOrId("sub_sub_items")
	subSubs, _ := app.FindRecordsByFilter(
		subSubItemsCol,
		"sub_item = {:id}",
		"", 0, 0,
		map[string]any{"id": labourItems[0].Id},
	)
	if len(subSubs) != 2 {
		t.Errorf("expected 2 sub-sub items under Installation Labour, got %d", len(subSubs))
	}
}

func TestSeed_SkipsWhenDataExists(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	// Create a BOQ first (not via Seed)
	proj := testhelpers.CreateTestProject(t, app, "Existing Project")
	testhelpers.CreateTestBOQ(t, app, proj.Id, "Pre-existing BOQ")

	// Seed should skip because BOQ data already exists
	if err := collections.Seed(app); err != nil {
		t.Fatalf("Seed() error: %v", err)
	}

	// Should still have only the pre-existing data
	boqsCol, _ := app.FindCollectionByNameOrId("boqs")
	boqs, _ := app.FindAllRecords(boqsCol)
	if len(boqs) != 1 {
		t.Errorf("expected 1 BOQ (pre-existing only), got %d", len(boqs))
	}
	if boqs[0].GetString("title") != "Pre-existing BOQ" {
		t.Errorf("expected pre-existing BOQ, got %q", boqs[0].GetString("title"))
	}
}
