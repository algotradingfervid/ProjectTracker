package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandleProjectDelete_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Delete Me")

	handler := HandleProjectDelete(app)

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+proj.Id+"?delete_boqs=true", nil)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("id", proj.Id)
	rec := httptest.NewRecorder()

	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	testhelpers.AssertHXRedirect(t, rec.Header().Get("HX-Redirect"), "/projects")

	// Verify deleted
	_, err := app.FindRecordById("projects", proj.Id)
	if err == nil {
		t.Error("expected project to be deleted")
	}
}

func TestHandleProjectDelete_NotFound(t *testing.T) {
	app := testhelpers.NewTestApp(t)

	handler := HandleProjectDelete(app)

	req := httptest.NewRequest(http.MethodDelete, "/projects/nonexistent", nil)
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

func TestHandleProjectDelete_DeleteBOQs(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "With BOQs")
	boq := testhelpers.CreateTestBOQ(t, app, proj.Id, "My BOQ")

	handler := HandleProjectDelete(app)

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+proj.Id+"?delete_boqs=true", nil)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("id", proj.Id)
	rec := httptest.NewRecorder()

	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	// BOQ should be deleted
	_, err := app.FindRecordById("boqs", boq.Id)
	if err == nil {
		t.Error("expected BOQ to be deleted")
	}
}

func TestHandleProjectDelete_UnlinkBOQs(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Unlink BOQs")
	boq := testhelpers.CreateTestBOQ(t, app, proj.Id, "Orphan BOQ")

	handler := HandleProjectDelete(app)

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+proj.Id+"?delete_boqs=false", nil)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("id", proj.Id)
	rec := httptest.NewRecorder()

	e := newTestRequestEvent(app, req, rec)

	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	// BOQ should still exist but project field should be empty
	orphan, err := app.FindRecordById("boqs", boq.Id)
	if err != nil {
		t.Fatalf("BOQ should still exist: %v", err)
	}
	if orphan.GetString("project") != "" {
		t.Errorf("expected BOQ project to be empty, got %q", orphan.GetString("project"))
	}
}
