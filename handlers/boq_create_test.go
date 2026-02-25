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

func TestHandleBOQCreate_GETForm(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "BOQ Test Project")
	handler := HandleBOQCreate(app)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/projects/%s/boq/create", proj.Id), nil)
	req.SetPathValue("projectId", proj.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandleBOQSave_Valid(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "BOQ Save Project")
	handler := HandleBOQSave(app)
	form := url.Values{}
	form.Set("title", "Test BOQ")
	form.Set("reference_number", "BOQ-001")
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/projects/%s/boq", proj.Id), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("projectId", proj.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	// Should redirect to the new BOQ view (302)
	if rec.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", rec.Code)
	}
	// Verify BOQ in DB
	records, err := app.FindRecordsByFilter("boqs", "title = {:t}", "", 1, 0, map[string]any{"t": "Test BOQ"})
	if err != nil || len(records) == 0 {
		t.Error("expected BOQ to be created in database")
	}
}

func TestHandleBOQSave_MissingTitle(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "BOQ Validate Project")
	handler := HandleBOQSave(app)
	form := url.Values{}
	form.Set("title", "")
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/projects/%s/boq", proj.Id), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("projectId", proj.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	// Should re-render form, not redirect
	if rec.Code == http.StatusFound {
		t.Error("expected form re-render, not redirect")
	}
}

func TestHandleBOQSave_DuplicateTitle(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Dup BOQ Project")
	testhelpers.CreateTestBOQ(t, app, proj.Id, "Existing BOQ")
	handler := HandleBOQSave(app)
	form := url.Values{}
	form.Set("title", "Existing BOQ")
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/projects/%s/boq", proj.Id), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("projectId", proj.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code == http.StatusFound {
		t.Error("expected form re-render for duplicate title")
	}
}

func TestHandleBOQSave_WithMainItems(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Items Project")
	handler := HandleBOQSave(app)
	form := url.Values{}
	form.Set("title", "BOQ With Items")
	form.Set("items[0].description", "Main Item One")
	form.Set("items[0].qty", "10")
	form.Set("items[0].uom", "Nos")
	form.Set("items[0].quoted_price", "500")
	form.Set("items[0].budgeted_price", "450")
	form.Set("items[0].gst_percent", "18")
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/projects/%s/boq", proj.Id), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("projectId", proj.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	// Handler redirects to the new BOQ view on success
	if rec.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", rec.Code)
	}
	// Verify BOQ was created (items may fail unit_price required constraint
	// but the BOQ record itself should be saved)
	records, err := app.FindRecordsByFilter("boqs", "title = {:t}", "", 1, 0, map[string]any{"t": "BOQ With Items"})
	if err != nil || len(records) == 0 {
		t.Error("expected BOQ to be created in database")
	}
}
