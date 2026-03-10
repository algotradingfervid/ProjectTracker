package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func HandleDCTemplateDelete(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		id := e.Request.PathValue("id")

		rec, err := app.FindRecordById("dc_templates", id)
		if err != nil || rec.GetString("project") != projectId {
			return ErrorToast(e, http.StatusNotFound, "DC template not found")
		}

		if err := app.Delete(rec); err != nil {
			log.Printf("dc_template_delete: could not delete template %s: %v", id, err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		redirectURL := fmt.Sprintf("/projects/%s/dc-templates/", projectId)
		SetToast(e, "success", "DC template deleted")

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", redirectURL)
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, redirectURL)
	}
}

func HandleDCTemplateDuplicate(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		id := e.Request.PathValue("id")

		src, err := app.FindRecordById("dc_templates", id)
		if err != nil || src.GetString("project") != projectId {
			return ErrorToast(e, http.StatusNotFound, "DC template not found")
		}

		// Duplicate the template
		col, err := app.FindCollectionByNameOrId("dc_templates")
		if err != nil {
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		newRec := core.NewRecord(col)
		newRec.Set("project", projectId)
		newRec.Set("name", src.GetString("name")+" (Copy)")
		newRec.Set("purpose", src.GetString("purpose"))

		if err := app.Save(newRec); err != nil {
			log.Printf("dc_template_duplicate: could not save duplicate: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		// Duplicate template items
		srcItems, _ := app.FindRecordsByFilter(
			"dc_template_items",
			"template = {:tid}",
			"", 0, 0,
			map[string]any{"tid": id},
		)

		itemCol, _ := app.FindCollectionByNameOrId("dc_template_items")
		if itemCol != nil {
			for _, si := range srcItems {
				newItem := core.NewRecord(itemCol)
				newItem.Set("template", newRec.Id)
				newItem.Set("source_item_type", si.GetString("source_item_type"))
				newItem.Set("source_item_id", si.GetString("source_item_id"))
				newItem.Set("default_quantity", si.GetFloat("default_quantity"))
				newItem.Set("serial_tracking", si.GetString("serial_tracking"))
				_ = app.Save(newItem)
			}
		}

		redirectURL := fmt.Sprintf("/projects/%s/dc-templates/", projectId)
		SetToast(e, "success", "DC template duplicated")

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", redirectURL)
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, redirectURL)
	}
}
