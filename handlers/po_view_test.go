package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandlePOView_FullDetails(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Full Details Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Precision Components Ltd")

	// Set bank details on vendor
	vendor.Set("bank_name", "HDFC Bank")
	vendor.Set("bank_account_no", "123456789")
	vendor.Set("bank_ifsc", "HDFC0001234")
	vendor.Set("bank_beneficiary_name", "Test Vendor")
	if err := app.Save(vendor); err != nil {
		t.Fatalf("failed to save vendor with bank details: %v", err)
	}

	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "PO-2026-FULL-001")
	testhelpers.CreateTestPOLineItem(t, app, po.Id, 1, "Steel Frame Assembly", 5, 1000, 18)
	testhelpers.CreateTestPOLineItem(t, app, po.Id, 2, "Copper Wiring Bundle", 10, 200, 12)

	handler := HandlePOView(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/po/"+po.Id, nil)
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
		"PO-2026-FULL-001",
		"Precision Components Ltd",
		"Steel Frame Assembly",
		"Copper Wiring Bundle",
		"PURCHASE ORDER",
	)
}

func TestHandlePOView_AddressMapping(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Address Mapping Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Mapping Vendor Co")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "PO-2026-ADDR-001")

	billFromAddr := testhelpers.CreateTestAddress(t, app, project.Id, "bill_from", "Our Billing Company")
	shipFromAddr := testhelpers.CreateTestAddress(t, app, project.Id, "ship_from", "Our Shipping Warehouse")

	// Link addresses to PO
	po.Set("bill_to_address", billFromAddr.Id)
	po.Set("ship_to_address", shipFromAddr.Id)
	if err := app.Save(po); err != nil {
		t.Fatalf("failed to save PO with address links: %v", err)
	}

	handler := HandlePOView(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/po/"+po.Id, nil)
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
		"Our Billing Company",
		"Our Shipping Warehouse",
	)
}

func TestHandlePOView_LineItemCalculations(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Line Item Calc Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Calc Vendor")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "PO-2026-CALC-001")

	// qty=10, rate=500, gst=18 → before=5000, gst=900, total=5900
	testhelpers.CreateTestPOLineItem(t, app, po.Id, 1, "High Voltage Cable", 10, 500, 18)

	handler := HandlePOView(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/po/"+po.Id, nil)
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
	// Before GST = 5000, GST Amount = 900, Total = 5900
	testhelpers.AssertHTMLContains(t, body,
		"5,000",
		"900",
		"5,900",
	)
}

func TestHandlePOView_GrandTotal(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Grand Total Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Grand Total Vendor")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "PO-2026-GRAND-001")

	// Item 1: qty=10, rate=100, gst=18 → before=1000, gst=180, total=1180
	testhelpers.CreateTestPOLineItem(t, app, po.Id, 1, "Item Alpha", 10, 100, 18)
	// Item 2: qty=5, rate=200, gst=18 → before=1000, gst=180, total=1180
	testhelpers.CreateTestPOLineItem(t, app, po.Id, 2, "Item Beta", 5, 200, 18)
	// Grand: before=2000, igst=360, grand=2360

	handler := HandlePOView(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/po/"+po.Id, nil)
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
		"2,000",
		"2,360",
	)
}

func TestHandlePOView_AmountInWords(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Amount In Words Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Words Vendor")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "PO-2026-WORDS-001")

	// qty=10, rate=500, gst=18 → grand total = 5900
	testhelpers.CreateTestPOLineItem(t, app, po.Id, 1, "Transformer Unit", 10, 500, 18)

	handler := HandlePOView(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/po/"+po.Id, nil)
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

	// Grand total = 5900 → "Five Thousand Nine Hundred Rupees Only/-"
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "Five Thousand Nine Hundred")
}

func TestHandlePOView_NotFound(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Not Found Project")

	handler := HandlePOView(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/po/nonexistent", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestHandlePOView_BankDetails(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Bank Details Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Acme Corp Vendor")

	// Set detailed bank fields on vendor
	vendor.Set("bank_beneficiary_name", "Acme Corp")
	vendor.Set("bank_name", "State Bank")
	vendor.Set("bank_account_no", "9876543210")
	vendor.Set("bank_ifsc", "SBIN0001234")
	vendor.Set("bank_branch", "Main Branch")
	if err := app.Save(vendor); err != nil {
		t.Fatalf("failed to save vendor with bank details: %v", err)
	}

	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "PO-2026-BANK-001")

	handler := HandlePOView(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/po/"+po.Id, nil)
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
		"Acme Corp",
		"State Bank",
		"9876543210",
		"SBIN0001234",
		"Main Branch",
	)
}
