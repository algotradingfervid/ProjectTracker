package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandleAddressExportExcel_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Export Addr Project")
	testhelpers.CreateTestAddress(t, app, project.Id, "bill_to", "Export Corp A")
	testhelpers.CreateTestAddress(t, app, project.Id, "bill_to", "Export Corp B")

	handler := HandleAddressExportExcel(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/addresses/bill-to/export", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("type", "bill-to")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Should return Excel file
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Errorf("unexpected content-type: %s", contentType)
	}
	disp := rec.Header().Get("Content-Disposition")
	if disp == "" {
		t.Error("expected Content-Disposition header")
	}
	if rec.Body.Len() == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestHandleAddressExportExcel_EmptyList(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Export Empty Project")

	handler := HandleAddressExportExcel(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/addresses/ship-to/export", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("type", "ship-to")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	// Should still return a valid Excel file (empty)
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Errorf("unexpected content-type: %s", contentType)
	}
}

func TestHandleAddressExportExcel_InvalidType(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Export BadType Project")

	handler := HandleAddressExportExcel(app)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("type", "invalid-type")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleAddressExportExcel_InvalidProject(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	handler := HandleAddressExportExcel(app)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetPathValue("projectId", "nonexistent")
	req.SetPathValue("type", "bill-to")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}
