package handlers

import (
	"net/http"

	"github.com/pocketbase/pocketbase"

	"projectcreation/templates"
)

// BuildSidebarData constructs the SidebarData from the current request context.
// It reads the active project from middleware context and queries address counts.
func BuildSidebarData(r *http.Request, app *pocketbase.PocketBase) templates.SidebarData {
	activeProj := GetActiveProject(r)
	if activeProj == nil {
		return templates.SidebarData{
			ActivePath: r.URL.Path,
		}
	}

	data := templates.SidebarData{
		ActiveProject: activeProj,
		ActivePath:    r.URL.Path,
	}

	// Load project record for config
	projRec, err := app.FindRecordById("projects", activeProj.ID)
	if err == nil {
		data.ShipToIsInstallAt = projRec.GetBool("ship_to_equals_install_at")
	}

	// Count BOQs for this project
	boqCol, _ := app.FindCollectionByNameOrId("boqs")
	if boqCol != nil {
		boqs, _ := app.FindRecordsByFilter(boqCol, "project = {:pid}", "", 0, 0, map[string]any{"pid": activeProj.ID})
		data.BOQCount = len(boqs)
	}

	// Count addresses by type
	addrCol, _ := app.FindCollectionByNameOrId("addresses")
	if addrCol != nil {
		types := map[string]*int{
			"bill_from":  &data.AddressCounts.BillFrom,
			"ship_from":  &data.AddressCounts.ShipFrom,
			"bill_to":    &data.AddressCounts.BillTo,
			"ship_to":    &data.AddressCounts.ShipTo,
			"install_at": &data.AddressCounts.InstallAt,
		}
		for addrType, countPtr := range types {
			records, err := app.FindRecordsByFilter(
				addrCol,
				"project = {:pid} && type = {:type}",
				"", 0, 0,
				map[string]any{"pid": activeProj.ID, "type": addrType},
			)
			if err == nil {
				*countPtr = len(records)
			}
		}
		data.AddressCounts.Total = data.AddressCounts.BillFrom +
			data.AddressCounts.ShipFrom +
			data.AddressCounts.BillTo +
			data.AddressCounts.ShipTo +
			data.AddressCounts.InstallAt
	}

	return data
}
