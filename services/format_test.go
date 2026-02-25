package services

import "testing"

func TestFormatINR_Values(t *testing.T) {
	tests := []struct {
		name   string
		input  float64
		expect string
	}{
		{"zero", 0, "₹0.00"},
		{"small integer", 5, "₹5.00"},
		{"with decimals", 42.50, "₹42.50"},
		{"hundreds", 999.99, "₹999.99"},
		{"thousands", 1234.56, "₹1,234.56"},
		{"ten thousands", 12345.00, "₹12,345.00"},
		{"lakhs", 123456.78, "₹1,23,456.78"},
		{"ten lakhs", 1234567.89, "₹12,34,567.89"},
		{"crores", 12345678.90, "₹1,23,45,678.90"},
		{"ten crores", 123456789.00, "₹12,34,56,789.00"},
		{"negative small", -100.00, "-₹100.00"},
		{"negative lakhs", -250000.50, "-₹2,50,000.50"},
		{"one rupee", 1, "₹1.00"},
		{"exact thousands boundary", 1000, "₹1,000.00"},
		{"exact lakh boundary", 100000, "₹1,00,000.00"},
		{"exact crore boundary", 10000000, "₹1,00,00,000.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatINR(tt.input)
			if got != tt.expect {
				t.Errorf("FormatINR(%v) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

func TestApplyIndianGrouping(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"single digit", "5", "5"},
		{"two digits", "42", "42"},
		{"three digits", "999", "999"},
		{"four digits", "1234", "1,234"},
		{"five digits", "12345", "12,345"},
		{"six digits", "123456", "1,23,456"},
		{"seven digits", "1234567", "12,34,567"},
		{"eight digits", "12345678", "1,23,45,678"},
		{"nine digits", "123456789", "12,34,56,789"},
		{"ten digits", "1234567890", "1,23,45,67,890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyIndianGrouping(tt.input)
			if got != tt.expect {
				t.Errorf("applyIndianGrouping(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}
