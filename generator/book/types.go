// Package book loads and validates the versioned Hatmax Book.
package book

import "io/fs"

// CurrentSchemaVersion is the Book schema understood by this package.
const CurrentSchemaVersion = 1

// Level describes the normative force of a Book rule.
type Level string

const (
	// LevelRequired marks behavior that conforming projects must provide.
	LevelRequired Level = "required"
	// LevelRecommended marks preferred behavior that produces warnings.
	LevelRecommended Level = "recommended"
	// LevelOptional marks an admitted extension point.
	LevelOptional Level = "optional"
	// LevelProhibited marks behavior that conforming projects must not use.
	LevelProhibited Level = "prohibited"
)

// VersionRange selects Hatmax releases compatible with a Book.
type VersionRange struct {
	Minimum          string `yaml:"minimum"`
	MaximumExclusive string `yaml:"maximum_exclusive"`
}

// Manifest identifies one complete Book and all its entries.
type Manifest struct {
	SchemaVersion int          `yaml:"schema_version"`
	BookVersion   int          `yaml:"book_version"`
	Hatmax        VersionRange `yaml:"hatmax"`
	Capabilities  []string     `yaml:"capabilities"`
	Archetypes    []string     `yaml:"archetypes"`
	Rules         []string     `yaml:"rules"`
}

// Obligation describes work and rules introduced by a capability or archetype.
type Obligation struct {
	ID         string       `yaml:"id"`
	Surfaces   []string     `yaml:"surfaces"`
	Rules      []string     `yaml:"rules"`
	DependsOn  []string     `yaml:"depends_on,omitempty"`
	Operations []string     `yaml:"operations,omitempty"`
	Files      []FileEffect `yaml:"files,omitempty"`
}

// FileEffectMode describes how an obligation affects one Book-owned path.
type FileEffectMode string

const (
	// FileEffectCreate requires a previously absent path.
	FileEffectCreate FileEffectMode = "create"
	// FileEffectUpdate requires a path produced by an earlier unit.
	FileEffectUpdate FileEffectMode = "update"
	// FileEffectEnsure creates the path once and updates it in later units.
	FileEffectEnsure FileEffectMode = "ensure"
)

// FileEffect is one static or product-name-templated path owned by a Book
// obligation. Only {feature} and {sequence} placeholders are admitted.
type FileEffect struct {
	Path       string         `yaml:"path"`
	Mode       FileEffectMode `yaml:"mode"`
	Operations []string       `yaml:"operations,omitempty"`
}

// Dependency admits one non-Hatmax module or tool for a bounded purpose.
type Dependency struct {
	Kind       string   `yaml:"kind"`
	Module     string   `yaml:"module"`
	Purpose    string   `yaml:"purpose"`
	Operations []string `yaml:"operations,omitempty"`
}

// Capability describes one supported technical behavior.
type Capability struct {
	ID               string       `yaml:"id"`
	Intent           string       `yaml:"intent"`
	Requires         []string     `yaml:"requires,omitempty"`
	IncompatibleWith []string     `yaml:"incompatible_with,omitempty"`
	Surfaces         []string     `yaml:"surfaces"`
	Packages         []string     `yaml:"packages,omitempty"`
	Dependencies     []Dependency `yaml:"dependencies,omitempty"`
	Obligations      []Obligation `yaml:"obligations"`
	Rules            []string     `yaml:"rules"`
	Extensions       []string     `yaml:"extensions,omitempty"`
	Unsupported      []string     `yaml:"unsupported,omitempty"`
}

// Operation describes obligations introduced by one archetype operation.
type Operation struct {
	ID                   string   `yaml:"id"`
	RequiredCapabilities []string `yaml:"required_capabilities,omitempty"`
	Obligations          []string `yaml:"obligations"`
}

// Archetype defines one canonical Hatmax assembly pattern.
type Archetype struct {
	ID                   string       `yaml:"id"`
	Operations           []Operation  `yaml:"operations"`
	RequiredCapabilities []string     `yaml:"required_capabilities,omitempty"`
	OptionalCapabilities []string     `yaml:"optional_capabilities,omitempty"`
	Surfaces             []string     `yaml:"surfaces"`
	Obligations          []Obligation `yaml:"obligations"`
	ImplementationSlots  []string     `yaml:"implementation_slots,omitempty"`
	Rules                []string     `yaml:"rules"`
}

// Rule defines one enforceable Hatmax construction rule.
type Rule struct {
	ID            string   `yaml:"id"`
	Level         Level    `yaml:"level"`
	Title         string   `yaml:"title"`
	Rationale     string   `yaml:"rationale"`
	AppliesWhen   []string `yaml:"applies_when,omitempty"`
	Requires      []string `yaml:"requires,omitempty"`
	Prohibits     []string `yaml:"prohibits,omitempty"`
	Diagnostics   []string `yaml:"diagnostics"`
	Capabilities  []string `yaml:"capabilities,omitempty"`
	Archetypes    []string `yaml:"archetypes,omitempty"`
	PositiveCases []string `yaml:"positive_examples,omitempty"`
	NegativeCases []string `yaml:"negative_examples,omitempty"`
}

// Selection is the deterministic Book context for one archetype and
// capability set.
type Selection struct {
	Archetype    Archetype
	Capabilities []Capability
	Rules        []Rule
}

// Book is one validated, immutable-in-practice Hatmax Book.
type Book struct {
	source fs.FS

	manifest        Manifest
	capabilityOrder []string
	archetypeOrder  []string
	ruleOrder       []string
	capabilities    map[string]Capability
	archetypes      map[string]Archetype
	rules           map[string]Rule
}

// Manifest returns a detached copy of the Book manifest.
func (b *Book) Manifest() Manifest {
	result := b.manifest
	result.Capabilities = cloneStrings(result.Capabilities)
	result.Archetypes = cloneStrings(result.Archetypes)
	result.Rules = cloneStrings(result.Rules)

	return result
}

// Capability returns one capability by stable ID.
func (b *Book) Capability(id string) (Capability, bool) {
	value, ok := b.capabilities[id]

	return cloneCapability(value), ok
}

// Archetype returns one archetype by stable ID.
func (b *Book) Archetype(id string) (Archetype, bool) {
	value, ok := b.archetypes[id]

	return cloneArchetype(value), ok
}

// Rule returns one rule by stable ID.
func (b *Book) Rule(id string) (Rule, bool) {
	value, ok := b.rules[id]

	return cloneRule(value), ok
}

// Capabilities returns capabilities in manifest order.
func (b *Book) Capabilities() []Capability {
	result := make([]Capability, 0, len(b.capabilityOrder))
	for _, id := range b.capabilityOrder {
		result = append(result, cloneCapability(b.capabilities[id]))
	}

	return result
}

// Archetypes returns archetypes in manifest order.
func (b *Book) Archetypes() []Archetype {
	result := make([]Archetype, 0, len(b.archetypeOrder))
	for _, id := range b.archetypeOrder {
		result = append(result, cloneArchetype(b.archetypes[id]))
	}

	return result
}

// Rules returns rules in manifest order.
func (b *Book) Rules() []Rule {
	result := make([]Rule, 0, len(b.ruleOrder))
	for _, id := range b.ruleOrder {
		result = append(result, cloneRule(b.rules[id]))
	}

	return result
}

func cloneStrings(values []string) []string {
	return append([]string(nil), values...)
}

func cloneObligations(values []Obligation) []Obligation {
	result := make([]Obligation, len(values))
	for index, value := range values {
		result[index] = value
		result[index].Surfaces = cloneStrings(value.Surfaces)
		result[index].Rules = cloneStrings(value.Rules)
		result[index].DependsOn = cloneStrings(value.DependsOn)
		result[index].Operations = cloneStrings(value.Operations)

		result[index].Files = append([]FileEffect(nil), value.Files...)
		for fileIndex := range result[index].Files {
			result[index].Files[fileIndex].Operations = cloneStrings(value.Files[fileIndex].Operations)
		}
	}

	return result
}

func cloneCapability(value Capability) Capability {
	value.Requires = cloneStrings(value.Requires)
	value.IncompatibleWith = cloneStrings(value.IncompatibleWith)
	value.Surfaces = cloneStrings(value.Surfaces)
	value.Packages = cloneStrings(value.Packages)

	value.Dependencies = append([]Dependency(nil), value.Dependencies...)
	for index := range value.Dependencies {
		value.Dependencies[index].Operations = cloneStrings(value.Dependencies[index].Operations)
	}

	value.Obligations = cloneObligations(value.Obligations)
	value.Rules = cloneStrings(value.Rules)
	value.Extensions = cloneStrings(value.Extensions)
	value.Unsupported = cloneStrings(value.Unsupported)

	return value
}

func cloneArchetype(value Archetype) Archetype {
	value.RequiredCapabilities = cloneStrings(value.RequiredCapabilities)
	value.OptionalCapabilities = cloneStrings(value.OptionalCapabilities)
	value.Surfaces = cloneStrings(value.Surfaces)
	value.Obligations = cloneObligations(value.Obligations)
	value.ImplementationSlots = cloneStrings(value.ImplementationSlots)
	value.Rules = cloneStrings(value.Rules)

	value.Operations = append([]Operation(nil), value.Operations...)
	for index := range value.Operations {
		value.Operations[index].RequiredCapabilities = cloneStrings(value.Operations[index].RequiredCapabilities)
		value.Operations[index].Obligations = cloneStrings(value.Operations[index].Obligations)
	}

	return value
}

func cloneRule(value Rule) Rule {
	value.AppliesWhen = cloneStrings(value.AppliesWhen)
	value.Requires = cloneStrings(value.Requires)
	value.Prohibits = cloneStrings(value.Prohibits)
	value.Diagnostics = cloneStrings(value.Diagnostics)
	value.Capabilities = cloneStrings(value.Capabilities)
	value.Archetypes = cloneStrings(value.Archetypes)
	value.PositiveCases = cloneStrings(value.PositiveCases)
	value.NegativeCases = cloneStrings(value.NegativeCases)

	return value
}
