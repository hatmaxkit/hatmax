package seed

import (
	"testing"
)

func TestNewRefMap(t *testing.T) {
	m := NewRefMap()
	if m == nil {
		t.Fatal("expected non-nil RefMap")
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %d entries", len(m))
	}
}

func TestRefMapRegister(t *testing.T) {
	tests := []struct {
		name string
		ref  Ref
		uuid string
	}{
		{
			name: "simple reference",
			ref:  "user:admin",
			uuid: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name: "empty uuid",
			ref:  "user:empty",
			uuid: "",
		},
		{
			name: "complex reference",
			ref:  "org:acme/user:john",
			uuid: "123e4567-e89b-12d3-a456-426614174000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewRefMap()
			m.Register(tt.ref, tt.uuid)

			if got, ok := m[tt.ref]; !ok {
				t.Error("expected reference to be registered")
			} else if got != tt.uuid {
				t.Errorf("got uuid %q, want %q", got, tt.uuid)
			}
		})
	}
}

func TestRefMapRegisterOverwrite(t *testing.T) {
	m := NewRefMap()
	ref := Ref("user:admin")

	m.Register(ref, "uuid-1")
	m.Register(ref, "uuid-2")

	if got := m[ref]; got != "uuid-2" {
		t.Errorf("expected overwritten value uuid-2, got %q", got)
	}
}

func TestRefMapResolve(t *testing.T) {
	m := NewRefMap()
	m.Register("user:admin", "uuid-admin")
	m.Register("user:guest", "uuid-guest")

	tests := []struct {
		name    string
		ref     Ref
		want    string
		wantErr bool
	}{
		{
			name:    "existing reference",
			ref:     "user:admin",
			want:    "uuid-admin",
			wantErr: false,
		},
		{
			name:    "another existing reference",
			ref:     "user:guest",
			want:    "uuid-guest",
			wantErr: false,
		},
		{
			name:    "non-existing reference",
			ref:     "user:unknown",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty reference",
			ref:     "",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := m.Resolve(tt.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRefMapMustResolve(t *testing.T) {
	m := NewRefMap()
	m.Register("user:admin", "uuid-admin")

	tests := []struct {
		name      string
		ref       Ref
		want      string
		wantPanic bool
	}{
		{
			name:      "existing reference",
			ref:       "user:admin",
			want:      "uuid-admin",
			wantPanic: false,
		},
		{
			name:      "non-existing reference",
			ref:       "user:unknown",
			want:      "",
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("expected panic but did not get one")
					}
				}()
			}

			got := m.MustResolve(tt.ref)

			if tt.wantPanic {
				t.Error("should have panicked")
			}

			if got != tt.want {
				t.Errorf("MustResolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRefMapHas(t *testing.T) {
	m := NewRefMap()
	m.Register("user:admin", "uuid-admin")

	tests := []struct {
		name string
		ref  Ref
		want bool
	}{
		{
			name: "existing reference",
			ref:  "user:admin",
			want: true,
		},
		{
			name: "non-existing reference",
			ref:  "user:unknown",
			want: false,
		},
		{
			name: "empty reference",
			ref:  "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.Has(tt.ref); got != tt.want {
				t.Errorf("Has() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRefType(t *testing.T) {
	var r Ref = "test-ref"
	if string(r) != "test-ref" {
		t.Errorf("Ref conversion failed, got %q", string(r))
	}
}
