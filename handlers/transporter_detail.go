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

func HandleTransporterDetail(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		id := e.Request.PathValue("id")

		rec, err := app.FindRecordById("transporters", id)
		if err != nil || rec.GetString("project") != projectId {
			return ErrorToast(e, http.StatusNotFound, "Transporter not found")
		}

		vehicles := fetchVehicles(app, id)

		data := templates.TransporterDetailData{
			ProjectID:     projectId,
			TransporterID: id,
			CompanyName:   rec.GetString("company_name"),
			ContactPerson: rec.GetString("contact_person"),
			Phone:         rec.GetString("phone"),
			GSTNumber:     rec.GetString("gst_number"),
			IsActive:      rec.GetBool("is_active"),
			Vehicles:      vehicles,
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.TransporterDetailContent(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.TransporterDetailPage(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

func HandleTransporterUpdate(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		projectId := e.Request.PathValue("projectId")
		id := e.Request.PathValue("id")

		rec, err := app.FindRecordById("transporters", id)
		if err != nil || rec.GetString("project") != projectId {
			return ErrorToast(e, http.StatusNotFound, "Transporter not found")
		}

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
			vehicles := fetchVehicles(app, id)
			data := templates.TransporterFormData{
				ProjectID:     projectId,
				TransporterID: id,
				CompanyName:   companyName,
				ContactPerson: contactPerson,
				Phone:         phone,
				GSTNumber:     gstNumber,
				IsActive:      isActive,
				Vehicles:      vehicles,
				Errors:        errors,
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

		rec.Set("company_name", companyName)
		rec.Set("contact_person", contactPerson)
		rec.Set("phone", phone)
		rec.Set("gst_number", gstNumber)
		rec.Set("is_active", isActive)

		if err := app.Save(rec); err != nil {
			log.Printf("transporter_detail: could not update transporter: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		redirectURL := fmt.Sprintf("/projects/%s/transporters/%s", projectId, id)
		SetToast(e, "success", "Transporter updated")

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", redirectURL)
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, redirectURL)
	}
}

func HandleTransporterToggle(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		id := e.Request.PathValue("id")

		rec, err := app.FindRecordById("transporters", id)
		if err != nil || rec.GetString("project") != projectId {
			return ErrorToast(e, http.StatusNotFound, "Transporter not found")
		}

		newActive := !rec.GetBool("is_active")
		rec.Set("is_active", newActive)

		if err := app.Save(rec); err != nil {
			log.Printf("transporter_toggle: could not toggle transporter: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		statusLabel := "deactivated"
		if newActive {
			statusLabel = "activated"
		}

		redirectURL := fmt.Sprintf("/projects/%s/transporters/%s", projectId, id)
		SetToast(e, "success", fmt.Sprintf("Transporter %s", statusLabel))

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", redirectURL)
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, redirectURL)
	}
}

func fetchVehicles(app *pocketbase.PocketBase, transporterID string) []templates.VehicleItem {
	records, err := app.FindRecordsByFilter(
		"transporter_vehicles",
		"transporter = {:tid}",
		"-created", 0, 0,
		map[string]any{"tid": transporterID},
	)
	if err != nil {
		return nil
	}

	var items []templates.VehicleItem
	for _, rec := range records {
		items = append(items, templates.VehicleItem{
			ID:            rec.Id,
			VehicleNumber: rec.GetString("vehicle_number"),
			VehicleType:   rec.GetString("vehicle_type"),
			DriverName:    rec.GetString("driver_name"),
			DriverPhone:   rec.GetString("driver_phone"),
		})
	}
	return items
}
