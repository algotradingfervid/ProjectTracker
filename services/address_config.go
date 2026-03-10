package services

import "encoding/json"

// ColumnDef describes a single column in an address configuration.
type ColumnDef struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Required    bool   `json:"required"`
	Type        string `json:"type"`
	Fixed       bool   `json:"fixed"`
	ShowInTable bool   `json:"show_in_table"`
	ShowInPrint bool   `json:"show_in_print"`
	SortOrder   int    `json:"sort_order"`
}

// ParseColumnDefs parses the JSON columns field from an address_configs record.
func ParseColumnDefs(jsonStr string) []ColumnDef {
	var cols []ColumnDef
	if err := json.Unmarshal([]byte(jsonStr), &cols); err != nil {
		return nil
	}
	return cols
}

// ColumnDefsToJSON serializes column definitions to JSON.
func ColumnDefsToJSON(cols []ColumnDef) string {
	b, _ := json.Marshal(cols)
	return string(b)
}

// DefaultColumnDefsJSON returns the default columns JSON string for an address type.
func DefaultColumnDefsJSON(addressType string) string {
	cols := defaultColumnDefs(addressType)
	b, _ := json.Marshal(cols)
	return string(b)
}

func defaultColumnDefs(addressType string) []ColumnDef {
	base := []ColumnDef{
		{Name: "company_name", Label: "Company Name", Required: true, Type: "text", ShowInTable: true, ShowInPrint: true, SortOrder: 1},
		{Name: "contact_person", Label: "Contact Person", Type: "text", ShowInTable: true, ShowInPrint: true, SortOrder: 2},
		{Name: "phone", Label: "Phone", Type: "text", ShowInTable: true, ShowInPrint: true, SortOrder: 3},
		{Name: "email", Label: "Email", Type: "text", ShowInTable: false, ShowInPrint: false, SortOrder: 4},
		{Name: "gstin", Label: "GSTIN", Type: "text", ShowInTable: true, ShowInPrint: true, SortOrder: 5},
		{Name: "pan", Label: "PAN", Type: "text", ShowInTable: false, ShowInPrint: false, SortOrder: 6},
		{Name: "cin", Label: "CIN", Type: "text", ShowInTable: false, ShowInPrint: false, SortOrder: 7},
		{Name: "address_line_1", Label: "Address Line 1", Required: true, Type: "text", ShowInTable: true, ShowInPrint: true, SortOrder: 8},
		{Name: "address_line_2", Label: "Address Line 2", Type: "text", ShowInTable: false, ShowInPrint: true, SortOrder: 9},
		{Name: "landmark", Label: "Landmark", Type: "text", ShowInTable: false, ShowInPrint: false, SortOrder: 10},
		{Name: "district", Label: "District", Type: "text", ShowInTable: false, ShowInPrint: false, SortOrder: 11},
		{Name: "city", Label: "City", Required: true, Type: "text", ShowInTable: true, ShowInPrint: true, SortOrder: 12},
		{Name: "state", Label: "State", Required: true, Type: "text", ShowInTable: true, ShowInPrint: true, SortOrder: 13},
		{Name: "pin_code", Label: "PIN Code", Required: true, Type: "text", ShowInTable: true, ShowInPrint: true, SortOrder: 14},
		{Name: "country", Label: "Country", Required: true, Type: "text", ShowInTable: false, ShowInPrint: true, SortOrder: 15},
		{Name: "fax", Label: "Fax", Type: "text", ShowInTable: false, ShowInPrint: false, SortOrder: 16},
		{Name: "website", Label: "Website", Type: "text", ShowInTable: false, ShowInPrint: false, SortOrder: 17},
	}

	switch addressType {
	case "bill_from":
		setReq(base, "company_name", "address_line_1", "city", "state", "pin_code", "country", "gstin")
	case "dispatch_from":
		setReq(base, "company_name", "address_line_1", "city", "state", "pin_code", "country")
	case "bill_to":
		setReq(base, "company_name", "contact_person", "address_line_1", "city", "state", "pin_code", "country", "gstin")
	case "ship_to", "install_at":
		setReq(base, "contact_person", "address_line_1", "city", "state", "pin_code", "country", "phone")
	}

	return base
}

func setReq(cols []ColumnDef, names ...string) {
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	for i := range cols {
		cols[i].Required = nameSet[cols[i].Name]
	}
}
