package hatmax

import "testing"

func TestHandlerInferRepoName(t *testing.T) {
	tests := []struct {
		name     string
		handler  Handler
		expected string
	}{
		{
			name: "basic repo name inference",
			handler: Handler{
				Model: "Item",
			},
			expected: "ItemRepo",
		},
		{
			name: "repo name with override",
			handler: Handler{
				Model: "Item",
				Overrides: &HandlerOverrides{
					RepoName: "CustomItemRepo",
				},
			},
			expected: "CustomItemRepo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.handler.InferRepoName(); got != tt.expected {
				t.Errorf("InferRepoName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHandlerInferMethodName(t *testing.T) {
	tests := []struct {
		name     string
		handler  Handler
		expected string
	}{
		{
			name: "list operation",
			handler: Handler{
				Operation: OpList,
			},
			expected: "List",
		},
		{
			name: "create operation",
			handler: Handler{
				Operation: OpCreate,
			},
			expected: "Create",
		},
		{
			name: "custom operation",
			handler: Handler{
				Operation:       OpCustom,
				CustomOperation: "toggle",
			},
			expected: "Toggle",
		},
		{
			name: "method name with override",
			handler: Handler{
				Operation: OpCreate,
				Overrides: &HandlerOverrides{
					MethodName: "CreateNew",
				},
			},
			expected: "CreateNew",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.handler.InferMethodName(); got != tt.expected {
				t.Errorf("InferMethodName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHandlerInferRepoCall(t *testing.T) {
	tests := []struct {
		name     string
		handler  Handler
		expected string
	}{
		{
			name: "basic repo call",
			handler: Handler{
				Model:     "Item",
				Operation: OpCreate,
			},
			expected: "ItemRepo.Create",
		},
		{
			name: "custom operation repo call",
			handler: Handler{
				Model:           "Item",
				Operation:       OpCustom,
				CustomOperation: "toggle",
			},
			expected: "ItemRepo.Toggle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.handler.InferRepoCall(); got != tt.expected {
				t.Errorf("InferRepoCall() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCapitalizeFirst(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"toggle", "Toggle"},
		{"search", "Search"},
		{"bulk_update", "Bulk_update"},
		{"", ""},
		{"a", "A"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := capitalizeFirst(tt.input); got != tt.expected {
				t.Errorf("capitalizeFirst(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
