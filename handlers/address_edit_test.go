package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandleAddressEdit_GET(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Edit Addr Project")
	addr := testhelpers.CreateTestAddress(t, app, project.Id, "bill_to", "Edit Corp")

	handler := HandleAddressEdit(app, AddressTypeBillTo)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/addresses/bill-to/"+addr.Id+"/edit", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("addressId", addr.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "Edit Corp")
}

func TestHandleAddressEdit_GET_HTMX(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Edit HTMX Project")
	addr := testhelpers.CreateTestAddress(t, app, project.Id, "ship_to", "HTMX Edit Corp")

	handler := HandleAddressEdit(app, AddressTypeShipTo)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetPathValue("projectId", project.Id)
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
}

func TestHandleAddressEdit_NotFound(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Edit NF Project")

	handler := HandleAddressEdit(app, AddressTypeBillTo)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("addressId", "nonexistent")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != 404 {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleAddressEdit_WrongType(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Wrong Type Project")
	addr := testhelpers.CreateTestAddress(t, app, project.Id, "bill_to", "Wrong Type Corp")

	// Try to edit a bill_to as ship_to
	handler := HandleAddressEdit(app, AddressTypeShipTo)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("addressId", addr.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != 403 {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestHandleAddressUpdate_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Update Addr Project")
	addr := testhelpers.CreateTestAddress(t, app, project.Id, "ship_to", "Original Corp")

	handler := HandleAddressUpdate(app, AddressTypeShipTo)

	form := url.Values{}
	form.Set("company_name", "Updated Corp")
	form.Set("contact_person", "Updated Person")
	form.Set("address_line_1", "123 Updated St")
	form.Set("city", "Mumbai")
	form.Set("state", "Maharashtra")
	form.Set("pin_code", "400001")
	form.Set("country", "India")
	form.Set("phone", "9876543210")

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("addressId", addr.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", rec.Code)
	}

	// Verify update
	updated, _ := app.FindRecordById("addresses", addr.Id)
	if updated.GetString("company_name") != "Updated Corp" {
		t.Errorf("company_name not updated, got %q", updated.GetString("company_name"))
	}
}

func TestHandleAddressUpdate_HTMX(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Update HTMX Project")
	addr := testhelpers.CreateTestAddress(t, app, project.Id, "ship_to", "HTMX Update Corp")

	handler := HandleAddressUpdate(app, AddressTypeShipTo)

	form := url.Values{}
	form.Set("company_name", "HTMX Updated")
	form.Set("address_line_1", "456 Updated Ave")
	form.Set("city", "Delhi")
	form.Set("state", "Delhi")
	form.Set("pin_code", "110001")
	form.Set("country", "India")
	form.Set("phone", "9876543211")
	form.Set("contact_person", "HTMX Person")

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("addressId", addr.Id)
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

func TestHandleAddressUpdate_NotFound(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Update NF Project")

	handler := HandleAddressUpdate(app, AddressTypeBillTo)

	form := url.Values{}
	form.Set("company_name", "No Update")
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("addressId", "nonexistent")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != 404 {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}
