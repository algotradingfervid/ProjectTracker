package collections_test

import (
	"testing"

	"projectcreation/collections"
	"projectcreation/testhelpers"

	"github.com/pocketbase/pocketbase/core"
)

// expectedCollections is the full list of collections that Setup() must create.
var expectedCollections = []string{
	"projects",
	"boqs",
	"main_boq_items",
	"sub_items",
	"sub_sub_items",
	"addresses",
	"project_address_settings",
	"vendors",
	"project_vendors",
	"purchase_orders",
	"po_line_items",
}

func TestSetup_AllCollectionsExist(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	for _, name := range expectedCollections {
		col, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			t.Errorf("collection %q not found after Setup(): %v", name, err)
			continue
		}
		if col.Name != name {
			t.Errorf("expected collection name %q, got %q", name, col.Name)
		}
	}
}

func TestSetup_Idempotent(t *testing.T) {
	app := testhelpers.NewTestApp(t) // Setup() already called once via NewTestApp

	// Collect IDs from first run
	ids := make(map[string]string)
	for _, name := range expectedCollections {
		col, _ := app.FindCollectionByNameOrId(name)
		ids[name] = col.Id
	}

	// Run Setup() again
	collections.Setup(app)

	// IDs should not change
	for _, name := range expectedCollections {
		col, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			t.Errorf("collection %q missing after second Setup(): %v", name, err)
			continue
		}
		if col.Id != ids[name] {
			t.Errorf("collection %q id changed after second Setup(): %s -> %s", name, ids[name], col.Id)
		}
	}
}

func TestSetup_ProjectsFields(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	col, _ := app.FindCollectionByNameOrId("projects")

	requiredFields := []string{"name", "status"}
	optionalFields := []string{"client_name", "reference_number", "ship_to_equals_install_at", "created", "updated"}

	for _, f := range requiredFields {
		if col.Fields.GetByName(f) == nil {
			t.Errorf("projects: missing required field %q", f)
		}
	}
	for _, f := range optionalFields {
		if col.Fields.GetByName(f) == nil {
			t.Errorf("projects: missing field %q", f)
		}
	}

	// Verify status is a select field with expected values
	statusField := col.Fields.GetByName("status")
	if sf, ok := statusField.(*core.SelectField); ok {
		expected := map[string]bool{"active": true, "completed": true, "on_hold": true}
		for _, v := range sf.Values {
			if !expected[v] {
				t.Errorf("unexpected status value: %q", v)
			}
			delete(expected, v)
		}
		for v := range expected {
			t.Errorf("missing status value: %q", v)
		}
	} else {
		t.Errorf("status field is not a SelectField")
	}
}

func TestSetup_BOQsFields(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	col, _ := app.FindCollectionByNameOrId("boqs")

	fields := []string{"title", "reference_number", "project", "created", "updated"}
	for _, f := range fields {
		if col.Fields.GetByName(f) == nil {
			t.Errorf("boqs: missing field %q", f)
		}
	}

	// Project relation should exist
	projectField := col.Fields.GetByName("project")
	if rf, ok := projectField.(*core.RelationField); ok {
		if rf.MaxSelect != 1 {
			t.Errorf("boqs.project: expected MaxSelect=1, got %d", rf.MaxSelect)
		}
	} else {
		t.Errorf("boqs.project is not a RelationField")
	}
}

func TestSetup_MainBOQItemsFields(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	col, _ := app.FindCollectionByNameOrId("main_boq_items")

	fields := []string{"boq", "sort_order", "description", "qty", "uom", "unit_price", "quoted_price", "budgeted_price", "hsn_code", "gst_percent"}
	for _, f := range fields {
		if col.Fields.GetByName(f) == nil {
			t.Errorf("main_boq_items: missing field %q", f)
		}
	}

	// boq relation with cascade delete
	boqField := col.Fields.GetByName("boq")
	if rf, ok := boqField.(*core.RelationField); ok {
		if !rf.CascadeDelete {
			t.Error("main_boq_items.boq: expected CascadeDelete=true")
		}
	}
}

func TestSetup_SubItemsFields(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	col, _ := app.FindCollectionByNameOrId("sub_items")

	fields := []string{"main_item", "sort_order", "type", "description", "qty_per_unit", "uom", "unit_price", "budgeted_price", "hsn_code", "gst_percent"}
	for _, f := range fields {
		if col.Fields.GetByName(f) == nil {
			t.Errorf("sub_items: missing field %q", f)
		}
	}

	// type field should have product/service values
	typeField := col.Fields.GetByName("type")
	if sf, ok := typeField.(*core.SelectField); ok {
		if len(sf.Values) != 2 {
			t.Errorf("sub_items.type: expected 2 values, got %d", len(sf.Values))
		}
	}

	// main_item with cascade delete
	mainItemField := col.Fields.GetByName("main_item")
	if rf, ok := mainItemField.(*core.RelationField); ok {
		if !rf.CascadeDelete {
			t.Error("sub_items.main_item: expected CascadeDelete=true")
		}
	}
}

func TestSetup_SubSubItemsFields(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	col, _ := app.FindCollectionByNameOrId("sub_sub_items")

	fields := []string{"sub_item", "sort_order", "type", "description", "qty_per_unit", "uom", "unit_price", "budgeted_price", "hsn_code", "gst_percent"}
	for _, f := range fields {
		if col.Fields.GetByName(f) == nil {
			t.Errorf("sub_sub_items: missing field %q", f)
		}
	}

	// sub_item with cascade delete
	subItemField := col.Fields.GetByName("sub_item")
	if rf, ok := subItemField.(*core.RelationField); ok {
		if !rf.CascadeDelete {
			t.Error("sub_sub_items.sub_item: expected CascadeDelete=true")
		}
	}
}

func TestSetup_AddressesFields(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	col, _ := app.FindCollectionByNameOrId("addresses")

	fields := []string{
		"address_type", "project", "company_name", "contact_person",
		"address_line_1", "address_line_2", "city", "state", "pin_code",
		"country", "landmark", "district", "phone", "email", "fax",
		"website", "gstin", "pan", "cin", "ship_to_parent",
		"created", "updated",
	}
	for _, f := range fields {
		if col.Fields.GetByName(f) == nil {
			t.Errorf("addresses: missing field %q", f)
		}
	}

	// address_type should be select with 5 types
	atField := col.Fields.GetByName("address_type")
	if sf, ok := atField.(*core.SelectField); ok {
		if len(sf.Values) != 5 {
			t.Errorf("addresses.address_type: expected 5 values, got %d", len(sf.Values))
		}
	}

	// project relation with cascade delete
	projectField := col.Fields.GetByName("project")
	if rf, ok := projectField.(*core.RelationField); ok {
		if !rf.CascadeDelete {
			t.Error("addresses.project: expected CascadeDelete=true")
		}
	}

	// ship_to_parent is self-referencing
	stpField := col.Fields.GetByName("ship_to_parent")
	if rf, ok := stpField.(*core.RelationField); ok {
		if rf.CollectionId != col.Id {
			t.Error("addresses.ship_to_parent: expected self-referencing relation")
		}
	}
}

func TestSetup_ProjectAddressSettingsFields(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	col, _ := app.FindCollectionByNameOrId("project_address_settings")

	reqFields := []string{
		"project", "address_type",
		"req_company_name", "req_contact_person",
		"req_address_line_1", "req_address_line_2",
		"req_city", "req_state", "req_pin_code", "req_country",
		"req_landmark", "req_district",
		"req_phone", "req_email", "req_fax", "req_website",
		"req_gstin", "req_pan", "req_cin",
		"created", "updated",
	}
	for _, f := range reqFields {
		if col.Fields.GetByName(f) == nil {
			t.Errorf("project_address_settings: missing field %q", f)
		}
	}
}

func TestSetup_VendorsFields(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	col, _ := app.FindCollectionByNameOrId("vendors")

	fields := []string{
		"name", "address_line_1", "address_line_2", "city", "state", "pin_code",
		"country", "gstin", "pan", "contact_name", "phone", "email", "website",
		"bank_beneficiary_name", "bank_name", "bank_account_no", "bank_ifsc",
		"bank_branch", "notes", "created", "updated",
	}
	for _, f := range fields {
		if col.Fields.GetByName(f) == nil {
			t.Errorf("vendors: missing field %q", f)
		}
	}
}

func TestSetup_ProjectVendorsFields(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	col, _ := app.FindCollectionByNameOrId("project_vendors")

	fields := []string{"project", "vendor", "created"}
	for _, f := range fields {
		if col.Fields.GetByName(f) == nil {
			t.Errorf("project_vendors: missing field %q", f)
		}
	}

	// Both relations should cascade delete
	for _, relName := range []string{"project", "vendor"} {
		field := col.Fields.GetByName(relName)
		if rf, ok := field.(*core.RelationField); ok {
			if !rf.CascadeDelete {
				t.Errorf("project_vendors.%s: expected CascadeDelete=true", relName)
			}
		}
	}
}

func TestSetup_PurchaseOrdersFields(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	col, _ := app.FindCollectionByNameOrId("purchase_orders")

	fields := []string{
		"po_number", "order_date", "quotation_ref", "ref_date",
		"payment_terms", "delivery_terms", "warranty_terms", "comments",
		"status", "project", "vendor", "bill_to_address", "ship_to_address",
		"created", "updated",
	}
	for _, f := range fields {
		if col.Fields.GetByName(f) == nil {
			t.Errorf("purchase_orders: missing field %q", f)
		}
	}

	// status select field
	statusField := col.Fields.GetByName("status")
	if sf, ok := statusField.(*core.SelectField); ok {
		expected := []string{"draft", "sent", "acknowledged", "completed", "cancelled"}
		if len(sf.Values) != len(expected) {
			t.Errorf("purchase_orders.status: expected %d values, got %d", len(expected), len(sf.Values))
		}
	}
}

func TestSetup_POLineItemsFields(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	col, _ := app.FindCollectionByNameOrId("po_line_items")

	fields := []string{
		"purchase_order", "sort_order", "description", "hsn_code",
		"qty", "uom", "rate", "gst_percent",
		"source_item_type", "source_item_id",
		"created", "updated",
	}
	for _, f := range fields {
		if col.Fields.GetByName(f) == nil {
			t.Errorf("po_line_items: missing field %q", f)
		}
	}

	// purchase_order with cascade delete
	poField := col.Fields.GetByName("purchase_order")
	if rf, ok := poField.(*core.RelationField); ok {
		if !rf.CascadeDelete {
			t.Error("po_line_items.purchase_order: expected CascadeDelete=true")
		}
	}
}

func TestSetup_CascadeDeleteHierarchy(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	// Create full hierarchy: project -> boq -> main_item -> sub_item -> sub_sub_item
	proj := testhelpers.CreateTestProject(t, app, "Cascade Test")
	boq := testhelpers.CreateTestBOQ(t, app, proj.Id, "Cascade BOQ")
	mainItem := testhelpers.CreateTestMainBOQItem(t, app, boq.Id, "Main Item")
	subItem := testhelpers.CreateTestSubItem(t, app, mainItem.Id, "Sub Item")
	subSubItem := testhelpers.CreateTestSubSubItem(t, app, subItem.Id, "Sub Sub Item")

	// Delete the BOQ — should cascade delete main -> sub -> sub_sub
	if err := app.Delete(boq); err != nil {
		t.Fatalf("failed to delete BOQ: %v", err)
	}

	// Verify all child records are deleted
	_, err := app.FindRecordById("main_boq_items", mainItem.Id)
	if err == nil {
		t.Error("main_boq_item should have been cascade-deleted")
	}
	_, err = app.FindRecordById("sub_items", subItem.Id)
	if err == nil {
		t.Error("sub_item should have been cascade-deleted")
	}
	_, err = app.FindRecordById("sub_sub_items", subSubItem.Id)
	if err == nil {
		t.Error("sub_sub_item should have been cascade-deleted")
	}
}

func TestSetup_AddressCascadeDeleteOnProject(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	proj := testhelpers.CreateTestProject(t, app, "Address Cascade")
	addr := testhelpers.CreateTestAddress(t, app, proj.Id, "bill_to", "Test Corp")

	if err := app.Delete(proj); err != nil {
		t.Fatalf("failed to delete project: %v", err)
	}

	_, err := app.FindRecordById("addresses", addr.Id)
	if err == nil {
		t.Error("address should have been cascade-deleted with project")
	}
}

func TestSetup_POCascadeDeleteOnProject(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	proj := testhelpers.CreateTestProject(t, app, "PO Cascade")
	vendor := testhelpers.CreateTestVendor(t, app, "Vendor A")
	po := testhelpers.CreateTestPurchaseOrder(t, app, proj.Id, vendor.Id, "PO-001")
	lineItem := testhelpers.CreateTestPOLineItem(t, app, po.Id, 1, "Item 1", 10, 100, 18)

	if err := app.Delete(proj); err != nil {
		t.Fatalf("failed to delete project: %v", err)
	}

	_, err := app.FindRecordById("purchase_orders", po.Id)
	if err == nil {
		t.Error("purchase_order should have been cascade-deleted with project")
	}
	_, err = app.FindRecordById("po_line_items", lineItem.Id)
	if err == nil {
		t.Error("po_line_item should have been cascade-deleted with purchase_order")
	}
}
