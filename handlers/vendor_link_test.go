package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"projectcreation/testhelpers"
)

func TestHandleVendorLink_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Link Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Link Vendor")
	handler := HandleVendorLink(app)

	req := httptest.NewRequest(http.MethodPost, "/projects/"+project.Id+"/vendors/"+vendor.Id+"/link", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", vendor.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify link was created
	links, err := app.FindRecordsByFilter("project_vendors",
		"project = {:projectId} && vendor = {:vendorId}", "", 1, 0,
		map[string]any{"projectId": project.Id, "vendorId": vendor.Id})
	if err != nil || len(links) == 0 {
		t.Error("expected project_vendors link to be created")
	}

	// Verify response contains LINKED button
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "LINKED")
}

func TestHandleVendorLink_AlreadyLinked(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Already Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Already Vendor")

	// Create initial link
	pvCol, _ := app.FindCollectionByNameOrId("project_vendors")
	link := core.NewRecord(pvCol)
	link.Set("project", project.Id)
	link.Set("vendor", vendor.Id)
	if err := app.Save(link); err != nil {
		t.Fatalf("failed to create initial link: %v", err)
	}

	handler := HandleVendorLink(app)

	req := httptest.NewRequest(http.MethodPost, "/projects/"+project.Id+"/vendors/"+vendor.Id+"/link", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", vendor.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify only one link exists (no duplicate)
	links, _ := app.FindRecordsByFilter("project_vendors",
		"project = {:projectId} && vendor = {:vendorId}", "", 0, 0,
		map[string]any{"projectId": project.Id, "vendorId": vendor.Id})
	if len(links) != 1 {
		t.Errorf("expected exactly 1 link, got %d", len(links))
	}
}

func TestHandleVendorLink_InvalidVendor(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Valid Project")
	handler := HandleVendorLink(app)

	req := httptest.NewRequest(http.MethodPost, "/projects/"+project.Id+"/vendors/nonexistent/link", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestHandleVendorLink_InvalidProject(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	vendor := testhelpers.CreateTestVendor(t, app, "Valid Vendor")
	handler := HandleVendorLink(app)

	req := httptest.NewRequest(http.MethodPost, "/projects/nonexistent/vendors/"+vendor.Id+"/link", nil)
	req.SetPathValue("projectId", "nonexistent")
	req.SetPathValue("id", vendor.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestHandleVendorUnlink_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Unlink Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Unlink Vendor")

	// Create link first
	pvCol, _ := app.FindCollectionByNameOrId("project_vendors")
	link := core.NewRecord(pvCol)
	link.Set("project", project.Id)
	link.Set("vendor", vendor.Id)
	app.Save(link)

	handler := HandleVendorUnlink(app)

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+project.Id+"/vendors/"+vendor.Id+"/link", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", vendor.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify link was deleted
	links, _ := app.FindRecordsByFilter("project_vendors",
		"project = {:projectId} && vendor = {:vendorId}", "", 1, 0,
		map[string]any{"projectId": project.Id, "vendorId": vendor.Id})
	if len(links) != 0 {
		t.Error("expected link to be deleted")
	}

	// Verify response contains LINK button (not LINKED)
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "LINK")
}

func TestHandleVendorUnlink_HasPOs(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "PO Project")
	vendor := testhelpers.CreateTestVendor(t, app, "PO Vendor")
	testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-TEST/25-26/001")

	// Create link
	pvCol, _ := app.FindCollectionByNameOrId("project_vendors")
	link := core.NewRecord(pvCol)
	link.Set("project", project.Id)
	link.Set("vendor", vendor.Id)
	app.Save(link)

	handler := HandleVendorUnlink(app)

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+project.Id+"/vendors/"+vendor.Id+"/link", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", vendor.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", rec.Code)
	}

	// Verify link still exists
	links, _ := app.FindRecordsByFilter("project_vendors",
		"project = {:projectId} && vendor = {:vendorId}", "", 1, 0,
		map[string]any{"projectId": project.Id, "vendorId": vendor.Id})
	if len(links) == 0 {
		t.Error("expected link to still exist")
	}
}

func TestHandleVendorUnlink_NotLinked(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "NoLink Project")
	vendor := testhelpers.CreateTestVendor(t, app, "NoLink Vendor")

	handler := HandleVendorUnlink(app)

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+project.Id+"/vendors/"+vendor.Id+"/link", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", vendor.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Should be 200 (idempotent, no error)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}
