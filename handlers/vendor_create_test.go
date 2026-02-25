package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandleVendorCreate_GET(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	handler := HandleVendorCreate(app)

	req := httptest.NewRequest(http.MethodGet, "/vendors/create", nil)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "VENDOR NAME", "BASIC INFORMATION", "ADDRESS", "BANK DETAILS")
}

func TestHandleVendorSave_Valid(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	handler := HandleVendorSave(app)

	form := url.Values{}
	form.Set("name", "Test Vendor")
	form.Set("city", "Bangalore")
	form.Set("gstin", "29AADCB2230M1ZV")
	form.Set("contact_name", "John Doe")
	form.Set("phone", "9876543210")
	form.Set("bank_name", "HDFC Bank")

	req := httptest.NewRequest(http.MethodPost, "/vendors",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	testhelpers.AssertHXRedirect(t, rec.Header().Get("HX-Redirect"), "/vendors")

	// Verify vendor was saved
	records, err := app.FindRecordsByFilter("vendors", "name = {:name}", "", 1, 0,
		map[string]any{"name": "Test Vendor"})
	if err != nil || len(records) == 0 {
		t.Error("expected vendor to be created in database")
	}
}

func TestHandleVendorSave_MissingName(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	handler := HandleVendorSave(app)

	form := url.Values{}
	form.Set("name", "")
	form.Set("city", "Mumbai")

	req := httptest.NewRequest(http.MethodPost, "/vendors",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Should re-render form, not redirect
	if rec.Header().Get("HX-Redirect") != "" {
		t.Error("expected no HX-Redirect for validation error")
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "Name is required")
}

func TestHandleVendorSave_ProjectContext(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	handler := HandleVendorSave(app)

	form := url.Values{}
	form.Set("name", "Project Vendor")

	req := httptest.NewRequest(http.MethodPost, "/projects/"+project.Id+"/vendors",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	testhelpers.AssertHXRedirect(t, rec.Header().Get("HX-Redirect"),
		"/projects/"+project.Id+"/vendors")

	// Verify vendor was created
	vendors, err := app.FindRecordsByFilter("vendors", "name = {:name}", "", 1, 0,
		map[string]any{"name": "Project Vendor"})
	if err != nil || len(vendors) == 0 {
		t.Fatal("expected vendor to be created")
	}

	// Verify project_vendors link was created
	links, err := app.FindRecordsByFilter("project_vendors",
		"project = {:projectId} && vendor = {:vendorId}", "", 1, 0,
		map[string]any{"projectId": project.Id, "vendorId": vendors[0].Id})
	if err != nil || len(links) == 0 {
		t.Error("expected project_vendors link to be created")
	}
}
