package hatmaxcli

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/interaction"
)

func TestAppRunsGenerateWithClarificationAndExplicitApproval(t *testing.T) {
	input := strings.NewReader("/invoices\nyes\n")

	var (
		output      bytes.Buffer
		errorOutput bytes.Buffer
	)

	factory := func(root string, approver interaction.Approver, clarifier interaction.Clarifier) (Runner, error) {
		if root != "/project" {
			t.Fatalf("factory root = %q, want /project", root)
		}

		return runnerFunc(func(ctx context.Context, runRoot, prompt string) interaction.Result {
			if runRoot != root || prompt != "Add invoices." {
				t.Fatalf("Run() inputs = %q, %q, want factory root and exact prompt", runRoot, prompt)
			}

			clarification, err := clarifier.Clarify(ctx, interaction.ClarificationRequest{
				Round: 1,
				Questions: []intent.Clarification{{
					Field:    "domain.route",
					Question: "Which route should expose invoices?",
				}},
			})
			if err != nil || clarification.Cancelled || len(clarification.Answers) != 1 || clarification.Answers[0].Answer != "/invoices" {
				t.Fatalf("Clarify() = %#v, %v, want explicit route", clarification, err)
			}

			decision, err := approver.Approve(ctx, interaction.ApprovalRequest{
				PlanDigest:         "sha256:plan",
				ProjectFingerprint: "sha256:project",
				PlanYAML:           []byte("intent: create_feature\ndigest: sha256:plan\n"),
			})
			if err != nil || decision != interaction.ApprovalGranted {
				t.Fatalf("Approve() = %q, %v, want approved", decision, err)
			}

			return interaction.Result{State: interaction.StateFinished, Outcome: interaction.OutcomeCompleted}
		}), nil
	}

	app := newTestApp(t, input, &output, &errorOutput, factory)
	exitCode := app.Run(context.Background(), []string{"generate", "Add invoices."})

	if exitCode != ExitSuccess {
		t.Fatalf("Run() exit = %d, want %d", exitCode, ExitSuccess)
	}

	text := output.String()
	for _, expected := range []string{
		"Which route should expose invoices? [domain.route]: ",
		"Plan:\nintent: create_feature\ndigest: sha256:plan\n",
		"Approve plan sha256:plan? [y/N]: ",
		"Outcome: completed\n",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("output does not contain %q:\n%s", expected, text)
		}
	}

	if errorOutput.Len() != 0 {
		t.Errorf("error output = %q, want empty", errorOutput.String())
	}
}

func TestAppRejectsInvalidCommandWithoutAssembly(t *testing.T) {
	tests := [][]string{
		nil,
		{"generate"},
		{"generate", ""},
		{"unknown", "request"},
		{"generate", "one", "two"},
	}

	for index, arguments := range tests {
		var (
			output      bytes.Buffer
			errorOutput bytes.Buffer
		)

		assembled := false
		app := newTestApp(t, strings.NewReader(""), &output, &errorOutput, func(string, interaction.Approver, interaction.Clarifier) (Runner, error) {
			assembled = true

			return nil, errors.New("unexpected assembly")
		})

		exitCode := app.Run(context.Background(), arguments)
		if exitCode != ExitUsage || assembled {
			t.Errorf("case %d = exit %d, assembled %t; want usage without assembly", index, exitCode, assembled)
		}

		if errorOutput.String() != "usage: hatmax generate \"<request>\"\n" || output.Len() != 0 {
			t.Errorf("case %d output = %q / %q, want usage on stderr", index, output.String(), errorOutput.String())
		}
	}
}

func TestTerminalApprovalDefaultsToRejection(t *testing.T) {
	tests := []string{"", "\n", "no\n", "approve\n"}

	for _, input := range tests {
		var output bytes.Buffer

		terminal := &terminalInteraction{reader: bufioReader(input), output: &output}

		decision, err := terminal.Approve(context.Background(), interaction.ApprovalRequest{
			PlanDigest: "sha256:plan",
			PlanYAML:   []byte("intent: create_feature\n"),
		})
		if err != nil {
			t.Fatalf("Approve(%q) error = %v", input, err)
		}

		if decision != interaction.ApprovalRejected {
			t.Errorf("Approve(%q) = %q, want rejected", input, decision)
		}
	}
}

func TestExitStatusMapsStableOutcomes(t *testing.T) {
	tests := []struct {
		outcome interaction.Outcome
		want    int
	}{
		{outcome: interaction.OutcomeCompleted, want: ExitSuccess},
		{outcome: interaction.OutcomeCancelled, want: ExitCancelled},
		{outcome: interaction.OutcomeClarificationRequired, want: ExitClarificationRequired},
		{outcome: interaction.OutcomeUnsupported, want: ExitUnsupported},
		{outcome: interaction.OutcomeIntentRejected, want: ExitIntentRejected},
		{outcome: interaction.OutcomePlanStale, want: ExitPlanStale},
		{outcome: interaction.OutcomeExecutionFailed, want: ExitExecutionFailed},
		{outcome: interaction.OutcomeFailed, want: ExitFailure},
	}

	for _, testCase := range tests {
		if result := exitStatus(testCase.outcome); result != testCase.want {
			t.Errorf("exitStatus(%q) = %d, want %d", testCase.outcome, result, testCase.want)
		}
	}
}

type runnerFunc func(context.Context, string, string) interaction.Result

func (run runnerFunc) Run(ctx context.Context, root, prompt string) interaction.Result {
	return run(ctx, root, prompt)
}

func newTestApp(
	t *testing.T,
	input *strings.Reader,
	output *bytes.Buffer,
	errorOutput *bytes.Buffer,
	factory CoordinatorFactory,
) *App {
	t.Helper()

	app, err := New(Config{
		Input:              input,
		Output:             output,
		ErrorOutput:        errorOutput,
		WorkingDirectory:   func() (string, error) { return "/project", nil },
		CoordinatorFactory: factory,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return app
}

func bufioReader(value string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(value))
}
