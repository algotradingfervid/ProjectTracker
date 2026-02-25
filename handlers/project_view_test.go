package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandleProjectView_Exists(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "View Me")

	handler := HandleProjectView(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+proj.Id, nil)
	req.SetPathValue("id", proj.Id)
	rec := httptest.NewRecorder()

	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	testhelpers.AssertHTMLContains(t, rec.Body.String(), "View Me")
}

func TestHandleProjectView_NotFound(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	handler := HandleProjectView(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/nonexistent", nil)
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

func TestHandleProjectView_MissingID(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	handler := HandleProjectView(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/", nil)
	rec := httptest.NewRecorder()

	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleProjectView_HTMXPartial(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "HTMX View")

	handler := HandleProjectView(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+proj.Id, nil)
	req.SetPathValue("id", proj.Id)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	testhelpers.AssertHTMLContains(t, rec.Body.String(), "HTMX View")
}
