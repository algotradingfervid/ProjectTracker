package handlers

import (
	"context"
	"net/http"
	"testing"

	"projectcreation/templates"
	"projectcreation/testhelpers"
)

// newRequestWithProject creates an HTTP request with an active project in context.
func newRequestWithProject(path string, proj *templates.ActiveProject) *http.Request {
	req, _ := http.NewRequest("GET", path, nil)
	ctx := context.WithValue(req.Context(), ActiveProjectKey, proj)
	return req.WithContext(ctx)
}

// newRequestNoProject creates an HTTP request with no active project in context.
func newRequestNoProject(path string) *http.Request {
	req, _ := http.NewRequest("GET", path, nil)
	return req
}

func TestBuildSidebarData_IncludesVendorCount(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor1 := testhelpers.CreateTestVendor(t, app, "Vendor A")
	vendor2 := testhelpers.CreateTestVendor(t, app, "Vendor B")
	testhelpers.LinkVendorToProject(t, app, project.Id, vendor1.Id)
	testhelpers.LinkVendorToProject(t, app, project.Id, vendor2.Id)

	req := newRequestWithProject("/projects/"+project.Id, &templates.ActiveProject{
		ID:   project.Id,
		Name: "Test Project",
	})

	data := BuildSidebarData(req, app)

	if data.VendorCount != 2 {
		t.Errorf("VendorCount = %d, want 2", data.VendorCount)
	}
}

func TestBuildSidebarData_IncludesPOCount(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Test Vendor")
	testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "PO-001")
	testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "PO-002")
	testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "PO-003")

	req := newRequestWithProject("/projects/"+project.Id, &templates.ActiveProject{
		ID:   project.Id,
		Name: "Test Project",
	})

	data := BuildSidebarData(req, app)

	if data.POCount != 3 {
		t.Errorf("POCount = %d, want 3", data.POCount)
	}
}

func TestBuildSidebarData_VendorCountZero(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Empty Project")

	req := newRequestWithProject("/projects/"+project.Id, &templates.ActiveProject{
		ID:   project.Id,
		Name: "Empty Project",
	})

	data := BuildSidebarData(req, app)

	if data.VendorCount != 0 {
		t.Errorf("VendorCount = %d, want 0", data.VendorCount)
	}
}

func TestBuildSidebarData_POCountZero(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Empty Project")

	req := newRequestWithProject("/projects/"+project.Id, &templates.ActiveProject{
		ID:   project.Id,
		Name: "Empty Project",
	})

	data := BuildSidebarData(req, app)

	if data.POCount != 0 {
		t.Errorf("POCount = %d, want 0", data.POCount)
	}
}

func TestSidebar_NoProject_ReturnsMinimalData(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	req := newRequestNoProject("/vendors")
	data := BuildSidebarData(req, app)

	if data.ActiveProject != nil {
		t.Error("ActiveProject should be nil when no project is active")
	}
	if data.ActivePath != "/vendors" {
		t.Errorf("ActivePath = %q, want %q", data.ActivePath, "/vendors")
	}
}

func TestSidebar_ActiveProject_PopulatesBothCounts(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Full Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Vendor A")
	testhelpers.LinkVendorToProject(t, app, project.Id, vendor.Id)
	testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "PO-001")

	req := newRequestWithProject("/projects/"+project.Id, &templates.ActiveProject{
		ID:   project.Id,
		Name: "Full Project",
	})

	data := BuildSidebarData(req, app)

	if data.ActiveProject == nil {
		t.Fatal("ActiveProject should not be nil")
	}
	if data.VendorCount != 1 {
		t.Errorf("VendorCount = %d, want 1", data.VendorCount)
	}
	if data.POCount != 1 {
		t.Errorf("POCount = %d, want 1", data.POCount)
	}
	if data.BOQCount != 0 {
		t.Errorf("BOQCount = %d, want 0 (no BOQs created)", data.BOQCount)
	}
}

func TestSidebar_ActiveProject_ActivePathSet(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")

	req := newRequestWithProject("/projects/"+project.Id+"/po", &templates.ActiveProject{
		ID:   project.Id,
		Name: "Test Project",
	})

	data := BuildSidebarData(req, app)

	if data.ActivePath != "/projects/"+project.Id+"/po" {
		t.Errorf("ActivePath = %q, want %q", data.ActivePath, "/projects/"+project.Id+"/po")
	}
}
