package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandleProjectCreate_RendersForm(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	handler := HandleProjectCreate(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/create", nil)
	rec := httptest.NewRecorder()

	// Create a minimal RequestEvent — PocketBase handlers need this
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestHandleProjectSave_ValidData(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	handler := HandleProjectSave(app)

	form := url.Values{}
	form.Set("name", "Test Project")
	form.Set("client_name", "Test Client")
	form.Set("reference_number", "REF-001")
	form.Set("status", "active")
	form.Set("ship_to_equals_install_at", "true")

	req := httptest.NewRequest(http.MethodPost, "/projects",
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

	testhelpers.AssertHXRedirect(t, rec.Header().Get("HX-Redirect"), "/projects")

	// Verify project was created in the database
	records, err := app.FindRecordsByFilter("projects", "name = {:name}", "", 1, 0,
		map[string]any{"name": "Test Project"})
	if err != nil || len(records) == 0 {
		t.Error("expected project to be created in database")
	}
}

func TestHandleProjectSave_MissingName(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	handler := HandleProjectSave(app)

	form := url.Values{}
	form.Set("name", "")
	form.Set("status", "active")

	req := httptest.NewRequest(http.MethodPost, "/projects",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Should re-render form (200) with errors, not redirect
	if rec.Header().Get("HX-Redirect") != "" {
		t.Error("expected no HX-Redirect for validation error")
	}
}

func TestHandleProjectSave_DuplicateName(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	// Create an existing project
	testhelpers.CreateTestProject(t, app, "Existing Project")

	handler := HandleProjectSave(app)

	form := url.Values{}
	form.Set("name", "Existing Project")
	form.Set("status", "active")

	req := httptest.NewRequest(http.MethodPost, "/projects",
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
		t.Error("expected no HX-Redirect for duplicate name error")
	}
}
