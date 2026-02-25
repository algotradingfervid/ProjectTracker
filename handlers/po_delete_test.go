package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandlePODelete_Draft(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Draft PO Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Draft Vendor")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-DRAFT/25-26/001")
	lineItem := testhelpers.CreateTestPOLineItem(t, app, po.Id, 1, "Test Line Item", 2, 1000, 18)

	handler := HandlePODelete(app)

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+project.Id+"/po/"+po.Id, nil)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", po.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	testhelpers.AssertHXRedirect(t, rec.Header().Get("HX-Redirect"), "/projects/"+project.Id+"/po")

	// Verify PO is gone from DB
	_, err := app.FindRecordById("purchase_orders", po.Id)
	if err == nil {
		t.Error("expected purchase order to be deleted, but it still exists")
	}

	// Verify line item is also gone (cascade delete)
	_, err = app.FindRecordById("po_line_items", lineItem.Id)
	if err == nil {
		t.Error("expected line item to be cascade-deleted with the purchase order, but it still exists")
	}
}

func TestHandlePODelete_Sent(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Sent PO Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Sent Vendor")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-SENT/25-26/001")

	po.Set("status", "sent")
	if err := app.Save(po); err != nil {
		t.Fatalf("failed to update PO status to sent: %v", err)
	}

	handler := HandlePODelete(app)

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+project.Id+"/po/"+po.Id, nil)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", po.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 Bad Request, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "Cannot delete a sent purchase order") {
		t.Errorf("expected body to contain %q, got: %s", "Cannot delete a sent purchase order", rec.Body.String())
	}

	// Verify PO still exists
	_, err := app.FindRecordById("purchase_orders", po.Id)
	if err != nil {
		t.Error("expected purchase order to still exist, but it was deleted")
	}
}

func TestHandlePODelete_Acknowledged(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Acknowledged PO Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Acknowledged Vendor")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-ACK/25-26/001")

	po.Set("status", "acknowledged")
	if err := app.Save(po); err != nil {
		t.Fatalf("failed to update PO status to acknowledged: %v", err)
	}

	handler := HandlePODelete(app)

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+project.Id+"/po/"+po.Id, nil)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", po.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 Bad Request, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "Cannot delete a acknowledged purchase order") {
		t.Errorf("expected body to contain %q, got: %s", "Cannot delete a acknowledged purchase order", rec.Body.String())
	}

	// Verify PO still exists
	_, err := app.FindRecordById("purchase_orders", po.Id)
	if err != nil {
		t.Error("expected purchase order to still exist, but it was deleted")
	}
}

func TestHandlePODelete_Completed(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Completed PO Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Completed Vendor")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-COMP/25-26/001")

	po.Set("status", "completed")
	if err := app.Save(po); err != nil {
		t.Fatalf("failed to update PO status to completed: %v", err)
	}

	handler := HandlePODelete(app)

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+project.Id+"/po/"+po.Id, nil)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", po.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 Bad Request, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "Cannot delete a completed purchase order") {
		t.Errorf("expected body to contain %q, got: %s", "Cannot delete a completed purchase order", rec.Body.String())
	}

	// Verify PO still exists
	_, err := app.FindRecordById("purchase_orders", po.Id)
	if err != nil {
		t.Error("expected purchase order to still exist, but it was deleted")
	}
}

func TestHandlePODelete_Cancelled(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Cancelled PO Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Cancelled Vendor")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "FSS-PO-CANCEL/25-26/001")

	po.Set("status", "cancelled")
	if err := app.Save(po); err != nil {
		t.Fatalf("failed to update PO status to cancelled: %v", err)
	}

	handler := HandlePODelete(app)

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+project.Id+"/po/"+po.Id, nil)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", po.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	testhelpers.AssertHXRedirect(t, rec.Header().Get("HX-Redirect"), "/projects/"+project.Id+"/po")

	// Verify PO is gone from DB
	_, err := app.FindRecordById("purchase_orders", po.Id)
	if err == nil {
		t.Error("expected cancelled purchase order to be deleted, but it still exists")
	}
}
