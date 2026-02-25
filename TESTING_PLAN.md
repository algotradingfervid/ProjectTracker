# Comprehensive Testing Plan

**Goal**: Maximize test coverage across unit, integration, and E2E layers.
**Current coverage**: 6.9% overall (handlers 25.2%, services 34.4%)
**Target coverage**: 70%+ overall, 80%+ for services, 60%+ for handlers

---

## Phase 1: Unit Tests (services/) — Pure Logic, No DB

These test pure functions with no PocketBase dependency. Fastest to write, highest value.

### 1.1 Address Validation (`services/address_validation_test.go`) — NEW
| Test | What it covers |
|------|---------------|
| `TestValidateGSTIN_Valid` | Valid 15-char GSTIN patterns |
| `TestValidateGSTIN_Invalid` | Empty, short, bad chars, wrong length |
| `TestValidatePAN_Valid` | Valid 10-char PAN |
| `TestValidatePAN_Invalid` | Wrong length, bad format |
| `TestValidatePINCode_Valid` | 6-digit codes |
| `TestValidatePINCode_Invalid` | Letters, too short/long |
| `TestValidatePhone_Valid` | 10-digit numbers |
| `TestValidatePhone_Invalid` | Letters, wrong length |
| `TestValidateEmail_Valid` | Standard email formats |
| `TestValidateEmail_Invalid` | Missing @, no domain |
| `TestValidateCIN_Valid` | Company registration numbers |
| `TestValidateCIN_Invalid` | Wrong format |
| `TestValidateAddressFormat` | Field-level validation rules (all field combos) |

### 1.2 CSV Import Validation (`services/csv_import_test.go`) — NEW
| Test | What it covers |
|------|---------------|
| `TestValidateAddressFile_ValidCSV` | Parses well-formed CSV, returns records |
| `TestValidateAddressFile_MissingHeaders` | Required headers missing → error |
| `TestValidateAddressFile_EmptyFile` | Empty file → error |
| `TestValidateAddressFile_InvalidRows` | Rows with bad GSTIN/PAN → validation errors per row |
| `TestValidateAddressFile_MixedValid` | Some valid, some invalid → both lists populated |
| `TestValidateAddressFile_ShipTo` | Ship-to specific fields parsed correctly |
| `TestValidateAddressFile_InstallAt` | Install-at specific fields parsed correctly |
| `TestGenerateErrorReport_WithErrors` | Returns valid Excel bytes |
| `TestGenerateErrorReport_NoErrors` | Empty error list → still valid Excel |

### 1.3 Excel Export (`services/export_excel_test.go`) — NEW
| Test | What it covers |
|------|---------------|
| `TestGenerateExcel_BasicBOQ` | Valid Excel bytes with items |
| `TestGenerateExcel_EmptyItems` | No items → valid Excel with headers only |
| `TestGenerateExcel_SubItemHierarchy` | Items with sub-items render correctly |
| `TestGenerateExcel_SubSubItemHierarchy` | Full 3-level hierarchy |
| `TestGenerateExcel_SpecialChars` | Description with special chars doesn't break |

### 1.4 PDF Export (`services/export_pdf_test.go`) — NEW
| Test | What it covers |
|------|---------------|
| `TestGeneratePDF_BasicBOQ` | Returns non-empty PDF bytes |
| `TestGeneratePDF_EmptyItems` | No items → valid PDF |
| `TestGeneratePDF_LargeDataset` | 100+ items doesn't crash |

### 1.5 Address Export (`services/address_export_test.go`) — NEW
| Test | What it covers |
|------|---------------|
| `TestGetAddressColumns_AllTypes` | Each address type returns correct columns |
| `TestGenerateAddressExcel_WithData` | Valid Excel bytes |
| `TestGenerateAddressExcel_EmptyData` | No records → valid Excel with headers |

### 1.6 Address Fields (`services/address_fields_test.go`) — NEW
| Test | What it covers |
|------|---------------|
| `TestShipToTemplateFields` | Returns expected field list |
| `TestInstallAtTemplateFields` | Returns expected field list |

### 1.7 PO PDF Export (`services/po_export_pdf_test.go`) — NEW
| Test | What it covers |
|------|---------------|
| `TestGeneratePOPDF_Complete` | Full PO with line items, vendor, addresses → valid PDF |
| `TestGeneratePOPDF_EmptyLineItems` | No items → valid PDF |
| `TestGeneratePOPDF_NilAddresses` | Missing addresses handled gracefully |
| `TestGeneratePOPDF_GSTCalculations` | GST totals in PDF match expected |

### 1.8 Dropdowns (`services/dropdowns_test.go`) — NEW
| Test | What it covers |
|------|---------------|
| `TestDropdownValues` | All dropdown helpers return non-empty slices |

**Phase 1 estimated new tests: ~50 test functions**
**Expected services/ coverage: 75-85%**

---

## Phase 2: Integration Tests (handlers/) — With Test PocketBase

These use `testhelpers.NewTestApp(t)` for a real PocketBase instance with temp DB.

### 2.1 Project Handlers — EXPAND existing

#### `handlers/project_edit_test.go` — NEW
| Test | What it covers |
|------|---------------|
| `TestProjectEdit_GETForm` | Renders edit form with pre-filled data |
| `TestProjectEdit_GETNotFound` | Non-existent ID → error |
| `TestProjectUpdate_ValidSave` | POST updates project name |
| `TestProjectUpdate_DuplicateName` | POST with existing name → validation error |
| `TestProjectUpdate_EmptyName` | POST empty name → validation error |

#### `handlers/project_list_test.go` — NEW
| Test | What it covers |
|------|---------------|
| `TestProjectList_Empty` | No projects → empty state message |
| `TestProjectList_Multiple` | Shows all projects |
| `TestProjectList_HTMXPartial` | HX-Request header → partial content only |
| `TestProjectList_FullPage` | No HX-Request → full page with layout |

#### `handlers/project_view_test.go` — NEW
| Test | What it covers |
|------|---------------|
| `TestProjectView_Exists` | Shows project details |
| `TestProjectView_NotFound` | 404 for non-existent ID |

#### `handlers/project_delete_test.go` — NEW
| Test | What it covers |
|------|---------------|
| `TestProjectDelete_Success` | Deletes project, returns redirect |
| `TestProjectDelete_NotFound` | Non-existent ID → error |
| `TestProjectDelete_CascadesBOQs` | Deleting project removes its BOQs |

#### `handlers/project_settings_test.go` — NEW
| Test | What it covers |
|------|---------------|
| `TestProjectSettings_GETForm` | Renders settings form |
| `TestProjectSettings_Save` | Updates ship_to_equals_install_at flag |
| `TestProjectSettingsAddressRequirements` | Saves required field config per type |

#### `handlers/project_activate_test.go` — NEW
| Test | What it covers |
|------|---------------|
| `TestProjectActivate_Success` | Sets active project cookie/session |
| `TestProjectDeactivate_Success` | Clears active project |

### 2.2 BOQ Handlers — ALL NEW

#### `handlers/boq_create_test.go` — NEW
| Test | What it covers |
|------|---------------|
| `TestBOQCreate_GETForm` | Renders create form within project context |
| `TestBOQSave_Valid` | POST creates BOQ linked to project |
| `TestBOQSave_MissingTitle` | Validation error for empty title |
| `TestBOQSave_DuplicateRefNumber` | Duplicate reference_number → error |

#### `handlers/boq_list_test.go` — NEW
| Test | What it covers |
|------|---------------|
| `TestBOQList_WithItems` | Shows BOQs for project |
| `TestBOQList_Empty` | Empty state message |
| `TestBOQList_OtherProject` | Only shows BOQs for current project |

#### `handlers/boq_view_test.go` — NEW
| Test | What it covers |
|------|---------------|
| `TestBOQView_WithItems` | Shows BOQ with main items and pricing |
| `TestBOQView_Empty` | BOQ with no items |
| `TestBOQView_PricingCalculations` | Totals match expected values |

#### `handlers/boq_edit_test.go` — NEW
| Test | What it covers |
|------|---------------|
| `TestBOQEdit_GETForm` | Renders edit form with items |
| `TestBOQUpdate_SaveItems` | POST updates item descriptions/quantities |
| `TestBOQUpdate_AddAndRemoveItems` | Complex save with hierarchy changes |
| `TestBOQViewMode_Switch` | GET view mode renders read-only |

#### `handlers/boq_delete_test.go` — NEW
| Test | What it covers |
|------|---------------|
| `TestBOQDelete_Success` | Deletes BOQ and cascades items |
| `TestBOQDelete_NotFound` | Non-existent ID → error |

#### `handlers/boq_export_test.go` — NEW
| Test | What it covers |
|------|---------------|
| `TestBOQExportExcel_Success` | Returns Excel content-type and bytes |
| `TestBOQExportPDF_Success` | Returns PDF content-type and bytes |
| `TestBOQExport_EmptyBOQ` | Export with no items → valid file |

### 2.3 BOQ Item Handlers — ALL NEW

#### `handlers/items_test.go` — NEW
| Test | What it covers |
|------|---------------|
| `TestAddMainItem_Success` | Creates main item under BOQ |
| `TestAddSubItem_Success` | Creates sub item under main item |
| `TestAddSubSubItem_Success` | Creates sub-sub item under sub item |
| `TestDeleteMainItem_Success` | Deletes main item + cascades |
| `TestDeleteSubItem_Success` | Deletes sub item + cascades |
| `TestDeleteSubSubItem_Success` | Deletes sub-sub item |
| `TestDeleteItem_NotFound` | Non-existent item → error |
| `TestExpandMainItem` | Lazy-loads sub items for HTMX |

#### `handlers/items_patch_test.go` — NEW
| Test | What it covers |
|------|---------------|
| `TestPatchMainItem_Description` | Updates description field |
| `TestPatchMainItem_Quantity` | Updates qty field |
| `TestPatchMainItem_UnitPrice` | Updates unit_price field |
| `TestPatchSubItem_QtyPerUnit` | Updates qty_per_unit |
| `TestPatchSubSubItem_UnitPrice` | Updates unit_price |
| `TestPatchItem_NotFound` | Non-existent item → error |
| `TestPatchItem_InvalidField` | Unknown field name → error or no-op |

### 2.4 Address Handlers — ALL NEW

#### `handlers/address_list_test.go` — NEW
| Test | What it covers |
|------|---------------|
| `TestAddressList_ByType` | Filters addresses by type |
| `TestAddressList_Empty` | Empty state per type |
| `TestAddressCount_ByType` | Returns correct count |

#### `handlers/address_crud_test.go` — NEW
| Test | What it covers |
|------|---------------|
| `TestAddressCreate_GETForm` | Renders form with correct type fields |
| `TestAddressSave_Valid` | Creates address with all fields |
| `TestAddressSave_ValidationErrors` | Missing required fields → errors |
| `TestAddressEdit_GETForm` | Pre-fills form with existing data |
| `TestAddressUpdate_Valid` | Updates address fields |
| `TestAddressDelete_Success` | Deletes single address |
| `TestAddressDelete_NotFound` | Non-existent → error |
| `TestAddressBulkDelete_Multiple` | Deletes selected addresses |
| `TestAddressDeleteInfo_ShowsDependents` | Shows linked records before delete |

#### `handlers/address_import_test.go` — NEW
| Test | What it covers |
|------|---------------|
| `TestAddressImportPage_Render` | Shows import form |
| `TestAddressValidate_ValidCSV` | Upload valid CSV → preview |
| `TestAddressValidate_InvalidCSV` | Upload bad CSV → error list |
| `TestAddressImportCommit_Success` | Commits validated records to DB |
| `TestAddressErrorReport_Download` | Returns downloadable error Excel |

#### `handlers/address_export_test.go` — NEW
| Test | What it covers |
|------|---------------|
| `TestAddressExportExcel_WithData` | Returns Excel file |
| `TestAddressTemplateDownload` | Returns template Excel |

### 2.5 Vendor Handlers — EXPAND existing

Additional tests for existing files:

| Test | What it covers |
|------|---------------|
| `TestVendorCreate_DuplicateName` | Duplicate vendor name handling |
| `TestVendorEdit_NotFound` | Non-existent vendor ID |
| `TestVendorUpdate_ValidationErrors` | Missing required fields |
| `TestVendorLink_AlreadyLinked` | Re-linking same vendor |
| `TestVendorUnlink_NotLinked` | Unlinking non-linked vendor |
| `TestVendorList_FilterByProject` | Only linked vendors for project |

### 2.6 PO Handlers — EXPAND existing

Additional tests beyond what exists:

| Test | What it covers |
|------|---------------|
| `TestPOCreate_WithoutActiveProject` | No project context → error/redirect |
| `TestPOSave_AllFields` | All optional fields populated |
| `TestPOView_WithGSTBreakdown` | GST subtotals displayed correctly |
| `TestPOExportPDF_Success` | Returns PDF bytes with correct headers |
| `TestPOBOQPicker_Render` | Modal shows BOQ items for selection |
| `TestPOBOQPicker_FilterByBOQ` | Filters items by selected BOQ |

### 2.7 Middleware & Helpers — EXPAND existing

#### `handlers/middleware_test.go` — NEW
| Test | What it covers |
|------|---------------|
| `TestActiveProjectMiddleware_WithCookie` | Sets project in context |
| `TestActiveProjectMiddleware_NoCookie` | No active project in context |
| `TestActiveProjectMiddleware_InvalidID` | Bad project ID → clears cookie |
| `TestGetActiveProject` | Extracts project from context |
| `TestGetHeaderData_WithProject` | Header data includes project info |
| `TestGetHeaderData_NoProject` | Header data without project |

**Phase 2 estimated new tests: ~80 test functions**
**Expected handlers/ coverage: 55-65%**

---

## Phase 3: Integration Tests (collections/) — DB Schema Verification

### 3.1 Collection Setup (`collections/setup_test.go`) — NEW
| Test | What it covers |
|------|---------------|
| `TestSetup_CreatesAllCollections` | All 11 collections exist after Setup() |
| `TestSetup_Idempotent` | Running Setup() twice doesn't error |
| `TestSetup_ProjectFields` | Projects collection has expected fields |
| `TestSetup_BOQFields` | BOQs collection has expected fields |
| `TestSetup_ItemHierarchy` | Relations configured correctly with cascade |
| `TestSetup_AddressFields` | Address collection has all field types |
| `TestSetup_VendorFields` | Vendor collection schema correct |
| `TestSetup_POFields` | Purchase order collection schema correct |
| `TestSetup_POLineItemFields` | PO line item schema correct |

### 3.2 Migrations (`collections/migrate_test.go`) — NEW
| Test | What it covers |
|------|---------------|
| `TestMigrateDefaultAddressSettings_CreatesDefaults` | Settings created for each type |
| `TestMigrateDefaultAddressSettings_Idempotent` | Second run doesn't duplicate |
| `TestMigrateOrphanBOQs_LinksToProject` | Orphaned BOQs get project |
| `TestMigrateOrphanBOQs_NoOrphans` | No-op when all BOQs have projects |

### 3.3 Seed Data (`collections/seed_test.go`) — NEW
| Test | What it covers |
|------|---------------|
| `TestSeed_CreatesProject` | Sample project exists after seed |
| `TestSeed_CreatesBOQs` | Sample BOQs with items |
| `TestSeed_Idempotent` | Second run doesn't duplicate |

**Phase 3 estimated new tests: ~16 test functions**
**Expected collections/ coverage: 60-70%**

---

## Phase 4: E2E Tests (Playwright) — Full User Workflows

Extend `tests/e2e/` with Playwright specs testing complete user journeys through the browser.

### 4.1 Project Workflows (`tests/e2e/tests/project-crud.spec.ts`) — EXPAND
| Test | What it covers |
|------|---------------|
| `delete a project` | Complete CRUD cycle (add delete to existing) |
| `activate and switch projects` | Project activation via sidebar |
| `project settings page` | Navigate to settings, toggle options |

### 4.2 BOQ Workflows (`tests/e2e/tests/boq-crud.spec.ts`) — NEW
| Test | What it covers |
|------|---------------|
| `create a BOQ` | Navigate to project → create BOQ form → submit |
| `add main item to BOQ` | Open BOQ → edit mode → add item → save |
| `add sub items` | Add sub-items under main item |
| `edit item fields inline` | Patch item description/qty via HTMX |
| `delete items` | Remove items from BOQ |
| `view BOQ with pricing` | Verify pricing totals display correctly |
| `export BOQ to Excel` | Click export → file downloads |
| `export BOQ to PDF` | Click export → file downloads |

### 4.3 Address Workflows (`tests/e2e/tests/address-management.spec.ts`) — NEW
| Test | What it covers |
|------|---------------|
| `create bill-from address` | Fill address form → save → appears in list |
| `edit address` | Edit existing address → verify changes |
| `delete address` | Delete confirmation → remove from list |
| `bulk delete addresses` | Select multiple → bulk delete |
| `switch address types` | Navigate between bill-from, ship-to, etc. |
| `import addresses from CSV` | Upload CSV → validate → commit |
| `export addresses to Excel` | Download Excel file |
| `download address template` | Download import template |

### 4.4 Vendor Workflows (`tests/e2e/tests/vendor-management.spec.ts`) — NEW
| Test | What it covers |
|------|---------------|
| `create vendor` | Fill vendor form → save |
| `edit vendor` | Edit details → save |
| `delete vendor` | Delete → removed from list |
| `link vendor to project` | Link from project context |
| `unlink vendor from project` | Unlink → vendor removed from project list |

### 4.5 Purchase Order Workflows (`tests/e2e/tests/po-management.spec.ts`) — NEW
| Test | What it covers |
|------|---------------|
| `create purchase order` | Select vendor + BOQ → save |
| `add line items manually` | Add items → verify totals |
| `add line items from BOQ` | Open BOQ picker → select items → add |
| `edit PO details` | Update terms, dates |
| `view PO with totals` | Verify GST calculations display |
| `export PO to PDF` | Download PO PDF |
| `delete purchase order` | Delete → removed from list |

### 4.6 Navigation & Layout (`tests/e2e/tests/navigation.spec.ts`) — NEW
| Test | What it covers |
|------|---------------|
| `sidebar navigation` | All sidebar links navigate correctly |
| `HTMX partial loads` | Content swaps without full page reload |
| `direct URL access` | Deep link to /projects/{id}/boq works |
| `browser back/forward` | History navigation works with hx-push-url |
| `responsive sidebar` | Sidebar behavior at different viewports |

**Phase 4 estimated new tests: ~35 E2E specs**

---

## Phase 5: Address Validation Integration (`services/`) — DB Required

These tests need a PocketBase instance because they call `GetRequiredFields()`.

### 5.1 Full Address Validation (`services/address_validation_integration_test.go`) — NEW
| Test | What it covers |
|------|---------------|
| `TestValidateAddress_BillFrom_AllRequired` | All required fields present → no errors |
| `TestValidateAddress_BillFrom_MissingRequired` | Missing required → error map |
| `TestValidateAddress_ShipTo_CustomRequirements` | Project-specific required fields |
| `TestValidateAddress_InvalidGSTIN` | Bad GSTIN format → error |
| `TestValidateAddress_InvalidPAN` | Bad PAN format → error |
| `TestGetRequiredFields_DefaultSettings` | Returns correct defaults per type |
| `TestGetRequiredFields_CustomSettings` | Returns custom project settings |

### 5.2 Address Import Commit (`services/address_import_integration_test.go`) — NEW
| Test | What it covers |
|------|---------------|
| `TestCommitAddressImport_Success` | Batch inserts addresses into DB |
| `TestCommitAddressImport_Empty` | No records → no-op |
| `TestCommitAddressImport_LargeBatch` | 100+ records → all saved |

### 5.3 Address Template Generation (`services/address_template_integration_test.go`) — NEW
| Test | What it covers |
|------|---------------|
| `TestGenerateAddressTemplate_ShipTo` | Template has correct columns |
| `TestGenerateAddressTemplate_InstallAt` | Template has correct columns |
| `TestGenerateAddressTemplate_WithDropdowns` | State dropdown in template |

### 5.4 PO Export Data (`services/po_export_data_integration_test.go`) — NEW
| Test | What it covers |
|------|---------------|
| `TestBuildPOExportData_Complete` | Assembles all PO data from DB |
| `TestBuildPOExportData_MissingVendor` | Handles missing vendor gracefully |
| `TestBuildPOExportData_NoLineItems` | PO with no items |
| `TestBuildPOExportData_WithAddresses` | All address types populated |

**Phase 5 estimated new tests: ~17 test functions**

---

## Implementation Priority & Estimates

| Phase | Type | New Tests | Effort | Coverage Impact |
|-------|------|-----------|--------|-----------------|
| **1** | Unit (services) | ~50 | Medium | services/ → 75-85% |
| **2** | Integration (handlers) | ~80 | Large | handlers/ → 55-65% |
| **3** | Integration (collections) | ~16 | Small | collections/ → 60-70% |
| **4** | E2E (Playwright) | ~35 | Large | Not in Go coverage, but validates full stack |
| **5** | Integration (services+DB) | ~17 | Medium | services/ → 85-90% |

**Total new tests: ~198**
**Projected overall Go coverage: 50-60%** (up from 6.9%)
**Projected coverage excluding templates/: 65-75%**

> Note: Templates (`_templ.go`) are generated code and typically not unit-tested directly.
> Their correctness is validated through handler integration tests (render output checks)
> and E2E tests (visual verification).

---

## Test Infrastructure Needed

### Already exists (no changes needed)
- `testhelpers.NewTestApp(t)` — creates PocketBase with temp dir
- `testhelpers.CreateTest*` — builders for all entity types
- `testhelpers.AssertHTMLContains` and `AssertHXRedirect`
- Playwright config with auto-start server

### May need additions
1. **`testhelpers.NewTestRequestEvent`** — standardize creating `core.RequestEvent` with proper App/Request/Response (partially exists in `test_helpers_test.go`)
2. **`testhelpers.CreateTestAddressSettings`** — create project address settings for validation tests
3. **`testhelpers.UploadCSVFile`** — helper to create multipart form with CSV for import tests
4. **E2E fixtures** — `tests/e2e/fixtures/` directory for sample CSV files, expected exports

---

## Execution Order Recommendation

1. **Start with Phase 1** — pure unit tests, no infrastructure needed, fastest feedback
2. **Phase 3 next** — validates DB schema, small scope, builds confidence in test app
3. **Phase 2** — bulk of the work, handler integration tests
4. **Phase 5** — service tests needing DB
5. **Phase 4 last** — E2E tests require running server, slowest to execute
