package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandleBOQDelete_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Delete BOQ Project")
	boq := testhelpers.CreateTestBOQ(t, app, proj.Id, "Delete This BOQ")
	handler := HandleBOQDelete(app)
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/projects/%s/boq/%s", proj.Id, boq.Id), nil)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", proj.Id)
	req.SetPathValue("id", boq.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	testhelpers.AssertHXRedirect(t, rec.Header().Get("HX-Redirect"), fmt.Sprintf("/projects/%s/boq", proj.Id))
	// Verify deleted
	_, err := app.FindRecordById("boqs", boq.Id)
	if err == nil {
		t.Error("expected BOQ to be deleted")
	}
}

func TestHandleBOQDelete_NotFound(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "No BOQ Project")
	handler := HandleBOQDelete(app)
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/projects/%s/boq/nonexistent", proj.Id), nil)
	req.SetPathValue("projectId", proj.Id)
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

func TestHandleBOQDelete_CascadeItems(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Cascade Project")
	boq := testhelpers.CreateTestBOQ(t, app, proj.Id, "Cascade BOQ")
	mainItem := testhelpers.CreateTestMainBOQItem(t, app, boq.Id, "Main Item")
	subItem := testhelpers.CreateTestSubItem(t, app, mainItem.Id, "Sub Item")
	subSubItem := testhelpers.CreateTestSubSubItem(t, app, subItem.Id, "Sub-Sub Item")
	handler := HandleBOQDelete(app)
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/projects/%s/boq/%s", proj.Id, boq.Id), nil)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("projectId", proj.Id)
	req.SetPathValue("id", boq.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	// All items should be cascade deleted
	if _, err := app.FindRecordById("main_boq_items", mainItem.Id); err == nil {
		t.Error("expected main item to be cascade deleted")
	}
	if _, err := app.FindRecordById("sub_items", subItem.Id); err == nil {
		t.Error("expected sub item to be cascade deleted")
	}
	if _, err := app.FindRecordById("sub_sub_items", subSubItem.Id); err == nil {
		t.Error("expected sub-sub item to be cascade deleted")
	}
}
