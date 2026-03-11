package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"projectcreation/testhelpers"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func createTestDCTemplate(t *testing.T, app *pocketbase.PocketBase, projectID, name string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("dc_templates")
	if err != nil {
		t.Fatalf("failed to find dc_templates collection: %v", err)
	}
	record := core.NewRecord(col)
	record.Set("project", projectID)
	record.Set("name", name)
	record.Set("purpose", "Test template")
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save test DC template: %v", err)
	}
	return record
}

func createTestTransporter(t *testing.T, app *pocketbase.PocketBase, projectID, companyName string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("transporters")
	if err != nil {
		t.Fatalf("failed to find transporters collection: %v", err)
	}
	record := core.NewRecord(col)
	record.Set("project", projectID)
	record.Set("company_name", companyName)
	record.Set("is_active", true)
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save test transporter: %v", err)
	}
	return record
}

func TestHandleDCWizardStep1_Renders(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Wizard Step1 Project")

	createTestDCTemplate(t, app, project.Id, "Standard Template")
	createTestTransporter(t, app, project.Id, "Fast Transport")

	handler := HandleDCWizardStep1(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/dcs/create", nil)
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	testhelpers.AssertHTMLContains(t, body, "Standard Template", "Fast Transport")
}

func TestHandleDCWizardStep1_HTMXPartial(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Wizard HTMX Project")

	createTestDCTemplate(t, app, project.Id, "HTMX Template")

	handler := HandleDCWizardStep1(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/dcs/create", nil)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	body := rec.Body.String()
	testhelpers.AssertHTMLContains(t, body, "HTMX Template")

	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("expected HTMX partial NOT to contain <!DOCTYPE html>")
	}
}

func TestHandleDCWizardStep2_RequiresTemplate(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Wizard Validation Project")

	createTestDCTemplate(t, app, project.Id, "Validation Template")

	handler := HandleDCWizardStep2(app)

	// POST without template_id should re-render step 1 with error
	req := httptest.NewRequest(http.MethodPost, "/projects/"+project.Id+"/dcs/create/step2", strings.NewReader("dc_type=direct&challan_date=2026-03-10"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	testhelpers.AssertHTMLContains(t, body, "DC Template is required")
}

func TestHandleDCDetail_NotFound(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Detail 404 Project")

	handler := HandleDCDetail(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/dcs/nonexistent", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		// ErrorToast returns 200 with HX-Retarget
		// Check for toast error message
	}
}

func TestHandleDCDetail_Renders(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Detail Render Project")

	dc := createTestDC(t, app, project.Id, "DC-DETAIL-001", "transit", "draft")

	handler := HandleDCDetail(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/dcs/"+dc.Id, nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", dc.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	testhelpers.AssertHTMLContains(t, body, "DC-DETAIL-001", "DELIVERY CHALLAN")
}

func TestHandleDCDelete_OnlyDraft(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Delete Draft Project")

	dc := createTestDC(t, app, project.Id, "DC-ISSUED-DEL", "transit", "issued")

	handler := HandleDCDelete(app)

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+project.Id+"/dcs/"+dc.Id, nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", dc.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Issued DC should not be deleted — should return error toast
	// The DC should still exist
	_, err := app.FindRecordById("delivery_challans", dc.Id)
	if err != nil {
		t.Error("expected issued DC to still exist after delete attempt, but it was not found")
	}
}

func TestHandleDCDelete_DraftSuccess(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Delete Success Project")

	dc := createTestDC(t, app, project.Id, "DC-DRAFT-DEL", "transit", "draft")

	handler := HandleDCDelete(app)

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+project.Id+"/dcs/"+dc.Id, nil)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", dc.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Draft DC should be deleted
	_, err := app.FindRecordById("delivery_challans", dc.Id)
	if err == nil {
		t.Error("expected draft DC to be deleted, but it still exists")
	}
}
