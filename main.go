package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/collections"
	"projectcreation/handlers"
)

func main() {
	app := pocketbase.New()

	// Create collections and seed data on startup
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		collections.Setup(app)
		if err := collections.Seed(app); err != nil {
			log.Printf("Warning: seed data failed: %v", err)
		}
		if err := collections.MigrateOrphanBOQsToProjects(app); err != nil {
			log.Printf("Warning: project migration failed: %v", err)
		}
		if err := collections.MigrateDefaultAddressSettings(app); err != nil {
			log.Printf("Warning: address settings migration failed: %v", err)
		}
		return se.Next()
	})

	// Serve static files from ./static
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/static/{path...}", apis.Static(os.DirFS("./static"), false))

		// Apply active project middleware globally
		se.Router.BindFunc(handlers.ActiveProjectMiddleware(app))

		// ── Project activation ───────────────────────────────────
		se.Router.POST("/projects/{id}/activate", handlers.HandleProjectActivate(app))
		se.Router.POST("/projects/deactivate", handlers.HandleProjectDeactivate(app))

		// ── Project CRUD ─────────────────────────────────────────
		se.Router.GET("/projects", handlers.HandleProjectList(app))
		se.Router.GET("/projects/create", handlers.HandleProjectCreate(app))
		se.Router.POST("/projects", handlers.HandleProjectSave(app))
		se.Router.GET("/projects/{id}/edit", handlers.HandleProjectEdit(app))
		se.Router.POST("/projects/{id}/save", handlers.HandleProjectUpdate(app))
		se.Router.DELETE("/projects/{id}", handlers.HandleProjectDelete(app))
		se.Router.GET("/projects/{id}/settings", handlers.HandleProjectSettings(app))
		se.Router.GET("/projects/{id}", handlers.HandleProjectView(app))
		se.Router.POST("/projects/{id}/settings", handlers.HandleProjectSettingsSave(app))

		// Project-scoped address routes
		addressTypes := []struct {
			slug     string
			addrType handlers.AddressType
		}{
			{"bill-from", handlers.AddressTypeBillFrom},
			{"ship-from", handlers.AddressTypeShipFrom},
			{"bill-to", handlers.AddressTypeBillTo},
			{"ship-to", handlers.AddressTypeShipTo},
			{"install-at", handlers.AddressTypeInstallAt},
		}
		for _, at := range addressTypes {
			se.Router.GET(
				"/projects/{projectId}/addresses/"+at.slug,
				handlers.HandleAddressList(app, at.addrType),
			)
			se.Router.GET(
				"/projects/{projectId}/addresses/"+at.slug+"/count",
				handlers.HandleAddressCount(app, at.addrType),
			)

			// Create form (GET renders form, POST saves new address)
			se.Router.GET(
				"/projects/{projectId}/addresses/"+at.slug+"/new",
				handlers.HandleAddressCreate(app, at.addrType),
			)
			se.Router.POST(
				"/projects/{projectId}/addresses/"+at.slug+"/new",
				handlers.HandleAddressSave(app, at.addrType),
			)

			// Edit form (GET renders form, POST updates address)
			se.Router.GET(
				"/projects/{projectId}/addresses/"+at.slug+"/{addressId}/edit",
				handlers.HandleAddressEdit(app, at.addrType),
			)
			se.Router.POST(
				"/projects/{projectId}/addresses/"+at.slug+"/{addressId}/edit",
				handlers.HandleAddressUpdate(app, at.addrType),
			)
		}

		// Address template download
		se.Router.GET("/projects/{projectId}/addresses/{type}/template",
			handlers.HandleAddressTemplateDownload(app))

		// Address import - upload & validate
		se.Router.GET("/projects/{projectId}/addresses/{type}/import",
			handlers.HandleAddressImportPage(app))
		se.Router.POST("/projects/{projectId}/addresses/{type}/import",
			handlers.HandleAddressValidate(app))

		// Address import - commit
		se.Router.POST("/projects/{projectId}/addresses/{type}/import/commit",
			handlers.HandleAddressImportCommit(app))

		// Address import - download error report
		se.Router.POST("/projects/{projectId}/addresses/{type}/import/errors",
			handlers.HandleAddressErrorReport(app))

		// Address export
		se.Router.GET("/projects/{projectId}/addresses/{type}/export",
			handlers.HandleAddressExportExcel(app))

		// Address delete operations (bulk must be before {addressId} to avoid matching "bulk" as an ID)
		se.Router.DELETE("/projects/{projectId}/addresses/{type}/bulk",
			handlers.HandleAddressBulkDelete(app))
		se.Router.GET("/projects/{projectId}/addresses/{type}/{addressId}/delete-info",
			handlers.HandleAddressDeleteInfo(app))
		se.Router.DELETE("/projects/{projectId}/addresses/{type}/{addressId}",
			handlers.HandleAddressDelete(app))

		// ── Project-scoped BOQ routes ───────────────────────────
		// BOQ creation
		se.Router.GET("/projects/{projectId}/boq/create", handlers.HandleBOQCreate(app))
		se.Router.POST("/projects/{projectId}/boq", handlers.HandleBOQSave(app))

		// BOQ edit mode
		se.Router.GET("/projects/{projectId}/boq/{id}/edit", handlers.HandleBOQEdit(app))
		se.Router.GET("/projects/{projectId}/boq/{id}/view", handlers.HandleBOQViewMode(app))
		se.Router.POST("/projects/{projectId}/boq/{id}/save", handlers.HandleBOQUpdate(app))

		// BOQ delete
		se.Router.DELETE("/projects/{projectId}/boq/{id}", handlers.HandleBOQDelete(app))

		// BOQ export
		se.Router.GET("/projects/{projectId}/boq/{id}/export/excel", handlers.HandleBOQExportExcel(app))
		se.Router.GET("/projects/{projectId}/boq/{id}/export/pdf", handlers.HandleBOQExportPDF(app))

		// BOQ edit - add items
		se.Router.POST("/projects/{projectId}/boq/{id}/main-items", handlers.HandleAddMainItem(app))
		se.Router.POST("/projects/{projectId}/boq/{id}/main-item/{mainItemId}/subitems", handlers.HandleAddSubItem(app))
		se.Router.POST("/projects/{projectId}/boq/{id}/subitem/{subItemId}/subsubitems", handlers.HandleAddSubSubItem(app))

		// BOQ edit - delete items
		se.Router.DELETE("/projects/{projectId}/boq/{id}/main-item/{itemId}", handlers.HandleDeleteMainItem(app))
		se.Router.DELETE("/projects/{projectId}/boq/{id}/subitem/{subItemId}", handlers.HandleDeleteSubItem(app))
		se.Router.DELETE("/projects/{projectId}/boq/{id}/subsubitem/{subSubItemId}", handlers.HandleDeleteSubSubItem(app))

		// BOQ edit - expand/collapse (lazy load sub-items)
		se.Router.GET("/projects/{projectId}/boq/{id}/main-item/{itemId}/subitems", handlers.HandleExpandMainItem(app))

		// BOQ edit - patch individual fields (optional auto-save)
		se.Router.PATCH("/projects/{projectId}/boq/{id}/main-item/{itemId}", handlers.HandlePatchMainItem(app))
		se.Router.PATCH("/projects/{projectId}/boq/{id}/subitem/{subItemId}", handlers.HandlePatchSubItem(app))
		se.Router.PATCH("/projects/{projectId}/boq/{id}/subsubitem/{subSubItemId}", handlers.HandlePatchSubSubItem(app))

		// BOQ view (must be after specific /boq/{id}/* routes)
		se.Router.GET("/projects/{projectId}/boq/{id}", handlers.HandleBOQView(app))

		// BOQ list page
		se.Router.GET("/projects/{projectId}/boq", handlers.HandleBOQList(app))

		// ── Vendor CRUD (global) ─────────────────────────────────
		se.Router.GET("/vendors", handlers.HandleVendorList(app))
		se.Router.GET("/vendors/create", handlers.HandleVendorCreate(app))
		se.Router.POST("/vendors", handlers.HandleVendorSave(app))
		se.Router.GET("/vendors/{id}/edit", handlers.HandleVendorEdit(app))
		se.Router.POST("/vendors/{id}/save", handlers.HandleVendorUpdate(app))
		se.Router.DELETE("/vendors/{id}", handlers.HandleVendorDelete(app))

		// ── Vendor (project-scoped) ──────────────────────────────
		se.Router.GET("/projects/{projectId}/vendors", handlers.HandleVendorList(app))
		se.Router.GET("/projects/{projectId}/vendors/create", handlers.HandleVendorCreate(app))
		se.Router.POST("/projects/{projectId}/vendors", handlers.HandleVendorSave(app))
		se.Router.POST("/projects/{projectId}/vendors/{id}/link", handlers.HandleVendorLink(app))
		se.Router.DELETE("/projects/{projectId}/vendors/{id}/link", handlers.HandleVendorUnlink(app))

		// ── Purchase Order CRUD ──────────────────────────────────
		se.Router.GET("/projects/{projectId}/po/create", handlers.HandlePOCreate(app))
		se.Router.POST("/projects/{projectId}/po", handlers.HandlePOSave(app))
		se.Router.GET("/projects/{projectId}/po/{id}/edit", handlers.HandlePOEdit(app))
		se.Router.POST("/projects/{projectId}/po/{id}/save", handlers.HandlePOUpdate(app))

		// ── PO Line Items ───────────────────────────────────────
		se.Router.POST("/projects/{projectId}/po/{id}/line-items", handlers.HandlePOAddLineItem(app))
		se.Router.POST("/projects/{projectId}/po/{id}/line-items/from-boq", handlers.HandlePOAddLineItemFromBOQ(app))
		se.Router.PATCH("/projects/{projectId}/po/{id}/line-items/{itemId}", handlers.HandlePOUpdateLineItem(app))
		se.Router.DELETE("/projects/{projectId}/po/{id}/line-items/{itemId}", handlers.HandlePODeleteLineItem(app))

		// ── BOQ Picker ──────────────────────────────────────────
		se.Router.GET("/projects/{projectId}/po/{id}/boq-picker", handlers.HandlePOBOQPicker(app))

		// ── PO Export ───────────────────────────────────────────
		se.Router.GET("/projects/{projectId}/po/{id}/export/pdf", handlers.HandlePOExportPDF(app))

		// ── PO List, View, Delete (after specific /po/{id}/* routes) ──
		se.Router.GET("/projects/{projectId}/po", handlers.HandlePOList(app))
		se.Router.GET("/projects/{projectId}/po/{id}", handlers.HandlePOView(app))
		se.Router.DELETE("/projects/{projectId}/po/{id}", handlers.HandlePODelete(app))

		// ── Legacy BOQ redirects ─────────────────────────────────
		se.Router.GET("/boq", func(e *core.RequestEvent) error {
			activeProject := handlers.GetActiveProject(e.Request)
			if activeProject != nil {
				return e.Redirect(http.StatusFound, fmt.Sprintf("/projects/%s/boq", activeProject.ID))
			}
			return e.Redirect(http.StatusFound, "/projects")
		})

		se.Router.GET("/boq/{id}", func(e *core.RequestEvent) error {
			boqID := e.Request.PathValue("id")
			boq, err := app.FindRecordById("boqs", boqID)
			if err != nil {
				return e.String(http.StatusNotFound, "BOQ not found")
			}
			projectID := boq.GetString("project")
			if projectID == "" {
				return e.String(http.StatusNotFound, "BOQ has no project")
			}
			return e.Redirect(http.StatusFound, fmt.Sprintf("/projects/%s/boq/%s", projectID, boqID))
		})

		// Redirect home to projects list
		se.Router.GET("/", func(e *core.RequestEvent) error {
			return e.Redirect(http.StatusFound, "/projects")
		})

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
