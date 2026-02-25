package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
)

// HandlePOExportPDF returns a handler that generates and downloads a PDF for a Purchase Order.
func HandlePOExportPDF(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		id := e.Request.PathValue("id")
		if id == "" {
			return e.String(http.StatusBadRequest, "Missing PO ID")
		}

		// Verify PO belongs to project
		po, err := app.FindRecordById("purchase_orders", id)
		if err != nil {
			log.Printf("po_export: PO not found %s: %v", id, err)
			return e.String(http.StatusNotFound, "Purchase order not found")
		}
		if po.GetString("project") != projectId {
			log.Printf("po_export: PO %s does not belong to project %s", id, projectId)
			return e.String(http.StatusNotFound, "Purchase order not found")
		}

		data, err := services.BuildPOExportData(app, id)
		if err != nil {
			log.Printf("po_export: failed to build data: %v", err)
			return e.String(http.StatusInternalServerError, "Failed to build PO data")
		}

		pdfBytes, err := services.GeneratePOPDF(data)
		if err != nil {
			log.Printf("po_export: failed to generate PDF: %v", err)
			return e.String(http.StatusInternalServerError, "Failed to generate PDF")
		}

		filename := fmt.Sprintf("%s.pdf", sanitizeFilename(data.PONumber))

		e.Response.Header().Set("Content-Type", "application/pdf")
		e.Response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		e.Response.Write(pdfBytes)
		return nil
	}
}
