package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/templates"
)

func HandleDCTemplateEdit(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		id := e.Request.PathValue("id")

		rec, err := app.FindRecordById("dc_templates", id)
		if err != nil || rec.GetString("project") != projectId {
			return ErrorToast(e, http.StatusNotFound, "DC template not found")
		}

		// Load existing template items
		existingItems, _ := app.FindRecordsByFilter(
			"dc_template_items",
			"template = {:tid}",
			"", 0, 0,
			map[string]any{"tid": id},
		)

		selectedItems := make(map[string]templates.SelectedTemplateItem)
		for _, item := range existingItems {
			key := fmt.Sprintf("%s:%s", item.GetString("source_item_type"), item.GetString("source_item_id"))
			selectedItems[key] = templates.SelectedTemplateItem{
				DefaultQuantity: item.GetFloat("default_quantity"),
				SerialTracking:  item.GetString("serial_tracking"),
			}
		}

		boqItems := fetchBOQItemsForProject(app, projectId)

		data := templates.DCTemplateFormData{
			ProjectID:     projectId,
			TemplateID:    id,
			Name:          rec.GetString("name"),
			Purpose:       rec.GetString("purpose"),
			BOQItems:      boqItems,
			SelectedItems: selectedItems,
			Errors:        make(map[string]string),
			IsEdit:        true,
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

func HandleDCTemplateUpdate(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		projectId := e.Request.PathValue("projectId")
		id := e.Request.PathValue("id")

		rec, err := app.FindRecordById("dc_templates", id)
		if err != nil || rec.GetString("project") != projectId {
			return ErrorToast(e, http.StatusNotFound, "DC template not found")
		}

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
				ProjectID:  projectId,
				TemplateID: id,
				Name:       name,
				Purpose:    purpose,
				BOQItems:   boqItems,
				Errors:     errors,
				IsEdit:     true,
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

		rec.Set("name", name)
		rec.Set("purpose", purpose)
		if err := app.Save(rec); err != nil {
			log.Printf("dc_template_edit: could not update template: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		// Delete existing items and re-save
		existingItems, _ := app.FindRecordsByFilter(
			"dc_template_items",
			"template = {:tid}",
			"", 0, 0,
			map[string]any{"tid": id},
		)
		for _, item := range existingItems {
			_ = app.Delete(item)
		}
		saveDCTemplateItems(app, id, e.Request)

		redirectURL := fmt.Sprintf("/projects/%s/dc-templates/", projectId)
		SetToast(e, "success", "DC template updated")

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", redirectURL)
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, redirectURL)
	}
}
