package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandleBOQList_WithBOQs(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "List Project")
	testhelpers.CreateTestBOQ(t, app, proj.Id, "BOQ Alpha")
	testhelpers.CreateTestBOQ(t, app, proj.Id, "BOQ Beta")
	handler := HandleBOQList(app)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/projects/%s/boq", proj.Id), nil)
	req.SetPathValue("projectId", proj.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "BOQ Alpha", "BOQ Beta")
}

func TestHandleBOQList_Empty(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Empty List")
	handler := HandleBOQList(app)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/projects/%s/boq", proj.Id), nil)
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

func TestHandleBOQList_OtherProjectBOQsNotShown(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj1 := testhelpers.CreateTestProject(t, app, "Project One")
	proj2 := testhelpers.CreateTestProject(t, app, "Project Two")
	testhelpers.CreateTestBOQ(t, app, proj1.Id, "Proj1 BOQ")
	testhelpers.CreateTestBOQ(t, app, proj2.Id, "Proj2 BOQ")
	handler := HandleBOQList(app)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/projects/%s/boq", proj1.Id), nil)
	req.SetPathValue("projectId", proj1.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	body := rec.Body.String()
	testhelpers.AssertHTMLContains(t, body, "Proj1 BOQ")
	if strings.Contains(body, "Proj2 BOQ") {
		t.Error("should not show BOQs from other projects")
	}
}

func TestHandleBOQList_HTMXPartial(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "HTMX List")
	testhelpers.CreateTestBOQ(t, app, proj.Id, "HTMX BOQ")
	handler := HandleBOQList(app)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/projects/%s/boq", proj.Id), nil)
	req.SetPathValue("projectId", proj.Id)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "HTMX BOQ")
}
