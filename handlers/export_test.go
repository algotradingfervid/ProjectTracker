package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"projectcreation/testhelpers"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"spaces to hyphens", "My BOQ File", "My-BOQ-File"},
		{"slashes to hyphens", "path/to/file", "path-to-file"},
		{"backslashes", "path\\to\\file", "path-to-file"},
		{"colons", "file:name", "file-name"},
		{"mixed", "A / B \\ C : D", "A---B---C---D"},
		{"no special chars", "simple", "simple"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildExportData_WithItems(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Export Project")
	boq := testhelpers.CreateTestBOQ(t, app, proj.Id, "Export BOQ")
	mainItem := testhelpers.CreateTestMainBOQItem(t, app, boq.Id, "Export Main Item")
	testhelpers.CreateTestSubItem(t, app, mainItem.Id, "Export Sub Item")

	data, err := buildExportData(app, boq.Id)
	if err != nil {
		t.Fatalf("buildExportData error: %v", err)
	}
	if data.Title != "Export BOQ" {
		t.Errorf("title = %q, want 'Export BOQ'", data.Title)
	}
	if len(data.Rows) < 2 {
		t.Errorf("expected at least 2 rows (main+sub), got %d", len(data.Rows))
	}
	// First row should be level 0 (main item)
	if data.Rows[0].Level != 0 {
		t.Errorf("first row level = %d, want 0", data.Rows[0].Level)
	}
	if data.Rows[0].Description != "Export Main Item" {
		t.Errorf("first row desc = %q", data.Rows[0].Description)
	}
}

func TestBuildExportData_Empty(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Empty Export")
	boq := testhelpers.CreateTestBOQ(t, app, proj.Id, "Empty BOQ")

	data, err := buildExportData(app, boq.Id)
	if err != nil {
		t.Fatalf("buildExportData error: %v", err)
	}
	if len(data.Rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(data.Rows))
	}
}

func TestBuildExportData_NotFound(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	_, err := buildExportData(app, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent BOQ")
	}
}

func TestHandleBOQExportExcel_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Excel Export Project")
	boq := testhelpers.CreateTestBOQ(t, app, proj.Id, "Excel BOQ")
	testhelpers.CreateTestMainBOQItem(t, app, boq.Id, "Excel Item")
	handler := HandleBOQExportExcel(app)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/projects/%s/boq/%s/export/excel", proj.Id, boq.Id), nil)
	req.SetPathValue("projectId", proj.Id)
	req.SetPathValue("id", boq.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "spreadsheetml") {
		t.Errorf("expected Excel content type, got %q", ct)
	}
	cd := rec.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "attachment") {
		t.Errorf("expected attachment disposition, got %q", cd)
	}
	if rec.Body.Len() == 0 {
		t.Error("expected non-empty body")
	}
}

func TestHandleBOQExportPDF_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "PDF Export Project")
	boq := testhelpers.CreateTestBOQ(t, app, proj.Id, "PDF BOQ")
	testhelpers.CreateTestMainBOQItem(t, app, boq.Id, "PDF Item")
	handler := HandleBOQExportPDF(app)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/projects/%s/boq/%s/export/pdf", proj.Id, boq.Id), nil)
	req.SetPathValue("projectId", proj.Id)
	req.SetPathValue("id", boq.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "application/pdf" {
		t.Errorf("expected application/pdf, got %q", ct)
	}
	if rec.Body.Len() == 0 {
		t.Error("expected non-empty PDF body")
	}
}

func TestHandleBOQExportExcel_NotFound(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	handler := HandleBOQExportExcel(app)
	req := httptest.NewRequest(http.MethodGet, "/projects/x/boq/nonexistent/export/excel", nil)
	req.SetPathValue("projectId", "x")
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
