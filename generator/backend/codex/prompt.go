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
	prompt.WriteString("Use only operations, archetypes, capabilities, and variants present in the request.\n")
	prompt.WriteString("Preserve project_fingerprint, hatmax_version, and book_version exactly in an intent.\n")
	prompt.WriteString("Documentation remains not_requested unless the request explicitly asks for documentation.\n")
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
