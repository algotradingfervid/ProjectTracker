package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"projectcreation/testhelpers"

	"github.com/pocketbase/pocketbase/core"
)

func createTestDC(t *testing.T, app interface{ FindCollectionByNameOrId(string) (*core.Collection, error); Save(core.Model) error }, projectID, dcNumber, dcType, status string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("delivery_challans")
	if err != nil {
		t.Fatalf("failed to find delivery_challans collection: %v", err)
	}
	record := core.NewRecord(col)
	record.Set("project", projectID)
	record.Set("dc_number", dcNumber)
	record.Set("dc_type", dcType)
	record.Set("status", status)
	record.Set("challan_date", "2026-03-10")
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save test DC: %v", err)
	}
	return record
}

func TestHandleDCList_Empty(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Empty DC Project")

	handler := HandleDCList(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/dcs/", nil)
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "No delivery challans")
}

func TestHandleDCList_WithData(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "DC List Project")

	createTestDC(t, app, project.Id, "DC-2026-001", "transit", "draft")
	createTestDC(t, app, project.Id, "DC-2026-002", "official", "issued")

	handler := HandleDCList(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/dcs/", nil)
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
	testhelpers.AssertHTMLContains(t, body, "DC-2026-001", "DC-2026-002")
}

func TestHandleDCList_FilterByType(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "DC Type Filter Project")

	createTestDC(t, app, project.Id, "DC-TRANSIT-001", "transit", "draft")
	createTestDC(t, app, project.Id, "DC-OFFICIAL-001", "official", "draft")

	handler := HandleDCList(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/dcs/?type=transit", nil)
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	body := rec.Body.String()
	testhelpers.AssertHTMLContains(t, body, "DC-TRANSIT-001")

	if strings.Contains(body, "DC-OFFICIAL-001") {
		t.Error("expected DC-OFFICIAL-001 NOT to appear when filtering by type=transit, but it was found")
	}
}

func TestHandleDCList_FilterByStatus(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "DC Status Filter Project")

	createTestDC(t, app, project.Id, "DC-DRAFT-001", "transit", "draft")
	createTestDC(t, app, project.Id, "DC-ISSUED-001", "transit", "issued")

	handler := HandleDCList(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/dcs/?status=draft", nil)
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	body := rec.Body.String()
	testhelpers.AssertHTMLContains(t, body, "DC-DRAFT-001")

	if strings.Contains(body, "DC-ISSUED-001") {
		t.Error("expected DC-ISSUED-001 NOT to appear when filtering by status=draft, but it was found")
	}
}

func TestHandleDCList_HTMXPartial(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "HTMX DC Project")

	createTestDC(t, app, project.Id, "DC-HTMX-001", "transit", "draft")

	handler := HandleDCList(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/dcs/", nil)
	req.Header.Set("HX-Request", "true")
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
	testhelpers.AssertHTMLContains(t, body, "DC-HTMX-001")

	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("expected HTMX partial NOT to contain <!DOCTYPE html>, but it was found")
	}
}

func TestHandleDCList_Search(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "DC Search Project")

	createTestDC(t, app, project.Id, "DC-ALPHA-001", "transit", "draft")
	createTestDC(t, app, project.Id, "DC-BETA-002", "transit", "draft")

	handler := HandleDCList(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/dcs/?search=ALPHA", nil)
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	body := rec.Body.String()
	testhelpers.AssertHTMLContains(t, body, "DC-ALPHA-001")

	if strings.Contains(body, "DC-BETA-002") {
		t.Error("expected DC-BETA-002 NOT to appear when searching for ALPHA, but it was found")
	}
}
