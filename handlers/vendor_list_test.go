package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"projectcreation/testhelpers"
)

func TestHandleVendorList_Empty(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	handler := HandleVendorList(app)

	req := httptest.NewRequest(http.MethodGet, "/vendors", nil)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "No vendors yet", "Add your first vendor")
}

func TestHandleVendorList_WithData(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	testhelpers.CreateTestVendor(t, app, "Acme Corp")
	testhelpers.CreateTestVendor(t, app, "Widget Inc")

	handler := HandleVendorList(app)

	req := httptest.NewRequest(http.MethodGet, "/vendors", nil)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "Acme Corp", "Widget Inc")
}

func TestHandleVendorList_HTMXPartial(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	handler := HandleVendorList(app)

	// Non-HTMX request should include full page shell
	req := httptest.NewRequest(http.MethodGet, "/vendors", nil)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	fullBody := rec.Body.String()

	// HTMX request should return partial only
	req2 := httptest.NewRequest(http.MethodGet, "/vendors", nil)
	req2.Header.Set("HX-Request", "true")
	rec2 := httptest.NewRecorder()
	e2 := newTestRequestEvent(app, req2, rec2)

	if err := handler(e2); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	partialBody := rec2.Body.String()

	// Full page body should be longer (includes layout shell)
	if len(partialBody) >= len(fullBody) {
		t.Error("expected HTMX partial to be shorter than full page")
	}
}

func TestHandleVendorList_ShowsLinkedStatus(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Status Project")
	vendor1 := testhelpers.CreateTestVendor(t, app, "Linked Vendor")
	testhelpers.CreateTestVendor(t, app, "Unlinked Vendor")

	// Link vendor1 to project
	pvCol, _ := app.FindCollectionByNameOrId("project_vendors")
	link := core.NewRecord(pvCol)
	link.Set("project", project.Id)
	link.Set("vendor", vendor1.Id)
	app.Save(link)

	handler := HandleVendorList(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/vendors", nil)
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	body := rec.Body.String()
	// Should show STATUS column header in project context
	testhelpers.AssertHTMLContains(t, body, "STATUS")
	// Should contain both vendor names
	testhelpers.AssertHTMLContains(t, body, "Linked Vendor", "Unlinked Vendor")
	// Should contain the LINKED button for vendor1
	testhelpers.AssertHTMLContains(t, body, "LINKED")
}
