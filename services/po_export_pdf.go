package services

import (
	"fmt"

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

// GeneratePOPDF creates a PDF document for a Purchase Order using maroto/v2.
// It returns the raw PDF bytes or an error.
func GeneratePOPDF(data *POExportData) ([]byte, error) {
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

	addPOHeader(m, data)
	addPOVendorBlock(m, data)
	addPOAddresses(m, data)
	addPOLineItemsTable(m, data)
	addPOTotals(m, data)
	addPOAmountInWords(m, data)
	addPOComments(m, data)
	addPOTerms(m, data)
	addPOBankDetails(m, data)
	addPOSignatures(m)

	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate PO PDF: %w", err)
	}

	return doc.GetBytes(), nil
}

// addPOHeader adds company name, "PURCHASE ORDER" title, address, email, and PO number.
func addPOHeader(m core.Maroto, data *POExportData) {
	// Row 1: Company name (left) + PURCHASE ORDER title (right)
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
				text.New("PURCHASE ORDER", props.Text{
					Size:  14,
					Style: fontstyle.Bold,
					Align: align.Right,
					Color: &props.Color{Red: 33, Green: 37, Blue: 41},
				}),
			),
		),
	)

	// Row 2: Company address + email (left) + PO number (right)
	m.AddRows(
		row.New(8).Add(
			col.New(6).Add(
				text.New(fmt.Sprintf("%s | %s", data.CompanyAddress, data.CompanyEmail), props.Text{
					Size:  8,
					Align: align.Left,
					Color: &props.Color{Red: 100, Green: 100, Blue: 100},
				}),
			),
			col.New(6).Add(
				text.New(fmt.Sprintf("PO #: %s", data.PONumber), props.Text{
					Size:  10,
					Style: fontstyle.Bold,
					Align: align.Right,
				}),
			),
		),
	)

	// Divider spacer
	m.AddRows(row.New(3))
}

// addPOVendorBlock adds vendor details on the left and order metadata on the right.
func addPOVendorBlock(m core.Maroto, data *POExportData) {
	labelStyle := props.Text{
		Size:  7,
		Style: fontstyle.Bold,
		Align: align.Left,
		Color: &props.Color{Red: 100, Green: 100, Blue: 100},
	}
	valueStyle := props.Text{
		Size:  8,
		Align: align.Left,
	}
	rightLabelStyle := props.Text{
		Size:  7,
		Style: fontstyle.Bold,
		Align: align.Right,
		Color: &props.Color{Red: 100, Green: 100, Blue: 100},
	}
	rightValueStyle := props.Text{
		Size:  8,
		Align: align.Right,
	}

	// Section label
	m.AddRows(
		row.New(6).Add(
			col.New(6).Add(text.New("VENDOR", labelStyle)),
			col.New(6).Add(text.New("ORDER DETAILS", rightLabelStyle)),
		),
	)

	// Vendor name + Order Date
	m.AddRows(
		row.New(7).Add(
			col.New(6).Add(text.New(data.Vendor.Name, props.Text{
				Size:  9,
				Style: fontstyle.Bold,
				Align: align.Left,
			})),
			col.New(3).Add(text.New("Order Date:", rightLabelStyle)),
			col.New(3).Add(text.New(data.OrderDate, rightValueStyle)),
		),
	)

	// Vendor address + Quotation Ref
	if data.Vendor.Address != "" {
		m.AddRows(
			row.New(7).Add(
				col.New(6).Add(text.New(data.Vendor.Address, valueStyle)),
				col.New(3).Add(text.New("Quotation Ref:", rightLabelStyle)),
				col.New(3).Add(text.New(data.QuotationRef, rightValueStyle)),
			),
		)
	}

	// GSTIN + Ref Date
	if data.Vendor.GSTIN != "" {
		m.AddRows(
			row.New(7).Add(
				col.New(6).Add(text.New(fmt.Sprintf("GSTIN: %s", data.Vendor.GSTIN), valueStyle)),
				col.New(3).Add(text.New("Ref Date:", rightLabelStyle)),
				col.New(3).Add(text.New(data.RefDate, rightValueStyle)),
			),
		)
	}

	// Contact info
	contactParts := []string{}
	if data.Vendor.ContactName != "" {
		contactParts = append(contactParts, data.Vendor.ContactName)
	}
	if data.Vendor.Phone != "" {
		contactParts = append(contactParts, data.Vendor.Phone)
	}
	if data.Vendor.Email != "" {
		contactParts = append(contactParts, data.Vendor.Email)
	}
	if len(contactParts) > 0 {
		m.AddRows(
			row.New(7).Add(
				col.New(12).Add(text.New(fmt.Sprintf("Contact: %s", joinNonEmpty(contactParts, " | ")), valueStyle)),
			),
		)
	}

	m.AddRows(row.New(3))
}

// addPOAddresses adds Bill To and Ship To address blocks side by side.
func addPOAddresses(m core.Maroto, data *POExportData) {
	sectionLabel := props.Text{
		Size:  7,
		Style: fontstyle.Bold,
		Align: align.Left,
		Color: &props.Color{Red: 100, Green: 100, Blue: 100},
	}
	valueStyle := props.Text{
		Size:  8,
		Align: align.Left,
	}
	boldValue := props.Text{
		Size:  8,
		Style: fontstyle.Bold,
		Align: align.Left,
	}

	headerBg := &props.Color{Red: 245, Green: 243, Blue: 239}
	headerCell := &props.Cell{BackgroundColor: headerBg}

	// Section headers
	m.AddRows(
		row.New(7).Add(
			col.New(6).Add(text.New("BILL TO", sectionLabel)).WithStyle(headerCell),
			col.New(6).Add(text.New("SHIP TO", sectionLabel)).WithStyle(headerCell),
		),
	)

	// Address content
	billCompany := ""
	billAddr := ""
	billContact := ""
	billPerson := ""
	billGSTIN := ""

	if data.BillTo != nil {
		billCompany = data.BillTo.CompanyName
		billAddr = data.BillTo.AddressLines
		billContact = data.BillTo.ContactNo
		billPerson = data.BillTo.ContactPerson
		billGSTIN = data.BillTo.GSTIN
	}

	shipCompany := ""
	shipAddr := ""
	shipContact := ""
	shipPerson := ""
	shipGSTIN := ""

	if data.ShipTo != nil {
		shipCompany = data.ShipTo.CompanyName
		shipAddr = data.ShipTo.AddressLines
		shipContact = data.ShipTo.ContactNo
		shipPerson = data.ShipTo.ContactPerson
		shipGSTIN = data.ShipTo.GSTIN
	}

	// Company names
	m.AddRows(
		row.New(7).Add(
			col.New(6).Add(text.New(billCompany, boldValue)),
			col.New(6).Add(text.New(shipCompany, boldValue)),
		),
	)

	// Address lines
	m.AddRows(
		row.New(7).Add(
			col.New(6).Add(text.New(billAddr, valueStyle)),
			col.New(6).Add(text.New(shipAddr, valueStyle)),
		),
	)

	// Contact person
	if billPerson != "" || shipPerson != "" {
		m.AddRows(
			row.New(7).Add(
				col.New(6).Add(text.New(fmtField("Contact", billPerson), valueStyle)),
				col.New(6).Add(text.New(fmtField("Contact", shipPerson), valueStyle)),
			),
		)
	}

	// Contact number
	if billContact != "" || shipContact != "" {
		m.AddRows(
			row.New(7).Add(
				col.New(6).Add(text.New(fmtField("Phone", billContact), valueStyle)),
				col.New(6).Add(text.New(fmtField("Phone", shipContact), valueStyle)),
			),
		)
	}

	// GSTIN
	if billGSTIN != "" || shipGSTIN != "" {
		m.AddRows(
			row.New(7).Add(
				col.New(6).Add(text.New(fmtField("GSTIN", billGSTIN), valueStyle)),
				col.New(6).Add(text.New(fmtField("GSTIN", shipGSTIN), valueStyle)),
			),
		)
	}

	m.AddRows(row.New(3))
}

// addPOLineItemsTable adds the line items table with header and body rows.
func addPOLineItemsTable(m core.Maroto, data *POExportData) {
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

	// Table header
	m.AddRows(
		row.New(8).Add(
			col.New(1).Add(text.New("SI No", headerText)).WithStyle(&headerCell),
			col.New(3).Add(text.New("Description", headerTextLeft)).WithStyle(&headerCell),
			col.New(1).Add(text.New("HSN", headerText)).WithStyle(&headerCell),
			col.New(1).Add(text.New("Qty", headerText)).WithStyle(&headerCell),
			col.New(1).Add(text.New("UoM", headerText)).WithStyle(&headerCell),
			col.New(1).Add(text.New("Rate", headerText)).WithStyle(&headerCell),
			col.New(1).Add(text.New("Before GST", headerText)).WithStyle(&headerCell),
			col.New(1).Add(text.New("GST%", headerText)).WithStyle(&headerCell),
			col.New(1).Add(text.New("GST Amt", headerText)).WithStyle(&headerCell),
			col.New(1).Add(text.New("Total", headerText)).WithStyle(&headerCell),
		),
	)

	// Table body with alternating backgrounds
	altBg := &props.Color{Red: 248, Green: 249, Blue: 250}

	for i, item := range data.LineItems {
		bodyText := props.Text{Size: 7, Align: align.Center}
		bodyTextLeft := props.Text{Size: 7, Align: align.Left}
		bodyTextRight := props.Text{Size: 7, Align: align.Right}

		var cellStyle *props.Cell
		if i%2 == 1 {
			cellStyle = &props.Cell{BackgroundColor: altBg}
		}

		colSINo := col.New(1).Add(text.New(fmt.Sprintf("%d", item.SINo), bodyText))
		colDesc := col.New(3).Add(text.New(item.Description, bodyTextLeft))
		colHSN := col.New(1).Add(text.New(item.HSNCode, bodyText))
		colQty := col.New(1).Add(text.New(formatQty(item.Qty), bodyTextRight))
		colUoM := col.New(1).Add(text.New(item.UoM, bodyText))
		colRate := col.New(1).Add(text.New(FormatINR(item.Rate), bodyTextRight))
		colBeforeGST := col.New(1).Add(text.New(FormatINR(item.BeforeGST), bodyTextRight))
		colGSTPct := col.New(1).Add(text.New(fmt.Sprintf("%.0f%%", item.GSTPercent), bodyText))
		colGSTAmt := col.New(1).Add(text.New(FormatINR(item.GSTAmount), bodyTextRight))
		colTotal := col.New(1).Add(text.New(FormatINR(item.TotalAmount), bodyTextRight))

		if cellStyle != nil {
			colSINo = colSINo.WithStyle(cellStyle)
			colDesc = colDesc.WithStyle(cellStyle)
			colHSN = colHSN.WithStyle(cellStyle)
			colQty = colQty.WithStyle(cellStyle)
			colUoM = colUoM.WithStyle(cellStyle)
			colRate = colRate.WithStyle(cellStyle)
			colBeforeGST = colBeforeGST.WithStyle(cellStyle)
			colGSTPct = colGSTPct.WithStyle(cellStyle)
			colGSTAmt = colGSTAmt.WithStyle(cellStyle)
			colTotal = colTotal.WithStyle(cellStyle)
		}

		m.AddRows(
			row.New(7).Add(
				colSINo, colDesc, colHSN, colQty, colUoM,
				colRate, colBeforeGST, colGSTPct, colGSTAmt, colTotal,
			),
		)
	}

	m.AddRows(row.New(2))
}

// addPOTotals adds right-aligned total rows.
func addPOTotals(m core.Maroto, data *POExportData) {
	summaryBg := &props.Color{Red: 245, Green: 245, Blue: 245}
	summaryCell := &props.Cell{BackgroundColor: summaryBg}

	labelStyle := props.Text{
		Size:  8,
		Style: fontstyle.Bold,
		Align: align.Right,
	}
	valueStyle := props.Text{
		Size:  8,
		Align: align.Right,
	}
	grandLabelStyle := props.Text{
		Size:  9,
		Style: fontstyle.Bold,
		Align: align.Right,
	}
	grandValueStyle := props.Text{
		Size:  9,
		Style: fontstyle.Bold,
		Align: align.Right,
	}

	// Total Before Tax
	m.AddRows(
		row.New(7).Add(
			col.New(9).Add(text.New("Total Before Tax", labelStyle)).WithStyle(summaryCell),
			col.New(3).Add(text.New(FormatINR(data.TotalBeforeTax), valueStyle)).WithStyle(summaryCell),
		),
	)

	// IGST
	igstLabel := fmt.Sprintf("IGST %.0f%%", data.IGSTPercent)
	m.AddRows(
		row.New(7).Add(
			col.New(9).Add(text.New(igstLabel, labelStyle)).WithStyle(summaryCell),
			col.New(3).Add(text.New(FormatINR(data.IGSTAmount), valueStyle)).WithStyle(summaryCell),
		),
	)

	// Round Off
	m.AddRows(
		row.New(7).Add(
			col.New(9).Add(text.New("Round Off", labelStyle)).WithStyle(summaryCell),
			col.New(3).Add(text.New(FormatINR(data.RoundOff), valueStyle)).WithStyle(summaryCell),
		),
	)

	// Grand Total
	grandBg := &props.Color{Red: 33, Green: 37, Blue: 41}
	grandCell := &props.Cell{BackgroundColor: grandBg}
	grandLabelStyle.Color = &props.Color{Red: 255, Green: 255, Blue: 255}
	grandValueStyle.Color = &props.Color{Red: 255, Green: 255, Blue: 255}

	m.AddRows(
		row.New(8).Add(
			col.New(9).Add(text.New("Grand Total", grandLabelStyle)).WithStyle(grandCell),
			col.New(3).Add(text.New(FormatINR(data.GrandTotal), grandValueStyle)).WithStyle(grandCell),
		),
	)

	m.AddRows(row.New(3))
}

// addPOAmountInWords adds the amount in words row.
func addPOAmountInWords(m core.Maroto, data *POExportData) {
	if data.AmountInWords == "" {
		return
	}

	m.AddRows(
		row.New(8).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("Amount in Words: %s", data.AmountInWords), props.Text{
					Size:  8,
					Style: fontstyle.BoldItalic,
					Align: align.Left,
				}),
			),
		),
	)

	m.AddRows(row.New(3))
}

// addPOComments adds the comments section if non-empty.
func addPOComments(m core.Maroto, data *POExportData) {
	if data.Comments == "" {
		return
	}

	sectionLabel := props.Text{
		Size:  7,
		Style: fontstyle.Bold,
		Align: align.Left,
		Color: &props.Color{Red: 100, Green: 100, Blue: 100},
	}

	m.AddRows(
		row.New(6).Add(
			col.New(12).Add(text.New("COMMENTS", sectionLabel)),
		),
	)
	m.AddRows(
		row.New(7).Add(
			col.New(12).Add(text.New(data.Comments, props.Text{
				Size:  8,
				Align: align.Left,
			})),
		),
	)

	m.AddRows(row.New(3))
}

// addPOTerms adds terms and conditions sections.
func addPOTerms(m core.Maroto, data *POExportData) {
	hasTerms := data.PaymentTerms != "" || data.DeliveryTerms != "" || data.WarrantyTerms != ""
	if !hasTerms {
		return
	}

	sectionLabel := props.Text{
		Size:  8,
		Style: fontstyle.Bold,
		Align: align.Left,
		Color: &props.Color{Red: 33, Green: 37, Blue: 41},
	}
	termLabel := props.Text{
		Size:  7,
		Style: fontstyle.Bold,
		Align: align.Left,
		Color: &props.Color{Red: 100, Green: 100, Blue: 100},
	}
	termValue := props.Text{
		Size:  8,
		Align: align.Left,
	}

	m.AddRows(
		row.New(7).Add(
			col.New(12).Add(text.New("TERMS & CONDITIONS", sectionLabel)),
		),
	)

	if data.PaymentTerms != "" {
		m.AddRows(
			row.New(6).Add(col.New(12).Add(text.New("Payment Terms", termLabel))),
		)
		m.AddRows(
			row.New(7).Add(col.New(12).Add(text.New(data.PaymentTerms, termValue))),
		)
	}

	if data.DeliveryTerms != "" {
		m.AddRows(
			row.New(6).Add(col.New(12).Add(text.New("Delivery Terms", termLabel))),
		)
		m.AddRows(
			row.New(7).Add(col.New(12).Add(text.New(data.DeliveryTerms, termValue))),
		)
	}

	if data.WarrantyTerms != "" {
		m.AddRows(
			row.New(6).Add(col.New(12).Add(text.New("Warranty Terms", termLabel))),
		)
		m.AddRows(
			row.New(7).Add(col.New(12).Add(text.New(data.WarrantyTerms, termValue))),
		)
	}

	m.AddRows(row.New(3))
}

// addPOBankDetails adds vendor bank details section.
func addPOBankDetails(m core.Maroto, data *POExportData) {
	v := data.Vendor
	hasBankDetails := v.BankBeneficiaryName != "" || v.BankName != "" || v.BankAccountNo != "" || v.BankIFSC != ""
	if !hasBankDetails {
		return
	}

	sectionLabel := props.Text{
		Size:  8,
		Style: fontstyle.Bold,
		Align: align.Left,
		Color: &props.Color{Red: 33, Green: 37, Blue: 41},
	}
	fieldLabel := props.Text{
		Size:  7,
		Style: fontstyle.Bold,
		Align: align.Left,
		Color: &props.Color{Red: 100, Green: 100, Blue: 100},
	}
	fieldValue := props.Text{
		Size:  8,
		Align: align.Left,
	}

	m.AddRows(
		row.New(7).Add(
			col.New(12).Add(text.New("BANK DETAILS (VENDOR)", sectionLabel)),
		),
	)

	bankRows := []struct{ label, value string }{
		{"Beneficiary Name", v.BankBeneficiaryName},
		{"Bank Name", v.BankName},
		{"Account No", v.BankAccountNo},
		{"IFSC Code", v.BankIFSC},
		{"Branch", v.BankBranch},
	}

	for _, br := range bankRows {
		if br.value == "" {
			continue
		}
		m.AddRows(
			row.New(7).Add(
				col.New(3).Add(text.New(br.label, fieldLabel)),
				col.New(9).Add(text.New(br.value, fieldValue)),
			),
		)
	}

	m.AddRows(row.New(3))
}

// addPOSignatures adds the signature section at the bottom.
func addPOSignatures(m core.Maroto) {
	m.AddRows(row.New(10))

	lineStyle := props.Text{
		Size:  8,
		Align: align.Center,
		Color: &props.Color{Red: 100, Green: 100, Blue: 100},
	}

	m.AddRows(
		row.New(6).Add(
			col.New(6).Add(text.New("____________________________", lineStyle)),
			col.New(6).Add(text.New("____________________________", lineStyle)),
		),
	)

	labelStyle := props.Text{
		Size:  7,
		Style: fontstyle.Bold,
		Align: align.Center,
		Color: &props.Color{Red: 100, Green: 100, Blue: 100},
	}

	m.AddRows(
		row.New(7).Add(
			col.New(6).Add(text.New("Supplier Signature", labelStyle)),
			col.New(6).Add(text.New("Authorized Signatory / Purchase Department", labelStyle)),
		),
	)
}

// joinNonEmpty joins non-empty strings with the given separator.
func joinNonEmpty(parts []string, sep string) string {
	var nonEmpty []string
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	result := ""
	for i, p := range nonEmpty {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
}

// fmtField returns "label: value" if value is non-empty, otherwise empty string.
func fmtField(label, value string) string {
	if value == "" {
		return ""
	}
	return fmt.Sprintf("%s: %s", label, value)
}
