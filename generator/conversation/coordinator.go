package conversation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"hatmax.adrianpk.com/generator/eval"
	"hatmax.adrianpk.com/generator/interaction"
	"hatmax.adrianpk.com/generator/project"
)

// TurnEngine is the backend-neutral Hatmax interaction kernel used by a
// persistent conversation.
type TurnEngine interface {
	PreviewTurn(context.Context, string, interaction.TurnRequest) interaction.Result
	RunApprovedTurn(context.Context, string, interaction.TurnRequest) interaction.Result
	ProjectFingerprint(context.Context, string) (project.Fingerprint, error)
}

// CoordinatorConfig supplies persistent state and deterministic interaction
// services. Clock and IDSource exist for repeatable tests.
type CoordinatorConfig struct {
	Store                   Store
	Engine                  TurnEngine
	BookContract            BookContract
	ApplicationBookContract BookContract
	BackendIdentity         BackendIdentity
	Clock                   func() time.Time
	IDSource                func(string) (string, error)
}

// Coordinator opens project-scoped or pre-project conversational sessions.
type Coordinator struct {
	store                   Store
	engine                  TurnEngine
	bookContract            BookContract
	applicationBookContract BookContract
	backendIdentity         BackendIdentity
	clock                   func() time.Time
	idSource                func(string) (string, error)
}

// SessionOptions selects a stored conversation or deliberately starts fresh.
type SessionOptions struct {
	ConversationID string
	Fresh          bool
}

// TurnRequest is one visible user turn. Decisions is optional while focused
// clarifications are pending; in that case Content is their combined answer.
type TurnRequest struct {
	Content   string
	Decisions []Decision
}

// SessionResult combines the durable snapshot with one kernel result.
type SessionResult struct {
	Conversation      Conversation
	OperationID       string
	Interaction       interaction.Result
	PersistenceFailed bool
}

// ActiveSession owns the scope lock and one resumable conversation.
type ActiveSession struct {
	coordinator       *Coordinator
	store             Session
	root              string
	current           Conversation
	persistenceFailed bool
	closed            bool
}

// NewCoordinator constructs the terminal-independent conversational state
// machine.
func NewCoordinator(config CoordinatorConfig) (*Coordinator, error) {
	if config.Store == nil || config.Engine == nil {
		return nil, errors.New("conversation store and interaction engine are required")
	}

	if config.BookContract.BookVersion < 1 || config.BookContract.InterpreterVersion < 1 {
		return nil, errors.New("positive Book and interpreter contracts are required")
	}

	applicationContract := config.ApplicationBookContract
	if applicationContract.BookVersion == 0 && applicationContract.InterpreterVersion == 0 {
		applicationContract = config.BookContract
	}

	if applicationContract.BookVersion < 1 || applicationContract.InterpreterVersion < 1 {
		return nil, errors.New("positive application Book and interpreter contracts are required")
	}

	if strings.TrimSpace(config.BackendIdentity.Adapter) == "" {
		return nil, errors.New("backend adapter identity is required")
	}

	clock := config.Clock
	if clock == nil {
		clock = time.Now
	}

	idSource := config.IDSource
	if idSource == nil {
		idSource = randomLocalID
	}

	return &Coordinator{
		store: config.Store, engine: config.Engine,
		bookContract: config.BookContract, applicationBookContract: applicationContract,
		backendIdentity: config.BackendIdentity,
		clock:           clock, idSource: idSource,
	}, nil
}

// Open resumes the compatible scope conversation by default, or applies an
// explicit selection/fresh request without changing project source.
func (c *Coordinator) Open(ctx context.Context, root string, options SessionOptions) (*ActiveSession, error) {
	absoluteRoot, kind, err := ResolveSessionRoot(root)
	if err != nil {
		return nil, err
	}

	scope, err := ResolveScope(kind, absoluteRoot)
	if err != nil {
		return nil, err
	}

	contract := c.bookContract
	if kind == ScopePreProject {
		contract = c.applicationBookContract
	}

	stored, err := c.store.Open(ctx, OpenRequest{
		Scope: scope, BookContract: contract, BackendIdentity: c.backendIdentity,
		ConversationID: options.ConversationID, Fresh: options.Fresh,
	})
	if err != nil {
		return nil, err
	}

	active := &ActiveSession{
		coordinator: c, store: stored, root: absoluteRoot, current: stored.Current(),
	}
	if kind != ScopeProject {
		return active, nil
	}

	fingerprint, inspectErr := c.engine.ProjectFingerprint(ctx, absoluteRoot)
	if inspectErr != nil {
		return active, nil
	}

	value := stored.Current()
	value.ReconcileResume(contract, fingerprint.Value, c.clock().UTC())

	replaceErr := stored.Replace(ctx, value)
	if replaceErr != nil {
		_ = stored.Close()

		return nil, replaceErr
	}

	active.current = stored.Current()

	return active, nil
}

// Current returns a mutation-isolated snapshot.
func (s *ActiveSession) Current() Conversation {
	return Clone(s.current)
}

// Turn classifies one message and persists only bounded user-visible context
// plus non-authoritative proposal state.
func (s *ActiveSession) Turn(ctx context.Context, request TurnRequest) (SessionResult, error) {
	if s.closed {
		return SessionResult{}, errors.New("conversation session is closed")
	}

	content := strings.TrimSpace(request.Content)
	if content == "" {
		return SessionResult{}, errors.New("conversation turn content is required")
	}

	value := Clone(s.current)
	dialogue := replayDialogue(value)
	operation := activeOperation(value)
	operationID := ""
	prompt := content
	decisions := []Decision{}
	turnKind := TurnDialogue

	if operation != nil && operation.Status == OperationClarifying {
		operationID = operation.ID
		prompt = operation.RequestSummary
		turnKind = TurnDecision

		var err error

		decisions, err = bindTurnDecisions(*operation, request)
		if err != nil {
			return SessionResult{}, err
		}
	}

	now := s.coordinator.clock().UTC()

	err := value.AppendTurn(RoleUser, turnKind, content, operationID, now)
	if err != nil {
		return SessionResult{}, err
	}

	clarifications := operationDecisions(operation, decisions)
	result := s.coordinator.engine.PreviewTurn(ctx, s.root, interaction.TurnRequest{
		Prompt: prompt, Conversation: dialogue, Clarifications: clarifications,
	})

	if operationID == "" && result.Outcome != interaction.OutcomeConversationResponse {
		generatedID, idErr := s.coordinator.idSource("operation")
		if idErr != nil {
			return SessionResult{}, fmt.Errorf("create operation identity: %w", idErr)
		}

		operationID = generatedID

		err = value.AddOperation(operationID, content, now)
		if err != nil {
			return SessionResult{}, err
		}

		bindLastTurn(&value, operationID, TurnGoal)
	}

	err = s.applyPreview(ctx, &value, operationID, decisions, result, now)
	if err != nil {
		return SessionResult{}, err
	}

	s.persist(ctx, &value, operationID, &result, now)

	return s.sessionResult(operationID, result), nil
}

// Approve recomputes and executes one stored proposal only for the exact
// currently displayed digest. Stored approval never enters this method.
func (s *ActiveSession) Approve(ctx context.Context, operationID, planDigest string) (SessionResult, error) {
	if s.closed {
		return SessionResult{}, errors.New("conversation session is closed")
	}

	value := Clone(s.current)

	operation, err := plannedOperation(value, operationID)
	if err != nil {
		return SessionResult{}, err
	}

	if planDigest == "" || planDigest != operation.PlanDigest {
		return s.persistInvalidApproval(ctx, value, operation, planDigest)
	}

	result := s.coordinator.engine.RunApprovedTurn(ctx, s.root, interaction.TurnRequest{
		Prompt:             operation.RequestSummary,
		Conversation:       replayDialogue(value),
		Clarifications:     operationDecisions(&operation, nil),
		ApprovalPlanDigest: planDigest,
	})

	now := s.coordinator.clock().UTC()

	updateBackendIdentity(&value, result)

	status := operationStatusForResult(result)

	summary := resultSummary(result)

	err = value.TransitionOperation(operation.ID, status, summary, now)
	if err != nil {
		return SessionResult{}, err
	}

	err = appendResultTurns(&value, operation.ID, result, summary, now)
	if err != nil {
		return SessionResult{}, err
	}

	s.persist(ctx, &value, operation.ID, &result, now)

	if !s.persistenceFailed && completedMutation(result.Outcome) && result.Plan != nil && result.Plan.Target != nil {
		projectScope, scopeErr := ResolveScope(ScopeProject, result.Plan.Target.Path)
		if scopeErr != nil {
			return SessionResult{}, scopeErr
		}

		rebindErr := s.store.Rebind(ctx, projectScope, s.coordinator.bookContract, now)
		if rebindErr != nil {
			return SessionResult{}, rebindErr
		}

		s.current = s.store.Current()
	}

	return s.sessionResult(operation.ID, result), nil
}

func completedMutation(outcome interaction.Outcome) bool {
	return outcome == interaction.OutcomeCompleted || outcome == interaction.OutcomeValidationIncomplete
}

// Cancel abandons one revisable proposal without discarding its visible goal,
// decisions, plan identity, or diagnostics.
func (s *ActiveSession) Cancel(ctx context.Context, operationID string) (SessionResult, error) {
	if s.closed {
		return SessionResult{}, errors.New("conversation session is closed")
	}

	value := Clone(s.current)

	operation := operationByID(value, operationID)
	if operation == nil {
		return SessionResult{}, modelError("conversation_operation_missing", "operation.id", "operation %q is not retained", operationID)
	}

	now := s.coordinator.clock().UTC()

	err := value.TransitionOperation(operationID, OperationCancelled, "cancelled by user", now)
	if err != nil {
		return SessionResult{}, err
	}

	err = value.AppendTurn(RoleHatmax, TurnResult, "Cancelled without project changes.", operationID, now)
	if err != nil {
		return SessionResult{}, err
	}

	result := interaction.Result{State: interaction.StateFinished, Outcome: interaction.OutcomeCancelled}
	s.persist(ctx, &value, operationID, &result, now)

	return s.sessionResult(operationID, result), nil
}

// Reset starts a fresh conversation for the current scope and preserves the
// previous snapshot for ordinary retention.
func (s *ActiveSession) Reset(ctx context.Context) (Conversation, error) {
	id, err := s.coordinator.idSource("conversation")
	if err != nil {
		return Conversation{}, fmt.Errorf("create conversation identity: %w", err)
	}

	now := s.coordinator.clock().UTC()
	if s.persistenceFailed {
		_, next, resetErr := s.current.Reset(id, now)
		if resetErr != nil {
			return Conversation{}, resetErr
		}

		s.current = next

		return Clone(next), nil
	}

	next, err := s.store.Reset(ctx, id, now)
	if err != nil {
		return Conversation{}, err
	}

	s.current = next

	return Clone(next), nil
}

// Close releases the scope lock.
func (s *ActiveSession) Close() error {
	if s.closed {
		return nil
	}

	s.closed = true

	return s.store.Close()
}

func (s *ActiveSession) applyPreview(
	ctx context.Context,
	value *Conversation,
	operationID string,
	decisions []Decision,
	result interaction.Result,
	now time.Time,
) error {
	updateBackendIdentity(value, result)

	switch result.Outcome {
	case interaction.OutcomeConversationResponse:
		if result.Response != nil {
			return value.AppendTurn(RoleHatmax, TurnDialogue, result.Response.Content, operationID, now)
		}
	case interaction.OutcomeClarificationRequired:
		operation := operationByID(*value, operationID)

		collected := append([]Decision{}, decisions...)
		if operation != nil {
			collected = mergeDecisions(operation.Decisions, decisions)
		}

		err := value.SetClarifyingOperation(operationID, result.Clarifications, collected, now)
		if err != nil {
			return err
		}

		for _, clarification := range result.Clarifications {
			err = value.AppendTurn(RoleHatmax, TurnClarification, clarification.Question, operationID, now)
			if err != nil {
				return err
			}
		}
	case interaction.OutcomePlanReady:
		if result.Plan == nil || result.Intent == nil {
			return errors.New("plan-ready interaction omitted its plan or typed intent")
		}

		if operation := operationByID(*value, operationID); operation != nil && len(decisions) > 0 {
			err := value.SetClarifyingOperation(
				operationID, nil, mergeDecisions(operation.Decisions, decisions), now,
			)
			if err != nil {
				return err
			}
		}

		if result.Plan.Target != nil && value.Scope.Kind == ScopePreProject {
			targetScope, err := ResolveScope(ScopePreProject, result.Plan.Target.Path)
			if err != nil {
				return err
			}

			if targetScope != value.Scope {
				err = s.store.Replace(ctx, *value)
				if err != nil {
					return err
				}

				err = s.store.Rebind(ctx, targetScope, value.BookContract, now)
				if err != nil {
					return err
				}

				*value = s.store.Current()
			}
		}

		fingerprint := result.Plan.ProjectFingerprint
		if fingerprint == "" {
			fingerprint = result.Plan.SourceFingerprint
		}

		err := value.SetPlannedOperation(
			operationID, result.Intent.SchemaVersion, *result.Intent,
			result.Plan.Digest, fingerprint, now,
		)
		if err != nil {
			return err
		}

		return value.AppendTurn(RoleHatmax, TurnResult, "Plan ready: "+result.Plan.Digest, operationID, now)
	case interaction.OutcomeUnsupported, interaction.OutcomeIntentRejected:
		err := value.TransitionOperation(operationID, OperationFailed, resultSummary(result), now)
		if err != nil {
			return err
		}

		err = appendResultTurns(value, operationID, result, resultSummary(result), now)
		if err != nil {
			return err
		}
	case interaction.OutcomeCancelled:
		return value.TransitionOperation(operationID, OperationCancelled, resultSummary(result), now)
	default:
		if operationID != "" {
			err := value.TransitionOperation(operationID, OperationFailed, resultSummary(result), now)
			if err != nil {
				return err
			}

			err = appendResultTurns(value, operationID, result, resultSummary(result), now)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *ActiveSession) persistInvalidApproval(
	ctx context.Context,
	value Conversation,
	operation ProposedOperation,
	planDigest string,
) (SessionResult, error) {
	result := interaction.Result{
		State: interaction.StateFinished, Outcome: interaction.OutcomePlanStale,
		Diagnostics: []interaction.Diagnostic{{
			Code: "HMGEN-APPROVAL-DIGEST-MISMATCH", Phase: interaction.PhaseApproval,
			Field: "plan_digest", Message: "approval does not match the current displayed plan",
		}},
	}

	now := s.coordinator.clock().UTC()

	content := result.Diagnostics[0].Message
	if planDigest == "" {
		content = "approval requires the current displayed plan digest"
	}

	err := value.AppendTurn(RoleHatmax, TurnDiagnostic, content, operation.ID, now)
	if err != nil {
		return SessionResult{}, err
	}

	s.persist(ctx, &value, operation.ID, &result, now)

	return s.sessionResult(operation.ID, result), nil
}

func (s *ActiveSession) persist(
	ctx context.Context,
	value *Conversation,
	operationID string,
	result *interaction.Result,
	now time.Time,
) {
	if s.persistenceFailed {
		s.current = Clone(*value)

		return
	}

	err := s.store.Replace(ctx, *value)
	if err == nil {
		s.current = s.store.Current()

		return
	}

	s.persistenceFailed = true

	result.Diagnostics = append(result.Diagnostics, interaction.Diagnostic{
		Code: "HMGEN-STATE-PERSISTENCE-FAILED", Phase: interaction.PhasePersistence,
		Field: "conversation_state", Message: "local conversation state could not be persisted; this session continues in memory only",
	})

	if operationID != "" {
		_ = value.AppendTurn(
			RoleHatmax,
			TurnDiagnostic,
			"HMGEN-STATE-PERSISTENCE-FAILED: local conversation state could not be persisted; this session continues in memory only",
			operationID,
			now,
		)
	}

	s.current = Clone(*value)
}

func (s *ActiveSession) sessionResult(operationID string, result interaction.Result) SessionResult {
	return SessionResult{
		Conversation: Clone(s.current), OperationID: operationID,
		Interaction: result, PersistenceFailed: s.persistenceFailed,
	}
}

// ResolveSessionRoot classifies one existing directory as a Hatmax project or
// a parent directory that can host an application conversation.
func ResolveSessionRoot(root string) (string, ScopeKind, error) {
	if strings.TrimSpace(root) == "" {
		return "", "", errors.New("conversation root is required")
	}

	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", "", fmt.Errorf("resolve conversation root: %w", err)
	}

	absolute = filepath.Clean(absolute)

	info, err := os.Stat(absolute)
	if err != nil {
		return "", "", fmt.Errorf("inspect conversation root: %w", err)
	}

	if !info.IsDir() {
		return "", "", errors.New("conversation root must be a directory")
	}

	kind := ScopePreProject

	module, moduleErr := os.Stat(filepath.Join(absolute, "go.mod"))
	if moduleErr == nil && !module.IsDir() {
		kind = ScopeProject
	} else if moduleErr != nil && !errors.Is(moduleErr, os.ErrNotExist) {
		return "", "", fmt.Errorf("inspect conversation module: %w", moduleErr)
	}

	return absolute, kind, nil
}

func replayDialogue(value Conversation) []eval.DialogueTurn {
	turns := value.RecentTurns(MaximumReplayTurns, MaximumReplayBytes)
	result := make([]eval.DialogueTurn, 0, len(turns))

	for _, turn := range turns {
		role := eval.DialogueRoleUser
		if turn.Role == RoleHatmax {
			role = eval.DialogueRoleHatmax
		}

		result = append(result, eval.DialogueTurn{Role: role, Content: turn.Content})
	}

	return result
}

func activeOperation(value Conversation) *ProposedOperation {
	for index := len(value.Operations) - 1; index >= 0; index-- {
		operation := value.Operations[index]
		if operation.Status == OperationCandidate || operation.Status == OperationClarifying {
			return &operation
		}
	}

	return nil
}

func operationByID(value Conversation, id string) *ProposedOperation {
	for _, operation := range value.Operations {
		if operation.ID == id {
			return &operation
		}
	}

	return nil
}

func plannedOperation(value Conversation, id string) (ProposedOperation, error) {
	operation := operationByID(value, id)
	if operation == nil {
		return ProposedOperation{}, modelError("conversation_operation_missing", "operation.id", "operation %q is not retained", id)
	}

	if operation.Status != OperationPlanned {
		return ProposedOperation{}, modelError("conversation_operation_transition_invalid", "operation.status", "operation %q is not awaiting approval", id)
	}

	return *operation, nil
}

func bindTurnDecisions(operation ProposedOperation, request TurnRequest) ([]Decision, error) {
	decisions := append([]Decision{}, request.Decisions...)
	if len(decisions) == 0 {
		answer := strings.TrimSpace(request.Content)
		decisions = make([]Decision, 0, len(operation.Pending))

		for _, pending := range operation.Pending {
			decisions = append(decisions, Decision{
				Field: pending.Field, Question: pending.Question, Answer: answer,
			})
		}
	}

	if len(decisions) != len(operation.Pending) {
		return nil, errors.New("every pending clarification requires one explicit answer")
	}

	byField := make(map[string]Decision, len(decisions))
	for _, decision := range decisions {
		if strings.TrimSpace(decision.Answer) == "" {
			return nil, errors.New("clarification answer is required")
		}

		byField[decision.Field] = decision
	}

	result := make([]Decision, 0, len(operation.Pending))
	for _, pending := range operation.Pending {
		decision, exists := byField[pending.Field]
		if !exists {
			return nil, fmt.Errorf("clarification field %q was not answered", pending.Field)
		}

		decision.Question = pending.Question
		result = append(result, decision)
	}

	return result, nil
}

func operationDecisions(operation *ProposedOperation, additional []Decision) []eval.ClarificationExchange {
	decisions := append([]Decision{}, additional...)
	if operation != nil {
		decisions = mergeDecisions(operation.Decisions, additional)
	}

	result := make([]eval.ClarificationExchange, 0, len(decisions))
	for _, decision := range decisions {
		result = append(result, eval.ClarificationExchange{
			Field: decision.Field, Question: decision.Question, Answer: decision.Answer,
		})
	}

	return result
}

func mergeDecisions(previous, additional []Decision) []Decision {
	result := append([]Decision{}, previous...)

	indexByField := make(map[string]int, len(result))
	for index, decision := range result {
		indexByField[decision.Field] = index
	}

	for _, decision := range additional {
		if index, exists := indexByField[decision.Field]; exists {
			result[index] = decision

			continue
		}

		indexByField[decision.Field] = len(result)
		result = append(result, decision)
	}

	return result
}

func bindLastTurn(value *Conversation, operationID string, kind TurnKind) {
	if len(value.Turns) == 0 {
		return
	}

	value.Turns[len(value.Turns)-1].OperationID = operationID
	value.Turns[len(value.Turns)-1].Kind = kind
}

func updateBackendIdentity(value *Conversation, result interaction.Result) {
	for index := len(result.Provenance.Interpretations) - 1; index >= 0; index-- {
		provenance := result.Provenance.Interpretations[index]
		if provenance.ThreadID == "" {
			continue
		}

		value.BackendThreadID = provenance.ThreadID
		value.BackendIdentity.Adapter = provenance.Adapter
		value.BackendIdentity.Version = provenance.BackendVersion
		value.BackendIdentity.Model = provenance.EffectiveModel

		return
	}
}

func operationStatusForResult(result interaction.Result) OperationStatus {
	switch result.Outcome {
	case interaction.OutcomeCompleted, interaction.OutcomeValidationIncomplete:
		return OperationCompleted
	case interaction.OutcomeCancelled:
		return OperationCancelled
	case interaction.OutcomePlanStale:
		return OperationStale
	default:
		return OperationFailed
	}
}

func resultSummary(result interaction.Result) string {
	summary := string(result.Outcome)
	if summary == "" {
		summary = "interaction failed"
	}

	if result.Outcome == interaction.OutcomeValidationIncomplete {
		summary += ": generated changes were applied; required external tests could not run"
	} else if len(result.Diagnostics) > 0 {
		summary += ": " + conversationDiagnosticMessage(result.Diagnostics[0])
	}

	if result.Outcome != interaction.OutcomeValidationIncomplete && len(result.RetainedChanges) > 0 {
		targets := make([]string, 0, len(result.RetainedChanges))
		for _, change := range result.RetainedChanges {
			targets = append(targets, change.Target)
		}

		summary += "; retained changes: " + strings.Join(targets, ", ")
	}

	if len(summary) > MaximumResultSummaryBytes {
		summary = summary[:MaximumResultSummaryBytes]
	}

	return summary
}

func appendResultTurns(value *Conversation, operationID string, result interaction.Result, summary string, now time.Time) error {
	if strings.TrimSpace(summary) != "" {
		err := value.AppendTurn(RoleHatmax, TurnResult, summary, operationID, now)
		if err != nil {
			return err
		}
	}

	for _, diagnostic := range result.Diagnostics {
		message := diagnostic.Code + ": " + conversationDiagnosticMessage(diagnostic)

		err := value.AppendTurn(RoleHatmax, TurnDiagnostic, message, operationID, now)
		if err != nil {
			return err
		}
	}

	return nil
}

func conversationDiagnosticMessage(diagnostic interaction.Diagnostic) string {
	normalized := strings.ToLower(diagnostic.Code + " " + diagnostic.Message)
	if strings.Contains(normalized, "docker") || strings.Contains(normalized, "testcontainers") {
		if strings.Contains(normalized, "permission denied") {
			return "Docker-based tests could not run because Docker is not accessible to the current user."
		}

		return "Docker-based tests could not run because Docker infrastructure is unavailable."
	}

	message, _, _ := strings.Cut(strings.TrimSpace(diagnostic.Message), "\n")
	if strings.HasPrefix(strings.ToLower(message), "observed ") {
		return "validation did not meet the expected condition"
	}

	const maximumConversationDiagnosticBytes = 512
	if len(message) > maximumConversationDiagnosticBytes {
		message = message[:maximumConversationDiagnosticBytes]
	}

	return message
}

func randomLocalID(prefix string) (string, error) {
	var buffer [16]byte

	_, err := rand.Read(buffer[:])
	if err != nil {
		return "", err
	}

	return prefix + "-" + hex.EncodeToString(buffer[:]), nil
}
