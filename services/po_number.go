package services

import (
	"time"

	"github.com/pocketbase/pocketbase"
)

// GeneratePONumber creates the next PO number for a project using the
// configurable numbering service. It reads po_prefix, po_number_format,
// po_separator, po_seq_padding, and po_seq_start from the project record.
func GeneratePONumber(app *pocketbase.PocketBase, projectId string, now time.Time) (string, error) {
	return NextDocNumber(app, projectId, "po", now)
}
