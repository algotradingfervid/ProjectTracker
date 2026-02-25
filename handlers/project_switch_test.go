package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandleProjectActivate_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Activate Me")

	handler := HandleProjectActivate(app)

	req := httptest.NewRequest(http.MethodPost, "/projects/"+proj.Id+"/activate", nil)
	req.SetPathValue("id", proj.Id)
	rec := httptest.NewRecorder()

	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	testhelpers.AssertHXRedirect(t, rec.Header().Get("HX-Redirect"), "/projects/"+proj.Id)

	// Check cookie was set
	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "active_project" && c.Value == proj.Id {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected active_project cookie to be set")
	}
}

func TestHandleProjectActivate_NotFound(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	handler := HandleProjectActivate(app)

	req := httptest.NewRequest(http.MethodPost, "/projects/nonexistent/activate", nil)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()

	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if rec.Code != 404 {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleProjectDeactivate_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	handler := HandleProjectDeactivate(app)

	req := httptest.NewRequest(http.MethodPost, "/projects/deactivate", nil)
	rec := httptest.NewRecorder()

	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	testhelpers.AssertHXRedirect(t, rec.Header().Get("HX-Redirect"), "/projects")

	// Check cookie was cleared
	cookies := rec.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "active_project" && c.MaxAge == -1 {
			return // pass
		}
	}
	t.Error("expected active_project cookie to be cleared")
}
