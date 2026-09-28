package codex

import (
	"context"
	"os"
	"testing"
	"time"

	"hatmax.adrianpk.com/generator/eval"
)

func TestAuthenticatedLiveSmoke(t *testing.T) {
	// This opt-in observation exercises saved ChatGPT authentication, the
	// resident daemon, isolated thread creation, and one structured turn. It
	// never prints account data, raw events, reasoning, or model output.
	if os.Getenv("HATMAX_CODEX_LIVE_SMOKE") != "1" {
		t.Skip("set HATMAX_CODEX_LIVE_SMOKE=1 to run the authenticated Codex smoke test")
	}

	interpreter, err := NewLocalInterpreter(InterpreterConfig{
		ContextRoot:     t.TempDir(),
		ProjectIdentity: "hatmax-authenticated-live-smoke-v1",
	})
	if err != nil {
		t.Fatalf("NewLocalInterpreter() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	result, err := interpreter.Interpret(ctx, liveSmokeRequest())
	if err != nil {
		t.Fatalf("authenticated Codex smoke failed: %v", err)
	}

	if result.Interpretation.Kind == "" || result.Provenance.EffectiveModel == "" {
		t.Fatal("authenticated Codex smoke returned incomplete bounded provenance")
	}

	t.Logf("authenticated Codex smoke passed: kind=%s model_selection=%s runtime_reused=%t thread_reused=%t timing=%s",
		result.Interpretation.Kind,
		result.Provenance.ModelSelection,
		result.Provenance.RuntimeReused,
		result.Provenance.ThreadReused,
		result.Provenance.Timing,
	)
}

func liveSmokeRequest() eval.Request {
	return eval.Request{
		ContractVersion: eval.CurrentContractVersion,
		Prompt:          "Create an invoice feature with a required reference string field",
		Project: eval.ProjectContext{
			Fingerprint:   "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			HatmaxVersion: "0.4.0",
		},
		Book: eval.BookContext{
			Version: 1,
			Archetypes: []eval.ArchetypeContext{{
				ID:                   "server_rendered_crud",
				Operations:           []string{"create_feature", "add_field", "add_validation"},
				RequiredCapabilities: []string{"postgres_persistence", "htmx_form", "runtime_validation"},
			}},
			Capabilities: []eval.CapabilityContext{
				{ID: "postgres_persistence", Intent: "Persist the feature in PostgreSQL"},
				{ID: "htmx_form", Intent: "Render server-owned HTMX forms"},
				{ID: "runtime_validation", Intent: "Validate input at runtime"},
			},
		},
	}
}
