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

	contextRoot := t.TempDir()

	interpreter, err := NewLocalInterpreter(InterpreterConfig{
		ContextRoot:     contextRoot,
		ProjectIdentity: "hatmax-authenticated-live-smoke-v1",
	})
	if err != nil {
		t.Fatalf("NewLocalInterpreter() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	first, err := interpreter.Interpret(ctx, liveSmokeRequest())
	if err != nil {
		t.Fatalf("initial authenticated Codex smoke failed: %v", err)
	}

	if first.Interpretation.Kind == "" || first.Provenance.EffectiveModel == "" || first.Provenance.ThreadReused {
		t.Fatalf("initial authenticated Codex smoke returned invalid bounded provenance: %#v", first.Provenance)
	}

	second, err := interpreter.Interpret(ctx, liveSmokeRequest())
	if err != nil {
		t.Fatalf("same-project authenticated Codex smoke failed: %v", err)
	}

	if !second.Provenance.RuntimeReused || !second.Provenance.ThreadReused {
		t.Fatalf("same-project request did not reuse runtime and thread: %#v", second.Provenance)
	}

	isolated, err := NewLocalInterpreter(InterpreterConfig{
		ContextRoot:     contextRoot,
		ProjectIdentity: "hatmax-authenticated-live-smoke-isolated-v1",
	})
	if err != nil {
		t.Fatalf("NewLocalInterpreter() for isolated project error = %v", err)
	}

	third, err := isolated.Interpret(ctx, liveSmokeRequest())
	if err != nil {
		t.Fatalf("different-project authenticated Codex smoke failed: %v", err)
	}

	if !third.Provenance.RuntimeReused || third.Provenance.ThreadReused {
		t.Fatalf("different-project request did not reuse only the runtime: %#v", third.Provenance)
	}

	t.Logf("authenticated Codex smoke passed: kind=%s model_selection=%s runtime_reused=%t same_project_thread_reused=%t different_project_thread_reused=%t timing=%s",
		second.Interpretation.Kind,
		second.Provenance.ModelSelection,
		second.Provenance.RuntimeReused,
		second.Provenance.ThreadReused,
		third.Provenance.ThreadReused,
		second.Provenance.Timing,
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
