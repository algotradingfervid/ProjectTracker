package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
	"projectcreation/templates"
)

// HandleBOQCreate returns a handler that renders the BOQ creation form.
func HandleBOQCreate(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		data := templates.BOQCreateData{
			Date:       time.Now().Format("2006-01-02"),
			UOMOptions: services.UOMOptions,
			GSTOptions: services.GSTOptions,
			Errors:     make(map[string]string),
		}
		component := templates.BOQCreatePage(data)
		return component.Render(e.Request.Context(), e.Response)
	}
}

// HandleBOQSave returns a handler that processes the BOQ creation form submission.
func HandleBOQSave(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := e.Request.ParseForm(); err != nil {
			log.Printf("boq_create: could not parse form: %v", err)
			return e.String(http.StatusBadRequest, "Invalid form data")
		}

		title := strings.TrimSpace(e.Request.FormValue("title"))
		refNumber := strings.TrimSpace(e.Request.FormValue("reference_number"))

		// Validate required fields
		errors := make(map[string]string)
		if title == "" {
			errors["title"] = "BOQ title is required"
		}

		// Check for duplicate title
		if title != "" {
			existing, _ := app.FindRecordsByFilter("boqs", "title = {:title}", "", 1, 0, map[string]any{"title": title})
			if len(existing) > 0 {
				errors["title"] = "A BOQ with this title already exists"
			}
		}

		// Check for duplicate reference number
		if refNumber != "" {
			existing, _ := app.FindRecordsByFilter("boqs", "reference_number = {:ref}", "", 1, 0, map[string]any{"ref": refNumber})
			if len(existing) > 0 {
				errors["reference_number"] = "A BOQ with this reference number already exists"
			}
		}

		if len(errors) > 0 {
			data := templates.BOQCreateData{
				Title:           title,
				ReferenceNumber: refNumber,
				Date:            e.Request.FormValue("date"),
				UOMOptions:      services.UOMOptions,
				GSTOptions:      services.GSTOptions,
				Errors:          errors,
			}
			component := templates.BOQCreatePage(data)
			return component.Render(e.Request.Context(), e.Response)
		}

		// Create BOQ record
		boqsCol, err := app.FindCollectionByNameOrId("boqs")
		if err != nil {
			log.Printf("boq_create: could not find boqs collection: %v", err)
			return e.String(http.StatusInternalServerError, "Internal error")
		}

		boqRecord := core.NewRecord(boqsCol)
		boqRecord.Set("title", title)
		boqRecord.Set("reference_number", refNumber)

		if err := app.Save(boqRecord); err != nil {
			log.Printf("boq_create: could not save BOQ: %v", err)
			return e.String(http.StatusInternalServerError, "Internal error")
		}

		boqID := boqRecord.Id

		// Get collections
		mainItemsCol, err := app.FindCollectionByNameOrId("main_boq_items")
		if err != nil {
			log.Printf("boq_create: could not find main_boq_items collection: %v", err)
			return e.Redirect(http.StatusFound, "/boq/"+boqID)
		}

		subItemsCol, err := app.FindCollectionByNameOrId("sub_items")
		if err != nil {
			log.Printf("boq_create: could not find sub_items collection: %v", err)
			subItemsCol = nil
		}

		subSubItemsCol, err := app.FindCollectionByNameOrId("sub_sub_items")
		if err != nil {
			log.Printf("boq_create: could not find sub_sub_items collection: %v", err)
			subSubItemsCol = nil
		}

		// Parse and save main items with budgeted price rollup:
		// sub-sub budgeted prices → sum into sub-item budgeted price
		// sub-item budgeted prices → sum into main item budgeted price
		for i := 0; ; i++ {
			prefix := fmt.Sprintf("items[%d].", i)
			desc := strings.TrimSpace(e.Request.FormValue(prefix + "description"))
			if desc == "" {
				break
			}

			qty, _ := strconv.ParseFloat(e.Request.FormValue(prefix+"qty"), 64)
			uom := e.Request.FormValue(prefix + "uom")
			if uom == "" {
				uom = "Nos"
			}
			quotedPrice, _ := strconv.ParseFloat(e.Request.FormValue(prefix+"quoted_price"), 64)
			manualBudgeted, _ := strconv.ParseFloat(e.Request.FormValue(prefix+"budgeted_price"), 64)
			gstPct, _ := strconv.ParseFloat(e.Request.FormValue(prefix+"gst_percent"), 64)
			if gstPct == 0 && e.Request.FormValue(prefix+"gst_percent") == "" {
				gstPct = 18
			}

			itemRecord := core.NewRecord(mainItemsCol)
			itemRecord.Set("boq", boqID)
			itemRecord.Set("sort_order", i+1)
			itemRecord.Set("description", desc)
			itemRecord.Set("qty", qty)
			itemRecord.Set("uom", uom)
			itemRecord.Set("quoted_price", quotedPrice)
			itemRecord.Set("gst_percent", gstPct)
			// budgeted_price set after sub-items are processed
			itemRecord.Set("budgeted_price", 0)

			if err := app.Save(itemRecord); err != nil {
				log.Printf("boq_create: could not save main item %d: %v", i+1, err)
				continue
			}

			mainItemID := itemRecord.Id
			var mainItemBudgetedTotal float64

			// Parse and save sub items
			if subItemsCol != nil {
				for si := 0; ; si++ {
					subPrefix := fmt.Sprintf("items[%d].subs[%d].", i, si)
					subDesc := strings.TrimSpace(e.Request.FormValue(subPrefix + "description"))
					if subDesc == "" {
						break
					}

					subType := e.Request.FormValue(subPrefix + "type")
					if subType == "" {
						subType = "product"
					}
					subQty, _ := strconv.ParseFloat(e.Request.FormValue(subPrefix+"qty_per_unit"), 64)
					subUOM := e.Request.FormValue(subPrefix + "uom")
					if subUOM == "" {
						subUOM = "Nos"
					}
					subManualBudgeted, _ := strconv.ParseFloat(e.Request.FormValue(subPrefix+"budgeted_price"), 64)
					subGst, _ := strconv.ParseFloat(e.Request.FormValue(subPrefix+"gst_percent"), 64)
					if subGst == 0 && e.Request.FormValue(subPrefix+"gst_percent") == "" {
						subGst = 18
					}

					// Pre-calculate sub-sub-item totals before saving the sub-item,
					// so unit_price has a valid value on first save (required field).
					type subSubEntry struct {
						ssType     string
						ssDesc     string
						ssQty      float64
						ssUOM      string
						ssBudgeted float64
						ssGst      float64
					}
					var subSubEntries []subSubEntry
					var subItemBudgetedTotal float64

					if subSubItemsCol != nil {
						for ssi := 0; ; ssi++ {
							ssPrefix := fmt.Sprintf("items[%d].subs[%d].sub_subs[%d].", i, si, ssi)
							ssDesc := strings.TrimSpace(e.Request.FormValue(ssPrefix + "description"))
							if ssDesc == "" {
								break
							}

							ssType := e.Request.FormValue(ssPrefix + "type")
							if ssType == "" {
								ssType = "product"
							}
							ssQty, _ := strconv.ParseFloat(e.Request.FormValue(ssPrefix+"qty_per_unit"), 64)
							ssUOM := e.Request.FormValue(ssPrefix + "uom")
							if ssUOM == "" {
								ssUOM = "Nos"
							}
							ssBudgeted, _ := strconv.ParseFloat(e.Request.FormValue(ssPrefix+"budgeted_price"), 64)
							ssGst, _ := strconv.ParseFloat(e.Request.FormValue(ssPrefix+"gst_percent"), 64)
							if ssGst == 0 && e.Request.FormValue(ssPrefix+"gst_percent") == "" {
								ssGst = 18
							}

							subSubEntries = append(subSubEntries, subSubEntry{
								ssType: ssType, ssDesc: ssDesc, ssQty: ssQty,
								ssUOM: ssUOM, ssBudgeted: ssBudgeted, ssGst: ssGst,
							})
							subItemBudgetedTotal += ssBudgeted
						}
					}

					// Determine unit_price before first save so the required field is satisfied
					var subUnitPrice float64
					if subItemBudgetedTotal > 0 {
						subUnitPrice = subItemBudgetedTotal
					} else if subManualBudgeted > 0 {
						subUnitPrice = subManualBudgeted
					}

					subRecord := core.NewRecord(subItemsCol)
					subRecord.Set("main_item", mainItemID)
					subRecord.Set("sort_order", si+1)
					subRecord.Set("type", subType)
					subRecord.Set("description", subDesc)
					subRecord.Set("qty_per_unit", subQty)
					subRecord.Set("uom", subUOM)
					subRecord.Set("gst_percent", subGst)
					subRecord.Set("unit_price", subUnitPrice)
					subRecord.Set("budgeted_price", subUnitPrice)

					if err := app.Save(subRecord); err != nil {
						log.Printf("boq_create: could not save sub item %d.%d: %v", i+1, si+1, err)
						continue
					}

					subItemID := subRecord.Id

					// Now save pre-parsed sub-sub items
					for ssi, ss := range subSubEntries {
						ssRecord := core.NewRecord(subSubItemsCol)
						ssRecord.Set("sub_item", subItemID)
						ssRecord.Set("sort_order", ssi+1)
						ssRecord.Set("type", ss.ssType)
						ssRecord.Set("description", ss.ssDesc)
						ssRecord.Set("qty_per_unit", ss.ssQty)
						ssRecord.Set("uom", ss.ssUOM)
						ssRecord.Set("unit_price", ss.ssBudgeted)
						ssRecord.Set("budgeted_price", ss.ssBudgeted)
						ssRecord.Set("gst_percent", ss.ssGst)

						if err := app.Save(ssRecord); err != nil {
							log.Printf("boq_create: could not save sub-sub item %d.%d.%d: %v", i+1, si+1, ssi+1, err)
						}
					}

					mainItemBudgetedTotal += subRecord.GetFloat("budgeted_price")
				}
			}

			// Roll up: if sub items exist, use their sum; otherwise use manual entry
			// Then multiply by main item qty for total budgeted
			var perUnitBudgeted float64
			if mainItemBudgetedTotal > 0 {
				perUnitBudgeted = mainItemBudgetedTotal
			} else {
				perUnitBudgeted = manualBudgeted
			}
			itemRecord.Set("budgeted_price", perUnitBudgeted*qty)
			if err := app.Save(itemRecord); err != nil {
				log.Printf("boq_create: could not update main item %d budgeted: %v", i+1, err)
			}
		}

		return e.Redirect(http.StatusFound, "/boq/"+boqID)
	}
}
