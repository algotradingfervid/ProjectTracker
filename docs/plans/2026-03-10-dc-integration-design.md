# Delivery Challan (DC) Integration — Approved Design

**Date**: 2026-03-10
**Status**: Approved

---

## Overview

Integrate Delivery Challan management functionality from the standalone DC application (ProjectManagementTool) natively into the ProjectCreation codebase. The logic is rebuilt from scratch using PocketBase collections, existing handler/template conventions, and the established HTMX + Alpine.js + Tailwind/DaisyUI design system.

### Key Decisions

- **No authentication** — skip for now, add later
- **Products derived from BOQ items** — no separate product catalog; DC templates reference sub_items/sub_sub_items
- **Address system restructured** — replace fixed-schema addresses with the DC app's configurable JSON-column approach
- **Full transporter module** — with vehicles, driver details, file uploads
- **Both workflows from day one** — Direct Shipments and Transfer DCs
- **Unified wizard** — single 4-step wizard with type toggle instead of two separate 5-step wizards
- **Essential reports first** — DC list with filters + single DC PDF/Excel export; analytics reports later
- **Configurable numbering** — unified system for PO + DC numbers with format tokens, padding, start number
- **Full serial tracking** — with per-item configurability (none/optional/required)
- **maroto for PDFs** — consistent with existing PO exports, no browser dependency

---

## Section 1: Data Model (PocketBase Collections)

### New Collections

| Collection | Purpose | Key Fields |
|-----------|---------|------------|
| `address_configs` | Per-type column definitions | `project`, `address_type`, `columns` (JSON) |
| `dc_templates` | Reusable BOQ item groupings | `project`, `name`, `purpose` |
| `dc_template_items` | Template ↔ BOQ item mapping | `template`, `source_item_type`, `source_item_id`, `default_quantity` |
| `transporters` | Transport companies | `project`, `company_name`, `contact_person`, `phone`, `gst_number`, `is_active` |
| `transporter_vehicles` | Vehicles per transporter | `transporter`, `vehicle_number`, `vehicle_type`, `driver_name`, `driver_phone`, `rc_image`, `driver_license` |
| `delivery_challans` | Core DC records (all types) | `project`, `dc_number`, `dc_type` (transit/official/transfer), `status` (draft/issued/splitting/split), `template`, `bill_from_address`, `dispatch_from_address`, `bill_to_address`, `ship_to_address`, `challan_date`, `shipment_group`, `transfer_dc` |
| `dc_line_items` | Products on a DC | `dc`, `source_item_type`, `source_item_id`, `quantity`, `rate`, `tax_percentage`, `taxable_amount`, `tax_amount`, `total_amount`, `line_order` |
| `serial_numbers` | Serial tracking per line item | `project`, `line_item`, `serial_number` |
| `dc_transit_details` | Transport info for transit DCs | `dc`, `transporter`, `vehicle`, `eway_bill_number`, `docket_number`, `notes` |
| `shipment_groups` | Groups transit + official DCs | `project`, `template`, `num_locations`, `tax_type`, `reverse_charge`, `status`, `transfer_dc`, `split` |
| `transfer_dcs` | Transfer DC metadata | `dc`, `hub_address`, `template`, `tax_type`, `reverse_charge`, `transporter`, `vehicle`, `eway_bill_number`, `docket_number`, `notes`, `num_destinations`, `num_split` |
| `transfer_dc_destinations` | Planned destinations | `transfer_dc`, `ship_to_address`, `split_group`, `is_split` |
| `transfer_dc_dest_quantities` | Qty per product per destination | `destination`, `source_item_type`, `source_item_id`, `quantity` |
| `transfer_dc_splits` | Links transfer DC → child shipments | `transfer_dc`, `shipment_group`, `split_number` |
| `number_sequences` | Unified atomic counter (PO + DC) | `project`, `sequence_type` (po/tdc/odc/stdc), `financial_year`, `last_number` |

### Modified Collections

| Collection | Changes |
|-----------|---------|
| `projects` | Add: `dc_prefix`, `dc_number_format`, `dc_number_separator`, `dc_seq_padding`, `dc_seq_start_po`, `dc_seq_start_tdc`, `dc_seq_start_odc`, `dc_seq_start_stdc`, `default_bill_from`, `default_dispatch_from` |
| `addresses` | Restructured: replace fixed fields with `config` (relation), `address_code`, `data` (JSON), `district_name`, `mandal_name`, `mandal_code` |
| `project_address_settings` | **Removed** — replaced by `address_configs` |

### Serial Tracking Configuration

The `serial_tracking` field (on dc_template_items or derived from BOQ item config) controls per-item behavior:

- `none` — serial field hidden for this item
- `optional` — serial field visible but not required
- `required` — serial count must match quantity

---

## Section 2: Address System Restructure

### Current System (Being Replaced)

- Fixed-schema `addresses` collection with hardcoded fields (company_name, contact_person, gstin, address_line1/2, city, state, pin_code)
- `project_address_settings` with boolean `req_*` fields per address type

### New System

**`address_configs` collection** — one record per address type per project:

```json
{
  "project": "project_id",
  "address_type": "bill_to",
  "columns": [
    { "name": "company_name", "label": "Company Name", "required": true, "type": "text", "fixed": true, "show_in_table": true, "show_in_print": true, "sort_order": 1 },
    { "name": "gstin", "label": "GSTIN", "required": false, "type": "text", "fixed": false, "show_in_table": true, "show_in_print": true, "sort_order": 2 }
  ]
}
```

**`addresses` collection** (restructured):

```json
{
  "config": "config_id",
  "address_code": "ADDR-001",
  "data": { "company_name": "ABC Corp", "gstin": "29XXXXX", "city": "Bangalore" },
  "district_name": "Hyderabad",
  "mandal_name": "Secunderabad",
  "mandal_code": "SEC01"
}
```

**Key behaviors:**
- Each project starts with default column definitions per address type (seeded on project creation)
- Users can add/remove/reorder columns per type via a config UI
- Address forms are dynamically generated from column definitions
- Bulk CSV upload adapts to configured columns
- `ship_to` and `install_at` retain fixed district/mandal fields
- Existing address data migrated from fixed fields to JSON format

**Impact on existing features:**
- PO handlers referencing addresses updated to read from `data` JSON
- Address list/create/edit/import templates rewritten for dynamic columns
- Export services adapted for new structure

---

## Section 3: Unified DC Wizard (4 Steps)

### Step 1: Setup

- DC Type toggle: **Direct Shipment** / **Transfer**
- Select DC Template (shows BOQ items in that template)
- Challan date (defaults to today)
- Transporter picker: dropdown → select vehicle → driver auto-fills
- Eway bill number, docket number (optional)
- Reverse charge toggle (default: No)

### Step 2: Destinations

- **Pre-filled from project settings** (read-only with override toggle):
  - Bill From address
  - Dispatch From address
- Select Bill To address (searchable picker)
- Number of destinations input
- Select Ship To addresses (one per destination, searchable picker)
- **Transfer only**: Select Hub Address
- **Auto-calculated tax type**: bill_from state vs ship_to state → IGST or CGST+SGST badge (override available)

### Step 3: Items & Serials (Combined)

Quantity grid: rows = template items, columns = destinations

| Item | HSN | UoM | Dest 1 | Dest 2 | Total |
|------|-----|-----|--------|--------|-------|
| Cable 4mm | 8544 | m | 100 | 50 | 150 |
| Switch 16A | 8536 | nos | 0 | 25 | 25 |

Below each item row (based on `serial_tracking`):
- **required**: expandable serial textarea, live count badge, validation errors if count != total qty
- **optional**: expandable serial textarea, no count enforcement
- **none**: serial section hidden

Real-time validation: duplicates within input, duplicates across project DB.

### Step 4: Review & Confirm

- Collapsible sections: Addresses, Transport, Items + pricing + tax, Serials
- Each section has "Edit" link → jumps back to that step
- Grand totals: taxable + tax + total
- Confirm → atomic creation

### What Gets Created

| DC Type | Creates |
|---------|---------|
| Direct Shipment | 1 Shipment Group → 1 Transit DC (all items, serials, pricing, transport) + N Official DCs (per-destination qty, no pricing/serials) |
| Transfer | 1 Transfer DC (all items, serials, pricing, transport) + destination plan with per-destination quantities |

### Post-Creation Flow

- Redirect to DC detail view (draft status)
- Actions: Edit (re-enter wizard), Delete, Issue
- Issue validates serials (required items: count must match qty, no DB duplicates)
- After issue: Print, Export PDF, Export Excel
- Transfer DCs after issue: Split wizard available

---

## Section 4: Split Wizard (Transfer DCs Only)

3-step wizard for splitting an issued Transfer DC into child shipments.

### Step 1: Select Destinations

- Show all unsplit destinations
- Checkbox selection
- Quantities preview table per destination

### Step 2: Transport & Serials

- Select transporter + vehicle (picker)
- Eway bill, docket number
- Serial assignment per destination per item (from parent's serial pool)
- Validation: count per item per destination must match quantity

### Step 3: Review & Confirm

- Summary of destinations, transport, serials
- Confirm → create child Shipment Group (1 Transit DC + N Official DCs)
- Mark destinations as `is_split=1`, increment `num_split`

### Status Transitions

```
Draft → (Issue) → Issued → (First split) → Splitting → (All split) → Split
                     ↑                           |
                     └── (Undo all splits) ──────┘
```

Status computation:
- `status == draft` → DRAFT
- `status == issued && num_split == 0` → ISSUED
- `num_split > 0 && num_split < num_destinations` → SPLITTING
- `num_split >= num_destinations` → SPLIT

---

## Section 5: Navigation & Sidebar

### Sidebar Structure (Project Active)

```
PROJECT DETAILS
  Overview
  BOQ
  Purchase Orders
  ─────────────
  DC MANAGEMENT
    DC Templates  (3)
    Transporters  (2)
    Delivery Challans  (12)
  ─────────────
  Vendors
  Addresses  ▸
    Bill From  (1)
    Dispatch From  (1)
    Bill To  (3)
    Ship To  (5)
    Install At  (2)
```

### Routes

```
/projects/{projectId}/dc-templates/              GET    list
/projects/{projectId}/dc-templates/create         GET    create form
/projects/{projectId}/dc-templates/create         POST   create
/projects/{projectId}/dc-templates/{id}           GET    detail
/projects/{projectId}/dc-templates/{id}/edit      GET    edit form
/projects/{projectId}/dc-templates/{id}/edit      POST   update
/projects/{projectId}/dc-templates/{id}/delete    POST   delete
/projects/{projectId}/dc-templates/{id}/duplicate POST   duplicate

/projects/{projectId}/transporters/               GET    list
/projects/{projectId}/transporters/create         GET    create form
/projects/{projectId}/transporters/create         POST   create
/projects/{projectId}/transporters/{id}           GET    detail
/projects/{projectId}/transporters/{id}/edit      POST   update
/projects/{projectId}/transporters/{id}/toggle    POST   toggle active
/projects/{projectId}/transporters/{id}/vehicles  POST   add vehicle
/projects/{projectId}/transporters/{id}/vehicles/{vid}  DELETE  remove vehicle

/projects/{projectId}/dcs/                        GET    list (filterable by type/status)
/projects/{projectId}/dcs/create                  GET    wizard step 1
/projects/{projectId}/dcs/create/step2            POST   wizard step 2
/projects/{projectId}/dcs/create/step3            POST   wizard step 3
/projects/{projectId}/dcs/create/step4            POST   wizard step 4
/projects/{projectId}/dcs/create                  POST   create DC
/projects/{projectId}/dcs/create/back-to-step{n}  POST   back navigation
/projects/{projectId}/dcs/{id}                    GET    detail
/projects/{projectId}/dcs/{id}/edit               GET    edit wizard
/projects/{projectId}/dcs/{id}/issue              POST   issue
/projects/{projectId}/dcs/{id}/delete             POST   delete (draft only)
/projects/{projectId}/dcs/{id}/print              GET    HTML print view
/projects/{projectId}/dcs/{id}/export/pdf         GET    PDF download
/projects/{projectId}/dcs/{id}/export/excel       GET    Excel download

/projects/{projectId}/shipment-groups/{id}        GET    group detail
/projects/{projectId}/shipment-groups/{id}/export/pdf  GET  merged PDF

/projects/{projectId}/transfer-dcs/{id}/split              GET    split wizard step 1
/projects/{projectId}/transfer-dcs/{id}/split/step2        POST   split step 2
/projects/{projectId}/transfer-dcs/{id}/split/step3        POST   split step 3
/projects/{projectId}/transfer-dcs/{id}/split              POST   create split
/projects/{projectId}/transfer-dcs/{id}/splits/{sid}/undo  POST   undo split
```

### SidebarData Additions

- `DCTemplateCount` (int)
- `TransporterCount` (int)
- `DCCount` (int)

### Project Settings Additions

- DC Prefix field
- DC Number Format configuration
- Separator, padding, start numbers per type
- Default Bill From / Dispatch From address pickers
- Live DC number preview

---

## Section 6: Configurable Numbering System

### `number_sequences` Collection

- `project` (relation)
- `sequence_type` (select: `po`, `tdc`, `odc`, `stdc`)
- `financial_year` (text: "2526")
- `last_number` (number)

### Project Settings Fields

- `number_format` — format template, e.g. `{PREFIX}{SEP}{TYPE}{SEP}{FY}{SEP}{SEQ}`
- `number_separator` — e.g. `-`, `/`
- `number_prefix` — e.g. `FSS`, `ABC`
- `seq_padding` — digits in sequence (3 → `001`, 4 → `0001`)
- `seq_start_po`, `seq_start_tdc`, `seq_start_odc`, `seq_start_stdc` — starting number per type

### Format Tokens

| Token | Resolves To | Example |
|-------|------------|---------|
| `{PREFIX}` | Project's `number_prefix` | `FSS` |
| `{TYPE}` | Sequence type code | `PO`, `TDC`, `ODC`, `STDC` |
| `{FY}` | Indian financial year | `2526` |
| `{SEQ}` | Zero-padded sequence | `001` |
| `{SEP}` | Configured separator | `-` |
| `{PROJECT_REF}` | Project's `reference_number` | `OAVS` |

### Examples

Format: `{PREFIX}{SEP}{TYPE}{SEP}{FY}{SEP}{SEQ}`

- PO: `FSS-PO-2526-001`
- Transit DC: `FSS-TDC-2526-001`
- Official DC: `FSS-ODC-2526-003`
- Transfer DC: `FSS-STDC-2526-001`

### Financial Year

Indian Apr-Mar. Compact format: `2526` = April 2025 – March 2026. Auto-calculated from document date.

### Sequence Logic

1. Find or create `number_sequences` record for (project, type, FY)
2. If new record, initialize `last_number` to `seq_start_{type} - 1`
3. Increment `last_number` atomically
4. Format using project's format template
5. Existing PO number generation migrates to this system

---

## Section 7: Export & Print

### PDF Export (maroto)

| Document | Content |
|----------|---------|
| Transit DC | Addresses, all items with pricing + tax, all serials, transporter details, company signature area |
| Official DC | Per-destination addresses, destination-specific items + quantities, no pricing, no serials |
| Transfer DC | Hub address, all items with pricing + serials, transporter, destination summary |
| Shipment Group | Merged PDF — Transit DC page + all Official DC pages |

### Excel Export (excelize)

| Document | Content |
|----------|---------|
| Single DC | Sheet 1: DC details + addresses. Sheet 2: Line items + pricing/tax. Sheet 3: Serial numbers |
| Shipment Group | Sheet per DC in the group |

### Print Views (HTML via templ)

- Same content as PDF, rendered as HTML for browser print
- Route: `/projects/{projectId}/dcs/{id}/print`
- Opens in new tab, styled for print media

### Company Branding

- Company name and logo from `app_settings` (existing)
- Signature area as placeholder on PDFs

---

## Section 8: Migration Strategy

### Address System Migration

1. **Create `address_configs`** — seed default column definitions per existing address type per project, matching current fixed fields:
   - company_name, contact_person, gstin, phone, email, address_line1, address_line2, city, state, pin_code

2. **Migrate `addresses` data** — for each existing record:
   - Read fixed fields → write into `data` JSON
   - Set `address_code` from record ID or company_name slug
   - Link to new `address_configs` via `config` relation

3. **Drop `project_address_settings`** — replaced by `address_configs.columns[].required`

4. **Update existing handlers** — PO view/edit, address list/create/edit, address import/export

### PO Number Migration

- Existing `services/po_number.go` migrates to unified `number_sequences` collection
- Existing PO numbers remain unchanged
- Initialize `number_sequences` for PO type with `last_number` = highest existing sequence per project per FY

### No Data Loss

All existing address data preserved, restructured from fixed fields to JSON format.
