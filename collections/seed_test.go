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

	// Verify 2 projects
	projectsCol, _ := app.FindCollectionByNameOrId("projects")
	projects, err := app.FindAllRecords(projectsCol)
	if err != nil {
		t.Fatalf("query projects error: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}

	// Verify 3 BOQs
	boqsCol, _ := app.FindCollectionByNameOrId("boqs")
	boqs, _ := app.FindAllRecords(boqsCol)
	if len(boqs) != 3 {
		t.Fatalf("expected 3 BOQs, got %d", len(boqs))
	}

	// Verify 18 main items (6 + 6 + 6)
	mainItemsCol, _ := app.FindCollectionByNameOrId("main_boq_items")
	mainItems, _ := app.FindAllRecords(mainItemsCol)
	if len(mainItems) != 18 {
		t.Errorf("expected 18 main items, got %d", len(mainItems))
	}

	// Verify sub items exist
	subItemsCol, _ := app.FindCollectionByNameOrId("sub_items")
	subItems, _ := app.FindAllRecords(subItemsCol)
	if len(subItems) == 0 {
		t.Error("expected sub items to be created")
	}

	// Verify sub_sub_items exist
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

	// Should still have exactly 2 projects
	projectsCol, _ := app.FindCollectionByNameOrId("projects")
	projects, _ := app.FindAllRecords(projectsCol)
	if len(projects) != 2 {
		t.Errorf("expected 2 projects after idempotent seed, got %d", len(projects))
	}

	// Should still have exactly 3 BOQs
	boqsCol, _ := app.FindCollectionByNameOrId("boqs")
	boqs, _ := app.FindAllRecords(boqsCol)
	if len(boqs) != 3 {
		t.Errorf("expected 3 BOQs after idempotent seed, got %d", len(boqs))
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
		map[string]any{"d": "Smart Classroom Hardware"},
	)
	if len(items) == 0 {
		t.Fatal("Smart Classroom Hardware main item not found")
	}

	item := items[0]
	if item.GetInt("qty") != 100 {
		t.Errorf("qty = %v, want 100", item.GetInt("qty"))
	}
	if item.GetString("uom") != "Nos" {
		t.Errorf("uom = %q, want %q", item.GetString("uom"), "Nos")
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

	// Find "Digital Podium Unit" sub item (should have 3 sub-sub items)
	subItemsCol, _ := app.FindCollectionByNameOrId("sub_items")
	podiumItems, _ := app.FindRecordsByFilter(
		subItemsCol,
		"description ~ {:d}",
		"", 1, 0,
		map[string]any{"d": "Digital Podium Unit"},
	)
	if len(podiumItems) == 0 {
		t.Fatal("Digital Podium Unit sub item not found")
	}

	// Check it has 3 sub-sub items
	subSubItemsCol, _ := app.FindCollectionByNameOrId("sub_sub_items")
	subSubs, _ := app.FindRecordsByFilter(
		subSubItemsCol,
		"sub_item = {:id}",
		"", 0, 0,
		map[string]any{"id": podiumItems[0].Id},
	)
	if len(subSubs) != 3 {
		t.Errorf("expected 3 sub-sub items under Digital Podium Unit, got %d", len(subSubs))
	}
}

func TestSeed_SkipsWhenDataExists(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	// Create a project first (not via Seed)
	testhelpers.CreateTestProject(t, app, "Existing Project")

	// Seed should skip because project data already exists
	if err := collections.Seed(app); err != nil {
		t.Fatalf("Seed() error: %v", err)
	}

	// Should still have only the pre-existing data
	projectsCol, _ := app.FindCollectionByNameOrId("projects")
	projects, _ := app.FindAllRecords(projectsCol)
	if len(projects) != 1 {
		t.Errorf("expected 1 project (pre-existing only), got %d", len(projects))
	}
	if projects[0].GetString("name") != "Existing Project" {
		t.Errorf("expected pre-existing project, got %q", projects[0].GetString("name"))
	}
}

func TestSeed_AddressesCreated(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	if err := collections.Seed(app); err != nil {
		t.Fatalf("Seed() error: %v", err)
	}

	addressesCol, _ := app.FindCollectionByNameOrId("addresses")
	addresses, _ := app.FindAllRecords(addressesCol)
	if len(addresses) != 9 {
		t.Errorf("expected 9 addresses, got %d", len(addresses))
	}

	// Verify address types
	typeCounts := map[string]int{}
	for _, a := range addresses {
		typeCounts[a.GetString("address_type")]++
	}
	if typeCounts["bill_from"] != 2 {
		t.Errorf("expected 2 bill_from addresses, got %d", typeCounts["bill_from"])
	}
	if typeCounts["bill_to"] != 2 {
		t.Errorf("expected 2 bill_to addresses, got %d", typeCounts["bill_to"])
	}
	if typeCounts["ship_to"] != 3 {
		t.Errorf("expected 3 ship_to addresses, got %d", typeCounts["ship_to"])
	}
	if typeCounts["install_at"] != 2 {
		t.Errorf("expected 2 install_at addresses, got %d", typeCounts["install_at"])
	}
}

func TestSeed_VendorsCreated(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	if err := collections.Seed(app); err != nil {
		t.Fatalf("Seed() error: %v", err)
	}

	vendorsCol, _ := app.FindCollectionByNameOrId("vendors")
	vendors, _ := app.FindAllRecords(vendorsCol)
	if len(vendors) != 6 {
		t.Errorf("expected 6 vendors, got %d", len(vendors))
	}

	// Verify project-vendor links
	pvCol, _ := app.FindCollectionByNameOrId("project_vendors")
	pvLinks, _ := app.FindAllRecords(pvCol)
	if len(pvLinks) != 6 {
		t.Errorf("expected 6 project-vendor links, got %d", len(pvLinks))
	}
}

func TestSeed_PurchaseOrdersCreated(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	if err := collections.Seed(app); err != nil {
		t.Fatalf("Seed() error: %v", err)
	}

	poCol, _ := app.FindCollectionByNameOrId("purchase_orders")
	pos, _ := app.FindAllRecords(poCol)
	if len(pos) != 4 {
		t.Errorf("expected 4 purchase orders, got %d", len(pos))
	}

	// Verify PO line items total: 3 + 3 + 4 + 3 = 13
	liCol, _ := app.FindCollectionByNameOrId("po_line_items")
	lineItems, _ := app.FindAllRecords(liCol)
	if len(lineItems) != 13 {
		t.Errorf("expected 13 PO line items, got %d", len(lineItems))
	}

	// Verify specific PO status
	sentPOs, _ := app.FindRecordsByFilter(poCol, "status = {:s}", "", 0, 0, map[string]any{"s": "sent"})
	if len(sentPOs) != 1 {
		t.Errorf("expected 1 PO with status 'sent', got %d", len(sentPOs))
	}
	ackPOs, _ := app.FindRecordsByFilter(poCol, "status = {:s}", "", 0, 0, map[string]any{"s": "acknowledged"})
	if len(ackPOs) != 1 {
		t.Errorf("expected 1 PO with status 'acknowledged', got %d", len(ackPOs))
	}
}

func TestSeed_InstallAtLinkedToShipTo(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	if err := collections.Seed(app); err != nil {
		t.Fatalf("Seed() error: %v", err)
	}

	addressesCol, _ := app.FindCollectionByNameOrId("addresses")
	installAddrs, _ := app.FindRecordsByFilter(
		addressesCol,
		"address_type = {:t}",
		"", 0, 0,
		map[string]any{"t": "install_at"},
	)
	if len(installAddrs) != 2 {
		t.Fatalf("expected 2 install_at addresses, got %d", len(installAddrs))
	}

	for _, addr := range installAddrs {
		parent := addr.GetString("ship_to_parent")
		if parent == "" {
			t.Errorf("install_at address %q missing ship_to_parent link", addr.GetString("company_name"))
		}
	}
}

func TestSeed_AddressSettingsCreated(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	if err := collections.Seed(app); err != nil {
		t.Fatalf("Seed() error: %v", err)
	}

	settingsCol, _ := app.FindCollectionByNameOrId("project_address_settings")
	settings, _ := app.FindAllRecords(settingsCol)
	if len(settings) != 4 {
		t.Errorf("expected 4 address settings records, got %d", len(settings))
	}

	// Each should have required fields set
	for _, s := range settings {
		if !s.GetBool("req_company_name") {
			t.Errorf("address setting for %s/%s: req_company_name should be true",
				s.GetString("project"), s.GetString("address_type"))
		}
		if !s.GetBool("req_gstin") {
			t.Errorf("address setting for %s/%s: req_gstin should be true",
				s.GetString("project"), s.GetString("address_type"))
		}
	}
}
