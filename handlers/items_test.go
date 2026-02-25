package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"projectcreation/testhelpers"
)

func TestHandleDeleteMainItem_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Item Delete Project")
	boq := testhelpers.CreateTestBOQ(t, app, proj.Id, "Item BOQ")
	mainItem := testhelpers.CreateTestMainBOQItem(t, app, boq.Id, "Delete This")
	handler := HandleDeleteMainItem(app)
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/projects/%s/boq/%s/items/%s", proj.Id, boq.Id, mainItem.Id), nil)
	req.SetPathValue("projectId", proj.Id)
	req.SetPathValue("id", boq.Id)
	req.SetPathValue("itemId", mainItem.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	// Item should be deleted
	_, err := app.FindRecordById("main_boq_items", mainItem.Id)
	if err == nil {
		t.Error("expected main item to be deleted")
	}
}

func TestHandleDeleteMainItem_NotFound(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Item NF Project")
	boq := testhelpers.CreateTestBOQ(t, app, proj.Id, "NF BOQ")
	handler := HandleDeleteMainItem(app)
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/projects/%s/boq/%s/items/nonexistent", proj.Id, boq.Id), nil)
	req.SetPathValue("projectId", proj.Id)
	req.SetPathValue("id", boq.Id)
	req.SetPathValue("itemId", "nonexistent")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != 404 {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleDeleteSubItem_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Sub Delete Project")
	boq := testhelpers.CreateTestBOQ(t, app, proj.Id, "Sub BOQ")
	mainItem := testhelpers.CreateTestMainBOQItem(t, app, boq.Id, "Parent Main")
	subItem := testhelpers.CreateTestSubItem(t, app, mainItem.Id, "Delete Sub")
	handler := HandleDeleteSubItem(app)
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/projects/%s/boq/%s/sub-items/%s", proj.Id, boq.Id, subItem.Id), nil)
	req.SetPathValue("projectId", proj.Id)
	req.SetPathValue("id", boq.Id)
	req.SetPathValue("subItemId", subItem.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	_, err := app.FindRecordById("sub_items", subItem.Id)
	if err == nil {
		t.Error("expected sub item to be deleted")
	}
}

func TestHandleDeleteSubSubItem_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "SubSub Delete Project")
	boq := testhelpers.CreateTestBOQ(t, app, proj.Id, "SubSub BOQ")
	mainItem := testhelpers.CreateTestMainBOQItem(t, app, boq.Id, "Top Main")
	subItem := testhelpers.CreateTestSubItem(t, app, mainItem.Id, "Mid Sub")
	subSubItem := testhelpers.CreateTestSubSubItem(t, app, subItem.Id, "Delete SubSub")
	handler := HandleDeleteSubSubItem(app)
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/projects/%s/boq/%s/sub-sub-items/%s", proj.Id, boq.Id, subSubItem.Id), nil)
	req.SetPathValue("projectId", proj.Id)
	req.SetPathValue("id", boq.Id)
	req.SetPathValue("subSubItemId", subSubItem.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	_, err := app.FindRecordById("sub_sub_items", subSubItem.Id)
	if err == nil {
		t.Error("expected sub-sub item to be deleted")
	}
}

func TestHandleExpandMainItem_Success(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Expand Project")
	boq := testhelpers.CreateTestBOQ(t, app, proj.Id, "Expand BOQ")
	mainItem := testhelpers.CreateTestMainBOQItem(t, app, boq.Id, "Expandable")
	testhelpers.CreateTestSubItem(t, app, mainItem.Id, "Visible Sub")
	handler := HandleExpandMainItem(app)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/projects/%s/boq/%s/items/%s/expand", proj.Id, boq.Id, mainItem.Id), nil)
	req.SetPathValue("projectId", proj.Id)
	req.SetPathValue("id", boq.Id)
	req.SetPathValue("itemId", mainItem.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	testhelpers.AssertHTMLContains(t, rec.Body.String(), "Visible Sub")
}

func TestHandlePatchMainItem_Description(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Patch Project")
	boq := testhelpers.CreateTestBOQ(t, app, proj.Id, "Patch BOQ")
	mainItem := testhelpers.CreateTestMainBOQItem(t, app, boq.Id, "Original Desc")
	handler := HandlePatchMainItem(app)
	form := url.Values{}
	form.Set("description", "Updated Desc")
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/projects/%s/boq/%s/items/%s", proj.Id, boq.Id, mainItem.Id), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("projectId", proj.Id)
	req.SetPathValue("id", boq.Id)
	req.SetPathValue("itemId", mainItem.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	// Verify JSON response
	var result map[string]float64
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	// Verify in DB
	updated, _ := app.FindRecordById("main_boq_items", mainItem.Id)
	if updated.GetString("description") != "Updated Desc" {
		t.Errorf("description not updated, got %q", updated.GetString("description"))
	}
}

func TestHandlePatchMainItem_Qty(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Qty Patch Project")
	boq := testhelpers.CreateTestBOQ(t, app, proj.Id, "Qty BOQ")
	mainItem := testhelpers.CreateTestMainBOQItem(t, app, boq.Id, "Qty Item")
	handler := HandlePatchMainItem(app)
	form := url.Values{}
	form.Set("qty", "25")
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/projects/%s/boq/%s/items/%s", proj.Id, boq.Id, mainItem.Id), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", boq.Id)
	req.SetPathValue("itemId", mainItem.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	updated, _ := app.FindRecordById("main_boq_items", mainItem.Id)
	if updated.GetFloat("qty") != 25 {
		t.Errorf("qty not updated, got %v", updated.GetFloat("qty"))
	}
}

func TestHandlePatchMainItem_NotFound(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	boq := testhelpers.CreateTestBOQ(t, app, testhelpers.CreateTestProject(t, app, "NF").Id, "NF BOQ")
	handler := HandlePatchMainItem(app)
	req := httptest.NewRequest(http.MethodPatch, "/boq/"+boq.Id+"/items/nonexistent", nil)
	req.SetPathValue("id", boq.Id)
	req.SetPathValue("itemId", "nonexistent")
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != 404 {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandlePatchSubItem_UnitPrice(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "Sub Patch Project")
	boq := testhelpers.CreateTestBOQ(t, app, proj.Id, "Sub Patch BOQ")
	mainItem := testhelpers.CreateTestMainBOQItem(t, app, boq.Id, "Parent")
	subItem := testhelpers.CreateTestSubItem(t, app, mainItem.Id, "Patchable Sub")
	handler := HandlePatchSubItem(app)
	form := url.Values{}
	form.Set("unit_price", "350")
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/boq/%s/sub-items/%s", boq.Id, subItem.Id), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", boq.Id)
	req.SetPathValue("subItemId", subItem.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var result map[string]float64
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestHandlePatchSubSubItem_UnitPrice(t *testing.T) {
	app := testhelpers.NewTestApp(t)
	proj := testhelpers.CreateTestProject(t, app, "SubSub Patch Project")
	boq := testhelpers.CreateTestBOQ(t, app, proj.Id, "SubSub Patch BOQ")
	mainItem := testhelpers.CreateTestMainBOQItem(t, app, boq.Id, "Top")
	subItem := testhelpers.CreateTestSubItem(t, app, mainItem.Id, "Mid")
	subSubItem := testhelpers.CreateTestSubSubItem(t, app, subItem.Id, "Bottom")
	handler := HandlePatchSubSubItem(app)
	form := url.Values{}
	form.Set("unit_price", "150")
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/boq/%s/sub-sub-items/%s", boq.Id, subSubItem.Id), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", boq.Id)
	req.SetPathValue("subSubItemId", subSubItem.Id)
	rec := httptest.NewRecorder()
	e := newTestRequestEvent(app, req, rec)
	if err := handler(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var result map[string]float64
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}
