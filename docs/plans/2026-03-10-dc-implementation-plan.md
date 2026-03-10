# Delivery Challans (DC) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a configurable Delivery Challans system with project-linked and standalone DCs, type-driven behavior, BOQ dispatch tracking, and PDF export.

**Architecture:** Type-driven configuration model — `dc_type_configs` drives DC behavior (priced/unpriced, multi-destination, serial numbers). DCs can link to projects (pulling BOQ items + addresses) or be standalone. PDF export replicates existing DC sample layout using maroto/v2.

**Tech Stack:** Go, PocketBase, Templ, HTMX, Alpine.js, Tailwind CSS v4, DaisyUI v5, maroto/v2 (PDF)

---

## Phase 1: Data Layer — Collections & Seed Data

### Task 1: Add DC Type Configs Collection

**Files:**
- Modify: `collections/setup.go`

**Step 1: Write the collection setup code**

Add to `Setup()` in `collections/setup.go`, after the Purchase Orders section:

```go
// ── DC Type Configs ─────────────────────────────────────────────
dcTypeConfigs := ensureCollection(app, "dc_type_configs", func(c *core.Collection) {
    c.Fields.Add(&core.TextField{Name: "name", Required: true})
    c.Fields.Add(&core.TextField{Name: "code", Required: true})
    c.Fields.Add(&core.BoolField{Name: "is_priced"})
    c.Fields.Add(&core.BoolField{Name: "is_multi_destination"})
    c.Fields.Add(&core.BoolField{Name: "requires_serial_numbers"})
    c.Fields.Add(&core.BoolField{Name: "allows_parent_dc"})
    c.Fields.Add(&core.TextField{Name: "description"})
    c.Fields.Add(&core.BoolField{Name: "active"})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
    c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
})
```

**Step 2: Run and verify**

Run: `go build ./...`
Expected: Compiles successfully.

**Step 3: Commit**

```bash
git add collections/setup.go
git commit -m "feat(dc): add dc_type_configs collection"
```

### Task 2: Add DC Number Series Collection

**Files:**
- Modify: `collections/setup.go`

**Step 1: Add collection after dc_type_configs**

```go
// ── DC Number Series ────────────────────────────────────────────
ensureCollection(app, "dc_number_series", func(c *core.Collection) {
    c.Fields.Add(&core.TextField{Name: "prefix", Required: true})
    c.Fields.Add(&core.TextField{Name: "financial_year", Required: true})
    c.Fields.Add(&core.NumberField{Name: "last_number"})
    c.Fields.Add(&core.BoolField{Name: "active"})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
    c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
})
ensureField(app, "dc_number_series", &core.RelationField{
    Name: "dc_type", Required: true,
    CollectionId: dcTypeConfigs.Id, CascadeDelete: false, MaxSelect: 1,
})
```

**Step 2: Run and verify**

Run: `go build ./...`

**Step 3: Commit**

```bash
git add collections/setup.go
git commit -m "feat(dc): add dc_number_series collection"
```

### Task 3: Add Delivery Challans Collection

**Files:**
- Modify: `collections/setup.go`

**Step 1: Add the main delivery_challans collection**

```go
// ── Delivery Challans ───────────────────────────────────────────
deliveryChallans := ensureCollection(app, "delivery_challans", func(c *core.Collection) {
    c.Fields.Add(&core.TextField{Name: "dc_number", Required: true})
    c.Fields.Add(&core.TextField{Name: "dc_date", Required: true})
    c.Fields.Add(&core.SelectField{
        Name: "status", Required: true,
        Values:    []string{"draft", "finalized", "dispatched", "delivered"},
        MaxSelect: 1,
    })
    c.Fields.Add(&core.TextField{Name: "po_number"})
    c.Fields.Add(&core.TextField{Name: "po_date"})

    // Bill From (flattened)
    c.Fields.Add(&core.TextField{Name: "bill_from_name"})
    c.Fields.Add(&core.TextField{Name: "bill_from_address1"})
    c.Fields.Add(&core.TextField{Name: "bill_from_address2"})
    c.Fields.Add(&core.TextField{Name: "bill_from_city"})
    c.Fields.Add(&core.TextField{Name: "bill_from_state"})
    c.Fields.Add(&core.TextField{Name: "bill_from_pincode"})
    c.Fields.Add(&core.TextField{Name: "bill_from_gstin"})
    c.Fields.Add(&core.TextField{Name: "bill_from_contact"})
    c.Fields.Add(&core.TextField{Name: "bill_from_phone"})

    // Bill To (flattened)
    c.Fields.Add(&core.TextField{Name: "bill_to_name"})
    c.Fields.Add(&core.TextField{Name: "bill_to_address1"})
    c.Fields.Add(&core.TextField{Name: "bill_to_address2"})
    c.Fields.Add(&core.TextField{Name: "bill_to_city"})
    c.Fields.Add(&core.TextField{Name: "bill_to_state"})
    c.Fields.Add(&core.TextField{Name: "bill_to_pincode"})
    c.Fields.Add(&core.TextField{Name: "bill_to_gstin"})
    c.Fields.Add(&core.TextField{Name: "bill_to_contact"})
    c.Fields.Add(&core.TextField{Name: "bill_to_phone"})

    // Dispatch From (flattened)
    c.Fields.Add(&core.TextField{Name: "dispatch_from_name"})
    c.Fields.Add(&core.TextField{Name: "dispatch_from_address1"})
    c.Fields.Add(&core.TextField{Name: "dispatch_from_address2"})
    c.Fields.Add(&core.TextField{Name: "dispatch_from_city"})
    c.Fields.Add(&core.TextField{Name: "dispatch_from_state"})
    c.Fields.Add(&core.TextField{Name: "dispatch_from_pincode"})
    c.Fields.Add(&core.TextField{Name: "dispatch_from_gstin"})
    c.Fields.Add(&core.TextField{Name: "dispatch_from_contact"})
    c.Fields.Add(&core.TextField{Name: "dispatch_from_phone"})

    // Ship To (flattened)
    c.Fields.Add(&core.TextField{Name: "ship_to_name"})
    c.Fields.Add(&core.TextField{Name: "ship_to_address1"})
    c.Fields.Add(&core.TextField{Name: "ship_to_address2"})
    c.Fields.Add(&core.TextField{Name: "ship_to_city"})
    c.Fields.Add(&core.TextField{Name: "ship_to_state"})
    c.Fields.Add(&core.TextField{Name: "ship_to_pincode"})
    c.Fields.Add(&core.TextField{Name: "ship_to_gstin"})
    c.Fields.Add(&core.TextField{Name: "ship_to_contact"})
    c.Fields.Add(&core.TextField{Name: "ship_to_phone"})

    // Transport
    c.Fields.Add(&core.TextField{Name: "transporter_name"})
    c.Fields.Add(&core.TextField{Name: "vehicle_number"})
    c.Fields.Add(&core.TextField{Name: "eway_bill_number"})
    c.Fields.Add(&core.TextField{Name: "docket_number"})
    c.Fields.Add(&core.BoolField{Name: "reverse_charge"})
    c.Fields.Add(&core.SelectField{
        Name: "transport_mode",
        Values:    []string{"road", "rail", "air", "ship"},
        MaxSelect: 1,
    })

    // Financials (priced DCs only)
    c.Fields.Add(&core.NumberField{Name: "total_taxable_value"})
    c.Fields.Add(&core.NumberField{Name: "total_cgst"})
    c.Fields.Add(&core.NumberField{Name: "total_sgst"})
    c.Fields.Add(&core.NumberField{Name: "total_igst"})
    c.Fields.Add(&core.NumberField{Name: "round_off"})
    c.Fields.Add(&core.NumberField{Name: "invoice_value"})

    // Over-dispatch
    c.Fields.Add(&core.TextField{Name: "over_dispatch_reason"})
    c.Fields.Add(&core.TextField{Name: "notes"})

    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
    c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
})

// Relations (added after creation for self-reference support)
ensureField(app, "delivery_challans", &core.RelationField{
    Name: "dc_type", Required: true,
    CollectionId: dcTypeConfigs.Id, CascadeDelete: false, MaxSelect: 1,
})
ensureField(app, "delivery_challans", &core.RelationField{
    Name: "project", Required: false,
    CollectionId: projects.Id, CascadeDelete: false, MaxSelect: 1,
})
ensureField(app, "delivery_challans", &core.RelationField{
    Name:          "parent_dc",
    Required:      false,
    CollectionId:  deliveryChallans.Id,
    CascadeDelete: false,
    MaxSelect:     1,
})
```

**Step 2: Run and verify**

Run: `go build ./...`

**Step 3: Commit**

```bash
git add collections/setup.go
git commit -m "feat(dc): add delivery_challans collection with address and transport fields"
```

### Task 4: Add DC Line Items, Destinations, Destination Items Collections

**Files:**
- Modify: `collections/setup.go`

**Step 1: Add the three child collections**

```go
// ── DC Line Items ───────────────────────────────────────────────
ensureCollection(app, "dc_line_items", func(c *core.Collection) {
    c.Fields.Add(&core.NumberField{Name: "sort_order", Required: true})
    c.Fields.Add(&core.TextField{Name: "description", Required: true})
    c.Fields.Add(&core.TextField{Name: "make_model"})
    c.Fields.Add(&core.TextField{Name: "uom"})
    c.Fields.Add(&core.TextField{Name: "hsn_code"})
    c.Fields.Add(&core.NumberField{Name: "qty", Required: true})
    c.Fields.Add(&core.NumberField{Name: "rate"})
    c.Fields.Add(&core.NumberField{Name: "taxable_value"})
    c.Fields.Add(&core.NumberField{Name: "gst_percent"})
    c.Fields.Add(&core.NumberField{Name: "gst_amount"})
    c.Fields.Add(&core.NumberField{Name: "total"})
    c.Fields.Add(&core.TextField{Name: "serial_numbers"})
    c.Fields.Add(&core.TextField{Name: "remarks"})
    c.Fields.Add(&core.TextField{Name: "boq_item_id"})
    c.Fields.Add(&core.TextField{Name: "boq_item_level"})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
    c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
})
ensureField(app, "dc_line_items", &core.RelationField{
    Name: "dc", Required: true,
    CollectionId: deliveryChallans.Id, CascadeDelete: true, MaxSelect: 1,
})

// ── DC Destinations (multi-destination DCs) ─────────────────────
dcDestinations := ensureCollection(app, "dc_destinations", func(c *core.Collection) {
    c.Fields.Add(&core.TextField{Name: "destination_name"})
    c.Fields.Add(&core.TextField{Name: "destination_address"})
    c.Fields.Add(&core.TextField{Name: "destination_city"})
    c.Fields.Add(&core.TextField{Name: "destination_state"})
    c.Fields.Add(&core.TextField{Name: "destination_pincode"})
    c.Fields.Add(&core.TextField{Name: "contact_person"})
    c.Fields.Add(&core.TextField{Name: "contact_phone"})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
})
ensureField(app, "dc_destinations", &core.RelationField{
    Name: "dc", Required: true,
    CollectionId: deliveryChallans.Id, CascadeDelete: true, MaxSelect: 1,
})

// ── DC Destination Items (qty per destination per item) ─────────
ensureCollection(app, "dc_destination_items", func(c *core.Collection) {
    c.Fields.Add(&core.NumberField{Name: "qty"})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
})
ensureField(app, "dc_destination_items", &core.RelationField{
    Name: "dc_destination", Required: true,
    CollectionId: dcDestinations.Id, CascadeDelete: true, MaxSelect: 1,
})
// Note: dc_line_item relation added after dc_line_items collection is created
// Use ensureField pattern since both collections exist by this point
```

After the dc_destination_items collection is created, add the line item relation. Since `dc_line_items` is created earlier in the same function, you need to look it up:

```go
dcLineItemsCol, _ := app.FindCollectionByNameOrId("dc_line_items")
if dcLineItemsCol != nil {
    ensureField(app, "dc_destination_items", &core.RelationField{
        Name: "dc_line_item", Required: true,
        CollectionId: dcLineItemsCol.Id, CascadeDelete: true, MaxSelect: 1,
    })
}
```

**Step 2: Run and verify**

Run: `go build ./...`

**Step 3: Commit**

```bash
git add collections/setup.go
git commit -m "feat(dc): add dc_line_items, dc_destinations, dc_destination_items collections"
```

### Task 5: Add Standalone Addresses Collection

**Files:**
- Modify: `collections/setup.go`

**Step 1: Add the standalone_addresses collection**

```go
// ── Standalone Addresses (for non-project DCs) ──────────────────
ensureCollection(app, "standalone_addresses", func(c *core.Collection) {
    c.Fields.Add(&core.SelectField{
        Name: "address_type", Required: true,
        Values:    []string{"bill_from", "bill_to", "dispatch_from", "ship_to"},
        MaxSelect: 1,
    })
    c.Fields.Add(&core.TextField{Name: "name", Required: true})
    c.Fields.Add(&core.TextField{Name: "address1"})
    c.Fields.Add(&core.TextField{Name: "address2"})
    c.Fields.Add(&core.TextField{Name: "city"})
    c.Fields.Add(&core.TextField{Name: "state"})
    c.Fields.Add(&core.TextField{Name: "pincode"})
    c.Fields.Add(&core.TextField{Name: "gstin"})
    c.Fields.Add(&core.TextField{Name: "contact_person"})
    c.Fields.Add(&core.TextField{Name: "phone"})
    c.Fields.Add(&core.EmailField{Name: "email"})
    c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
    c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
})
```

**Step 2: Run and verify**

Run: `go build ./...`

**Step 3: Commit**

```bash
git add collections/setup.go
git commit -m "feat(dc): add standalone_addresses collection for non-project DCs"
```

### Task 6: Seed DC Type Configs and Number Series

**Files:**
- Modify: `collections/seed.go`

**Step 1: Add DC seed data at the end of `Seed()` function, before the final log line**

Add DC type config and number series seed data:

```go
// ── DC Type Configs ────────────────────────────────────────────
dcTypeCol, err := app.FindCollectionByNameOrId("dc_type_configs")
if err != nil {
    return fmt.Errorf("seed: could not find dc_type_configs collection: %w", err)
}

// Check idempotency
existingTypes, _ := app.FindAllRecords(dcTypeCol)
if len(existingTypes) == 0 {
    dcTypes := []struct {
        name, code, description string
        isPriced, isMultiDest, requiresSerials, allowsParent, active bool
    }{
        {"Transfer Delivery Challan", "TDC", "Standard transfer of goods between locations", true, true, false, false, true},
        {"Transit Delivery Challan", "TTDC", "Intermediate movement referencing a parent DC", true, false, false, true, true},
        {"Official Delivery Challan", "ODC", "Final delivery with receiver sign-off, unpriced", false, false, true, true, true},
        {"Return Delivery Challan", "RDC", "Defective or vendor return goods", true, false, false, false, true},
        {"Sample Delivery Challan", "SDC", "Samples for testing or evaluation", false, false, true, false, true},
    }

    dcSeriesCol, _ := app.FindCollectionByNameOrId("dc_number_series")

    for _, dt := range dcTypes {
        r := core.NewRecord(dcTypeCol)
        r.Set("name", dt.name)
        r.Set("code", dt.code)
        r.Set("description", dt.description)
        r.Set("is_priced", dt.isPriced)
        r.Set("is_multi_destination", dt.isMultiDest)
        r.Set("requires_serial_numbers", dt.requiresSerials)
        r.Set("allows_parent_dc", dt.allowsParent)
        r.Set("active", dt.active)
        if err := app.Save(r); err != nil {
            return fmt.Errorf("seed: save dc_type %q: %w", dt.code, err)
        }

        // Create default number series for each type
        if dcSeriesCol != nil {
            sr := core.NewRecord(dcSeriesCol)
            sr.Set("prefix", "OAVS")
            sr.Set("dc_type", r.Id)
            sr.Set("financial_year", "2526")
            sr.Set("last_number", 0)
            sr.Set("active", true)
            if err := app.Save(sr); err != nil {
                return fmt.Errorf("seed: save dc_number_series for %q: %w", dt.code, err)
            }
        }
    }
    log.Println("seed: DC type configs and number series inserted")
}
```

**Step 2: Run server to verify seed data loads**

Run: `go build ./... && go run main.go serve` (then Ctrl+C after startup)
Expected: Log shows "DC type configs and number series inserted" on first run.

**Step 3: Commit**

```bash
git add collections/seed.go
git commit -m "feat(dc): seed DC type configs (TDC, TTDC, ODC, RDC, SDC) and number series"
```

---

## Phase 2: Services Layer

### Task 7: DC Number Generation Service

**Files:**
- Create: `services/dc_number.go`
- Create: `services/dc_number_test.go`

**Step 1: Write the failing test**

```go
package services

import (
    "testing"
)

func TestFormatDCNumber(t *testing.T) {
    tests := []struct {
        name   string
        prefix string
        code   string
        fy     string
        seq    int
        want   string
    }{
        {"basic", "OAVS", "TDC", "2526", 1, "OAVS-TDC-2526-001"},
        {"double digit", "OAVS", "ODC", "2526", 13, "OAVS-ODC-2526-013"},
        {"triple digit", "FRV", "RDC", "2526", 100, "FRV-RDC-2526-100"},
        {"four digit", "OAVS", "TDC", "2526", 1234, "OAVS-TDC-2526-1234"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := FormatDCNumber(tt.prefix, tt.code, tt.fy, tt.seq)
            if got != tt.want {
                t.Errorf("FormatDCNumber() = %q, want %q", got, tt.want)
            }
        })
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./services/ -run TestFormatDCNumber -v`
Expected: FAIL — `FormatDCNumber` undefined.

**Step 3: Write the implementation**

```go
package services

import (
    "fmt"

    "github.com/pocketbase/pocketbase"
)

// FormatDCNumber formats a DC number from its components.
// Format: {prefix}-{typeCode}-{FY}-{seq padded to min 3 digits}
func FormatDCNumber(prefix, typeCode, fy string, seq int) string {
    return fmt.Sprintf("%s-%s-%s-%03d", prefix, typeCode, fy, seq)
}

// GenerateDCNumber finds the active series for a DC type, atomically increments
// the counter, and returns the formatted DC number.
func GenerateDCNumber(app *pocketbase.PocketBase, dcTypeId string) (string, error) {
    // 1. Look up the DC type to get the code
    dcType, err := app.FindRecordById("dc_type_configs", dcTypeId)
    if err != nil {
        return "", fmt.Errorf("dc type not found: %w", err)
    }
    typeCode := dcType.GetString("code")

    // 2. Find the active series for this type
    series, err := app.FindRecordsByFilter(
        "dc_number_series",
        "dc_type = {:typeId} && active = true",
        "-created",
        1,
        0,
        map[string]any{"typeId": dcTypeId},
    )
    if err != nil || len(series) == 0 {
        return "", fmt.Errorf("no active number series found for DC type %s", typeCode)
    }

    record := series[0]
    prefix := record.GetString("prefix")
    fy := record.GetString("financial_year")
    lastNum := record.GetInt("last_number")
    nextNum := lastNum + 1

    // 3. Increment and save
    record.Set("last_number", nextNum)
    if err := app.Save(record); err != nil {
        return "", fmt.Errorf("failed to increment DC number series: %w", err)
    }

    return FormatDCNumber(prefix, typeCode, fy, nextNum), nil
}
```

**Step 4: Run tests**

Run: `go test ./services/ -run TestFormatDCNumber -v`
Expected: PASS

**Step 5: Commit**

```bash
git add services/dc_number.go services/dc_number_test.go
git commit -m "feat(dc): add DC number generation service with formatting and auto-increment"
```

### Task 8: DC Calculations Service

**Files:**
- Create: `services/dc_calculations.go`
- Create: `services/dc_calculations_test.go`

**Step 1: Write the failing test**

```go
package services

import (
    "math"
    "testing"
)

func TestCalcDCLineItem(t *testing.T) {
    tests := []struct {
        name       string
        qty, rate  float64
        gstPercent float64
        wantTax    float64
        wantGST    float64
        wantTotal  float64
    }{
        {"basic 18%", 3, 12500, 18, 37500, 6750, 44250},
        {"zero qty", 0, 12500, 18, 0, 0, 0},
        {"zero rate", 3, 0, 18, 0, 0, 0},
        {"no gst", 10, 1000, 0, 10000, 0, 10000},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            taxable, gstAmt, total := CalcDCLineItem(tt.qty, tt.rate, tt.gstPercent)
            if math.Abs(taxable-tt.wantTax) > 0.01 {
                t.Errorf("taxable = %f, want %f", taxable, tt.wantTax)
            }
            if math.Abs(gstAmt-tt.wantGST) > 0.01 {
                t.Errorf("gstAmt = %f, want %f", gstAmt, tt.wantGST)
            }
            if math.Abs(total-tt.wantTotal) > 0.01 {
                t.Errorf("total = %f, want %f", total, tt.wantTotal)
            }
        })
    }
}

func TestCalcDCTotals(t *testing.T) {
    items := []DCLineItemCalc{
        {TaxableValue: 37500, GSTPercent: 18, GSTAmount: 6750, Total: 44250},
        {TaxableValue: 12500, GSTPercent: 18, GSTAmount: 2250, Total: 14750},
    }
    totals := CalcDCTotals(items)

    if math.Abs(totals.TotalTaxable-50000) > 0.01 {
        t.Errorf("TotalTaxable = %f, want 50000", totals.TotalTaxable)
    }
    if math.Abs(totals.InvoiceValue-59000) > 1 {
        t.Errorf("InvoiceValue = %f, want ~59000", totals.InvoiceValue)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./services/ -run TestCalcDC -v`
Expected: FAIL

**Step 3: Write the implementation**

```go
package services

import "math"

// DCLineItemCalc holds calculated values for a single DC line item.
type DCLineItemCalc struct {
    TaxableValue float64
    GSTPercent   float64
    GSTAmount    float64
    Total        float64
}

// DCTotals holds aggregated totals for a DC.
type DCTotals struct {
    TotalTaxable float64
    TotalCGST    float64
    TotalSGST    float64
    TotalIGST    float64
    RoundOff     float64
    InvoiceValue float64
}

// CalcDCLineItem calculates taxable value, GST amount, and total for a line item.
func CalcDCLineItem(qty, rate, gstPercent float64) (taxable, gstAmt, total float64) {
    taxable = qty * rate
    gstAmt = taxable * gstPercent / 100
    total = taxable + gstAmt
    return
}

// CalcDCTotals aggregates line item calculations into DC-level totals.
// Assumes intra-state (CGST/SGST split). GST is split 50/50 into CGST and SGST.
func CalcDCTotals(items []DCLineItemCalc) DCTotals {
    var totals DCTotals
    for _, item := range items {
        totals.TotalTaxable += item.TaxableValue
        totals.TotalCGST += item.GSTAmount / 2
        totals.TotalSGST += item.GSTAmount / 2
    }
    rawTotal := totals.TotalTaxable + totals.TotalCGST + totals.TotalSGST
    totals.InvoiceValue = math.Round(rawTotal)
    totals.RoundOff = totals.InvoiceValue - rawTotal
    return totals
}
```

**Step 4: Run tests**

Run: `go test ./services/ -run TestCalcDC -v`
Expected: PASS

**Step 5: Commit**

```bash
git add services/dc_calculations.go services/dc_calculations_test.go
git commit -m "feat(dc): add DC line item and totals calculation service"
```

### Task 9: BOQ Dispatch Tracker Service

**Files:**
- Create: `services/dc_boq_tracker.go`

**Step 1: Write the implementation**

```go
package services

import (
    "fmt"

    "github.com/pocketbase/pocketbase"
)

// BOQDispatchInfo holds dispatch tracking info for a BOQ item.
type BOQDispatchInfo struct {
    BOQItemID     string
    BOQItemLevel  string
    TotalQty      float64
    DispatchedQty float64
    AvailableQty  float64
}

// GetBOQItemDispatchedQty sums the qty dispatched across all DCs for a specific BOQ item.
func GetBOQItemDispatchedQty(app *pocketbase.PocketBase, boqItemId, level string) float64 {
    records, err := app.FindRecordsByFilter(
        "dc_line_items",
        "boq_item_id = {:itemId} && boq_item_level = {:level}",
        "",
        0,
        0,
        map[string]any{"itemId": boqItemId, "level": level},
    )
    if err != nil {
        return 0
    }
    var total float64
    for _, r := range records {
        total += r.GetFloat("qty")
    }
    return total
}

// CheckOverDispatch checks if the requested qty exceeds the available BOQ qty.
// Returns whether it's an over-dispatch and the available qty.
func CheckOverDispatch(app *pocketbase.PocketBase, boqItemId, level string, boqTotalQty, requestedQty float64) (isOver bool, available float64) {
    dispatched := GetBOQItemDispatchedQty(app, boqItemId, level)
    available = boqTotalQty - dispatched
    isOver = requestedQty > available
    return
}

// GetBOQItemForDC retrieves a BOQ item record and returns its details for populating a DC line item.
func GetBOQItemForDC(app *pocketbase.PocketBase, itemId, level string) (description, uom, hsnCode string, qty float64, unitPrice float64, gstPercent float64, err error) {
    collectionName := ""
    switch level {
    case "main":
        collectionName = "main_boq_items"
    case "sub":
        collectionName = "sub_items"
    case "sub_sub":
        collectionName = "sub_sub_items"
    default:
        err = fmt.Errorf("unknown BOQ item level: %s", level)
        return
    }

    record, findErr := app.FindRecordById(collectionName, itemId)
    if findErr != nil {
        err = fmt.Errorf("BOQ item not found: %w", findErr)
        return
    }

    description = record.GetString("description")
    uom = record.GetString("uom")
    hsnCode = record.GetString("hsn_code")
    gstPercent = record.GetFloat("gst_percent")

    switch level {
    case "main":
        qty = record.GetFloat("qty")
        unitPrice = record.GetFloat("unit_price")
    case "sub", "sub_sub":
        qty = record.GetFloat("qty_per_unit")
        unitPrice = record.GetFloat("unit_price")
    }
    return
}
```

**Step 2: Run and verify**

Run: `go build ./...`
Expected: Compiles.

**Step 3: Commit**

```bash
git add services/dc_boq_tracker.go
git commit -m "feat(dc): add BOQ dispatch tracking service"
```

---

## Phase 3: Settings UI (DC Types, Number Series, Standalone Addresses)

### Task 10: DC Type Config Settings — Handler & Template

**Files:**
- Create: `handlers/dc_type_config.go`
- Create: `templates/dc_type_config.templ`
- Modify: `main.go` (add routes)

**Step 1: Create the handler**

File: `handlers/dc_type_config.go`

Implement `HandleDCTypeConfigList(app)` and `HandleDCTypeConfigSave(app)` following the same pattern as `HandleAppSettings` in `handlers/app_settings.go`:
- GET renders list of all dc_type_configs with create/edit form
- POST validates and saves (create or update)
- HTMX detection for partial/full page rendering

**Step 2: Create the template**

File: `templates/dc_type_config.templ`

Data struct:
```go
type DCTypeConfigData struct {
    Types      []DCTypeConfigItem
    EditType   *DCTypeConfigItem // nil for create mode
    Errors     map[string]string
    FormData   DCTypeConfigItem
}

type DCTypeConfigItem struct {
    ID                   string
    Name                 string
    Code                 string
    IsPriced             bool
    IsMultiDestination   bool
    RequiresSerialNumbers bool
    AllowsParentDC       bool
    Description          string
    Active               bool
}
```

Template components: `DCTypeConfigContent(data)` and `DCTypeConfigPage(data, header, sidebar)`.

Table with columns: Name, Code, Priced, Multi-Dest, Serials, Parent DC, Active, Actions (Edit).
Below table: Create/Edit form with text inputs and toggle switches.

**Step 3: Add routes to main.go**

```go
// ── DC Type Config (settings) ──────────────────────────────
se.Router.GET("/settings/dc-types", handlers.HandleDCTypeConfigList(app))
se.Router.POST("/settings/dc-types", handlers.HandleDCTypeConfigSave(app))
se.Router.GET("/settings/dc-types/{id}/edit", handlers.HandleDCTypeConfigEdit(app))
se.Router.POST("/settings/dc-types/{id}/save", handlers.HandleDCTypeConfigUpdate(app))
```

**Step 4: Run templ generate and verify**

Run: `templ generate && go build ./...`

**Step 5: Commit**

```bash
git add handlers/dc_type_config.go templates/dc_type_config.templ templates/dc_type_config_templ.go main.go
git commit -m "feat(dc): add DC type config settings page with CRUD"
```

### Task 11: DC Number Series Settings — Handler & Template

**Files:**
- Create: `handlers/dc_number_series.go`
- Create: `templates/dc_number_series.templ`
- Modify: `main.go` (add routes)

Follow the same pattern as Task 10. Key features:
- List all series with prefix, type name, FY, last number, active status
- Create form: prefix, dc_type dropdown, financial_year, starting number
- "Start New Series" action: deactivate current active series, create new one with counter at 0
- Validation: only one active series per prefix+type+FY combination

**Routes:**
```go
se.Router.GET("/settings/dc-series", handlers.HandleDCNumberSeriesList(app))
se.Router.POST("/settings/dc-series", handlers.HandleDCNumberSeriesSave(app))
se.Router.POST("/settings/dc-series/{id}/save", handlers.HandleDCNumberSeriesUpdate(app))
```

**Step: Run, verify, commit**

```bash
templ generate && go build ./...
git add handlers/dc_number_series.go templates/dc_number_series.templ templates/dc_number_series_templ.go main.go
git commit -m "feat(dc): add DC number series settings page"
```

### Task 12: Standalone Address Book — Handler & Template

**Files:**
- Create: `handlers/dc_standalone_addr.go`
- Create: `templates/dc_standalone_addr.templ`
- Modify: `main.go` (add routes)

Follow existing address management patterns. Key features:
- List addresses filtered by type tabs (bill_from, bill_to, dispatch_from, ship_to)
- Create/Edit form with standard address fields
- Delete with confirmation

**Routes:**
```go
se.Router.GET("/settings/dc-addresses", handlers.HandleStandaloneAddrList(app))
se.Router.POST("/settings/dc-addresses", handlers.HandleStandaloneAddrSave(app))
se.Router.GET("/settings/dc-addresses/{id}/edit", handlers.HandleStandaloneAddrEdit(app))
se.Router.POST("/settings/dc-addresses/{id}/save", handlers.HandleStandaloneAddrUpdate(app))
se.Router.DELETE("/settings/dc-addresses/{id}", handlers.HandleStandaloneAddrDelete(app))
```

**Step: Run, verify, commit**

```bash
templ generate && go build ./...
git add handlers/dc_standalone_addr.go templates/dc_standalone_addr.templ templates/dc_standalone_addr_templ.go main.go
git commit -m "feat(dc): add standalone address book for non-project DCs"
```

---

## Phase 4: DC List & View Pages

### Task 13: DC List — Handler & Template (Global + Project-scoped)

**Files:**
- Create: `handlers/dc_list.go`
- Create: `templates/dc_list.templ`
- Modify: `main.go` (add routes)

**Handler logic (`HandleDCList`):**
- Accept optional `projectId` path param — if present, filter by project
- Query params for filters: `type`, `status`, `date_from`, `date_to`
- Build filter string dynamically from query params
- Fetch dc_type name for each DC (join or separate query)
- Return list data

**Template data struct:**
```go
type DCListData struct {
    DCs          []DCListItem
    ProjectID    string // empty for global view
    ProjectName  string
    DCTypes      []DCTypeOption // for filter dropdown
    FilterType   string
    FilterStatus string
}

type DCListItem struct {
    ID         string
    DCNumber   string
    TypeName   string
    TypeCode   string
    DCDate     string
    ProjectName string // empty if non-project
    ShipToName string
    ShipToCity string
    Status     string
}

type DCTypeOption struct {
    ID   string
    Name string
    Code string
}
```

**Routes:**
```go
// Global DC routes
se.Router.GET("/dcs", handlers.HandleDCList(app))

// Project-scoped DC routes
se.Router.GET("/projects/{projectId}/dcs", handlers.HandleDCList(app))
```

Note: Same handler serves both — it checks for `projectId` path param.

**Template:** Table with filter bar, status badges (same color scheme as PO: draft=gray, finalized=blue, dispatched=amber, delivered=green). "Create DC" button. Each row links to DC view.

**Step: Run, verify, commit**

```bash
templ generate && go build ./...
git add handlers/dc_list.go templates/dc_list.templ templates/dc_list_templ.go main.go
git commit -m "feat(dc): add DC list page with filters (global + project-scoped)"
```

### Task 14: DC View — Handler & Template

**Files:**
- Create: `handlers/dc_view.go`
- Create: `templates/dc_view.templ`
- Modify: `main.go` (add routes)

**Handler logic (`HandleDCView`):**
- Fetch DC by ID
- Fetch dc_type_configs record for behavior flags
- Fetch line items sorted by sort_order
- Fetch parent DC if exists
- Fetch child DCs (where parent_dc = this DC's ID)
- If priced: calculate totals using `CalcDCTotals`
- If multi-destination: fetch destinations and destination items
- Build view data struct

**Template data struct:**
```go
type DCViewData struct {
    // DC Header
    ID, DCNumber, DCDate, Status string
    TypeName, TypeCode string
    IsPriced, IsMultiDestination bool
    ProjectID, ProjectName string
    ParentDCID, ParentDCNumber string
    PONumber, PODate string

    // Addresses (4-box)
    BillFrom, BillTo, DispatchFrom, ShipTo DCViewAddress

    // Transport
    TransporterName, VehicleNumber, EwayBillNumber string
    DocketNumber, TransportMode string
    ReverseCharge bool

    // Line Items
    LineItems []DCViewLineItem

    // Totals (priced only)
    TotalTaxable, TotalCGST, TotalSGST string
    RoundOff, InvoiceValue, AmountInWords string

    // Destinations (multi-destination only)
    Destinations []DCViewDestination

    // Child DCs
    ChildDCs []DCViewChildDC

    // Over-dispatch
    OverDispatchReason string
    Notes string
}
```

**Template layout:** Breadcrumbs, status badge, action buttons (Edit, Export PDF, status transitions via HTMX POST), 4-box address layout, line items table (priced or unpriced based on type), totals section, destinations annexure, serial numbers, DC chain section.

**Routes:**
```go
se.Router.GET("/dcs/{id}", handlers.HandleDCView(app))
se.Router.GET("/projects/{projectId}/dcs/{id}", handlers.HandleDCView(app))
```

**Step: Run, verify, commit**

```bash
templ generate && go build ./...
git add handlers/dc_view.go templates/dc_view.templ templates/dc_view_templ.go main.go
git commit -m "feat(dc): add DC view page with type-driven layout"
```

### Task 15: DC Status Transitions

**Files:**
- Create: `handlers/dc_status.go`
- Modify: `main.go` (add routes)

**Handler logic:**
- `HandleDCStatusUpdate(app)` — POST handler
- Validates transition: draft→finalized, finalized→dispatched, dispatched→delivered
- Updates status field
- Returns HTMX redirect back to DC view

**Routes:**
```go
se.Router.POST("/dcs/{id}/status", handlers.HandleDCStatusUpdate(app))
```

**Step: Run, verify, commit**

```bash
git add handlers/dc_status.go main.go
git commit -m "feat(dc): add DC status transition handler"
```

---

## Phase 5: DC Create & Edit

### Task 16: DC Create — Handler & Template

**Files:**
- Create: `handlers/dc_create.go`
- Create: `templates/dc_create.templ`
- Modify: `main.go` (add routes)

This is the most complex task. Build incrementally.

**Step 1: Basic create form (type + date + addresses + transport)**

Handler `HandleDCCreate(app)`:
- Fetch all active dc_type_configs for dropdown
- If projectId in path, fetch project's addresses for pickers
- If no project, fetch standalone addresses
- Render create form

Handler `HandleDCSave(app)`:
- Parse form
- Validate required fields (dc_type, dc_date)
- Generate DC number via `GenerateDCNumber`
- Create delivery_challans record with status "draft"
- Redirect to DC view

**Template:** Multi-section form with Alpine.js for dynamic show/hide based on DC type selection. Use `x-data` to track selected type's flags (from hidden data attributes) and conditionally show pricing columns, multi-destination section, etc.

**Routes:**
```go
se.Router.GET("/dcs/create", handlers.HandleDCCreate(app))
se.Router.POST("/dcs", handlers.HandleDCSave(app))
se.Router.GET("/projects/{projectId}/dcs/create", handlers.HandleDCCreate(app))
se.Router.POST("/projects/{projectId}/dcs", handlers.HandleDCSave(app))
```

**Step 2: Run, verify, commit**

```bash
templ generate && go build ./...
git add handlers/dc_create.go templates/dc_create.templ templates/dc_create_templ.go main.go
git commit -m "feat(dc): add DC create form with type selection and address entry"
```

### Task 17: DC Line Items — HTMX Partial Handlers

**Files:**
- Create: `handlers/dc_line_items.go`
- Create: `templates/dc_line_item_row.templ`
- Modify: `main.go` (add routes)

**Handler logic:**
- `HandleDCAddLineItem(app)` — POST, creates dc_line_items record, returns new row HTML
- `HandleDCUpdateLineItem(app)` — PATCH, updates fields
- `HandleDCRemoveLineItem(app)` — DELETE, removes and returns empty response

Line item row template renders one table row with editable fields. For priced DCs, includes rate/taxable/GST/total columns. For unpriced DCs, includes serial_numbers and remarks columns.

**Routes:**
```go
se.Router.POST("/dcs/{id}/line-items", handlers.HandleDCAddLineItem(app))
se.Router.PATCH("/dcs/{id}/line-items/{itemId}", handlers.HandleDCUpdateLineItem(app))
se.Router.DELETE("/dcs/{id}/line-items/{itemId}", handlers.HandleDCRemoveLineItem(app))
```

**Step: Run, verify, commit**

```bash
templ generate && go build ./...
git add handlers/dc_line_items.go templates/dc_line_item_row.templ templates/dc_line_item_row_templ.go main.go
git commit -m "feat(dc): add DC line item CRUD handlers with HTMX partials"
```

### Task 18: BOQ Item Lookup — HTMX Endpoint

**Files:**
- Create: `handlers/dc_boq_lookup.go`
- Create: `templates/dc_boq_lookup.templ`
- Modify: `main.go` (add route)

**Handler logic (`HandleBOQItemsLookup`):**
- GET with query param `q` for search text
- Fetch all BOQ items (main, sub, sub_sub) for the project
- For each, compute dispatched qty and available qty using `GetBOQItemDispatchedQty`
- Filter by search text (description match)
- Return HTMX partial with selectable rows showing: description, UOM, HSN, total qty, dispatched, available

**Route:**
```go
se.Router.GET("/projects/{projectId}/boq-items-lookup", handlers.HandleBOQItemsLookup(app))
```

**Step: Run, verify, commit**

```bash
templ generate && go build ./...
git add handlers/dc_boq_lookup.go templates/dc_boq_lookup.templ templates/dc_boq_lookup_templ.go main.go
git commit -m "feat(dc): add BOQ item lookup endpoint with dispatch tracking"
```

### Task 19: DC Edit — Handler

**Files:**
- Create: `handlers/dc_edit.go`
- Modify: `main.go` (add routes)

**Handler logic:**
- `HandleDCEdit(app)` — GET, loads existing DC data into the create form template (reuse `dc_create.templ` with pre-filled data)
- `HandleDCUpdate(app)` — POST, validates, updates record

Only allow editing DCs in "draft" status. If not draft, redirect to view with error toast.

**Routes:**
```go
se.Router.GET("/dcs/{id}/edit", handlers.HandleDCEdit(app))
se.Router.POST("/dcs/{id}/save", handlers.HandleDCUpdate(app))
```

**Step: Run, verify, commit**

```bash
git add handlers/dc_edit.go main.go
git commit -m "feat(dc): add DC edit handler (draft status only)"
```

### Task 20: DC Delete — Handler

**Files:**
- Create: `handlers/dc_delete.go`
- Modify: `main.go` (add route)

**Handler logic:**
- Only allow deleting DCs in "draft" status
- Delete cascades to line items, destinations, destination items (via PocketBase CascadeDelete)

**Route:**
```go
se.Router.DELETE("/dcs/{id}", handlers.HandleDCDelete(app))
```

**Step: Run, verify, commit**

```bash
git add handlers/dc_delete.go main.go
git commit -m "feat(dc): add DC delete handler (draft only)"
```

---

## Phase 6: Navigation & Sidebar Integration

### Task 21: Add DC Count to Sidebar

**Files:**
- Modify: `templates/sidebar.templ` — add DCCount field to SidebarData, add DC link in project section and global section
- Modify: `handlers/sidebar_helpers.go` — count DCs in BuildSidebarData

**Step 1: Add DCCount to SidebarData**

In `templates/sidebar.templ`, add to the `SidebarData` struct:
```go
DCCount int
```

**Step 2: Add DC sidebar link in `SidebarProjectSection`**

After the PURCHASE ORDERS `@SidebarSubLink(...)`, add:
```go
<!-- Delivery Challans Link -->
@SidebarSubLink(
    fmt.Sprintf("/projects/%s/dcs", data.ActiveProject.ID),
    "DELIVERY CHALLANS",
    isPathActive(data.ActivePath, fmt.Sprintf("/projects/%s/dcs", data.ActiveProject.ID)),
    data.DCCount,
)
```

**Step 3: Add global DC link**

In `SidebarWithProject`, add a global "DELIVERY CHALLANS" link before "ALL VENDORS" (in the `if data.ActiveProject != nil` block) and before "VENDORS" (in the else block). Use a truck/delivery SVG icon.

Also add an `isDCGlobalPath` helper:
```go
func isDCGlobalPath(currentPath string) bool {
    return currentPath == "/dcs" || strings.HasPrefix(currentPath, "/dcs/")
}
```

**Step 4: Count DCs in BuildSidebarData**

In `handlers/sidebar_helpers.go`, add after PO count:
```go
// Count DCs for this project
dcRecords, _ := app.FindRecordsByFilter("delivery_challans", "project = {:pid}", "", 0, 0, map[string]any{"pid": activeProj.ID})
data.DCCount = len(dcRecords)
```

**Step 5: Run templ generate and verify**

Run: `templ generate && go build ./...`

**Step 6: Commit**

```bash
git add templates/sidebar.templ templates/sidebar_templ.go handlers/sidebar_helpers.go
git commit -m "feat(dc): integrate DC count and links in sidebar navigation"
```

---

## Phase 7: PDF Export

### Task 22: DC Export Data Assembly Service

**Files:**
- Create: `services/dc_export_data.go`

**Step 1: Write the export data struct and builder**

Follow the pattern from `services/po_export_data.go`. Create:

```go
type DCExportData struct {
    // Company branding
    CompanyName  string
    LogoBytes    []byte
    LogoFilename string

    // DC Header
    DCNumber, DCDate, Status string
    TypeName, TypeCode string
    IsPriced bool
    RefDCNumber string
    PONumber, PODate string

    // Addresses
    BillFrom, BillTo, DispatchFrom, ShipTo *DCExportAddress

    // Transport
    TransporterName, VehicleNumber string
    EwayBillNumber, DocketNumber string
    ReverseCharge bool

    // Line Items
    LineItems []DCExportLineItem

    // Totals (priced only)
    TotalTaxable, TotalCGST, TotalSGST float64
    RoundOff, InvoiceValue float64
    AmountInWords string

    // Destinations
    Destinations []DCExportDestination
    DestinationItems map[string]map[string]float64 // destID -> lineItemID -> qty

    // Serial Numbers
    HasSerialNumbers bool
    SerialNumberItems []DCExportSerialItem
}
```

Builder function `BuildDCExportData(app, dcId)` fetches all records and assembles the struct.

**Step 2: Run and verify**

Run: `go build ./...`

**Step 3: Commit**

```bash
git add services/dc_export_data.go
git commit -m "feat(dc): add DC export data assembly service"
```

### Task 23: DC PDF Generation Service

**Files:**
- Create: `services/dc_export_pdf.go`

**Step 1: Write the PDF generation**

Follow the pattern from `services/po_export_pdf.go` using maroto/v2. Key sections:

1. `GenerateDCPDF(data *DCExportData) ([]byte, error)` — main entry point
2. `addDCHeader(m, data)` — company logo, DC number, date, transport details
3. `addDCAddresses(m, data)` — 4-box layout (Bill From, Bill To, Dispatch From, Ship To)
4. `addDCLineItemsTable(m, data)` — priced table (if `IsPriced`) or unpriced table
5. `addDCTotals(m, data)` — taxable, CGST, SGST, round off, invoice value (priced only)
6. `addDCAmountInWords(m, data)` — amount in words (priced only)
7. `addDCCertification(m, data)` — "It is certified..." statement
8. `addDCSignatures(m)` — receiver + authorized signatory blocks
9. `addDCDestinationAnnexure(m, data)` — Annexure 1 for multi-destination DCs
10. `addDCSerialNumberAnnexure(m, data)` — Annexure 2 if serial numbers present

**Step 2: Run and verify**

Run: `go build ./...`

**Step 3: Commit**

```bash
git add services/dc_export_pdf.go
git commit -m "feat(dc): add DC PDF generation using maroto/v2"
```

### Task 24: DC Export PDF Handler

**Files:**
- Create: `handlers/dc_export_pdf.go`
- Modify: `main.go` (add route)

**Handler logic:**
- Call `BuildDCExportData(app, id)`
- Call `GenerateDCPDF(data)`
- Set content headers: `Content-Type: application/pdf`, `Content-Disposition: attachment; filename=DC_{number}.pdf`
- Write PDF bytes to response

Follow the pattern from the PO export handler.

**Route:**
```go
se.Router.GET("/dcs/{id}/export/pdf", handlers.HandleDCExportPDF(app))
```

**Step: Run, verify, commit**

```bash
git add handlers/dc_export_pdf.go main.go
git commit -m "feat(dc): add DC PDF export handler"
```

---

## Phase 8: Multi-Destination Support

### Task 25: DC Destinations — HTMX Handlers & Template

**Files:**
- Create: `handlers/dc_destinations.go`
- Create: `templates/dc_destination.templ`
- Modify: `main.go` (add routes)

**Handler logic:**
- `HandleDCAddDestination(app)` — POST, creates dc_destinations record + dc_destination_items for each line item
- `HandleDCUpdateDestination(app)` — PATCH, updates destination details and qty allocations
- `HandleDCRemoveDestination(app)` — DELETE

Template renders a destination row with address fields and qty inputs per line item.

Validation: sum of destination qtys per line item must equal the line item's total qty.

**Routes:**
```go
se.Router.POST("/dcs/{id}/destinations", handlers.HandleDCAddDestination(app))
se.Router.PATCH("/dcs/{id}/destinations/{destId}", handlers.HandleDCUpdateDestination(app))
se.Router.DELETE("/dcs/{id}/destinations/{destId}", handlers.HandleDCRemoveDestination(app))
```

**Step: Run, verify, commit**

```bash
templ generate && go build ./...
git add handlers/dc_destinations.go templates/dc_destination.templ templates/dc_destination_templ.go main.go
git commit -m "feat(dc): add multi-destination HTMX handlers and template"
```

---

## Phase 9: Integration Testing & Polish

### Task 26: Integration Tests for DC Handlers

**Files:**
- Create: `handlers/dc_list_test.go`
- Create: `handlers/dc_create_test.go`

**Step 1: Write tests**

Use `testhelpers.NewTestApp(t)` to create a test PocketBase instance. Test:
1. DC list returns 200 with empty list
2. DC create form renders with DC type dropdown
3. DC save creates a record with auto-generated number
4. DC view returns correct data
5. DC status transition works (draft → finalized)
6. DC delete works for draft, rejects for non-draft

**Step 2: Run tests**

Run: `go test ./handlers/ -run TestDC -v`

**Step 3: Commit**

```bash
git add handlers/dc_list_test.go handlers/dc_create_test.go
git commit -m "test(dc): add integration tests for DC handlers"
```

### Task 27: Service Unit Tests

**Files:**
- Create: `services/dc_boq_tracker_test.go`

**Step 1: Write tests for BOQ tracker**

Test `CheckOverDispatch` scenarios:
- Normal dispatch (qty within available)
- Exact match (qty equals available)
- Over-dispatch (qty exceeds available)

**Step 2: Run tests**

Run: `go test ./services/ -run TestCheckOverDispatch -v`

**Step 3: Commit**

```bash
git add services/dc_boq_tracker_test.go
git commit -m "test(dc): add unit tests for BOQ dispatch tracker"
```

### Task 28: End-to-End Smoke Test

**Step 1: Start server and manually verify**

Run: `make run` or `./restart.sh`

Verify:
1. Navigate to Settings → DC Types — see 5 seeded types
2. Navigate to Settings → DC Series — see series with OAVS prefix
3. Navigate to global /dcs — empty list renders
4. Click "Create DC" — form renders with type dropdown
5. Select a type, fill minimum fields, save — DC created with auto-number
6. View DC — all fields render correctly
7. Export PDF — PDF downloads with correct layout
8. Navigate within project → Delivery Challans — project-scoped view works
9. Create project DC → "Pull from BOQ" shows BOQ items
10. Sidebar shows DC count

**Step 2: Fix any issues found**

**Step 3: Final commit**

```bash
git add -A
git commit -m "feat(dc): delivery challans feature complete — all phases implemented"
```

---

## Task Dependency Summary

```
Phase 1 (Tasks 1-6): Data Layer — sequential, each builds on previous
Phase 2 (Tasks 7-9): Services — can be parallel (independent services)
Phase 3 (Tasks 10-12): Settings UI — can be parallel (independent pages)
Phase 4 (Tasks 13-15): List & View — sequential (view depends on list routes)
Phase 5 (Tasks 16-20): Create & Edit — sequential (edit reuses create template)
Phase 6 (Task 21): Sidebar — depends on Phase 4 (needs routes registered)
Phase 7 (Tasks 22-24): PDF Export — depends on Phase 2 (needs services)
Phase 8 (Task 25): Multi-Destination — depends on Phase 5 (needs create form)
Phase 9 (Tasks 26-28): Testing — depends on all previous phases
```

## Key Reference Files

| Purpose | File |
|---------|------|
| Collection patterns | `collections/setup.go` |
| Seed data patterns | `collections/seed.go` |
| Route registration | `main.go` |
| Handler pattern (view) | `handlers/po_view.go` |
| Handler pattern (settings) | `handlers/app_settings.go` |
| Sidebar integration | `templates/sidebar.templ`, `handlers/sidebar_helpers.go` |
| Middleware (context) | `handlers/middleware.go` |
| PDF generation | `services/po_export_pdf.go` |
| Export data assembly | `services/po_export_data.go` |
| Calculations | `services/export_data.go` (CalcPOLineItem, CalcPOTotals) |
| Template conventions | `.claude/rules/template-conventions.md` |
| Handler conventions | `.claude/rules/handler-conventions.md` |
| Sample DC PDFs | `DCs/DC_OAVS-STDC-2526-002 (11).pdf`, `DCs/DC_OAVS-TDC-2526-013 (4).pdf`, `DCs/DC_OAVS-ODC-2526-020 (2).pdf` |
