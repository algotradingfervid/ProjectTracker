package handlers

import (
	"fmt"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
)

// HandleDCIssue handles POST to issue a delivery challan.
func HandleDCIssue(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		dcId := e.Request.PathValue("id")

		// Verify DC belongs to project
		dc, err := app.FindRecordById("delivery_challans", dcId)
		if err != nil {
			return ErrorToast(e, http.StatusNotFound, "Delivery challan not found")
		}
		if dc.GetString("project") != projectId {
			return ErrorToast(e, http.StatusNotFound, "Delivery challan not found in this project")
		}

		if err := services.IssueSingleDC(app, dcId); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Issue failed: "+err.Error())
		}

		SetToast(e, "success", "Delivery challan issued successfully")

		// Redirect back to the DC detail to show updated status
		redirectURL := fmt.Sprintf("/projects/%s/dcs/%s", projectId, dcId)
		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", redirectURL)
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, redirectURL)
	}
}
