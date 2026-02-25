package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandleAddressCreate_GET(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Addr Create Project")

	handler := HandleAddressCreate(app, AddressTypeBillTo)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/addresses/bill-to/new", nil)
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "company_name", "Bill To")
}

func TestHandleAddressCreate_GET_HTMX(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Addr Create HTMX")

	handler := HandleAddressCreate(app, AddressTypeShipTo)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/addresses/ship-to/new", nil)
	req.SetPathValue("projectId", project.Id)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestHandleAddressCreate_GET_InvalidProject(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	handler := HandleAddressCreate(app, AddressTypeBillTo)

	req := httptest.NewRequest(http.MethodGet, "/projects/nonexistent/addresses/bill-to/new", nil)
	req.SetPathValue("projectId", "nonexistent")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != 404 {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleAddressSave_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Addr Save Project")

	handler := HandleAddressSave(app, AddressTypeShipTo)

	form := url.Values{}
	form.Set("company_name", "New Ship Corp")
	form.Set("contact_person", "Jane Doe")
	form.Set("address_line_1", "123 Ship Lane")
	form.Set("city", "Mumbai")
	form.Set("state", "Maharashtra")
	form.Set("pin_code", "400001")
	form.Set("country", "India")
	form.Set("phone", "9876543210")

	req := httptest.NewRequest(http.MethodPost, "/projects/"+project.Id+"/addresses/ship-to/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d", rec.Code)
	}
}

func TestHandleAddressSave_HTMX(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Addr Save HTMX")

	handler := HandleAddressSave(app, AddressTypeShipTo)

	form := url.Values{}
	form.Set("company_name", "HTMX Ship Corp")
	form.Set("address_line_1", "456 Ship Ave")
	form.Set("city", "Delhi")
	form.Set("state", "Delhi")
	form.Set("pin_code", "110001")
	form.Set("country", "India")
	form.Set("phone", "9876543211")
	form.Set("contact_person", "John")

	req := httptest.NewRequest(http.MethodPost, "/projects/"+project.Id+"/addresses/ship-to/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
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

func TestHandleAddressSave_InvalidProject(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	handler := HandleAddressSave(app, AddressTypeBillTo)

	form := url.Values{}
	form.Set("company_name", "Should Fail")
	req := httptest.NewRequest(http.MethodPost, "/projects/nonexistent/addresses/bill-to/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("projectId", "nonexistent")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != 404 {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleAddressCreate_InstallAt(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "InstallAt Project")
	// Create a ship_to address for the dropdown
	testhelpers.CreateTestAddress(t, app, project.Id, "ship_to", "Parent Ship Co")

	handler := HandleAddressCreate(app, AddressTypeInstallAt)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/addresses/install-at/new", nil)
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
