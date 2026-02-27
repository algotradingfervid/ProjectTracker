package collections

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// Setup programmatically creates/ensures the projects, boqs, main_boq_items,
// sub_items, sub_sub_items, addresses, and project_address_settings collections.
func Setup(app *pocketbase.PocketBase) {
	// ── Projects (top-level container for BOQs and addresses) ────────
	projects := ensureCollection(app, "projects", func(c *core.Collection) {
		c.Fields.Add(&core.TextField{Name: "name", Required: true})
		c.Fields.Add(&core.TextField{Name: "client_name", Required: false})
		c.Fields.Add(&core.TextField{Name: "reference_number", Required: false})
		c.Fields.Add(&core.SelectField{
			Name:      "status",
			Required:  true,
			Values:    []string{"active", "completed", "on_hold"},
			MaxSelect: 1,
		})
		c.Fields.Add(&core.BoolField{Name: "ship_to_equals_install_at"})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	})

	// ── BOQs ─────────────────────────────────────────────────────────
	boqs := ensureCollection(app, "boqs", func(c *core.Collection) {
		c.Fields.Add(&core.TextField{Name: "title", Required: true})
		c.Fields.Add(&core.TextField{Name: "reference_number", Required: false})
		c.Fields.Add(&core.RelationField{
			Name:          "project",
			Required:      false,
			CollectionId:  projects.Id,
			CascadeDelete: false,
			MaxSelect:     1,
		})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	})

	// Add project relation to existing boqs collection (idempotent)
	ensureField(app, "boqs", &core.RelationField{
		Name:          "project",
		Required:      false,
		CollectionId:  projects.Id,
		CascadeDelete: false,
		MaxSelect:     1,
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
		c.Fields.Add(&core.NumberField{Name: "unit_price", Required: true})
		c.Fields.Add(&core.NumberField{Name: "quoted_price", Required: false})
		c.Fields.Add(&core.NumberField{Name: "budgeted_price", Required: false})
		c.Fields.Add(&core.NumberField{Name: "actual_price", Required: false})
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
		c.Fields.Add(&core.NumberField{Name: "actual_price", Required: false})
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
		c.Fields.Add(&core.NumberField{Name: "actual_price", Required: false})
		c.Fields.Add(&core.TextField{Name: "hsn_code", Required: false})
		c.Fields.Add(&core.NumberField{Name: "gst_percent", Required: true})
	})

	// ── Addresses ────────────────────────────────────────────────────
	addresses := ensureCollection(app, "addresses", func(c *core.Collection) {
		c.Fields.Add(&core.SelectField{
			Name:      "address_type",
			Required:  true,
			Values:    []string{"bill_from", "ship_from", "bill_to", "ship_to", "install_at"},
			MaxSelect: 1,
		})
		c.Fields.Add(&core.RelationField{
			Name:          "project",
			Required:      true,
			CollectionId:  projects.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})

		// Company & contact fields
		c.Fields.Add(&core.TextField{Name: "company_name", Required: false})
		c.Fields.Add(&core.TextField{Name: "contact_person", Required: false})

		// Address fields
		c.Fields.Add(&core.TextField{Name: "address_line_1", Required: false})
		c.Fields.Add(&core.TextField{Name: "address_line_2", Required: false})
		c.Fields.Add(&core.TextField{Name: "city", Required: false})
		c.Fields.Add(&core.TextField{Name: "state", Required: false})
		c.Fields.Add(&core.TextField{Name: "pin_code", Required: false})
		c.Fields.Add(&core.TextField{Name: "country", Required: false})
		c.Fields.Add(&core.TextField{Name: "landmark", Required: false})
		c.Fields.Add(&core.TextField{Name: "district", Required: false})

		// Contact fields
		c.Fields.Add(&core.TextField{Name: "phone", Required: false})
		c.Fields.Add(&core.EmailField{Name: "email", Required: false})
		c.Fields.Add(&core.TextField{Name: "fax", Required: false})
		c.Fields.Add(&core.URLField{Name: "website", Required: false})

		// Tax / registration fields
		c.Fields.Add(&core.TextField{Name: "gstin", Required: false})
		c.Fields.Add(&core.TextField{Name: "pan", Required: false})
		c.Fields.Add(&core.TextField{Name: "cin", Required: false})

		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	})

	// Add ship_to_parent self-relation (must be after addresses is created)
	ensureField(app, "addresses", &core.RelationField{
		Name:          "ship_to_parent",
		Required:      false,
		CollectionId:  addresses.Id,
		CascadeDelete: false,
		MaxSelect:     1,
	})

	// ── Project Address Settings ─────────────────────────────────────
	ensureCollection(app, "project_address_settings", func(c *core.Collection) {
		c.Fields.Add(&core.RelationField{
			Name:          "project",
			Required:      true,
			CollectionId:  projects.Id,
			CascadeDelete: true,
			MaxSelect:     1,
		})
		c.Fields.Add(&core.SelectField{
			Name:      "address_type",
			Required:  true,
			Values:    []string{"bill_from", "ship_from", "bill_to", "ship_to", "install_at"},
			MaxSelect: 1,
		})

		// Boolean fields: true = this field is required for this address type
		c.Fields.Add(&core.BoolField{Name: "req_company_name"})
		c.Fields.Add(&core.BoolField{Name: "req_contact_person"})
		c.Fields.Add(&core.BoolField{Name: "req_address_line_1"})
		c.Fields.Add(&core.BoolField{Name: "req_address_line_2"})
		c.Fields.Add(&core.BoolField{Name: "req_city"})
		c.Fields.Add(&core.BoolField{Name: "req_state"})
		c.Fields.Add(&core.BoolField{Name: "req_pin_code"})
		c.Fields.Add(&core.BoolField{Name: "req_country"})
		c.Fields.Add(&core.BoolField{Name: "req_landmark"})
		c.Fields.Add(&core.BoolField{Name: "req_district"})
		c.Fields.Add(&core.BoolField{Name: "req_phone"})
		c.Fields.Add(&core.BoolField{Name: "req_email"})
		c.Fields.Add(&core.BoolField{Name: "req_fax"})
		c.Fields.Add(&core.BoolField{Name: "req_website"})
		c.Fields.Add(&core.BoolField{Name: "req_gstin"})
		c.Fields.Add(&core.BoolField{Name: "req_pan"})
		c.Fields.Add(&core.BoolField{Name: "req_cin"})

		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	})

	// ── Vendors (global vendor directory) ────────────────────────────
	vendors := ensureCollection(app, "vendors", func(c *core.Collection) {
		c.Fields.Add(&core.TextField{Name: "name", Required: true})
		c.Fields.Add(&core.TextField{Name: "address_line_1"})
		c.Fields.Add(&core.TextField{Name: "address_line_2"})
		c.Fields.Add(&core.TextField{Name: "city"})
		c.Fields.Add(&core.TextField{Name: "state"})
		c.Fields.Add(&core.TextField{Name: "pin_code"})
		c.Fields.Add(&core.TextField{Name: "country"})
		c.Fields.Add(&core.TextField{Name: "gstin"})
		c.Fields.Add(&core.TextField{Name: "pan"})
		c.Fields.Add(&core.TextField{Name: "contact_name"})
		c.Fields.Add(&core.TextField{Name: "phone"})
		c.Fields.Add(&core.EmailField{Name: "email"})
		c.Fields.Add(&core.URLField{Name: "website"})
		c.Fields.Add(&core.TextField{Name: "bank_beneficiary_name"})
		c.Fields.Add(&core.TextField{Name: "bank_name"})
		c.Fields.Add(&core.TextField{Name: "bank_account_no"})
		c.Fields.Add(&core.TextField{Name: "bank_ifsc"})
		c.Fields.Add(&core.TextField{Name: "bank_branch"})
		c.Fields.Add(&core.TextField{Name: "notes"})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	})

	// ── Project Vendors (linking table) ──────────────────────────────
	ensureCollection(app, "project_vendors", func(c *core.Collection) {
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
	})
	ensureField(app, "project_vendors", &core.RelationField{
		Name: "project", Required: true,
		CollectionId: projects.Id, CascadeDelete: true, MaxSelect: 1,
	})
	ensureField(app, "project_vendors", &core.RelationField{
		Name: "vendor", Required: true,
		CollectionId: vendors.Id, CascadeDelete: true, MaxSelect: 1,
	})

	// ── Purchase Orders ──────────────────────────────────────────────
	purchaseOrders := ensureCollection(app, "purchase_orders", func(c *core.Collection) {
		c.Fields.Add(&core.TextField{Name: "po_number", Required: true})
		c.Fields.Add(&core.TextField{Name: "order_date"})
		c.Fields.Add(&core.TextField{Name: "quotation_ref"})
		c.Fields.Add(&core.TextField{Name: "ref_date"})
		c.Fields.Add(&core.TextField{Name: "payment_terms"})
		c.Fields.Add(&core.TextField{Name: "delivery_terms"})
		c.Fields.Add(&core.TextField{Name: "warranty_terms"})
		c.Fields.Add(&core.TextField{Name: "comments"})
		c.Fields.Add(&core.SelectField{
			Name: "status", Required: true,
			Values:    []string{"draft", "sent", "acknowledged", "completed", "cancelled"},
			MaxSelect: 1,
		})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	})
	ensureField(app, "purchase_orders", &core.RelationField{
		Name: "project", Required: true,
		CollectionId: projects.Id, CascadeDelete: true, MaxSelect: 1,
	})
	ensureField(app, "purchase_orders", &core.RelationField{
		Name: "vendor", Required: true,
		CollectionId: vendors.Id, CascadeDelete: false, MaxSelect: 1,
	})
	ensureField(app, "purchase_orders", &core.RelationField{
		Name: "bill_to_address", Required: false,
		CollectionId: addresses.Id, CascadeDelete: false, MaxSelect: 1,
	})
	ensureField(app, "purchase_orders", &core.RelationField{
		Name: "ship_to_address", Required: false,
		CollectionId: addresses.Id, CascadeDelete: false, MaxSelect: 1,
	})

	// ── PO Line Items ────────────────────────────────────────────────
	ensureCollection(app, "po_line_items", func(c *core.Collection) {
		c.Fields.Add(&core.NumberField{Name: "sort_order", Required: true})
		c.Fields.Add(&core.TextField{Name: "description", Required: true})
		c.Fields.Add(&core.TextField{Name: "hsn_code"})
		c.Fields.Add(&core.NumberField{Name: "qty", Required: true})
		c.Fields.Add(&core.TextField{Name: "uom", Required: true})
		c.Fields.Add(&core.NumberField{Name: "rate", Required: true})
		c.Fields.Add(&core.NumberField{Name: "gst_percent", Required: true})
		c.Fields.Add(&core.SelectField{
			Name:      "source_item_type",
			Values:    []string{"main_item", "sub_item", "sub_sub_item", "manual"},
			MaxSelect: 1,
		})
		c.Fields.Add(&core.TextField{Name: "source_item_id"})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	})
	ensureField(app, "po_line_items", &core.RelationField{
		Name: "purchase_order", Required: true,
		CollectionId: purchaseOrders.Id, CascadeDelete: true, MaxSelect: 1,
	})
}

// ensureField adds a field to an existing collection if it doesn't already exist.
// It is a no-op if the field is already present.
func ensureField(app *pocketbase.PocketBase, collectionName string, field core.Field) {
	col, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		log.Printf("ensureField: collection %q not found, skipping field add.\n", collectionName)
		return
	}

	existingField := col.Fields.GetByName(field.GetName())
	if existingField != nil {
		return
	}

	col.Fields.Add(field)
	if err := app.Save(col); err != nil {
		log.Printf("ensureField: failed to add field %q to %q: %v\n", field.GetName(), collectionName, err)
	} else {
		log.Printf("ensureField: added field %q to collection %q\n", field.GetName(), collectionName)
	}
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
