package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandlePOEdit_GET(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Acme Supplies")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "PO-2026-001")
	testhelpers.CreateTestPOLineItem(t, app, po.Id, 1, "Steel Frame Assembly", 10, 5000, 18)
	testhelpers.CreateTestPOLineItem(t, app, po.Id, 2, "Copper Wiring Bundle", 25, 800, 12)

	handler := HandlePOEdit(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/po/"+po.Id+"/edit", nil)
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", po.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(),
		"PO-2026-001",
		"Acme Supplies",
		"Steel Frame Assembly",
		"Copper Wiring Bundle",
	)
}

func TestHandlePOEdit_GET_NotFound(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")

	handler := HandlePOEdit(app)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Id+"/po/nonexistent/edit", nil)
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

func TestHandlePOUpdate_Valid(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Acme Supplies")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "PO-2026-002")

	handler := HandlePOUpdate(app)

	form := url.Values{}
	form.Set("order_date", "2026-03-15")
	form.Set("payment_terms", "Net 30")
	form.Set("comments", "Updated")

	req := httptest.NewRequest(http.MethodPost,
		"/projects/"+project.Id+"/po/"+po.Id+"/save",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", po.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	testhelpers.AssertHXRedirect(t, rec.Header().Get("HX-Redirect"),
		"/projects/"+project.Id+"/po/"+po.Id+"/edit")

	updated, err := app.FindRecordById("purchase_orders", po.Id)
	if err != nil {
		t.Fatalf("could not find updated PO: %v", err)
	}
	if updated.GetString("order_date") != "2026-03-15" {
		t.Errorf("expected order_date '2026-03-15', got %q", updated.GetString("order_date"))
	}
	if updated.GetString("payment_terms") != "Net 30" {
		t.Errorf("expected payment_terms 'Net 30', got %q", updated.GetString("payment_terms"))
	}
	if updated.GetString("comments") != "Updated" {
		t.Errorf("expected comments 'Updated', got %q", updated.GetString("comments"))
	}
}

func TestHandlePOUpdate_StatusTransition_DraftToSent(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Acme Supplies")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "PO-2026-003")

	handler := HandlePOUpdate(app)

	form := url.Values{}
	form.Set("new_status", "sent")

	req := httptest.NewRequest(http.MethodPost,
		"/projects/"+project.Id+"/po/"+po.Id+"/save",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", po.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	testhelpers.AssertHXRedirect(t, rec.Header().Get("HX-Redirect"),
		"/projects/"+project.Id+"/po/"+po.Id+"/edit")

	updated, err := app.FindRecordById("purchase_orders", po.Id)
	if err != nil {
		t.Fatalf("could not find updated PO: %v", err)
	}
	if updated.GetString("status") != "sent" {
		t.Errorf("expected status 'sent', got %q", updated.GetString("status"))
	}
}

func TestHandlePOUpdate_StatusTransition_SentToDraft(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor := testhelpers.CreateTestVendor(t, app, "Acme Supplies")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor.Id, "PO-2026-004")

	// Manually advance status to "sent"
	po.Set("status", "sent")
	if err := app.Save(po); err != nil {
		t.Fatalf("failed to set PO status to sent: %v", err)
	}

	handler := HandlePOUpdate(app)

	form := url.Values{}
	form.Set("new_status", "draft")

	req := httptest.NewRequest(http.MethodPost,
		"/projects/"+project.Id+"/po/"+po.Id+"/save",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", po.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Expect form re-render (no HX-Redirect) with error message
	if hxRedirect := rec.Header().Get("HX-Redirect"); hxRedirect != "" {
		t.Errorf("expected no HX-Redirect on invalid transition, but got %q", hxRedirect)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "Cannot change status back to draft")

	// Status must remain "sent"
	unchanged, err := app.FindRecordById("purchase_orders", po.Id)
	if err != nil {
		t.Fatalf("could not find PO after failed transition: %v", err)
	}
	if unchanged.GetString("status") != "sent" {
		t.Errorf("expected status to remain 'sent', got %q", unchanged.GetString("status"))
	}
}

func TestHandlePOUpdate_CannotChangeVendor(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	project := testhelpers.CreateTestProject(t, app, "Test Project")
	vendor1 := testhelpers.CreateTestVendor(t, app, "Original Vendor")
	vendor2 := testhelpers.CreateTestVendor(t, app, "Replacement Vendor")
	po := testhelpers.CreateTestPurchaseOrder(t, app, project.Id, vendor1.Id, "PO-2026-005")

	handler := HandlePOUpdate(app)

	// Submit vendor_id for vendor2 — the handler must ignore it
	form := url.Values{}
	form.Set("vendor_id", vendor2.Id)
	form.Set("comments", "trying to change vendor")

	req := httptest.NewRequest(http.MethodPost,
		"/projects/"+project.Id+"/po/"+po.Id+"/save",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", project.Id)
	req.SetPathValue("id", po.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Verify the vendor field was not changed
	unchanged, err := app.FindRecordById("purchase_orders", po.Id)
	if err != nil {
		t.Fatalf("could not find PO after update: %v", err)
	}
	if unchanged.GetString("vendor") != vendor1.Id {
		t.Errorf("expected vendor to remain %q (vendor1), got %q", vendor1.Id, unchanged.GetString("vendor"))
	}
}
