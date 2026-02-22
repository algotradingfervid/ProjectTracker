package handlers

import (
	"log"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
	"projectcreation/templates"
)

// HandleBOQList returns a handler that renders the BOQ list page.
func HandleBOQList(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		boqsCol, err := app.FindCollectionByNameOrId("boqs")
		if err != nil {
			log.Printf("boq_list: could not find boqs collection: %v", err)
			return e.String(500, "Internal error")
		}

		records, err := app.FindAllRecords(boqsCol)
		if err != nil {
			log.Printf("boq_list: could not query boqs: %v", err)
			return e.String(500, "Internal error")
		}

		mainItemsCol, err := app.FindCollectionByNameOrId("main_boq_items")
		if err != nil {
			log.Printf("boq_list: could not find main_boq_items collection: %v", err)
			return e.String(500, "Internal error")
		}

		var items []templates.BOQListItem
		var grandTotalQuoted, grandTotalBudgeted float64

		for _, rec := range records {
			boqID := rec.Id

			// Find all main items for this BOQ
			mainItems, err := app.FindRecordsByFilter(mainItemsCol, "boq = {:boqId}", "-sort_order", 0, 0, map[string]any{"boqId": boqID})
			if err != nil {
				log.Printf("boq_list: could not query main items for BOQ %s: %v", boqID, err)
				mainItems = nil
			}

			var totalQuoted, totalBudgeted float64
			for _, mi := range mainItems {
				qty := mi.GetFloat("qty")
				quotedPrice := mi.GetFloat("quoted_price")
				budgetedPrice := mi.GetFloat("budgeted_price")

				totalQuoted += qty * quotedPrice
				totalBudgeted += budgetedPrice
			}

			grandTotalQuoted += totalQuoted
			grandTotalBudgeted += totalBudgeted

			createdDate := "—"
			if dt := rec.GetDateTime("created"); !dt.IsZero() {
				createdDate = dt.Time().Format("02 Jan 2006")
			} else if dt := rec.GetDateTime("updated"); !dt.IsZero() {
				createdDate = dt.Time().Format("02 Jan 2006")
			}

			items = append(items, templates.BOQListItem{
				ID:            boqID,
				Title:         rec.GetString("title"),
				RefNumber:     rec.GetString("reference_number"),
				CreatedDate:   createdDate,
				TotalQuoted:   services.FormatINR(totalQuoted),
				TotalBudgeted: services.FormatINR(totalBudgeted),
				ItemCount:     len(mainItems),
				IsOverBudget:  totalBudgeted > totalQuoted,
			})
		}

		margin := grandTotalQuoted - grandTotalBudgeted

		data := templates.BOQListData{
			Items:            items,
			TotalBOQs:        len(records),
			SumQuoted:        services.FormatINR(grandTotalQuoted),
			SumBudgeted:      services.FormatINR(grandTotalBudgeted),
			Margin:           services.FormatINR(margin),
			IsPositiveMargin: margin >= 0,
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.BOQListContent(data)
		} else {
			component = templates.BOQListPage(data)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}
