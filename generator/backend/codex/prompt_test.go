package codex

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"hatmax.adrianpk.com/generator/eval"
)

func TestCompileTurnContainsOnlyBoundedRequestAndSchema(t *testing.T) {
	// The compiled turn carries the bounded request, output schema, and no model
	// override or executable instruction.
	request := promptTestRequest()

	turn, err := CompileTurn("thread-1", request)
	if err != nil {
		t.Fatalf("CompileTurn() error = %v", err)
	}

	if !strings.Contains(turn.Prompt, `"prompt":"Add an invoice feature"`) {
		t.Fatalf("prompt does not contain request: %q", turn.Prompt)
	}

	for _, forbidden := range []string{"/target/project", "git commit", "shell_command", "file_path"} {
		if strings.Contains(turn.Prompt, forbidden) {
			t.Fatalf("prompt contains forbidden value %q", forbidden)
		}
	}

	encoded, err := json.Marshal(turn.parameters())
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	if strings.Contains(string(encoded), `"model"`) || !strings.Contains(string(encoded), `"approvalPolicy":"never"`) {
		t.Fatalf("turn parameters = %s", encoded)
	}

	if turn.Schema["type"] != "object" {
		t.Fatalf("schema type = %#v", turn.Schema["type"])
	}
}

func TestCompileTurnIncludesClarificationHistory(t *testing.T) {
	// Explicit answers are authoritative and remain inside the bounded request.
	request := promptTestRequest()
	request.Clarifications = []eval.ClarificationExchange{{
		Field:    "feature",
		Question: "Which feature?",
		Answer:   "invoice",
	}}

	turn, err := CompileTurn("thread-1", request)
	if err != nil {
		t.Fatalf("CompileTurn() error = %v", err)
	}

	if !strings.Contains(turn.Prompt, `"answer":"invoice"`) {
		t.Fatalf("prompt does not contain clarification: %q", turn.Prompt)
	}
}

func TestCompileTurnRejectsInvalidBoundary(t *testing.T) {
	// Thread and contract identity are mandatory before model invocation.
	request := promptTestRequest()
	request.ContractVersion++

	_, err := CompileTurn("thread-1", request)
	assertBackendCode(t, err, eval.BackendOutputInvalid)

	request.ContractVersion = eval.CurrentContractVersion
	_, err = CompileTurn("", request)
	assertBackendCode(t, err, eval.BackendThreadFailed)
}

func TestDefaultTurnTimeoutIsTwoMinutes(t *testing.T) {
	// The product default is stable and visible to callers.
	if DefaultTurnTimeout != 2*time.Minute {
		t.Fatalf("DefaultTurnTimeout = %s", DefaultTurnTimeout)
	}
}

func promptTestRequest() eval.Request {
	return eval.Request{
		ContractVersion: eval.CurrentContractVersion,
		Prompt:          "Add an invoice feature",
		Project: eval.ProjectContext{
			Fingerprint:   "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			HatmaxVersion: "0.4.0",
		},
		Book: eval.BookContext{
			Version: 1,
			Archetypes: []eval.ArchetypeContext{{
				ID:                   "server_rendered_crud",
				Operations:           []string{"create_feature"},
				RequiredCapabilities: []string{"postgres_persistence", "htmx_form"},
			}},
		},
	}
}
