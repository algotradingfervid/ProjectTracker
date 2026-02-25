package services

import (
	"math"
	"testing"
)

func TestCalcPOLineItemTotals(t *testing.T) {
	tests := []struct {
		name       string
		rate       float64
		qty        float64
		gstPercent float64
		wantBefore float64
		wantGST    float64
		wantTotal  float64
	}{
		{"basic", 1000, 5, 18, 5000, 900, 5900},
		{"zero_qty", 1000, 0, 18, 0, 0, 0},
		{"zero_gst", 500, 10, 0, 5000, 0, 5000},
		{"fractional", 123.45, 3, 12, 370.35, 44.442, 414.792},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcPOLineItem(tt.rate, tt.qty, tt.gstPercent)
			if !floatClose(got.BeforeGST, tt.wantBefore) {
				t.Errorf("BeforeGST = %f, want %f", got.BeforeGST, tt.wantBefore)
			}
			if !floatClose(got.GSTAmount, tt.wantGST) {
				t.Errorf("GSTAmount = %f, want %f", got.GSTAmount, tt.wantGST)
			}
			if !floatClose(got.Total, tt.wantTotal) {
				t.Errorf("Total = %f, want %f", got.Total, tt.wantTotal)
			}
		})
	}
}

func TestCalcPOTotals(t *testing.T) {
	items := []POLineItemCalc{
		CalcPOLineItem(1000, 5, 18), // 5000 + 900 = 5900
		CalcPOLineItem(2000, 3, 18), // 6000 + 1080 = 7080
	}
	got := CalcPOTotals(items)

	if !floatClose(got.TotalBeforeTax, 11000) {
		t.Errorf("TotalBeforeTax = %f, want 11000", got.TotalBeforeTax)
	}
	if !floatClose(got.IGSTAmount, 1980) {
		t.Errorf("IGSTAmount = %f, want 1980", got.IGSTAmount)
	}
	if !floatClose(got.GrandTotal, 12980) {
		t.Errorf("GrandTotal = %f, want 12980", got.GrandTotal)
	}
}

func TestCalcPORoundOff(t *testing.T) {
	tests := []struct {
		name      string
		amount    float64
		wantRound float64
	}{
		{"exact", 5000.00, 0.00},
		{"round_down", 5000.30, -0.30},
		{"round_up", 5000.70, 0.30},
		{"at_threshold", 5000.50, 0.50}, // math.Round rounds half away from zero
		{"just_below", 5000.49, -0.49},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := []POLineItemCalc{{BeforeGST: tt.amount, GSTAmount: 0}}
			got := CalcPOTotals(items)
			if !floatClose(got.RoundOff, tt.wantRound) {
				t.Errorf("RoundOff for %f = %f, want %f", tt.amount, got.RoundOff, tt.wantRound)
			}
		})
	}
}

func TestAmountToWords_IndianFormat(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
		expect string
	}{
		{"zero", 0, "Zero Rupees Only/-"},
		{"single_digit", 5, "Five Rupees Only/-"},
		{"teens", 15, "Fifteen Rupees Only/-"},
		{"hundreds", 500, "Five Hundred Rupees Only/-"},
		{"thousands", 5000, "Five Thousand Rupees Only/-"},
		{"lakhs", 913183, "Nine Lakhs Thirteen Thousand One Hundred and Eighty Three Rupees Only/-"},
		{"crores", 12345678, "One Crores Twenty Three Lakhs Forty Five Thousand Six Hundred and Seventy Eight Rupees Only/-"},
		{"exact_lakh", 100000, "One Lakhs Rupees Only/-"},
		{"hundred_and", 150, "One Hundred and Fifty Rupees Only/-"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AmountToWords(tt.amount)
			if got != tt.expect {
				t.Errorf("AmountToWords(%v) = %q, want %q", tt.amount, got, tt.expect)
			}
		})
	}
}

func floatClose(a, b float64) bool {
	return math.Abs(a-b) < 0.001
}
