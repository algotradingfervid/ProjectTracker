package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/templates"
)

// fetchBOQItemsForProject returns all sub_items and sub_sub_items for the project's BOQs,
// grouped by main item description for display in the template item picker.
func fetchBOQItemsForProject(app *pocketbase.PocketBase, projectId string) []templates.BOQItemGroup {
	boqs, err := app.FindRecordsByFilter("boqs", "project = {:pid}", "title", 0, 0, map[string]any{"pid": projectId})
	if err != nil {
		log.Printf("dc_template_create: could not query boqs: %v", err)
		return nil
	}

	var groups []templates.BOQItemGroup
	for _, boq := range boqs {
		mainItems, err := app.FindRecordsByFilter("main_boq_items", "boq = {:bid}", "sort_order", 0, 0, map[string]any{"bid": boq.Id})
		if err != nil {
			continue
		}

		for _, mi := range mainItems {
			group := templates.BOQItemGroup{
				MainItemID:          mi.Id,
				MainItemDescription: mi.GetString("description"),
				BOQTitle:            boq.GetString("title"),
			}

			// Fetch sub_items
			subItems, _ := app.FindRecordsByFilter("sub_items", "main_item = {:mid}", "sort_order", 0, 0, map[string]any{"mid": mi.Id})
			for _, si := range subItems {
				group.Items = append(group.Items, templates.BOQPickerItem{
					ID:          si.Id,
					Type:        "sub_item",
					Description: si.GetString("description"),
					UOM:         si.GetString("uom"),
					HSNCode:     si.GetString("hsn_code"),
				})

				// Fetch sub_sub_items
				subSubItems, _ := app.FindRecordsByFilter("sub_sub_items", "sub_item = {:sid}", "sort_order", 0, 0, map[string]any{"sid": si.Id})
				for _, ssi := range subSubItems {
					group.Items = append(group.Items, templates.BOQPickerItem{
						ID:          ssi.Id,
						Type:        "sub_sub_item",
						Description: fmt.Sprintf("  └ %s", ssi.GetString("description")),
						UOM:         ssi.GetString("uom"),
						HSNCode:     ssi.GetString("hsn_code"),
					})
				}
			}

			if len(group.Items) > 0 {
				groups = append(groups, group)
			}
		}
	}
	return groups
}

func HandleDCTemplateCreate(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")

		boqItems := fetchBOQItemsForProject(app, projectId)

		data := templates.DCTemplateFormData{
			ProjectID: projectId,
			BOQItems:  boqItems,
			Errors:    make(map[string]string),
			IsEdit:    false,
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.DCTemplateFormContent(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.DCTemplateFormPage(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

func HandleDCTemplateSave(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		projectId := e.Request.PathValue("projectId")
		name := strings.TrimSpace(e.Request.FormValue("name"))
		purpose := strings.TrimSpace(e.Request.FormValue("purpose"))

		errors := make(map[string]string)
		if name == "" {
			errors["name"] = "Template name is required"
		}

		if len(errors) > 0 {
			SetToast(e, "warning", "Please fix the errors below")
			boqItems := fetchBOQItemsForProject(app, projectId)
			data := templates.DCTemplateFormData{
				ProjectID: projectId,
				Name:      name,
				Purpose:   purpose,
				BOQItems:  boqItems,
				Errors:    errors,
				IsEdit:    false,
			}
			var component templ.Component
			if e.Request.Header.Get("HX-Request") == "true" {
				component = templates.DCTemplateFormContent(data)
			} else {
				headerData := GetHeaderData(e.Request)
				sidebarData := GetSidebarData(e.Request)
				component = templates.DCTemplateFormPage(data, headerData, sidebarData)
			}
			return component.Render(e.Request.Context(), e.Response)
		}

		// Save template
		col, err := app.FindCollectionByNameOrId("dc_templates")
		if err != nil {
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		record := core.NewRecord(col)
		record.Set("project", projectId)
		record.Set("name", name)
		record.Set("purpose", purpose)

		if err := app.Save(record); err != nil {
			log.Printf("dc_template_create: could not save template: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		// Save template items
		saveDCTemplateItems(app, record.Id, e.Request)

		redirectURL := fmt.Sprintf("/projects/%s/dc-templates/", projectId)
		SetToast(e, "success", "DC template created")

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", redirectURL)
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, redirectURL)
	}
}

// saveDCTemplateItems parses form data for template items and saves them.
// Form fields: item_ids[] (comma-separated "type:id"), item_qty_{type}_{id}, item_serial_{type}_{id}
func saveDCTemplateItems(app *pocketbase.PocketBase, templateID string, r *http.Request) {
	itemIDs := r.Form["item_ids"]

	col, err := app.FindCollectionByNameOrId("dc_template_items")
	if err != nil {
		log.Printf("dc_template_create: could not find dc_template_items collection: %v", err)
		return
	}

	for _, itemKey := range itemIDs {
		parts := strings.SplitN(itemKey, ":", 2)
		if len(parts) != 2 {
			continue
		}
		itemType := parts[0]
		itemID := parts[1]

		qtyStr := r.FormValue(fmt.Sprintf("item_qty_%s_%s", itemType, itemID))
		qty, _ := strconv.ParseFloat(qtyStr, 64)

		serial := r.FormValue(fmt.Sprintf("item_serial_%s_%s", itemType, itemID))
		if serial == "" {
			serial = "none"
		}

		rec := core.NewRecord(col)
		rec.Set("template", templateID)
		rec.Set("source_item_type", itemType)
		rec.Set("source_item_id", itemID)
		rec.Set("default_quantity", qty)
		rec.Set("serial_tracking", serial)

		if err := app.Save(rec); err != nil {
			log.Printf("dc_template_create: could not save template item: %v", err)
		}
	}
}
