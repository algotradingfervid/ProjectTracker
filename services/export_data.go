package services

// ExportRow represents a single row in the BOQ export (main, sub, or sub-sub item).
type ExportRow struct {
	Level         int     // 0 = main item, 1 = sub-item, 2 = sub-sub-item
	Index         string  // "1", "1.1", "1.1.1" etc
	Description   string
	Qty           float64
	UOM           string
	QuotedPrice   float64
	BudgetedPrice float64
	HSNCode       string
	GSTPercent    float64
}

// ExportData holds all data needed for export.
type ExportData struct {
	Title           string
	ReferenceNumber string
	CreatedDate     string
	Rows            []ExportRow
	TotalQuoted     float64
	TotalBudgeted   float64
	Margin          float64
	MarginPercent   float64
}
