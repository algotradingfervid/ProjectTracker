package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/collections"
	"projectcreation/templates"
)

var ProjectStatusOptions = []string{"active", "completed", "on_hold"}

func HandleProjectCreate(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		data := templates.ProjectCreateData{
			Status:                "active",
			ShipToEqualsInstallAt: true,
			StatusOptions:         ProjectStatusOptions,
			Errors:                make(map[string]string),
		}
		headerData := GetHeaderData(e.Request)
		sidebarData := GetSidebarData(e.Request)
		component := templates.ProjectCreatePage(data, headerData, sidebarData)
		return component.Render(e.Request.Context(), e.Response)
	}
}

func HandleProjectSave(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		name := strings.TrimSpace(e.Request.FormValue("name"))
		clientName := strings.TrimSpace(e.Request.FormValue("client_name"))
		refNumber := strings.TrimSpace(e.Request.FormValue("reference_number"))
		status := strings.TrimSpace(e.Request.FormValue("status"))
		shipToEqualsInstallAt := e.Request.FormValue("ship_to_equals_install_at") == "on" ||
			e.Request.FormValue("ship_to_equals_install_at") == "true"

		errors := make(map[string]string)
		if name == "" {
			errors["name"] = "Project name is required"
		}

		validStatus := false
		for _, s := range ProjectStatusOptions {
			if status == s {
				validStatus = true
				break
			}
		}
		if !validStatus {
			status = "active"
		}

		if name != "" {
			existing, _ := app.FindRecordsByFilter(
				"projects",
				"name = {:name}",
				"", 1, 0,
				map[string]any{"name": name},
			)
			if len(existing) > 0 {
				errors["name"] = "A project with this name already exists"
			}
		}

		if len(errors) > 0 {
			SetToast(e, "warning", "Please fix the errors below")
			data := templates.ProjectCreateData{
				Name:                  name,
				ClientName:            clientName,
				ReferenceNumber:       refNumber,
				Status:                status,
				ShipToEqualsInstallAt: shipToEqualsInstallAt,
				StatusOptions:         ProjectStatusOptions,
				Errors:                errors,
			}
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component := templates.ProjectCreatePage(data, headerData, sidebarData)
			return component.Render(e.Request.Context(), e.Response)
		}

		projectsCol, err := app.FindCollectionByNameOrId("projects")
		if err != nil {
			log.Printf("project_create: could not find projects collection: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		record := core.NewRecord(projectsCol)
		record.Set("name", name)
		record.Set("client_name", clientName)
		record.Set("reference_number", refNumber)
		record.Set("status", status)
		record.Set("ship_to_equals_install_at", shipToEqualsInstallAt)

		if err := app.Save(record); err != nil {
			log.Printf("project_create: could not save project: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		if err := collections.MigrateDefaultAddressSettings(app); err != nil {
			log.Printf("project_create: failed to create default address settings: %v", err)
		}

		SetToast(e, "success", "Project created successfully")

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", "/projects")
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, "/projects")
	}
}
