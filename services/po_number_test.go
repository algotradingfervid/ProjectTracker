package services

import (
	"testing"
	"time"
)

func TestGeneratePONumber_FiscalYear(t *testing.T) {
	tests := []struct {
		name   string
		date   time.Time
		expect string
	}{
		{"april_start", time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), "26-27"},
		{"march_end", time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC), "25-26"},
		{"january", time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC), "25-26"},
		{"may", time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC), "26-27"},
		{"december", time.Date(2025, time.December, 31, 0, 0, 0, 0, time.UTC), "25-26"},
		{"april_2025", time.Date(2025, time.April, 1, 0, 0, 0, 0, time.UTC), "25-26"},
		{"year_2000", time.Date(2000, time.June, 1, 0, 0, 0, 0, time.UTC), "00-01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFiscalYear(tt.date)
			if got != tt.expect {
				t.Errorf("GetFiscalYear(%v) = %q, want %q", tt.date, got, tt.expect)
			}
		})
	}
}

func TestPONumberFormat(t *testing.T) {
	// Test the format construction logic directly
	// Format: FSS-PO-{ref}/{fiscal_year}/{sequence}

	t.Run("format_with_ref", func(t *testing.T) {
		ref := "G10X-4"
		fy := "25-26"
		seq := 1
		expected := "FSS-PO-G10X-4/25-26/001"
		got := formatPONumber(ref, fy, seq)
		if got != expected {
			t.Errorf("formatPONumber(%q, %q, %d) = %q, want %q", ref, fy, seq, got, expected)
		}
	})

	t.Run("format_sequential", func(t *testing.T) {
		ref := "G10X-4"
		fy := "25-26"
		seq := 4
		expected := "FSS-PO-G10X-4/25-26/004"
		got := formatPONumber(ref, fy, seq)
		if got != expected {
			t.Errorf("formatPONumber(%q, %q, %d) = %q, want %q", ref, fy, seq, got, expected)
		}
	})

	t.Run("format_high_number", func(t *testing.T) {
		ref := "PROJ-123"
		fy := "26-27"
		seq := 99
		expected := "FSS-PO-PROJ-123/26-27/099"
		got := formatPONumber(ref, fy, seq)
		if got != expected {
			t.Errorf("formatPONumber(%q, %q, %d) = %q, want %q", ref, fy, seq, got, expected)
		}
	})
}
