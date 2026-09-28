package book

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"gopkg.in/yaml.v3"
)

const manifestPath = "manifest.yaml"

// Load reads and validates a Hatmax Book from an fs.FS.
func Load(source fs.FS) (*Book, error) {
	if source == nil {
		return nil, fmt.Errorf("book source is nil")
	}

	manifestData, err := fs.ReadFile(source, manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", manifestPath, err)
	}

	var manifest Manifest

	err = decodeYAML(manifestPath, manifestData, &manifest)
	if err != nil {
		return nil, err
	}

	result := &Book{
		source:          source,
		manifest:        manifest,
		capabilities:    make(map[string]Capability),
		archetypes:      make(map[string]Archetype),
		rules:           make(map[string]Rule),
		capabilityOrder: make([]string, 0, len(manifest.Capabilities)),
		archetypeOrder:  make([]string, 0, len(manifest.Archetypes)),
		ruleOrder:       make([]string, 0, len(manifest.Rules)),
	}

	err = result.validateManifest()
	if err != nil {
		return nil, err
	}

	err = result.loadCapabilities()
	if err != nil {
		return nil, err
	}

	err = result.loadArchetypes()
	if err != nil {
		return nil, err
	}

	err = result.loadRules()
	if err != nil {
		return nil, err
	}

	err = result.validate()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (b *Book) loadCapabilities() error {
	for _, entryPath := range b.manifest.Capabilities {
		var value Capability

		err := b.loadEntry("capabilities", entryPath, &value)
		if err != nil {
			return err
		}

		if _, exists := b.capabilities[value.ID]; exists {
			return validationError("book_duplicate_id", entryPath, "duplicate capability ID %q", value.ID)
		}

		b.capabilities[value.ID] = value
		b.capabilityOrder = append(b.capabilityOrder, value.ID)
	}

	return nil
}

func (b *Book) loadArchetypes() error {
	for _, entryPath := range b.manifest.Archetypes {
		var value Archetype

		err := b.loadEntry("archetypes", entryPath, &value)
		if err != nil {
			return err
		}

		if _, exists := b.archetypes[value.ID]; exists {
			return validationError("book_duplicate_id", entryPath, "duplicate archetype ID %q", value.ID)
		}

		b.archetypes[value.ID] = value
		b.archetypeOrder = append(b.archetypeOrder, value.ID)
	}

	return nil
}

func (b *Book) loadRules() error {
	for _, entryPath := range b.manifest.Rules {
		var value Rule

		err := b.loadEntry("rules", entryPath, &value)
		if err != nil {
			return err
		}

		if _, exists := b.rules[value.ID]; exists {
			return validationError("book_duplicate_id", entryPath, "duplicate rule ID %q", value.ID)
		}

		b.rules[value.ID] = value
		b.ruleOrder = append(b.ruleOrder, value.ID)
	}

	return nil
}

func (b *Book) loadEntry(directory, entryPath string, target any) error {
	err := validateEntryPath(directory, entryPath)
	if err != nil {
		return err
	}

	data, err := fs.ReadFile(b.source, entryPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", entryPath, err)
	}

	return decodeYAML(entryPath, data, target)
}

func validateEntryPath(directory, entryPath string) error {
	if !fs.ValidPath(entryPath) || !strings.HasPrefix(entryPath, directory+"/") || !strings.HasSuffix(entryPath, ".yaml") {
		return validationError("book_invalid_path", manifestPath, "invalid %s entry path %q", directory, entryPath)
	}

	return nil
}

func decodeYAML(name string, data []byte, target any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	err := decoder.Decode(target)
	if err != nil {
		return fmt.Errorf("decode %s: %w", name, err)
	}

	var extra any

	err = decoder.Decode(&extra)
	if err == io.EOF {
		return nil
	}

	if err != nil {
		return fmt.Errorf("decode trailing content in %s: %w", name, err)
	}

	return fmt.Errorf("decode %s: multiple YAML documents are not allowed", name)
}
