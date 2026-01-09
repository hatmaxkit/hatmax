package web

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestParseForm(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "valid form",
			body:    "name=test&active=true",
			wantErr: false,
		},
		{
			name:    "empty form",
			body:    "",
			wantErr: false,
		},
		{
			name:    "invalid encoding",
			body:    "%",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			form, err := ParseForm(r)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseForm() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && form == nil {
				t.Errorf("ParseForm() returned nil form")
			}
		})
	}
}

func TestFormValuesString(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("name=john&email="))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	form, _ := ParseForm(r)

	tests := []struct {
		name  string
		field string
		want  string
	}{
		{
			name:  "existing field",
			field: "name",
			want:  "john",
		},
		{
			name:  "empty field",
			field: "email",
			want:  "",
		},
		{
			name:  "missing field",
			field: "missing",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := form.String(tt.field)
			if got != tt.want {
				t.Errorf("FormValues.String(%s) = %v, want %v", tt.field, got, tt.want)
			}
		})
	}
}

func TestFormValuesStringOr(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("name=john&empty="))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	form, _ := ParseForm(r)

	tests := []struct {
		name   string
		field  string
		defVal string
		want   string
	}{
		{
			name:   "existing field",
			field:  "name",
			defVal: "default",
			want:   "john",
		},
		{
			name:   "empty field with default",
			field:  "empty",
			defVal: "default",
			want:   "default",
		},
		{
			name:   "missing field with default",
			field:  "missing",
			defVal: "fallback",
			want:   "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := form.StringOr(tt.field, tt.defVal)
			if got != tt.want {
				t.Errorf("FormValues.StringOr(%s, %s) = %v, want %v", tt.field, tt.defVal, got, tt.want)
			}
		})
	}
}

func TestFormValuesBool(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("active=true&enabled=on&disabled=false&missing="))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	form, _ := ParseForm(r)

	tests := []struct {
		name  string
		field string
		want  bool
	}{
		{
			name:  "true value",
			field: "active",
			want:  true,
		},
		{
			name:  "on value",
			field: "enabled",
			want:  true,
		},
		{
			name:  "false value",
			field: "disabled",
			want:  false,
		},
		{
			name:  "missing value",
			field: "missing",
			want:  false,
		},
		{
			name:  "nonexistent field",
			field: "nothere",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := form.Bool(tt.field)
			if got != tt.want {
				t.Errorf("FormValues.Bool(%s) = %v, want %v", tt.field, got, tt.want)
			}
		})
	}
}

func TestFormValuesUUID(t *testing.T) {
	validID := uuid.New().String()
	r := httptest.NewRequest("POST", "/", strings.NewReader("id="+validID+"&invalid=not-uuid&empty="))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	form, _ := ParseForm(r)

	tests := []struct {
		name    string
		field   string
		wantErr bool
	}{
		{
			name:    "valid uuid",
			field:   "id",
			wantErr: false,
		},
		{
			name:    "invalid uuid",
			field:   "invalid",
			wantErr: true,
		},
		{
			name:    "empty uuid",
			field:   "empty",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := form.UUID(tt.field)

			if (err != nil) != tt.wantErr {
				t.Errorf("FormValues.UUID(%s) error = %v, wantErr %v", tt.field, err, tt.wantErr)
			}

			if !tt.wantErr && got == uuid.Nil {
				t.Errorf("FormValues.UUID(%s) returned nil UUID", tt.field)
			}
		})
	}
}

func TestParseIDParam(t *testing.T) {
	validID := uuid.New()
	invalidID := "not-a-uuid"

	tests := []struct {
		name    string
		param   string
		value   string
		wantErr bool
		wantID  uuid.UUID
	}{
		{
			name:    "valid uuid param",
			param:   "id",
			value:   validID.String(),
			wantErr: false,
			wantID:  validID,
		},
		{
			name:    "invalid uuid param",
			param:   "id",
			value:   invalidID,
			wantErr: true,
		},
		{
			name:    "empty param",
			param:   "id",
			value:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add(tt.param, tt.value)
			ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
			r = r.WithContext(ctx)

			got, err := ParseIDParam(r, tt.param)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIDParam() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && got != tt.wantID {
				t.Errorf("ParseIDParam() = %v, want %v", got, tt.wantID)
			}
		})
	}
}
