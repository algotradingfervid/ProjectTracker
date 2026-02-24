package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// HandleAddressDelete handles DELETE /projects/{projectId}/addresses/{type}/{addressId}
// For Ship To addresses, it nullifies the ship_to_parent field on linked Install At addresses
// before deleting.
func HandleAddressDelete(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		addrTypeSlug := e.Request.PathValue("type")
		addressID := e.Request.PathValue("addressId")

		if projectID == "" || addrTypeSlug == "" || addressID == "" {
			return e.String(http.StatusBadRequest, "Missing required parameters")
		}

		// Convert URL slug to DB type
		addrType := AddressType(strings.ReplaceAll(addrTypeSlug, "-", "_"))

		if _, ok := AddressTypeDisplayLabels[addrType]; !ok {
			return e.String(http.StatusBadRequest, "Invalid address type")
		}

		// Find the address record
		record, err := app.FindRecordById("addresses", addressID)
		if err != nil {
			log.Printf("address_delete: not found %s: %v", addressID, err)
			return e.String(http.StatusNotFound, "Address not found")
		}

		// Verify the address belongs to this project
		if record.GetString("project") != projectID {
			return e.String(http.StatusForbidden, "Address does not belong to this project")
		}

		// If deleting a Ship To address, nullify linked Install At addresses
		if addrType == AddressTypeShipTo {
			if err := nullifyLinkedInstallAtAddresses(app, addressID); err != nil {
				log.Printf("address_delete: failed to nullify linked install_at: %v", err)
				return e.String(http.StatusInternalServerError, "Failed to update linked addresses")
			}
		}

		// Delete the address
		if err := app.Delete(record); err != nil {
			log.Printf("address_delete: failed to delete %s: %v", addressID, err)
			return e.String(http.StatusInternalServerError, "Failed to delete address")
		}

		// HTMX response: redirect to refresh the address list
		if e.Request.Header.Get("HX-Request") == "true" {
			listURL := fmt.Sprintf("/projects/%s/addresses/%s", projectID, addrTypeSlug)
			e.Response.Header().Set("HX-Redirect", listURL)
			return e.String(http.StatusOK, "")
		}

		return e.Redirect(http.StatusFound,
			fmt.Sprintf("/projects/%s/addresses/%s", projectID, addrTypeSlug))
	}
}

// HandleAddressDeleteInfo returns metadata needed for delete confirmation.
// GET /projects/{projectId}/addresses/{type}/{addressId}/delete-info
// Returns JSON: {"linked_count": N, "company_name": "..."}
func HandleAddressDeleteInfo(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		addrTypeSlug := e.Request.PathValue("type")
		addressID := e.Request.PathValue("addressId")

		addrType := AddressType(strings.ReplaceAll(addrTypeSlug, "-", "_"))

		if addrType != AddressTypeShipTo {
			return e.JSON(http.StatusOK, map[string]any{"linked_count": 0})
		}

		count, err := countLinkedInstallAtAddresses(app, addressID)
		if err != nil {
			log.Printf("address_delete_info: %v", err)
			count = 0
		}

		record, _ := app.FindRecordById("addresses", addressID)
		companyName := ""
		if record != nil {
			companyName = record.GetString("company_name")
		}

		return e.JSON(http.StatusOK, map[string]any{
			"linked_count": count,
			"company_name": companyName,
		})
	}
}

// BulkDeleteRequest is the expected JSON body for bulk delete.
type BulkDeleteRequest struct {
	IDs []string `json:"ids"`
}

// HandleAddressBulkDelete handles DELETE /projects/{projectId}/addresses/{type}/bulk
// Expects JSON body: {"ids": ["id1", "id2", ...]}
func HandleAddressBulkDelete(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		addrTypeSlug := e.Request.PathValue("type")

		if projectID == "" || addrTypeSlug == "" {
			return e.String(http.StatusBadRequest, "Missing required parameters")
		}

		addrType := AddressType(strings.ReplaceAll(addrTypeSlug, "-", "_"))

		if _, ok := AddressTypeDisplayLabels[addrType]; !ok {
			return e.String(http.StatusBadRequest, "Invalid address type")
		}

		// Parse request body
		var req BulkDeleteRequest
		if err := e.BindBody(&req); err != nil {
			return e.String(http.StatusBadRequest, "Invalid request body")
		}

		if len(req.IDs) == 0 {
			return e.String(http.StatusBadRequest, "No IDs provided")
		}

		// Delete each address
		var deleteErrors []string
		for _, id := range req.IDs {
			record, err := app.FindRecordById("addresses", id)
			if err != nil {
				deleteErrors = append(deleteErrors, fmt.Sprintf("%s: not found", id))
				continue
			}

			// Verify ownership
			if record.GetString("project") != projectID {
				deleteErrors = append(deleteErrors, fmt.Sprintf("%s: wrong project", id))
				continue
			}

			// Nullify linked Install At addresses if deleting Ship To
			if addrType == AddressTypeShipTo {
				if err := nullifyLinkedInstallAtAddresses(app, id); err != nil {
					log.Printf("bulk_delete: nullify linked for %s: %v", id, err)
				}
			}

			if err := app.Delete(record); err != nil {
				deleteErrors = append(deleteErrors, fmt.Sprintf("%s: delete failed", id))
				log.Printf("bulk_delete: failed %s: %v", id, err)
			}
		}

		if len(deleteErrors) > 0 {
			log.Printf("bulk_delete: partial errors: %v", deleteErrors)
		}

		// HTMX response: refresh address list
		if e.Request.Header.Get("HX-Request") == "true" {
			listURL := fmt.Sprintf("/projects/%s/addresses/%s", projectID, addrTypeSlug)
			e.Response.Header().Set("HX-Redirect", listURL)
			return e.String(http.StatusOK, "")
		}

		return e.JSON(http.StatusOK, map[string]any{
			"deleted": len(req.IDs) - len(deleteErrors),
			"errors":  deleteErrors,
		})
	}
}

// nullifyLinkedInstallAtAddresses finds all Install At addresses that reference
// the given Ship To address ID and sets their ship_to_parent to empty.
func nullifyLinkedInstallAtAddresses(app *pocketbase.PocketBase, shipToID string) error {
	addressesCol, err := app.FindCollectionByNameOrId("addresses")
	if err != nil {
		return fmt.Errorf("addresses collection not found: %w", err)
	}

	linked, err := app.FindRecordsByFilter(
		addressesCol,
		"address_type = 'install_at' && ship_to_parent = {:shipToId}",
		"",
		0, 0,
		map[string]any{"shipToId": shipToID},
	)
	if err != nil {
		return fmt.Errorf("query linked install_at: %w", err)
	}

	for _, rec := range linked {
		rec.Set("ship_to_parent", "")
		if err := app.Save(rec); err != nil {
			return fmt.Errorf("nullify ship_to_parent on %s: %w", rec.Id, err)
		}
	}

	return nil
}

// countLinkedInstallAtAddresses returns the number of Install At addresses
// linked to a given Ship To address.
func countLinkedInstallAtAddresses(app *pocketbase.PocketBase, shipToID string) (int, error) {
	addressesCol, err := app.FindCollectionByNameOrId("addresses")
	if err != nil {
		return 0, err
	}

	linked, err := app.FindRecordsByFilter(
		addressesCol,
		"address_type = 'install_at' && ship_to_parent = {:shipToId}",
		"",
		0, 0,
		map[string]any{"shipToId": shipToID},
	)
	if err != nil {
		return 0, err
	}

	return len(linked), nil
}
