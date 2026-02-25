package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func HandlePODelete(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		id := e.Request.PathValue("id")

		po, err := app.FindRecordById("purchase_orders", id)
		if err != nil {
			return ErrorToast(e, http.StatusNotFound, "Purchase order not found")
		}

		if po.GetString("project") != projectId {
			return ErrorToast(e, http.StatusNotFound, "Purchase order not found")
		}

		status := po.GetString("status")
		if status != "draft" && status != "cancelled" {
			return ErrorToast(e, http.StatusBadRequest,
				fmt.Sprintf("Cannot delete a %s purchase order", status))
		}

		if err := app.Delete(po); err != nil {
			log.Printf("Error deleting purchase order %s: %v", id, err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		redirectURL := fmt.Sprintf("/projects/%s/po", projectId)

		SetToast(e, "success", "Purchase order deleted successfully")

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", redirectURL)
			return e.String(http.StatusOK, "")
		}

		return e.Redirect(http.StatusFound, redirectURL)
	}
}
