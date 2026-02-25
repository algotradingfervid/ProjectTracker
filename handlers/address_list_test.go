package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandleAddressList_GET(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Addr List Project")
	testhelpers.CreateTestAddress(t, app, project.Id, "bill_to", "Alpha Corp")
	testhelpers.CreateTestAddress(t, app, project.Id, "bill_to", "Beta Corp")

	handler := HandleAddressList(app, AddressTypeBillTo)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/addresses/bill-to", nil)
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "Alpha Corp", "Beta Corp", "Bill To")
}

func TestHandleAddressList_HTMX(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Addr HTMX Project")
	testhelpers.CreateTestAddress(t, app, project.Id, "ship_to", "Ship Co")

	handler := HandleAddressList(app, AddressTypeShipTo)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/addresses/ship-to", nil)
	req.SetPathValue("projectId", project.Id)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "Ship Co")
}

func TestHandleAddressList_EmptyList(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Empty Addr Project")

	handler := HandleAddressList(app, AddressTypeBillFrom)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/addresses/bill-from", nil)
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestHandleAddressList_Search(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Search Addr Project")
	testhelpers.CreateTestAddress(t, app, project.Id, "bill_to", "Findable Corp")
	testhelpers.CreateTestAddress(t, app, project.Id, "bill_to", "Other Corp")

	handler := HandleAddressList(app, AddressTypeBillTo)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/addresses/bill-to?search=Findable", nil)
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "Findable Corp")
}

func TestHandleAddressList_Pagination(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Paginated Project")
	for i := 0; i < 5; i++ {
		testhelpers.CreateTestAddress(t, app, project.Id, "ship_to", "Company "+string(rune('A'+i)))
	}

	handler := HandleAddressList(app, AddressTypeShipTo)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/addresses/ship-to?page=1&page_size=2", nil)
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestHandleAddressList_InvalidProject(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	handler := HandleAddressList(app, AddressTypeBillTo)

	req := httptest.NewRequest(http.MethodGet, "/projects/nonexistent/addresses/bill-to", nil)
	req.SetPathValue("projectId", "nonexistent")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != 404 {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestHandleAddressCount(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Count Project")
	testhelpers.CreateTestAddress(t, app, project.Id, "bill_to", "Count Corp A")
	testhelpers.CreateTestAddress(t, app, project.Id, "bill_to", "Count Corp B")

	handler := HandleAddressCount(app, AddressTypeBillTo)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/addresses/bill-to/count", nil)
	req.SetPathValue("projectId", project.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Body.String() != "2" {
		t.Errorf("expected count 2, got %q", rec.Body.String())
	}
}

func TestParseAddressListParams(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantPage int
		wantSize int
		wantSort string
	}{
		{"defaults", "", 1, 20, "company_name"},
		{"custom page", "?page=3&page_size=10", 3, 10, "company_name"},
		{"sort desc", "?sort_by=city&sort_order=desc", 1, 20, "city"},
		{"invalid sort field", "?sort_by=invalid_field", 1, 20, "company_name"},
		{"page_size too large", "?page_size=200", 1, 20, "company_name"},
	}

	app := testhelpers.NewTestApp(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test"+tt.query, nil)
			rec := httptest.NewRecorder()
			e := newTestRequestEvent(app, req, rec)

			params := parseAddressListParams(e)
			if params.Page != tt.wantPage {
				t.Errorf("Page = %d, want %d", params.Page, tt.wantPage)
			}
			if params.PageSize != tt.wantSize {
				t.Errorf("PageSize = %d, want %d", params.PageSize, tt.wantSize)
			}
			if params.SortBy != tt.wantSort {
				t.Errorf("SortBy = %q, want %q", params.SortBy, tt.wantSort)
			}
		})
	}
}

func TestBuildAddressFilter(t *testing.T) {
	filter, params := buildAddressFilter("proj1", AddressTypeBillTo, "")
	if filter != "project = {:projectId} && address_type = {:addressType}" {
		t.Errorf("unexpected filter: %s", filter)
	}
	if params["projectId"] != "proj1" {
		t.Errorf("expected projectId param")
	}

	filter, params = buildAddressFilter("proj1", AddressTypeBillTo, "test")
	if params["search"] != "test" {
		t.Errorf("expected search param")
	}
}

func TestBuildSortString(t *testing.T) {
	if got := buildSortString("city", "asc"); got != "city" {
		t.Errorf("expected 'city', got %q", got)
	}
	if got := buildSortString("city", "desc"); got != "-city" {
		t.Errorf("expected '-city', got %q", got)
	}
}

func TestAddressTypeLabel(t *testing.T) {
	if got := addressTypeLabel(AddressTypeBillTo); got != "Bill To" {
		t.Errorf("expected 'Bill To', got %q", got)
	}
	if got := addressTypeLabel(AddressType("unknown")); got != "unknown" {
		t.Errorf("expected 'unknown', got %q", got)
	}
}
