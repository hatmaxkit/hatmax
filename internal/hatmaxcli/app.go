// Package hatmaxcli implements the line-oriented Hatmax command surface.
package hatmaxcli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

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

// Config supplies process IO and assembly boundaries.
type Config struct {
	Input              io.Reader
	Output             io.Writer
	ErrorOutput        io.Writer
	WorkingDirectory   func() (string, error)
	CoordinatorFactory CoordinatorFactory
}

// App parses commands and binds terminal IO to the interaction coordinator.
type App struct {
	input              io.Reader
	output             io.Writer
	errorOutput        io.Writer
	workingDirectory   func() (string, error)
	coordinatorFactory CoordinatorFactory
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

	return &App{
		input:              config.Input,
		output:             config.Output,
		errorOutput:        config.ErrorOutput,
		workingDirectory:   config.WorkingDirectory,
		coordinatorFactory: config.CoordinatorFactory,
	}, nil
}

// Run executes one command and returns a stable process exit status.
func (app *App) Run(ctx context.Context, arguments []string) int {
	prompt, valid := parseGenerate(arguments)
	if !valid {
		_, _ = fmt.Fprintln(app.errorOutput, `usage: hatmax generate "<request>"`)

		return ExitUsage
	}

	root, err := app.workingDirectory()
	if err != nil {
		_, _ = fmt.Fprintf(app.errorOutput, "hatmax: determine project directory: %v\n", err)

		return ExitFailure
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

	result := runner.Run(ctx, root, prompt)

	err = writeBasicResult(app.output, result)
	if err != nil {
		_, _ = fmt.Fprintf(app.errorOutput, "hatmax: write result: %v\n", err)

		return ExitFailure
	}

	return exitStatus(result.Outcome)
}

func parseGenerate(arguments []string) (string, bool) {
	if len(arguments) != 2 || arguments[0] != "generate" {
		return "", false
	}

	prompt := strings.TrimSpace(arguments[1])

	return prompt, prompt != ""
}

func writeBasicResult(output io.Writer, result interaction.Result) error {
	_, err := fmt.Fprintf(output, "Outcome: %s\n", result.Outcome)

	return err
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
