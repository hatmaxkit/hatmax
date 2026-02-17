package format

import "testing"

func TestPrice(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		currency string
		want     string
	}{
		{"USD", 150000, "USD", "$150,000"},
		{"EUR symbol after", 150000, "EUR", "150,000 €"},
		{"GBP", 99.99, "GBP", "£100"},
		{"PLN symbol after", 1500, "PLN", "1,500 zł"},
		{"ARS", 50000, "ARS", "$50,000"},
		{"BRL", 1000, "BRL", "R$1,000"},
		{"MXN", 25000, "MXN", "$25,000"},
		{"unknown currency uses code", 100, "XYZ", "XYZ100"},
		{"zero", 0, "USD", "$0"},
		{"negative", -500, "USD", "$-500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Price(tt.amount, tt.currency)
			if got != tt.want {
				t.Errorf("Price(%v, %q) = %q, want %q", tt.amount, tt.currency, got, tt.want)
			}
		})
	}
}

func TestPriceWithDecimals(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		currency string
		want     string
	}{
		{"USD", 99.99, "USD", "$99.99"},
		{"EUR symbol after", 99.99, "EUR", "99.99 €"},
		{"whole number", 100.00, "USD", "$100"},
		{"unknown currency", 50.5, "XYZ", "XYZ50.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PriceWithDecimals(tt.amount, tt.currency)
			if got != tt.want {
				t.Errorf("PriceWithDecimals(%v, %q) = %q, want %q", tt.amount, tt.currency, got, tt.want)
			}
		})
	}
}

func TestPriceRange(t *testing.T) {
	tests := []struct {
		name     string
		min      float64
		max      float64
		currency string
		want     string
	}{
		{"USD range", 100, 500, "USD", "$100 - $500"},
		{"EUR range", 100, 500, "EUR", "100 € - 500 €"},
		{"same value", 100, 100, "USD", "$100 - $100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PriceRange(tt.min, tt.max, tt.currency)
			if got != tt.want {
				t.Errorf("PriceRange(%v, %v, %q) = %q, want %q", tt.min, tt.max, tt.currency, got, tt.want)
			}
		})
	}
}

func TestRegisterCurrency(t *testing.T) {
	RegisterCurrency("CLP", "$", SymbolBefore)
	got := Price(1000, "CLP")
	want := "$1,000"
	if got != want {
		t.Errorf("Price after RegisterCurrency = %q, want %q", got, want)
	}

	RegisterCurrency("JPY", "¥", SymbolBefore)
	got = Price(10000, "JPY")
	want = "¥10,000"
	if got != want {
		t.Errorf("Price for JPY = %q, want %q", got, want)
	}
}
