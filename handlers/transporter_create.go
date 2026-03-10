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

func HandleTransporterCreate(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")

		data := templates.TransporterFormData{
			ProjectID: projectId,
			IsActive:  true,
			Errors:    make(map[string]string),
			IsEdit:    false,
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

func HandleTransporterSave(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		projectId := e.Request.PathValue("projectId")
		companyName := strings.TrimSpace(e.Request.FormValue("company_name"))
		contactPerson := strings.TrimSpace(e.Request.FormValue("contact_person"))
		phone := strings.TrimSpace(e.Request.FormValue("phone"))
		gstNumber := strings.TrimSpace(e.Request.FormValue("gst_number"))
		isActive := e.Request.FormValue("is_active") == "on"

		errors := make(map[string]string)
		if companyName == "" {
			errors["company_name"] = "Company name is required"
		}

		if len(errors) > 0 {
			SetToast(e, "warning", "Please fix the errors below")
			data := templates.TransporterFormData{
				ProjectID:     projectId,
				CompanyName:   companyName,
				ContactPerson: contactPerson,
				Phone:         phone,
				GSTNumber:     gstNumber,
				IsActive:      isActive,
				Errors:        errors,
				IsEdit:        false,
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

		col, err := app.FindCollectionByNameOrId("transporters")
		if err != nil {
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		record := core.NewRecord(col)
		record.Set("project", projectId)
		record.Set("company_name", companyName)
		record.Set("contact_person", contactPerson)
		record.Set("phone", phone)
		record.Set("gst_number", gstNumber)
		record.Set("is_active", isActive)

		if err := app.Save(record); err != nil {
			log.Printf("transporter_create: could not save transporter: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		redirectURL := fmt.Sprintf("/projects/%s/transporters/%s", projectId, record.Id)
		SetToast(e, "success", "Transporter created")

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", redirectURL)
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, redirectURL)
	}
}
