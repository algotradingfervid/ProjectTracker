package main

import (
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
		return se.Next()
	})

	// Serve static files from ./static
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/static/{path...}", apis.Static(os.DirFS("./static"), false))

		// BOQ creation
		se.Router.GET("/boq/create", handlers.HandleBOQCreate(app))
		se.Router.POST("/boq", handlers.HandleBOQSave(app))

		// BOQ edit mode
		se.Router.GET("/boq/{id}/edit", handlers.HandleBOQEdit(app))
		se.Router.GET("/boq/{id}/view", handlers.HandleBOQViewMode(app))
		se.Router.POST("/boq/{id}/save", handlers.HandleBOQUpdate(app))

		// BOQ delete
		se.Router.DELETE("/boq/{id}", handlers.HandleBOQDelete(app))

		// BOQ export
		se.Router.GET("/boq/{id}/export/excel", handlers.HandleBOQExportExcel(app))
		se.Router.GET("/boq/{id}/export/pdf", handlers.HandleBOQExportPDF(app))

		// BOQ edit - add items (new REST-style endpoints)
		se.Router.POST("/boq/{id}/main-items", handlers.HandleAddMainItem(app))
		se.Router.POST("/boq/{id}/main-item/{mainItemId}/subitems", handlers.HandleAddSubItem(app))
		se.Router.POST("/boq/{id}/subitem/{subItemId}/subsubitems", handlers.HandleAddSubSubItem(app))

		// BOQ edit - delete items
		se.Router.DELETE("/boq/{id}/main-item/{itemId}", handlers.HandleDeleteMainItem(app))
		se.Router.DELETE("/boq/{id}/subitem/{subItemId}", handlers.HandleDeleteSubItem(app))
		se.Router.DELETE("/boq/{id}/subsubitem/{subSubItemId}", handlers.HandleDeleteSubSubItem(app))

		// BOQ edit - expand/collapse (lazy load sub-items)
		se.Router.GET("/boq/{id}/main-item/{itemId}/subitems", handlers.HandleExpandMainItem(app))

		// BOQ edit - patch individual fields (optional auto-save)
		se.Router.PATCH("/boq/{id}/main-item/{itemId}", handlers.HandlePatchMainItem(app))
		se.Router.PATCH("/boq/{id}/subitem/{subItemId}", handlers.HandlePatchSubItem(app))
		se.Router.PATCH("/boq/{id}/subsubitem/{subSubItemId}", handlers.HandlePatchSubSubItem(app))

		// BOQ view (must be after specific /boq/{id}/* routes)
		se.Router.GET("/boq/{id}", handlers.HandleBOQView(app))

		// BOQ list page
		se.Router.GET("/boq", handlers.HandleBOQList(app))

		// Redirect home to BOQ list
		se.Router.GET("/", func(e *core.RequestEvent) error {
			return e.Redirect(http.StatusFound, "/boq")
		})

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
