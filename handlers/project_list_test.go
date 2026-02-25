package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandleProjectList_Empty(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	handler := HandleProjectList(app)

	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	rec := httptest.NewRecorder()

	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandleProjectList_WithProjects(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	testhelpers.CreateTestProject(t, app, "Alpha Project")
	testhelpers.CreateTestProject(t, app, "Beta Project")

	handler := HandleProjectList(app)

	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	rec := httptest.NewRecorder()

	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	testhelpers.AssertHTMLContains(t, rec.Body.String(), "Alpha Project", "Beta Project")
}

func TestHandleProjectList_HTMXPartial(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	testhelpers.CreateTestProject(t, app, "Test Project")

	handler := HandleProjectList(app)

	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	testhelpers.AssertHTMLContains(t, rec.Body.String(), "Test Project")
}

func TestHandleProjectList_WithBOQCount(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Counted Project")
	testhelpers.CreateTestBOQ(t, app, proj.Id, "BOQ One")
	testhelpers.CreateTestBOQ(t, app, proj.Id, "BOQ Two")

	handler := HandleProjectList(app)

	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	rec := httptest.NewRecorder()

	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	testhelpers.AssertHTMLContains(t, rec.Body.String(), "Counted Project")
}
