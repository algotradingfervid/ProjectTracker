package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandlePOCreate_GET(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Acme Supplies")
	testhelpers.LinkVendorToProject(t, app, project.Id, vendor.Id)

	handler := HandlePOCreate(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/po/create", nil)
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "Acme Supplies")
}

func TestHandlePOCreate_GET_AddressMappingCorrect(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Address Mapping Project")

	testhelpers.CreateTestAddress(t, app, project.Id, "bill_from", "BillFrom Corp A")
	testhelpers.CreateTestAddress(t, app, project.Id, "bill_from", "BillFrom Corp B")
	testhelpers.CreateTestAddress(t, app, project.Id, "ship_from", "ShipFrom Warehouse X")
	testhelpers.CreateTestAddress(t, app, project.Id, "ship_from", "ShipFrom Warehouse Y")

	handler := HandlePOCreate(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/po/create", nil)
	req.SetPathValue("projectId", project.Id)
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
		"BillFrom Corp A",
		"BillFrom Corp B",
		"ShipFrom Warehouse X",
		"ShipFrom Warehouse Y",
	)
}

func TestHandlePOCreate_GET_AddressMapping_NoBillFrom(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "No Bill From Project")

	// Only a ship_from address — should not cause bill_from section to show an address
	testhelpers.CreateTestAddress(t, app, project.Id, "ship_from", "ShipFrom Only Co")

	handler := HandlePOCreate(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/po/create", nil)
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "No Bill From addresses configured")
}

func TestHandlePOCreate_GET_AddressMapping_NoShipFrom(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "No Ship From Project")

	// Only a bill_from address — should not cause ship_from section to show an address
	testhelpers.CreateTestAddress(t, app, project.Id, "bill_from", "BillFrom Only Co")

	handler := HandlePOCreate(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/po/create", nil)
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "No Ship From addresses configured")
}

func TestHandlePOCreate_GET_AddressMapping_DoesNotShowBillTo(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Bill To Exclusion Project")

	testhelpers.CreateTestAddress(t, app, project.Id, "bill_from", "Our Company LLC")
	testhelpers.CreateTestAddress(t, app, project.Id, "bill_to", "Client Corp — Should Not Appear")

	handler := HandlePOCreate(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/po/create", nil)
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	body := rec.Body.String()
	testhelpers.AssertHTMLContains(t, body, "Our Company LLC")

	if strings.Contains(body, "Client Corp — Should Not Appear") {
		t.Error("expected bill_to address company name NOT to appear in the form, but it was found")
	}
}

func TestHandlePOCreate_GET_AddressMapping_DoesNotShowShipTo(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Ship To Exclusion Project")

	testhelpers.CreateTestAddress(t, app, project.Id, "ship_from", "Our Warehouse Ltd")
	testhelpers.CreateTestAddress(t, app, project.Id, "ship_to", "Client Site — Should Not Appear")

	handler := HandlePOCreate(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/po/create", nil)
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	body := rec.Body.String()
	testhelpers.AssertHTMLContains(t, body, "Our Warehouse Ltd")

	if strings.Contains(body, "Client Site — Should Not Appear") {
		t.Error("expected ship_to address company name NOT to appear in the form, but it was found")
	}
}

func TestHandlePOCreate_GET_NoLinkedVendors(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "No Vendor Project")

	handler := HandlePOCreate(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/po/create", nil)
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "No vendors linked")
}

func TestHandlePOSave_Valid(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Valid PO Project")
	project.Set("reference_number", "VALID-REF")
	if err := app.Save(project); err != nil {
		t.Fatalf("failed to update project reference_number: %v", err)
	}

	vendor := testhelpers.CreateTestVendor(t, app, "Reliable Vendor")
	testhelpers.LinkVendorToProject(t, app, project.Id, vendor.Id)

	handler := HandlePOSave(app)

	form := url.Values{}
	form.Set("vendor_id", vendor.Id)

	req := httptest.NewRequest(http.MethodPost, "/projects/"+project.Id+"/po",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	redirectURL := rec.Header().Get("HX-Redirect")
	expectedPrefix := "/projects/" + project.Id + "/po/"
	if !strings.HasPrefix(redirectURL, expectedPrefix) {
		t.Errorf("expected HX-Redirect to start with %q, got %q", expectedPrefix, redirectURL)
	}

	// Verify PO was saved in DB
	pos, err := app.FindRecordsByFilter("purchase_orders",
		"project = {:projectId}", "", 0, 0,
		map[string]any{"projectId": project.Id})
	if err != nil || len(pos) == 0 {
		t.Fatal("expected purchase order to be created in database")
	}

	// Verify status is draft
	if pos[0].GetString("status") != "draft" {
		t.Errorf("expected PO status %q, got %q", "draft", pos[0].GetString("status"))
	}
}

func TestHandlePOSave_MissingVendor(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Missing Vendor Project")

	handler := HandlePOSave(app)

	form := url.Values{}
	form.Set("vendor_id", "")

	req := httptest.NewRequest(http.MethodPost, "/projects/"+project.Id+"/po",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Should re-render form, not redirect
	if rec.Header().Get("HX-Redirect") != "" {
		t.Error("expected no HX-Redirect for validation error")
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "Vendor is required")
}

func TestHandlePOSave_PONumberGenerated(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "PO Number Test Project")
	project.Set("reference_number", "TEST-REF")
	if err := app.Save(project); err != nil {
		t.Fatalf("failed to update project reference_number: %v", err)
	}

	vendor := testhelpers.CreateTestVendor(t, app, "Number Gen Vendor")
	testhelpers.LinkVendorToProject(t, app, project.Id, vendor.Id)

	handler := HandlePOSave(app)

	form := url.Values{}
	form.Set("vendor_id", vendor.Id)

	req := httptest.NewRequest(http.MethodPost, "/projects/"+project.Id+"/po",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Query the saved PO
	pos, err := app.FindRecordsByFilter("purchase_orders",
		"project = {:projectId}", "", 0, 0,
		map[string]any{"projectId": project.Id})
	if err != nil || len(pos) == 0 {
		t.Fatal("expected purchase order to be created in database")
	}

	poNumber := pos[0].GetString("po_number")
	expectedPrefix := "FSS-PO-TEST-REF-"
	if !strings.HasPrefix(poNumber, expectedPrefix) {
		t.Errorf("expected po_number to start with %q, got %q", expectedPrefix, poNumber)
	}
}

func TestHandlePOSave_PONumberSequential(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Sequential PO Project")
	project.Set("reference_number", "SEQ-TEST")
	if err := app.Save(project); err != nil {
		t.Fatalf("failed to update project reference_number: %v", err)
	}

	vendor := testhelpers.CreateTestVendor(t, app, "Sequential Vendor")
	testhelpers.LinkVendorToProject(t, app, project.Id, vendor.Id)

	handler := HandlePOSave(app)

	postPO := func() {
		t.Helper()
		form := url.Values{}
		form.Set("vendor_id", vendor.Id)

		req := httptest.NewRequest(http.MethodPost, "/projects/"+project.Id+"/po",
			strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")
		req.SetPathValue("projectId", project.Id)
		rec := httptest.NewRecorder()
		e := newTestRequestEvent(app, req, rec)

		if err := handler(e); err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	}

	// Post first PO
	postPO()
	// Post second PO
	postPO()

	// Query all POs for the project, ordered by creation time
	pos, err := app.FindRecordsByFilter("purchase_orders",
		"project = {:projectId}", "created", 0, 0,
		map[string]any{"projectId": project.Id})
	if err != nil {
		t.Fatalf("failed to query purchase orders: %v", err)
	}
	if len(pos) != 2 {
		t.Fatalf("expected 2 purchase orders, got %d", len(pos))
	}

	firstPONumber := pos[0].GetString("po_number")
	secondPONumber := pos[1].GetString("po_number")

	if !strings.HasSuffix(firstPONumber, "-001") {
		t.Errorf("expected first PO number to end with -001, got %q", firstPONumber)
	}
	if !strings.HasSuffix(secondPONumber, "-002") {
		t.Errorf("expected second PO number to end with -002, got %q", secondPONumber)
	}
}
