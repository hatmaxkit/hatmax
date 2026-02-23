package image

import "testing"

func TestValidVariantTypes(t *testing.T) {
	types := ValidVariantTypes()

	if len(types) != 4 {
		t.Errorf("expected 4 variant types, got %d", len(types))
	}

	expected := []VariantType{VariantOriginal, VariantLarge, VariantMedium, VariantThumbnail}
	for i, vt := range expected {
		if types[i] != vt {
			t.Errorf("types[%d] = %q, want %q", i, types[i], vt)
		}
	}
}

func TestDefaultVariantSpecs(t *testing.T) {
	specs := DefaultVariantSpecs()

	if len(specs) != 3 {
		t.Errorf("expected 3 specs, got %d", len(specs))
	}

	tests := []struct {
		idx       int
		wantType  VariantType
		wantWidth int
	}{
		{0, VariantLarge, 1200},
		{1, VariantMedium, 800},
		{2, VariantThumbnail, 300},
	}

	for _, tt := range tests {
		if specs[tt.idx].Type != tt.wantType {
			t.Errorf("specs[%d].Type = %q, want %q", tt.idx, specs[tt.idx].Type, tt.wantType)
		}

		if specs[tt.idx].MaxWidth != tt.wantWidth {
			t.Errorf("specs[%d].MaxWidth = %d, want %d", tt.idx, specs[tt.idx].MaxWidth, tt.wantWidth)
		}
	}
}

func TestVariantTypeConstants(t *testing.T) {
	tests := []struct {
		vt   VariantType
		want string
	}{
		{VariantOriginal, "original"},
		{VariantLarge, "large"},
		{VariantMedium, "medium"},
		{VariantThumbnail, "thumbnail"},
	}

	for _, tt := range tests {
		if string(tt.vt) != tt.want {
			t.Errorf("VariantType = %q, want %q", tt.vt, tt.want)
		}
	}
}
