package intent

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"golang.org/x/mod/semver"
	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/project"
)

// ValidationContext binds an intent to one inspected project, relevant-state
// fingerprint, and validated Book.
type ValidationContext struct {
	Inventory   project.Inventory
	Target      *project.TargetInventory
	Fingerprint project.Fingerprint
	Book        *book.Book
}

// Validate semantically validates a typed intent without producing a plan or
// modifying the project.
func Validate(value Intent, context ValidationContext) Result {
	normalize(&value)
	normalizeApplicationContext(&value, context.Target)

	result := Result{Intent: value}

	err := ValidateSchema(value)
	if err != nil {
		result.Status = StatusInvalid
		result.Diagnostics = []Diagnostic{schemaDiagnostic(err)}

		return result
	}

	if context.Book == nil {
		return incompatibleResult(value, "HMGEN-BOOK-MISSING", "book_version", "a validated Hatmax Book is required")
	}

	manifest := context.Book.Manifest()
	if value.BookVersion != manifest.BookVersion || context.Fingerprint.BookVersion != value.BookVersion {
		return incompatibleResult(
			value,
			"HMGEN-BOOK-VERSION",
			"book_version",
			fmt.Sprintf("Book version %d does not match the selected project context", value.BookVersion),
		)
	}

	if intentFingerprint(value) != context.Fingerprint.Value {
		return incompatibleResult(
			value,
			fingerprintDiagnosticCode(value),
			fingerprintField(value),
			"source fingerprint does not match the inspected relevant state",
		)
	}

	versionDiagnostic := validateHatmaxVersion(value, context)
	if versionDiagnostic != nil {
		return Result{
			Status:      StatusIncompatible,
			Intent:      value,
			Diagnostics: []Diagnostic{*versionDiagnostic},
		}
	}

	archetype, exists := context.Book.Archetype(value.Archetype)
	if !exists {
		return incompatibleResult(
			value,
			"HMGEN-ARCHETYPE-UNSUPPORTED",
			"archetype",
			fmt.Sprintf("archetype %q is not supported by the selected Book", value.Archetype),
		)
	}

	operation, exists := findOperation(archetype, value.Operation)
	if !exists {
		return incompatibleResult(
			value,
			"HMGEN-OPERATION-UNSUPPORTED",
			"operation",
			fmt.Sprintf("operation %q is not supported by archetype %q", value.Operation, value.Archetype),
		)
	}

	selection, capabilityResult := validateCapabilities(value, context.Book, archetype, operation)
	if capabilityResult != nil {
		return *capabilityResult
	}

	var diagnostics []Diagnostic
	var clarifications []Clarification
	if value.Operation == OperationCreateApplication {
		diagnostics, clarifications = validateApplication(value, context.Target)
	} else {
		diagnostics, clarifications = validateDomain(value, context.Inventory)
	}
	if len(diagnostics) > 0 {
		sortDiagnostics(diagnostics)

		return Result{
			Status:      StatusIncompatible,
			Intent:      value,
			Selection:   selection,
			Diagnostics: diagnostics,
		}
	}

	if len(clarifications) > 0 {
		sortClarifications(clarifications)

		return Result{
			Status:         StatusClarificationRequired,
			Intent:         value,
			Selection:      selection,
			Clarifications: clarifications,
		}
	}

	exceptionResult := validateExceptions(value, context.Book, selection)
	if exceptionResult != nil {
		return *exceptionResult
	}

	result.Status = StatusAdmitted
	result.Selection = selection
	result.Diagnostics = []Diagnostic{}
	result.Clarifications = []Clarification{}

	return result
}

func schemaDiagnostic(err error) Diagnostic {
	var schemaErr SchemaError
	if errors.As(err, &schemaErr) {
		return Diagnostic{
			Code:    "HMGEN-" + strings.ToUpper(strings.ReplaceAll(schemaErr.Code, "_", "-")),
			Field:   schemaErr.Path,
			Message: schemaErr.Message,
		}
	}

	return Diagnostic{
		Code:    "HMGEN-INTENT-SCHEMA",
		Message: err.Error(),
	}
}

func incompatibleResult(value Intent, code, field, message string) Result {
	return Result{
		Status: StatusIncompatible,
		Intent: value,
		Diagnostics: []Diagnostic{{
			Code:    code,
			Field:   field,
			Message: message,
		}},
	}
}

func validateHatmaxVersion(value Intent, context ValidationContext) *Diagnostic {
	intentVersion := canonicalVersion(value.HatmaxVersion)
	if intentVersion == "" {
		return &Diagnostic{
			Code:    "HMGEN-HATMAX-VERSION",
			Field:   "hatmax_version",
			Message: fmt.Sprintf("Hatmax version %q is invalid", value.HatmaxVersion),
		}
	}

	if value.Operation == OperationCreateApplication {
		supported, err := context.Book.SupportsHatmax(intentVersion)
		if err != nil || !supported {
			return &Diagnostic{
				Code: "HMGEN-BOOK-INCOMPATIBLE", Field: "hatmax_version", Message: fmt.Sprintf("selected Book does not support Hatmax %q", value.HatmaxVersion),
			}
		}

		return nil
	}

	projectVersion := canonicalVersion(context.Inventory.Module.Hatmax.Version)
	if projectVersion == "" {
		return &Diagnostic{
			Code:    "HMGEN-HATMAX-MISSING",
			Field:   "hatmax_version",
			Message: "the inspected project has no versioned Hatmax dependency",
		}
	}

	if intentVersion != projectVersion {
		return &Diagnostic{
			Code:    "HMGEN-HATMAX-VERSION-MISMATCH",
			Field:   "hatmax_version",
			Message: fmt.Sprintf("intent Hatmax version %q does not match project version %q", value.HatmaxVersion, context.Inventory.Module.Hatmax.Version),
		}
	}

	compatibility, err := context.Inventory.CompatibilityWith(context.Book)
	if err != nil {
		return &Diagnostic{
			Code:    "HMGEN-HATMAX-VERSION",
			Field:   "hatmax_version",
			Message: err.Error(),
		}
	}

	if compatibility != project.BookCompatibilityCompatible {
		return &Diagnostic{
			Code:    "HMGEN-BOOK-INCOMPATIBLE",
			Field:   "hatmax_version",
			Message: fmt.Sprintf("selected Book does not support Hatmax %q", value.HatmaxVersion),
		}
	}

	return nil
}

func intentFingerprint(value Intent) string {
	if value.SchemaVersion == ApplicationSchemaVersion {
		return value.SourceFingerprint
	}

	return value.ProjectFingerprint
}

func fingerprintField(value Intent) string {
	if value.SchemaVersion == ApplicationSchemaVersion {
		return "source_fingerprint"
	}

	return "project_fingerprint"
}

func fingerprintDiagnosticCode(value Intent) string {
	if value.SchemaVersion == ApplicationSchemaVersion {
		return "HMGEN-SOURCE-FINGERPRINT"
	}

	return "HMGEN-PROJECT-FINGERPRINT"
}

func canonicalVersion(value string) string {
	normalized := value
	if !strings.HasPrefix(normalized, "v") {
		normalized = "v" + normalized
	}

	return semver.Canonical(normalized)
}

func findOperation(archetype book.Archetype, operation Operation) (book.Operation, bool) {
	for _, candidate := range archetype.Operations {
		if candidate.ID == string(operation) {
			return candidate, true
		}
	}

	return book.Operation{}, false
}

func sortDiagnostics(diagnostics []Diagnostic) {
	sort.Slice(diagnostics, func(left, right int) bool {
		if diagnostics[left].Field != diagnostics[right].Field {
			return diagnostics[left].Field < diagnostics[right].Field
		}

		return diagnostics[left].Code < diagnostics[right].Code
	})
}

func sortClarifications(clarifications []Clarification) {
	sort.Slice(clarifications, func(left, right int) bool {
		return clarifications[left].Field < clarifications[right].Field
	})
}
