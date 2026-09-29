package codex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"hatmax.adrianpk.com/generator/eval"
)

const (
	// DefaultTurnTimeout bounds one interpretation attempt.
	DefaultTurnTimeout = 2 * time.Minute
	// maximumCompiledInputBytes bounds instructions plus the serialized request.
	maximumCompiledInputBytes = 128 << 10
)

// CompiledTurn is the complete bounded input for one schema-constrained turn.
type CompiledTurn struct {
	ThreadID string
	Prompt   string
	Schema   map[string]any
}

// CompileTurn converts one provider-neutral request into the only input Codex
// receives. It contains no target-project path, source file, or command.
func CompileTurn(threadID string, request eval.Request) (CompiledTurn, error) {
	if strings.TrimSpace(threadID) == "" {
		return CompiledTurn{}, backendError(eval.BackendThreadFailed, "compile_turn", "Codex thread identifier is required")
	}

	if request.ContractVersion != eval.CurrentContractVersion {
		return CompiledTurn{}, backendError(eval.BackendOutputInvalid, "compile_turn", "Hatmax interpreter contract version is unsupported")
	}

	requestDocument, err := json.Marshal(request)
	if err != nil {
		return CompiledTurn{}, backendError(eval.BackendOutputInvalid, "compile_turn", "Hatmax interpreter request could not be encoded")
	}

	prompt := interpretationPrompt(requestDocument)
	if len(prompt) > maximumCompiledInputBytes {
		return CompiledTurn{}, backendError(eval.BackendOutputInvalid, "compile_turn", "Hatmax interpreter input exceeds its size limit")
	}

	schemaDocument, err := eval.InterpretationOutputSchema()
	if err != nil {
		return CompiledTurn{}, backendError(eval.BackendOutputInvalid, "compile_schema", "Hatmax interpretation schema could not be encoded")
	}

	var schema map[string]any

	err = json.Unmarshal(schemaDocument, &schema)
	if err != nil {
		return CompiledTurn{}, backendError(eval.BackendOutputInvalid, "compile_schema", "Hatmax interpretation schema is invalid")
	}

	return CompiledTurn{ThreadID: threadID, Prompt: prompt, Schema: schema}, nil
}

func (turn CompiledTurn) parameters() turnStartParameters {
	return turnStartParameters{
		ThreadID:       turn.ThreadID,
		ApprovalPolicy: "never",
		Input: []turnInput{{
			Type: "text",
			Text: turn.Prompt,
		}},
		OutputSchema: turn.Schema,
		TurnTrigger:  "hatmax",
	}
}

func interpretationPrompt(request []byte) string {
	var prompt bytes.Buffer

	prompt.WriteString("Interpret the bounded Hatmax request below.\n")
	prompt.WriteString("Choose exactly one schema result: intent, clarification_required, or unsupported.\n")
	prompt.WriteString("Set payloads for every unchosen result kind to null.\n")
	prompt.WriteString("Use only operations, archetypes, capabilities, and variants present in the request.\n")
	prompt.WriteString("Preserve the supplied project or target fingerprint, hatmax_version, and book_version exactly in an intent.\n")
	prompt.WriteString("For create_application, use intent schema version 3, operation create_application, archetype server_rendered_hatmax_application, and source_fingerprint from target context.\n")
	prompt.WriteString("For create_application, set project_fingerprint and feature to empty strings, domain values to null, capabilities to an empty array, documentation to not_requested, and application target base to session_directory.\n")
	prompt.WriteString("For create_application, derive project_slug and target.directory as lower-kebab words from application.display_name. Use target.remote_module_path when non-empty; otherwise ask only for application.module_path after the application name is known.\n")
	prompt.WriteString("Application description, niche, and initial features are optional and must not trigger clarification. PostgreSQL is implicit in the application archetype and must not be requested from the user.\n")
	prompt.WriteString("Represent each explicitly requested initial feature with a semantic feature name and create-feature domain. Required fields use required true; Hatmax derives runtime validation and all canonical feature capabilities.\n")
	prompt.WriteString("Return semantic feature and field names without translating them into Go or URL syntax. Hatmax derives canonical identifiers, routes, plurals, and default labels.\n")
	prompt.WriteString("For create_feature, set entity, route, and labels to null unless the user explicitly chose a distinct product name, route, or display label.\n")
	prompt.WriteString("For create_feature, use domain.fields and set domain.field and domain.validation to null. A required field uses its required boolean, not domain.validation.\n")
	prompt.WriteString("For add_field, use domain.field and set domain.fields and domain.validation to null.\n")
	prompt.WriteString("For add_validation, use domain.validation and set domain.fields and domain.field to null.\n")
	prompt.WriteString("Set every inapplicable optional domain property to null.\n")
	prompt.WriteString("Documentation remains not_requested unless the request explicitly asks for documentation.\n")
	prompt.WriteString("For documentation-only requests, use document_feature with document_existing_behavior and leave domain values null.\n")
	prompt.WriteString("For requests combining implementation and documentation, keep the implementation operation and use document_planned_change.\n")
	prompt.WriteString("Classify each explicit reader need as tutorial, how_to, reference, or explanation and return only subject and reader_goal.\n")
	prompt.WriteString("If the request asks to document behavior but the reader need is ambiguous, ask one focused clarification instead of selecting every quadrant.\n")
	prompt.WriteString("Do not produce a plan, file path, edit, command, dependency, approval decision, prose, or Markdown.\n")
	prompt.WriteString("Do not use tools or external context. Return only the JSON value constrained by the supplied schema.\n")
	prompt.WriteString("HATMAX_REQUEST_JSON\n")
	prompt.Write(request)
	prompt.WriteByte('\n')
	prompt.WriteString("END_HATMAX_REQUEST_JSON")

	return prompt.String()
}

type turnStartParameters struct {
	ThreadID       string         `json:"threadId"`
	Input          []turnInput    `json:"input"`
	OutputSchema   map[string]any `json:"outputSchema"`
	ApprovalPolicy string         `json:"approvalPolicy"`
	TurnTrigger    string         `json:"turnTrigger"`
}

type turnInput struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (turn CompiledTurn) String() string {
	return fmt.Sprintf("Codex interpretation turn for %s", turn.ThreadID)
}
