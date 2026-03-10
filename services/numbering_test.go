package services

import (
	"testing"
	"time"
)

func TestGetFinancialYear(t *testing.T) {
	tests := []struct {
		name   string
		month  int
		year   int
		expect string
	}{
		{"april_start", 4, 2025, "2526"},
		{"march_end", 3, 2026, "2526"},
		{"january", 1, 2026, "2526"},
		{"december", 12, 2025, "2526"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := time.Date(tt.year, time.Month(tt.month), 15, 0, 0, 0, 0, time.UTC)
			got := GetFinancialYear(d)
			if got != tt.expect {
				t.Errorf("GetFinancialYear(%v) = %q, want %q", d, got, tt.expect)
			}
		})
	}
}

func TestFormatDocNumber(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		sep     string
		prefix  string
		seqType string
		fy      string
		seq     int
		padding int
		projRef string
		expect  string
	}{
		{"po_standard", "{PREFIX}{SEP}{TYPE}{SEP}{PROJECT_REF}{SEP}{FY}{SEP}{SEQ}", "-", "FSS", "po", "2526", 1, 3, "OAVS", "FSS-PO-OAVS-2526-001"},
		{"dc_standard", "{PREFIX}{SEP}{TYPE}{SEP}{FY}{SEP}{SEQ}", "-", "ABC", "tdc", "2526", 1, 3, "", "ABC-TDC-2526-001"},
		{"dc_official", "{PREFIX}{SEP}{TYPE}{SEP}{FY}{SEP}{SEQ}", "-", "ABC", "odc", "2526", 5, 3, "", "ABC-ODC-2526-005"},
		{"dc_transfer", "{PREFIX}{SEP}{TYPE}{SEP}{FY}{SEP}{SEQ}", "-", "ABC", "stdc", "2526", 1, 4, "", "ABC-STDC-2526-0001"},
		{"custom_sep", "{PREFIX}{SEP}{TYPE}{SEP}{FY}{SEP}{SEQ}", "/", "XYZ", "odc", "2526", 42, 4, "", "XYZ/ODC/2526/0042"},
		{"po_with_ref", "{PREFIX}{SEP}{TYPE}{SEP}{PROJECT_REF}{SEP}{FY}{SEP}{SEQ}", "-", "FSS", "po", "2526", 42, 4, "OAVS", "FSS-PO-OAVS-2526-0042"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDocNumber(tt.format, tt.sep, tt.prefix, tt.seqType, tt.fy, tt.seq, tt.padding, tt.projRef)
			if got != tt.expect {
				t.Errorf("got %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestNumberingConfigForType(t *testing.T) {
	tests := []struct {
		seqType     string
		expectGroup string
	}{
		{"po", "po"},
		{"tdc", "dc"},
		{"odc", "dc"},
		{"stdc", "dc"},
	}
	for _, tt := range tests {
		t.Run(tt.seqType, func(t *testing.T) {
			got := ConfigGroupForType(tt.seqType)
			if got != tt.expectGroup {
				t.Errorf("ConfigGroupForType(%q) = %q, want %q", tt.seqType, got, tt.expectGroup)
			}
		})
	}
}
