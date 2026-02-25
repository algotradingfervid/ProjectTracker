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

func HandleVendorCreate(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")

		data := templates.VendorFormData{
			ProjectID: projectID,
			Errors:    make(map[string]string),
		}

		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.VendorFormContent(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.VendorFormPage(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

func HandleVendorSave(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		projectID := e.Request.PathValue("projectId")

		data := templates.VendorFormData{
			ProjectID:           projectID,
			Name:                strings.TrimSpace(e.Request.FormValue("name")),
			ContactName:         strings.TrimSpace(e.Request.FormValue("contact_name")),
			Phone:               strings.TrimSpace(e.Request.FormValue("phone")),
			Email:               strings.TrimSpace(e.Request.FormValue("email")),
			AddressLine1:        strings.TrimSpace(e.Request.FormValue("address_line_1")),
			AddressLine2:        strings.TrimSpace(e.Request.FormValue("address_line_2")),
			City:                strings.TrimSpace(e.Request.FormValue("city")),
			State:               strings.TrimSpace(e.Request.FormValue("state")),
			PinCode:             strings.TrimSpace(e.Request.FormValue("pin_code")),
			Country:             strings.TrimSpace(e.Request.FormValue("country")),
			GSTIN:               strings.TrimSpace(e.Request.FormValue("gstin")),
			PAN:                 strings.TrimSpace(e.Request.FormValue("pan")),
			Website:             strings.TrimSpace(e.Request.FormValue("website")),
			BankBeneficiaryName: strings.TrimSpace(e.Request.FormValue("bank_beneficiary_name")),
			BankName:            strings.TrimSpace(e.Request.FormValue("bank_name")),
			BankAccountNo:       strings.TrimSpace(e.Request.FormValue("bank_account_no")),
			BankIFSC:            strings.TrimSpace(e.Request.FormValue("bank_ifsc")),
			BankBranch:          strings.TrimSpace(e.Request.FormValue("bank_branch")),
			Notes:               strings.TrimSpace(e.Request.FormValue("notes")),
			Errors:              make(map[string]string),
		}

		// Validation
		if data.Name == "" {
			data.Errors["name"] = "Name is required"
		}

		if len(data.Errors) > 0 {
			SetToast(e, "warning", "Please fix the errors below")
			var component templ.Component
			if e.Request.Header.Get("HX-Request") == "true" {
				component = templates.VendorFormContent(data)
			} else {
				headerData := GetHeaderData(e.Request)
				sidebarData := GetSidebarData(e.Request)
				component = templates.VendorFormPage(data, headerData, sidebarData)
			}
			return component.Render(e.Request.Context(), e.Response)
		}

		// Save vendor
		vendorsCol, err := app.FindCollectionByNameOrId("vendors")
		if err != nil {
			log.Printf("vendor_create: could not find vendors collection: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		record := core.NewRecord(vendorsCol)
		setVendorFields(record, data)

		if err := app.Save(record); err != nil {
			log.Printf("vendor_create: could not save vendor: %v", err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		// If project context, auto-link vendor to project
		if projectID != "" {
			pvCol, err := app.FindCollectionByNameOrId("project_vendors")
			if err == nil {
				link := core.NewRecord(pvCol)
				link.Set("project", projectID)
				link.Set("vendor", record.Id)
				if err := app.Save(link); err != nil {
					log.Printf("vendor_create: could not link vendor to project: %v", err)
				}
			}
		}

		redirectURL := "/vendors"
		if projectID != "" {
			redirectURL = fmt.Sprintf("/projects/%s/vendors", projectID)
		}

		SetToast(e, "success", "Vendor created successfully")

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", redirectURL)
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, redirectURL)
	}
}

// setVendorFields sets all vendor fields on a record from form data.
func setVendorFields(record *core.Record, data templates.VendorFormData) {
	record.Set("name", data.Name)
	record.Set("contact_name", data.ContactName)
	record.Set("phone", data.Phone)
	record.Set("email", data.Email)
	record.Set("address_line_1", data.AddressLine1)
	record.Set("address_line_2", data.AddressLine2)
	record.Set("city", data.City)
	record.Set("state", data.State)
	record.Set("pin_code", data.PinCode)
	record.Set("country", data.Country)
	record.Set("gstin", data.GSTIN)
	record.Set("pan", data.PAN)
	record.Set("website", data.Website)
	record.Set("bank_beneficiary_name", data.BankBeneficiaryName)
	record.Set("bank_name", data.BankName)
	record.Set("bank_account_no", data.BankAccountNo)
	record.Set("bank_ifsc", data.BankIFSC)
	record.Set("bank_branch", data.BankBranch)
	record.Set("notes", data.Notes)
}
