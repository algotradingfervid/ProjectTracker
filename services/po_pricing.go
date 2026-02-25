package services

import (
	"math"
	"strings"
)

// POLineItemCalc holds the calculated totals for a single PO line item.
type POLineItemCalc struct {
	Rate       float64
	Qty        float64
	GSTPercent float64
	BeforeGST  float64 // Rate * Qty
	GSTAmount  float64 // BeforeGST * GSTPercent / 100
	Total      float64 // BeforeGST + GSTAmount
}

// POTotals holds the aggregated totals for a purchase order.
type POTotals struct {
	TotalBeforeTax float64
	IGSTPercent    float64
	IGSTAmount     float64
	RoundOff       float64
	GrandTotal     float64
}

// CalcPOLineItem calculates the totals for a single PO line item.
func CalcPOLineItem(rate, qty, gstPercent float64) POLineItemCalc {
	beforeGST := rate * qty
	gstAmount := beforeGST * gstPercent / 100
	return POLineItemCalc{
		Rate:       rate,
		Qty:        qty,
		GSTPercent: gstPercent,
		BeforeGST:  beforeGST,
		GSTAmount:  gstAmount,
		Total:      beforeGST + gstAmount,
	}
}

// CalcPOTotals computes the aggregate totals for all line items in a PO.
// It sums all line item totals, computes IGST based on the weighted average
// GST percent, applies round-off to nearest rupee, and returns the grand total.
func CalcPOTotals(items []POLineItemCalc) POTotals {
	var totalBeforeTax float64
	var totalGST float64

	for _, item := range items {
		totalBeforeTax += item.BeforeGST
		totalGST += item.GSTAmount
	}

	// Determine effective IGST percent from the items
	var igstPercent float64
	if totalBeforeTax > 0 {
		igstPercent = (totalGST / totalBeforeTax) * 100
	}

	subtotal := totalBeforeTax + totalGST
	roundOff := calcRoundOff(subtotal)
	grandTotal := subtotal + roundOff

	return POTotals{
		TotalBeforeTax: totalBeforeTax,
		IGSTPercent:    igstPercent,
		IGSTAmount:     totalGST,
		RoundOff:       roundOff,
		GrandTotal:     grandTotal,
	}
}

// calcRoundOff rounds to nearest rupee with ±0.50 threshold.
// If fractional part < 0.50, round down (negative round-off).
// If fractional part >= 0.50, round up (positive round-off).
func calcRoundOff(amount float64) float64 {
	rounded := math.Round(amount)
	return rounded - amount
}

// AmountToWords converts a numeric amount to Indian English words.
// Example: 913183.00 → "Nine Lakhs Thirteen Thousand One Hundred and Eighty Three Rupees Only/-"
func AmountToWords(amount float64) string {
	if amount < 0 {
		return "Negative " + AmountToWords(-amount)
	}

	rupees := int64(math.Round(amount))

	if rupees == 0 {
		return "Zero Rupees Only/-"
	}

	words := convertToIndianWords(rupees)
	return words + " Rupees Only/-"
}

func convertToIndianWords(n int64) string {
	if n == 0 {
		return ""
	}

	var parts []string

	// Crores (10,000,000)
	if n >= 10000000 {
		crores := n / 10000000
		parts = append(parts, convertUnder100(crores)+" Crores")
		n %= 10000000
	}

	// Lakhs (100,000)
	if n >= 100000 {
		lakhs := n / 100000
		parts = append(parts, convertUnder100(lakhs)+" Lakhs")
		n %= 100000
	}

	// Thousands (1,000)
	if n >= 1000 {
		thousands := n / 1000
		parts = append(parts, convertUnder100(thousands)+" Thousand")
		n %= 1000
	}

	// Hundreds
	if n >= 100 {
		hundreds := n / 100
		parts = append(parts, ones[hundreds]+" Hundred")
		n %= 100
	}

	// Remaining (1-99)
	if n > 0 {
		if len(parts) > 0 {
			parts = append(parts, "and "+convertUnder100(n))
		} else {
			parts = append(parts, convertUnder100(n))
		}
	}

	return strings.Join(parts, " ")
}

func convertUnder100(n int64) string {
	if n < 20 {
		return ones[n]
	}
	result := tens[n/10]
	if n%10 != 0 {
		result += " " + ones[n%10]
	}
	return result
}

var ones = []string{
	"", "One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine",
	"Ten", "Eleven", "Twelve", "Thirteen", "Fourteen", "Fifteen", "Sixteen",
	"Seventeen", "Eighteen", "Nineteen",
}

var tens = []string{
	"", "", "Twenty", "Thirty", "Forty", "Fifty", "Sixty", "Seventy", "Eighty", "Ninety",
}
