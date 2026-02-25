package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandleVendorDelete_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	vendor := testhelpers.CreateTestVendor(t, app, "Delete Me")
	handler := HandleVendorDelete(app)

	req := httptest.NewRequest(http.MethodDelete, "/vendors/"+vendor.Id, nil)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("id", vendor.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	testhelpers.AssertHXRedirect(t, rec.Header().Get("HX-Redirect"), "/vendors")

	// Verify vendor was deleted
	_, err := app.FindRecordById("vendors", vendor.Id)
	if err == nil {
		t.Error("expected vendor to be deleted")
	}
}

func TestHandleVendorDelete_LinkedToPO(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "PO Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Linked Vendor")
	testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-TEST/25-26/001")

	handler := HandleVendorDelete(app)

	req := httptest.NewRequest(http.MethodDelete, "/vendors/"+vendor.Id, nil)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("id", vendor.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusConflict {
		t.Errorf("expected status 409 Conflict, got %d", rec.Code)
	}

	// Verify vendor was NOT deleted
	_, err := app.FindRecordById("vendors", vendor.Id)
	if err != nil {
		t.Error("vendor should not have been deleted")
	}
}
