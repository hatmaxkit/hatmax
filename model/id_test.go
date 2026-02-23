package model

import (
	"testing"

	"github.com/google/uuid"
)

func TestGenerateID(t *testing.T) {
	var id string
	GenerateID(&id)

	if id == "" {
		t.Error("expected ID to be generated")
	}

	_, err := uuid.Parse(id)
	if err != nil {
		t.Errorf("expected valid UUID, got error: %v", err)
	}
}

func TestNewID(t *testing.T) {
	id := NewID()

	if id == "" {
		t.Error("expected ID to be generated")
	}

	_, err := uuid.Parse(id)
	if err != nil {
		t.Errorf("expected valid UUID, got error: %v", err)
	}
}

func TestGenerateIDUniqueness(t *testing.T) {
	var id1, id2 string
	GenerateID(&id1)
	GenerateID(&id2)

	if id1 == id2 {
		t.Error("expected unique IDs")
	}
}

func TestNewIDUniqueness(t *testing.T) {
	id1 := NewID()
	id2 := NewID()

	if id1 == id2 {
		t.Error("expected unique IDs")
	}
}

func TestParseID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid UUID",
			input:   "550e8400-e29b-41d4-a716-446655440000",
			wantErr: false,
		},
		{
			name:    "valid UUID uppercase",
			input:   "550E8400-E29B-41D4-A716-446655440000",
			wantErr: false,
		},
		{
			name:    "invalid UUID",
			input:   "not-a-uuid",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "partial UUID",
			input:   "550e8400-e29b-41d4",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseID() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr && got.String() == "" {
				t.Error("expected valid UUID")
			}
		})
	}
}

func TestNullUUID(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	emptyStr := ""
	invalidUUID := "invalid"

	tests := []struct {
		name      string
		input     *string
		wantValid bool
	}{
		{
			name:      "valid UUID string",
			input:     &validUUID,
			wantValid: true,
		},
		{
			name:      "nil pointer",
			input:     nil,
			wantValid: false,
		},
		{
			name:      "empty string",
			input:     &emptyStr,
			wantValid: false,
		},
		{
			name:      "invalid UUID string",
			input:     &invalidUUID,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NullUUID(tt.input)
			if got.Valid != tt.wantValid {
				t.Errorf("NullUUID() Valid = %v, want %v", got.Valid, tt.wantValid)
			}

			if tt.wantValid && got.UUID.String() != validUUID {
				t.Errorf("NullUUID() UUID = %v, want %v", got.UUID.String(), validUUID)
			}
		})
	}
}

func TestFromNullUUID(t *testing.T) {
	validUUID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	tests := []struct {
		name    string
		input   uuid.NullUUID
		want    *string
		wantNil bool
	}{
		{
			name:    "valid NullUUID",
			input:   uuid.NullUUID{UUID: validUUID, Valid: true},
			wantNil: false,
		},
		{
			name:    "invalid NullUUID",
			input:   uuid.NullUUID{Valid: false},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromNullUUID(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Errorf("FromNullUUID() = %v, want nil", *got)
				}

				return
			}

			if got == nil {
				t.Error("FromNullUUID() = nil, want non-nil")

				return
			}

			if *got != validUUID.String() {
				t.Errorf("FromNullUUID() = %v, want %v", *got, validUUID.String())
			}
		})
	}
}
