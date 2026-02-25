package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandleBOQView_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "View BOQ Project")
	boq := testhelpers.CreateTestBOQ(t, app, project.Id, "View Test BOQ")
	mainItem := testhelpers.CreateTestMainBOQItem(t, app, boq.Id, "Main Item 1")
	subItem := testhelpers.CreateTestSubItem(t, app, mainItem.Id, "Sub Item 1")
	testhelpers.CreateTestSubSubItem(t, app, subItem.Id, "Sub-Sub Item 1")

	handler := HandleBOQView(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/boq/"+boq.Id, nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", boq.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(),
		"View Test BOQ", "Main Item 1", "Sub Item 1", "Sub-Sub Item 1")
}

func TestHandleBOQView_HTMX(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "View HTMX BOQ Project")
	boq := testhelpers.CreateTestBOQ(t, app, project.Id, "HTMX View BOQ")

	handler := HandleBOQView(app)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", boq.Id)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "HTMX View BOQ")
}

func TestHandleBOQView_NotFound(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	handler := HandleBOQView(app)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetPathValue("projectId", "proj1")
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != 404 {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleBOQView_EmptyBOQ(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Empty View BOQ")
	boq := testhelpers.CreateTestBOQ(t, app, project.Id, "Empty BOQ")

	handler := HandleBOQView(app)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", boq.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "Empty BOQ")
}

func TestFormatQty(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{10, "10"},
		{10.5, "10.50"},
		{0, "0"},
		{100.99, "100.99"},
	}
	for _, tt := range tests {
		got := formatQty(tt.input)
		if got != tt.want {
			t.Errorf("formatQty(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
