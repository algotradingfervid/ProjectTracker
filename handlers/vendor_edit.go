package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/templates"
)

func HandleVendorEdit(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		vendorID := e.Request.PathValue("id")
		if vendorID == "" {
			return ErrorToast(e, http.StatusBadRequest, "Missing vendor ID")
		}

		record, err := app.FindRecordById("vendors", vendorID)
		if err != nil {
			log.Printf("vendor_edit: could not find vendor %s: %v", vendorID, err)
			return ErrorToast(e, http.StatusNotFound, "Vendor not found")
		}

		data := templates.VendorFormData{
			ID:                  vendorID,
			Name:                record.GetString("name"),
			ContactName:         record.GetString("contact_name"),
			Phone:               record.GetString("phone"),
			Email:               record.GetString("email"),
			AddressLine1:        record.GetString("address_line_1"),
			AddressLine2:        record.GetString("address_line_2"),
			City:                record.GetString("city"),
			State:               record.GetString("state"),
			PinCode:             record.GetString("pin_code"),
			Country:             record.GetString("country"),
			GSTIN:               record.GetString("gstin"),
			PAN:                 record.GetString("pan"),
			Website:             record.GetString("website"),
			BankBeneficiaryName: record.GetString("bank_beneficiary_name"),
			BankName:            record.GetString("bank_name"),
			BankAccountNo:       record.GetString("bank_account_no"),
			BankIFSC:            record.GetString("bank_ifsc"),
			BankBranch:          record.GetString("bank_branch"),
			Notes:               record.GetString("notes"),
			IsEdit:              true,
			Errors:              make(map[string]string),
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

func HandleVendorUpdate(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		vendorID := e.Request.PathValue("id")
		if vendorID == "" {
			return ErrorToast(e, http.StatusBadRequest, "Missing vendor ID")
		}

		if err := e.Request.ParseForm(); err != nil {
			return ErrorToast(e, http.StatusBadRequest, "Invalid form data")
		}

		record, err := app.FindRecordById("vendors", vendorID)
		if err != nil {
			log.Printf("vendor_update: could not find vendor %s: %v", vendorID, err)
			return ErrorToast(e, http.StatusNotFound, "Vendor not found")
		}

		data := templates.VendorFormData{
			ID:                  vendorID,
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
			IsEdit:              true,
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

		setVendorFields(record, data)

		if err := app.Save(record); err != nil {
			log.Printf("vendor_update: could not save vendor %s: %v", vendorID, err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		SetToast(e, "success", "Vendor updated successfully")

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", "/vendors")
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, "/vendors")
	}
}
