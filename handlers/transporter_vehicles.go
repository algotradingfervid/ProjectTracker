package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func HandleVehicleAdd(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		transporterID := e.Request.PathValue("id")

		// Verify transporter belongs to project
		transporter, err := app.FindRecordById("transporters", transporterID)
		if err != nil || transporter.GetString("project") != projectId {
			return ErrorToast(e, http.StatusNotFound, "Transporter not found")
		}

		if err := e.Request.ParseMultipartForm(5 << 20); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		vehicleNumber := strings.TrimSpace(e.Request.FormValue("vehicle_number"))
		vehicleType := strings.TrimSpace(e.Request.FormValue("vehicle_type"))
		driverName := strings.TrimSpace(e.Request.FormValue("driver_name"))
		driverPhone := strings.TrimSpace(e.Request.FormValue("driver_phone"))

		if vehicleNumber == "" {
			return ErrorToast(e, http.StatusBadRequest, "Vehicle number is required")
		}

		col, err := app.FindCollectionByNameOrId("transporter_vehicles")
		if err != nil {
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		record := core.NewRecord(col)
		record.Set("transporter", transporterID)
		record.Set("vehicle_number", vehicleNumber)
		record.Set("vehicle_type", vehicleType)
		record.Set("driver_name", driverName)
		record.Set("driver_phone", driverPhone)

		// Handle file uploads via multipart form headers
		if e.Request.MultipartForm != nil {
			if files := e.Request.MultipartForm.File["rc_image"]; len(files) > 0 {
				record.Set("rc_image", files[0])
			}
			if files := e.Request.MultipartForm.File["driver_license"]; len(files) > 0 {
				record.Set("driver_license", files[0])
			}
		}

		if err := app.Save(record); err != nil {
			log.Printf("vehicle_add: could not save vehicle: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		redirectURL := fmt.Sprintf("/projects/%s/transporters/%s", projectId, transporterID)
		SetToast(e, "success", "Vehicle added")

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", redirectURL)
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, redirectURL)
	}
}

func HandleVehicleDelete(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectId := e.Request.PathValue("projectId")
		transporterID := e.Request.PathValue("id")
		vehicleID := e.Request.PathValue("vid")

		// Verify transporter belongs to project
		transporter, err := app.FindRecordById("transporters", transporterID)
		if err != nil || transporter.GetString("project") != projectId {
			return ErrorToast(e, http.StatusNotFound, "Transporter not found")
		}

		vehicle, err := app.FindRecordById("transporter_vehicles", vehicleID)
		if err != nil || vehicle.GetString("transporter") != transporterID {
			return ErrorToast(e, http.StatusNotFound, "Vehicle not found")
		}

		if err := app.Delete(vehicle); err != nil {
			log.Printf("vehicle_delete: could not delete vehicle %s: %v", vehicleID, err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		redirectURL := fmt.Sprintf("/projects/%s/transporters/%s", projectId, transporterID)
		SetToast(e, "success", "Vehicle deleted")

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", redirectURL)
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, redirectURL)
	}
}
