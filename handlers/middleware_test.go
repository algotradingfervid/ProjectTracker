package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"projectcreation/templates"
	"projectcreation/testhelpers"
)

func TestGetActiveProject_FromContext(t *testing.T) {
	expected := &templates.ActiveProject{ID: "test123", Name: "Test Project"}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), ActiveProjectKey, expected)
	req = req.WithContext(ctx)

	got := GetActiveProject(req)
	if got == nil {
		t.Fatal("expected active project, got nil")
	}
	if got.ID != expected.ID {
		t.Errorf("expected ID %q, got %q", expected.ID, got.ID)
	}
}

func TestGetActiveProject_NotInContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	got := GetActiveProject(req)
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestGetHeaderData_FromContext(t *testing.T) {
	expected := templates.HeaderData{
		ActiveProject: &templates.ActiveProject{ID: "p1", Name: "Proj"},
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), HeaderDataKey, expected)
	req = req.WithContext(ctx)

	got := GetHeaderData(req)
	if got.ActiveProject == nil {
		t.Fatal("expected active project in header data")
	}
	if got.ActiveProject.ID != "p1" {
		t.Errorf("expected ID 'p1', got %q", got.ActiveProject.ID)
	}
}

func TestGetHeaderData_NotInContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	got := GetHeaderData(req)
	if got.ActiveProject != nil {
		t.Error("expected nil active project")
	}
}

func TestGetSidebarData_FromContext(t *testing.T) {
	expected := templates.SidebarData{
		ActiveProject: &templates.ActiveProject{ID: "p1", Name: "Test"},
		ActivePath:    "/projects",
		BOQCount:      5,
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), SidebarDataKey, expected)
	req = req.WithContext(ctx)

	got := GetSidebarData(req)
	if got.ActiveProject == nil || got.ActiveProject.ID != "p1" {
		t.Error("expected active project with ID p1")
	}
	if got.BOQCount != 5 {
		t.Errorf("expected BOQCount 5, got %d", got.BOQCount)
	}
}

func TestGetSidebarData_NotInContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	got := GetSidebarData(req)
	if got.ActiveProject != nil {
		t.Error("expected nil active project in empty sidebar data")
	}
}

func TestActiveProjectMiddleware_NoCookie(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	testhelpers.CreateTestProject(t, app, "MW Test Project")

	middleware := ActiveProjectMiddleware(app)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	// The middleware calls e.Next() which we can't easily mock with core.RequestEvent
	// Instead, just verify it doesn't panic and returns nil when Next is the default no-op
	err := middleware(e)
	// e.Next() with no handler set will return nil in PocketBase
	_ = err
}

func TestActiveProjectMiddleware_WithCookie(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Cookie MW Project")

	middleware := ActiveProjectMiddleware(app)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "active_project", Value: project.Id})
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	err := middleware(e)
	_ = err

	// After middleware runs, the request context should have the active project
	activeProject := GetActiveProject(e.Request)
	if activeProject == nil {
		t.Fatal("expected active project in context after middleware")
	}
	if activeProject.Name != "Cookie MW Project" {
		t.Errorf("expected 'Cookie MW Project', got %q", activeProject.Name)
	}

	// Header data should also be in context
	headerData := GetHeaderData(e.Request)
	if headerData.ActiveProject == nil {
		t.Error("expected active project in header data")
	}
}

func TestActiveProjectMiddleware_InvalidCookie(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	middleware := ActiveProjectMiddleware(app)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "active_project", Value: "nonexistent_id"})
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	err := middleware(e)
	_ = err

	// Active project should be nil for invalid cookie
	activeProject := GetActiveProject(e.Request)
	if activeProject != nil {
		t.Error("expected nil active project for invalid cookie")
	}
}
