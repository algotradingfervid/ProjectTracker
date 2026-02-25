package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandleAddressDelete_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Delete Addr Project")
	addr := testhelpers.CreateTestAddress(t, app, project.Id, "bill_to", "Delete Me Corp")

	handler := HandleAddressDelete(app)

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+project.Id+"/addresses/bill-to/"+addr.Id, nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("type", "bill-to")
	req.SetPathValue("addressId", addr.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", rec.Code)
	}

	// Verify deleted
	if _, err := app.FindRecordById("addresses", addr.Id); err == nil {
		t.Error("address should have been deleted")
	}
}

func TestHandleAddressDelete_HTMX(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Delete HTMX Proj")
	addr := testhelpers.CreateTestAddress(t, app, project.Id, "ship_to", "HTMX Delete Corp")

	handler := HandleAddressDelete(app)

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+project.Id+"/addresses/ship-to/"+addr.Id, nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("type", "ship-to")
	req.SetPathValue("addressId", addr.Id)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	testhelpers.AssertHXRedirect(t, rec.Header().Get("HX-Redirect"), "/projects/"+project.Id+"/addresses/ship-to")
}

func TestHandleAddressDelete_NotFound(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Del NotFound Project")

	handler := HandleAddressDelete(app)

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+project.Id+"/addresses/bill-to/nonexistent", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("type", "bill-to")
	req.SetPathValue("addressId", "nonexistent")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleAddressDelete_WrongProject(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project1 := testhelpers.CreateTestProject(t, app, "Project One")
	project2 := testhelpers.CreateTestProject(t, app, "Project Two")
	addr := testhelpers.CreateTestAddress(t, app, project1.Id, "bill_to", "Wrong Proj Corp")

	handler := HandleAddressDelete(app)

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+project2.Id+"/addresses/bill-to/"+addr.Id, nil)
	req.SetPathValue("projectId", project2.Id)
	req.SetPathValue("type", "bill-to")
	req.SetPathValue("addressId", addr.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestHandleAddressDelete_InvalidType(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	handler := HandleAddressDelete(app)

	req := httptest.NewRequest(http.MethodDelete, "/test", nil)
	req.SetPathValue("projectId", "proj1")
	req.SetPathValue("type", "invalid-type")
	req.SetPathValue("addressId", "addr1")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleAddressDeleteInfo_ShipTo(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "DelInfo Project")
	addr := testhelpers.CreateTestAddress(t, app, project.Id, "ship_to", "Info Corp")

	handler := HandleAddressDeleteInfo(app)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetPathValue("type", "ship-to")
	req.SetPathValue("addressId", addr.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandleAddressDeleteInfo_NonShipTo(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	handler := HandleAddressDeleteInfo(app)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetPathValue("type", "bill-to")
	req.SetPathValue("addressId", "addr1")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandleAddressBulkDelete_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Bulk Del Project")
	addr1 := testhelpers.CreateTestAddress(t, app, project.Id, "bill_to", "Bulk A")
	addr2 := testhelpers.CreateTestAddress(t, app, project.Id, "bill_to", "Bulk B")

	handler := HandleAddressBulkDelete(app)

	body := `{"ids":["` + addr1.Id + `","` + addr2.Id + `"]}`
	req := httptest.NewRequest(http.MethodDelete, "/projects/"+project.Id+"/addresses/bill-to/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("type", "bill-to")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandleAddressBulkDelete_EmptyIDs(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Bulk Empty Project")

	handler := HandleAddressBulkDelete(app)

	body := `{"ids":[]}`
	req := httptest.NewRequest(http.MethodDelete, "/projects/"+project.Id+"/addresses/bill-to/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("type", "bill-to")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestNullifyLinkedInstallAtAddresses(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Nullify Project")
	shipTo := testhelpers.CreateTestAddress(t, app, project.Id, "ship_to", "Parent Ship")

	// Create install_at linked to ship_to
	col, _ := app.FindCollectionByNameOrId("addresses")
	installAt := testhelpers.CreateTestAddress(t, app, project.Id, "install_at", "Linked Install")
	installAt.Set("ship_to_parent", shipTo.Id)
	app.Save(installAt)

	if err := nullifyLinkedInstallAtAddresses(app, shipTo.Id); err != nil {
		t.Fatalf("nullify failed: %v", err)
	}

	// Verify parent was cleared
	_ = col
	updated, _ := app.FindRecordById("addresses", installAt.Id)
	if updated.GetString("ship_to_parent") != "" {
		t.Error("ship_to_parent should be empty after nullify")
	}
}
