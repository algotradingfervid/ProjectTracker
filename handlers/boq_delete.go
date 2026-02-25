package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// HandleBOQDelete returns a handler that deletes a BOQ and all its items (via cascade).
func HandleBOQDelete(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		boqID := e.Request.PathValue("id")
		if boqID == "" {
			return ErrorToast(e, http.StatusBadRequest, "Missing BOQ ID")
		}

		// Find the BOQ record
		boqRecord, err := app.FindRecordById("boqs", boqID)
		if err != nil {
			log.Printf("boq_delete: could not find BOQ %s: %v", boqID, err)
			return ErrorToast(e, http.StatusNotFound, "BOQ not found")
		}

		// Delete the BOQ — cascade delete handles main_boq_items, sub_items, sub_sub_items
		if err := app.Delete(boqRecord); err != nil {
			log.Printf("boq_delete: failed to delete BOQ %s: %v", boqID, err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		// Redirect to BOQ list
		SetToast(e, "success", "BOQ deleted successfully")
		boqListURL := fmt.Sprintf("/projects/%s/boq", projectID)
		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", boqListURL)
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, boqListURL)
	}
}
