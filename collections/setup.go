package collections

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// Setup programmatically creates/ensures the boqs, main_boq_items,
// sub_items and sub_sub_items collections exist.
func Setup(app *pocketbase.PocketBase) {
	boqs := ensureCollection(app, "boqs", func(c *core.Collection) {
		c.Fields.Add(&core.TextField{Name: "title", Required: true})
		c.Fields.Add(&core.TextField{Name: "reference_number", Required: false})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	})

	mainBOQItems := ensureCollection(app, "main_boq_items", func(c *core.Collection) {
		c.Fields.Add(&core.RelationField{
			Name:          "boq",
			Required:      true,
			CollectionId:  boqs.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})
		c.Fields.Add(&core.NumberField{Name: "sort_order", Required: true})
		c.Fields.Add(&core.TextField{Name: "description", Required: true})
		c.Fields.Add(&core.NumberField{Name: "qty", Required: true})
		c.Fields.Add(&core.TextField{Name: "uom", Required: true})
		c.Fields.Add(&core.NumberField{Name: "quoted_price", Required: true})
		c.Fields.Add(&core.NumberField{Name: "budgeted_price", Required: false})
		c.Fields.Add(&core.TextField{Name: "hsn_code", Required: false})
		c.Fields.Add(&core.NumberField{Name: "gst_percent", Required: true})
	})

	subItems := ensureCollection(app, "sub_items", func(c *core.Collection) {
		c.Fields.Add(&core.RelationField{
			Name:          "main_item",
			Required:      true,
			CollectionId:  mainBOQItems.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})
		c.Fields.Add(&core.NumberField{Name: "sort_order", Required: true})
		c.Fields.Add(&core.SelectField{
			Name:      "type",
			Required:  true,
			Values:    []string{"product", "service"},
			MaxSelect: 1,
		})
		c.Fields.Add(&core.TextField{Name: "description", Required: true})
		c.Fields.Add(&core.NumberField{Name: "qty_per_unit", Required: true})
		c.Fields.Add(&core.TextField{Name: "uom", Required: true})
		c.Fields.Add(&core.NumberField{Name: "unit_price", Required: true})
		c.Fields.Add(&core.NumberField{Name: "budgeted_price", Required: false})
		c.Fields.Add(&core.TextField{Name: "hsn_code", Required: false})
		c.Fields.Add(&core.NumberField{Name: "gst_percent", Required: true})
	})

	ensureCollection(app, "sub_sub_items", func(c *core.Collection) {
		c.Fields.Add(&core.RelationField{
			Name:          "sub_item",
			Required:      true,
			CollectionId:  subItems.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})
		c.Fields.Add(&core.NumberField{Name: "sort_order", Required: true})
		c.Fields.Add(&core.SelectField{
			Name:      "type",
			Required:  true,
			Values:    []string{"product", "service"},
			MaxSelect: 1,
		})
		c.Fields.Add(&core.TextField{Name: "description", Required: true})
		c.Fields.Add(&core.NumberField{Name: "qty_per_unit", Required: true})
		c.Fields.Add(&core.TextField{Name: "uom", Required: true})
		c.Fields.Add(&core.NumberField{Name: "unit_price", Required: true})
		c.Fields.Add(&core.NumberField{Name: "budgeted_price", Required: false})
		c.Fields.Add(&core.TextField{Name: "hsn_code", Required: false})
		c.Fields.Add(&core.NumberField{Name: "gst_percent", Required: true})
	})
}

// ensureCollection checks if a collection already exists by name. If it does,
// the existing collection is returned. Otherwise a new base collection is
// created, the addFields callback is invoked to populate its fields, and the
// collection is saved.
func ensureCollection(app *pocketbase.PocketBase, name string, addFields func(*core.Collection)) *core.Collection {
	existing, err := app.FindCollectionByNameOrId(name)
	if err == nil && existing != nil {
		log.Printf("Collection %q already exists, skipping creation.\n", name)
		return existing
	}

	collection := core.NewBaseCollection(name)
	addFields(collection)

	if err := app.Save(collection); err != nil {
		log.Fatalf("Failed to create collection %q: %v", name, err)
	}

	fmt.Printf("Created collection %q (id=%s)\n", name, collection.Id)
	return collection
}
