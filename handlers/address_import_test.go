package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandleAddressImportPage_ShipTo(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Import Page Project")

	handler := HandleAddressImportPage(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/addresses/ship-to/import", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("type", "ship-to")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandleAddressImportPage_HTMX(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Import HTMX Project")

	handler := HandleAddressImportPage(app)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("type", "install-at")
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandleAddressImportPage_InvalidType(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Import BadType Project")

	handler := HandleAddressImportPage(app)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("type", "bill-to")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleAddressImportCommit_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Import Commit Project")

	handler := HandleAddressImportCommit(app)

	parsedJSON := `[{"company_name":"Imported Corp","address_line_1":"1 Import St","city":"Mumbai","state":"Maharashtra","pin_code":"400001","country":"India","phone":"9876543210","contact_person":"J"}]`
	form := url.Values{}
	form.Set("parsed_rows_json", parsedJSON)

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("type", "ship-to")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandleAddressImportCommit_MissingData(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Import NoData Project")

	handler := HandleAddressImportCommit(app)

	form := url.Values{}
	form.Set("parsed_rows_json", "")

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("type", "ship-to")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleAddressImportCommit_InvalidProject(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	handler := HandleAddressImportCommit(app)

	form := url.Values{}
	form.Set("parsed_rows_json", "[]")

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("projectId", "nonexistent")
	req.SetPathValue("type", "ship-to")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleAddressErrorReport(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	handler := HandleAddressErrorReport(app)

	body := `[{"row":1,"field":"company_name","message":"required","value":""}]`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("type", "ship-to")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Errorf("unexpected content-type: %s", contentType)
	}
}

func TestSlugToDBType(t *testing.T) {
	tests := []struct {
		slug string
		want string
	}{
		{"ship-to", "ship_to"},
		{"install-at", "install_at"},
		{"bill-to", "bill-to"},
		{"other", "other"},
	}
	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			if got := slugToDBType(tt.slug); got != tt.want {
				t.Errorf("slugToDBType(%q) = %q, want %q", tt.slug, got, tt.want)
			}
		})
	}
}
