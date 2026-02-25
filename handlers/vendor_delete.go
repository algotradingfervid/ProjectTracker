package handlers

import (
	"log"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func HandleVendorDelete(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		vendorID := e.Request.PathValue("id")
		if vendorID == "" {
			return ErrorToast(e, http.StatusBadRequest, "Missing vendor ID")
		}

		record, err := app.FindRecordById("vendors", vendorID)
		if err != nil {
			log.Printf("vendor_delete: could not find vendor %s: %v", vendorID, err)
			return ErrorToast(e, http.StatusNotFound, "Vendor not found")
		}

		// Check if vendor has linked purchase orders
		pos, err := app.FindRecordsByFilter(
			"purchase_orders",
			"vendor = {:vendorId}",
			"", 1, 0,
			map[string]any{"vendorId": vendorID},
		)
		if err == nil && len(pos) > 0 {
			return ErrorToast(e, http.StatusConflict, "Cannot delete vendor — it has existing purchase orders")
		}

		if err := app.Delete(record); err != nil {
			log.Printf("vendor_delete: failed to delete vendor %s: %v", vendorID, err)
			return ErrorToast(e, http.StatusInternalServerError, "Something went wrong. Please try again.")
		}

		log.Printf("vendor_delete: deleted vendor %s\n", vendorID)

		SetToast(e, "success", "Vendor deleted successfully")

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", "/vendors")
			return e.String(http.StatusOK, "")
		}
		return e.Redirect(http.StatusFound, "/vendors")
	}
}
