// Package services provides pricing calculation functions for BOQ items.
package services

func CalcSubSubItemBudgeted(qtyPerUnit, unitPrice float64) float64 {
	return qtyPerUnit * unitPrice
}

func CalcSubItemBudgeted(subSubBudgetedPrices []float64, qtyPerUnit, unitPrice float64) float64 {
	if len(subSubBudgetedPrices) > 0 {
		var sum float64
		for _, p := range subSubBudgetedPrices {
			sum += p
		}
		return sum
	}
	return qtyPerUnit * unitPrice
}

func CalcMainItemBudgeted(subBudgetedPrices []float64, manualBudgeted float64) float64 {
	if len(subBudgetedPrices) > 0 {
		var sum float64
		for _, p := range subBudgetedPrices {
			sum += p
		}
		return sum
	}
	return manualBudgeted
}

func CalcMainItemTotal(budgetedPrice, qty float64) float64 {
	return budgetedPrice * qty
}

type BOQTotals struct {
	TotalQuoted   float64
	TotalBudgeted float64
	Margin        float64
	MarginPercent float64
}

type MainItemForTotals struct {
	Qty           float64
	QuotedPrice   float64
	BudgetedPrice float64
}

func CalcBOQTotals(items []MainItemForTotals) BOQTotals {
	var totals BOQTotals
	for _, item := range items {
		totals.TotalQuoted += item.QuotedPrice * item.Qty
		totals.TotalBudgeted += item.BudgetedPrice
	}
	totals.Margin = totals.TotalQuoted - totals.TotalBudgeted
	if totals.TotalQuoted != 0 {
		totals.MarginPercent = (totals.Margin / totals.TotalQuoted) * 100
	}
	return totals
}
