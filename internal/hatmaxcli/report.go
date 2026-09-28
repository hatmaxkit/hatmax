package hatmaxcli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"hatmax.adrianpk.com/generator/execute"
	"hatmax.adrianpk.com/generator/interaction"
)

func writeResult(output io.Writer, result interaction.Result) error {
	var report strings.Builder

	fmt.Fprintf(&report, "Outcome: %s\n", result.Outcome)
	writePlanSummary(&report, result)
	writeProvenance(&report, result)
	writeClarifications(&report, result)
	writeFreshnessChanges(&report, result)
	writeExecution(&report, result)
	writeValidation(&report, result)
	writeDiagnostics(&report, result)

	_, err := io.WriteString(output, report.String())

	return err
}

func writePlanSummary(report *strings.Builder, result interaction.Result) {
	if result.Plan == nil {
		return
	}

	fmt.Fprintln(report, "Plan summary:")
	fmt.Fprintf(report, "  Intent: %s\n", result.Plan.Intent)
	fmt.Fprintf(report, "  Feature: %s\n", result.Plan.Feature)
	fmt.Fprintf(report, "  Digest: %s\n", result.Plan.Digest)
	fmt.Fprintf(report, "  Project fingerprint: %s\n", result.Plan.ProjectFingerprint)
	fmt.Fprintf(report, "  Hatmax version: %s\n", result.Plan.HatmaxVersion)
	fmt.Fprintf(report, "  Book version: %d\n", result.Plan.BookVersion)
	fmt.Fprintf(report, "  Affected surfaces: %s\n", stringList(result.Plan.AffectedSurfaces))
}

func writeProvenance(report *strings.Builder, result interaction.Result) {
	if len(result.Provenance.Interpretations) == 0 {
		return
	}

	fmt.Fprintln(report, "Interpreter provenance:")

	for index, provenance := range result.Provenance.Interpretations {
		fmt.Fprintf(report, "  Turn %d:\n", index+1)
		fmt.Fprintf(report, "    Adapter: %s\n", provenance.Adapter)
		fmt.Fprintf(report, "    Contract version: %d\n", provenance.ContractVersion)
		fmt.Fprintf(report, "    Backend version: %s\n", provenance.BackendVersion)
		fmt.Fprintf(report, "    Protocol version: %s\n", provenance.ProtocolVersion)
		fmt.Fprintf(report, "    Model selection: %s\n", provenance.ModelSelection)
		fmt.Fprintf(report, "    Effective model: %s\n", provenance.EffectiveModel)
		fmt.Fprintf(report, "    Runtime reused: %t\n", provenance.RuntimeReused)
		fmt.Fprintf(report, "    Thread reused: %t\n", provenance.ThreadReused)
		fmt.Fprintf(report, "    Timing: %s\n", provenance.Timing)
	}
}

func writeClarifications(report *strings.Builder, result interaction.Result) {
	if len(result.Clarifications) == 0 {
		return
	}

	fmt.Fprintln(report, "Pending clarifications:")

	for _, clarification := range result.Clarifications {
		fmt.Fprintf(report, "  %s: %s\n", clarification.Field, clarification.Question)
	}
}

func writeFreshnessChanges(report *strings.Builder, result interaction.Result) {
	if len(result.FreshnessChanges) == 0 {
		return
	}

	fmt.Fprintln(report, "Project drift:")

	for _, change := range result.FreshnessChanges {
		fmt.Fprintf(
			report,
			"  %s %s path=%s surface=%s\n",
			change.Kind,
			change.Class,
			change.Path,
			change.Surface,
		)
	}
}

func writeExecution(report *strings.Builder, result interaction.Result) {
	if result.Execution == nil {
		return
	}

	fmt.Fprintln(report, "Execution:")

	for _, change := range result.Execution.Changes {
		fmt.Fprintf(report, "  %s %s %s\n", change.Status, change.Kind, change.Target)
	}

	changed, unchanged := surfaceDisposition(result.Manifest, result.Execution)
	fmt.Fprintf(report, "  Changed surfaces: %s\n", stringList(changed))
	fmt.Fprintf(report, "  Unchanged surfaces: %s\n", stringList(unchanged))
}

func writeValidation(report *strings.Builder, result interaction.Result) {
	if result.Report == nil {
		return
	}

	fmt.Fprintln(report, "Validation:")
	fmt.Fprintf(report, "  Conformance passed: %t\n", result.Report.Conformance.Passed)

	for _, command := range result.Report.Commands {
		arguments, _ := json.Marshal(command.Args)
		fmt.Fprintf(
			report,
			"  Command %s: kind=%s cwd=%s args=%s exit=%d\n",
			command.Name,
			command.Kind,
			command.WorkingDirectory,
			arguments,
			command.ExitCode,
		)

		if command.Output != "" {
			fmt.Fprintf(report, "    Output: %s\n", indentMultiline(strings.TrimSuffix(command.Output, "\n"), "    "))
		}
	}

	for _, repair := range result.Report.Repairs {
		fmt.Fprintf(report, "  Repair: %s\n", repair)
	}

	for _, warning := range result.Report.Warnings {
		fmt.Fprintf(report, "  Warning: %s\n", warning)
	}
}

func writeDiagnostics(report *strings.Builder, result interaction.Result) {
	if len(result.Diagnostics) == 0 {
		return
	}

	fmt.Fprintln(report, "Diagnostics:")

	for _, diagnostic := range result.Diagnostics {
		fmt.Fprintf(
			report,
			"  %s phase=%s field=%s: %s\n",
			diagnostic.Code,
			diagnostic.Phase,
			diagnostic.Field,
			diagnostic.Message,
		)
	}
}

func surfaceDisposition(manifest *execute.Manifest, result *execute.Result) ([]string, []string) {
	if manifest == nil {
		return nil, nil
	}

	editSurfaces := make(map[string]string, len(manifest.Edits))
	for _, edit := range manifest.Edits {
		editSurfaces[edit.ID] = edit.Surface
	}

	changedSet := make(map[string]struct{})

	for _, change := range result.Changes {
		if change.Status != execute.ChangeApplied {
			continue
		}

		surface := editSurfaces[change.EditID]
		if surface != "" {
			changedSet[surface] = struct{}{}
		}
	}

	changed := make([]string, 0, len(changedSet))

	unchanged := make([]string, 0, len(manifest.AllowedSurfaces)-len(changedSet))
	for _, surface := range manifest.AllowedSurfaces {
		if _, ok := changedSet[surface]; ok {
			changed = append(changed, surface)
		} else {
			unchanged = append(unchanged, surface)
		}
	}

	sort.Strings(changed)
	sort.Strings(unchanged)

	return changed, unchanged
}

func stringList(values []string) string {
	if len(values) == 0 {
		return "[]"
	}

	encoded, _ := json.Marshal(values)

	return string(encoded)
}

func indentMultiline(value, indent string) string {
	return strings.ReplaceAll(value, "\n", "\n"+indent)
}
