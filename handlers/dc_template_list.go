package handlers

import (
	"log"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/templates"
)

func HandleDCTemplateList(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")

		records, err := app.FindRecordsByFilter(
			"dc_templates",
			"project = {:projectId}",
			"-created",
			0, 0,
			map[string]any{"projectId": projectId},
		)
		if err != nil {
			log.Printf("dc_template_list: could not query dc_templates: %v", err)
			records = nil
		}

		var items []templates.DCTemplateListItem
		for _, rec := range records {
			// Count template items
			templateItems, _ := app.FindRecordsByFilter(
				"dc_template_items",
				"template = {:tid}",
				"", 0, 0,
				map[string]any{"tid": rec.Id},
			)

			items = append(items, templates.DCTemplateListItem{
				ID:        rec.Id,
				Name:      rec.GetString("name"),
				Purpose:   rec.GetString("purpose"),
				ItemCount: len(templateItems),
				Created:   rec.GetString("created"),
			})
		}

		data := templates.DCTemplateListData{
			ProjectID: projectId,
			Templates: items,
			Total:     len(items),
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.DCTemplateListContent(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.DCTemplateListPage(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}
