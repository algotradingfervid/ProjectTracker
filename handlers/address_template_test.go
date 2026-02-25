package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandleAddressTemplateDownload_ShipTo(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Template ShipTo Project")

	handler := HandleAddressTemplateDownload(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/addresses/ship_to/template", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("type", "ship_to")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Errorf("unexpected content-type: %s", contentType)
	}
	disp := rec.Header().Get("Content-Disposition")
	if disp == "" {
		t.Error("expected Content-Disposition header")
	}
}

func TestHandleAddressTemplateDownload_InstallAt(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Template InstallAt Project")

	handler := HandleAddressTemplateDownload(app)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("type", "install_at")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Errorf("unexpected content-type: %s", contentType)
	}
}

func TestHandleAddressTemplateDownload_InvalidType(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Template BadType Project")

	handler := HandleAddressTemplateDownload(app)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("type", "bill_to")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleAddressTemplateDownload_InvalidProject(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	handler := HandleAddressTemplateDownload(app)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetPathValue("projectId", "nonexistent")
	req.SetPathValue("type", "ship_to")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleAddressTemplateDownload_MissingProject(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	handler := HandleAddressTemplateDownload(app)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetPathValue("projectId", "")
	req.SetPathValue("type", "ship_to")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
