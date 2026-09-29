package book

import "testing"

func TestLoadReleaseTwoAddsApplicationArchetype(t *testing.T) {
	selected, err := LoadRelease(2)
	if err != nil {
		t.Fatalf("LoadRelease(2) error = %v", err)
	}

	if selected.Manifest().BookVersion != 2 {
		t.Fatalf("BookVersion = %d, want 2", selected.Manifest().BookVersion)
	}

	application, exists := selected.Archetype("server_rendered_hatmax_application")
	if !exists || len(application.Operations) != 1 || application.Operations[0].ID != "create_application" {
		t.Fatalf("application archetype = %#v, want create_application", application)
	}

	mainEffect := false

	for _, obligation := range application.Obligations {
		for _, effect := range obligation.Files {
			if effect.Path == "main.go" && effect.Mode == FileEffectCreate {
				mainEffect = true
			}
		}
	}

	if !mainEffect {
		t.Error("application archetype does not own the main.go create effect")
	}

	crud, exists := selected.Archetype("server_rendered_crud")
	if !exists {
		t.Fatal("Book release 2 does not contain the compatible CRUD archetype")
	}

	operation, exists := findBookOperation(crud, "create_feature")
	if !exists || !containsBookString(operation.RequiredCapabilities, "htmx_form") || !containsBookString(operation.RequiredCapabilities, "runtime_validation") {
		t.Errorf("create_feature capabilities = %v, want canonical HTMX and validation", operation.RequiredCapabilities)
	}
}

func TestLoadReleaseKeepsDefaultOnBookOne(t *testing.T) {
	selected, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault() error = %v", err)
	}

	if selected.Manifest().BookVersion != 1 {
		t.Errorf("default BookVersion = %d, want delivered release 1", selected.Manifest().BookVersion)
	}
}

func findBookOperation(archetype Archetype, id string) (Operation, bool) {
	for _, operation := range archetype.Operations {
		if operation.ID == id {
			return operation, true
		}
	}

	return Operation{}, false
}

func containsBookString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}

	return false
}
