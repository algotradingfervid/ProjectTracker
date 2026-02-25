package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandleVendorEdit_GET(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	vendor := testhelpers.CreateTestVendor(t, app, "Edit Me Vendor")
	handler := HandleVendorEdit(app)

	req := httptest.NewRequest(http.MethodGet, "/vendors/"+vendor.Id+"/edit", nil)
	req.SetPathValue("id", vendor.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "Edit Me Vendor", "SAVE CHANGES")
}

func TestHandleVendorEdit_NotFound(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	handler := HandleVendorEdit(app)

	req := httptest.NewRequest(http.MethodGet, "/vendors/nonexistent/edit", nil)
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

func TestHandleVendorUpdate_Valid(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	vendor := testhelpers.CreateTestVendor(t, app, "Old Name")
	handler := HandleVendorUpdate(app)

	form := url.Values{}
	form.Set("name", "New Name")
	form.Set("city", "Delhi")
	form.Set("bank_name", "SBI")

	req := httptest.NewRequest(http.MethodPost, "/vendors/"+vendor.Id+"/save",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("id", vendor.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	testhelpers.AssertHXRedirect(t, rec.Header().Get("HX-Redirect"), "/vendors")

	// Verify update
	updated, err := app.FindRecordById("vendors", vendor.Id)
	if err != nil {
		t.Fatal("could not find updated vendor")
	}
	if updated.GetString("name") != "New Name" {
		t.Errorf("expected name 'New Name', got %q", updated.GetString("name"))
	}
	if updated.GetString("city") != "Delhi" {
		t.Errorf("expected city 'Delhi', got %q", updated.GetString("city"))
	}
}
