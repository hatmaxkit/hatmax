package ui

import (
	"html/template"
	"testing"
)

func TestFormatPrice(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		currency string
		want     string
	}{
		{
			name:     "USD format",
			amount:   150000,
			currency: "USD",
			want:     "$150,000",
		},
		{
			name:     "EUR format with symbol after",
			amount:   150000,
			currency: "EUR",
			want:     "150,000 €",
		},
		{
			name:     "PLN format with symbol after",
			amount:   500000,
			currency: "PLN",
			want:     "500,000 zł",
		},
		{
			name:     "unknown currency uses code as symbol",
			amount:   1000,
			currency: "XYZ",
			want:     "XYZ1,000",
		},
		{
			name:     "small amount",
			amount:   99.99,
			currency: "USD",
			want:     "$100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatPrice(tt.amount, tt.currency)
			if got != tt.want {
				t.Errorf("FormatPrice(%v, %q) = %q, want %q", tt.amount, tt.currency, got, tt.want)
			}
		})
	}
}

func TestPriceTag(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		currency string
		want     template.HTML
	}{
		{
			name:     "USD price tag",
			amount:   250000,
			currency: "USD",
			want:     `<span class="price">$250,000</span>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PriceTag(tt.amount, tt.currency)
			if got != tt.want {
				t.Errorf("PriceTag(%v, %q) = %q, want %q", tt.amount, tt.currency, got, tt.want)
			}
		})
	}
}

func TestPriceTagNegotiable(t *testing.T) {
	tests := []struct {
		name       string
		amount     float64
		currency   string
		negotiable bool
		want       template.HTML
	}{
		{
			name:       "negotiable price",
			amount:     300000,
			currency:   "USD",
			negotiable: true,
			want:       `<span class="price">$300,000 <small class="price__tag">Negotiable</small></span>`,
		},
		{
			name:       "non-negotiable price",
			amount:     300000,
			currency:   "USD",
			negotiable: false,
			want:       `<span class="price">$300,000</span>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PriceTagNegotiable(tt.amount, tt.currency, tt.negotiable)
			if got != tt.want {
				t.Errorf("PriceTagNegotiable(%v, %q, %v) = %q, want %q", tt.amount, tt.currency, tt.negotiable, got, tt.want)
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
		want     template.HTML
	}{
		{
			name:     "USD price range",
			min:      100000,
			max:      200000,
			currency: "USD",
			want:     `<span class="price-range">$100,000 - $200,000</span>`,
		},
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
	tests := []struct {
		name     string
		code     string
		symbol   string
		position CurrencyPosition
	}{
		{
			name:     "new currency with symbol before",
			code:     "TEST1",
			symbol:   "T$",
			position: SymbolBefore,
		},
		{
			name:     "new currency with symbol after",
			code:     "TEST2",
			symbol:   "TK",
			position: SymbolAfter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterCurrency(tt.code, tt.symbol, tt.position)

			if CurrencySymbol[tt.code] != tt.symbol {
				t.Errorf("CurrencySymbol[%q] = %q, want %q", tt.code, CurrencySymbol[tt.code], tt.symbol)
			}

			format, ok := CurrencyFormats[tt.code]
			if !ok {
				t.Fatalf("CurrencyFormats[%q] not found", tt.code)
			}
			if format.Symbol != tt.symbol {
				t.Errorf("format.Symbol = %q, want %q", format.Symbol, tt.symbol)
			}
			if format.Position != tt.position {
				t.Errorf("format.Position = %v, want %v", format.Position, tt.position)
			}
		})
	}
}

func TestRegisterCurrencyUsedInFormatPrice(t *testing.T) {
	RegisterCurrency("JPY", "¥", SymbolBefore)

	got := FormatPrice(1000, "JPY")
	want := "¥1,000"
	if got != want {
		t.Errorf("FormatPrice with registered currency = %q, want %q", got, want)
	}

	RegisterCurrency("CHF", "Fr.", SymbolAfter)

	got = FormatPrice(500, "CHF")
	want = "500 Fr."
	if got != want {
		t.Errorf("FormatPrice with symbol after = %q, want %q", got, want)
	}
}
