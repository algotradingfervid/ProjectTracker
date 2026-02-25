package handlers

import (
	"log"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/templates"
)

// HandleVendorLink links a vendor to a project (creates project_vendors record).
// POST /projects/{projectId}/vendors/{id}/link
func HandleVendorLink(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		vendorID := e.Request.PathValue("id")

		if projectID == "" || vendorID == "" {
			return ErrorToast(e, http.StatusBadRequest, "Missing project or vendor ID")
		}

		// Verify project exists
		if _, err := app.FindRecordById("projects", projectID); err != nil {
			return ErrorToast(e, http.StatusNotFound, "Project not found")
		}

		// Verify vendor exists
		if _, err := app.FindRecordById("vendors", vendorID); err != nil {
			return ErrorToast(e, http.StatusNotFound, "Vendor not found")
		}

		// Check if already linked (idempotent)
		existing, _ := app.FindRecordsByFilter(
			"project_vendors",
			"project = {:projectId} && vendor = {:vendorId}",
			"", 1, 0,
			map[string]any{"projectId": projectID, "vendorId": vendorID},
		)

		if len(existing) == 0 {
			// Create link
			pvCol, err := app.FindCollectionByNameOrId("project_vendors")
			if err != nil {
				log.Printf("vendor_link: could not find project_vendors collection: %v", err)
				return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
			}

			link := core.NewRecord(pvCol)
			link.Set("project", projectID)
			link.Set("vendor", vendorID)
			if err := app.Save(link); err != nil {
				log.Printf("vendor_link: could not save link: %v", err)
				return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
			}
		}

		SetToast(e, "success", "Vendor linked to project")

		// Return updated toggle button partial
		component := templates.VendorLinkToggle(projectID, vendorID, true)
		return component.Render(e.Request.Context(), e.Response)
	}
}

// HandleVendorUnlink removes a vendor from a project (deletes project_vendors record).
// DELETE /projects/{projectId}/vendors/{id}/link
func HandleVendorUnlink(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		vendorID := e.Request.PathValue("id")

		if projectID == "" || vendorID == "" {
			return ErrorToast(e, http.StatusBadRequest, "Missing project or vendor ID")
		}

		// Check if vendor has POs in this project
		pos, _ := app.FindRecordsByFilter(
			"purchase_orders",
			"project = {:projectId} && vendor = {:vendorId}",
			"", 1, 0,
			map[string]any{"projectId": projectID, "vendorId": vendorID},
		)
		if len(pos) > 0 {
			return ErrorToast(e, http.StatusConflict, "Cannot unlink vendor — it has purchase orders in this project")
		}

		// Find and delete link (idempotent - no error if not linked)
		links, _ := app.FindRecordsByFilter(
			"project_vendors",
			"project = {:projectId} && vendor = {:vendorId}",
			"", 1, 0,
			map[string]any{"projectId": projectID, "vendorId": vendorID},
		)
		for _, link := range links {
			if err := app.Delete(link); err != nil {
				log.Printf("vendor_unlink: could not delete link: %v", err)
				return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
			}
		}

		SetToast(e, "success", "Vendor unlinked from project")

		// Return updated toggle button partial
		component := templates.VendorLinkToggle(projectID, vendorID, false)
		return component.Render(e.Request.Context(), e.Response)
	}
}
