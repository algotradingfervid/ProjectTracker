package handlers

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/templates"
)

func HandleTransporterEdit(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		id := e.Request.PathValue("id")

		rec, err := app.FindRecordById("transporters", id)
		if err != nil || rec.GetString("project") != projectId {
			return ErrorToast(e, http.StatusNotFound, "Transporter not found")
		}

		vehicles := fetchVehicles(app, id)

		data := templates.TransporterFormData{
			ProjectID:     projectId,
			TransporterID: id,
			CompanyName:   rec.GetString("company_name"),
			ContactPerson: rec.GetString("contact_person"),
			Phone:         rec.GetString("phone"),
			GSTNumber:     rec.GetString("gst_number"),
			IsActive:      rec.GetBool("is_active"),
			Vehicles:      vehicles,
			Errors:        make(map[string]string),
			IsEdit:        true,
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.TransporterFormContent(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.TransporterFormPage(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}
