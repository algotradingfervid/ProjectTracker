package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
)

// buildExportData fetches the BOQ and all nested items, returning an ExportData struct.
func buildExportData(app *pocketbase.PocketBase, boqID string) (services.ExportData, error) {
	boqRecord, err := app.FindRecordById("boqs", boqID)
	if err != nil {
		return services.ExportData{}, fmt.Errorf("BOQ not found: %w", err)
	}

	mainItemsCol, err := app.FindCollectionByNameOrId("main_boq_items")
	if err != nil {
		return services.ExportData{}, fmt.Errorf("collection not found: %w", err)
	}

	subItemsCol, err := app.FindCollectionByNameOrId("sub_items")
	if err != nil {
		return services.ExportData{}, fmt.Errorf("collection not found: %w", err)
	}

	subSubItemsCol, err := app.FindCollectionByNameOrId("sub_sub_items")
	if err != nil {
		return services.ExportData{}, fmt.Errorf("collection not found: %w", err)
	}

	mainItems, err := app.FindRecordsByFilter(mainItemsCol, "boq = {:boqId}", "sort_order", 0, 0, map[string]any{"boqId": boqID})
	if err != nil {
		mainItems = nil
	}

	var rows []services.ExportRow
	var totalQuoted, totalBudgeted float64

	for i, mi := range mainItems {
		qty := mi.GetFloat("qty")
		quotedPrice := mi.GetFloat("quoted_price")
		budgetedPrice := mi.GetFloat("budgeted_price")

		totalQuoted += qty * quotedPrice
		totalBudgeted += budgetedPrice

		// Compute budgeted per unit for display
		budgetedPerUnit := budgetedPrice
		if qty != 0 {
			budgetedPerUnit = budgetedPrice / qty
		}

		// Main item row
		rows = append(rows, services.ExportRow{
			Level:         0,
			Index:         fmt.Sprintf("%d", i+1),
			Description:   mi.GetString("description"),
			Qty:           qty,
			UOM:           mi.GetString("uom"),
			QuotedPrice:   quotedPrice,
			BudgetedPrice: budgetedPerUnit,
			HSNCode:       mi.GetString("hsn_code"),
			GSTPercent:    mi.GetFloat("gst_percent"),
		})

		// Fetch sub-items
		subItems, err := app.FindRecordsByFilter(subItemsCol, "main_item = {:mainId}", "sort_order", 0, 0, map[string]any{"mainId": mi.Id})
		if err != nil {
			subItems = nil
		}

		for j, si := range subItems {
			rows = append(rows, services.ExportRow{
				Level:         1,
				Index:         fmt.Sprintf("%d.%d", i+1, j+1),
				Description:   si.GetString("description"),
				Qty:           si.GetFloat("qty_per_unit"),
				UOM:           si.GetString("uom"),
				QuotedPrice:   si.GetFloat("unit_price"),
				BudgetedPrice: si.GetFloat("budgeted_price"),
				HSNCode:       si.GetString("hsn_code"),
				GSTPercent:    si.GetFloat("gst_percent"),
			})

			// Fetch sub-sub-items
			subSubItems, err := app.FindRecordsByFilter(subSubItemsCol, "sub_item = {:subId}", "sort_order", 0, 0, map[string]any{"subId": si.Id})
			if err != nil {
				subSubItems = nil
			}

			for k, ssi := range subSubItems {
				rows = append(rows, services.ExportRow{
					Level:         2,
					Index:         fmt.Sprintf("%d.%d.%d", i+1, j+1, k+1),
					Description:   ssi.GetString("description"),
					Qty:           ssi.GetFloat("qty_per_unit"),
					UOM:           ssi.GetString("uom"),
					QuotedPrice:   ssi.GetFloat("unit_price"),
					BudgetedPrice: ssi.GetFloat("budgeted_price"),
					HSNCode:       ssi.GetString("hsn_code"),
					GSTPercent:    ssi.GetFloat("gst_percent"),
				})
			}
		}
	}

	margin := totalQuoted - totalBudgeted
	var marginPercent float64
	if totalQuoted != 0 {
		marginPercent = (margin / totalQuoted) * 100
	}

	createdDate := "—"
	if dt := boqRecord.GetDateTime("created"); !dt.IsZero() {
		createdDate = dt.Time().Format("02 Jan 2006")
	}

	return services.ExportData{
		Title:           boqRecord.GetString("title"),
		ReferenceNumber: boqRecord.GetString("reference_number"),
		CreatedDate:     createdDate,
		Rows:            rows,
		TotalQuoted:     totalQuoted,
		TotalBudgeted:   totalBudgeted,
		Margin:          margin,
		MarginPercent:   marginPercent,
	}, nil
}

// sanitizeFilename removes characters that are unsafe for filenames.
func sanitizeFilename(s string) string {
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	s = strings.ReplaceAll(s, ":", "-")
	return s
}

// HandleBOQExportExcel returns a handler that generates and downloads an Excel file for a BOQ.
func HandleBOQExportExcel(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		boqID := e.Request.PathValue("id")
		if boqID == "" {
			return e.String(http.StatusBadRequest, "Missing BOQ ID")
		}

		data, err := buildExportData(app, boqID)
		if err != nil {
			log.Printf("export_excel: %v", err)
			return e.String(http.StatusNotFound, "BOQ not found")
		}

		xlsxBytes, err := services.GenerateExcel(data)
		if err != nil {
			log.Printf("export_excel: failed to generate: %v", err)
			return e.String(http.StatusInternalServerError, "Failed to generate Excel file")
		}

		filename := fmt.Sprintf("BOQ_%s_%d.xlsx", sanitizeFilename(data.Title), time.Now().Year())

		e.Response.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		e.Response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		e.Response.Write(xlsxBytes)
		return nil
	}
}

// HandleBOQExportPDF returns a handler that generates and downloads a PDF file for a BOQ.
func HandleBOQExportPDF(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		boqID := e.Request.PathValue("id")
		if boqID == "" {
			return e.String(http.StatusBadRequest, "Missing BOQ ID")
		}

		data, err := buildExportData(app, boqID)
		if err != nil {
			log.Printf("export_pdf: %v", err)
			return e.String(http.StatusNotFound, "BOQ not found")
		}

		pdfBytes, err := services.GeneratePDF(data)
		if err != nil {
			log.Printf("export_pdf: failed to generate: %v", err)
			return e.String(http.StatusInternalServerError, "Failed to generate PDF file")
		}

		filename := fmt.Sprintf("BOQ_%s_%d.pdf", sanitizeFilename(data.Title), time.Now().Year())

		e.Response.Header().Set("Content-Type", "application/pdf")
		e.Response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		e.Response.Write(pdfBytes)
		return nil
	}
}
