package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandleProjectSettings_GETForm(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Settings Project")
	handler := HandleProjectSettings(app)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/projects/%s/settings", proj.Id), nil)
	req.SetPathValue("id", proj.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "Settings Project")
}

func TestHandleProjectSettings_NotFound(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	handler := HandleProjectSettings(app)
	req := httptest.NewRequest(http.MethodGet, "/projects/nonexistent/settings", nil)
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

func TestHandleProjectSettingsSave_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Save Settings")
	handler := HandleProjectSettingsSave(app)
	form := url.Values{}
	form.Set("ship_to_equals_install_at", "on")
	form.Set("bill_from.req_company_name", "true")
	form.Set("bill_from.req_gstin", "true")
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/projects/%s/settings", proj.Id), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("id", proj.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	expectedRedirect := fmt.Sprintf("/projects/%s/settings", proj.Id)
	testhelpers.AssertHXRedirect(t, rec.Header().Get("HX-Redirect"), expectedRedirect)
	// Verify ship_to_equals_install_at was saved
	updated, _ := app.FindRecordById("projects", proj.Id)
	if !updated.GetBool("ship_to_equals_install_at") {
		t.Error("expected ship_to_equals_install_at to be true")
	}
}

func TestHandleProjectSettingsSave_NotFound(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	handler := HandleProjectSettingsSave(app)
	form := url.Values{}
	form.Set("ship_to_equals_install_at", "on")
	req := httptest.NewRequest(http.MethodPost, "/projects/nonexistent/settings", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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
