// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package interaction

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/eval"
	"hatmax.adrianpk.com/generator/execute"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

// Config supplies the backend and user-interaction ports required by a
// Coordinator. Book defaults to the embedded canonical Book.
type Config struct {
	Interpreter       eval.Interpreter
	Approver          Approver
	Clarifier         Clarifier
	Book              *book.Book
	InspectionOptions project.Options
}

// Coordinator owns project inspection, deterministic planning, approval, and
// execution coordination around one backend-neutral interpreter.
type Coordinator struct {
	interpreter       eval.Interpreter
	approver          Approver
	clarifier         Clarifier
	selectedBook      *book.Book
	applicationBook   *book.Book
	inspectionOptions project.Options
}

type preparedInteraction struct {
	inventory project.Inventory
	target    *project.TargetInventory
	book      *book.Book
	plan      plan.Plan
	planYAML  []byte
	result    Result
}

type preparationOptions struct {
	dialogue           []eval.DialogueTurn
	clarifications     []eval.ClarificationExchange
	preview            bool
	approvedPlanDigest string
}

// New constructs a coordinator without connecting to an interpreter or
// inspecting a project.
func New(config Config) (*Coordinator, error) {
	if config.Interpreter == nil {
		return nil, errors.New("interaction interpreter is required")
	}

	if config.Approver == nil {
		return nil, errors.New("interaction approver is required")
	}

	selectedBook := config.Book
	if selectedBook == nil {
		var err error

		selectedBook, err = book.LoadDefault()
		if err != nil {
			return nil, fmt.Errorf("load default Hatmax Book: %w", err)
		}
	}

	applicationBook, err := book.LoadRelease(2)
	if err != nil {
		return nil, fmt.Errorf("load application Hatmax Book: %w", err)
	}

	if config.Book != nil && config.Book.Manifest().BookVersion == 2 {
		applicationBook = config.Book
	}

	return &Coordinator{
		interpreter:       config.Interpreter,
		approver:          config.Approver,
		clarifier:         config.Clarifier,
		selectedBook:      selectedBook,
		applicationBook:   applicationBook,
		inspectionOptions: config.InspectionOptions,
	}, nil
}

// ProjectFingerprint reinspects one existing project using the same complete
// Book projection used before conversational planning.
func (c *Coordinator) ProjectFingerprint(ctx context.Context, root string) (project.Fingerprint, error) {
	inventory, err := project.InspectWithOptions(ctx, root, c.inspectionOptions)
	if err != nil {
		return project.Fingerprint{}, err
	}

	return inventory.Fingerprint(bookFingerprintRequest(c.selectedBook))
}

func (c *Coordinator) prepareApproved(ctx context.Context, root, prompt string) (preparedInteraction, *Result) {
	return c.prepare(ctx, root, prompt, preparationOptions{})
}

func (c *Coordinator) prepare(
	ctx context.Context,
	root string,
	prompt string,
	options preparationOptions,
) (preparedInteraction, *Result) {
	baseResult := Result{
		State:       StateInspecting,
		Transitions: []State{StateInspecting},
		Diagnostics: []Diagnostic{},
	}

	inventory, err := project.InspectWithOptions(ctx, root, c.inspectionOptions)
	if err != nil {
		_, moduleErr := os.Stat(filepath.Join(root, "go.mod"))
		if errors.Is(moduleErr, os.ErrNotExist) {
			return c.prepareApplication(ctx, root, prompt, baseResult, options)
		}

		return preparedInteraction{}, terminalFailure(baseResult, OutcomeFailed, PhaseInspection, "HMGEN-INSPECTION-FAILED", err)
	}

	fingerprintRequest := bookFingerprintRequest(c.selectedBook)

	fingerprint, err := inventory.Fingerprint(fingerprintRequest)
	if err != nil {
		return preparedInteraction{}, terminalFailure(baseResult, OutcomeFailed, PhaseInspection, "HMGEN-FINGERPRINT-FAILED", err)
	}

	baseResult.State = StateInterpreting
	baseResult.Transitions = append(baseResult.Transitions, StateInterpreting)
	clarifications := append([]eval.ClarificationExchange{}, options.clarifications...)

	for round := 0; ; round++ {
		evaluation, evaluationErr := eval.EvaluateDialogue(ctx, c.interpreter, prompt, options.dialogue, clarifications, eval.Context{
			Inventory:   inventory,
			Fingerprint: fingerprint,
			Book:        c.selectedBook,
		})
		if evaluationErr != nil {
			return preparedInteraction{}, terminalFailure(baseResult, cancelledOutcome(ctx), PhaseInterpretation, "HMGEN-INTERPRETATION-FAILED", evaluationErr)
		}

		baseResult.Provenance.Interpretations = append(baseResult.Provenance.Interpretations, evaluation.Provenance)

		if evaluation.Response != nil || evaluation.Status != intent.StatusClarificationRequired {
			return c.finishPlanning(ctx, inventory, nil, c.selectedBook, evaluation, baseResult, options)
		}

		baseResult.Clarifications = cloneIntentClarifications(evaluation.Clarifications)
		baseResult.Transitions = append(baseResult.Transitions, StateCandidateChange, StateClarifying)

		if c.clarifier == nil {
			if options.preview {
				baseResult.State = StateClarifying

				return preparedInteraction{}, outcomeResult(baseResult, OutcomeClarificationRequired)
			}

			return preparedInteraction{}, terminalResult(baseResult, OutcomeClarificationRequired)
		}

		if round >= MaximumClarificationRounds || len(clarifications)+len(evaluation.Clarifications) > MaximumClarificationExchanges {
			baseResult.Diagnostics = append(baseResult.Diagnostics, Diagnostic{
				Code:    "HMGEN-CLARIFICATION-LIMIT",
				Phase:   PhaseClarification,
				Field:   "clarifications",
				Message: "clarification limit reached before the intent was complete",
			})

			return preparedInteraction{}, terminalResult(baseResult, OutcomeClarificationRequired)
		}

		baseResult.State = StateClarifying

		response, clarificationErr := c.clarifier.Clarify(ctx, ClarificationRequest{
			Round:     round + 1,
			Questions: cloneIntentClarifications(evaluation.Clarifications),
		})
		if clarificationErr != nil {
			return preparedInteraction{}, terminalFailure(baseResult, cancelledOutcome(ctx), PhaseClarification, "HMGEN-CLARIFICATION-FAILED", clarificationErr)
		}

		if response.Cancelled {
			return preparedInteraction{}, terminalResult(baseResult, OutcomeCancelled)
		}

		exchanges, responseErr := bindClarifications(evaluation.Clarifications, response.Answers)
		if responseErr != nil {
			return preparedInteraction{}, terminalFailure(baseResult, OutcomeFailed, PhaseClarification, "HMGEN-CLARIFICATION-INVALID", responseErr)
		}

		clarifications = append(clarifications, exchanges...)
		baseResult.Provenance.ClarificationRounds++
		baseResult.State = StateInterpreting
	}
}

func (c *Coordinator) prepareApplication(
	ctx context.Context,
	root string,
	prompt string,
	baseResult Result,
	options preparationOptions,
) (preparedInteraction, *Result) {
	selectedBook := c.applicationBook

	provisional, err := project.InspectTarget(ctx, project.TargetRequest{
		Parent: root, Target: ".hatmax-proposed-application", Book: selectedBook, Options: c.inspectionOptions,
	})
	if err != nil {
		return preparedInteraction{}, terminalFailure(baseResult, OutcomeFailed, PhaseInspection, "HMGEN-TARGET-INSPECTION-FAILED", err)
	}

	fingerprint, err := provisional.Fingerprint(selectedBook.Manifest().BookVersion)
	if err != nil {
		return preparedInteraction{}, terminalFailure(baseResult, OutcomeFailed, PhaseInspection, "HMGEN-FINGERPRINT-FAILED", err)
	}

	currentTarget := provisional
	targetResolved := false

	clarifications := append([]eval.ClarificationExchange{}, options.clarifications...)
	baseResult.State = StateInterpreting
	baseResult.Transitions = append(baseResult.Transitions, StateInterpreting)

	for round := 0; ; round++ {
		evaluation, evaluationErr := eval.EvaluateDialogue(ctx, c.interpreter, prompt, options.dialogue, clarifications, eval.Context{
			Target: &currentTarget, TargetResolved: targetResolved, Fingerprint: fingerprint, Book: selectedBook,
		})
		if evaluationErr != nil {
			return preparedInteraction{}, terminalFailure(baseResult, cancelledOutcome(ctx), PhaseInterpretation, "HMGEN-INTERPRETATION-FAILED", evaluationErr)
		}

		baseResult.Provenance.Interpretations = append(baseResult.Provenance.Interpretations, evaluation.Provenance)

		if evaluation.Intent != nil && evaluation.Intent.Operation == intent.OperationCreateApplication {
			rebound, target, targetFingerprint, bindErr := c.bindApplicationTarget(ctx, root, *evaluation.Intent, evaluation.Provenance)
			if bindErr != nil {
				return preparedInteraction{}, terminalFailure(baseResult, OutcomeFailed, PhaseInspection, "HMGEN-TARGET-INSPECTION-FAILED", bindErr)
			}

			evaluation = rebound
			currentTarget = target
			fingerprint = targetFingerprint
			targetResolved = true
		}

		if evaluation.Response != nil || evaluation.Status != intent.StatusClarificationRequired {
			return c.finishPlanning(ctx, project.Inventory{}, &currentTarget, selectedBook, evaluation, baseResult, options)
		}

		baseResult.Clarifications = cloneIntentClarifications(evaluation.Clarifications)
		baseResult.Transitions = append(baseResult.Transitions, StateCandidateChange, StateClarifying)

		if c.clarifier == nil {
			if options.preview {
				baseResult.State = StateClarifying

				return preparedInteraction{}, outcomeResult(baseResult, OutcomeClarificationRequired)
			}

			return preparedInteraction{}, terminalResult(baseResult, OutcomeClarificationRequired)
		}

		if round >= MaximumClarificationRounds || len(clarifications)+len(evaluation.Clarifications) > MaximumClarificationExchanges {
			baseResult.Diagnostics = append(baseResult.Diagnostics, Diagnostic{
				Code: "HMGEN-CLARIFICATION-LIMIT", Phase: PhaseClarification, Field: "clarifications",
				Message: "clarification limit reached before the intent was complete",
			})

			return preparedInteraction{}, terminalResult(baseResult, OutcomeClarificationRequired)
		}

		baseResult.State = StateClarifying

		response, clarificationErr := c.clarifier.Clarify(ctx, ClarificationRequest{
			Round: round + 1, Questions: cloneIntentClarifications(evaluation.Clarifications),
		})
		if clarificationErr != nil {
			return preparedInteraction{}, terminalFailure(baseResult, cancelledOutcome(ctx), PhaseClarification, "HMGEN-CLARIFICATION-FAILED", clarificationErr)
		}

		if response.Cancelled {
			return preparedInteraction{}, terminalResult(baseResult, OutcomeCancelled)
		}

		exchanges, responseErr := bindClarifications(evaluation.Clarifications, response.Answers)
		if responseErr != nil {
			return preparedInteraction{}, terminalFailure(baseResult, OutcomeFailed, PhaseClarification, "HMGEN-CLARIFICATION-INVALID", responseErr)
		}

		clarifications = append(clarifications, exchanges...)
		baseResult.Provenance.ClarificationRounds++
		baseResult.State = StateInterpreting
	}
}

func (c *Coordinator) bindApplicationTarget(
	ctx context.Context,
	root string,
	value intent.Intent,
	provenance eval.Provenance,
) (eval.Result, project.TargetInventory, project.Fingerprint, error) {
	plannedPaths, err := plan.ApplicationTargetPaths(value, c.applicationBook)
	if err != nil {
		return eval.Result{}, project.TargetInventory{}, project.Fingerprint{}, err
	}

	if value.Target == nil || strings.TrimSpace(value.Target.Directory) == "" {
		return eval.Result{}, project.TargetInventory{}, project.Fingerprint{}, errors.New("application target directory is required")
	}

	target, err := project.InspectTarget(ctx, project.TargetRequest{
		Parent: root, Target: value.Target.Directory, PlannedPaths: plannedPaths,
		Book: c.applicationBook, Options: c.inspectionOptions,
	})
	if err != nil {
		return eval.Result{}, project.TargetInventory{}, project.Fingerprint{}, err
	}

	fingerprint, err := target.Fingerprint(c.applicationBook.Manifest().BookVersion)
	if err != nil {
		return eval.Result{}, project.TargetInventory{}, project.Fingerprint{}, err
	}

	value.SourceFingerprint = fingerprint.Value

	rebound, err := eval.EvaluateIntent(value, provenance, eval.Context{
		Target: &target, Fingerprint: fingerprint, Book: c.applicationBook,
	})
	if err != nil {
		return eval.Result{}, project.TargetInventory{}, project.Fingerprint{}, err
	}

	return rebound, target, fingerprint, nil
}

func (c *Coordinator) finishPlanning(
	ctx context.Context,
	inventory project.Inventory,
	target *project.TargetInventory,
	selectedBook *book.Book,
	evaluation eval.Result,
	baseResult Result,
	options preparationOptions,
) (preparedInteraction, *Result) {
	if evaluation.Response != nil {
		baseResult.State = StateConversation
		baseResult.Transitions = append(baseResult.Transitions, StateConversation)
		baseResult.Response = &eval.ConversationResponse{Content: evaluation.Response.Content}

		return preparedInteraction{}, outcomeResult(baseResult, OutcomeConversationResponse)
	}

	if evaluation.Status != intent.StatusAdmitted || evaluation.Plan == nil {
		baseResult.Transitions = append(baseResult.Transitions, StateCandidateChange)
		baseResult.Diagnostics = intentDiagnostics(evaluation.Diagnostics)

		outcome := OutcomeIntentRejected
		if evaluation.Status == intent.StatusCapabilityUnsupported {
			outcome = OutcomeUnsupported
		}

		return preparedInteraction{}, terminalResult(baseResult, outcome)
	}

	sealedPlan := *evaluation.Plan

	baseResult.Transitions = append(baseResult.Transitions, StateCandidateChange)

	serialized, err := plan.MarshalYAML(sealedPlan)
	if err != nil {
		return preparedInteraction{}, terminalFailure(baseResult, OutcomeFailed, PhaseApproval, "HMGEN-PLAN-SERIALIZATION-FAILED", err)
	}

	if len(serialized) > MaximumPlanPresentationBytes {
		return preparedInteraction{}, terminalFailure(
			baseResult,
			OutcomeFailed,
			PhaseApproval,
			"HMGEN-PLAN-PRESENTATION-LIMIT",
			fmt.Errorf("canonical plan exceeds %d bytes", MaximumPlanPresentationBytes),
		)
	}

	lifecycle, err := plan.Activate(sealedPlan)
	if err != nil {
		return preparedInteraction{}, terminalFailure(baseResult, OutcomeFailed, PhaseApproval, "HMGEN-PLAN-ACTIVATION-FAILED", err)
	}

	baseResult.State = StateAwaitingApproval
	baseResult.Plan = &sealedPlan
	baseResult.Intent = evaluation.Intent

	baseResult.PlanYAML = append([]byte{}, serialized...)

	baseResult.Clarifications = []intent.Clarification{}
	if options.preview {
		baseResult.State = StatePlanReady
		baseResult.Transitions = append(baseResult.Transitions, StatePlanReady)

		return preparedInteraction{}, outcomeResult(baseResult, OutcomePlanReady)
	}

	if options.approvedPlanDigest != "" {
		if options.approvedPlanDigest != sealedPlan.Digest {
			return preparedInteraction{}, terminalFailure(
				baseResult,
				OutcomePlanStale,
				PhaseApproval,
				"HMGEN-APPROVAL-DIGEST-MISMATCH",
				fmt.Errorf("approved plan digest does not match the current sealed plan"),
			)
		}
	} else {
		decision, approvalErr := c.approver.Approve(ctx, ApprovalRequest{
			PlanDigest:         sealedPlan.Digest,
			ProjectFingerprint: sealedPlan.ProjectFingerprint,
			SourceFingerprint:  sealedPlan.SourceFingerprint,
			PlanYAML:           append([]byte{}, serialized...),
		})
		if approvalErr != nil {
			return preparedInteraction{}, terminalFailure(baseResult, cancelledOutcome(ctx), PhaseApproval, "HMGEN-APPROVAL-FAILED", approvalErr)
		}

		if decision == ApprovalRejected {
			lifecycle.Reject()

			return preparedInteraction{}, terminalResult(baseResult, OutcomeCancelled)
		}

		if decision != ApprovalGranted {
			return preparedInteraction{}, terminalFailure(
				baseResult,
				OutcomeFailed,
				PhaseApproval,
				"HMGEN-APPROVAL-INVALID",
				fmt.Errorf("unknown approval decision %q", decision),
			)
		}
	}

	if target != nil {
		currentTarget, targetErr := project.InspectTarget(ctx, project.TargetRequest{
			Parent: target.Parent, Target: target.Target, PlannedPaths: target.PlannedPaths,
			Book: selectedBook, Options: c.inspectionOptions,
		})
		if targetErr != nil {
			return preparedInteraction{}, terminalFailure(baseResult, OutcomeFailed, PhaseFreshness, "HMGEN-REINSPECTION-FAILED", targetErr)
		}

		currentFingerprint, fingerprintErr := currentTarget.Fingerprint(sealedPlan.BookVersion)
		if fingerprintErr != nil {
			return preparedInteraction{}, terminalFailure(baseResult, OutcomeFailed, PhaseFreshness, "HMGEN-FINGERPRINT-FAILED", fingerprintErr)
		}

		transition, claimErr := lifecycle.Claim(currentFingerprint)
		if claimErr != nil {
			return preparedInteraction{}, terminalFailure(baseResult, OutcomeFailed, PhaseFreshness, "HMGEN-PLAN-FRESHNESS-FAILED", claimErr)
		}

		if transition.State == plan.LifecycleStale {
			baseResult.FreshnessChanges = append([]project.Change{}, transition.Changes...)

			return preparedInteraction{}, terminalResult(baseResult, OutcomePlanStale)
		}

		if transition.State != plan.LifecycleConsumed {
			return preparedInteraction{}, terminalFailure(baseResult, OutcomeFailed, PhaseFreshness, "HMGEN-PLAN-UNAVAILABLE", fmt.Errorf("plan lifecycle entered %q", transition.State))
		}

		return preparedInteraction{
			target: &currentTarget, book: selectedBook, plan: sealedPlan,
			planYAML: append([]byte{}, serialized...), result: baseResult,
		}, nil
	}

	currentInventory, err := project.InspectWithOptions(ctx, inventory.Root, c.inspectionOptions)
	if err != nil {
		return preparedInteraction{}, terminalFailure(baseResult, OutcomeFailed, PhaseFreshness, "HMGEN-REINSPECTION-FAILED", err)
	}

	currentFingerprint, err := currentInventory.Fingerprint(project.FingerprintRequest{
		BookVersion:          sealedPlan.BookVersion,
		SelectedPaths:        sealedPlan.FingerprintInputs.SelectedPaths,
		SelectedDependencies: sealedPlan.FingerprintInputs.SelectedDependencies,
		PlannedSurfaces:      sealedPlan.FingerprintInputs.PlannedSurfaces,
	})
	if err != nil {
		return preparedInteraction{}, terminalFailure(baseResult, OutcomeFailed, PhaseFreshness, "HMGEN-FINGERPRINT-FAILED", err)
	}

	transition, err := lifecycle.Claim(currentFingerprint)
	if err != nil {
		return preparedInteraction{}, terminalFailure(baseResult, OutcomeFailed, PhaseFreshness, "HMGEN-PLAN-FRESHNESS-FAILED", err)
	}

	if transition.State == plan.LifecycleStale {
		baseResult.FreshnessChanges = append([]project.Change{}, transition.Changes...)
		for _, diagnostic := range transition.Diagnostics {
			baseResult.Diagnostics = append(baseResult.Diagnostics, Diagnostic{
				Code:    diagnostic.Code,
				Phase:   PhaseFreshness,
				Field:   diagnostic.Field,
				Message: diagnostic.Message,
			})
		}

		return preparedInteraction{}, terminalResult(baseResult, OutcomePlanStale)
	}

	if transition.State != plan.LifecycleConsumed {
		return preparedInteraction{}, terminalFailure(
			baseResult,
			OutcomeFailed,
			PhaseFreshness,
			"HMGEN-PLAN-UNAVAILABLE",
			fmt.Errorf("plan lifecycle entered %q", transition.State),
		)
	}

	return preparedInteraction{
		inventory: currentInventory,
		book:      selectedBook,
		plan:      sealedPlan,
		planYAML:  append([]byte{}, serialized...),
		result:    baseResult,
	}, nil
}

func bookFingerprintRequest(selectedBook *book.Book) project.FingerprintRequest {
	surfaces := make(map[string]struct{})
	dependencies := make(map[string]struct{})

	for _, archetype := range selectedBook.Archetypes() {
		collectStrings(surfaces, archetype.Surfaces)

		for _, obligation := range archetype.Obligations {
			collectStrings(surfaces, obligation.Surfaces)
		}
	}

	for _, capability := range selectedBook.Capabilities() {
		collectStrings(surfaces, capability.Surfaces)

		for _, obligation := range capability.Obligations {
			collectStrings(surfaces, obligation.Surfaces)
		}

		for _, dependency := range capability.Dependencies {
			dependencies[dependency.Module] = struct{}{}
		}
	}

	return project.FingerprintRequest{
		BookVersion:          selectedBook.Manifest().BookVersion,
		SelectedDependencies: sortedKeys(dependencies),
		PlannedSurfaces:      sortedKeys(surfaces),
	}
}

func collectStrings(target map[string]struct{}, values []string) {
	for _, value := range values {
		target[value] = struct{}{}
	}
}

func sortedKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}

	sort.Strings(result)

	return result
}

func bindClarifications(
	questions []intent.Clarification,
	answers []ClarificationAnswer,
) ([]eval.ClarificationExchange, error) {
	if len(questions) != len(answers) {
		return nil, errors.New("every clarification question requires exactly one answer")
	}

	byField := make(map[string]string, len(answers))
	for _, answer := range answers {
		field := strings.TrimSpace(answer.Field)

		value := strings.TrimSpace(answer.Answer)
		if field == "" || value == "" {
			return nil, errors.New("clarification field and answer are required")
		}

		if len(answer.Field) > eval.MaximumClarificationTextBytes || len(answer.Answer) > eval.MaximumClarificationTextBytes {
			return nil, fmt.Errorf("clarification values exceed %d bytes", eval.MaximumClarificationTextBytes)
		}

		if _, exists := byField[field]; exists {
			return nil, fmt.Errorf("clarification field %q is duplicated", field)
		}

		byField[field] = value
	}

	result := make([]eval.ClarificationExchange, 0, len(questions))
	for _, question := range questions {
		answer, exists := byField[question.Field]
		if !exists {
			return nil, fmt.Errorf("clarification field %q was not answered", question.Field)
		}

		result = append(result, eval.ClarificationExchange{
			Field:    question.Field,
			Question: question.Question,
			Answer:   answer,
		})
	}

	return result, nil
}

func intentDiagnostics(values []intent.Diagnostic) []Diagnostic {
	result := make([]Diagnostic, 0, len(values))
	for _, value := range values {
		result = append(result, Diagnostic{
			Code:    value.Code,
			Phase:   PhaseInterpretation,
			Field:   value.Field,
			Message: value.Message,
		})
	}

	return result
}

func terminalFailure(base Result, outcome Outcome, phase Phase, fallbackCode string, err error) *Result {
	base.Diagnostics = append(base.Diagnostics, diagnosticFromError(phase, fallbackCode, err))

	return terminalResult(base, outcome)
}

func outcomeResult(base Result, outcome Outcome) *Result {
	base.Outcome = outcome

	return &base
}

func terminalResult(base Result, outcome Outcome) *Result {
	base.State = StateFinished
	base.Outcome = outcome

	return &base
}

func cancelledOutcome(ctx context.Context) Outcome {
	if ctx.Err() != nil {
		return OutcomeCancelled
	}

	return OutcomeFailed
}

func diagnosticFromError(phase Phase, fallbackCode string, err error) Diagnostic {
	diagnostic := Diagnostic{Code: fallbackCode, Phase: phase, Message: err.Error()}

	var projectErr project.Error
	if errors.As(err, &projectErr) {
		diagnostic.Code = stableCode(projectErr.Code)
		diagnostic.Field = projectErr.Path

		return diagnostic
	}

	var evaluationErr eval.Error
	if errors.As(err, &evaluationErr) {
		diagnostic.Code = stableCode(evaluationErr.Code)
		diagnostic.Field = evaluationErr.Field

		return diagnostic
	}

	var backendErr eval.BackendError
	if errors.As(err, &backendErr) {
		diagnostic.Code = stableCode(string(backendErr.Code))
		diagnostic.Field = backendErr.Operation

		return diagnostic
	}

	var executionErr execute.Error
	if errors.As(err, &executionErr) {
		diagnostic.Code = stableCode(executionErr.Code)
		diagnostic.Field = executionErr.Path
	}

	return diagnostic
}

func stableCode(value string) string {
	return "HMGEN-" + strings.ToUpper(strings.ReplaceAll(value, "_", "-"))
}

func cloneIntentClarifications(values []intent.Clarification) []intent.Clarification {
	return append([]intent.Clarification{}, values...)
}
