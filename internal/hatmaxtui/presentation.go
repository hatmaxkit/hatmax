// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package hatmaxtui

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/textarea"
	"charm.land/lipgloss/v2"
	"hatmax.adrianpk.com/generator/conversation"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/interaction"
	"hatmax.adrianpk.com/generator/plan"
)

const (
	composerPromptWidth      = 2
	surfaceHorizontalMargin  = 1
	surfaceHorizontalPadding = 1
	composerHorizontalFrame  = 2 * (surfaceHorizontalMargin + surfaceHorizontalPadding)
)

var (
	hatmaxBlue     = lipgloss.Color("#4F7FA8")
	activityTrack  = lipgloss.Color("#30323A")
	userBackground = lipgloss.Color("#292C34")
	mutedText      = lipgloss.Color("#7C8290")
)

func composerPrompt(info textarea.PromptInfo) string {
	if info.LineNumber == 0 {
		return "> "
	}

	return ""
}

func composerStyles(styles textarea.Styles) textarea.Styles {
	styles.Focused.Base = styles.Focused.Base.Background(userBackground)
	styles.Focused.CursorLine = styles.Focused.CursorLine.Background(userBackground)
	styles.Focused.Text = styles.Focused.Text.Background(userBackground)
	styles.Focused.Placeholder = styles.Focused.Placeholder.Background(userBackground)
	styles.Focused.Prompt = styles.Focused.Prompt.Background(userBackground).Foreground(hatmaxBlue)
	styles.Focused.EndOfBuffer = styles.Focused.EndOfBuffer.Background(userBackground)
	styles.Blurred.Base = styles.Blurred.Base.Background(userBackground)
	styles.Blurred.CursorLine = styles.Blurred.CursorLine.Background(userBackground)
	styles.Blurred.Text = styles.Blurred.Text.Background(userBackground)
	styles.Blurred.Placeholder = styles.Blurred.Placeholder.Background(userBackground)
	styles.Blurred.Prompt = styles.Blurred.Prompt.Background(userBackground).Foreground(hatmaxBlue)
	styles.Blurred.EndOfBuffer = styles.Blurred.EndOfBuffer.Background(userBackground)

	return styles
}

func (m *model) resizeComposer() {
	contentWidth := max(1, m.width-composerHorizontalFrame-composerPromptWidth)
	height := composerContentHeight(m.composer.Value(), contentWidth)
	m.composer.SetHeight(height)
}

func composerContentHeight(content string, width int) int {
	if width < 1 {
		return 1
	}

	height := 0

	for line := range strings.SplitSeq(content, "\n") {
		lineHeight := max(1, (lipgloss.Width(line)+width-1)/width)
		height += lineHeight
	}

	return min(composerMaxLines, max(1, height))
}

func (m *model) resizeSurfaces() {
	m.viewport.SetWidth(m.width)

	reservedRows := 1 + m.composer.Height() + 1
	if m.busy {
		reservedRows++
	}

	m.viewport.SetHeight(max(1, m.height-reservedRows))
}

func headingView() string {
	return lipgloss.NewStyle().Bold(true).Foreground(hatmaxBlue).Render("Hatmax")
}

func activityView(width int, frame int) string {
	barWidth := max(1, width-2*surfaceHorizontalMargin)
	segmentWidth := min(barWidth, max(4, min(24, barWidth/8)))
	travel := barWidth - segmentWidth
	position := 0

	if travel > 0 {
		cycle := travel * 2
		position = frame % cycle

		if position > travel {
			position = cycle - position
		}
	}

	track := lipgloss.NewStyle().Background(activityTrack)
	pulse := lipgloss.NewStyle().Background(hatmaxBlue)
	bar := track.Render(strings.Repeat(" ", position)) +
		pulse.Render(strings.Repeat(" ", segmentWidth)) +
		track.Render(strings.Repeat(" ", barWidth-position-segmentWidth))

	return strings.Repeat(" ", surfaceHorizontalMargin) + bar
}

func composerView(content string, width int) string {
	return lipgloss.NewStyle().
		Background(userBackground).
		Width(max(1, width-2*surfaceHorizontalMargin)).
		MarginLeft(surfaceHorizontalMargin).
		Padding(0, surfaceHorizontalPadding).
		Render(content)
}

func (m model) footerView() string {
	hints := strings.Join(m.footerHints(), "  ")
	right := m.footerRight()
	contentWidth := max(1, m.width-2*surfaceHorizontalMargin)
	available := max(0, contentWidth-lipgloss.Width(right)-1)
	hints = truncatePlain(hints, available)
	hints = lipgloss.NewStyle().Foreground(mutedText).Render(hints)
	right = lipgloss.NewStyle().Foreground(hatmaxBlue).Render(right)

	gap := max(1, contentWidth-lipgloss.Width(hints)-lipgloss.Width(right))

	return strings.Repeat(" ", surfaceHorizontalMargin) +
		hints + strings.Repeat(" ", gap) + right
}

func (m model) footerHints() []string {
	if m.busy {
		return []string{"Esc cancel"}
	}

	hints := make([]string, 0, 5)
	if m.pendingID != "" {
		hints = append(hints, "Ctrl+A approve")
	}

	if m.detail != "" {
		label := "Ctrl+D details"
		if m.showDetail {
			label = "Ctrl+D summary"
		}

		hints = append(hints, label)
	}

	if m.showHelp {
		hints = append(hints, "Ctrl+J newline", "Ctrl+N new conversation")
		if m.pendingID != "" {
			hints = append(hints, "Esc cancel")
		} else if strings.TrimSpace(m.composer.Value()) != "" {
			hints = append(hints, "Esc clear")
		}

		return append(hints, "Ctrl+C quit")
	}

	if m.pendingID != "" {
		hints = append(hints, "Esc cancel")
	} else if strings.TrimSpace(m.composer.Value()) != "" {
		hints = append(hints, "Esc clear")
	}

	return hints
}

func (m model) footerRight() string {
	if m.busy {
		return m.status + "…"
	}

	if m.showHelp {
		return "F1 close"
	}

	return "F1 help"
}

func truncatePlain(value string, width int) string {
	if width <= 0 {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= width {
		return value
	}

	if width == 1 {
		return "…"
	}

	return string(runes[:width-1]) + "…"
}

func conversationPresentation(
	value conversation.Conversation,
	summary string,
	detail string,
	showDetail bool,
	width int,
) string {
	blocks := make([]string, 0, len(value.Turns)+2)
	if len(value.Turns) == 0 && summary == "" {
		blocks = append(blocks, lipgloss.NewStyle().
			Foreground(mutedText).
			Width(max(1, width)).
			Render("Build and evolve Hatmax applications through conversation."))
	}

	for index := 0; index < len(value.Turns); {
		turn := value.Turns[index]
		if summary != "" && turn.Kind == conversation.TurnResult && strings.HasPrefix(turn.Content, "Plan ready:") {
			index++

			continue
		}

		if turn.Role == conversation.RoleHatmax && turn.Kind == conversation.TurnResult {
			diagnostics := make([]string, 0, 1)
			index++

			for index < len(value.Turns) {
				next := value.Turns[index]
				if next.Role != conversation.RoleHatmax || next.Kind != conversation.TurnDiagnostic || next.OperationID != turn.OperationID {
					break
				}

				diagnostics = append(diagnostics, next.Content)
				index++
			}

			blocks = append(blocks, assistantTurn(resultGroupPresentation(turn.Content, diagnostics), width))

			continue
		}

		if turn.Role == conversation.RoleHatmax && turn.Kind == conversation.TurnClarification {
			questions := []string{turn.Content}
			index++

			for index < len(value.Turns) {
				next := value.Turns[index]
				if next.Role != conversation.RoleHatmax || next.Kind != conversation.TurnClarification {
					break
				}

				questions = append(questions, next.Content)
				index++
			}

			blocks = append(blocks, assistantTurn(clarificationPresentation(questions), width))

			continue
		}

		blocks = append(blocks, conversationTurn(turn, width))
		index++
	}

	if summary != "" {
		blocks = append(blocks, assistantTurn(summary, width))
	}

	if showDetail && detail != "" {
		blocks = append(blocks, technicalDetail(detail, width))
	}

	return strings.Join(blocks, "\n\n")
}

func clarificationPresentation(questions []string) string {
	if len(questions) == 1 {
		return questions[0]
	}

	var output strings.Builder
	output.WriteString("Before I can continue, I need these details:")

	for index, question := range questions {
		fmt.Fprintf(&output, "\n%d. %s", index+1, question)
	}

	return output.String()
}

func conversationTurn(turn conversation.Turn, width int) string {
	content := turn.Content
	if turn.Kind == conversation.TurnDiagnostic {
		content = diagnosticPresentation(content)
	} else if turn.Kind == conversation.TurnResult {
		content = resultTurnPresentation(content)
	}

	if turn.Role == conversation.RoleUser {
		return lipgloss.NewStyle().
			Background(userBackground).
			Width(max(1, width-2*surfaceHorizontalMargin)).
			MarginLeft(surfaceHorizontalMargin).
			Padding(0, surfaceHorizontalPadding).
			Render("You\n" + content)
	}

	return assistantTurn(content, width)
}

func assistantTurn(content string, width int) string {
	label := lipgloss.NewStyle().Bold(true).Foreground(hatmaxBlue).Render("Hatmax")
	body := lipgloss.NewStyle().Width(max(1, width)).Render(content)

	return label + "\n" + body
}

func technicalDetail(content string, width int) string {
	label := lipgloss.NewStyle().Bold(true).Foreground(mutedText).Render("Technical details")
	body := lipgloss.NewStyle().Foreground(mutedText).Width(max(1, width)).Render(content)

	return label + "\n" + body
}

func diagnosticPresentation(content string) string {
	code, message, found := strings.Cut(content, ": ")
	if !found || !strings.HasPrefix(code, "HMGEN-") {
		return conciseDiagnosticMessage("", content)
	}

	return conciseDiagnosticMessage(code, message)
}

func failureSummary(lead string, err error) string {
	if err == nil {
		return lead
	}

	return lead + "\n" + diagnosticPresentation(err.Error())
}

func resultTurnPresentation(content string) string {
	prefix, message, found := strings.Cut(content, ": ")
	if !found {
		return humanizeIdentifier(content)
	}

	if prefix == "validation_incomplete" {
		return "Validation incomplete\nThe generated changes were applied, but required external tests could not run."
	}

	labels := map[string]string{
		"completed":        "Completed",
		"execution_failed": "Execution failed",
		"plan_stale":       "Plan stale",
		"intent_rejected":  "Request rejected",
		"unsupported":      "Unsupported request",
		"cancelled":        "Cancelled",
		"failed":           "Failed",
	}

	label, known := labels[prefix]
	if !known {
		return conciseDiagnosticMessage("", content)
	}

	message = conciseDiagnosticMessage("", message)
	if message == "" {
		return label
	}

	return label + "\n" + message
}

func resultGroupPresentation(content string, diagnostics []string) string {
	prefix, _, _ := strings.Cut(content, ": ")
	if prefix != "validation_incomplete" || len(diagnostics) == 0 {
		return resultTurnPresentation(content)
	}

	return "Validation incomplete\nThe generated changes were applied. " + diagnosticPresentation(diagnostics[0])
}

func conciseDiagnosticMessage(code string, message string) string {
	normalized := strings.ToLower(code + " " + message)
	if strings.Contains(normalized, "docker") || strings.Contains(normalized, "testcontainers") {
		if strings.Contains(normalized, "permission denied") {
			return "Docker-based tests could not run because Docker is not accessible to the current user."
		}

		return "Docker-based tests could not run because Docker infrastructure is unavailable."
	}

	if strings.Contains(normalized, "executable file not found") {
		return "A required executable is not available on PATH."
	}

	message, _, _ = strings.Cut(strings.TrimSpace(message), "\n")
	if strings.HasPrefix(strings.ToLower(message), "observed ") {
		return "Validation did not meet the expected condition."
	}

	return truncatePlain(message, 240)
}

func resultSummary(result conversation.SessionResult) string {
	if result.Interaction.Outcome != interaction.OutcomePlanReady || result.Interaction.Plan == nil {
		return ""
	}

	return planSummary(result.Interaction.Plan)
}

func planSummary(value *plan.Plan) string {
	if value == nil {
		return ""
	}

	var output strings.Builder
	output.WriteString("Plan ready\n\n")
	writePlanGoal(&output, value)
	writePlanDomain(&output, value)
	writePlanCapabilities(&output, value)
	writePlanSurfaces(&output, value)
	writePlanDocumentation(&output, value)

	return strings.TrimSpace(output.String())
}

func writePlanGoal(output *strings.Builder, value *plan.Plan) {
	switch value.Intent {
	case intent.OperationCreateApplication:
		name := "a Hatmax application"
		if value.Application != nil && value.Application.DisplayName != "" {
			name = "the " + value.Application.DisplayName + " Hatmax application"
		}

		fmt.Fprintf(output, "Create %s.\n", name)

		if value.Application != nil && value.Application.ModulePath != "" {
			fmt.Fprintf(output, "Module: %s\n", value.Application.ModulePath)
		}

		if value.Target != nil && value.Target.Directory != "" {
			fmt.Fprintf(output, "Directory: %s\n", value.Target.Directory)
		}

		writeInitialFeatures(output, value.Units)
	case intent.OperationCreateFeature:
		fmt.Fprintf(output, "Create the %s feature.\n", humanizeIdentifier(value.Feature))
	case intent.OperationAddField:
		field := value.Domain.Field
		if field == nil {
			fmt.Fprintf(output, "Add a field to the %s feature.\n", humanizeIdentifier(value.Feature))
		} else {
			fmt.Fprintf(
				output,
				"Add the %s to the %s feature.\n",
				fieldGoalPresentation(*field),
				humanizeIdentifier(value.Feature),
			)
		}
	case intent.OperationAddValidation:
		rule := value.Domain.Validation
		if rule == nil {
			fmt.Fprintf(output, "Add validation to the %s feature.\n", humanizeIdentifier(value.Feature))
		} else {
			fmt.Fprintf(
				output,
				"Add %s validation to %s in the %s feature.\n",
				humanizeIdentifier(rule.Kind),
				humanizeIdentifier(rule.Field),
				humanizeIdentifier(value.Feature),
			)
		}
	case intent.OperationDocumentFeature:
		fmt.Fprintf(output, "Document the %s feature.\n", humanizeIdentifier(value.Feature))
	default:
		if value.Intent == "" {
			output.WriteString("Review the proposed Hatmax change.\n")
		} else {
			fmt.Fprintf(output, "%s.\n", humanizeIdentifier(string(value.Intent)))
		}
	}
}

func writeInitialFeatures(output *strings.Builder, units []plan.Unit) {
	for _, unit := range units {
		if unit.Intent != intent.OperationCreateFeature {
			continue
		}

		fmt.Fprintf(output, "\nInitial feature: %s\n", humanizeIdentifier(unit.Feature))
		writeFields(output, unit.Domain.Fields)
		writeDomainMetadata(output, unit.Domain)
	}
}

func writePlanDomain(output *strings.Builder, value *plan.Plan) {
	if value.Intent == intent.OperationCreateApplication {
		return
	}

	writeFields(output, value.Domain.Fields)
	writeDomainMetadata(output, value.Domain)
}

func writeFields(output *strings.Builder, fields []intent.Field) {
	if len(fields) == 0 {
		return
	}

	output.WriteString("Fields:\n")

	for _, field := range fields {
		fmt.Fprintf(output, "- %s\n", fieldPresentation(field))
	}
}

func writeDomainMetadata(output *strings.Builder, domain intent.Domain) {
	if domain.Entity != "" {
		fmt.Fprintf(output, "Entity: %s\n", domain.Entity)
	}

	if domain.Route != "" {
		fmt.Fprintf(output, "Route: %s\n", domain.Route)
	}

	for _, rule := range domain.Rules {
		fmt.Fprintf(output, "Behavior: %s\n", rule.Description)
	}
}

func fieldPresentation(field intent.Field) string {
	requirement := "optional"
	if field.Required {
		requirement = "required"
	}

	return fmt.Sprintf("%s: %s, %s", humanizeIdentifier(field.Name), field.Type, requirement)
}

func fieldGoalPresentation(field intent.Field) string {
	requirement := "optional"
	if field.Required {
		requirement = "required"
	}

	return fmt.Sprintf("%s %s field %s", requirement, field.Type, humanizeIdentifier(field.Name))
}

func writePlanCapabilities(output *strings.Builder, value *plan.Plan) {
	capabilities := append([]string{}, value.Capabilities...)
	for _, unit := range value.Units {
		capabilities = append(capabilities, unit.Capabilities...)
	}

	capabilities = uniqueStrings(capabilities)
	if len(capabilities) == 0 {
		return
	}

	labels := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		labels = append(labels, capabilityLabel(capability))
	}

	fmt.Fprintf(output, "\nIncludes: %s\n", strings.Join(labels, ", "))
}

func writePlanSurfaces(output *strings.Builder, value *plan.Plan) {
	surfaces := append([]string{}, value.AffectedSurfaces...)
	for _, unit := range value.Units {
		surfaces = append(surfaces, unit.AffectedSurfaces...)
	}

	surfaces = uniqueStrings(surfaces)
	if len(surfaces) == 0 {
		return
	}

	labels := make([]string, 0, len(surfaces))
	for _, surface := range surfaces {
		labels = append(labels, humanizeIdentifier(surface))
	}

	fmt.Fprintf(output, "Changes: %s\n", strings.Join(labels, ", "))
}

func writePlanDocumentation(output *strings.Builder, value *plan.Plan) {
	if value.Documentation == "" {
		return
	}

	fmt.Fprintf(output, "Documentation: %s", documentationLabel(value.Documentation))

	if len(value.DocumentationTargets) == 0 {
		output.WriteByte('\n')

		return
	}

	labels := make([]string, 0, len(value.DocumentationTargets))
	for _, target := range value.DocumentationTargets {
		labels = append(labels, humanizeIdentifier(string(target.Quadrant))+" for "+target.Subject)
	}

	fmt.Fprintf(output, " (%s)\n", strings.Join(labels, ", "))
}

func capabilityLabel(value string) string {
	switch value {
	case "postgres_persistence":
		return "PostgreSQL persistence"
	case "htmx_form":
		return "server-rendered HTMX form"
	case "runtime_validation":
		return "runtime validation"
	default:
		return humanizeIdentifier(value)
	}
}

func documentationLabel(value intent.Documentation) string {
	switch value {
	case intent.DocumentationNotRequested:
		return "not requested"
	case intent.DocumentationExisting:
		return "existing behavior"
	case intent.DocumentationPlanned:
		return "planned change"
	default:
		return humanizeIdentifier(string(value))
	}
}

func humanizeIdentifier(value string) string {
	words := strings.ReplaceAll(strings.ReplaceAll(value, "_", " "), "-", " ")

	words = strings.TrimSpace(words)
	if words == "" {
		return ""
	}

	runes := []rune(words)
	runes[0] = unicode.ToUpper(runes[0])

	return string(runes)
}

func uniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || slices.Contains(result, value) {
			continue
		}

		result = append(result, value)
	}

	return result
}

func outcomeStatus(outcome interaction.Outcome) string {
	switch outcome {
	case interaction.OutcomeConversationResponse:
		return "Ready"
	case interaction.OutcomeClarificationRequired:
		return "Needs input"
	case interaction.OutcomePlanReady:
		return "Plan ready"
	case interaction.OutcomeCompleted:
		return "Completed"
	case interaction.OutcomeValidationIncomplete:
		return "Validation incomplete"
	case interaction.OutcomeUnsupported:
		return "Unsupported"
	case interaction.OutcomeIntentRejected:
		return "Rejected"
	case interaction.OutcomePlanStale:
		return "Plan stale"
	case interaction.OutcomeExecutionFailed, interaction.OutcomeFailed:
		return "Failed"
	case interaction.OutcomeCancelled:
		return "Cancelled"
	default:
		return "Failed"
	}
}

func resultDetail(result conversation.SessionResult) string {
	var detail strings.Builder

	interactionResult := result.Interaction

	if len(interactionResult.PlanYAML) > 0 {
		fmt.Fprintf(&detail, "Typed plan\n%s", interactionResult.PlanYAML)

		if !strings.HasSuffix(detail.String(), "\n") {
			detail.WriteByte('\n')
		}
	}

	for _, diagnostic := range interactionResult.Diagnostics {
		fmt.Fprintf(
			&detail,
			"%s (%s): %s\n",
			diagnostic.Code,
			diagnostic.Phase,
			conciseDiagnosticMessage(diagnostic.Code, diagnostic.Message),
		)
	}

	for _, change := range interactionResult.FreshnessChanges {
		fmt.Fprintf(&detail, "Stale %s: %s\n", change.Kind, change.Path)
	}

	for _, change := range interactionResult.RetainedChanges {
		fmt.Fprintf(&detail, "Retained %s: %s\n", change.Kind, change.Target)
	}

	if interactionResult.Report != nil {
		fmt.Fprintf(&detail, "Conformance: %t\n", interactionResult.Report.Conformance.Passed)

		for _, command := range interactionResult.Report.Commands {
			fmt.Fprintf(&detail, "Validation %s: exit %d\n", command.Name, command.ExitCode)
		}
	}

	if result.PersistenceFailed {
		detail.WriteString("Local conversation persistence failed; this session continues in memory only.\n")
	}

	return strings.TrimSpace(detail.String())
}
