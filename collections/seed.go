package collections

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// Seed populates the boqs collection (and related items) with realistic
// BOQ data when the database is empty. It is safe to call on every startup
// because it returns early if any BOQ records already exist.
func Seed(app *pocketbase.PocketBase) error {
	// ── check if boqs collection already has data ──────────────────────
	boqsCol, err := app.FindCollectionByNameOrId("boqs")
	if err != nil {
		return fmt.Errorf("seed: could not find boqs collection: %w", err)
	}

	existing, err := app.FindAllRecords(boqsCol)
	if err != nil {
		return fmt.Errorf("seed: could not query boqs: %w", err)
	}
	if len(existing) > 0 {
		return nil // already seeded
	}

	log.Println("seed: boqs collection is empty – inserting seed data …")

	// ── 1. Create the BOQ ─────────────────────────────────────────────
	boqRecord := core.NewRecord(boqsCol)
	boqRecord.Set("title", "Interior Fit-Out — Block A")
	boqRecord.Set("reference_number", "PO-2025-001")
	if err := app.Save(boqRecord); err != nil {
		return fmt.Errorf("seed: save boq: %w", err)
	}
	boqID := boqRecord.Id

	// ── helper collections ────────────────────────────────────────────
	mainItemsCol, err := app.FindCollectionByNameOrId("main_boq_items")
	if err != nil {
		return fmt.Errorf("seed: could not find main_boq_items collection: %w", err)
	}
	subItemsCol, err := app.FindCollectionByNameOrId("sub_items")
	if err != nil {
		return fmt.Errorf("seed: could not find sub_items collection: %w", err)
	}
	subSubItemsCol, err := app.FindCollectionByNameOrId("sub_sub_items")
	if err != nil {
		return fmt.Errorf("seed: could not find sub_sub_items collection: %w", err)
	}

	// ── helper: create a sub-sub item ─────────────────────────────────
	type subSubDef struct {
		sortOrder    int
		itemType     string
		description  string
		qtyPerUnit   float64
		uom          string
		unitPrice    float64
		gstPercent   int
		hsnCode      string
	}

	createSubSubItem := func(parentID string, d subSubDef) error {
		budgeted := d.qtyPerUnit * d.unitPrice

		r := core.NewRecord(subSubItemsCol)
		r.Set("sub_item", parentID)
		r.Set("sort_order", d.sortOrder)
		r.Set("type", d.itemType)
		r.Set("description", d.description)
		r.Set("qty_per_unit", d.qtyPerUnit)
		r.Set("uom", d.uom)
		r.Set("unit_price", d.unitPrice)
		r.Set("budgeted_price", budgeted)
		r.Set("gst_percent", d.gstPercent)
		if d.hsnCode != "" {
			r.Set("hsn_code", d.hsnCode)
		}
		return app.Save(r)
	}

	// ── helper: create a sub item (with optional sub-sub items) ───────
	type subDef struct {
		sortOrder   int
		itemType    string
		description string
		qtyPerUnit  float64
		uom         string
		unitPrice   float64
		gstPercent  int
		hsnCode     string
		subSubs     []subSubDef
	}

	createSubItem := func(mainItemID string, d subDef) (float64, error) {
		var budgeted float64
		if len(d.subSubs) > 0 {
			// budgeted_price = sum of sub-sub budgeted prices
			for _, ss := range d.subSubs {
				budgeted += ss.qtyPerUnit * ss.unitPrice
			}
		} else {
			budgeted = d.qtyPerUnit * d.unitPrice
		}

		r := core.NewRecord(subItemsCol)
		r.Set("main_item", mainItemID)
		r.Set("sort_order", d.sortOrder)
		r.Set("type", d.itemType)
		r.Set("description", d.description)
		r.Set("qty_per_unit", d.qtyPerUnit)
		r.Set("uom", d.uom)
		r.Set("unit_price", d.unitPrice)
		r.Set("budgeted_price", budgeted)
		r.Set("gst_percent", d.gstPercent)
		if d.hsnCode != "" {
			r.Set("hsn_code", d.hsnCode)
		}
		if err := app.Save(r); err != nil {
			return 0, err
		}

		// create sub-sub items
		for _, ss := range d.subSubs {
			if err := createSubSubItem(r.Id, ss); err != nil {
				return 0, fmt.Errorf("seed: save sub_sub_item %q: %w", ss.description, err)
			}
		}

		return budgeted, nil
	}

	// ── helper: create a main item with its sub items ─────────────────
	type mainDef struct {
		sortOrder   int
		description string
		qty         float64
		uom         string
		quotedPrice float64
		gstPercent  int
		subs        []subDef
	}

	createMainItem := func(d mainDef) error {
		// First pass: compute budgeted_price = sum(sub budgeted) * qty
		var perUnitTotal float64
		for _, s := range d.subs {
			if len(s.subSubs) > 0 {
				for _, ss := range s.subSubs {
					perUnitTotal += ss.qtyPerUnit * ss.unitPrice
				}
			} else {
				perUnitTotal += s.qtyPerUnit * s.unitPrice
			}
		}
		budgeted := perUnitTotal * d.qty

		r := core.NewRecord(mainItemsCol)
		r.Set("boq", boqID)
		r.Set("sort_order", d.sortOrder)
		r.Set("description", d.description)
		r.Set("qty", d.qty)
		r.Set("uom", d.uom)
		r.Set("quoted_price", d.quotedPrice)
		r.Set("budgeted_price", budgeted)
		r.Set("gst_percent", d.gstPercent)
		if err := app.Save(r); err != nil {
			return fmt.Errorf("seed: save main_boq_item %q: %w", d.description, err)
		}

		for _, s := range d.subs {
			if _, err := createSubItem(r.Id, s); err != nil {
				return fmt.Errorf("seed: save sub_item %q: %w", s.description, err)
			}
		}
		return nil
	}

	// ── Main Item 1: Wall Work ────────────────────────────────────────
	err = createMainItem(mainDef{
		sortOrder:   1,
		description: "Wall Partition & Finishing Work",
		qty:         450,
		uom:         "sqft",
		quotedPrice: 185.00,
		gstPercent:  18,
		subs: []subDef{
			{
				sortOrder:   1,
				itemType:    "product",
				description: "Gypsum Board 12.5mm",
				qtyPerUnit:  1.1,
				uom:         "sqft",
				unitPrice:   65.00,
				gstPercent:  18,
				hsnCode:     "6809",
			},
			{
				sortOrder:   2,
				itemType:    "product",
				description: "GI Metal Frame & Channel",
				qtyPerUnit:  1.0,
				uom:         "sqft",
				unitPrice:   45.00,
				gstPercent:  18,
				hsnCode:     "7308",
			},
			{
				sortOrder:   3,
				itemType:    "service",
				description: "Installation Labour",
				qtyPerUnit:  1.0,
				uom:         "sqft",
				unitPrice:   35.00,
				gstPercent:  18,
				subSubs: []subSubDef{
					{
						sortOrder:   1,
						itemType:    "service",
						description: "Framing & Fixing",
						qtyPerUnit:  0.6,
						uom:         "sqft",
						unitPrice:   20.00,
						gstPercent:  18,
					},
					{
						sortOrder:   2,
						itemType:    "service",
						description: "Finishing & Putty",
						qtyPerUnit:  0.4,
						uom:         "sqft",
						unitPrice:   15.00,
						gstPercent:  18,
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	// ── Main Item 2: Plumbing ─────────────────────────────────────────
	err = createMainItem(mainDef{
		sortOrder:   2,
		description: "Plumbing & Sanitary Installation",
		qty:         12,
		uom:         "points",
		quotedPrice: 3500.00,
		gstPercent:  18,
		subs: []subDef{
			{
				sortOrder:   1,
				itemType:    "product",
				description: "CPVC Pipe & Fittings",
				qtyPerUnit:  8.0,
				uom:         "meters",
				unitPrice:   120.00,
				gstPercent:  18,
				hsnCode:     "3917",
			},
			{
				sortOrder:   2,
				itemType:    "product",
				description: "Sanitary Fixtures",
				qtyPerUnit:  1.0,
				uom:         "nos",
				unitPrice:   1800.00,
				gstPercent:  18,
				hsnCode:     "6910",
			},
			{
				sortOrder:   3,
				itemType:    "service",
				description: "Plumbing Labour",
				qtyPerUnit:  1.0,
				uom:         "points",
				unitPrice:   650.00,
				gstPercent:  18,
			},
		},
	})
	if err != nil {
		return err
	}

	// ── Main Item 3: Electrical ───────────────────────────────────────
	err = createMainItem(mainDef{
		sortOrder:   3,
		description: "Electrical Wiring & Fixtures",
		qty:         25,
		uom:         "points",
		quotedPrice: 2200.00,
		gstPercent:  18,
		subs: []subDef{
			{
				sortOrder:   1,
				itemType:    "product",
				description: "Copper Wiring 2.5mm",
				qtyPerUnit:  12.0,
				uom:         "meters",
				unitPrice:   28.00,
				gstPercent:  18,
				hsnCode:     "7413",
			},
			{
				sortOrder:   2,
				itemType:    "product",
				description: "Switches & Sockets",
				qtyPerUnit:  2.0,
				uom:         "nos",
				unitPrice:   180.00,
				gstPercent:  18,
				hsnCode:     "8536",
			},
			{
				sortOrder:   3,
				itemType:    "service",
				description: "Electrical Labour",
				qtyPerUnit:  1.0,
				uom:         "points",
				unitPrice:   450.00,
				gstPercent:  18,
				subSubs: []subSubDef{
					{
						sortOrder:   1,
						itemType:    "service",
						description: "Conduit & Wiring Work",
						qtyPerUnit:  1.0,
						uom:         "points",
						unitPrice:   450.00,
						gstPercent:  18,
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	log.Println("seed: BOQ seed data inserted successfully")
	return nil
}
