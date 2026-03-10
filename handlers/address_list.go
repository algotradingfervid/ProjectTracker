package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"


	"github.com/a-h/templ"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"projectcreation/services"
	"projectcreation/templates"
)

// AddressType represents the valid address types in the system.
type AddressType string

const (
	AddressTypeBillFrom     AddressType = "bill_from"
	AddressTypeShipFrom     AddressType = "ship_from"
	AddressTypeDispatchFrom AddressType = "dispatch_from"
	AddressTypeBillTo       AddressType = "bill_to"
	AddressTypeShipTo       AddressType = "ship_to"
	AddressTypeInstallAt    AddressType = "install_at"
)

// AddressTypeDisplayLabels maps address type to human-readable labels.
var AddressTypeDisplayLabels = map[AddressType]string{
	AddressTypeBillFrom:     "Bill From",
	AddressTypeShipFrom:     "Ship From",
	AddressTypeDispatchFrom: "Dispatch From",
	AddressTypeBillTo:       "Bill To",
	AddressTypeShipTo:       "Ship To",
	AddressTypeInstallAt:    "Install At",
}

// ValidAddressTypes is the ordered list of all address types.
var ValidAddressTypes = []AddressType{
	AddressTypeBillFrom,
	AddressTypeDispatchFrom,
	AddressTypeBillTo,
	AddressTypeShipTo,
	AddressTypeInstallAt,
}

func addressTypeLabel(at AddressType) string {
	if label, ok := AddressTypeDisplayLabels[at]; ok {
		return label
	}
	return string(at)
}

const (
	defaultPageSize  = 20
	maxPageSize      = 100
	defaultPage      = 1
	defaultSortBy    = "company_name"
	defaultSortOrder = "asc"
)

// addressListParams holds parsed query parameters for the address list.
type addressListParams struct {
	Page      int
	PageSize  int
	Search    string
	SortBy    string
	SortOrder string
}

// parseAddressListParams extracts and validates query parameters from the request.
func parseAddressListParams(e *core.RequestEvent) addressListParams {
	params := addressListParams{
		Page:      defaultPage,
		PageSize:  defaultPageSize,
		SortBy:    defaultSortBy,
		SortOrder: defaultSortOrder,
	}

	if p := e.Request.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			params.Page = v
		}
	}

	if ps := e.Request.URL.Query().Get("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= maxPageSize {
			params.PageSize = v
		}
	}

	params.Search = strings.TrimSpace(e.Request.URL.Query().Get("search"))

	if sb := e.Request.URL.Query().Get("sort_by"); sb != "" {
		allowedSorts := map[string]bool{
			"company_name": true, "contact_person": true, "city": true,
			"state": true, "pin_code": true, "gstin": true, "pan": true,
			"email": true, "phone": true, "created": true, "updated": true,
			"country": true, "district": true,
		}
		if allowedSorts[sb] {
			params.SortBy = sb
		}
	}

	if so := e.Request.URL.Query().Get("sort_order"); so == "desc" {
		params.SortOrder = "desc"
	}

	return params
}

// buildAddressFilter constructs a PocketBase filter string and bind params.
func buildAddressFilter(projectID string, addressType AddressType, search string) (string, map[string]any) {
	filter := "project = {:projectId} && address_type = {:addressType}"
	params := map[string]any{
		"projectId":   projectID,
		"addressType": string(addressType),
	}

	if search != "" {
		filter += " && (company_name ~ {:search} || city ~ {:search} || state ~ {:search} || contact_person ~ {:search})"
		params["search"] = search
	}

	return filter, params
}

// buildSortString returns the PocketBase sort expression.
func buildSortString(sortBy, sortOrder string) string {
	if sortOrder == "desc" {
		return "-" + sortBy
	}
	return sortBy
}

// HandleAddressList returns a handler that lists addresses of a given type for a project.
func HandleAddressList(app *pocketbase.PocketBase, addressType AddressType) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		if projectID == "" {
			return e.String(400, "Missing project ID")
		}

		// Activate this project via cookie
		http.SetCookie(e.Response, &http.Cookie{
			Name:   "active_project",
			Value:  projectID,
			Path:   "/",
			MaxAge: 86400 * 30,
		})

		// Verify project exists
		projectRecord, err := app.FindRecordById("projects", projectID)
		if err != nil {
			log.Printf("address_list: could not find project %s: %v", projectID, err)
			return e.String(404, "Project not found")
		}

		// Parse query parameters
		params := parseAddressListParams(e)

		// Build filter and sort
		filter, filterParams := buildAddressFilter(projectID, addressType, params.Search)
		sortStr := buildSortString(params.SortBy, params.SortOrder)

		// Get total count for pagination
		addressesCol, err := app.FindCollectionByNameOrId("addresses")
		if err != nil {
			log.Printf("address_list: could not find addresses collection: %v", err)
			return e.String(500, "Internal error")
		}

		allRecords, err := app.FindRecordsByFilter(addressesCol, filter, sortStr, 0, 0, filterParams)
		if err != nil {
			log.Printf("address_list: could not count addresses for project %s type %s: %v", projectID, addressType, err)
			allRecords = nil
		}
		totalCount := len(allRecords)

		// Calculate pagination
		totalPages := int(math.Ceil(float64(totalCount) / float64(params.PageSize)))
		if totalPages < 1 {
			totalPages = 1
		}
		if params.Page > totalPages {
			params.Page = totalPages
		}

		// Fetch paginated records
		offset := (params.Page - 1) * params.PageSize
		records, err := app.FindRecordsByFilter(
			addressesCol, filter, sortStr, params.PageSize, offset, filterParams,
		)
		if err != nil {
			log.Printf("address_list: could not query addresses for project %s type %s: %v", projectID, addressType, err)
			records = nil
		}

		// Map records to view models
		var items []templates.AddressListItem
		for i, rec := range records {
			data := readAddressData(rec)
			items = append(items, templates.AddressListItem{
				ID:            rec.Id,
				Index:         offset + i + 1,
				AddressCode:   rec.GetString("address_code"),
				Data:          data,
				CompanyName:   data["company_name"],
				ContactPerson: data["contact_person"],
				AddressLine1:  data["address_line_1"],
				AddressLine2:  data["address_line_2"],
				City:          data["city"],
				State:         data["state"],
				PinCode:       data["pin_code"],
				Country:       data["country"],
				Phone:         data["phone"],
				Email:         data["email"],
				GSTIN:         data["gstin"],
				PAN:           data["pan"],
				CIN:           data["cin"],
				Website:       data["website"],
				Fax:           data["fax"],
				Landmark:      data["landmark"],
				District:      data["district"],
				DistrictName:  rec.GetString("district_name"),
				MandalName:    rec.GetString("mandal_name"),
				MandalCode:    rec.GetString("mandal_code"),
			})
		}

		// Build page number list for pagination controls
		var pageNumbers []int
		for p := 1; p <= totalPages; p++ {
			pageNumbers = append(pageNumbers, p)
		}

		// Build template data
		data := templates.AddressListData{
			ProjectID:    projectID,
			ProjectName:  projectRecord.GetString("name"),
			AddressType:  string(addressType),
			AddressLabel: addressTypeLabel(addressType),
			Items:        items,
			TotalCount:   totalCount,
			Page:         params.Page,
			PageSize:     params.PageSize,
			TotalPages:   totalPages,
			PageNumbers:  pageNumbers,
			HasPrev:      params.Page > 1,
			HasNext:      params.Page < totalPages,
			Search:       params.Search,
			SortBy:       params.SortBy,
			SortOrder:    params.SortOrder,
		}

		// Render: partial for HTMX, full page otherwise
		var component templ.Component
		if e.Request.Header.Get("HX-Request") == "true" {
			component = templates.AddressListContent(data)
		} else {
			headerData := GetHeaderData(e.Request)
			sidebarData := GetSidebarData(e.Request)
			component = templates.AddressListPage(data, headerData, sidebarData)
		}
		return component.Render(e.Request.Context(), e.Response)
	}
}

// HandleAddressCount returns a handler that returns the address count for a given type.
func HandleAddressCount(app *pocketbase.PocketBase, addressType AddressType) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		projectID := e.Request.PathValue("projectId")
		if projectID == "" {
			return e.String(400, "Missing project ID")
		}

		addressesCol, err := app.FindCollectionByNameOrId("addresses")
		if err != nil {
			return e.String(500, "Internal error")
		}

		filter := "project = {:projectId} && address_type = {:addressType}"
		filterParams := map[string]any{
			"projectId":   projectID,
			"addressType": string(addressType),
		}

		records, err := app.FindRecordsByFilter(addressesCol, filter, "", 0, 0, filterParams)
		if err != nil {
			log.Printf("address_count: error querying count for project %s type %s: %v", projectID, addressType, err)
			return e.String(200, "0")
		}

		return e.String(200, fmt.Sprintf("%d", len(records)))
	}
}

// readAddressData reads address field values from JSON data field with fallback to fixed fields.
func readAddressData(rec *core.Record) map[string]string {
	return services.ReadAddressData(rec)
}

// getOrCreateAddressConfig fetches or creates the address_config for a project/type.
// Returns the config record and parsed column definitions.
func getOrCreateAddressConfig(app *pocketbase.PocketBase, projectID string, addressType AddressType) (*core.Record, []services.ColumnDef) {
	configType := mapToConfigType(addressType)

	configCol, err := app.FindCollectionByNameOrId("address_configs")
	if err != nil {
		return nil, nil
	}

	records, err := app.FindRecordsByFilter(
		configCol, "project = {:pid} && address_type = {:type}",
		"", 1, 0,
		map[string]any{"pid": projectID, "type": configType},
	)
	if err == nil && len(records) > 0 {
		cols := services.ParseColumnDefs(records[0].GetString("columns"))
		return records[0], cols
	}

	// Create default config
	rec := core.NewRecord(configCol)
	rec.Set("project", projectID)
	rec.Set("address_type", configType)
	rec.Set("columns", services.DefaultColumnDefsJSON(configType))
	if err := app.Save(rec); err != nil {
		log.Printf("getOrCreateAddressConfig: failed to create config: %v", err)
		return nil, nil
	}

	cols := services.ParseColumnDefs(rec.GetString("columns"))
	return rec, cols
}

// mapToConfigType maps handler address types to address_configs type values.
func mapToConfigType(at AddressType) string {
	if at == AddressTypeShipFrom {
		return "dispatch_from"
	}
	return string(at)
}

// buildAddressDataJSON builds a JSON string from address form fields.
func buildAddressDataJSON(fields map[string]string) string {
	b, _ := json.Marshal(fields)
	return string(b)
}

// generateAddressCode creates an address code from the company name or ID.
func generateAddressCode(companyName, fallbackID string) string {
	code := companyName
	if code == "" {
		code = fallbackID
	}
	return strings.ReplaceAll(strings.ToUpper(code), " ", "-")
}

