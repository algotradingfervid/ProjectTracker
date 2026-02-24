package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
)

// HandleAddressTemplateDownload serves the Excel template for address import.
// Route: GET /projects/{projectId}/addresses/{type}/template
func HandleAddressTemplateDownload(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		addressType := e.Request.PathValue("type") // "ship_to" or "install_at"

		if projectID == "" {
			return e.String(http.StatusBadRequest, "Missing project ID")
		}
		if addressType != "ship_to" && addressType != "install_at" {
			return e.String(http.StatusBadRequest, "Invalid address type. Must be ship_to or install_at")
		}

		// Verify project exists
		if _, err := app.FindRecordById("projects", projectID); err != nil {
			return e.String(http.StatusNotFound, "Project not found")
		}

		xlsxBytes, err := services.GenerateAddressTemplate(app, projectID, addressType)
		if err != nil {
			log.Printf("address_template: failed to generate: %v", err)
			return e.String(http.StatusInternalServerError, "Failed to generate template")
		}

		typeName := "ShipTo"
		if addressType == "install_at" {
			typeName = "InstallAt"
		}
		filename := fmt.Sprintf("%s_Template_%d.xlsx", typeName, time.Now().Year())

		e.Response.Header().Set("Content-Type",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		e.Response.Header().Set("Content-Disposition",
			fmt.Sprintf(`attachment; filename="%s"`, filename))
		e.Response.Write(xlsxBytes)
		return nil
	}
}
