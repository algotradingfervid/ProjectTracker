package services

import (
	"fmt"
	"strings"

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

// DCExportAddress holds address data for DC PDF export.
type DCExportAddress struct {
	CompanyName   string
	AddressLines  string
	ContactPerson string
	Phone         string
	GSTIN         string
}

// DCExportLineItem holds a single line item for DC PDF export.
type DCExportLineItem struct {
	SINo        int
	Description string
	HSNCode     string
	Qty         int
	UOM         string
	Rate        float64
	TaxPercent  float64
	Taxable     float64
	TaxAmount   float64
	Total       float64
	Serials     []string
}

// DCExportTransport holds transport details for DC PDF.
type DCExportTransport struct {
	TransporterName string
	VehicleNumber   string
	EwayBillNumber  string
	DocketNumber    string
}

// DCExportData holds all data for generating a DC PDF.
type DCExportData struct {
	CompanyName  string
	LogoBytes    []byte
	LogoFilename string

	DCNumber    string
	DCType      string // transit, official, transfer
	Status      string
	ChallanDate string

	BillFrom     *DCExportAddress
	DispatchFrom *DCExportAddress
	BillTo       *DCExportAddress
	ShipTo       *DCExportAddress

	Transport *DCExportTransport
	LineItems []DCExportLineItem

	TotalTaxable float64
	TotalTax     float64
	GrandTotal   float64

	// Transfer DC specific
	HubAddress   string
	Destinations []string // destination names for transfer DCs
}

// GenerateDCPDF creates a PDF document for a Delivery Challan.
func GenerateDCPDF(data *DCExportData) ([]byte, error) {
	cfg := config.NewBuilder().
		WithOrientation(orientation.Vertical).
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

	addDCHeader(m, data)
	addDCAddresses(m, data)
	addDCTransport(m, data)
	addDCLineItems(m, data)

	if data.DCType != "official" {
		addDCTotals(m, data)
	}

	if data.DCType == "transfer" && len(data.Destinations) > 0 {
		addDCDestinations(m, data)
	}

	addDCSerials(m, data)
	addDCSignatures(m)

	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate DC PDF: %w", err)
	}

	return doc.GetBytes(), nil
}

func addDCHeader(m core.Maroto, data *DCExportData) {
	// Company name + DC title
	m.AddRows(
		row.New(10).Add(
			col.New(6).Add(
				text.New(data.CompanyName, props.Text{
					Size:  14,
					Style: fontstyle.Bold,
					Align: align.Left,
				}),
			),
			col.New(6).Add(
				text.New("DELIVERY CHALLAN", props.Text{
					Size:  14,
					Style: fontstyle.Bold,
					Align: align.Right,
					Color: &props.Color{Red: 33, Green: 37, Blue: 41},
				}),
			),
		),
	)

	// DC number + type + date
	typeLabel := strings.ToUpper(data.DCType)
	m.AddRows(
		row.New(7).Add(
			col.New(4).Add(
				text.New(fmt.Sprintf("DC #: %s", data.DCNumber), props.Text{
					Size:  10,
					Style: fontstyle.Bold,
					Align: align.Left,
				}),
			),
			col.New(4).Add(
				text.New(fmt.Sprintf("Type: %s", typeLabel), props.Text{
					Size:  9,
					Align: align.Center,
					Color: &props.Color{Red: 100, Green: 100, Blue: 100},
				}),
			),
			col.New(4).Add(
				text.New(fmt.Sprintf("Date: %s", data.ChallanDate), props.Text{
					Size:  9,
					Align: align.Right,
					Color: &props.Color{Red: 100, Green: 100, Blue: 100},
				}),
			),
		),
	)

	m.AddRows(row.New(3))
}

func addDCAddresses(m core.Maroto, data *DCExportData) {
	sectionLabel := props.Text{
		Size:  7,
		Style: fontstyle.Bold,
		Align: align.Left,
		Color: &props.Color{Red: 100, Green: 100, Blue: 100},
	}
	headerBg := &props.Color{Red: 245, Green: 243, Blue: 239}
	headerCell := &props.Cell{BackgroundColor: headerBg}

	// Bill From + Dispatch From
	m.AddRows(
		row.New(7).Add(
			col.New(6).Add(text.New("BILL FROM", sectionLabel)).WithStyle(headerCell),
			col.New(6).Add(text.New("DISPATCH FROM", sectionLabel)).WithStyle(headerCell),
		),
	)
	addDCAddressPair(m, data.BillFrom, data.DispatchFrom)

	// Bill To + Ship To
	m.AddRows(
		row.New(7).Add(
			col.New(6).Add(text.New("BILL TO", sectionLabel)).WithStyle(headerCell),
			col.New(6).Add(text.New("SHIP TO", sectionLabel)).WithStyle(headerCell),
		),
	)
	addDCAddressPair(m, data.BillTo, data.ShipTo)

	m.AddRows(row.New(3))
}

func addDCAddressPair(m core.Maroto, left, right *DCExportAddress) {
	boldValue := props.Text{Size: 8, Style: fontstyle.Bold, Align: align.Left}
	valueStyle := props.Text{Size: 8, Align: align.Left}

	leftName, rightName := "", ""
	leftAddr, rightAddr := "", ""
	leftGSTIN, rightGSTIN := "", ""
	leftContact, rightContact := "", ""

	if left != nil {
		leftName = left.CompanyName
		leftAddr = left.AddressLines
		leftGSTIN = left.GSTIN
		leftContact = joinDCContact(left.ContactPerson, left.Phone)
	}
	if right != nil {
		rightName = right.CompanyName
		rightAddr = right.AddressLines
		rightGSTIN = right.GSTIN
		rightContact = joinDCContact(right.ContactPerson, right.Phone)
	}

	m.AddRows(row.New(7).Add(
		col.New(6).Add(text.New(leftName, boldValue)),
		col.New(6).Add(text.New(rightName, boldValue)),
	))
	if leftAddr != "" || rightAddr != "" {
		m.AddRows(row.New(7).Add(
			col.New(6).Add(text.New(leftAddr, valueStyle)),
			col.New(6).Add(text.New(rightAddr, valueStyle)),
		))
	}
	if leftGSTIN != "" || rightGSTIN != "" {
		m.AddRows(row.New(7).Add(
			col.New(6).Add(text.New(fmtField("GSTIN", leftGSTIN), valueStyle)),
			col.New(6).Add(text.New(fmtField("GSTIN", rightGSTIN), valueStyle)),
		))
	}
	if leftContact != "" || rightContact != "" {
		m.AddRows(row.New(7).Add(
			col.New(6).Add(text.New(leftContact, valueStyle)),
			col.New(6).Add(text.New(rightContact, valueStyle)),
		))
	}
}

func joinDCContact(person, phone string) string {
	parts := []string{}
	if person != "" {
		parts = append(parts, person)
	}
	if phone != "" {
		parts = append(parts, phone)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " | ")
}

func addDCTransport(m core.Maroto, data *DCExportData) {
	if data.Transport == nil {
		return
	}
	t := data.Transport
	if t.TransporterName == "" && t.VehicleNumber == "" && t.EwayBillNumber == "" && t.DocketNumber == "" {
		return
	}

	sectionLabel := props.Text{
		Size:  7,
		Style: fontstyle.Bold,
		Align: align.Left,
		Color: &props.Color{Red: 100, Green: 100, Blue: 100},
	}
	headerBg := &props.Color{Red: 245, Green: 243, Blue: 239}
	headerCell := &props.Cell{BackgroundColor: headerBg}

	m.AddRows(
		row.New(7).Add(
			col.New(12).Add(text.New("TRANSPORT DETAILS", sectionLabel)).WithStyle(headerCell),
		),
	)

	valueStyle := props.Text{Size: 8, Align: align.Left}
	labelStyle := props.Text{Size: 7, Style: fontstyle.Bold, Align: align.Left, Color: &props.Color{Red: 100, Green: 100, Blue: 100}}

	// Row 1: Transporter + Vehicle
	m.AddRows(row.New(7).Add(
		col.New(2).Add(text.New("Transporter:", labelStyle)),
		col.New(4).Add(text.New(t.TransporterName, valueStyle)),
		col.New(2).Add(text.New("Vehicle:", labelStyle)),
		col.New(4).Add(text.New(t.VehicleNumber, valueStyle)),
	))

	// Row 2: E-Way Bill + Docket
	if t.EwayBillNumber != "" || t.DocketNumber != "" {
		m.AddRows(row.New(7).Add(
			col.New(2).Add(text.New("E-Way Bill:", labelStyle)),
			col.New(4).Add(text.New(t.EwayBillNumber, valueStyle)),
			col.New(2).Add(text.New("Docket No:", labelStyle)),
			col.New(4).Add(text.New(t.DocketNumber, valueStyle)),
		))
	}

	m.AddRows(row.New(3))
}

func addDCLineItems(m core.Maroto, data *DCExportData) {
	headerBg := &props.Color{Red: 33, Green: 37, Blue: 41}
	headerText := props.Text{
		Size:  7,
		Style: fontstyle.Bold,
		Align: align.Center,
		Color: &props.Color{Red: 255, Green: 255, Blue: 255},
	}
	headerTextLeft := headerText
	headerTextLeft.Align = align.Left
	headerCell := props.Cell{BackgroundColor: headerBg}

	if data.DCType == "official" {
		// Official DC: no pricing columns
		m.AddRows(
			row.New(8).Add(
				col.New(1).Add(text.New("SI No", headerText)).WithStyle(&headerCell),
				col.New(5).Add(text.New("Description", headerTextLeft)).WithStyle(&headerCell),
				col.New(2).Add(text.New("HSN", headerText)).WithStyle(&headerCell),
				col.New(2).Add(text.New("UoM", headerText)).WithStyle(&headerCell),
				col.New(2).Add(text.New("Qty", headerText)).WithStyle(&headerCell),
			),
		)

		altBg := &props.Color{Red: 248, Green: 249, Blue: 250}
		for i, item := range data.LineItems {
			bodyText := props.Text{Size: 7, Align: align.Center}
			bodyTextLeft := props.Text{Size: 7, Align: align.Left}
			bodyTextRight := props.Text{Size: 7, Align: align.Right}

			var cellStyle *props.Cell
			if i%2 == 1 {
				cellStyle = &props.Cell{BackgroundColor: altBg}
			}

			colSI := col.New(1).Add(text.New(fmt.Sprintf("%d", item.SINo), bodyText))
			colDesc := col.New(5).Add(text.New(item.Description, bodyTextLeft))
			colHSN := col.New(2).Add(text.New(item.HSNCode, bodyText))
			colUoM := col.New(2).Add(text.New(item.UOM, bodyText))
			colQty := col.New(2).Add(text.New(fmt.Sprintf("%d", item.Qty), bodyTextRight))

			if cellStyle != nil {
				colSI = colSI.WithStyle(cellStyle)
				colDesc = colDesc.WithStyle(cellStyle)
				colHSN = colHSN.WithStyle(cellStyle)
				colUoM = colUoM.WithStyle(cellStyle)
				colQty = colQty.WithStyle(cellStyle)
			}

			m.AddRows(row.New(7).Add(colSI, colDesc, colHSN, colUoM, colQty))
		}
	} else {
		// Transit/Transfer DC: full pricing columns
		m.AddRows(
			row.New(8).Add(
				col.New(1).Add(text.New("SI No", headerText)).WithStyle(&headerCell),
				col.New(3).Add(text.New("Description", headerTextLeft)).WithStyle(&headerCell),
				col.New(1).Add(text.New("HSN", headerText)).WithStyle(&headerCell),
				col.New(1).Add(text.New("Qty", headerText)).WithStyle(&headerCell),
				col.New(1).Add(text.New("UoM", headerText)).WithStyle(&headerCell),
				col.New(1).Add(text.New("Rate", headerText)).WithStyle(&headerCell),
				col.New(1).Add(text.New("Taxable", headerText)).WithStyle(&headerCell),
				col.New(1).Add(text.New("Tax%", headerText)).WithStyle(&headerCell),
				col.New(1).Add(text.New("Tax Amt", headerText)).WithStyle(&headerCell),
				col.New(1).Add(text.New("Total", headerText)).WithStyle(&headerCell),
			),
		)

		altBg := &props.Color{Red: 248, Green: 249, Blue: 250}
		for i, item := range data.LineItems {
			bodyText := props.Text{Size: 7, Align: align.Center}
			bodyTextLeft := props.Text{Size: 7, Align: align.Left}
			bodyTextRight := props.Text{Size: 7, Align: align.Right}

			var cellStyle *props.Cell
			if i%2 == 1 {
				cellStyle = &props.Cell{BackgroundColor: altBg}
			}

			colSI := col.New(1).Add(text.New(fmt.Sprintf("%d", item.SINo), bodyText))
			colDesc := col.New(3).Add(text.New(item.Description, bodyTextLeft))
			colHSN := col.New(1).Add(text.New(item.HSNCode, bodyText))
			colQty := col.New(1).Add(text.New(fmt.Sprintf("%d", item.Qty), bodyTextRight))
			colUoM := col.New(1).Add(text.New(item.UOM, bodyText))
			colRate := col.New(1).Add(text.New(FormatINR(item.Rate), bodyTextRight))
			colTaxable := col.New(1).Add(text.New(FormatINR(item.Taxable), bodyTextRight))
			colTaxPct := col.New(1).Add(text.New(fmt.Sprintf("%.0f%%", item.TaxPercent), bodyText))
			colTaxAmt := col.New(1).Add(text.New(FormatINR(item.TaxAmount), bodyTextRight))
			colTotal := col.New(1).Add(text.New(FormatINR(item.Total), bodyTextRight))

			if cellStyle != nil {
				colSI = colSI.WithStyle(cellStyle)
				colDesc = colDesc.WithStyle(cellStyle)
				colHSN = colHSN.WithStyle(cellStyle)
				colQty = colQty.WithStyle(cellStyle)
				colUoM = colUoM.WithStyle(cellStyle)
				colRate = colRate.WithStyle(cellStyle)
				colTaxable = colTaxable.WithStyle(cellStyle)
				colTaxPct = colTaxPct.WithStyle(cellStyle)
				colTaxAmt = colTaxAmt.WithStyle(cellStyle)
				colTotal = colTotal.WithStyle(cellStyle)
			}

			m.AddRows(row.New(7).Add(
				colSI, colDesc, colHSN, colQty, colUoM,
				colRate, colTaxable, colTaxPct, colTaxAmt, colTotal,
			))
		}
	}

	m.AddRows(row.New(2))
}

func addDCTotals(m core.Maroto, data *DCExportData) {
	summaryBg := &props.Color{Red: 245, Green: 245, Blue: 245}
	summaryCell := &props.Cell{BackgroundColor: summaryBg}

	labelStyle := props.Text{Size: 8, Style: fontstyle.Bold, Align: align.Right}
	valueStyle := props.Text{Size: 8, Align: align.Right}

	m.AddRows(row.New(7).Add(
		col.New(9).Add(text.New("Taxable Amount", labelStyle)).WithStyle(summaryCell),
		col.New(3).Add(text.New(FormatINR(data.TotalTaxable), valueStyle)).WithStyle(summaryCell),
	))

	m.AddRows(row.New(7).Add(
		col.New(9).Add(text.New("Tax Amount", labelStyle)).WithStyle(summaryCell),
		col.New(3).Add(text.New(FormatINR(data.TotalTax), valueStyle)).WithStyle(summaryCell),
	))

	grandBg := &props.Color{Red: 33, Green: 37, Blue: 41}
	grandCell := &props.Cell{BackgroundColor: grandBg}
	grandLabel := props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right, Color: &props.Color{Red: 255, Green: 255, Blue: 255}}
	grandValue := props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right, Color: &props.Color{Red: 255, Green: 255, Blue: 255}}

	m.AddRows(row.New(8).Add(
		col.New(9).Add(text.New("Grand Total", grandLabel)).WithStyle(grandCell),
		col.New(3).Add(text.New(FormatINR(data.GrandTotal), grandValue)).WithStyle(grandCell),
	))

	m.AddRows(row.New(3))
}

func addDCDestinations(m core.Maroto, data *DCExportData) {
	sectionLabel := props.Text{
		Size:  7,
		Style: fontstyle.Bold,
		Align: align.Left,
		Color: &props.Color{Red: 100, Green: 100, Blue: 100},
	}
	headerBg := &props.Color{Red: 245, Green: 243, Blue: 239}
	headerCell := &props.Cell{BackgroundColor: headerBg}

	m.AddRows(row.New(7).Add(
		col.New(12).Add(text.New(fmt.Sprintf("DESTINATION PLAN (%d locations)", len(data.Destinations)), sectionLabel)).WithStyle(headerCell),
	))

	if data.HubAddress != "" {
		m.AddRows(row.New(7).Add(
			col.New(12).Add(text.New(fmt.Sprintf("Hub: %s", data.HubAddress), props.Text{Size: 8, Align: align.Left})),
		))
	}

	for i, dest := range data.Destinations {
		m.AddRows(row.New(6).Add(
			col.New(12).Add(text.New(fmt.Sprintf("%d. %s", i+1, dest), props.Text{Size: 8, Align: align.Left})),
		))
	}

	m.AddRows(row.New(3))
}

func addDCSerials(m core.Maroto, data *DCExportData) {
	hasSerials := false
	for _, item := range data.LineItems {
		if len(item.Serials) > 0 {
			hasSerials = true
			break
		}
	}
	if !hasSerials {
		return
	}

	sectionLabel := props.Text{
		Size:  7,
		Style: fontstyle.Bold,
		Align: align.Left,
		Color: &props.Color{Red: 100, Green: 100, Blue: 100},
	}
	headerBg := &props.Color{Red: 245, Green: 243, Blue: 239}
	headerCell := &props.Cell{BackgroundColor: headerBg}

	m.AddRows(row.New(7).Add(
		col.New(12).Add(text.New("SERIAL NUMBERS", sectionLabel)).WithStyle(headerCell),
	))

	for _, item := range data.LineItems {
		if len(item.Serials) == 0 {
			continue
		}
		m.AddRows(row.New(6).Add(
			col.New(12).Add(text.New(
				fmt.Sprintf("%s (%d serials)", item.Description, len(item.Serials)),
				props.Text{Size: 8, Style: fontstyle.Bold, Align: align.Left},
			)),
		))
		// Serials in comma-separated lines
		serialText := strings.Join(item.Serials, ", ")
		m.AddRows(row.New(7).Add(
			col.New(12).Add(text.New(serialText, props.Text{Size: 7, Align: align.Left})),
		))
	}

	m.AddRows(row.New(3))
}

func addDCSignatures(m core.Maroto) {
	m.AddRows(row.New(10))

	lineStyle := props.Text{
		Size:  8,
		Align: align.Center,
		Color: &props.Color{Red: 100, Green: 100, Blue: 100},
	}

	m.AddRows(row.New(6).Add(
		col.New(6).Add(text.New("____________________________", lineStyle)),
		col.New(6).Add(text.New("____________________________", lineStyle)),
	))

	labelStyle := props.Text{
		Size:  7,
		Style: fontstyle.Bold,
		Align: align.Center,
		Color: &props.Color{Red: 100, Green: 100, Blue: 100},
	}

	m.AddRows(row.New(7).Add(
		col.New(6).Add(text.New("Receiver Signature", labelStyle)),
		col.New(6).Add(text.New("Authorized Signatory", labelStyle)),
	))
}
