// Package testhelpers provides utilities for testing PocketBase-based applications.
package testhelpers

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/collections"
)

// NewTestApp creates a PocketBase instance backed by a temporary directory.
// It bootstraps the app and runs collections.Setup to create all tables.
// The temporary directory is cleaned up automatically when the test finishes.
func NewTestApp(t *testing.T) *pocketbase.PocketBase {
	t.Helper()

	tmpDir := t.TempDir()
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: tmpDir,
	})

	if err := app.Bootstrap(); err != nil {
		t.Fatalf("failed to bootstrap test app: %v", err)
	}

	collections.Setup(app)

	return app
}

// CreateTestProject creates a project record with the given name and returns it.
func CreateTestProject(t *testing.T, app *pocketbase.PocketBase, name string) *core.Record {
	t.Helper()

	col, err := app.FindCollectionByNameOrId("projects")
	if err != nil {
		t.Fatalf("failed to find projects collection: %v", err)
	}

	record := core.NewRecord(col)
	record.Set("name", name)
	record.Set("status", "active")
	record.Set("ship_to_equals_install_at", true)

	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save test project: %v", err)
	}

	return record
}

// CreateTestBOQ creates a BOQ record linked to a project and returns it.
func CreateTestBOQ(t *testing.T, app *pocketbase.PocketBase, projectID, title string) *core.Record {
	t.Helper()

	col, err := app.FindCollectionByNameOrId("boqs")
	if err != nil {
		t.Fatalf("failed to find boqs collection: %v", err)
	}

	record := core.NewRecord(col)
	record.Set("title", title)
	record.Set("project", projectID)

	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save test BOQ: %v", err)
	}

	return record
}

// AssertHTMLContains checks that body contains all specified fragments.
func AssertHTMLContains(t *testing.T, body string, fragments ...string) {
	t.Helper()

	for _, frag := range fragments {
		if !strings.Contains(body, frag) {
			t.Errorf("expected HTML to contain %q, but it was not found\nbody (first 500 chars): %s",
				frag, truncate(body, 500))
		}
	}
}

// AssertHXRedirect checks that the response has an HX-Redirect header with the expected URL.
func AssertHXRedirect(t *testing.T, headerVal, expectedURL string) {
	t.Helper()

	if headerVal != expectedURL {
		t.Errorf("expected HX-Redirect %q, got %q", expectedURL, headerVal)
	}
}

// CreateTestVendor creates a vendor record with the given name and returns it.
func CreateTestVendor(t *testing.T, app *pocketbase.PocketBase, name string) *core.Record {
	t.Helper()

	col, err := app.FindCollectionByNameOrId("vendors")
	if err != nil {
		t.Fatalf("failed to find vendors collection: %v", err)
	}

	record := core.NewRecord(col)
	record.Set("name", name)
	record.Set("city", "Mumbai")
	record.Set("gstin", "27AADCB2230M1ZV")
	record.Set("contact_name", "Test Contact")
	record.Set("phone", "9876543210")

	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save test vendor: %v", err)
	}

	return record
}

// CreateTestPurchaseOrder creates a PO record linked to a project and vendor.
func CreateTestPurchaseOrder(t *testing.T, app *pocketbase.PocketBase, projectID, vendorID, poNumber string) *core.Record {
	t.Helper()

	col, err := app.FindCollectionByNameOrId("purchase_orders")
	if err != nil {
		t.Fatalf("failed to find purchase_orders collection: %v", err)
	}

	record := core.NewRecord(col)
	record.Set("project", projectID)
	record.Set("vendor", vendorID)
	record.Set("po_number", poNumber)
	record.Set("status", "draft")

	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save test PO: %v", err)
	}

	return record
}

// CreateTestAddress creates an address record linked to a project.
func CreateTestAddress(t *testing.T, app *pocketbase.PocketBase, projectID, addressType, companyName string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("addresses")
	if err != nil {
		t.Fatalf("failed to find addresses collection: %v", err)
	}
	record := core.NewRecord(col)
	record.Set("project", projectID)
	record.Set("address_type", addressType)
	record.Set("company_name", companyName)
	record.Set("address_line_1", "123 Test Street")
	record.Set("city", "Mumbai")
	record.Set("state", "Maharashtra")
	record.Set("pin_code", "400001")
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save test address: %v", err)
	}
	return record
}

// LinkVendorToProject creates a project_vendors link record.
func LinkVendorToProject(t *testing.T, app *pocketbase.PocketBase, projectID, vendorID string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("project_vendors")
	if err != nil {
		t.Fatalf("failed to find project_vendors collection: %v", err)
	}
	record := core.NewRecord(col)
	record.Set("project", projectID)
	record.Set("vendor", vendorID)
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save project-vendor link: %v", err)
	}
	return record
}

// CreateTestPOLineItem creates a PO line item record.
func CreateTestPOLineItem(t *testing.T, app *pocketbase.PocketBase, poID string, sortOrder int, description string, qty, rate, gstPercent float64) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("po_line_items")
	if err != nil {
		t.Fatalf("failed to find po_line_items collection: %v", err)
	}
	record := core.NewRecord(col)
	record.Set("purchase_order", poID)
	record.Set("sort_order", sortOrder)
	record.Set("description", description)
	record.Set("hsn_code", "8504")
	record.Set("qty", qty)
	record.Set("uom", "Nos")
	record.Set("rate", rate)
	record.Set("gst_percent", gstPercent)
	record.Set("source_item_type", "manual")
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save test PO line item: %v", err)
	}
	return record
}

// CreateTestMainBOQItem creates a main BOQ item for BOQ picker tests.
func CreateTestMainBOQItem(t *testing.T, app *pocketbase.PocketBase, boqID, description string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("main_boq_items")
	if err != nil {
		t.Fatalf("failed to find main_boq_items collection: %v", err)
	}
	record := core.NewRecord(col)
	record.Set("boq", boqID)
	record.Set("description", description)
	record.Set("hsn_code", "8504")
	record.Set("qty", 10)
	record.Set("uom", "Nos")
	record.Set("unit_price", 500.0)
	record.Set("gst_percent", 18.0)
	record.Set("sort_order", 1)
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save test main BOQ item: %v", err)
	}
	return record
}

// CreateTestSubItem creates a sub item under a main BOQ item.
func CreateTestSubItem(t *testing.T, app *pocketbase.PocketBase, mainItemID, description string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("sub_items")
	if err != nil {
		t.Fatalf("failed to find sub_items collection: %v", err)
	}
	record := core.NewRecord(col)
	record.Set("main_item", mainItemID)
	record.Set("type", "product")
	record.Set("description", description)
	record.Set("hsn_code", "8504")
	record.Set("qty_per_unit", 5)
	record.Set("uom", "Mtrs")
	record.Set("unit_price", 200.0)
	record.Set("gst_percent", 18.0)
	record.Set("sort_order", 1)
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save test sub item: %v", err)
	}
	return record
}

// CreateTestSubSubItem creates a sub-sub item under a sub item.
func CreateTestSubSubItem(t *testing.T, app *pocketbase.PocketBase, subItemID, description string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("sub_sub_items")
	if err != nil {
		t.Fatalf("failed to find sub_sub_items collection: %v", err)
	}
	record := core.NewRecord(col)
	record.Set("sub_item", subItemID)
	record.Set("type", "product")
	record.Set("description", description)
	record.Set("hsn_code", "8504")
	record.Set("qty_per_unit", 2)
	record.Set("uom", "Nos")
	record.Set("unit_price", 100.0)
	record.Set("gst_percent", 18.0)
	record.Set("sort_order", 1)
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save test sub-sub item: %v", err)
	}
	return record
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
