package settings

import (
	"context"
	"errors"
	"testing"
)

type memStore struct {
	data map[string]string
	err  error
}

func newMemStore() *memStore {
	return &memStore{data: make(map[string]string)}
}

func (m *memStore) Get(ctx context.Context, key string) (string, error) {
	if m.err != nil {
		return "", m.err
	}

	return m.data[key], nil
}

func (m *memStore) Set(ctx context.Context, key, value string) error {
	if m.err != nil {
		return m.err
	}

	m.data[key] = value

	return nil
}

func (m *memStore) All(ctx context.Context) ([]Value, error) {
	if m.err != nil {
		return nil, m.err
	}

	var out []Value
	for k, v := range m.data {
		out = append(out, Value{Key: k, Raw: v})
	}

	return out, nil
}

func (m *memStore) Delete(ctx context.Context, key string) error {
	if m.err != nil {
		return m.err
	}

	delete(m.data, key)

	return nil
}

func TestServiceGetString(t *testing.T) {
	reg := NewRegistry()
	reg.Register(Schema{Key: "site.name", Type: String, Default: "Default Site"})

	store := newMemStore()
	svc := NewService(reg, store)
	ctx := context.Background()

	t.Run("returns default when not set", func(t *testing.T) {
		got, err := svc.GetString(ctx, "site.name")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != "Default Site" {
			t.Errorf("got %q, want %q", got, "Default Site")
		}
	})

	t.Run("returns stored value", func(t *testing.T) {
		store.data["site.name"] = "Custom Site"

		got, err := svc.GetString(ctx, "site.name")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != "Custom Site" {
			t.Errorf("got %q, want %q", got, "Custom Site")
		}
	})

	t.Run("returns empty for unknown key", func(t *testing.T) {
		got, err := svc.GetString(ctx, "unknown.key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != "" {
			t.Errorf("got %q, want empty string", got)
		}
	})
}

func TestServiceGetInt(t *testing.T) {
	reg := NewRegistry()
	reg.Register(Schema{Key: "site.limit", Type: Int, Default: "50"})

	store := newMemStore()
	svc := NewService(reg, store)
	ctx := context.Background()

	t.Run("returns default when not set", func(t *testing.T) {
		got, err := svc.GetInt(ctx, "site.limit")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != 50 {
			t.Errorf("got %d, want %d", got, 50)
		}
	})

	t.Run("returns stored value", func(t *testing.T) {
		store.data["site.limit"] = "100"

		got, err := svc.GetInt(ctx, "site.limit")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != 100 {
			t.Errorf("got %d, want %d", got, 100)
		}
	})

	t.Run("returns zero for unknown key", func(t *testing.T) {
		got, err := svc.GetInt(ctx, "unknown.key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != 0 {
			t.Errorf("got %d, want 0", got)
		}
	})

	t.Run("returns error for invalid stored value", func(t *testing.T) {
		store.data["bad.int"] = "notanint"

		_, err := svc.GetInt(ctx, "bad.int")
		if err == nil {
			t.Error("expected error for invalid int")
		}
	})
}

func TestServiceGetBool(t *testing.T) {
	reg := NewRegistry()
	reg.Register(Schema{Key: "feature.enabled", Type: Bool, Default: "true"})

	store := newMemStore()
	svc := NewService(reg, store)
	ctx := context.Background()

	t.Run("returns default when not set", func(t *testing.T) {
		got, err := svc.GetBool(ctx, "feature.enabled")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !got {
			t.Error("got false, want true")
		}
	})

	t.Run("returns stored value", func(t *testing.T) {
		store.data["feature.enabled"] = "false"

		got, err := svc.GetBool(ctx, "feature.enabled")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got {
			t.Error("got true, want false")
		}
	})

	t.Run("returns false for unknown key", func(t *testing.T) {
		got, err := svc.GetBool(ctx, "unknown.key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got {
			t.Error("got true, want false")
		}
	})

	t.Run("returns error for invalid stored value", func(t *testing.T) {
		store.data["bad.bool"] = "notabool"

		_, err := svc.GetBool(ctx, "bad.bool")
		if err == nil {
			t.Error("expected error for invalid bool")
		}
	})
}

func TestServiceSet(t *testing.T) {
	reg := NewRegistry()
	reg.Register(Schema{Key: "site.limit", Type: Int, Min: ptr(1), Max: ptr(100)})

	store := newMemStore()
	svc := NewService(reg, store)
	ctx := context.Background()

	t.Run("validates and stores", func(t *testing.T) {
		err := svc.Set(ctx, "site.limit", "50")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if store.data["site.limit"] != "50" {
			t.Errorf("got %q, want %q", store.data["site.limit"], "50")
		}
	})

	t.Run("rejects invalid value", func(t *testing.T) {
		err := svc.Set(ctx, "site.limit", "200")
		if err == nil {
			t.Error("expected validation error")
		}
	})

	t.Run("allows unregistered keys", func(t *testing.T) {
		err := svc.Set(ctx, "custom.key", "anyvalue")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if store.data["custom.key"] != "anyvalue" {
			t.Errorf("got %q, want %q", store.data["custom.key"], "anyvalue")
		}
	})
}

func TestServiceDelete(t *testing.T) {
	store := newMemStore()
	store.data["to.delete"] = "value"

	svc := NewService(NewRegistry(), store)
	ctx := context.Background()

	err := svc.Delete(ctx, "to.delete")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := store.data["to.delete"]; ok {
		t.Error("expected key to be deleted")
	}
}

func TestServiceAll(t *testing.T) {
	store := newMemStore()
	store.data["a"] = "1"
	store.data["b"] = "2"

	svc := NewService(NewRegistry(), store)
	ctx := context.Background()

	all, err := svc.All(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(all) != 2 {
		t.Errorf("got %d values, want 2", len(all))
	}
}

func TestServiceStoreError(t *testing.T) {
	store := newMemStore()
	store.err = errors.New("store error")

	reg := NewRegistry()
	reg.Register(Schema{Key: "test.key", Default: "fallback"})

	svc := NewService(reg, store)
	ctx := context.Background()

	t.Run("GetString returns default on error", func(t *testing.T) {
		got, err := svc.GetString(ctx, "test.key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != "fallback" {
			t.Errorf("got %q, want %q", got, "fallback")
		}
	})

	t.Run("Set returns store error", func(t *testing.T) {
		err := svc.Set(ctx, "test.key", "value")
		if err == nil {
			t.Error("expected store error")
		}
	})
}
