// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package hatmaxcli

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"hatmax.adrianpk.com/generator/conversation"
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

		expectedUsage := "usage:\n" +
			"  hatmax generate \"<request>\"\n" +
			"  hatmax conversation new\n" +
			"  hatmax conversation list\n" +
			"  hatmax conversation resume <conversation-id>\n"
		if errorOutput.String() != expectedUsage || output.Len() != 0 {
			t.Errorf("case %d output = %q / %q, want usage on stderr", index, output.String(), errorOutput.String())
		}
	}
}

func TestConversationCommandsControlOnlyLocalSelection(t *testing.T) {
	now := time.Date(2026, time.September, 29, 21, 0, 0, 0, time.UTC)
	manager := &fakeConversationManager{
		session: &fakeConversationSession{value: conversation.Conversation{
			ID: "conversation-new", Status: conversation.StatusActive,
		}},
		summaries: []conversation.Summary{{
			ID: "conversation-new", Status: conversation.StatusActive, UpdatedAt: now,
		}},
	}

	tests := []struct {
		arguments []string
		contains  string
		assert    func(*testing.T, conversation.SessionOptions)
	}{
		{
			arguments: []string{"conversation", "new"},
			contains:  "Conversation: conversation-new\nStatus: active\n",
			assert: func(t *testing.T, options conversation.SessionOptions) {
				t.Helper()

				if !options.Fresh || options.ConversationID != "" {
					t.Fatalf("new options = %#v", options)
				}
			},
		},
		{
			arguments: []string{"conversation", "resume", "conversation-old"},
			contains:  "Conversation: conversation-new\nStatus: active\n",
			assert: func(t *testing.T, options conversation.SessionOptions) {
				t.Helper()

				if options.Fresh || options.ConversationID != "conversation-old" {
					t.Fatalf("resume options = %#v", options)
				}
			},
		},
	}

	for _, testCase := range tests {
		var output, errorOutput bytes.Buffer

		app := newConversationTestApp(t, &output, &errorOutput, manager)

		if exit := app.Run(context.Background(), testCase.arguments); exit != ExitSuccess {
			t.Fatalf("Run(%v) exit = %d", testCase.arguments, exit)
		}

		if output.String() != testCase.contains || errorOutput.Len() != 0 {
			t.Fatalf("Run(%v) output = %q / %q", testCase.arguments, output.String(), errorOutput.String())
		}

		testCase.assert(t, manager.openOptions)
	}

	var output, errorOutput bytes.Buffer

	app := newConversationTestApp(t, &output, &errorOutput, manager)
	if exit := app.Run(context.Background(), []string{"conversation", "list"}); exit != ExitSuccess {
		t.Fatalf("conversation list exit = %d", exit)
	}

	if output.String() != "conversation-new\tactive\t2026-09-29T21:00:00Z\n" || errorOutput.Len() != 0 {
		t.Fatalf("conversation list output = %q / %q", output.String(), errorOutput.String())
	}
}

func TestHmAndHatmaxGenerateUseTheSameHeadlessKernel(t *testing.T) {
	factory := func(string, interaction.Approver, interaction.Clarifier) (Runner, error) {
		return runnerFunc(func(context.Context, string, string) interaction.Result {
			return interaction.Result{State: interaction.StateFinished, Outcome: interaction.OutcomeCompleted}
		}), nil
	}

	run := func(commandName string) (int, string, string) {
		var output, errorOutput bytes.Buffer

		app, err := New(Config{
			Input: strings.NewReader(""), Output: &output, ErrorOutput: &errorOutput,
			WorkingDirectory:   func() (string, error) { return "/project", nil },
			CoordinatorFactory: factory, CommandName: commandName,
		})
		if err != nil {
			t.Fatalf("New(%s) error = %v", commandName, err)
		}

		return app.Run(context.Background(), []string{"generate", "Add invoices."}), output.String(), errorOutput.String()
	}

	hmExit, hmOutput, hmError := run("hm")

	hatmaxExit, hatmaxOutput, hatmaxError := run("hatmax")
	if hmExit != hatmaxExit || hmOutput != hatmaxOutput || hmError != hatmaxError {
		t.Fatalf(
			"hm = %d %q %q; hatmax = %d %q %q",
			hmExit, hmOutput, hmError, hatmaxExit, hatmaxOutput, hatmaxError,
		)
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

func newConversationTestApp(
	t *testing.T,
	output *bytes.Buffer,
	errorOutput *bytes.Buffer,
	manager ConversationManager,
) *App {
	t.Helper()

	app, err := New(Config{
		Input: strings.NewReader(""), Output: output, ErrorOutput: errorOutput,
		WorkingDirectory: func() (string, error) { return "/project", nil },
		CoordinatorFactory: func(string, interaction.Approver, interaction.Clarifier) (Runner, error) {
			return nil, errors.New("generate coordinator must not be assembled")
		},
		ConversationFactory: func(string) (ConversationManager, error) { return manager, nil },
		CommandName:         "hm",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return app
}

type fakeConversationManager struct {
	session     ConversationSession
	summaries   []conversation.Summary
	openOptions conversation.SessionOptions
}

func (manager *fakeConversationManager) OpenConversation(
	_ context.Context,
	_ string,
	options conversation.SessionOptions,
) (ConversationSession, error) {
	manager.openOptions = options

	return manager.session, nil
}

func (manager *fakeConversationManager) List(
	context.Context,
	string,
) ([]conversation.Summary, error) {
	return append([]conversation.Summary{}, manager.summaries...), nil
}

type fakeConversationSession struct {
	value conversation.Conversation
}

func (session *fakeConversationSession) Current() conversation.Conversation {
	return session.value
}

func (session *fakeConversationSession) Close() error {
	return nil
}

func bufioReader(value string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(value))
}
