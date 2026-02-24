# Phase 14: Address Delete Operations

## Overview & Objectives

Implement single and bulk delete operations for addresses within a project. Deleting addresses requires careful handling of relationships: when a Ship To address is deleted, any linked Install At addresses should have their `ship_to_parent` field nullified (not cascade-deleted). The UI uses confirmation dialogs matching the existing BOQ delete patterns (HTMX `hx-confirm` for simple cases, Alpine.js modals for cases requiring extra context like linked address counts).

---

## Files to Create/Modify

| Action | Path |
|--------|------|
| **Create** | `handlers/address_delete.go` |
| **Modify** | `main.go` (register delete routes) |
| **Modify** | Address list template (e.g., `templates/address_list.templ`) to add delete buttons and bulk selection UI |

---

## Detailed Implementation Steps

### Step 1: Single Address Delete Handler

Create `handlers/address_delete.go`:

```go
package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// HandleAddressDelete handles DELETE /projects/{projectId}/addresses/{type}/{addressId}
// For Ship To addresses, it nullifies the ship_to_parent field on linked Install At addresses
// before deleting.
func HandleAddressDelete(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		addrType := e.Request.PathValue("type")
		addressID := e.Request.PathValue("addressId")

		if projectID == "" || addrType == "" || addressID == "" {
			return e.String(http.StatusBadRequest, "Missing required parameters")
		}

		// Validate address type
		if _, ok := addressTypeLabels[addrType]; !ok {
			return e.String(http.StatusBadRequest, "Invalid address type")
		}

		collectionName := addrType + "_addresses"

		// Find the address record
		record, err := app.FindRecordById(collectionName, addressID)
		if err != nil {
			log.Printf("address_delete: not found %s/%s: %v", collectionName, addressID, err)
			return e.String(http.StatusNotFound, "Address not found")
		}

		// Verify the address belongs to this project
		if record.GetString("project") != projectID {
			return e.String(http.StatusForbidden, "Address does not belong to this project")
		}

		// If deleting a Ship To address, nullify linked Install At addresses
		if addrType == "ship_to" {
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
			listURL := fmt.Sprintf("/projects/%s/addresses/%s", projectID, addrType)
			e.Response.Header().Set("HX-Redirect", listURL)
			return e.String(http.StatusOK, "")
		}

		return e.Redirect(http.StatusFound,
			fmt.Sprintf("/projects/%s/addresses/%s", projectID, addrType))
	}
}
```

### Step 2: Nullify Linked Install At Addresses

```go
// nullifyLinkedInstallAtAddresses finds all Install At addresses that reference
// the given Ship To address ID and sets their ship_to_parent to empty.
func nullifyLinkedInstallAtAddresses(app *pocketbase.PocketBase, shipToID string) error {
	installAtCol, err := app.FindCollectionByNameOrId("install_at_addresses")
	if err != nil {
		return fmt.Errorf("install_at_addresses collection not found: %w", err)
	}

	linked, err := app.FindRecordsByFilter(
		installAtCol,
		"ship_to_parent = {:shipToId}",
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
// linked to a given Ship To address. Used for the confirmation warning.
func countLinkedInstallAtAddresses(app *pocketbase.PocketBase, shipToID string) (int, error) {
	installAtCol, err := app.FindCollectionByNameOrId("install_at_addresses")
	if err != nil {
		return 0, err
	}

	linked, err := app.FindRecordsByFilter(
		installAtCol,
		"ship_to_parent = {:shipToId}",
		"",
		0, 0,
		map[string]any{"shipToId": shipToID},
	)
	if err != nil {
		return 0, err
	}

	return len(linked), nil
}
```

### Step 3: Ship To Delete Confirmation Info Endpoint

This optional endpoint returns the count of linked Install At addresses, used by the frontend to build a meaningful confirmation message before the user commits to deletion.

```go
// HandleAddressDeleteInfo returns metadata needed for delete confirmation.
// GET /projects/{projectId}/addresses/{type}/{addressId}/delete-info
// Returns JSON: {"linked_count": N, "company_name": "..."}
func HandleAddressDeleteInfo(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		addrType := e.Request.PathValue("type")
		addressID := e.Request.PathValue("addressId")

		if addrType != "ship_to" {
			return e.JSON(http.StatusOK, map[string]any{"linked_count": 0})
		}

		count, err := countLinkedInstallAtAddresses(app, addressID)
		if err != nil {
			log.Printf("address_delete_info: %v", err)
			count = 0
		}

		record, _ := app.FindRecordById("ship_to_addresses", addressID)
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
```

### Step 4: Bulk Delete Handler

```go
// BulkDeleteRequest is the expected JSON body for bulk delete.
type BulkDeleteRequest struct {
	IDs []string `json:"ids"`
}

// HandleAddressBulkDelete handles DELETE /projects/{projectId}/addresses/{type}/bulk
// Expects JSON body: {"ids": ["id1", "id2", ...]}
func HandleAddressBulkDelete(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		addrType := e.Request.PathValue("type")

		if projectID == "" || addrType == "" {
			return e.String(http.StatusBadRequest, "Missing required parameters")
		}

		if _, ok := addressTypeLabels[addrType]; !ok {
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

		collectionName := addrType + "_addresses"

		// Delete each address
		var deleteErrors []string
		for _, id := range req.IDs {
			record, err := app.FindRecordById(collectionName, id)
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
			if addrType == "ship_to" {
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
			listURL := fmt.Sprintf("/projects/%s/addresses/%s", projectID, addrType)
			e.Response.Header().Set("HX-Redirect", listURL)
			return e.String(http.StatusOK, "")
		}

		return e.JSON(http.StatusOK, map[string]any{
			"deleted": len(req.IDs) - len(deleteErrors),
			"errors":  deleteErrors,
		})
	}
}
```

### Step 5: Register Routes in `main.go`

Add these routes in the `app.OnServe().BindFunc` block:

```go
// Address delete operations
se.Router.GET("/projects/{projectId}/addresses/{type}/{addressId}/delete-info",
    handlers.HandleAddressDeleteInfo(app))
se.Router.DELETE("/projects/{projectId}/addresses/{type}/{addressId}",
    handlers.HandleAddressDelete(app))
se.Router.DELETE("/projects/{projectId}/addresses/{type}/bulk",
    handlers.HandleAddressBulkDelete(app))
```

**Important**: The `/bulk` route must be registered before the `/{addressId}` route, or use path specificity to avoid `bulk` being matched as an `addressId`. PocketBase's router handles this correctly when the literal path `/bulk` is registered first.

### Step 6: Update Address List Template

Add delete UI elements to the address list template. This follows the existing BOQ delete pattern from `boq_list.templ`.

#### Single Delete Button (per row)

For non-Ship-To addresses, use a simple `hx-confirm`:

```html
<button
    hx-delete={fmt.Sprintf("/projects/%s/addresses/%s/%s", projectID, addrType, addr.ID)}
    hx-confirm="Are you sure you want to delete this address?"
    class="btn btn-ghost btn-sm text-error"
>
    <!-- trash icon SVG -->
</button>
```

For Ship To addresses, use an Alpine.js modal that fetches linked count first:

```html
<div x-data="{ showDeleteModal: false, linkedCount: 0, companyName: '' }">
    <button
        @click={fmt.Sprintf(`
            fetch('/projects/%s/addresses/ship_to/%s/delete-info')
                .then(r => r.json())
                .then(d => {
                    linkedCount = d.linked_count;
                    companyName = d.company_name;
                    showDeleteModal = true;
                })
        `, projectID, addr.ID)}
        class="btn btn-ghost btn-sm text-error"
    >
        <!-- trash icon SVG -->
    </button>

    <!-- Delete confirmation modal -->
    <div x-show="showDeleteModal" x-cloak class="fixed inset-0 z-50 flex items-center justify-center"
         style="background: rgba(0,0,0,0.5);">
        <div class="bg-white p-6 rounded shadow-lg max-w-md" @click.outside="showDeleteModal = false">
            <h3 class="text-lg font-bold mb-2">Delete Ship To Address</h3>
            <p x-text="`Delete '${companyName}'?`" class="mb-2"></p>
            <template x-if="linkedCount > 0">
                <p class="text-warning mb-4"
                   x-text="`Warning: ${linkedCount} Install At address(es) are linked to this Ship To. Their link will be removed.`">
                </p>
            </template>
            <div class="flex justify-end gap-2 mt-4">
                <button @click="showDeleteModal = false" class="btn btn-ghost">Cancel</button>
                <button
                    @click={fmt.Sprintf(`
                        htmx.ajax('DELETE', '/projects/%s/addresses/ship_to/%s', {target: '#main-content'});
                        showDeleteModal = false;
                    `, projectID, addr.ID)}
                    class="btn btn-error"
                >
                    Delete
                </button>
            </div>
        </div>
    </div>
</div>
```

#### Bulk Delete UI

Add checkbox selection and a bulk delete button at the top of the address list:

```html
<div x-data="{ selected: [], selectAll: false }">
    <!-- Bulk actions bar (visible when items selected) -->
    <div x-show="selected.length > 0" class="flex items-center gap-4 mb-4 p-3 bg-base-200 rounded">
        <span x-text="`${selected.length} selected`" class="text-sm font-medium"></span>
        <button
            @click={fmt.Sprintf(`
                if (confirm('Delete ' + selected.length + ' addresses?')) {
                    htmx.ajax('DELETE', '/projects/%s/addresses/%s/bulk', {
                        target: '#main-content',
                        values: JSON.stringify({ids: selected}),
                        headers: {'Content-Type': 'application/json'}
                    });
                }
            `, projectID, addrType)}
            class="btn btn-error btn-sm"
        >
            Delete Selected
        </button>
    </div>

    <!-- Table header with select-all checkbox -->
    <div class="flex items-center ...">
        <input type="checkbox"
               x-model="selectAll"
               @change="selected = selectAll ? allIds : []"
               class="checkbox checkbox-sm" />
        <!-- ... other header columns ... -->
    </div>

    <!-- Each row has a checkbox -->
    <div class="flex items-center ...">
        <input type="checkbox"
               :value="addr.ID"
               x-model="selected"
               class="checkbox checkbox-sm" />
        <!-- ... other data columns ... -->
    </div>
</div>
```

For Ship To bulk delete, the same cascade logic applies server-side. The confirmation message for bulk delete can be simpler since showing per-address linked counts would be cumbersome. The server handles nullification for all selected Ship To addresses.

---

## Dependencies on Other Phases

- **Phase 10-12 (assumed)**: Address collections, CRUD handlers, and list templates must exist.
- **Phase 13**: Not directly dependent, but the `addressTypeLabels` map defined in `handlers/address_export.go` (Phase 13) can be shared. If Phase 13 is not yet complete, duplicate the map in `address_delete.go` or extract it to a shared file like `handlers/address_common.go`.
- The `sanitizeFilename` helper and HX-Request/HX-Redirect pattern are already established in `handlers/export.go` and `handlers/boq_delete.go`.

---

## Testing / Verification Steps

1. **Single delete - non-Ship-To types**:
   - Create a Bill To address. Click delete. Confirm the `hx-confirm` dialog appears.
   - After confirming, verify the address is removed and the list refreshes.
   - Verify the address is gone from PocketBase admin panel.

2. **Single delete - Ship To with linked Install At**:
   - Create a Ship To address and 3 Install At addresses linked to it.
   - Click delete on the Ship To. Verify the Alpine.js modal shows "3 Install At address(es) are linked."
   - Confirm deletion. Verify:
     - Ship To address is deleted.
     - All 3 Install At addresses still exist but their `ship_to_parent` field is now empty.
   - Check in PocketBase admin that Install At records are intact with nullified parent.

3. **Single delete - Ship To with no linked Install At**:
   - Delete a Ship To that has no linked Install At. Modal should show no warning.

4. **Bulk delete**:
   - Select 3 addresses using checkboxes. Click "Delete Selected."
   - Verify all 3 are deleted and the list refreshes.
   - Select all using the header checkbox, then deselect one. Verify count is correct.

5. **Bulk delete - Ship To cascade**:
   - Select 2 Ship To addresses for bulk delete, each with linked Install At addresses.
   - After deletion, verify all linked Install At addresses have nullified `ship_to_parent`.

6. **Error cases**:
   - Try deleting an address that belongs to a different project: expect 403.
   - Try bulk delete with an empty IDs array: expect 400.
   - Try deleting a non-existent address ID: expect 404.

7. **Regression**:
   - BOQ delete (`DELETE /boq/{id}`) still works with cascade as before.
   - No compilation errors.

---

## Acceptance Criteria

- [ ] `DELETE /projects/{projectId}/addresses/{type}/{addressId}` deletes a single address and returns HX-Redirect for HTMX requests.
- [ ] `DELETE /projects/{projectId}/addresses/{type}/bulk` accepts a JSON body with IDs and deletes all specified addresses.
- [ ] Deleting a Ship To address nullifies (sets to empty) the `ship_to_parent` field on all linked Install At addresses; it does NOT delete the Install At addresses.
- [ ] Ship To single-delete shows an Alpine.js modal with the count of linked Install At addresses before confirmation.
- [ ] Non-Ship-To single-delete uses a simpler `hx-confirm` dialog.
- [ ] Bulk delete with Ship To addresses applies the same nullification cascade for each deleted Ship To.
- [ ] Address ownership is validated: cannot delete an address belonging to a different project.
- [ ] HTMX responses trigger a redirect that refreshes the address list after deletion.
- [ ] Select-all checkbox and individual checkboxes work correctly for bulk selection.
- [ ] Bulk action bar appears only when at least one address is selected.
- [ ] All error cases (invalid type, missing ID, wrong project) return appropriate HTTP status codes.
