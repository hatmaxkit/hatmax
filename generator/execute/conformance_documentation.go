package execute

import (
	"bytes"
	"fmt"
	"strings"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

const (
	documentationScopeRule       = "hatmax.documentation.explicit_intent"
	documentationDiataxisRule    = "hatmax.documentation.diataxis"
	documentationManagedRule     = "hatmax.documentation.managed_sections"
	documentationNavigationRule  = "hatmax.documentation.index_reachability"
	documentationEvidenceCode    = "HMGEN-DOCUMENTATION-EVIDENCE"
	documentationPlaceholderCode = "HMGEN-DOCUMENTATION-DIATAXIS"
)

func checkDocumentationConformance(snapshot conformanceSnapshot, diagnostics *[]Diagnostic) {
	if snapshot.value.Documentation == intent.DocumentationNotRequested {
		checkInactiveDocumentation(snapshot, diagnostics)

		return
	}

	documentationPlan := snapshot.value.DocumentationPlan
	evidence := snapshot.value.DocumentationEvidence

	if documentationPlan == nil || evidence == nil {
		addConformanceDiagnostic(diagnostics, documentationEvidenceCode, documentationDiataxisRule, "documentation", "documentation_plan", "planned evidence missing", "sealed documentation plan and feature evidence")

		return
	}

	expected := make(map[string]struct{}, len(documentationPlan.Targets)+len(documentationPlan.Indexes))
	for _, target := range documentationPlan.Targets {
		expected[target.Path] = struct{}{}
	}

	for _, index := range documentationPlan.Indexes {
		expected[index.Path] = struct{}{}
	}

	edits := documentationManifestEdits(snapshot, expected, diagnostics)
	for _, target := range documentationPlan.Targets {
		checkDocumentationTarget(snapshot, edits[target.Path], target, *evidence, diagnostics)
	}

	for _, index := range documentationPlan.Indexes {
		checkDocumentationIndex(snapshot, edits[index.Path], index, diagnostics)
	}
}

func checkInactiveDocumentation(snapshot conformanceSnapshot, diagnostics *[]Diagnostic) {
	if !selectedRule(snapshot.value, documentationScopeRule) {
		return
	}

	for _, edit := range snapshot.manifest.Edits {
		if edit.Surface == "documentation" {
			addConformanceDiagnostic(diagnostics, "HMGEN-DOCUMENTATION-SCOPE", documentationScopeRule, "documentation", edit.Target, "documentation edit declared without intent", "no documentation mutations")
		}
	}
}

func documentationManifestEdits(
	snapshot conformanceSnapshot,
	expected map[string]struct{},
	diagnostics *[]Diagnostic,
) map[string]Edit {
	result := make(map[string]Edit, len(expected))

	for _, edit := range snapshot.manifest.Edits {
		if edit.Surface != "documentation" {
			if snapshot.value.Intent == intent.OperationDocumentFeature {
				addConformanceDiagnostic(diagnostics, "HMGEN-DOCUMENTATION-SCOPE", documentationScopeRule, edit.Surface, edit.Target, "non-documentation edit in a documentation-only manifest", "documentation edits only")
			}

			continue
		}

		if _, exists := expected[edit.Target]; !exists {
			addConformanceDiagnostic(diagnostics, "HMGEN-DOCUMENTATION-SCOPE", documentationScopeRule, "documentation", edit.Target, "documentation edit is not declared by the plan", "exact planned documentation target")

			continue
		}

		result[edit.Target] = edit
	}

	for target := range expected {
		if _, exists := result[target]; !exists {
			addConformanceDiagnostic(diagnostics, "HMGEN-DOCUMENTATION-SCOPE", documentationScopeRule, "documentation", target, "planned documentation edit is missing", "one manifest edit for every planned target")
		}
	}

	if snapshot.value.Intent == intent.OperationDocumentFeature && (len(snapshot.manifest.AllowedSurfaces) != 1 || snapshot.manifest.AllowedSurfaces[0] != "documentation" || len(snapshot.value.AllowedEffects.Dependencies) != 0) {
		addConformanceDiagnostic(diagnostics, "HMGEN-DOCUMENTATION-SCOPE", documentationScopeRule, "documentation", "allowed_effects", "documentation-only effects exceed the documentation surface", "documentation surface without runtime dependencies")
	}

	return result
}

func checkDocumentationTarget(
	snapshot conformanceSnapshot,
	edit Edit,
	target plan.DocumentationTargetEffect,
	evidence project.FeatureEvidence,
	diagnostics *[]Diagnostic,
) {
	content, exists := snapshot.files[target.Path]
	if !exists {
		addConformanceDiagnostic(diagnostics, "HMGEN-DOCUMENTATION-DIATAXIS", documentationDiataxisRule, "documentation", target.Path, "target is missing", "canonical generated target")

		return
	}

	managed, valid := checkManagedDocumentation(edit, target.Path, content, diagnostics)
	if !valid {
		return
	}

	checkDocumentationTargetStructure(target, managed, diagnostics)
	checkDocumentationEvidence(target, managed, evidence, diagnostics)
	checkDocumentationLinks(snapshot.inventory, target.Path, diagnostics)

	if containsDocumentationPlaceholder(managed) {
		addConformanceDiagnostic(diagnostics, documentationPlaceholderCode, documentationDiataxisRule, "documentation", target.Path, "placeholder or generator commentary found", "reader-ready evidence-backed documentation")
	}
}

func checkDocumentationIndex(
	snapshot conformanceSnapshot,
	edit Edit,
	index plan.DocumentationIndexEffect,
	diagnostics *[]Diagnostic,
) {
	content, exists := snapshot.files[index.Path]
	if !exists {
		addConformanceDiagnostic(diagnostics, "HMGEN-DOCUMENTATION-INDEX", documentationNavigationRule, "documentation", index.Path, "index is missing", "canonical generated index")

		return
	}

	managed, valid := checkManagedDocumentation(edit, index.Path, content, diagnostics)
	if !valid {
		return
	}

	if !bytes.Contains(managed, []byte("# "+index.Title)) || !bytes.Contains(managed, []byte("## Generated Navigation")) {
		addConformanceDiagnostic(diagnostics, "HMGEN-DOCUMENTATION-INDEX", documentationNavigationRule, "documentation", index.Path, "canonical index headings are missing", "title and generated navigation section")
	}

	for _, link := range index.RequiredLinks {
		expected := fmt.Sprintf("[%s](%s)", link.Title, link.Relative)
		if !bytes.Contains(managed, []byte(expected)) {
			addConformanceDiagnostic(diagnostics, "HMGEN-DOCUMENTATION-INDEX", documentationNavigationRule, "documentation", index.Path, "required link is missing", expected)
		}
	}

	checkDocumentationLinks(snapshot.inventory, index.Path, diagnostics)
}

func checkManagedDocumentation(edit Edit, target string, content []byte, diagnostics *[]Diagnostic) ([]byte, bool) {
	managed, err := managedSectionContent(content)
	if err != nil {
		addConformanceDiagnostic(diagnostics, "HMGEN-DOCUMENTATION-MANAGED-SECTION", documentationManagedRule, "documentation", target, err.Error(), "one balanced Hatmax managed section")

		return nil, false
	}

	for _, condition := range edit.Postconditions {
		if condition.Kind != ConditionManagedOutsideDigest {
			continue
		}

		digest, digestErr := managedOutsideDigest(content)
		if digestErr != nil || digest != condition.Value {
			addConformanceDiagnostic(diagnostics, "HMGEN-DOCUMENTATION-MANAGED-SECTION", documentationManagedRule, "documentation", target, "content outside the managed section changed", "original user-owned bytes")
		}
	}

	return managed, true
}

func checkDocumentationTargetStructure(
	target plan.DocumentationTargetEffect,
	managed []byte,
	diagnostics *[]Diagnostic,
) {
	if !bytes.Contains(managed, []byte("# "+target.Title)) {
		addConformanceDiagnostic(diagnostics, "HMGEN-DOCUMENTATION-DIATAXIS", documentationDiataxisRule, "documentation", target.Path, "reader-oriented title is missing", target.Title)
	}

	required, forbidden := documentationHeadings(target.Quadrant)
	for _, heading := range required {
		if !bytes.Contains(managed, []byte(heading)) {
			addConformanceDiagnostic(diagnostics, "HMGEN-DOCUMENTATION-DIATAXIS", documentationDiataxisRule, "documentation", target.Path, "quadrant structure is incomplete", heading)
		}
	}

	for _, heading := range forbidden {
		if bytes.Contains(managed, []byte(heading)) {
			addConformanceDiagnostic(diagnostics, "HMGEN-DOCUMENTATION-DIATAXIS", documentationDiataxisRule, "documentation", target.Path, "mixed-quadrant heading found", "only canonical headings for "+string(target.Quadrant))
		}
	}

	backlink := "[Back to " + documentationQuadrantLabel(target.Quadrant) + "](../index.md)"
	if !bytes.Contains(managed, []byte(backlink)) {
		addConformanceDiagnostic(diagnostics, "HMGEN-DOCUMENTATION-INDEX", documentationNavigationRule, "documentation", target.Path, "quadrant backlink is missing", backlink)
	}
}

func checkDocumentationEvidence(
	target plan.DocumentationTargetEffect,
	managed []byte,
	evidence project.FeatureEvidence,
	diagnostics *[]Diagnostic,
) {
	required := []string{evidence.Entity, evidence.Route}
	if target.Quadrant == intent.DocumentationReference || target.Quadrant == intent.DocumentationExplanation {
		required = append(required, evidence.Table)
	}

	if target.Quadrant == intent.DocumentationReference || target.Quadrant == intent.DocumentationTutorial || target.Quadrant == intent.DocumentationHowTo {
		for _, field := range evidence.Fields {
			required = append(required, field.Name)
		}
	}

	for _, marker := range required {
		if marker != "" && !bytes.Contains(managed, []byte(marker)) {
			addConformanceDiagnostic(diagnostics, documentationEvidenceCode, documentationDiataxisRule, "documentation", target.Path, "document does not agree with sealed feature evidence", marker)
		}
	}
}

func checkDocumentationLinks(inventory project.Inventory, target string, diagnostics *[]Diagnostic) {
	file, exists := inventory.DocumentationFile(target)
	if !exists {
		return
	}

	for _, link := range file.LocalLinks {
		if !link.Exists {
			addConformanceDiagnostic(diagnostics, "HMGEN-DOCUMENTATION-INDEX", documentationNavigationRule, "documentation", target, "local link does not resolve", link.Target)
		}
	}
}

func documentationHeadings(quadrant intent.DocumentationQuadrant) ([]string, []string) {
	groups := map[intent.DocumentationQuadrant][]string{
		intent.DocumentationTutorial:    {"## Outcome", "## Before You Begin", "## Guided Steps", "## Continue"},
		intent.DocumentationHowTo:       {"## Goal", "## Procedure", "## Related Documentation"},
		intent.DocumentationReference:   {"## Contract", "## Fields", "## Validation", "## Integration"},
		intent.DocumentationExplanation: {"## Context", "## Ownership", "## Tradeoffs", "## Relationship to the Application"},
	}

	required := groups[quadrant]
	forbidden := make([]string, 0)

	for candidate, headings := range groups {
		if candidate != quadrant {
			forbidden = append(forbidden, headings...)
		}
	}

	return required, forbidden
}

func containsDocumentationPlaceholder(content []byte) bool {
	lower := strings.ToLower(string(content))

	return strings.Contains(lower, "todo") || strings.Contains(lower, "tbd") || strings.Contains(lower, "placeholder") || strings.Contains(lower, "model output")
}
