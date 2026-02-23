package settings

import (
	"sort"
	"testing"
)

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()

	schema := Schema{Key: "site.name", Type: String, Default: "Test"}
	r.Register(schema)

	got, ok := r.Get("site.name")
	if !ok {
		t.Fatal("expected schema to be found")
	}

	if got.Key != "site.name" {
		t.Errorf("got key %q, want %q", got.Key, "site.name")
	}

	if got.Default != "Test" {
		t.Errorf("got default %q, want %q", got.Default, "Test")
	}
}

func TestRegistryGetNotFound(t *testing.T) {
	r := NewRegistry()

	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("expected schema not to be found")
	}
}

func TestRegistryAll(t *testing.T) {
	r := NewRegistry()
	r.Register(Schema{Key: "a.one", Type: String})
	r.Register(Schema{Key: "b.two", Type: Int})
	r.Register(Schema{Key: "a.three", Type: Bool})

	all := r.All()
	if len(all) != 3 {
		t.Errorf("got %d schemas, want 3", len(all))
	}
}

func TestRegistryByPrefix(t *testing.T) {
	r := NewRegistry()
	r.Register(Schema{Key: "site.name", Type: String})
	r.Register(Schema{Key: "site.logo", Type: String})
	r.Register(Schema{Key: "feature.dark", Type: Bool})
	r.Register(Schema{Key: "site.max", Type: Int})

	site := r.ByPrefix("site.")
	if len(site) != 3 {
		t.Errorf("got %d schemas for site., want 3", len(site))
	}

	keys := make([]string, len(site))
	for i, s := range site {
		keys[i] = s.Key
	}

	sort.Strings(keys)

	want := []string{"site.logo", "site.max", "site.name"}
	for i, k := range keys {
		if k != want[i] {
			t.Errorf("got key %q at index %d, want %q", k, i, want[i])
		}
	}
}

func TestRegistryByPrefixEmpty(t *testing.T) {
	r := NewRegistry()
	r.Register(Schema{Key: "site.name", Type: String})

	result := r.ByPrefix("other.")
	if len(result) != 0 {
		t.Errorf("got %d schemas, want 0", len(result))
	}
}

func TestRegistryOverwrite(t *testing.T) {
	r := NewRegistry()
	r.Register(Schema{Key: "test.key", Default: "first"})
	r.Register(Schema{Key: "test.key", Default: "second"})

	got, _ := r.Get("test.key")
	if got.Default != "second" {
		t.Errorf("got default %q, want %q", got.Default, "second")
	}

	if len(r.All()) != 1 {
		t.Errorf("got %d schemas, want 1", len(r.All()))
	}
}
