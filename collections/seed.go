package collections

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// ── Definition structs ───────────────────────────────────────────────────

type subSubDef struct {
	sortOrder   int
	itemType    string
	description string
	qtyPerUnit  float64
	uom         string
	unitPrice   float64
	gstPercent  int
	hsnCode     string
}

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

type mainDef struct {
	sortOrder   int
	description string
	qty         float64
	uom         string
	quotedPrice float64
	gstPercent  int
	subs        []subDef
}

type addressDef struct {
	addressType    string
	companyName    string
	contactPerson  string
	addressLine1   string
	addressLine2   string
	city           string
	state          string
	pinCode        string
	country        string
	district       string
	phone          string
	email          string
	gstin          string
	pan            string
	shipToParentID string // set after creation for install_at linking
}

type vendorDef struct {
	name              string
	addressLine1      string
	city              string
	state             string
	pinCode           string
	country           string
	gstin             string
	pan               string
	contactName       string
	phone             string
	email             string
	website           string
	bankBeneficiary   string
	bankName          string
	bankAccountNo     string
	bankIFSC          string
	bankBranch        string
}

type poLineItemDef struct {
	sortOrder      int
	description    string
	hsnCode        string
	qty            float64
	uom            string
	rate           float64
	gstPercent     int
	sourceItemType string
}

type purchaseOrderDef struct {
	poNumber      string
	orderDate     string
	quotationRef  string
	paymentTerms  string
	deliveryTerms string
	warrantyTerms string
	status        string
	lineItems     []poLineItemDef
}

// Seed populates all collections with realistic OAVS Smart Classrooms
// and Smart Labs data. It is safe to call on every startup because it
// returns early if any project records already exist.
func Seed(app *pocketbase.PocketBase) error {
	// ── idempotency: skip if projects already exist ──────────────────
	projectsCol, err := app.FindCollectionByNameOrId("projects")
	if err != nil {
		return fmt.Errorf("seed: could not find projects collection: %w", err)
	}
	existing, err := app.FindAllRecords(projectsCol)
	if err != nil {
		return fmt.Errorf("seed: could not query projects: %w", err)
	}
	if len(existing) > 0 {
		return nil // already seeded
	}

	log.Println("seed: projects collection is empty – inserting seed data …")

	// ── lookup helper collections ────────────────────────────────────
	boqsCol, err := app.FindCollectionByNameOrId("boqs")
	if err != nil {
		return fmt.Errorf("seed: could not find boqs collection: %w", err)
	}
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
	addressesCol, err := app.FindCollectionByNameOrId("addresses")
	if err != nil {
		return fmt.Errorf("seed: could not find addresses collection: %w", err)
	}
	settingsCol, err := app.FindCollectionByNameOrId("project_address_settings")
	if err != nil {
		return fmt.Errorf("seed: could not find project_address_settings collection: %w", err)
	}
	vendorsCol, err := app.FindCollectionByNameOrId("vendors")
	if err != nil {
		return fmt.Errorf("seed: could not find vendors collection: %w", err)
	}
	projectVendorsCol, err := app.FindCollectionByNameOrId("project_vendors")
	if err != nil {
		return fmt.Errorf("seed: could not find project_vendors collection: %w", err)
	}
	poCol, err := app.FindCollectionByNameOrId("purchase_orders")
	if err != nil {
		return fmt.Errorf("seed: could not find purchase_orders collection: %w", err)
	}
	poLineItemsCol, err := app.FindCollectionByNameOrId("po_line_items")
	if err != nil {
		return fmt.Errorf("seed: could not find po_line_items collection: %w", err)
	}

	// ── helper: create sub-sub item ──────────────────────────────────
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

	// ── helper: create sub item ──────────────────────────────────────
	createSubItem := func(mainItemID string, d subDef) (float64, error) {
		var budgeted float64
		if len(d.subSubs) > 0 {
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

		for _, ss := range d.subSubs {
			if err := createSubSubItem(r.Id, ss); err != nil {
				return 0, fmt.Errorf("seed: save sub_sub_item %q: %w", ss.description, err)
			}
		}
		return budgeted, nil
	}

	// ── helper: create main item (accepts boqID as parameter) ────────
	createMainItem := func(boqID string, d mainDef) error {
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
		r.Set("unit_price", perUnitTotal)
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

	// ── helper: create address ───────────────────────────────────────
	createAddress := func(projectID string, d addressDef) (*core.Record, error) {
		r := core.NewRecord(addressesCol)
		r.Set("project", projectID)
		r.Set("address_type", d.addressType)
		r.Set("company_name", d.companyName)
		r.Set("contact_person", d.contactPerson)
		r.Set("address_line_1", d.addressLine1)
		r.Set("address_line_2", d.addressLine2)
		r.Set("city", d.city)
		r.Set("state", d.state)
		r.Set("pin_code", d.pinCode)
		r.Set("country", d.country)
		r.Set("district", d.district)
		r.Set("phone", d.phone)
		r.Set("email", d.email)
		r.Set("gstin", d.gstin)
		r.Set("pan", d.pan)
		if d.shipToParentID != "" {
			r.Set("ship_to_parent", d.shipToParentID)
		}
		if err := app.Save(r); err != nil {
			return nil, fmt.Errorf("seed: save address %q (%s): %w", d.companyName, d.addressType, err)
		}
		return r, nil
	}

	// ── helper: create address settings ──────────────────────────────
	createAddressSettings := func(projectID, addressType string) error {
		r := core.NewRecord(settingsCol)
		r.Set("project", projectID)
		r.Set("address_type", addressType)
		r.Set("req_company_name", true)
		r.Set("req_address_line_1", true)
		r.Set("req_city", true)
		r.Set("req_state", true)
		r.Set("req_pin_code", true)
		r.Set("req_gstin", true)
		return app.Save(r)
	}

	// ── helper: create vendor ────────────────────────────────────────
	createVendor := func(d vendorDef) (*core.Record, error) {
		r := core.NewRecord(vendorsCol)
		r.Set("name", d.name)
		r.Set("address_line_1", d.addressLine1)
		r.Set("city", d.city)
		r.Set("state", d.state)
		r.Set("pin_code", d.pinCode)
		r.Set("country", d.country)
		r.Set("gstin", d.gstin)
		r.Set("pan", d.pan)
		r.Set("contact_name", d.contactName)
		r.Set("phone", d.phone)
		r.Set("email", d.email)
		r.Set("website", d.website)
		r.Set("bank_beneficiary_name", d.bankBeneficiary)
		r.Set("bank_name", d.bankName)
		r.Set("bank_account_no", d.bankAccountNo)
		r.Set("bank_ifsc", d.bankIFSC)
		r.Set("bank_branch", d.bankBranch)
		if err := app.Save(r); err != nil {
			return nil, fmt.Errorf("seed: save vendor %q: %w", d.name, err)
		}
		return r, nil
	}

	// ── helper: link vendor to project ───────────────────────────────
	linkVendorToProject := func(projectID, vendorID string) error {
		r := core.NewRecord(projectVendorsCol)
		r.Set("project", projectID)
		r.Set("vendor", vendorID)
		return app.Save(r)
	}

	// ── helper: create purchase order with line items ────────────────
	createPO := func(projectID, vendorID, billToID, shipToID string, d purchaseOrderDef) error {
		r := core.NewRecord(poCol)
		r.Set("project", projectID)
		r.Set("vendor", vendorID)
		r.Set("po_number", d.poNumber)
		r.Set("order_date", d.orderDate)
		r.Set("quotation_ref", d.quotationRef)
		r.Set("payment_terms", d.paymentTerms)
		r.Set("delivery_terms", d.deliveryTerms)
		r.Set("warranty_terms", d.warrantyTerms)
		r.Set("status", d.status)
		if billToID != "" {
			r.Set("bill_to_address", billToID)
		}
		if shipToID != "" {
			r.Set("ship_to_address", shipToID)
		}
		if err := app.Save(r); err != nil {
			return fmt.Errorf("seed: save PO %q: %w", d.poNumber, err)
		}

		for _, li := range d.lineItems {
			lr := core.NewRecord(poLineItemsCol)
			lr.Set("purchase_order", r.Id)
			lr.Set("sort_order", li.sortOrder)
			lr.Set("description", li.description)
			lr.Set("hsn_code", li.hsnCode)
			lr.Set("qty", li.qty)
			lr.Set("uom", li.uom)
			lr.Set("rate", li.rate)
			lr.Set("gst_percent", li.gstPercent)
			if li.sourceItemType != "" {
				lr.Set("source_item_type", li.sourceItemType)
			}
			if err := app.Save(lr); err != nil {
				return fmt.Errorf("seed: save PO line item %q: %w", li.description, err)
			}
		}
		return nil
	}

	// ══════════════════════════════════════════════════════════════════
	// PROJECT 1: OAVS Smart Classrooms
	// ══════════════════════════════════════════════════════════════════

	p1 := core.NewRecord(projectsCol)
	p1.Set("name", "OAVS Smart Classroom — Phase I (Koraput Division)")
	p1.Set("client_name", "Odisha Adarsha Vidyalaya Sangathan (OAVS)")
	p1.Set("reference_number", "OAVS/SC/2025-26/001")
	p1.Set("status", "active")
	p1.Set("ship_to_equals_install_at", false)
	if err := app.Save(p1); err != nil {
		return fmt.Errorf("seed: save project 1: %w", err)
	}

	// ── BOQ: Smart Classroom Package ─────────────────────────────────
	boq1 := core.NewRecord(boqsCol)
	boq1.Set("title", "Smart Classroom Package — 100 Classrooms")
	boq1.Set("reference_number", "OAVS/SC/BOQ/001")
	boq1.Set("project", p1.Id)
	if err := app.Save(boq1); err != nil {
		return fmt.Errorf("seed: save boq1: %w", err)
	}

	// Main Item 1: Smart Classroom Hardware
	if err := createMainItem(boq1.Id, mainDef{
		sortOrder: 1, description: "Smart Classroom Hardware", qty: 100, uom: "Nos", quotedPrice: 285000, gstPercent: 18,
		subs: []subDef{
			{sortOrder: 1, itemType: "product", description: "75\" Interactive Flat Panel (4K)", qtyPerUnit: 1, uom: "Nos", unitPrice: 145000, gstPercent: 18, hsnCode: "8528"},
			{sortOrder: 2, itemType: "product", description: "OPS Module — Intel i5, 8GB, 256GB SSD", qtyPerUnit: 1, uom: "Nos", unitPrice: 42000, gstPercent: 18, hsnCode: "8471"},
			{sortOrder: 3, itemType: "product", description: "1KVA Online UPS with Battery", qtyPerUnit: 1, uom: "Nos", unitPrice: 18000, gstPercent: 18, hsnCode: "8504"},
			{sortOrder: 4, itemType: "product", description: "Wall Mount Bracket (Heavy Duty)", qtyPerUnit: 1, uom: "Nos", unitPrice: 3500, gstPercent: 18, hsnCode: "7326"},
			{sortOrder: 5, itemType: "product", description: "PTZ Camera 1080p USB", qtyPerUnit: 1, uom: "Nos", unitPrice: 12000, gstPercent: 18, hsnCode: "8525"},
			{sortOrder: 6, itemType: "product", description: "Wireless Microphone Set (Lapel + Handheld)", qtyPerUnit: 1, uom: "Set", unitPrice: 8500, gstPercent: 18, hsnCode: "8518"},
			{sortOrder: 7, itemType: "product", description: "Wall-Mount Speakers (Pair)", qtyPerUnit: 1, uom: "Pair", unitPrice: 4500, gstPercent: 18, hsnCode: "8518"},
			{sortOrder: 8, itemType: "product", description: "Document Camera 8MP", qtyPerUnit: 1, uom: "Nos", unitPrice: 15000, gstPercent: 18, hsnCode: "8525"},
		},
	}); err != nil {
		return err
	}

	// Main Item 2: Digital Podium
	if err := createMainItem(boq1.Id, mainDef{
		sortOrder: 2, description: "Digital Podium", qty: 100, uom: "Nos", quotedPrice: 65000, gstPercent: 18,
		subs: []subDef{
			{
				sortOrder: 1, itemType: "product", description: "Digital Podium Unit (Steel, Powder Coated)", qtyPerUnit: 1, uom: "Nos", unitPrice: 55000, gstPercent: 18, hsnCode: "9403",
				subSubs: []subSubDef{
					{sortOrder: 1, itemType: "product", description: "Frame & Cabinet Assembly", qtyPerUnit: 1, uom: "Nos", unitPrice: 35000, gstPercent: 18, hsnCode: "9403"},
					{sortOrder: 2, itemType: "product", description: "Gooseneck Microphone (Integrated)", qtyPerUnit: 1, uom: "Nos", unitPrice: 4500, gstPercent: 18, hsnCode: "8518"},
					{sortOrder: 3, itemType: "service", description: "Internal Wiring & Assembly", qtyPerUnit: 1, uom: "Nos", unitPrice: 2500, gstPercent: 18, hsnCode: "8544"},
				},
			},
		},
	}); err != nil {
		return err
	}

	// Main Item 3: Networking & Cabling
	if err := createMainItem(boq1.Id, mainDef{
		sortOrder: 3, description: "Networking & Cabling", qty: 100, uom: "Lot", quotedPrice: 45000, gstPercent: 18,
		subs: []subDef{
			{sortOrder: 1, itemType: "product", description: "8-Port PoE+ Managed Switch", qtyPerUnit: 1, uom: "Nos", unitPrice: 12000, gstPercent: 18, hsnCode: "8517"},
			{sortOrder: 2, itemType: "product", description: "WiFi 6 Access Point (Ceiling Mount)", qtyPerUnit: 1, uom: "Nos", unitPrice: 8500, gstPercent: 18, hsnCode: "8517"},
			{sortOrder: 3, itemType: "product", description: "CAT6 Cabling with Termination", qtyPerUnit: 30, uom: "Mtrs", unitPrice: 85, gstPercent: 18, hsnCode: "8544"},
			{sortOrder: 4, itemType: "product", description: "HDMI 4K Cable (10m)", qtyPerUnit: 2, uom: "Nos", unitPrice: 1200, gstPercent: 18, hsnCode: "8544"},
			{sortOrder: 5, itemType: "product", description: "Power Cabling & Conduit", qtyPerUnit: 20, uom: "Mtrs", unitPrice: 120, gstPercent: 18, hsnCode: "8544"},
			{sortOrder: 6, itemType: "product", description: "Cable Tray (Perforated, GI)", qtyPerUnit: 10, uom: "Mtrs", unitPrice: 350, gstPercent: 18, hsnCode: "7610"},
		},
	}); err != nil {
		return err
	}

	// Main Item 4: Furniture & Fixtures
	if err := createMainItem(boq1.Id, mainDef{
		sortOrder: 4, description: "Furniture & Fixtures", qty: 100, uom: "Nos", quotedPrice: 28000, gstPercent: 18,
		subs: []subDef{
			{sortOrder: 1, itemType: "product", description: "Back Panel (Acoustic, 8x4 ft)", qtyPerUnit: 1, uom: "Nos", unitPrice: 12000, gstPercent: 18, hsnCode: "9403"},
			{sortOrder: 2, itemType: "product", description: "Teacher Table & Chair Set", qtyPerUnit: 1, uom: "Set", unitPrice: 15000, gstPercent: 18, hsnCode: "9403"},
		},
	}); err != nil {
		return err
	}

	// Main Item 5: Installation & Commissioning
	if err := createMainItem(boq1.Id, mainDef{
		sortOrder: 5, description: "Installation & Commissioning", qty: 100, uom: "Lot", quotedPrice: 18000, gstPercent: 18,
		subs: []subDef{
			{
				sortOrder: 1, itemType: "service", description: "Installation per Classroom", qtyPerUnit: 1, uom: "Lot", unitPrice: 15000, gstPercent: 18, hsnCode: "9987",
				subSubs: []subSubDef{
					{sortOrder: 1, itemType: "service", description: "Equipment Mounting & Fixing", qtyPerUnit: 1, uom: "Lot", unitPrice: 6000, gstPercent: 18, hsnCode: "9987"},
					{sortOrder: 2, itemType: "service", description: "Network Configuration & Testing", qtyPerUnit: 1, uom: "Lot", unitPrice: 4000, gstPercent: 18, hsnCode: "9987"},
					{sortOrder: 3, itemType: "service", description: "Final Integration Testing & Handover", qtyPerUnit: 1, uom: "Lot", unitPrice: 5000, gstPercent: 18, hsnCode: "9987"},
				},
			},
			{sortOrder: 2, itemType: "service", description: "Site Survey & Assessment", qtyPerUnit: 1, uom: "Lot", unitPrice: 3000, gstPercent: 18, hsnCode: "9987"},
		},
	}); err != nil {
		return err
	}

	// Main Item 6: Training & Support
	if err := createMainItem(boq1.Id, mainDef{
		sortOrder: 6, description: "Training & Support", qty: 50, uom: "Schools", quotedPrice: 35000, gstPercent: 18,
		subs: []subDef{
			{sortOrder: 1, itemType: "service", description: "Teacher Training — 2 Day Workshop", qtyPerUnit: 1, uom: "Batch", unitPrice: 25000, gstPercent: 18, hsnCode: "9992"},
			{sortOrder: 2, itemType: "service", description: "Digital Content License (1 Year)", qtyPerUnit: 1, uom: "License", unitPrice: 8000, gstPercent: 18, hsnCode: "9973"},
		},
	}); err != nil {
		return err
	}

	// ── Project 1 Addresses ──────────────────────────────────────────
	_, err = createAddress(p1.Id, addressDef{
		addressType: "bill_from", companyName: "SmartEd Solutions Pvt. Ltd.",
		contactPerson: "Rajesh Kumar", addressLine1: "Plot 42, Saheed Nagar",
		city: "Bhubaneswar", state: "Odisha", pinCode: "751007", country: "India",
		phone: "0674-2546789", email: "accounts@smartedsolutions.in",
		gstin: "21AABCS1234F1Z5", pan: "AABCS1234F",
	})
	if err != nil {
		return err
	}

	p1BillTo, err := createAddress(p1.Id, addressDef{
		addressType: "bill_to", companyName: "Odisha Adarsha Vidyalaya Sangathan (OAVS)",
		contactPerson: "Dr. Subash Patra", addressLine1: "N1/9, Sainik School Road, Nayapalli",
		city: "Bhubaneswar", state: "Odisha", pinCode: "751005", country: "India",
		phone: "0674-2390500", email: "director@oavs.edu.in",
		gstin: "21AABCO5678G1Z5", pan: "AABCO5678G",
	})
	if err != nil {
		return err
	}

	p1ShipTo1, err := createAddress(p1.Id, addressDef{
		addressType: "ship_to", companyName: "OAV Koraput",
		contactPerson: "Sri Mohan Rao (Principal)", addressLine1: "Jeypore Road, Near Collectorate",
		city: "Koraput", state: "Odisha", pinCode: "764020", country: "India",
		district: "Koraput", phone: "06852-251234",
	})
	if err != nil {
		return err
	}

	p1ShipTo2, err := createAddress(p1.Id, addressDef{
		addressType: "ship_to", companyName: "OAV Rayagada",
		contactPerson: "Smt. Priya Mishra (Principal)", addressLine1: "Station Road, Ward No. 12",
		city: "Rayagada", state: "Odisha", pinCode: "765001", country: "India",
		district: "Rayagada", phone: "06856-222345",
	})
	if err != nil {
		return err
	}

	if _, err := createAddress(p1.Id, addressDef{
		addressType: "install_at", companyName: "Smart Classroom Block, OAV Koraput",
		contactPerson: "Sri Mohan Rao (Principal)", addressLine1: "Jeypore Road, Near Collectorate",
		city: "Koraput", state: "Odisha", pinCode: "764020", country: "India",
		district: "Koraput", shipToParentID: p1ShipTo1.Id,
	}); err != nil {
		return err
	}

	if _, err := createAddress(p1.Id, addressDef{
		addressType: "install_at", companyName: "Smart Classroom Block, OAV Rayagada",
		contactPerson: "Smt. Priya Mishra (Principal)", addressLine1: "Station Road, Ward No. 12",
		city: "Rayagada", state: "Odisha", pinCode: "765001", country: "India",
		district: "Rayagada", shipToParentID: p1ShipTo2.Id,
	}); err != nil {
		return err
	}

	// ── Project 1 Address Settings ───────────────────────────────────
	if err := createAddressSettings(p1.Id, "bill_from"); err != nil {
		return fmt.Errorf("seed: address settings bill_from p1: %w", err)
	}
	if err := createAddressSettings(p1.Id, "bill_to"); err != nil {
		return fmt.Errorf("seed: address settings bill_to p1: %w", err)
	}

	// ── Project 1 Vendors ────────────────────────────────────────────
	v1, err := createVendor(vendorDef{
		name: "ViewSonic India Pvt. Ltd.", addressLine1: "Unit 301, Bandra Kurla Complex",
		city: "Mumbai", state: "Maharashtra", pinCode: "400051", country: "India",
		gstin: "27AABCV1234F1Z5", pan: "AABCV1234F",
		contactName: "Amit Shah", phone: "022-40123456", email: "govt.sales@viewsonic.co.in",
		website: "https://www.viewsonic.com/in/",
		bankBeneficiary: "ViewSonic India Pvt. Ltd.", bankName: "HDFC Bank",
		bankAccountNo: "50200045678901", bankIFSC: "HDFC0000123", bankBranch: "BKC, Mumbai",
	})
	if err != nil {
		return err
	}

	v2, err := createVendor(vendorDef{
		name: "Ahuja Radios", addressLine1: "286, Okhla Industrial Estate, Phase-III",
		city: "New Delhi", state: "Delhi", pinCode: "110020", country: "India",
		gstin: "07AAACA5765F1Z5", pan: "AAACA5765F",
		contactName: "Vikram Ahuja", phone: "011-40567890", email: "sales@ahujaradios.com",
		website: "https://www.ahujaradios.com",
		bankBeneficiary: "Ahuja Radios", bankName: "State Bank of India",
		bankAccountNo: "30987654321", bankIFSC: "SBIN0001234", bankBranch: "Okhla, New Delhi",
	})
	if err != nil {
		return err
	}

	v3, err := createVendor(vendorDef{
		name: "DigiNet Infra Solutions", addressLine1: "Plot 18, Chandrasekharpur",
		city: "Bhubaneswar", state: "Odisha", pinCode: "751016", country: "India",
		gstin: "21AABCD1234E1Z5", pan: "AABCD1234E",
		contactName: "Sanjay Mohapatra", phone: "0674-2745678", email: "info@diginetinfra.in",
		bankBeneficiary: "DigiNet Infra Solutions", bankName: "ICICI Bank",
		bankAccountNo: "123456789012", bankIFSC: "ICIC0000456", bankBranch: "Chandrasekharpur, Bhubaneswar",
	})
	if err != nil {
		return err
	}

	// Link vendors to project 1
	for _, vid := range []string{v1.Id, v2.Id, v3.Id} {
		if err := linkVendorToProject(p1.Id, vid); err != nil {
			return fmt.Errorf("seed: link vendor to project 1: %w", err)
		}
	}

	// ── Project 1 Purchase Orders ────────────────────────────────────
	if err := createPO(p1.Id, v1.Id, p1BillTo.Id, p1ShipTo1.Id, purchaseOrderDef{
		poNumber: "PO-FY26-001", orderDate: "2025-07-15", quotationRef: "VS/Q/2025/1089",
		paymentTerms: "30% advance, 60% on delivery, 10% post-installation",
		deliveryTerms: "Ex-works Mumbai; freight to site extra",
		warrantyTerms: "3 years onsite comprehensive warranty",
		status: "sent",
		lineItems: []poLineItemDef{
			{sortOrder: 1, description: "75\" Interactive Flat Panel (4K) — ViewSonic IFP7550", hsnCode: "8528", qty: 100, uom: "Nos", rate: 142000, gstPercent: 18, sourceItemType: "sub_item"},
			{sortOrder: 2, description: "OPS Module i5/8GB/256GB — ViewSonic NMP-712", hsnCode: "8471", qty: 100, uom: "Nos", rate: 41000, gstPercent: 18, sourceItemType: "sub_item"},
			{sortOrder: 3, description: "PTZ Camera 1080p USB — ViewSonic VB-CAM-002", hsnCode: "8525", qty: 100, uom: "Nos", rate: 11500, gstPercent: 18, sourceItemType: "sub_item"},
		},
	}); err != nil {
		return err
	}

	if err := createPO(p1.Id, v3.Id, p1BillTo.Id, p1ShipTo1.Id, purchaseOrderDef{
		poNumber: "PO-FY26-002", orderDate: "2025-08-01",
		paymentTerms: "50% advance, 50% on completion",
		deliveryTerms: "Delivery & installation at site",
		status: "draft",
		lineItems: []poLineItemDef{
			{sortOrder: 1, description: "CAT6 Cabling with Termination — 100 classrooms", hsnCode: "8544", qty: 3000, uom: "Mtrs", rate: 82, gstPercent: 18, sourceItemType: "sub_item"},
			{sortOrder: 2, description: "8-Port PoE+ Managed Switch", hsnCode: "8517", qty: 100, uom: "Nos", rate: 11500, gstPercent: 18, sourceItemType: "sub_item"},
			{sortOrder: 3, description: "Installation per Classroom (complete)", hsnCode: "9987", qty: 100, uom: "Lot", rate: 14500, gstPercent: 18, sourceItemType: "sub_item"},
		},
	}); err != nil {
		return err
	}

	// ══════════════════════════════════════════════════════════════════
	// PROJECT 2: Smart Labs — Model School, Bangalore
	// ══════════════════════════════════════════════════════════════════

	p2 := core.NewRecord(projectsCol)
	p2.Set("name", "Smart Labs — Model School, Bangalore")
	p2.Set("client_name", "Karnataka State Education Department")
	p2.Set("reference_number", "KSED/SL/2025-26/047")
	p2.Set("status", "active")
	p2.Set("ship_to_equals_install_at", true)
	if err := app.Save(p2); err != nil {
		return fmt.Errorf("seed: save project 2: %w", err)
	}

	// ── BOQ 2a: Computer Lab — 30 Stations ───────────────────────────
	boq2a := core.NewRecord(boqsCol)
	boq2a.Set("title", "Computer Lab — 30 Stations")
	boq2a.Set("reference_number", "KSED/SL/BOQ/047-A")
	boq2a.Set("project", p2.Id)
	if err := app.Save(boq2a); err != nil {
		return fmt.Errorf("seed: save boq2a: %w", err)
	}

	// Computer Lab Main Item 1: Computing Hardware
	if err := createMainItem(boq2a.Id, mainDef{
		sortOrder: 1, description: "Computing Hardware", qty: 1, uom: "Lab", quotedPrice: 1850000, gstPercent: 18,
		subs: []subDef{
			{sortOrder: 1, itemType: "product", description: "Student Desktop (i3/8GB/512GB SSD/21.5\" Monitor)", qtyPerUnit: 30, uom: "Nos", unitPrice: 38000, gstPercent: 18, hsnCode: "8471"},
			{sortOrder: 2, itemType: "product", description: "Teacher Desktop (i5/16GB/512GB SSD/23.8\" Monitor)", qtyPerUnit: 1, uom: "Nos", unitPrice: 55000, gstPercent: 18, hsnCode: "8471"},
			{sortOrder: 3, itemType: "product", description: "Lab Server (Xeon/32GB/2TB RAID/Rack Mount)", qtyPerUnit: 1, uom: "Nos", unitPrice: 185000, gstPercent: 18, hsnCode: "8471"},
			{sortOrder: 4, itemType: "product", description: "Laser Printer (A4, Network, Duplex)", qtyPerUnit: 1, uom: "Nos", unitPrice: 28000, gstPercent: 18, hsnCode: "8443"},
		},
	}); err != nil {
		return err
	}

	// Computer Lab Main Item 2: Networking & Power
	if err := createMainItem(boq2a.Id, mainDef{
		sortOrder: 2, description: "Networking & Power", qty: 1, uom: "Lab", quotedPrice: 520000, gstPercent: 18,
		subs: []subDef{
			{sortOrder: 1, itemType: "product", description: "24-Port Gigabit PoE Managed Switch", qtyPerUnit: 2, uom: "Nos", unitPrice: 35000, gstPercent: 18, hsnCode: "8517"},
			{sortOrder: 2, itemType: "product", description: "WiFi 6 Access Point (Ceiling)", qtyPerUnit: 2, uom: "Nos", unitPrice: 8500, gstPercent: 18, hsnCode: "8517"},
			{sortOrder: 3, itemType: "product", description: "Server Rack 12U with Accessories", qtyPerUnit: 1, uom: "Nos", unitPrice: 18000, gstPercent: 18, hsnCode: "8517"},
			{sortOrder: 4, itemType: "product", description: "5KVA Online UPS (for Server + Network)", qtyPerUnit: 1, uom: "Nos", unitPrice: 85000, gstPercent: 18, hsnCode: "8504"},
			{sortOrder: 5, itemType: "product", description: "Desktop UPS 600VA", qtyPerUnit: 30, uom: "Nos", unitPrice: 3500, gstPercent: 18, hsnCode: "8504"},
			{sortOrder: 6, itemType: "product", description: "LAN Cabling (CAT6) with Termination", qtyPerUnit: 32, uom: "Points", unitPrice: 2500, gstPercent: 18, hsnCode: "8544"},
			{sortOrder: 7, itemType: "product", description: "Electrical Points with Conduit", qtyPerUnit: 32, uom: "Points", unitPrice: 1800, gstPercent: 18, hsnCode: "8536"},
		},
	}); err != nil {
		return err
	}

	// Computer Lab Main Item 3: Display & Presentation
	if err := createMainItem(boq2a.Id, mainDef{
		sortOrder: 3, description: "Display & Presentation", qty: 1, uom: "Lab", quotedPrice: 150000, gstPercent: 18,
		subs: []subDef{
			{sortOrder: 1, itemType: "product", description: "75\" Interactive Flat Panel (4K)", qtyPerUnit: 1, uom: "Nos", unitPrice: 145000, gstPercent: 18, hsnCode: "8528"},
		},
	}); err != nil {
		return err
	}

	// Computer Lab Main Item 4: Furniture
	if err := createMainItem(boq2a.Id, mainDef{
		sortOrder: 4, description: "Furniture", qty: 1, uom: "Lab", quotedPrice: 380000, gstPercent: 18,
		subs: []subDef{
			{sortOrder: 1, itemType: "product", description: "Student Computer Table (with keyboard tray)", qtyPerUnit: 30, uom: "Nos", unitPrice: 6500, gstPercent: 18, hsnCode: "9403"},
			{sortOrder: 2, itemType: "product", description: "Student Revolving Chair", qtyPerUnit: 30, uom: "Nos", unitPrice: 3800, gstPercent: 18, hsnCode: "9401"},
			{sortOrder: 3, itemType: "product", description: "Teacher Table (L-shaped, with drawers)", qtyPerUnit: 1, uom: "Nos", unitPrice: 12000, gstPercent: 18, hsnCode: "9403"},
			{sortOrder: 4, itemType: "product", description: "Teacher Executive Chair", qtyPerUnit: 1, uom: "Nos", unitPrice: 8500, gstPercent: 18, hsnCode: "9401"},
		},
	}); err != nil {
		return err
	}

	// Computer Lab Main Item 5: Software Licenses
	if err := createMainItem(boq2a.Id, mainDef{
		sortOrder: 5, description: "Software Licenses", qty: 1, uom: "Lab", quotedPrice: 350000, gstPercent: 18,
		subs: []subDef{
			{sortOrder: 1, itemType: "product", description: "Windows 11 Pro + MS Office 2024 License", qtyPerUnit: 31, uom: "Nos", unitPrice: 12000, gstPercent: 18, hsnCode: "8523"},
		},
	}); err != nil {
		return err
	}

	// Computer Lab Main Item 6: Installation & Services
	if err := createMainItem(boq2a.Id, mainDef{
		sortOrder: 6, description: "Installation & Services", qty: 1, uom: "Lab", quotedPrice: 95000, gstPercent: 18,
		subs: []subDef{
			{sortOrder: 1, itemType: "service", description: "Site Survey & Assessment", qtyPerUnit: 1, uom: "Lot", unitPrice: 8000, gstPercent: 18, hsnCode: "9983"},
			{
				sortOrder: 2, itemType: "service", description: "Installation & Commissioning", qtyPerUnit: 1, uom: "Lot", unitPrice: 65000, gstPercent: 18, hsnCode: "9983",
				subSubs: []subSubDef{
					{sortOrder: 1, itemType: "service", description: "Hardware Setup & Mounting", qtyPerUnit: 1, uom: "Lot", unitPrice: 25000, gstPercent: 18, hsnCode: "9983"},
					{sortOrder: 2, itemType: "service", description: "Network Configuration & Testing", qtyPerUnit: 1, uom: "Lot", unitPrice: 20000, gstPercent: 18, hsnCode: "9983"},
					{sortOrder: 3, itemType: "service", description: "Software Installation & Imaging", qtyPerUnit: 1, uom: "Lot", unitPrice: 15000, gstPercent: 18, hsnCode: "9983"},
				},
			},
			{sortOrder: 3, itemType: "service", description: "Training — 3 Day Teacher Workshop", qtyPerUnit: 1, uom: "Batch", unitPrice: 20000, gstPercent: 18, hsnCode: "9992"},
		},
	}); err != nil {
		return err
	}

	// ── BOQ 2b: STEM / Robotics Lab — 20 Stations ───────────────────
	boq2b := core.NewRecord(boqsCol)
	boq2b.Set("title", "STEM / Robotics Lab — 20 Stations")
	boq2b.Set("reference_number", "KSED/SL/BOQ/047-B")
	boq2b.Set("project", p2.Id)
	if err := app.Save(boq2b); err != nil {
		return fmt.Errorf("seed: save boq2b: %w", err)
	}

	// STEM Lab Main Item 1: Robotics & IoT Kits
	if err := createMainItem(boq2b.Id, mainDef{
		sortOrder: 1, description: "Robotics & IoT Kits", qty: 1, uom: "Lab", quotedPrice: 850000, gstPercent: 18,
		subs: []subDef{
			{sortOrder: 1, itemType: "product", description: "Arduino Mega Starter Kit", qtyPerUnit: 20, uom: "Nos", unitPrice: 4500, gstPercent: 18, hsnCode: "8542"},
			{sortOrder: 2, itemType: "product", description: "Raspberry Pi 4 Kit (4GB, Case, SD, PSU)", qtyPerUnit: 10, uom: "Nos", unitPrice: 8500, gstPercent: 18, hsnCode: "8471"},
			{sortOrder: 3, itemType: "product", description: "Basic Robotics Kit (2WD Chassis, Motors, Sensors)", qtyPerUnit: 15, uom: "Nos", unitPrice: 12000, gstPercent: 18, hsnCode: "8479"},
			{sortOrder: 4, itemType: "product", description: "Advanced Robotics Kit (6DOF Arm, Gripper)", qtyPerUnit: 5, uom: "Nos", unitPrice: 45000, gstPercent: 18, hsnCode: "8479"},
			{sortOrder: 5, itemType: "product", description: "IoT Sensor & Actuator Kit (Temp, Humidity, Relay)", qtyPerUnit: 15, uom: "Nos", unitPrice: 3500, gstPercent: 18, hsnCode: "8542"},
		},
	}); err != nil {
		return err
	}

	// STEM Lab Main Item 2: 3D Printing Equipment
	if err := createMainItem(boq2b.Id, mainDef{
		sortOrder: 2, description: "3D Printing Equipment", qty: 1, uom: "Lab", quotedPrice: 200000, gstPercent: 18,
		subs: []subDef{
			{sortOrder: 1, itemType: "product", description: "FDM 3D Printer (220x220x250mm build)", qtyPerUnit: 2, uom: "Nos", unitPrice: 65000, gstPercent: 18, hsnCode: "8477"},
			{sortOrder: 2, itemType: "product", description: "PLA Filament 1.75mm (1kg spools, assorted)", qtyPerUnit: 10, uom: "Nos", unitPrice: 1800, gstPercent: 18, hsnCode: "3916"},
			{sortOrder: 3, itemType: "product", description: "3D Printing Tools & Accessories Kit", qtyPerUnit: 2, uom: "Set", unitPrice: 5000, gstPercent: 18, hsnCode: "8205"},
		},
	}); err != nil {
		return err
	}

	// STEM Lab Main Item 3: Electronics & Test Equipment
	if err := createMainItem(boq2b.Id, mainDef{
		sortOrder: 3, description: "Electronics & Test Equipment", qty: 1, uom: "Lab", quotedPrice: 280000, gstPercent: 18,
		subs: []subDef{
			{sortOrder: 1, itemType: "product", description: "Temperature-Controlled Soldering Station", qtyPerUnit: 10, uom: "Nos", unitPrice: 4500, gstPercent: 18, hsnCode: "8515"},
			{sortOrder: 2, itemType: "product", description: "Digital Multimeter (True RMS)", qtyPerUnit: 15, uom: "Nos", unitPrice: 2800, gstPercent: 18, hsnCode: "9030"},
			{sortOrder: 3, itemType: "product", description: "Digital Storage Oscilloscope (100MHz, 2ch)", qtyPerUnit: 2, uom: "Nos", unitPrice: 42000, gstPercent: 18, hsnCode: "9030"},
			{sortOrder: 4, itemType: "product", description: "Electronics Tool Kit (Pliers, Cutters, Tweezers)", qtyPerUnit: 10, uom: "Set", unitPrice: 3500, gstPercent: 18, hsnCode: "8205"},
		},
	}); err != nil {
		return err
	}

	// STEM Lab Main Item 4: Computing & Display
	if err := createMainItem(boq2b.Id, mainDef{
		sortOrder: 4, description: "Computing & Display", qty: 1, uom: "Lab", quotedPrice: 650000, gstPercent: 18,
		subs: []subDef{
			{sortOrder: 1, itemType: "product", description: "Student Laptop (i5/16GB/512GB SSD/14\")", qtyPerUnit: 10, uom: "Nos", unitPrice: 52000, gstPercent: 18, hsnCode: "8471"},
			{sortOrder: 2, itemType: "product", description: "75\" Interactive Flat Panel (4K)", qtyPerUnit: 1, uom: "Nos", unitPrice: 145000, gstPercent: 18, hsnCode: "8528"},
		},
	}); err != nil {
		return err
	}

	// STEM Lab Main Item 5: Furniture & Safety
	if err := createMainItem(boq2b.Id, mainDef{
		sortOrder: 5, description: "Furniture & Safety", qty: 1, uom: "Lab", quotedPrice: 350000, gstPercent: 18,
		subs: []subDef{
			{sortOrder: 1, itemType: "product", description: "STEM Workbench (1200x800mm, ESD top)", qtyPerUnit: 10, uom: "Nos", unitPrice: 18000, gstPercent: 18, hsnCode: "9403"},
			{sortOrder: 2, itemType: "product", description: "Anti-static Lab Stool (Adjustable)", qtyPerUnit: 20, uom: "Nos", unitPrice: 4500, gstPercent: 18, hsnCode: "9401"},
			{sortOrder: 3, itemType: "product", description: "Storage Cabinet (Metal, 4-shelf, Lockable)", qtyPerUnit: 4, uom: "Nos", unitPrice: 12000, gstPercent: 18, hsnCode: "9403"},
			{sortOrder: 4, itemType: "product", description: "Safety Equipment Set (Fire Ext, First Aid, ESD Mats)", qtyPerUnit: 1, uom: "Set", unitPrice: 15000, gstPercent: 18, hsnCode: "8424"},
		},
	}); err != nil {
		return err
	}

	// STEM Lab Main Item 6: Installation & Services
	if err := createMainItem(boq2b.Id, mainDef{
		sortOrder: 6, description: "Installation & Services", qty: 1, uom: "Lab", quotedPrice: 120000, gstPercent: 18,
		subs: []subDef{
			{sortOrder: 1, itemType: "service", description: "Site Survey & Assessment", qtyPerUnit: 1, uom: "Lot", unitPrice: 8000, gstPercent: 18, hsnCode: "9983"},
			{sortOrder: 2, itemType: "service", description: "Installation & Commissioning", qtyPerUnit: 1, uom: "Lot", unitPrice: 45000, gstPercent: 18, hsnCode: "9983"},
			{sortOrder: 3, itemType: "service", description: "Training — 5 Day STEM Workshop", qtyPerUnit: 1, uom: "Batch", unitPrice: 35000, gstPercent: 18, hsnCode: "9992"},
			{sortOrder: 4, itemType: "service", description: "Annual Maintenance Contract (Year 1)", qtyPerUnit: 1, uom: "Year", unitPrice: 25000, gstPercent: 18, hsnCode: "9987"},
		},
	}); err != nil {
		return err
	}

	// ── Project 2 Addresses ──────────────────────────────────────────
	_, err = createAddress(p2.Id, addressDef{
		addressType: "bill_from", companyName: "TechEd Systems Pvt. Ltd.",
		contactPerson: "Anil Sharma", addressLine1: "No. 45, 1st Cross, Koramangala 4th Block",
		city: "Bangalore", state: "Karnataka", pinCode: "560034", country: "India",
		phone: "080-41234567", email: "billing@techedsystems.in",
		gstin: "29AABCT5678F1Z5", pan: "AABCT5678F",
	})
	if err != nil {
		return err
	}

	p2BillTo, err := createAddress(p2.Id, addressDef{
		addressType: "bill_to", companyName: "Karnataka State Education Department",
		contactPerson: "Sri K. Venkatesh (Director)", addressLine1: "MS Building, Dr. Ambedkar Veedhi",
		city: "Bangalore", state: "Karnataka", pinCode: "560001", country: "India",
		phone: "080-22342345", email: "director@schooleducation.kar.nic.in",
		gstin: "29AABCK9012G1Z5", pan: "AABCK9012G",
	})
	if err != nil {
		return err
	}

	p2ShipTo, err := createAddress(p2.Id, addressDef{
		addressType: "ship_to", companyName: "Govt. Model School, Indiranagar",
		contactPerson: "Dr. Lakshmi Devi (Principal)", addressLine1: "100 Feet Road, Indiranagar",
		city: "Bangalore", state: "Karnataka", pinCode: "560038", country: "India",
		phone: "080-25281234",
	})
	if err != nil {
		return err
	}

	// ── Project 2 Address Settings ───────────────────────────────────
	if err := createAddressSettings(p2.Id, "bill_from"); err != nil {
		return fmt.Errorf("seed: address settings bill_from p2: %w", err)
	}
	if err := createAddressSettings(p2.Id, "bill_to"); err != nil {
		return fmt.Errorf("seed: address settings bill_to p2: %w", err)
	}

	// ── Project 2 Vendors ────────────────────────────────────────────
	v4, err := createVendor(vendorDef{
		name: "HP India Sales Pvt. Ltd.", addressLine1: "No. 24, Vittal Mallya Road",
		city: "Bangalore", state: "Karnataka", pinCode: "560001", country: "India",
		gstin: "29AAACH1234F1Z5", pan: "AAACH1234F",
		contactName: "Priya Rajan", phone: "080-40456789", email: "govt.orders@hp.com",
		website: "https://www.hp.com/in-en/",
		bankBeneficiary: "HP India Sales Pvt. Ltd.", bankName: "Citibank N.A.",
		bankAccountNo: "0045678901", bankIFSC: "CITI0000001", bankBranch: "MG Road, Bangalore",
	})
	if err != nil {
		return err
	}

	v5, err := createVendor(vendorDef{
		name: "STEMpedia Learning Pvt. Ltd.", addressLine1: "B-604, Titanium City Centre",
		city: "Ahmedabad", state: "Gujarat", pinCode: "380015", country: "India",
		gstin: "24AABCS5678G1Z5", pan: "AABCS5678G",
		contactName: "Dhruv Patel", phone: "079-48123456", email: "orders@stempedia.com",
		website: "https://www.stempedia.com",
		bankBeneficiary: "STEMpedia Learning Pvt. Ltd.", bankName: "Kotak Mahindra Bank",
		bankAccountNo: "7812345678", bankIFSC: "KKBK0000789", bankBranch: "SG Highway, Ahmedabad",
	})
	if err != nil {
		return err
	}

	v6, err := createVendor(vendorDef{
		name: "Godrej Interio (Godrej & Boyce Mfg.)", addressLine1: "Plant 18, Pirojshanagar, Vikhroli",
		city: "Mumbai", state: "Maharashtra", pinCode: "400079", country: "India",
		gstin: "27AAACG1234H1Z5", pan: "AAACG1234H",
		contactName: "Meera Kulkarni", phone: "022-67961234", email: "institutional@godrejinterio.com",
		website: "https://www.godrejinterio.com",
		bankBeneficiary: "Godrej & Boyce Mfg. Co. Ltd.", bankName: "Bank of Baroda",
		bankAccountNo: "98765432100", bankIFSC: "BARB0VIKHR1", bankBranch: "Vikhroli, Mumbai",
	})
	if err != nil {
		return err
	}

	// Link vendors to project 2
	for _, vid := range []string{v4.Id, v5.Id, v6.Id} {
		if err := linkVendorToProject(p2.Id, vid); err != nil {
			return fmt.Errorf("seed: link vendor to project 2: %w", err)
		}
	}

	// ── Project 2 Purchase Orders ────────────────────────────────────
	if err := createPO(p2.Id, v4.Id, p2BillTo.Id, p2ShipTo.Id, purchaseOrderDef{
		poNumber: "PO-FY26-003", orderDate: "2025-06-20", quotationRef: "HP/EDU/Q/2025/4721",
		paymentTerms: "100% against delivery with installation certificate",
		deliveryTerms: "FOR destination, Bangalore",
		warrantyTerms: "3 years onsite next-business-day warranty",
		status: "acknowledged",
		lineItems: []poLineItemDef{
			{sortOrder: 1, description: "HP ProDesk 400 G9 SFF (i3/8GB/512GB) + HP P22v Monitor", hsnCode: "8471", qty: 30, uom: "Nos", rate: 37500, gstPercent: 18, sourceItemType: "sub_item"},
			{sortOrder: 2, description: "HP ProDesk 400 G9 SFF (i5/16GB/512GB) + HP E24 Monitor", hsnCode: "8471", qty: 1, uom: "Nos", rate: 54000, gstPercent: 18, sourceItemType: "sub_item"},
			{sortOrder: 3, description: "HP ProLiant DL20 Gen10+ (Xeon/32GB/2TB)", hsnCode: "8471", qty: 1, uom: "Nos", rate: 182000, gstPercent: 18, sourceItemType: "sub_item"},
			{sortOrder: 4, description: "HP ProBook 440 G10 (i5/16GB/512GB/14\")", hsnCode: "8471", qty: 10, uom: "Nos", rate: 51000, gstPercent: 18, sourceItemType: "sub_item"},
		},
	}); err != nil {
		return err
	}

	if err := createPO(p2.Id, v5.Id, p2BillTo.Id, p2ShipTo.Id, purchaseOrderDef{
		poNumber: "PO-FY26-004", orderDate: "2025-07-05",
		paymentTerms: "50% advance, 50% on delivery",
		deliveryTerms: "Delivery to school, training at site",
		status: "draft",
		lineItems: []poLineItemDef{
			{sortOrder: 1, description: "Arduino Mega Starter Kit (with sensors & actuators)", hsnCode: "8542", qty: 20, uom: "Nos", rate: 4200, gstPercent: 18, sourceItemType: "sub_item"},
			{sortOrder: 2, description: "Basic Robotics Kit (2WD + sensors + programming guide)", hsnCode: "8479", qty: 15, uom: "Nos", rate: 11500, gstPercent: 18, sourceItemType: "sub_item"},
			{sortOrder: 3, description: "IoT Sensor & Actuator Kit (DHT11, MQ2, Relay, Servo)", hsnCode: "8542", qty: 15, uom: "Nos", rate: 3200, gstPercent: 18, sourceItemType: "sub_item"},
		},
	}); err != nil {
		return err
	}

	log.Println("seed: all seed data inserted successfully (2 projects, 3 BOQs, 9 addresses, 6 vendors, 4 POs)")
	return nil
}
