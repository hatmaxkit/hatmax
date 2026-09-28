package project

import (
	"context"
	"strings"
	"testing"
)

func TestInspectDocumentationRecordsCanonicalStructureOnly(t *testing.T) {
	root := copyFixture(t, "supported")
	appendProjectFile(t, root, "Makefile", "\ndocs-check:\n\t@echo docs\n")
	writeProjectFile(t, root, "docs/index.md", managedDocumentation("[Reference](reference/index.md)"))
	writeProjectFile(t, root, "docs/reference/index.md", managedDocumentation("[Property](property/)\n[Missing](missing/)"))
	writeProjectFile(t, root, "docs/reference/property/index.md", managedDocumentation("Property contracts."))
	writeProjectFile(t, root, "docs/how-to/index.md", "# User-owned how-to index\n")
	writeProjectFile(t, root, "docs/tutorials/property/index.md", "<!-- hatmax:generated:start -->\nIncomplete\n")
	writeProjectFile(t, root, "docs/notes.md", "# Unstructured notes\n")

	inventory, err := Inspect(context.Background(), root)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if inventory.Documentation.Root != "docs" || len(inventory.Documentation.Files) != 5 {
		t.Fatalf("Documentation = %#v, want five canonical files", inventory.Documentation)
	}

	property, exists := inventory.DocumentationFile("docs/reference/property/index.md")
	if !exists || property.Quadrant != "reference" || property.Subject != "property" || property.Index || property.ManagedState != DocumentationManaged || property.Digest == "" {
		t.Errorf("property documentation = %#v, %v", property, exists)
	}

	reference, exists := inventory.DocumentationFile("docs/reference/index.md")
	if !exists || len(reference.LocalLinks) != 2 || !reference.LocalLinks[0].Exists || reference.LocalLinks[1].Exists {
		t.Errorf("reference links = %#v, want existing property and missing target", reference.LocalLinks)
	}

	howTo, exists := inventory.DocumentationFile("docs/how-to/index.md")
	if !exists || howTo.ManagedState != DocumentationUnmanaged {
		t.Errorf("how-to index = %#v, want unmanaged", howTo)
	}

	tutorial, exists := inventory.DocumentationFile("docs/tutorials/property/index.md")
	if !exists || tutorial.ManagedState != DocumentationMarkersInvalid {
		t.Errorf("tutorial target = %#v, want invalid markers", tutorial)
	}

	if len(inventory.Documentation.OtherPaths) != 1 || inventory.Documentation.OtherPaths[0] != "docs/notes.md" {
		t.Errorf("OtherPaths = %v, want docs/notes.md", inventory.Documentation.OtherPaths)
	}

	if len(inventory.Documentation.ValidationCommands) != 1 || inventory.Documentation.ValidationCommands[0].Name != "docs-check" {
		t.Errorf("ValidationCommands = %#v, want docs-check", inventory.Documentation.ValidationCommands)
	}

	reference.LocalLinks[0].Target = "changed"

	stored, _ := inventory.DocumentationFile("docs/reference/index.md")
	if stored.LocalLinks[0].Target == "changed" {
		t.Error("DocumentationFile() exposed inventory storage")
	}
}

func TestInspectDocumentationBoundsCanonicalMarkdown(t *testing.T) {
	root := copyFixture(t, "supported")
	writeProjectFile(t, root, "docs/reference/property/index.md", strings.Repeat("x", 128))

	inventory, err := InspectWithOptions(context.Background(), root, Options{MaximumFileSize: 64})
	if err != nil {
		t.Fatalf("InspectWithOptions() error = %v", err)
	}

	property, exists := inventory.DocumentationFile("docs/reference/property/index.md")
	if !exists || property.ManagedState != DocumentationUnreadable || property.Digest != "" || len(property.LocalLinks) != 0 {
		t.Errorf("bounded property documentation = %#v, %v", property, exists)
	}
}

func managedDocumentation(content string) string {
	return "# Wrapper\n\n<!-- hatmax:generated:start -->\n" + content + "\n<!-- hatmax:generated:end -->\n"
}
