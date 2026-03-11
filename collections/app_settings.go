package collections

import (
	"fmt"
	"io"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const defaultCompanyName = "Fervidsmart Solutions Pvt Ltd"

// GetAppSettings returns the singleton app_settings record.
// If none exists, it creates one with default values.
func GetAppSettings(app *pocketbase.PocketBase) (*core.Record, error) {
	col, err := app.FindCollectionByNameOrId("app_settings")
	if err != nil {
		return nil, fmt.Errorf("app_settings collection not found: %w", err)
	}

	records, err := app.FindAllRecords(col)
	if err != nil {
		return nil, fmt.Errorf("could not query app_settings: %w", err)
	}
	if len(records) > 0 {
		return records[0], nil
	}

	// Create default record
	record := core.NewRecord(col)
	record.Set("company_name", defaultCompanyName)
	if err := app.Save(record); err != nil {
		return nil, fmt.Errorf("could not create default app_settings: %w", err)
	}
	log.Println("app_settings: created default settings record")
	return record, nil
}

// GetCompanyName returns the company name from app_settings, or the default.
func GetCompanyName(app *pocketbase.PocketBase) string {
	record, err := GetAppSettings(app)
	if err != nil {
		return defaultCompanyName
	}
	name := record.GetString("company_name")
	if name == "" {
		return defaultCompanyName
	}
	return name
}

// GetLogoURL returns the URL for the uploaded logo, or empty string if none.
// PocketBase serves files at /api/files/{collectionId}/{recordId}/{filename}
func GetLogoURL(app *pocketbase.PocketBase) string {
	record, err := GetAppSettings(app)
	if err != nil {
		return ""
	}
	logo := record.GetString("logo")
	if logo == "" {
		return ""
	}
	col, err := app.FindCollectionByNameOrId("app_settings")
	if err != nil {
		return ""
	}
	return fmt.Sprintf("/api/files/%s/%s/%s", col.Id, record.Id, logo)
}

// GetLogoBytes reads the logo file from PocketBase storage and returns
// the raw bytes and filename. Returns nil, "" if no logo is set.
func GetLogoBytes(app *pocketbase.PocketBase) ([]byte, string, error) {
	record, err := GetAppSettings(app)
	if err != nil {
		return nil, "", err
	}
	logo := record.GetString("logo")
	if logo == "" {
		return nil, "", nil
	}

	fs, err := app.NewFilesystem()
	if err != nil {
		return nil, "", fmt.Errorf("could not create filesystem: %w", err)
	}
	defer fs.Close()

	col, err := app.FindCollectionByNameOrId("app_settings")
	if err != nil {
		return nil, "", err
	}

	key := fmt.Sprintf("%s/%s/%s", col.Id, record.Id, logo)
	reader, err := fs.GetReader(key)
	if err != nil {
		return nil, "", fmt.Errorf("could not read logo file: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", fmt.Errorf("could not read logo bytes: %w", err)
	}

	return data, logo, nil
}
