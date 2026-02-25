package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandlePOExportPDF_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "PO PDF Project")
	vendor := testhelpers.CreateTestVendor(t, app, "PDF Vendor")
	testhelpers.LinkVendorToProject(t, app, project.Id, vendor.Id)
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "PO-PDF-001")
	testhelpers.CreateTestPOLineItem(t, app, po.Id, 1, "PDF Item", 10, 500, 18)

	handler := HandlePOExportPDF(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/po/"+po.Id+"/export/pdf", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", po.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/pdf" {
		t.Errorf("expected content-type application/pdf, got %s", contentType)
	}
	disp := rec.Header().Get("Content-Disposition")
	if disp == "" {
		t.Error("expected Content-Disposition header")
	}
	if rec.Body.Len() == 0 {
		t.Error("expected non-empty PDF body")
	}
}

func TestHandlePOExportPDF_NotFound(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "PO PDF NF Project")

	handler := HandlePOExportPDF(app)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandlePOExportPDF_WrongProject(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project1 := testhelpers.CreateTestProject(t, app, "PO PDF Proj1")
	project2 := testhelpers.CreateTestProject(t, app, "PO PDF Proj2")
	vendor := testhelpers.CreateTestVendor(t, app, "Wrong Proj Vendor")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project1.Id, vendor.Id, "PO-WRONG-001")

	handler := HandlePOExportPDF(app)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetPathValue("projectId", project2.Id)
	req.SetPathValue("id", po.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandlePOExportPDF_MissingID(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	handler := HandlePOExportPDF(app)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetPathValue("projectId", "proj1")
	req.SetPathValue("id", "")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
