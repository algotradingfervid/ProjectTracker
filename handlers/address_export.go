package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
)

// HandleAddressExportExcel exports all addresses of a given type for a project as an Excel file.
// Route: GET /projects/{projectId}/addresses/{type}/export
func HandleAddressExportExcel(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		addrTypeSlug := e.Request.PathValue("type")

		if projectID == "" || addrTypeSlug == "" {
			return e.String(http.StatusBadRequest, "Missing project ID or address type")
		}

		// Convert URL slug (e.g. "bill-from") to DB type (e.g. "bill_from")
		addrType := AddressType(strings.ReplaceAll(addrTypeSlug, "-", "_"))

		typeLabel, ok := AddressTypeDisplayLabels[addrType]
		if !ok {
			return e.String(http.StatusBadRequest, "Invalid address type")
		}

		// Fetch project record for the name
		project, err := app.FindRecordById("projects", projectID)
		if err != nil {
			log.Printf("address_export: project not found %s: %v", projectID, err)
			return e.String(http.StatusNotFound, "Project not found")
		}
		projectName := project.GetString("name")

		// Fetch addresses collection
		addressesCol, err := app.FindCollectionByNameOrId("addresses")
		if err != nil {
			log.Printf("address_export: addresses collection not found: %v", err)
			return e.String(http.StatusInternalServerError, "Address collection not found")
		}

		// Query all addresses for this project and type
		records, err := app.FindRecordsByFilter(
			addressesCol,
			"project = {:projectId} && address_type = {:addressType}",
			"created",
			0, 0,
			map[string]any{
				"projectId":   projectID,
				"addressType": string(addrType),
			},
		)
		if err != nil {
			log.Printf("address_export: query failed: %v", err)
			records = nil
		}

		// Build columns and rows
		columns := services.GetAddressColumns(string(addrType))
		var rows []map[string]string

		for _, rec := range records {
			row := make(map[string]string)
			for _, col := range columns {
				if col.Field == "_ship_to_parent_name" {
					// Resolve Ship To parent company name
					parentID := rec.GetString("ship_to_parent")
					if parentID != "" {
						parentRec, err := app.FindRecordById("addresses", parentID)
						if err == nil {
							row[col.Field] = parentRec.GetString("company_name")
						} else {
							row[col.Field] = "(unknown)"
						}
					} else {
						row[col.Field] = ""
					}
				} else {
					row[col.Field] = rec.GetString(col.Field)
				}
			}
			rows = append(rows, row)
		}

		exportData := services.AddressExportData{
			ProjectName: projectName,
			AddressType: string(addrType),
			TypeLabel:   typeLabel,
			Columns:     columns,
			Rows:        rows,
		}

		xlsxBytes, err := services.GenerateAddressExcel(exportData)
		if err != nil {
			log.Printf("address_export: generate failed: %v", err)
			return e.String(http.StatusInternalServerError, "Failed to generate Excel file")
		}

		// Filename: {ProjectName}_{AddressType}_Addresses.xlsx
		filename := fmt.Sprintf("%s_%s_Addresses.xlsx",
			sanitizeFilename(projectName),
			strings.ReplaceAll(typeLabel, " ", ""),
		)

		e.Response.Header().Set("Content-Type",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		e.Response.Header().Set("Content-Disposition",
			fmt.Sprintf(`attachment; filename="%s"`, filename))
		e.Response.Write(xlsxBytes)
		return nil
	}
}
