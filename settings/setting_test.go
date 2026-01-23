package settings

import "testing"

func ptr(i int) *int { return &i }

func TestSchemaValidate(t *testing.T) {
	tests := []struct {
		name    string
		schema  Schema
		value   string
		wantErr bool
	}{
		{
			name:    "required empty",
			schema:  Schema{Key: "test.key", Required: true},
			value:   "",
			wantErr: true,
		},
		{
			name:    "required with value",
			schema:  Schema{Key: "test.key", Required: true},
			value:   "hello",
			wantErr: false,
		},
		{
			name:    "optional empty",
			schema:  Schema{Key: "test.key"},
			value:   "",
			wantErr: false,
		},
		{
			name:    "bool valid true",
			schema:  Schema{Key: "test.flag", Type: Bool},
			value:   "true",
			wantErr: false,
		},
		{
			name:    "bool valid false",
			schema:  Schema{Key: "test.flag", Type: Bool},
			value:   "false",
			wantErr: false,
		},
		{
			name:    "bool valid 1",
			schema:  Schema{Key: "test.flag", Type: Bool},
			value:   "1",
			wantErr: false,
		},
		{
			name:    "bool invalid",
			schema:  Schema{Key: "test.flag", Type: Bool},
			value:   "notabool",
			wantErr: true,
		},
		{
			name:    "int valid",
			schema:  Schema{Key: "test.count", Type: Int},
			value:   "42",
			wantErr: false,
		},
		{
			name:    "int valid negative",
			schema:  Schema{Key: "test.count", Type: Int},
			value:   "-10",
			wantErr: false,
		},
		{
			name:    "int invalid",
			schema:  Schema{Key: "test.count", Type: Int},
			value:   "abc",
			wantErr: true,
		},
		{
			name:    "int below min",
			schema:  Schema{Key: "test.count", Type: Int, Min: ptr(10)},
			value:   "5",
			wantErr: true,
		},
		{
			name:    "int at min",
			schema:  Schema{Key: "test.count", Type: Int, Min: ptr(10)},
			value:   "10",
			wantErr: false,
		},
		{
			name:    "int above max",
			schema:  Schema{Key: "test.count", Type: Int, Max: ptr(100)},
			value:   "150",
			wantErr: true,
		},
		{
			name:    "int at max",
			schema:  Schema{Key: "test.count", Type: Int, Max: ptr(100)},
			value:   "100",
			wantErr: false,
		},
		{
			name:    "int within range",
			schema:  Schema{Key: "test.count", Type: Int, Min: ptr(10), Max: ptr(100)},
			value:   "50",
			wantErr: false,
		},
		{
			name:    "enum valid",
			schema:  Schema{Key: "test.color", Type: Enum, Options: []string{"red", "green", "blue"}},
			value:   "green",
			wantErr: false,
		},
		{
			name:    "enum invalid",
			schema:  Schema{Key: "test.color", Type: Enum, Options: []string{"red", "green", "blue"}},
			value:   "yellow",
			wantErr: true,
		},
		{
			name:    "enum empty options",
			schema:  Schema{Key: "test.color", Type: Enum, Options: []string{}},
			value:   "any",
			wantErr: true,
		},
		{
			name:    "string type passes",
			schema:  Schema{Key: "test.name", Type: String},
			value:   "anything",
			wantErr: false,
		},
		{
			name:    "empty skips validation",
			schema:  Schema{Key: "test.opt", Type: Int, Min: ptr(10)},
			value:   "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Schema.Validate(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}
