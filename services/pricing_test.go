package services

import (
	"math"
	"testing"
)

func TestCalcSubSubItemBudgeted(t *testing.T) {
	tests := []struct {
		name       string
		qtyPerUnit float64
		unitPrice  float64
		expect     float64
	}{
		{"basic multiplication", 10, 50, 500},
		{"zero qty", 0, 100, 0},
		{"zero price", 5, 0, 0},
		{"decimal values", 2.5, 100.50, 251.25},
		{"both zero", 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcSubSubItemBudgeted(tt.qtyPerUnit, tt.unitPrice)
			if got != tt.expect {
				t.Errorf("CalcSubSubItemBudgeted(%v, %v) = %v, want %v",
					tt.qtyPerUnit, tt.unitPrice, got, tt.expect)
			}
		})
	}
}

func TestCalcSubItemBudgeted(t *testing.T) {
	tests := []struct {
		name       string
		subPrices  []float64
		qtyPerUnit float64
		unitPrice  float64
		expect     float64
	}{
		{"with sub-sub items", []float64{100, 200, 300}, 0, 0, 600},
		{"without sub-sub items", nil, 10, 50, 500},
		{"empty sub-sub items", []float64{}, 10, 50, 500},
		{"single sub-sub item", []float64{250}, 0, 0, 250},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcSubItemBudgeted(tt.subPrices, tt.qtyPerUnit, tt.unitPrice)
			if got != tt.expect {
				t.Errorf("CalcSubItemBudgeted(%v, %v, %v) = %v, want %v",
					tt.subPrices, tt.qtyPerUnit, tt.unitPrice, got, tt.expect)
			}
		})
	}
}

func TestCalcMainItemBudgeted(t *testing.T) {
	tests := []struct {
		name           string
		subPrices      []float64
		manualBudgeted float64
		expect         float64
	}{
		{"with sub items", []float64{500, 300}, 0, 800},
		{"without sub items", nil, 1500, 1500},
		{"empty sub items", []float64{}, 1500, 1500},
		{"sub items override manual", []float64{100}, 9999, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcMainItemBudgeted(tt.subPrices, tt.manualBudgeted)
			if got != tt.expect {
				t.Errorf("CalcMainItemBudgeted(%v, %v) = %v, want %v",
					tt.subPrices, tt.manualBudgeted, got, tt.expect)
			}
		})
	}
}

func TestCalcMainItemTotal(t *testing.T) {
	tests := []struct {
		name          string
		budgetedPrice float64
		qty           float64
		expect        float64
	}{
		{"basic", 500, 10, 5000},
		{"zero qty", 500, 0, 0},
		{"zero price", 0, 10, 0},
		{"decimal", 99.99, 3, 299.97},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcMainItemTotal(tt.budgetedPrice, tt.qty)
			if math.Abs(got-tt.expect) > 0.001 {
				t.Errorf("CalcMainItemTotal(%v, %v) = %v, want %v",
					tt.budgetedPrice, tt.qty, got, tt.expect)
			}
		})
	}
}

func TestCalcBOQTotals(t *testing.T) {
	tests := []struct {
		name          string
		items         []MainItemForTotals
		expectQuoted  float64
		expectBudget  float64
		expectMargin  float64
		expectPercent float64
	}{
		{
			name: "basic margin",
			items: []MainItemForTotals{
				{Qty: 1, QuotedPrice: 1000, BudgetedPrice: 800},
			},
			expectQuoted:  1000,
			expectBudget:  800,
			expectMargin:  200,
			expectPercent: 20,
		},
		{
			name: "multiple items",
			items: []MainItemForTotals{
				{Qty: 2, QuotedPrice: 500, BudgetedPrice: 400},
				{Qty: 1, QuotedPrice: 1000, BudgetedPrice: 600},
			},
			expectQuoted:  2000,
			expectBudget:  1000,
			expectMargin:  1000,
			expectPercent: 50,
		},
		{
			name:          "empty items",
			items:         []MainItemForTotals{},
			expectQuoted:  0,
			expectBudget:  0,
			expectMargin:  0,
			expectPercent: 0,
		},
		{
			name:          "nil items",
			items:         nil,
			expectQuoted:  0,
			expectBudget:  0,
			expectMargin:  0,
			expectPercent: 0,
		},
		{
			name: "zero quoted price",
			items: []MainItemForTotals{
				{Qty: 1, QuotedPrice: 0, BudgetedPrice: 500},
			},
			expectQuoted:  0,
			expectBudget:  500,
			expectMargin:  -500,
			expectPercent: 0, // division by zero guarded
		},
		{
			name: "negative margin",
			items: []MainItemForTotals{
				{Qty: 1, QuotedPrice: 800, BudgetedPrice: 1000},
			},
			expectQuoted:  800,
			expectBudget:  1000,
			expectMargin:  -200,
			expectPercent: -25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcBOQTotals(tt.items)
			if math.Abs(got.TotalQuoted-tt.expectQuoted) > 0.001 {
				t.Errorf("TotalQuoted = %v, want %v", got.TotalQuoted, tt.expectQuoted)
			}
			if math.Abs(got.TotalBudgeted-tt.expectBudget) > 0.001 {
				t.Errorf("TotalBudgeted = %v, want %v", got.TotalBudgeted, tt.expectBudget)
			}
			if math.Abs(got.Margin-tt.expectMargin) > 0.001 {
				t.Errorf("Margin = %v, want %v", got.Margin, tt.expectMargin)
			}
			if math.Abs(got.MarginPercent-tt.expectPercent) > 0.001 {
				t.Errorf("MarginPercent = %v, want %v", got.MarginPercent, tt.expectPercent)
			}
		})
	}
}
