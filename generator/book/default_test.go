package book

import "testing"

func TestLoadDefault(t *testing.T) {
	loaded, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault() error = %v", err)
	}

	supported, err := loaded.SupportsHatmax("v0.4.0")
	if err != nil {
		t.Fatalf("SupportsHatmax() error = %v", err)
	}

	if !supported {
		t.Fatal("SupportsHatmax() = false, want true")
	}

	selection, err := loaded.Select(
		"server_rendered_crud",
		[]string{"postgres_persistence", "htmx_form", "runtime_validation"},
	)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}

	if len(selection.Capabilities) != 3 {
		t.Fatalf("len(Selection.Capabilities) = %d, want 3", len(selection.Capabilities))
	}

	if len(selection.Rules) != 12 {
		t.Fatalf("len(Selection.Rules) = %d, want 12", len(selection.Rules))
	}
}

func TestLoadDefaultIncludesBoxedDocumentationContract(t *testing.T) {
	loaded, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault() error = %v", err)
	}

	archetype, exists := loaded.Archetype("server_rendered_crud")
	if !exists {
		t.Fatal("server_rendered_crud archetype is missing")
	}

	operationFound := false
	for _, operation := range archetype.Operations {
		if operation.ID != "document_feature" {
			continue
		}

		operationFound = len(operation.Obligations) == 2
	}

	if !operationFound {
		t.Errorf("document_feature operation = %#v, want two boxed documentation obligations", archetype.Operations)
	}

	for _, rule := range []string{
		"hatmax.documentation.explicit_intent",
		"hatmax.documentation.diataxis",
		"hatmax.documentation.managed_sections",
		"hatmax.documentation.index_reachability",
	} {
		if _, exists := loaded.Rule(rule); !exists {
			t.Errorf("Book rule %q is missing", rule)
		}
	}
}
