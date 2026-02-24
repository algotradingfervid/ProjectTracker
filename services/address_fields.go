package services

// TemplateField describes one column in an address import Excel template.
type TemplateField struct {
	Key            string // internal name, matches PocketBase field name
	Label          string // human-readable header shown in Excel
	Description    string // shown on the Instructions sheet
	FormatRule     string // e.g. "6 digits", "15-char GSTIN", ""
	ExampleValue   string // shown on the Instructions sheet
	AlwaysRequired bool   // true = required regardless of project settings
}

// ShipToTemplateFields returns the ordered list of fields for Ship To address templates.
func ShipToTemplateFields() []TemplateField {
	return []TemplateField{
		{Key: "company_name", Label: "Company Name", Description: "Company or organisation name", ExampleValue: "Acme Corp"},
		{Key: "contact_person", Label: "Contact Person", Description: "Primary contact at the address", ExampleValue: "Rajesh Kumar"},
		{Key: "address_line_1", Label: "Address Line 1", Description: "Street address", ExampleValue: "123 MG Road", AlwaysRequired: true},
		{Key: "address_line_2", Label: "Address Line 2", Description: "Locality / landmark", ExampleValue: "Near City Mall"},
		{Key: "city", Label: "City", Description: "City name", ExampleValue: "Mumbai", AlwaysRequired: true},
		{Key: "state", Label: "State", Description: "Indian state (select from dropdown)", ExampleValue: "Maharashtra", AlwaysRequired: true},
		{Key: "pin_code", Label: "PIN Code", Description: "6-digit Indian postal code", FormatRule: "Exactly 6 digits", ExampleValue: "400001"},
		{Key: "country", Label: "Country", Description: "Country (select from dropdown)", ExampleValue: "India", AlwaysRequired: true},
		{Key: "landmark", Label: "Landmark", Description: "Nearby landmark for reference", ExampleValue: "Opposite City Mall"},
		{Key: "district", Label: "District", Description: "District name", ExampleValue: "Mumbai Suburban"},
		{Key: "phone", Label: "Phone", Description: "10-digit mobile number", FormatRule: "10 digits starting with 6-9", ExampleValue: "9876543210"},
		{Key: "email", Label: "Email", Description: "Email address", FormatRule: "Valid email format", ExampleValue: "rajesh@example.com"},
		{Key: "fax", Label: "Fax", Description: "Fax number", ExampleValue: "022-12345678"},
		{Key: "website", Label: "Website", Description: "Website URL", FormatRule: "Valid URL", ExampleValue: "https://example.com"},
		{Key: "gstin", Label: "GSTIN", Description: "15-character GST Identification Number", FormatRule: "Format: 22AAAAA0000A1Z5", ExampleValue: "27AAPFU0939F1ZV"},
		{Key: "pan", Label: "PAN", Description: "10-character Permanent Account Number", FormatRule: "Format: ABCDE1234F", ExampleValue: "ABCDE1234F"},
		{Key: "cin", Label: "CIN", Description: "21-character Corporate Identity Number", FormatRule: "Format: U12345AB1234ABC123456", ExampleValue: "U74999MH2000PTC123456"},
	}
}

// InstallAtTemplateFields returns the ordered list of fields for Install At address templates.
// It prepends a "Ship To Reference" column that links to a Ship To company_name.
func InstallAtTemplateFields() []TemplateField {
	installFields := []TemplateField{
		{Key: "ship_to_reference", Label: "Ship To Reference", Description: "Must match an existing Ship To 'Company Name' in this project", FormatRule: "Exact match required", ExampleValue: "Acme Corp", AlwaysRequired: true},
	}
	return append(installFields, ShipToTemplateFields()...)
}
