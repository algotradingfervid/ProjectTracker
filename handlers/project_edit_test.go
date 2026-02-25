package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandleProjectEdit_GETForm(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Edit Me")

	handler := HandleProjectEdit(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+proj.Id+"/edit", nil)
	req.SetPathValue("id", proj.Id)
	rec := httptest.NewRecorder()

	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	testhelpers.AssertHTMLContains(t, rec.Body.String(), "Edit Me")
}

func TestHandleProjectEdit_NotFound(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	handler := HandleProjectEdit(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/nonexistent/edit", nil)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()

	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleProjectUpdate_ValidSave(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Original Name")

	handler := HandleProjectUpdate(app)

	form := url.Values{}
	form.Set("name", "Updated Name")
	form.Set("client_name", "New Client")
	form.Set("status", "active")

	req := httptest.NewRequest(http.MethodPost, "/projects/"+proj.Id+"/save",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("id", proj.Id)
	rec := httptest.NewRecorder()

	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	testhelpers.AssertHXRedirect(t, rec.Header().Get("HX-Redirect"), "/projects")

	// Verify update in DB
	updated, err := app.FindRecordById("projects", proj.Id)
	if err != nil {
		t.Fatalf("could not find updated project: %v", err)
	}
	if updated.GetString("name") != "Updated Name" {
		t.Errorf("name not updated, got %q", updated.GetString("name"))
	}
}

func TestHandleProjectUpdate_EmptyName(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Keep This")

	handler := HandleProjectUpdate(app)

	form := url.Values{}
	form.Set("name", "")
	form.Set("status", "active")

	req := httptest.NewRequest(http.MethodPost, "/projects/"+proj.Id+"/save",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("id", proj.Id)
	rec := httptest.NewRecorder()

	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	// Should re-render form, not redirect
	if rec.Header().Get("HX-Redirect") != "" {
		t.Error("expected no HX-Redirect for validation error")
	}
}

func TestHandleProjectUpdate_DuplicateName(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	testhelpers.CreateTestProject(t, app, "Existing Name")
	proj := testhelpers.CreateTestProject(t, app, "My Project")

	handler := HandleProjectUpdate(app)

	form := url.Values{}
	form.Set("name", "Existing Name")
	form.Set("status", "active")

	req := httptest.NewRequest(http.MethodPost, "/projects/"+proj.Id+"/save",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("id", proj.Id)
	rec := httptest.NewRecorder()

	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	// Should re-render form, not redirect
	if rec.Header().Get("HX-Redirect") != "" {
		t.Error("expected no HX-Redirect for duplicate name")
	}
}
