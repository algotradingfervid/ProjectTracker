package services

import (
	"fmt"
	"math"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/orientation"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// GeneratePDF creates a PDF document from BOQ export data using maroto/v2.
// It returns the raw PDF bytes or an error.
func GeneratePDF(data ExportData) ([]byte, error) {
	cfg := config.NewBuilder().
		WithOrientation(orientation.Horizontal).
		WithPageSize(pagesize.A4).
		WithLeftMargin(10).
		WithTopMargin(10).
		WithRightMargin(10).
		WithPageNumber(props.PageNumber{
			Pattern: "Page {current} of {total}",
			Place:   props.RightBottom,
			Size:    7,
			Color:   &props.Color{Red: 120, Green: 120, Blue: 120},
		}).
		Build()

	m := maroto.New(cfg)

	// --- Header Section ---
	addHeader(m, data)

	// --- Table Header ---
	addTableHeader(m)

	// --- Table Body ---
	for _, r := range data.Rows {
		addTableRow(m, r)
	}

	// --- Summary Section ---
	addSummary(m, data)

	// --- Footer with generated date ---
	addFooter(m, data)

	// Generate PDF bytes
	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return doc.GetBytes(), nil
}

// addHeader adds the title, reference number, and date to the PDF.
func addHeader(m core.Maroto, data ExportData) {
	// Title row
	m.AddRows(
		row.New(12).Add(
			col.New(12).Add(
				text.New(data.Title, props.Text{
					Size:  16,
					Style: fontstyle.Bold,
					Align: align.Center,
				}),
			),
		),
	)

	// Reference number and date row
	m.AddRows(
		row.New(8).Add(
			col.New(6).Add(
				text.New(fmt.Sprintf("Reference: %s", data.ReferenceNumber), props.Text{
					Size:  9,
					Align: align.Left,
					Color: &props.Color{Red: 80, Green: 80, Blue: 80},
				}),
			),
			col.New(6).Add(
				text.New(fmt.Sprintf("Date: %s", data.CreatedDate), props.Text{
					Size:  9,
					Align: align.Right,
					Color: &props.Color{Red: 80, Green: 80, Blue: 80},
				}),
			),
		),
	)

	// Spacer
	m.AddRows(row.New(4))
}

// addTableHeader adds the column header row for the BOQ table.
func addTableHeader(m core.Maroto) {
	headerBg := &props.Color{Red: 33, Green: 37, Blue: 41}
	headerText := props.Text{
		Size:  8,
		Style: fontstyle.Bold,
		Align: align.Center,
		Color: &props.Color{Red: 255, Green: 255, Blue: 255},
	}
	headerTextLeft := headerText
	headerTextLeft.Align = align.Left

	headerCell := props.Cell{BackgroundColor: headerBg}

	m.AddRows(
		row.New(8).Add(
			col.New(1).Add(
				text.New("#", headerText),
			).WithStyle(&headerCell),
			col.New(3).Add(
				text.New("Description", headerTextLeft),
			).WithStyle(&headerCell),
			col.New(1).Add(
				text.New("Qty", headerText),
			).WithStyle(&headerCell),
			col.New(1).Add(
				text.New("UOM", headerText),
			).WithStyle(&headerCell),
			col.New(2).Add(
				text.New("Quoted Price", headerText),
			).WithStyle(&headerCell),
			col.New(2).Add(
				text.New("Budgeted Price", headerText),
			).WithStyle(&headerCell),
			col.New(1).Add(
				text.New("HSN", headerText),
			).WithStyle(&headerCell),
			col.New(1).Add(
				text.New("GST%", headerText),
			).WithStyle(&headerCell),
		),
	)
}

// addTableRow adds a single data row to the BOQ table, styled by indent level.
func addTableRow(m core.Maroto, r ExportRow) {
	// Determine text style and background based on level.
	var cellStyle *props.Cell
	var textSize float64 = 7
	var textStyle fontstyle.Type = fontstyle.Normal
	descPrefix := ""

	switch r.Level {
	case 0:
		// Main item: bold, white background (no cell style needed).
		textStyle = fontstyle.Bold
		textSize = 8
	case 1:
		// Sub-item: indented, light gray background.
		descPrefix = "  "
		bg := &props.Color{Red: 245, Green: 245, Blue: 245}
		cellStyle = &props.Cell{BackgroundColor: bg}
	case 2:
		// Sub-sub-item: double-indented, darker gray background.
		descPrefix = "    "
		bg := &props.Color{Red: 235, Green: 235, Blue: 235}
		cellStyle = &props.Cell{BackgroundColor: bg}
	}

	baseText := props.Text{
		Size:  textSize,
		Style: textStyle,
		Align: align.Center,
	}
	leftText := baseText
	leftText.Align = align.Left
	rightText := baseText
	rightText.Align = align.Right

	// Format quantity: whole number if no fractional part, otherwise 2 decimals.
	qtyStr := formatQty(r.Qty)

	// Format GST percentage.
	gstStr := fmt.Sprintf("%.0f%%", r.GSTPercent)

	// Build columns.
	colIndex := col.New(1).Add(text.New(r.Index, baseText))
	colDesc := col.New(3).Add(text.New(descPrefix+r.Description, leftText))
	colQty := col.New(1).Add(text.New(qtyStr, rightText))
	colUOM := col.New(1).Add(text.New(r.UOM, baseText))
	colQuoted := col.New(2).Add(text.New(FormatINR(r.QuotedPrice), rightText))
	colBudgeted := col.New(2).Add(text.New(FormatINR(r.BudgetedPrice), rightText))
	colHSN := col.New(1).Add(text.New(r.HSNCode, baseText))
	colGST := col.New(1).Add(text.New(gstStr, baseText))

	// Apply background style if needed.
	if cellStyle != nil {
		colIndex = colIndex.WithStyle(cellStyle)
		colDesc = colDesc.WithStyle(cellStyle)
		colQty = colQty.WithStyle(cellStyle)
		colUOM = colUOM.WithStyle(cellStyle)
		colQuoted = colQuoted.WithStyle(cellStyle)
		colBudgeted = colBudgeted.WithStyle(cellStyle)
		colHSN = colHSN.WithStyle(cellStyle)
		colGST = colGST.WithStyle(cellStyle)
	}

	m.AddRows(
		row.New(7).Add(
			colIndex,
			colDesc,
			colQty,
			colUOM,
			colQuoted,
			colBudgeted,
			colHSN,
			colGST,
		),
	)
}

// addSummary adds the totals and margin section at the bottom of the PDF.
func addSummary(m core.Maroto, data ExportData) {
	// Spacer before summary
	m.AddRows(row.New(6))

	summaryBg := &props.Color{Red: 240, Green: 240, Blue: 240}
	summaryCell := &props.Cell{BackgroundColor: summaryBg}

	labelStyle := props.Text{
		Size:  9,
		Style: fontstyle.Bold,
		Align: align.Right,
	}
	valueStyle := props.Text{
		Size:  9,
		Style: fontstyle.Bold,
		Align: align.Right,
	}

	// Total Quoted
	m.AddRows(
		row.New(8).Add(
			col.New(8).Add(
				text.New("Total Quoted Amount", labelStyle),
			).WithStyle(summaryCell),
			col.New(4).Add(
				text.New(FormatINR(data.TotalQuoted), valueStyle),
			).WithStyle(summaryCell),
		),
	)

	// Total Budgeted
	m.AddRows(
		row.New(8).Add(
			col.New(8).Add(
				text.New("Total Budgeted Amount", labelStyle),
			).WithStyle(summaryCell),
			col.New(4).Add(
				text.New(FormatINR(data.TotalBudgeted), valueStyle),
			).WithStyle(summaryCell),
		),
	)

	// Margin
	marginLabel := fmt.Sprintf("Margin (%.1f%%)", data.MarginPercent)
	m.AddRows(
		row.New(8).Add(
			col.New(8).Add(
				text.New(marginLabel, labelStyle),
			).WithStyle(summaryCell),
			col.New(4).Add(
				text.New(FormatINR(data.Margin), valueStyle),
			).WithStyle(summaryCell),
		),
	)
}

// addFooter adds the generated-date line at the bottom.
func addFooter(m core.Maroto, data ExportData) {
	m.AddRows(row.New(6))
	m.AddRows(
		row.New(6).Add(
			col.New(12).Add(
				text.New(
					fmt.Sprintf("Generated on %s", data.CreatedDate),
					props.Text{
						Size:  7,
						Align: align.Left,
						Color: &props.Color{Red: 140, Green: 140, Blue: 140},
					},
				),
			),
		),
	)
}

// formatQty returns a string representation of the quantity value.
// Whole numbers are formatted without decimals; fractional values get 2 decimal places.
func formatQty(qty float64) string {
	if qty == math.Trunc(qty) {
		return fmt.Sprintf("%.0f", qty)
	}
	return fmt.Sprintf("%.2f", qty)
}
