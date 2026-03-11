package handlers

import (
	"fmt"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// HandleDCEdit redirects a draft DC back into the wizard for editing.
// Currently redirects to the DC wizard step 1 with the existing template pre-selected.
func HandleDCEdit(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		dcId := e.Request.PathValue("id")

		dc, err := app.FindRecordById("delivery_challans", dcId)
		if err != nil {
			return ErrorToast(e, http.StatusNotFound, "Delivery challan not found")
		}
		if dc.GetString("project") != projectId {
			return ErrorToast(e, http.StatusNotFound, "Delivery challan not found in this project")
		}
		if dc.GetString("status") != "draft" {
			return ErrorToast(e, http.StatusBadRequest, "Only draft DCs can be edited")
		}

		// For now, redirect to the create wizard — user will need to re-enter data
		// A full edit implementation would pre-populate the wizard with existing data
		SetToast(e, "info", "Please create a new DC — edit with pre-population coming soon. Delete the old DC if needed.")
		redirectURL := fmt.Sprintf("/projects/%s/dcs/create", projectId)
		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", redirectURL)
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, redirectURL)
	}
}
