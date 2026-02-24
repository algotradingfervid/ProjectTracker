package services

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pocketbase/pocketbase"
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
func ValidateGSTIN(gstin string) bool {
	gstin = strings.TrimSpace(strings.ToUpper(gstin))
	if gstin == "" {
		return true
	}
	return len(gstin) == 15 && gstinPattern.MatchString(gstin)
}

// ValidatePAN validates a PAN number (10-character alphanumeric).
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

// ValidateAddressFormat validates format-specific fields (GSTIN, PAN, etc.)
// and returns a map of field -> error message for any format violations.
func ValidateAddressFormat(fields map[string]string) map[string]string {
	errors := make(map[string]string)

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

// AddressField represents a single address field with its DB name and display label.
type AddressField struct {
	Name  string
	Label string
}

// AllAddressFields is the ordered list of all address fields that can be configured.
var AllAddressFields = []AddressField{
	{Name: "company_name", Label: "Company Name"},
	{Name: "contact_person", Label: "Contact Person"},
	{Name: "address_line_1", Label: "Address Line 1"},
	{Name: "address_line_2", Label: "Address Line 2"},
	{Name: "city", Label: "City"},
	{Name: "state", Label: "State"},
	{Name: "pin_code", Label: "PIN Code"},
	{Name: "country", Label: "Country"},
	{Name: "landmark", Label: "Landmark"},
	{Name: "district", Label: "District"},
	{Name: "phone", Label: "Phone"},
	{Name: "email", Label: "Email"},
	{Name: "fax", Label: "Fax"},
	{Name: "website", Label: "Website"},
	{Name: "gstin", Label: "GSTIN"},
	{Name: "pan", Label: "PAN"},
	{Name: "cin", Label: "CIN"},
}

// ValidateAddress checks an address record against the project's required-field settings
// for the given address type. Returns a map of field -> error message for any violations.
func ValidateAddress(app *pocketbase.PocketBase, projectID, addressType string, formData map[string]string) map[string]string {
	errors := make(map[string]string)

	settingsCol, err := app.FindCollectionByNameOrId("project_address_settings")
	if err != nil {
		return errors
	}

	settings, err := app.FindRecordsByFilter(
		settingsCol,
		"project = {:projectId} && address_type = {:addrType}",
		"", 1, 0,
		map[string]any{
			"projectId": projectID,
			"addrType":  addressType,
		},
	)
	if err != nil || len(settings) == 0 {
		return errors
	}

	setting := settings[0]

	for _, field := range AllAddressFields {
		reqFieldName := "req_" + field.Name
		if setting.GetBool(reqFieldName) {
			val := formData[field.Name]
			if val == "" {
				errors[field.Name] = fmt.Sprintf("%s is required", field.Label)
			}
		}
	}

	return errors
}

// GetRequiredFields returns a map of field names that are required for a given
// project + address type. Used by templates to mark required fields in the form.
func GetRequiredFields(app *pocketbase.PocketBase, projectID, addressType string) map[string]bool {
	required := make(map[string]bool)

	settingsCol, err := app.FindCollectionByNameOrId("project_address_settings")
	if err != nil {
		return required
	}

	settings, err := app.FindRecordsByFilter(
		settingsCol,
		"project = {:projectId} && address_type = {:addrType}",
		"", 1, 0,
		map[string]any{
			"projectId": projectID,
			"addrType":  addressType,
		},
	)
	if err != nil || len(settings) == 0 {
		return required
	}

	setting := settings[0]
	for _, field := range AllAddressFields {
		reqFieldName := "req_" + field.Name
		if setting.GetBool(reqFieldName) {
			required[field.Name] = true
		}
	}

	return required
}
