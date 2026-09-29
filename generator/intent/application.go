package intent

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/module"
	"hatmax.adrianpk.com/generator/project"
)

func normalizeApplicationContext(value *Intent, target *project.TargetInventory) {
	if value.Operation != OperationCreateApplication || value.Application == nil || target == nil {
		return
	}

	if value.Application.ModulePath == "" && target.RemoteModulePath != "" {
		value.Application.ModulePath = target.RemoteModulePath
	}
}

func validateApplication(value Intent, target *project.TargetInventory) ([]Diagnostic, []Clarification) {
	diagnostics := make([]Diagnostic, 0)
	clarifications := make([]Clarification, 0)

	if target == nil {
		diagnostics = append(diagnostics, Diagnostic{
			Code: "HMGEN-TARGET-MISSING", Field: "target", Message: "application creation requires inspected target state",
		})

		return diagnostics, clarifications
	}

	validateApplicationIdentity(value.Application, &diagnostics, &clarifications)
	validateApplicationTarget(value.Target, target, &diagnostics)
	validateTargetAdmission(target, &diagnostics)
	validateInitialFeatures(value.InitialFeatures, &diagnostics, &clarifications)

	sortDiagnostics(diagnostics)
	sortClarifications(clarifications)

	return diagnostics, clarifications
}

func validateApplicationIdentity(
	identity *ApplicationIdentity,
	diagnostics *[]Diagnostic,
	clarifications *[]Clarification,
) {
	if identity == nil {
		return
	}

	words, valid := semanticNameWords(identity.DisplayName)
	if identity.DisplayName == "" {
		*clarifications = append(*clarifications, Clarification{
			Field: "application.display_name", Question: "What should the application be called?",
		})
	} else if !valid {
		*diagnostics = append(*diagnostics, Diagnostic{
			Code: "HMGEN-APPLICATION-NAME", Field: "application.display_name", Message: "application name must contain unambiguous ASCII words",
		})
	}

	if identity.ProjectSlug == "" && identity.DisplayName != "" {
		*diagnostics = append(*diagnostics, Diagnostic{
			Code: "HMGEN-PROJECT-SLUG", Field: "application.project_slug", Message: "project slug could not be derived",
		})
	} else if identity.ProjectSlug != "" && (!featureKebabPattern.MatchString(identity.ProjectSlug) || valid && identity.ProjectSlug != strings.Join(words, "-")) {
		*diagnostics = append(*diagnostics, Diagnostic{
			Code: "HMGEN-PROJECT-SLUG", Field: "application.project_slug", Message: fmt.Sprintf("project slug %q must be the lower-kebab application name", identity.ProjectSlug),
		})
	}

	if identity.ModulePath == "" {
		*clarifications = append(*clarifications, Clarification{
			Field: "application.module_path", Question: "What Go module path should the application use?",
		})
	} else {
		err := module.CheckPath(identity.ModulePath)
		if err != nil {
			*diagnostics = append(*diagnostics, Diagnostic{
				Code: "HMGEN-MODULE-PATH", Field: "application.module_path", Message: fmt.Sprintf("module path %q is invalid", identity.ModulePath),
			})
		}
	}
}

func validateApplicationTarget(target *ApplicationTarget, inventory *project.TargetInventory, diagnostics *[]Diagnostic) {
	if target == nil {
		return
	}

	if target.Base != "session_directory" {
		*diagnostics = append(*diagnostics, Diagnostic{
			Code: "HMGEN-TARGET-BASE", Field: "target.base", Message: fmt.Sprintf("target base %q is not supported", target.Base),
		})
	}

	if target.Directory == "" {
		return
	}

	if !featureKebabPattern.MatchString(target.Directory) {
		*diagnostics = append(*diagnostics, Diagnostic{
			Code: "HMGEN-TARGET-DIRECTORY", Field: "target.directory", Message: fmt.Sprintf("target directory %q must use lower kebab case", target.Directory),
		})

		return
	}

	if filepath.Base(inventory.Target) != target.Directory {
		*diagnostics = append(*diagnostics, Diagnostic{
			Code: "HMGEN-TARGET-MISMATCH", Field: "target.directory", Message: "target directory does not match the inspected target",
		})
	}
}

func validateTargetAdmission(target *project.TargetInventory, diagnostics *[]Diagnostic) {
	switch target.Admission {
	case project.TargetAbsent, project.TargetEmpty, project.TargetPreservable:
		return
	case project.TargetCompatibleProject:
		*diagnostics = append(*diagnostics, Diagnostic{
			Code: "HMGEN-TARGET-EVOLUTION-REQUIRED", Field: "target", Message: "target is an existing compatible Hatmax project and requires an evolution operation",
		})
	case project.TargetIncompatible:
		*diagnostics = append(*diagnostics, Diagnostic{
			Code: "HMGEN-TARGET-INCOMPATIBLE", Field: "target", Message: "target is incompatible with canonical application creation",
		})
	default:
		*diagnostics = append(*diagnostics, Diagnostic{
			Code: "HMGEN-TARGET-STATE", Field: "target", Message: fmt.Sprintf("target admission %q is unknown", target.Admission),
		})
	}
}

func validateInitialFeatures(features []InitialFeature, diagnostics *[]Diagnostic, clarifications *[]Clarification) {
	seen := make(map[string]struct{}, len(features))
	for index, feature := range features {
		prefix := fmt.Sprintf("initial_features[%d]", index)
		if !featureNamePattern.MatchString(feature.Feature) {
			*diagnostics = append(*diagnostics, Diagnostic{
				Code: "HMGEN-FEATURE-NAME", Field: prefix + ".feature", Message: fmt.Sprintf("feature %q must use lower snake case", feature.Feature),
			})
		}

		if _, exists := seen[feature.Feature]; exists {
			*diagnostics = append(*diagnostics, Diagnostic{
				Code: "HMGEN-FEATURE-DUPLICATE", Field: prefix + ".feature", Message: fmt.Sprintf("feature %q is duplicated", feature.Feature),
			})
		}

		seen[feature.Feature] = struct{}{}

		candidate := Intent{Operation: OperationCreateFeature, Feature: feature.Feature, Domain: feature.Domain}
		if hasRequiredField(feature.Domain.Fields) {
			candidate.Capabilities = []string{"runtime_validation"}
		}

		featureDiagnostics, featureClarifications := ValidateDomainDecisions(candidate)
		for _, diagnostic := range featureDiagnostics {
			diagnostic.Field = prefix + "." + diagnostic.Field
			*diagnostics = append(*diagnostics, diagnostic)
		}

		for _, clarification := range featureClarifications {
			clarification.Field = prefix + "." + clarification.Field
			*clarifications = append(*clarifications, clarification)
		}
	}
}

func sortInitialFeatures(features []InitialFeature) {
	sort.SliceStable(features, func(left, right int) bool {
		return features[left].Feature < features[right].Feature
	})
}
