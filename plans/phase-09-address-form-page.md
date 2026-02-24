# Phase 9: Address Form Page (Create & Edit)

## Overview & Objectives

Create a dedicated full-page form for creating and editing addresses. The form is shared between create and edit modes, organized into logical sections (Company Info, Address Details, Contact Info, Tax/Legal Info), with dynamic required field indicators driven by per-project `project_address_settings`, server-side validation with inline error messages, and proper navigation back to the address list.

### Key Goals

- Separate page for add/edit (not modal, not inline) -- consistent with existing BOQ create pattern
- Single `address_form.templ` template reused for both create and edit
- Handlers: `HandleAddressCreate` (GET form + POST save) and `HandleAddressEdit` (GET form + PATCH update)
- Dynamic required fields based on `project_address_settings` per project per address type
- Server-side validation: GSTIN (15-char alphanumeric), PAN (10-char alphanumeric), PIN Code (6 digits), Email format, Phone (10 digits)
- Breadcrumbs: Projects / {Project Name} / {Address Type} / Add|Edit
- For Install At type: optional "Ship To Parent" dropdown

---

## Files to Create/Modify

| File | Action | Purpose |
|------|--------|---------|
| `handlers/address_create.go` | **Create** | GET form render + POST save handler for new addresses |
| `handlers/address_edit.go` | **Create** | GET form render + PATCH update handler for editing addresses |
| `templates/address_form.templ` | **Create** | Shared form template for create and edit |
| `services/address_validation.go` | **Create** | Validation logic for address fields |
| `services/indian_states.go` | **Create** | List of Indian states for the State dropdown |
| `main.go` | **Modify** | Register create and edit routes |

---

## Detailed Implementation Steps

### Step 1: Indian States Reference Data

Create `services/indian_states.go` with the full list of Indian states and union territories.

```go
package services

// IndianStates is the list of Indian states and union territories for the State dropdown.
var IndianStates = []string{
    "Andhra Pradesh",
    "Arunachal Pradesh",
    "Assam",
    "Bihar",
    "Chhattisgarh",
    "Goa",
    "Gujarat",
    "Haryana",
    "Himachal Pradesh",
    "Jharkhand",
    "Karnataka",
    "Kerala",
    "Madhya Pradesh",
    "Maharashtra",
    "Manipur",
    "Meghalaya",
    "Mizoram",
    "Nagaland",
    "Odisha",
    "Punjab",
    "Rajasthan",
    "Sikkim",
    "Tamil Nadu",
    "Telangana",
    "Tripura",
    "Uttar Pradesh",
    "Uttarakhand",
    "West Bengal",
    // Union Territories
    "Andaman and Nicobar Islands",
    "Chandigarh",
    "Dadra and Nagar Haveli and Daman and Diu",
    "Delhi",
    "Jammu and Kashmir",
    "Ladakh",
    "Lakshadweep",
    "Puducherry",
}

// Countries is a minimal list with India as default.
var Countries = []string{
    "India",
    "United States",
    "United Kingdom",
    "United Arab Emirates",
    "Singapore",
    "Australia",
    "Canada",
    "Germany",
    "Japan",
    "Other",
}
```

### Step 2: Address Validation Service

Create `services/address_validation.go` with validation rules for all address fields.

```go
package services

import (
    "regexp"
    "strings"
)

// Validation regex patterns
var (
    gstinPattern = regexp.MustCompile(`^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z]{1}[1-9A-Z]{1}Z[0-9A-Z]{1}$`)
    panPattern   = regexp.MustCompile(`^[A-Z]{5}[0-9]{4}[A-Z]{1}$`)
    pinPattern   = regexp.MustCompile(`^[1-9][0-9]{5}$`)
    phonePattern = regexp.MustCompile(`^[6-9][0-9]{9}$`)
    emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
    cinPattern   = regexp.MustCompile(`^[A-Z]{1}[0-9]{5}[A-Z]{2}[0-9]{4}[A-Z]{3}[0-9]{6}$`)
)

// ValidateGSTIN validates a GSTIN (15-character alphanumeric).
// Format: 2 digits + 5 uppercase letters + 4 digits + 1 letter + 1 alphanumeric + Z + 1 alphanumeric
func ValidateGSTIN(gstin string) bool {
    gstin = strings.TrimSpace(strings.ToUpper(gstin))
    if gstin == "" {
        return true // empty is valid (required check is separate)
    }
    return len(gstin) == 15 && gstinPattern.MatchString(gstin)
}

// ValidatePAN validates a PAN number (10-character alphanumeric).
// Format: 5 uppercase letters + 4 digits + 1 uppercase letter
func ValidatePAN(pan string) bool {
    pan = strings.TrimSpace(strings.ToUpper(pan))
    if pan == "" {
        return true
    }
    return len(pan) == 10 && panPattern.MatchString(pan)
}

// ValidatePINCode validates an Indian PIN code (6 digits, first digit non-zero).
func ValidatePINCode(pin string) bool {
    pin = strings.TrimSpace(pin)
    if pin == "" {
        return true
    }
    return len(pin) == 6 && pinPattern.MatchString(pin)
}

// ValidatePhone validates an Indian mobile number (10 digits starting with 6-9).
func ValidatePhone(phone string) bool {
    phone = strings.TrimSpace(phone)
    if phone == "" {
        return true
    }
    return len(phone) == 10 && phonePattern.MatchString(phone)
}

// ValidateEmail validates an email address format.
func ValidateEmail(email string) bool {
    email = strings.TrimSpace(email)
    if email == "" {
        return true
    }
    return emailPattern.MatchString(email)
}

// ValidateCIN validates a Corporate Identity Number (21 characters).
func ValidateCIN(cin string) bool {
    cin = strings.TrimSpace(strings.ToUpper(cin))
    if cin == "" {
        return true
    }
    return len(cin) == 21 && cinPattern.MatchString(cin)
}

// AddressFieldErrors holds field-level error messages.
type AddressFieldErrors map[string]string

// ValidateAddressFields validates all address fields and returns a map of field -> error message.
// requiredFields is a set of field names that are required for this address type in this project.
func ValidateAddressFields(fields map[string]string, requiredFields map[string]bool) AddressFieldErrors {
    errors := make(AddressFieldErrors)

    // Check required fields
    for field, required := range requiredFields {
        if required && strings.TrimSpace(fields[field]) == "" {
            errors[field] = fieldLabel(field) + " is required"
        }
    }

    // Format validations (only if value is non-empty)
    if v := fields["gstin"]; v != "" && !ValidateGSTIN(v) {
        errors["gstin"] = "Invalid GSTIN format (expected: 15-character, e.g., 27AAPFU0939F1ZV)"
    }
    if v := fields["pan"]; v != "" && !ValidatePAN(v) {
        errors["pan"] = "Invalid PAN format (expected: 10-character, e.g., ABCDE1234F)"
    }
    if v := fields["pin_code"]; v != "" && !ValidatePINCode(v) {
        errors["pin_code"] = "Invalid PIN Code (expected: 6 digits, e.g., 400001)"
    }
    if v := fields["phone"]; v != "" && !ValidatePhone(v) {
        errors["phone"] = "Invalid phone number (expected: 10 digits starting with 6-9)"
    }
    if v := fields["email"]; v != "" && !ValidateEmail(v) {
        errors["email"] = "Invalid email format"
    }
    if v := fields["cin"]; v != "" && !ValidateCIN(v) {
        errors["cin"] = "Invalid CIN format (expected: 21-character)"
    }

    return errors
}

// fieldLabel returns a human-readable label for a field name.
func fieldLabel(field string) string {
    labels := map[string]string{
        "company_name":   "Company Name",
        "contact_person": "Contact Person",
        "address_line_1": "Address Line 1",
        "address_line_2": "Address Line 2",
        "city":           "City",
        "state":          "State",
        "pin_code":       "PIN Code",
        "country":        "Country",
        "phone":          "Phone",
        "email":          "Email",
        "gstin":          "GSTIN",
        "pan":            "PAN",
        "cin":            "CIN",
        "website":        "Website",
        "fax":            "Fax",
        "landmark":       "Landmark",
        "district":       "District",
    }
    if l, ok := labels[field]; ok {
        return l
    }
    return field
}
```

### Step 3: Define Template Data Structs

In `templates/address_form.templ`:

```go
package templates

import "fmt"

// AddressFormData holds all data needed to render the address create/edit form.
type AddressFormData struct {
    // Mode
    IsEdit    bool
    AddressID string // empty for create

    // Context
    ProjectID    string
    ProjectName  string
    AddressType  string // slug: "bill_from", "ship_from", etc.
    AddressLabel string // display: "Bill From", "Ship From", etc.

    // Field values (pre-populated for edit, empty for create)
    CompanyName   string
    ContactPerson string
    AddressLine1  string
    AddressLine2  string
    City          string
    State         string
    PinCode       string
    Country       string
    Phone         string
    Email         string
    GSTIN         string
    PAN           string
    CIN           string
    Website       string
    Fax           string
    Landmark      string
    District      string

    // Required field indicators (from project_address_settings)
    RequiredFields map[string]bool

    // Validation errors (field name -> error message)
    Errors map[string]string

    // Dropdown options
    StateOptions   []string
    CountryOptions []string

    // Install At specific: Ship To parent addresses for linking
    ShowShipToParent bool     // true only for install_at when project toggle is OFF
    ShipToAddresses  []ShipToOption
    SelectedShipToID string
}

// ShipToOption represents a Ship To address for the parent dropdown.
type ShipToOption struct {
    ID          string
    CompanyName string
    City        string
}
```

### Step 4: Implement HandleAddressCreate (GET)

Create `handlers/address_create.go`:

```go
package handlers

import (
    "log"

    "github.com/a-h/templ"
    "github.com/pocketbase/pocketbase"
    "github.com/pocketbase/pocketbase/core"

    "projectcreation/services"
    "projectcreation/templates"
)

// HandleAddressCreate returns a handler that renders the address creation form.
func HandleAddressCreate(app *pocketbase.PocketBase, addressType AddressType) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        projectID := e.Request.PathValue("projectId")
        if projectID == "" {
            return e.String(400, "Missing project ID")
        }

        // 1. Verify project exists
        projectRecord, err := app.FindRecordById("projects", projectID)
        if err != nil {
            log.Printf("address_create: could not find project %s: %v", projectID, err)
            return e.String(404, "Project not found")
        }

        // 2. Fetch required fields from project_address_settings
        requiredFields := fetchRequiredFields(app, projectID, addressType)

        // 3. Prepare Ship To parent options for Install At type
        var shipToAddresses []templates.ShipToOption
        var showShipToParent bool
        if addressType == AddressTypeInstallAt {
            showShipToParent, shipToAddresses = fetchShipToOptions(app, projectID)
        }

        // 4. Build form data
        data := templates.AddressFormData{
            IsEdit:         false,
            ProjectID:      projectID,
            ProjectName:    projectRecord.GetString("name"),
            AddressType:    string(addressType),
            AddressLabel:   addressTypeLabel(addressType),
            Country:        "India", // default country
            RequiredFields: requiredFields,
            Errors:         make(map[string]string),
            StateOptions:   services.IndianStates,
            CountryOptions: services.Countries,
            ShowShipToParent: showShipToParent,
            ShipToAddresses:  shipToAddresses,
        }

        // 5. Render
        var component templ.Component
        if e.Request.Header.Get("HX-Request") == "true" {
            component = templates.AddressFormContent(data)
        } else {
            component = templates.AddressFormPage(data)
        }
        return component.Render(e.Request.Context(), e.Response)
    }
}

// HandleAddressSave returns a handler that processes the address creation form submission.
func HandleAddressSave(app *pocketbase.PocketBase, addressType AddressType) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        projectID := e.Request.PathValue("projectId")
        if projectID == "" {
            return e.String(400, "Missing project ID")
        }

        if err := e.Request.ParseForm(); err != nil {
            return e.String(400, "Invalid form data")
        }

        // 1. Verify project exists
        projectRecord, err := app.FindRecordById("projects", projectID)
        if err != nil {
            log.Printf("address_save: could not find project %s: %v", projectID, err)
            return e.String(404, "Project not found")
        }

        // 2. Extract form fields into a map
        fields := extractAddressFields(e)

        // 3. Fetch required fields from project_address_settings
        requiredFields := fetchRequiredFields(app, projectID, addressType)

        // 4. Validate
        errors := services.ValidateAddressFields(fields, requiredFields)

        if len(errors) > 0 {
            // Re-render form with errors
            var shipToAddresses []templates.ShipToOption
            var showShipToParent bool
            if addressType == AddressTypeInstallAt {
                showShipToParent, shipToAddresses = fetchShipToOptions(app, projectID)
            }

            data := templates.AddressFormData{
                IsEdit:          false,
                ProjectID:       projectID,
                ProjectName:     projectRecord.GetString("name"),
                AddressType:     string(addressType),
                AddressLabel:    addressTypeLabel(addressType),
                CompanyName:     fields["company_name"],
                ContactPerson:   fields["contact_person"],
                AddressLine1:    fields["address_line_1"],
                AddressLine2:    fields["address_line_2"],
                City:            fields["city"],
                State:           fields["state"],
                PinCode:         fields["pin_code"],
                Country:         fields["country"],
                Phone:           fields["phone"],
                Email:           fields["email"],
                GSTIN:           fields["gstin"],
                PAN:             fields["pan"],
                CIN:             fields["cin"],
                Website:         fields["website"],
                Fax:             fields["fax"],
                Landmark:        fields["landmark"],
                District:        fields["district"],
                RequiredFields:  requiredFields,
                Errors:          errors,
                StateOptions:    services.IndianStates,
                CountryOptions:  services.Countries,
                ShowShipToParent: showShipToParent,
                ShipToAddresses:  shipToAddresses,
                SelectedShipToID: e.Request.FormValue("ship_to_parent"),
            }

            component := templates.AddressFormPage(data)
            return component.Render(e.Request.Context(), e.Response)
        }

        // 5. Create the address record
        addressesCol, err := app.FindCollectionByNameOrId("addresses")
        if err != nil {
            log.Printf("address_save: could not find addresses collection: %v", err)
            return e.String(500, "Internal error")
        }

        record := core.NewRecord(addressesCol)
        record.Set("project", projectID)
        record.Set("address_type", string(addressType))
        setAddressRecordFields(record, fields)

        // Set ship_to_parent for install_at type
        if addressType == AddressTypeInstallAt {
            if parentID := e.Request.FormValue("ship_to_parent"); parentID != "" {
                record.Set("ship_to_parent", parentID)
            }
        }

        if err := app.Save(record); err != nil {
            log.Printf("address_save: could not save address: %v", err)
            return e.String(500, "Failed to save address")
        }

        // 6. Redirect to address list
        slugType := addressTypeToSlug(addressType)
        return e.Redirect(302, fmt.Sprintf("/projects/%s/addresses/%s", projectID, slugType))
    }
}

// --- Helper functions ---

// extractAddressFields extracts all address fields from the request form.
func extractAddressFields(e *core.RequestEvent) map[string]string {
    return map[string]string{
        "company_name":   strings.TrimSpace(e.Request.FormValue("company_name")),
        "contact_person": strings.TrimSpace(e.Request.FormValue("contact_person")),
        "address_line_1": strings.TrimSpace(e.Request.FormValue("address_line_1")),
        "address_line_2": strings.TrimSpace(e.Request.FormValue("address_line_2")),
        "city":           strings.TrimSpace(e.Request.FormValue("city")),
        "state":          strings.TrimSpace(e.Request.FormValue("state")),
        "pin_code":       strings.TrimSpace(e.Request.FormValue("pin_code")),
        "country":        strings.TrimSpace(e.Request.FormValue("country")),
        "phone":          strings.TrimSpace(e.Request.FormValue("phone")),
        "email":          strings.TrimSpace(e.Request.FormValue("email")),
        "gstin":          strings.TrimSpace(strings.ToUpper(e.Request.FormValue("gstin"))),
        "pan":            strings.TrimSpace(strings.ToUpper(e.Request.FormValue("pan"))),
        "cin":            strings.TrimSpace(strings.ToUpper(e.Request.FormValue("cin"))),
        "website":        strings.TrimSpace(e.Request.FormValue("website")),
        "fax":            strings.TrimSpace(e.Request.FormValue("fax")),
        "landmark":       strings.TrimSpace(e.Request.FormValue("landmark")),
        "district":       strings.TrimSpace(e.Request.FormValue("district")),
    }
}

// setAddressRecordFields sets all address fields on a PocketBase record.
func setAddressRecordFields(record *core.Record, fields map[string]string) {
    for key, val := range fields {
        record.Set(key, val)
    }
}

// fetchRequiredFields loads the required field configuration from project_address_settings.
func fetchRequiredFields(app *pocketbase.PocketBase, projectID string, addressType AddressType) map[string]bool {
    defaults := map[string]bool{
        "company_name":   true,
        "address_line_1": true,
        "city":           true,
        "state":          true,
        "pin_code":       true,
    }

    settingsCol, err := app.FindCollectionByNameOrId("project_address_settings")
    if err != nil {
        log.Printf("fetchRequiredFields: collection not found: %v", err)
        return defaults
    }

    records, err := app.FindRecordsByFilter(
        settingsCol,
        "project = {:projectId} && address_type = {:addressType}",
        "", 1, 0,
        map[string]any{"projectId": projectID, "addressType": string(addressType)},
    )
    if err != nil || len(records) == 0 {
        return defaults
    }

    // Parse the required_fields JSON array from the settings record
    rec := records[0]
    requiredFields := make(map[string]bool)

    // The required_fields column stores a JSON array of field names
    // e.g., ["company_name", "address_line_1", "city", "state", "pin_code", "gstin"]
    fieldsList := rec.GetStringSlice("required_fields")
    for _, f := range fieldsList {
        requiredFields[f] = true
    }

    if len(requiredFields) == 0 {
        return defaults
    }
    return requiredFields
}

// fetchShipToOptions loads Ship To addresses for the Install At parent dropdown.
// Returns (showDropdown, shipToAddresses).
func fetchShipToOptions(app *pocketbase.PocketBase, projectID string) (bool, []templates.ShipToOption) {
    // Check project setting for whether install_at requires ship_to parent
    // If the project toggle "install_at_linked_to_ship_to" is ON, we hide the dropdown
    // (auto-linking is handled elsewhere). If OFF, show the optional dropdown.
    projectRecord, err := app.FindRecordById("projects", projectID)
    if err != nil {
        return false, nil
    }

    // If the toggle is ON, no dropdown needed
    if projectRecord.GetBool("install_at_linked_to_ship_to") {
        return false, nil
    }

    // Fetch all ship_to addresses for this project
    addressesCol, err := app.FindCollectionByNameOrId("addresses")
    if err != nil {
        return true, nil
    }

    records, err := app.FindRecordsByFilter(
        addressesCol,
        "project = {:projectId} && address_type = 'ship_to'",
        "company_name", 0, 0,
        map[string]any{"projectId": projectID},
    )
    if err != nil {
        return true, nil
    }

    var options []templates.ShipToOption
    for _, rec := range records {
        options = append(options, templates.ShipToOption{
            ID:          rec.Id,
            CompanyName: rec.GetString("company_name"),
            City:        rec.GetString("city"),
        })
    }

    return true, options
}

// addressTypeToSlug converts an AddressType to URL slug format.
func addressTypeToSlug(at AddressType) string {
    return strings.ReplaceAll(string(at), "_", "-")
}
```

### Step 5: Implement HandleAddressEdit (GET + PATCH)

Create `handlers/address_edit.go`:

```go
package handlers

import (
    "fmt"
    "log"
    "strings"

    "github.com/a-h/templ"
    "github.com/pocketbase/pocketbase"
    "github.com/pocketbase/pocketbase/core"

    "projectcreation/services"
    "projectcreation/templates"
)

// HandleAddressEdit returns a handler that renders the address edit form pre-populated with existing data.
func HandleAddressEdit(app *pocketbase.PocketBase, addressType AddressType) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        projectID := e.Request.PathValue("projectId")
        addressID := e.Request.PathValue("addressId")
        if projectID == "" || addressID == "" {
            return e.String(400, "Missing project ID or address ID")
        }

        // 1. Verify project exists
        projectRecord, err := app.FindRecordById("projects", projectID)
        if err != nil {
            log.Printf("address_edit: could not find project %s: %v", projectID, err)
            return e.String(404, "Project not found")
        }

        // 2. Fetch the address record
        addressRecord, err := app.FindRecordById("addresses", addressID)
        if err != nil {
            log.Printf("address_edit: could not find address %s: %v", addressID, err)
            return e.String(404, "Address not found")
        }

        // 3. Verify address belongs to this project and address type
        if addressRecord.GetString("project") != projectID ||
            addressRecord.GetString("address_type") != string(addressType) {
            return e.String(403, "Address does not belong to this project/type")
        }

        // 4. Fetch required fields
        requiredFields := fetchRequiredFields(app, projectID, addressType)

        // 5. Prepare Ship To options for Install At
        var shipToAddresses []templates.ShipToOption
        var showShipToParent bool
        if addressType == AddressTypeInstallAt {
            showShipToParent, shipToAddresses = fetchShipToOptions(app, projectID)
        }

        // 6. Build form data from existing record
        data := templates.AddressFormData{
            IsEdit:           true,
            AddressID:        addressID,
            ProjectID:        projectID,
            ProjectName:      projectRecord.GetString("name"),
            AddressType:      string(addressType),
            AddressLabel:     addressTypeLabel(addressType),
            CompanyName:      addressRecord.GetString("company_name"),
            ContactPerson:    addressRecord.GetString("contact_person"),
            AddressLine1:     addressRecord.GetString("address_line_1"),
            AddressLine2:     addressRecord.GetString("address_line_2"),
            City:             addressRecord.GetString("city"),
            State:            addressRecord.GetString("state"),
            PinCode:          addressRecord.GetString("pin_code"),
            Country:          addressRecord.GetString("country"),
            Phone:            addressRecord.GetString("phone"),
            Email:            addressRecord.GetString("email"),
            GSTIN:            addressRecord.GetString("gstin"),
            PAN:              addressRecord.GetString("pan"),
            CIN:              addressRecord.GetString("cin"),
            Website:          addressRecord.GetString("website"),
            Fax:              addressRecord.GetString("fax"),
            Landmark:         addressRecord.GetString("landmark"),
            District:         addressRecord.GetString("district"),
            RequiredFields:   requiredFields,
            Errors:           make(map[string]string),
            StateOptions:     services.IndianStates,
            CountryOptions:   services.Countries,
            ShowShipToParent: showShipToParent,
            ShipToAddresses:  shipToAddresses,
            SelectedShipToID: addressRecord.GetString("ship_to_parent"),
        }

        // 7. Render
        var component templ.Component
        if e.Request.Header.Get("HX-Request") == "true" {
            component = templates.AddressFormContent(data)
        } else {
            component = templates.AddressFormPage(data)
        }
        return component.Render(e.Request.Context(), e.Response)
    }
}

// HandleAddressUpdate returns a handler that processes the address edit form submission (PATCH).
func HandleAddressUpdate(app *pocketbase.PocketBase, addressType AddressType) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        projectID := e.Request.PathValue("projectId")
        addressID := e.Request.PathValue("addressId")
        if projectID == "" || addressID == "" {
            return e.String(400, "Missing project ID or address ID")
        }

        if err := e.Request.ParseForm(); err != nil {
            return e.String(400, "Invalid form data")
        }

        // 1. Verify project exists
        projectRecord, err := app.FindRecordById("projects", projectID)
        if err != nil {
            return e.String(404, "Project not found")
        }

        // 2. Fetch existing address
        addressRecord, err := app.FindRecordById("addresses", addressID)
        if err != nil {
            return e.String(404, "Address not found")
        }

        if addressRecord.GetString("project") != projectID ||
            addressRecord.GetString("address_type") != string(addressType) {
            return e.String(403, "Address does not belong to this project/type")
        }

        // 3. Extract and validate fields
        fields := extractAddressFields(e)
        requiredFields := fetchRequiredFields(app, projectID, addressType)
        errors := services.ValidateAddressFields(fields, requiredFields)

        if len(errors) > 0 {
            var shipToAddresses []templates.ShipToOption
            var showShipToParent bool
            if addressType == AddressTypeInstallAt {
                showShipToParent, shipToAddresses = fetchShipToOptions(app, projectID)
            }

            data := templates.AddressFormData{
                IsEdit:           true,
                AddressID:        addressID,
                ProjectID:        projectID,
                ProjectName:      projectRecord.GetString("name"),
                AddressType:      string(addressType),
                AddressLabel:     addressTypeLabel(addressType),
                CompanyName:      fields["company_name"],
                ContactPerson:    fields["contact_person"],
                AddressLine1:     fields["address_line_1"],
                AddressLine2:     fields["address_line_2"],
                City:             fields["city"],
                State:            fields["state"],
                PinCode:          fields["pin_code"],
                Country:          fields["country"],
                Phone:            fields["phone"],
                Email:            fields["email"],
                GSTIN:            fields["gstin"],
                PAN:              fields["pan"],
                CIN:              fields["cin"],
                Website:          fields["website"],
                Fax:              fields["fax"],
                Landmark:         fields["landmark"],
                District:         fields["district"],
                RequiredFields:   requiredFields,
                Errors:           errors,
                StateOptions:     services.IndianStates,
                CountryOptions:   services.Countries,
                ShowShipToParent: showShipToParent,
                ShipToAddresses:  shipToAddresses,
                SelectedShipToID: e.Request.FormValue("ship_to_parent"),
            }

            component := templates.AddressFormPage(data)
            return component.Render(e.Request.Context(), e.Response)
        }

        // 4. Update the record
        setAddressRecordFields(addressRecord, fields)

        if addressType == AddressTypeInstallAt {
            if parentID := e.Request.FormValue("ship_to_parent"); parentID != "" {
                addressRecord.Set("ship_to_parent", parentID)
            } else {
                addressRecord.Set("ship_to_parent", "")
            }
        }

        if err := app.Save(addressRecord); err != nil {
            log.Printf("address_update: could not save address %s: %v", addressID, err)
            return e.String(500, "Failed to update address")
        }

        // 5. Redirect to address list
        slugType := addressTypeToSlug(addressType)
        return e.Redirect(302, fmt.Sprintf("/projects/%s/addresses/%s", projectID, slugType))
    }
}
```

### Step 6: Create the Form Template

Create `templates/address_form.templ`. The form is organized into 4 sections following the card-based layout pattern from `boq_create.templ`.

```templ
templ AddressFormContent(data AddressFormData) {
    <!-- Breadcrumbs -->
    <div class="flex items-center" style="gap: 6px; margin-bottom: 16px;">
        <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 500; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.5px;">
            PROJECTS
        </span>
        <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; color: var(--text-muted);">/</span>
        <a
            href={ templ.SafeURL(fmt.Sprintf("/projects/%s", data.ProjectID)) }
            style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 500; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.5px; text-decoration: none;"
        >
            { data.ProjectName }
        </a>
        <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; color: var(--text-muted);">/</span>
        <a
            href={ templ.SafeURL(fmt.Sprintf("/projects/%s/addresses/%s", data.ProjectID, data.AddressType)) }
            style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 500; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.5px; text-decoration: none;"
        >
            { data.AddressLabel }
        </a>
        <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; color: var(--text-muted);">/</span>
        <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; color: var(--terracotta); text-transform: uppercase; letter-spacing: 0.5px;">
            if data.IsEdit {
                EDIT
            } else {
                ADD NEW
            }
        </span>
    </div>

    <!-- Page Header -->
    <div>
        <h1 style="font-family: 'Space Grotesk', sans-serif; font-size: 32px; font-weight: 700; color: var(--text-primary); margin: 0;">
            if data.IsEdit {
                Edit { data.AddressLabel } Address
            } else {
                Add { data.AddressLabel } Address
            }
        </h1>
        <p style="font-family: 'Inter', sans-serif; font-size: 14px; color: var(--text-secondary); margin-top: 8px;">
            if data.IsEdit {
                Update the address details below
            } else {
                Fill in the address details below
            }
        </p>
    </div>

    <!-- Validation Error Banner -->
    if len(data.Errors) > 0 {
        <div style="background-color: #FEE2E2; border: 1px solid #EF4444; padding: 12px 16px; margin-top: 24px;">
            <div style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: #DC2626; margin-bottom: 4px;">
                PLEASE FIX THE FOLLOWING ERRORS
            </div>
            for field, msg := range data.Errors {
                <div style="font-family: 'Inter', sans-serif; font-size: 13px; color: #DC2626;">
                    { msg }
                </div>
            }
        </div>
    }

    <!-- Form -->
    <form
        if data.IsEdit {
            method="POST"
            action={ templ.SafeURL(fmt.Sprintf("/projects/%s/addresses/%s/%s/edit", data.ProjectID, data.AddressType, data.AddressID)) }
        } else {
            method="POST"
            action={ templ.SafeURL(fmt.Sprintf("/projects/%s/addresses/%s/new", data.ProjectID, data.AddressType)) }
        }
        style="margin-top: 24px;"
    >
        <!-- Hidden field for PATCH method override (edit mode) -->
        if data.IsEdit {
            <input type="hidden" name="_method" value="PATCH" />
        }

        <!-- ========================================= -->
        <!-- SECTION 1: Company Information -->
        <!-- ========================================= -->
        <div style="background-color: var(--bg-card);">
            <div style="background-color: #E2DED6; padding: 16px 24px;">
                <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary);">
                    COMPANY INFORMATION
                </span>
            </div>
            <div style="padding: 24px;">
                <!-- Row: Company Name + Contact Person -->
                <div class="flex" style="gap: 24px; margin-bottom: 16px;">
                    <div class="flex-1">
                        @addressField("company_name", "COMPANY NAME", "text", data.CompanyName, "Enter company name", data.RequiredFields["company_name"], data.Errors["company_name"])
                    </div>
                    <div class="flex-1">
                        @addressField("contact_person", "CONTACT PERSON", "text", data.ContactPerson, "Enter contact person name", data.RequiredFields["contact_person"], data.Errors["contact_person"])
                    </div>
                </div>
            </div>
        </div>

        <!-- ========================================= -->
        <!-- SECTION 2: Address Details -->
        <!-- ========================================= -->
        <div style="background-color: var(--bg-card); margin-top: 24px;">
            <div style="background-color: #E2DED6; padding: 16px 24px;">
                <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary);">
                    ADDRESS DETAILS
                </span>
            </div>
            <div style="padding: 24px;">
                <!-- Row: Address Line 1 -->
                <div style="margin-bottom: 16px;">
                    @addressField("address_line_1", "ADDRESS LINE 1", "text", data.AddressLine1, "Street address, building name", data.RequiredFields["address_line_1"], data.Errors["address_line_1"])
                </div>
                <!-- Row: Address Line 2 -->
                <div style="margin-bottom: 16px;">
                    @addressField("address_line_2", "ADDRESS LINE 2", "text", data.AddressLine2, "Floor, suite, unit (optional)", data.RequiredFields["address_line_2"], data.Errors["address_line_2"])
                </div>
                <!-- Row: Landmark + District -->
                <div class="flex" style="gap: 24px; margin-bottom: 16px;">
                    <div class="flex-1">
                        @addressField("landmark", "LANDMARK", "text", data.Landmark, "Nearby landmark", data.RequiredFields["landmark"], data.Errors["landmark"])
                    </div>
                    <div class="flex-1">
                        @addressField("district", "DISTRICT", "text", data.District, "Enter district", data.RequiredFields["district"], data.Errors["district"])
                    </div>
                </div>
                <!-- Row: City + State + PIN Code -->
                <div class="flex" style="gap: 24px; margin-bottom: 16px;">
                    <div class="flex-1">
                        @addressField("city", "CITY", "text", data.City, "Enter city", data.RequiredFields["city"], data.Errors["city"])
                    </div>
                    <div class="flex-1">
                        <!-- State dropdown -->
                        <label for="state" style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary); display: block; margin-bottom: 6px;">
                            STATE
                            if data.RequiredFields["state"] {
                                <span style="color: var(--terracotta);">*</span>
                            }
                        </label>
                        <select
                            id="state" name="state"
                            if data.RequiredFields["state"] {
                                required
                            }
                            style="width: 100%; padding: 10px 14px; font-family: 'Inter', sans-serif; font-size: 14px; color: var(--text-primary); background-color: var(--bg-page); border: 1px solid var(--border-light); border-radius: 0; outline: none; box-sizing: border-box; appearance: auto;"
                        >
                            <option value="">Select state</option>
                            for _, state := range data.StateOptions {
                                <option value={ state } selected?={ data.State == state }>{ state }</option>
                            }
                        </select>
                        if data.Errors["state"] != "" {
                            <div style="font-family: 'Inter', sans-serif; font-size: 12px; color: #DC2626; margin-top: 4px;">
                                { data.Errors["state"] }
                            </div>
                        }
                    </div>
                    <div style="width: 200px; min-width: 200px;">
                        @addressField("pin_code", "PIN CODE", "text", data.PinCode, "6-digit PIN", data.RequiredFields["pin_code"], data.Errors["pin_code"])
                    </div>
                </div>
                <!-- Row: Country -->
                <div style="width: 300px;">
                    <label for="country" style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary); display: block; margin-bottom: 6px;">
                        COUNTRY
                        if data.RequiredFields["country"] {
                            <span style="color: var(--terracotta);">*</span>
                        }
                    </label>
                    <select
                        id="country" name="country"
                        if data.RequiredFields["country"] {
                            required
                        }
                        style="width: 100%; padding: 10px 14px; font-family: 'Inter', sans-serif; font-size: 14px; color: var(--text-primary); background-color: var(--bg-page); border: 1px solid var(--border-light); border-radius: 0; outline: none; box-sizing: border-box; appearance: auto;"
                    >
                        for _, country := range data.CountryOptions {
                            <option value={ country } selected?={ data.Country == country }>{ country }</option>
                        }
                    </select>
                </div>
            </div>
        </div>

        <!-- ========================================= -->
        <!-- SECTION 3: Contact Information -->
        <!-- ========================================= -->
        <div style="background-color: var(--bg-card); margin-top: 24px;">
            <div style="background-color: #E2DED6; padding: 16px 24px;">
                <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary);">
                    CONTACT INFORMATION
                </span>
            </div>
            <div style="padding: 24px;">
                <!-- Row: Phone + Email -->
                <div class="flex" style="gap: 24px; margin-bottom: 16px;">
                    <div class="flex-1">
                        @addressField("phone", "PHONE", "tel", data.Phone, "10-digit mobile number", data.RequiredFields["phone"], data.Errors["phone"])
                    </div>
                    <div class="flex-1">
                        @addressField("email", "EMAIL", "email", data.Email, "email@example.com", data.RequiredFields["email"], data.Errors["email"])
                    </div>
                </div>
                <!-- Row: Website + Fax -->
                <div class="flex" style="gap: 24px;">
                    <div class="flex-1">
                        @addressField("website", "WEBSITE", "url", data.Website, "https://www.example.com", data.RequiredFields["website"], data.Errors["website"])
                    </div>
                    <div class="flex-1">
                        @addressField("fax", "FAX", "text", data.Fax, "Enter fax number", data.RequiredFields["fax"], data.Errors["fax"])
                    </div>
                </div>
            </div>
        </div>

        <!-- ========================================= -->
        <!-- SECTION 4: Tax & Legal Information -->
        <!-- ========================================= -->
        <div style="background-color: var(--bg-card); margin-top: 24px;">
            <div style="background-color: #E2DED6; padding: 16px 24px;">
                <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary);">
                    TAX & LEGAL INFORMATION
                </span>
            </div>
            <div style="padding: 24px;">
                <!-- Row: GSTIN + PAN -->
                <div class="flex" style="gap: 24px; margin-bottom: 16px;">
                    <div class="flex-1">
                        @addressField("gstin", "GSTIN", "text", data.GSTIN, "e.g., 27AAPFU0939F1ZV", data.RequiredFields["gstin"], data.Errors["gstin"])
                    </div>
                    <div class="flex-1">
                        @addressField("pan", "PAN", "text", data.PAN, "e.g., ABCDE1234F", data.RequiredFields["pan"], data.Errors["pan"])
                    </div>
                </div>
                <!-- Row: CIN -->
                <div style="width: 50%;">
                    @addressField("cin", "CIN", "text", data.CIN, "Corporate Identity Number", data.RequiredFields["cin"], data.Errors["cin"])
                </div>
            </div>
        </div>

        <!-- ========================================= -->
        <!-- SECTION 5: Ship To Parent (Install At only) -->
        <!-- ========================================= -->
        if data.ShowShipToParent {
            <div style="background-color: var(--bg-card); margin-top: 24px;">
                <div style="background-color: #E2DED6; padding: 16px 24px;">
                    <span style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary);">
                        LINKED SHIP TO ADDRESS (OPTIONAL)
                    </span>
                </div>
                <div style="padding: 24px;">
                    <label for="ship_to_parent" style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary); display: block; margin-bottom: 6px;">
                        SHIP TO PARENT
                    </label>
                    <select
                        id="ship_to_parent" name="ship_to_parent"
                        style="width: 100%; max-width: 500px; padding: 10px 14px; font-family: 'Inter', sans-serif; font-size: 14px; color: var(--text-primary); background-color: var(--bg-page); border: 1px solid var(--border-light); border-radius: 0; outline: none; box-sizing: border-box; appearance: auto;"
                    >
                        <option value="">None (standalone Install At)</option>
                        for _, st := range data.ShipToAddresses {
                            <option value={ st.ID } selected?={ data.SelectedShipToID == st.ID }>
                                { st.CompanyName } - { st.City }
                            </option>
                        }
                    </select>
                    <p style="font-family: 'Inter', sans-serif; font-size: 12px; color: var(--text-muted); margin-top: 6px;">
                        Optionally link this Install At address to a Ship To address
                    </p>
                </div>
            </div>
        }

        <!-- ========================================= -->
        <!-- Action Buttons -->
        <!-- ========================================= -->
        <div class="flex justify-end" style="gap: 12px; margin-top: 24px;">
            <a
                href={ templ.SafeURL(fmt.Sprintf("/projects/%s/addresses/%s", data.ProjectID, data.AddressType)) }
                class="flex items-center justify-center"
                style="padding: 12px 24px; font-family: 'Space Grotesk', sans-serif; font-size: 13px; font-weight: 600; letter-spacing: 1px; color: var(--text-primary); background-color: var(--bg-card); border: none; text-decoration: none;"
            >
                CANCEL
            </a>
            <button
                type="submit"
                class="flex items-center justify-center"
                style="padding: 12px 24px; font-family: 'Space Grotesk', sans-serif; font-size: 13px; font-weight: 600; letter-spacing: 1px; color: var(--text-light); background-color: var(--terracotta); border: none; cursor: pointer;"
            >
                if data.IsEdit {
                    UPDATE ADDRESS
                } else {
                    SAVE ADDRESS
                }
            </button>
        </div>
    </form>
}
```

#### Reusable Field Component

```templ
// addressField renders a single form field with label, required indicator, and error message.
templ addressField(name string, label string, inputType string, value string, placeholder string, required bool, errorMsg string) {
    <label
        for={ name }
        style="font-family: 'Space Grotesk', sans-serif; font-size: 11px; font-weight: 600; letter-spacing: 1px; color: var(--text-secondary); display: block; margin-bottom: 6px;"
    >
        { label }
        if required {
            <span style="color: var(--terracotta);">*</span>
        }
    </label>
    <input
        type={ inputType }
        id={ name }
        name={ name }
        value={ value }
        placeholder={ placeholder }
        if required {
            required
        }
        if errorMsg != "" {
            style="width: 100%; padding: 10px 14px; font-family: 'Inter', sans-serif; font-size: 14px; color: var(--text-primary); background-color: var(--bg-page); border: 1px solid #EF4444; border-radius: 0; outline: none; box-sizing: border-box;"
        } else {
            style="width: 100%; padding: 10px 14px; font-family: 'Inter', sans-serif; font-size: 14px; color: var(--text-primary); background-color: var(--bg-page); border: 1px solid var(--border-light); border-radius: 0; outline: none; box-sizing: border-box;"
        }
    />
    if errorMsg != "" {
        <div style="font-family: 'Inter', sans-serif; font-size: 12px; color: #DC2626; margin-top: 4px;">
            { errorMsg }
        </div>
    }
}
```

#### Full Page Wrapper

```templ
templ AddressFormPage(data AddressFormData) {
    if data.IsEdit {
        @PageShell("Edit " + data.AddressLabel + " Address - Project Creation") {
            @AddressFormContent(data)
        }
    } else {
        @PageShell("Add " + data.AddressLabel + " Address - Project Creation") {
            @AddressFormContent(data)
        }
    }
}
```

### Step 7: Register Routes in main.go

Add these routes inside the existing `app.OnServe().BindFunc()` block, alongside the address list routes from Phase 7:

```go
for _, at := range addressTypes {
    // ... existing list and count routes from Phase 7 ...

    // Create form (GET renders form, POST saves new address)
    se.Router.GET(
        "/projects/{projectId}/addresses/"+at.slug+"/new",
        handlers.HandleAddressCreate(app, at.addrType),
    )
    se.Router.POST(
        "/projects/{projectId}/addresses/"+at.slug+"/new",
        handlers.HandleAddressSave(app, at.addrType),
    )

    // Edit form (GET renders form, PATCH updates address)
    se.Router.GET(
        "/projects/{projectId}/addresses/"+at.slug+"/{addressId}/edit",
        handlers.HandleAddressEdit(app, at.addrType),
    )
    se.Router.PATCH(
        "/projects/{projectId}/addresses/"+at.slug+"/{addressId}/edit",
        handlers.HandleAddressUpdate(app, at.addrType),
    )
}
```

**Complete Route Table for Address Form:**

| HTTP Method | Route | Handler | Description |
|-------------|-------|---------|-------------|
| GET | `/projects/{projectId}/addresses/{type}/new` | `HandleAddressCreate` | Render empty form |
| POST | `/projects/{projectId}/addresses/{type}/new` | `HandleAddressSave` | Save new address |
| GET | `/projects/{projectId}/addresses/{type}/{addressId}/edit` | `HandleAddressEdit` | Render pre-populated form |
| PATCH | `/projects/{projectId}/addresses/{type}/{addressId}/edit` | `HandleAddressUpdate` | Update existing address |

---

## Form Layout Summary

The form is organized into 4 (or 5 for Install At) card sections:

### Section 1: Company Information
| Row | Fields |
|-----|--------|
| 1 | Company Name (flex-1) + Contact Person (flex-1) |

### Section 2: Address Details
| Row | Fields |
|-----|--------|
| 1 | Address Line 1 (full width) |
| 2 | Address Line 2 (full width) |
| 3 | Landmark (flex-1) + District (flex-1) |
| 4 | City (flex-1) + State dropdown (flex-1) + PIN Code (200px) |
| 5 | Country dropdown (300px) |

### Section 3: Contact Information
| Row | Fields |
|-----|--------|
| 1 | Phone (flex-1) + Email (flex-1) |
| 2 | Website (flex-1) + Fax (flex-1) |

### Section 4: Tax & Legal Information
| Row | Fields |
|-----|--------|
| 1 | GSTIN (flex-1) + PAN (flex-1) |
| 2 | CIN (50% width) |

### Section 5: Ship To Parent (Install At only)
| Row | Fields |
|-----|--------|
| 1 | Ship To Parent dropdown (full width, max 500px) |

---

## Validation Rules

| Field | Type | Validation | Format Example |
|-------|------|-----------|----------------|
| GSTIN | text, uppercase | 15 chars, regex `^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z]{1}[1-9A-Z]{1}Z[0-9A-Z]{1}$` | `27AAPFU0939F1ZV` |
| PAN | text, uppercase | 10 chars, regex `^[A-Z]{5}[0-9]{4}[A-Z]{1}$` | `ABCDE1234F` |
| PIN Code | text | 6 digits, first non-zero, regex `^[1-9][0-9]{5}$` | `400001` |
| Phone | tel | 10 digits, starts with 6-9, regex `^[6-9][0-9]{9}$` | `9876543210` |
| Email | email | Standard email regex | `user@example.com` |
| CIN | text, uppercase | 21 chars, regex pattern | `U12345MH2020PTC123456` |
| All others | text | No format validation, only required check | -- |

### Required Field Behavior

- Required fields are determined per project per address type via the `project_address_settings` collection
- If no settings record exists, defaults apply: company_name, address_line_1, city, state, pin_code
- Required fields show a terracotta `*` asterisk next to the label
- Required fields add the HTML `required` attribute
- Server-side validation checks required fields and returns errors inline
- On validation failure, the form re-renders with all entered values preserved and error messages below each invalid field

---

## Dependencies on Other Phases

| Dependency | Phase | Required For |
|-----------|-------|--------------|
| `addresses` PocketBase collection | Phase 5/6 | Saving/updating address records |
| `projects` PocketBase collection | Phase 1-3 | Project name, settings |
| `project_address_settings` collection | Phase 5/6 | Required field configuration |
| `HandleAddressList` handler | Phase 7 | Redirect target after save/update |
| `AddressType` constants | Phase 7 | Shared address type definitions |
| `PageShell` template | Existing | Full page wrapper |
| `addressTypeToSlug` helper | Phase 7 (shared) | URL slug generation |

---

## Testing / Verification Steps

### Create Form Testing

1. **Load create form** - navigate to `/projects/{id}/addresses/bill-from/new`
   - Verify breadcrumbs: PROJECTS / {Name} / Bill From / ADD NEW
   - Verify 4 section cards render correctly
   - Verify Country defaults to "India"
   - Verify State dropdown populated with all Indian states

2. **Required field indicators** - verify asterisks appear next to required fields based on project settings

3. **Submit empty required fields** - submit form with required fields empty
   - Verify error banner appears at top
   - Verify inline error messages appear below each invalid field
   - Verify field values are preserved (no data loss)

4. **GSTIN validation** - enter invalid GSTIN (e.g., "ABC")
   - Verify specific error message about format
   - Enter valid GSTIN "27AAPFU0939F1ZV" -- should pass

5. **PAN validation** - enter "12345" (invalid)
   - Verify error. Enter "ABCDE1234F" -- should pass

6. **PIN Code validation** - enter "12345" (5 digits, invalid)
   - Verify error. Enter "400001" -- should pass

7. **Phone validation** - enter "1234567890" (starts with 1, invalid)
   - Verify error. Enter "9876543210" -- should pass

8. **Successful create** - fill all required fields with valid data, submit
   - Verify redirect to address list page
   - Verify new address appears in the list

### Edit Form Testing

9. **Load edit form** - navigate to `/projects/{id}/addresses/bill-from/{addressId}/edit`
   - Verify all fields pre-populated with existing data
   - Verify breadcrumbs show EDIT instead of ADD NEW
   - Verify button says "UPDATE ADDRESS"

10. **Edit and save** - modify a field, submit
    - Verify redirect to list
    - Verify changes are persisted

11. **Validation on edit** - clear a required field and submit
    - Verify validation errors appear
    - Verify other field values preserved

### Install At Specific Testing

12. **Ship To Parent dropdown** - for install_at type, verify dropdown appears
    - Verify it lists all Ship To addresses for the project
    - Verify "None" option is available

13. **Project toggle OFF** - when `install_at_linked_to_ship_to` is false, dropdown shows
14. **Project toggle ON** - when toggle is true, dropdown is hidden

### HTMX Testing

15. **Partial rendering** - request form with `HX-Request: true`
    - Verify only form content partial returned

### Cancel Testing

16. **Cancel button** - click Cancel
    - Verify navigation back to address list (no data saved)

---

## Acceptance Criteria

- [ ] `handlers/address_create.go` compiles and handles GET (render form) and POST (save)
- [ ] `handlers/address_edit.go` compiles and handles GET (render form) and PATCH (update)
- [ ] `templates/address_form.templ` compiles via `templ generate`
- [ ] `services/address_validation.go` compiles with all validation functions
- [ ] `services/indian_states.go` provides full list of Indian states and union territories
- [ ] Form is organized into 4 card sections: Company Info, Address Details, Contact Info, Tax/Legal Info
- [ ] All 17 data fields are present in the form with appropriate input types
- [ ] State field is a dropdown with Indian states
- [ ] Country field is a dropdown defaulting to "India"
- [ ] Required field indicators (`*`) are dynamic based on `project_address_settings`
- [ ] Default required fields (when no settings): company_name, address_line_1, city, state, pin_code
- [ ] GSTIN validation: 15-char alphanumeric with correct regex pattern
- [ ] PAN validation: 10-char with correct regex pattern
- [ ] PIN Code validation: exactly 6 digits, first digit non-zero
- [ ] Phone validation: 10 digits starting with 6-9
- [ ] Email validation: standard email format regex
- [ ] CIN validation: 21-char with correct regex pattern
- [ ] Validation errors display inline below each field with red border and message
- [ ] Error banner appears at top of form when validation fails
- [ ] Form values are preserved on validation failure (no data loss)
- [ ] Successful create redirects to address list page
- [ ] Successful update redirects to address list page
- [ ] Edit form pre-populates all fields from existing record
- [ ] Breadcrumbs show correct path: Projects / {Name} / {Type} / Add|Edit
- [ ] Cancel button navigates back to address list without saving
- [ ] Install At type shows Ship To Parent dropdown when project toggle is OFF
- [ ] Install At type hides Ship To Parent dropdown when project toggle is ON
- [ ] HTMX partial rendering works for both create and edit forms
- [ ] Routes registered in main.go for all 5 address types (create + edit = 20 routes)
- [ ] `addressField` templ component is reusable across all text input fields
- [ ] Error field border changes to red (#EF4444) when validation error exists
- [ ] GSTIN and PAN inputs auto-uppercase on the server side
