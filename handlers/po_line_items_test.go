package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"projectcreation/testhelpers"
)

// ---------------------------------------------------------------------------
// Manual Line Item Tests
// ---------------------------------------------------------------------------

func TestHandlePOAddLineItem_Manual(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Test Vendor")
	testhelpers.LinkVendorToProject(t, app, project.Id, vendor.Id)
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-TEST/25-26/001")

	handler := HandlePOAddLineItem(app)

	form := url.Values{}
	form.Set("description", "Test Item")
	form.Set("hsn_code", "8504")
	form.Set("qty", "10")
	form.Set("uom", "Nos")
	form.Set("rate", "500")
	form.Set("gst_percent", "18")

	req := httptest.NewRequest(http.MethodPost, "/projects/"+project.Id+"/po/"+po.Id+"/line-items",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", po.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify line item was created in DB
	items, err := app.FindRecordsByFilter(
		"po_line_items",
		"purchase_order = {:poId}",
		"sort_order",
		0,
		0,
		map[string]any{"poId": po.Id},
	)
	if err != nil {
		t.Fatalf("failed to query po_line_items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 line item, got %d", len(items))
	}

	item := items[0]
	if got := item.GetString("description"); got != "Test Item" {
		t.Errorf("expected description %q, got %q", "Test Item", got)
	}
	if got := item.GetString("hsn_code"); got != "8504" {
		t.Errorf("expected hsn_code %q, got %q", "8504", got)
	}
	if got := item.GetFloat("qty"); got != 10 {
		t.Errorf("expected qty 10, got %v", got)
	}
	if got := item.GetString("uom"); got != "Nos" {
		t.Errorf("expected uom %q, got %q", "Nos", got)
	}
	if got := item.GetFloat("rate"); got != 500 {
		t.Errorf("expected rate 500, got %v", got)
	}
	if got := item.GetFloat("gst_percent"); got != 18 {
		t.Errorf("expected gst_percent 18, got %v", got)
	}
	if got := item.GetString("source_item_type"); got != "manual" {
		t.Errorf("expected source_item_type %q, got %q", "manual", got)
	}
}

func TestHandlePOAddLineItem_Manual_MissingDescription(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Test Vendor")
	testhelpers.LinkVendorToProject(t, app, project.Id, vendor.Id)
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-TEST/25-26/001")

	handler := HandlePOAddLineItem(app)

	form := url.Values{}
	form.Set("description", "")
	form.Set("hsn_code", "8504")
	form.Set("qty", "10")
	form.Set("uom", "Nos")
	form.Set("rate", "500")
	form.Set("gst_percent", "18")

	req := httptest.NewRequest(http.MethodPost, "/projects/"+project.Id+"/po/"+po.Id+"/line-items",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", po.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Validation failure re-renders the section partial (no HX-Redirect) with status 200.
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Header().Get("HX-Redirect") != "" {
		t.Error("expected no HX-Redirect for validation error")
	}

	// Verify NO line item was created
	items, err := app.FindRecordsByFilter(
		"po_line_items",
		"purchase_order = {:poId}",
		"",
		0,
		0,
		map[string]any{"poId": po.Id},
	)
	if err != nil {
		t.Fatalf("failed to query po_line_items: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected no line items to be created, got %d", len(items))
	}
}

func TestHandlePOAddLineItem_SortOrder(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Test Vendor")
	testhelpers.LinkVendorToProject(t, app, project.Id, vendor.Id)
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-TEST/25-26/001")

	handler := HandlePOAddLineItem(app)

	postItem := func(description string) {
		t.Helper()
		form := url.Values{}
		form.Set("description", description)
		form.Set("qty", "5")
		form.Set("uom", "Nos")
		form.Set("rate", "100")
		form.Set("gst_percent", "18")

		req := httptest.NewRequest(http.MethodPost, "/projects/"+project.Id+"/po/"+po.Id+"/line-items",
			strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")
		req.SetPathValue("projectId", project.Id)
		req.SetPathValue("id", po.Id)
		rec := httptest.NewRecorder()
		e := newTestRequestEvent(app, req, rec)

		if err := handler(e); err != nil {
			t.Fatalf("handler returned error for %q: %v", description, err)
		}
	}

	postItem("First Item")
	postItem("Second Item")

	items, err := app.FindRecordsByFilter(
		"po_line_items",
		"purchase_order = {:poId}",
		"sort_order",
		0,
		0,
		map[string]any{"poId": po.Id},
	)
	if err != nil {
		t.Fatalf("failed to query po_line_items: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 line items, got %d", len(items))
	}

	if got := items[0].GetInt("sort_order"); got != 1 {
		t.Errorf("first item: expected sort_order 1, got %d", got)
	}
	if got := items[1].GetInt("sort_order"); got != 2 {
		t.Errorf("second item: expected sort_order 2, got %d", got)
	}
}

// ---------------------------------------------------------------------------
// BOQ Line Item Tests
// ---------------------------------------------------------------------------

func TestHandlePOAddLineItemFromBOQ_MainItem(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Test Vendor")
	testhelpers.LinkVendorToProject(t, app, project.Id, vendor.Id)
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-TEST/25-26/001")
	boq := testhelpers.CreateTestBOQ(t, app, project.Id, "Main BOQ")
	mainItem := testhelpers.CreateTestMainBOQItem(t, app, boq.Id, "Main BOQ Item Description")

	handler := HandlePOAddLineItemFromBOQ(app)

	form := url.Values{}
	form.Set("source_item_type", "main_item")
	form.Set("source_item_id", mainItem.Id)

	req := httptest.NewRequest(http.MethodPost, "/projects/"+project.Id+"/po/"+po.Id+"/line-items/from-boq",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", po.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	items, err := app.FindRecordsByFilter(
		"po_line_items",
		"purchase_order = {:poId}",
		"sort_order",
		0,
		0,
		map[string]any{"poId": po.Id},
	)
	if err != nil {
		t.Fatalf("failed to query po_line_items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 line item, got %d", len(items))
	}

	item := items[0]
	if got := item.GetString("description"); got != "Main BOQ Item Description" {
		t.Errorf("expected description %q, got %q", "Main BOQ Item Description", got)
	}
	if got := item.GetString("source_item_type"); got != "main_item" {
		t.Errorf("expected source_item_type %q, got %q", "main_item", got)
	}
	if got := item.GetString("source_item_id"); got != mainItem.Id {
		t.Errorf("expected source_item_id %q, got %q", mainItem.Id, got)
	}
}

func TestHandlePOAddLineItemFromBOQ_SubItem(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Test Vendor")
	testhelpers.LinkVendorToProject(t, app, project.Id, vendor.Id)
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-TEST/25-26/001")
	boq := testhelpers.CreateTestBOQ(t, app, project.Id, "Sub BOQ")
	mainItem := testhelpers.CreateTestMainBOQItem(t, app, boq.Id, "Main Item")
	subItem := testhelpers.CreateTestSubItem(t, app, mainItem.Id, "Sub Item Description")

	handler := HandlePOAddLineItemFromBOQ(app)

	form := url.Values{}
	form.Set("source_item_type", "sub_item")
	form.Set("source_item_id", subItem.Id)

	req := httptest.NewRequest(http.MethodPost, "/projects/"+project.Id+"/po/"+po.Id+"/line-items/from-boq",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", po.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	items, err := app.FindRecordsByFilter(
		"po_line_items",
		"purchase_order = {:poId}",
		"sort_order",
		0,
		0,
		map[string]any{"poId": po.Id},
	)
	if err != nil {
		t.Fatalf("failed to query po_line_items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 line item, got %d", len(items))
	}

	item := items[0]
	if got := item.GetString("description"); got != "Sub Item Description" {
		t.Errorf("expected description %q, got %q", "Sub Item Description", got)
	}
	if got := item.GetString("source_item_type"); got != "sub_item" {
		t.Errorf("expected source_item_type %q, got %q", "sub_item", got)
	}
	if got := item.GetString("source_item_id"); got != subItem.Id {
		t.Errorf("expected source_item_id %q, got %q", subItem.Id, got)
	}
	// Sub items use qty_per_unit=5 per the test helper
	if got := item.GetFloat("qty"); got != 5 {
		t.Errorf("expected qty 5 (from qty_per_unit), got %v", got)
	}
}

func TestHandlePOAddLineItemFromBOQ_SubSubItem(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Test Vendor")
	testhelpers.LinkVendorToProject(t, app, project.Id, vendor.Id)
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-TEST/25-26/001")
	boq := testhelpers.CreateTestBOQ(t, app, project.Id, "SubSub BOQ")
	mainItem := testhelpers.CreateTestMainBOQItem(t, app, boq.Id, "Main Item")
	subItem := testhelpers.CreateTestSubItem(t, app, mainItem.Id, "Sub Item")
	subSubItem := testhelpers.CreateTestSubSubItem(t, app, subItem.Id, "Sub Sub Item Description")

	handler := HandlePOAddLineItemFromBOQ(app)

	form := url.Values{}
	form.Set("source_item_type", "sub_sub_item")
	form.Set("source_item_id", subSubItem.Id)

	req := httptest.NewRequest(http.MethodPost, "/projects/"+project.Id+"/po/"+po.Id+"/line-items/from-boq",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", po.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	items, err := app.FindRecordsByFilter(
		"po_line_items",
		"purchase_order = {:poId}",
		"sort_order",
		0,
		0,
		map[string]any{"poId": po.Id},
	)
	if err != nil {
		t.Fatalf("failed to query po_line_items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 line item, got %d", len(items))
	}

	item := items[0]
	if got := item.GetString("description"); got != "Sub Sub Item Description" {
		t.Errorf("expected description %q, got %q", "Sub Sub Item Description", got)
	}
	if got := item.GetString("source_item_type"); got != "sub_sub_item" {
		t.Errorf("expected source_item_type %q, got %q", "sub_sub_item", got)
	}
	if got := item.GetString("source_item_id"); got != subSubItem.Id {
		t.Errorf("expected source_item_id %q, got %q", subSubItem.Id, got)
	}
	// Sub-sub items use qty_per_unit=2 per the test helper
	if got := item.GetFloat("qty"); got != 2 {
		t.Errorf("expected qty 2 (from qty_per_unit), got %v", got)
	}
}

// ---------------------------------------------------------------------------
// Update and Delete Tests
// ---------------------------------------------------------------------------

func TestHandlePODeleteLineItem(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Test Vendor")
	testhelpers.LinkVendorToProject(t, app, project.Id, vendor.Id)
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-TEST/25-26/001")
	lineItem1 := testhelpers.CreateTestPOLineItem(t, app, po.Id, 1, "First Item", 10, 500, 18)
	lineItem2 := testhelpers.CreateTestPOLineItem(t, app, po.Id, 2, "Second Item", 5, 200, 18)

	handler := HandlePODeleteLineItem(app)

	req := httptest.NewRequest(http.MethodDelete,
		"/projects/"+project.Id+"/po/"+po.Id+"/line-items/"+lineItem1.Id, nil)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", po.Id)
	req.SetPathValue("itemId", lineItem1.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify first line item is gone
	_, err := app.FindRecordById("po_line_items", lineItem1.Id)
	if err == nil {
		t.Error("expected first line item to be deleted")
	}

	// Verify second line item still exists
	_, err = app.FindRecordById("po_line_items", lineItem2.Id)
	if err != nil {
		t.Errorf("expected second line item to still exist, got error: %v", err)
	}
}

func TestHandlePOUpdateLineItem(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Test Vendor")
	testhelpers.LinkVendorToProject(t, app, project.Id, vendor.Id)
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-TEST/25-26/001")
	lineItem := testhelpers.CreateTestPOLineItem(t, app, po.Id, 1, "Original Item", 10, 500, 18)

	handler := HandlePOUpdateLineItem(app)

	form := url.Values{}
	form.Set("qty", "20")
	form.Set("rate", "150.00")

	req := httptest.NewRequest(http.MethodPatch,
		"/projects/"+project.Id+"/po/"+po.Id+"/line-items/"+lineItem.Id,
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", po.Id)
	req.SetPathValue("itemId", lineItem.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify updated in DB
	updated, err := app.FindRecordById("po_line_items", lineItem.Id)
	if err != nil {
		t.Fatalf("failed to fetch updated line item: %v", err)
	}
	if got := updated.GetFloat("qty"); got != 20 {
		t.Errorf("expected qty 20, got %v", got)
	}
	if got := updated.GetFloat("rate"); got != 150 {
		t.Errorf("expected rate 150, got %v", got)
	}
}

func TestHandlePOUpdateLineItem_NotBelongsToPO(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Test Vendor")
	testhelpers.LinkVendorToProject(t, app, project.Id, vendor.Id)
	po1 := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-TEST/25-26/001")
	po2 := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-TEST/25-26/002")
	lineItemOnPO2 := testhelpers.CreateTestPOLineItem(t, app, po2.Id, 1, "PO2 Item", 10, 500, 18)

	handler := HandlePOUpdateLineItem(app)

	form := url.Values{}
	form.Set("qty", "20")
	form.Set("rate", "150.00")

	// PATCH to PO1's endpoint but using PO2's line item ID
	req := httptest.NewRequest(http.MethodPatch,
		"/projects/"+project.Id+"/po/"+po1.Id+"/line-items/"+lineItemOnPO2.Id,
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", po1.Id)
	req.SetPathValue("itemId", lineItemOnPO2.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// BOQ Picker Tests
// ---------------------------------------------------------------------------

func TestHandlePOBOQPicker_ShowsAllLevels(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Test Vendor")
	testhelpers.LinkVendorToProject(t, app, project.Id, vendor.Id)
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-TEST/25-26/001")
	boq := testhelpers.CreateTestBOQ(t, app, project.Id, "Electrical Works")
	mainItem := testhelpers.CreateTestMainBOQItem(t, app, boq.Id, "Panel Installation")
	subItem := testhelpers.CreateTestSubItem(t, app, mainItem.Id, "Cable Laying")
	testhelpers.CreateTestSubSubItem(t, app, subItem.Id, "Conduit Fitting")

	handler := HandlePOBOQPicker(app)

	req := httptest.NewRequest(http.MethodGet,
		"/projects/"+project.Id+"/po/"+po.Id+"/boq-picker", nil)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", po.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	testhelpers.AssertHTMLContains(t, body,
		"Electrical Works",
		"Panel Installation",
		"Cable Laying",
		"Conduit Fitting",
	)
}

func TestHandlePOBOQPicker_EmptyBOQ(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Test Vendor")
	testhelpers.LinkVendorToProject(t, app, project.Id, vendor.Id)
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-TEST/25-26/001")

	handler := HandlePOBOQPicker(app)

	req := httptest.NewRequest(http.MethodGet,
		"/projects/"+project.Id+"/po/"+po.Id+"/boq-picker", nil)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", po.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	testhelpers.AssertHTMLContains(t, rec.Body.String(), "No BOQ items found")
}
