package services

import (
	"testing"
	"time"
)

func TestGeneratePONumber_FinancialYear(t *testing.T) {
	// Tests that the financial year calculation used by GeneratePONumber
	// (via NextDocNumber -> GetFinancialYear) produces correct results.
	tests := []struct {
		name   string
		date   time.Time
		expect string
	}{
		{"april_start", time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), "2627"},
		{"march_end", time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC), "2526"},
		{"january", time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC), "2526"},
		{"may", time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC), "2627"},
		{"december", time.Date(2025, time.December, 31, 0, 0, 0, 0, time.UTC), "2526"},
		{"april_2025", time.Date(2025, time.April, 1, 0, 0, 0, 0, time.UTC), "2526"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFinancialYear(tt.date)
			if got != tt.expect {
				t.Errorf("GetFinancialYear(%v) = %q, want %q", tt.date, got, tt.expect)
			}
		})
	}
}

func TestPONumberFormat(t *testing.T) {
	// Tests the configurable format used by the numbering service.
	// Default PO format: {PREFIX}{SEP}{TYPE}{SEP}{FY}{SEP}{SEQ}
	defaultFormat := "{PREFIX}{SEP}{TYPE}{SEP}{PROJECT_REF}{SEP}{FY}{SEP}{SEQ}"

	t.Run("format_with_ref", func(t *testing.T) {
		expected := "FSS-PO-G10X-4-2526-001"
		got := FormatDocNumber(defaultFormat, "-", "FSS", "po", "2526", 1, 3, "G10X-4")
		if got != expected {
			t.Errorf("got %q, want %q", got, expected)
		}
	})

	t.Run("format_sequential", func(t *testing.T) {
		expected := "FSS-PO-G10X-4-2526-004"
		got := FormatDocNumber(defaultFormat, "-", "FSS", "po", "2526", 4, 3, "G10X-4")
		if got != expected {
			t.Errorf("got %q, want %q", got, expected)
		}
	})

	t.Run("format_high_number", func(t *testing.T) {
		expected := "FSS-PO-PROJ-123-2627-099"
		got := FormatDocNumber(defaultFormat, "-", "FSS", "po", "2627", 99, 3, "PROJ-123")
		if got != expected {
			t.Errorf("got %q, want %q", got, expected)
		}
	})
}
