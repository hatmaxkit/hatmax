// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

// Package hatmaxcli implements the line-oriented Hatmax command surface.
package hatmaxcli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"hatmax.adrianpk.com/generator/conversation"
	"hatmax.adrianpk.com/generator/interaction"
)

const (
	// ExitSuccess reports completed generation.
	ExitSuccess = 0
	// ExitFailure reports setup or unclassified interaction failure.
	ExitFailure = 1
	// ExitUsage reports invalid command arguments.
	ExitUsage = 2
	// ExitCancelled reports explicit or input-driven cancellation.
	ExitCancelled = 3
	// ExitClarificationRequired reports unresolved product decisions.
	ExitClarificationRequired = 4
	// ExitUnsupported reports a request outside the Hatmax Book.
	ExitUnsupported = 5
	// ExitIntentRejected reports deterministic intent rejection.
	ExitIntentRejected = 6
	// ExitPlanStale reports approval invalidated by project drift.
	ExitPlanStale = 7
	// ExitExecutionFailed reports renderer, mutation, or validation failure.
	ExitExecutionFailed = 8
)

// Runner is the complete interaction boundary required by the terminal app.
type Runner interface {
	Run(context.Context, string, string) interaction.Result
}

// CoordinatorFactory assembles one runner for a canonical project root and
// the terminal-owned interaction ports.
type CoordinatorFactory func(string, interaction.Approver, interaction.Clarifier) (Runner, error)

// ConversationManager controls user-local conversation selection without
// granting project mutation authority.
type ConversationManager interface {
	OpenConversation(context.Context, string, conversation.SessionOptions) (ConversationSession, error)
	List(context.Context, string) ([]conversation.Summary, error)
}

// ConversationSession is the read-only lifecycle needed by headless
// conversation selection commands.
type ConversationSession interface {
	Current() conversation.Conversation
	Close() error
}

// ConversationManagerFactory assembles local conversation state for one root.
type ConversationManagerFactory func(string) (ConversationManager, error)

// Config supplies process IO and assembly boundaries.
type Config struct {
	Input               io.Reader
	Output              io.Writer
	ErrorOutput         io.Writer
	WorkingDirectory    func() (string, error)
	CoordinatorFactory  CoordinatorFactory
	ConversationFactory ConversationManagerFactory
	CommandName         string
}

// App parses commands and binds terminal IO to the interaction coordinator.
type App struct {
	input               io.Reader
	output              io.Writer
	errorOutput         io.Writer
	workingDirectory    func() (string, error)
	coordinatorFactory  CoordinatorFactory
	conversationFactory ConversationManagerFactory
	commandName         string
}

// New constructs a terminal app without inspecting a project or connecting
// to an interpreter backend.
func New(config Config) (*App, error) {
	if config.Input == nil || config.Output == nil || config.ErrorOutput == nil {
		return nil, errors.New("terminal input, output, and error output are required")
	}

	if config.WorkingDirectory == nil || config.CoordinatorFactory == nil {
		return nil, errors.New("working-directory and coordinator factories are required")
	}

	commandName := strings.TrimSpace(config.CommandName)
	if commandName == "" {
		commandName = "hatmax"
	}

	return &App{
		input:               config.Input,
		output:              config.Output,
		errorOutput:         config.ErrorOutput,
		workingDirectory:    config.WorkingDirectory,
		coordinatorFactory:  config.CoordinatorFactory,
		conversationFactory: config.ConversationFactory,
		commandName:         commandName,
	}, nil
}

// Run executes one command and returns a stable process exit status.
func (app *App) Run(ctx context.Context, arguments []string) int {
	command, valid := parseCommand(arguments)
	if !valid {
		app.writeUsage()

		return ExitUsage
	}

	root, err := app.workingDirectory()
	if err != nil {
		_, _ = fmt.Fprintf(app.errorOutput, "hatmax: determine project directory: %v\n", err)

		return ExitFailure
	}

	if command.kind != commandGenerate {
		return app.runConversation(ctx, root, command)
	}

	terminal := &terminalInteraction{
		reader: bufio.NewReader(app.input),
		output: app.output,
	}

	runner, err := app.coordinatorFactory(root, terminal, terminal)
	if err != nil {
		_, _ = fmt.Fprintf(app.errorOutput, "hatmax: initialize generator: %v\n", err)

		return ExitFailure
	}

	result := runner.Run(ctx, root, command.prompt)

	err = writeResult(app.output, result)
	if err != nil {
		_, _ = fmt.Fprintf(app.errorOutput, "hatmax: write result: %v\n", err)

		return ExitFailure
	}

	return exitStatus(result.Outcome)
}

type commandKind string

const (
	commandGenerate           commandKind = "generate"
	commandConversationNew    commandKind = "conversation_new"
	commandConversationList   commandKind = "conversation_list"
	commandConversationResume commandKind = "conversation_resume"
)

type parsedCommand struct {
	kind           commandKind
	prompt         string
	conversationID string
}

func parseCommand(arguments []string) (parsedCommand, bool) {
	if prompt, valid := parseGenerate(arguments); valid {
		return parsedCommand{kind: commandGenerate, prompt: prompt}, true
	}

	if len(arguments) == 2 && arguments[0] == "conversation" {
		switch arguments[1] {
		case "new":
			return parsedCommand{kind: commandConversationNew}, true
		case "list":
			return parsedCommand{kind: commandConversationList}, true
		}
	}

	if len(arguments) == 3 && arguments[0] == "conversation" && arguments[1] == "resume" {
		id := strings.TrimSpace(arguments[2])
		if id != "" {
			return parsedCommand{kind: commandConversationResume, conversationID: id}, true
		}
	}

	return parsedCommand{}, false
}

func (app *App) runConversation(ctx context.Context, root string, command parsedCommand) int {
	if app.conversationFactory == nil {
		_, _ = fmt.Fprintf(app.errorOutput, "%s: conversation commands are unavailable\n", app.commandName)

		return ExitFailure
	}

	manager, err := app.conversationFactory(root)
	if err != nil {
		_, _ = fmt.Fprintf(app.errorOutput, "%s: initialize conversation: %v\n", app.commandName, err)

		return ExitFailure
	}

	if command.kind == commandConversationList {
		return app.listConversations(ctx, manager, root)
	}

	options := conversation.SessionOptions{Fresh: command.kind == commandConversationNew}
	if command.kind == commandConversationResume {
		options.ConversationID = command.conversationID
	}

	session, err := manager.OpenConversation(ctx, root, options)
	if err != nil {
		_, _ = fmt.Fprintf(app.errorOutput, "%s: open conversation: %v\n", app.commandName, err)

		return ExitFailure
	}
	defer session.Close()

	value := session.Current()

	_, err = fmt.Fprintf(app.output, "Conversation: %s\nStatus: %s\n", value.ID, value.Status)
	if err != nil {
		_, _ = fmt.Fprintf(app.errorOutput, "%s: write conversation: %v\n", app.commandName, err)

		return ExitFailure
	}

	return ExitSuccess
}

func (app *App) listConversations(ctx context.Context, manager ConversationManager, root string) int {
	values, err := manager.List(ctx, root)
	if err != nil {
		_, _ = fmt.Fprintf(app.errorOutput, "%s: list conversations: %v\n", app.commandName, err)

		return ExitFailure
	}

	if len(values) == 0 {
		_, err = fmt.Fprintln(app.output, "No conversations.")
	} else {
		for _, value := range values {
			_, err = fmt.Fprintf(
				app.output,
				"%s\t%s\t%s\n",
				value.ID,
				value.Status,
				value.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			)
			if err != nil {
				break
			}
		}
	}

	if err != nil {
		_, _ = fmt.Fprintf(app.errorOutput, "%s: write conversations: %v\n", app.commandName, err)

		return ExitFailure
	}

	return ExitSuccess
}

func (app *App) writeUsage() {
	_, _ = fmt.Fprintf(
		app.errorOutput,
		"usage:\n  %s generate \"<request>\"\n  %s conversation new\n  %s conversation list\n  %s conversation resume <conversation-id>\n",
		app.commandName,
		app.commandName,
		app.commandName,
		app.commandName,
	)
}

func parseGenerate(arguments []string) (string, bool) {
	if len(arguments) != 2 || arguments[0] != "generate" {
		return "", false
	}

	prompt := strings.TrimSpace(arguments[1])

	return prompt, prompt != ""
}

func exitStatus(outcome interaction.Outcome) int {
	switch outcome {
	case interaction.OutcomeCompleted:
		return ExitSuccess
	case interaction.OutcomeCancelled:
		return ExitCancelled
	case interaction.OutcomeClarificationRequired:
		return ExitClarificationRequired
	case interaction.OutcomeUnsupported:
		return ExitUnsupported
	case interaction.OutcomeIntentRejected:
		return ExitIntentRejected
	case interaction.OutcomePlanStale:
		return ExitPlanStale
	case interaction.OutcomeExecutionFailed:
		return ExitExecutionFailed
	default:
		return ExitFailure
	}
}

type terminalInteraction struct {
	reader *bufio.Reader
	output io.Writer
}

func (terminal *terminalInteraction) Approve(
	_ context.Context,
	request interaction.ApprovalRequest,
) (interaction.ApprovalDecision, error) {
	_, err := fmt.Fprintf(
		terminal.output,
		"Plan:\n%sApprove plan %s? [y/N]: ",
		planPresentation(request.PlanYAML),
		request.PlanDigest,
	)
	if err != nil {
		return "", err
	}

	answer, available, err := terminal.readLine()
	if err != nil {
		return "", err
	}

	if !available {
		return interaction.ApprovalRejected, nil
	}

	normalized := strings.ToLower(strings.TrimSpace(answer))
	if normalized == "y" || normalized == "yes" {
		return interaction.ApprovalGranted, nil
	}

	return interaction.ApprovalRejected, nil
}

func (terminal *terminalInteraction) Clarify(
	_ context.Context,
	request interaction.ClarificationRequest,
) (interaction.ClarificationResponse, error) {
	answers := make([]interaction.ClarificationAnswer, 0, len(request.Questions))

	for _, question := range request.Questions {
		_, err := fmt.Fprintf(terminal.output, "%s [%s]: ", question.Question, question.Field)
		if err != nil {
			return interaction.ClarificationResponse{}, err
		}

		answer, available, err := terminal.readLine()
		if err != nil {
			return interaction.ClarificationResponse{}, err
		}

		answer = strings.TrimSpace(answer)
		if !available || answer == "" {
			return interaction.ClarificationResponse{Cancelled: true}, nil
		}

		answers = append(answers, interaction.ClarificationAnswer{
			Field:  question.Field,
			Answer: answer,
		})
	}

	return interaction.ClarificationResponse{Answers: answers}, nil
}

func (terminal *terminalInteraction) readLine() (string, bool, error) {
	line, err := terminal.reader.ReadString('\n')
	if err == nil {
		return line, true, nil
	}

	if errors.Is(err, io.EOF) {
		return line, line != "", nil
	}

	return "", false, err
}

func planPresentation(value []byte) string {
	if len(value) == 0 {
		return "\n"
	}

	text := string(value)
	if strings.HasSuffix(text, "\n") {
		return text
	}

	return text + "\n"
}
