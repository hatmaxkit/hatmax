package settings

import "testing"

func TestNamespaceSchemaDisplayLabel(t *testing.T) {
	tests := []struct {
		name      string
		namespace NamespaceSchema
		want      string
	}{
		{
			name:      "returns label when set",
			namespace: NamespaceSchema{Key: "security", Label: "Security"},
			want:      "Security",
		},
		{
			name:      "falls back to key",
			namespace: NamespaceSchema{Key: "security"},
			want:      "security",
		},
		{
			name:      "empty label falls back to key",
			namespace: NamespaceSchema{Key: "security", Label: ""},
			want:      "security",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.namespace.DisplayLabel(); got != tt.want {
				t.Errorf("NamespaceSchema.DisplayLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}
