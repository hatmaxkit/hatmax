package plan

import (
	"sync"

	"hatmax.adrianpk.com/generator/project"
)

// LifecycleState describes whether a sealed plan remains available for one
// execution handoff.
type LifecycleState string

const (
	// LifecycleReady means the plan may still be claimed once.
	LifecycleReady LifecycleState = "ready"
	// LifecycleConsumed means the plan was claimed for execution.
	LifecycleConsumed LifecycleState = "consumed"
	// LifecycleRejected means the user or product surface rejected the plan.
	LifecycleRejected LifecycleState = "rejected"
	// LifecycleStale means relevant project drift invalidated the plan.
	LifecycleStale LifecycleState = "stale"
)

// TransitionResult reports one lifecycle transition without executing a plan.
type TransitionResult struct {
	State       LifecycleState   `json:"state" yaml:"state"`
	Diagnostics []Diagnostic     `json:"diagnostics" yaml:"diagnostics"`
	Changes     []project.Change `json:"changes" yaml:"changes"`
}

// Lifecycle owns an immutable sealed plan and enforces its single-use state.
type Lifecycle struct {
	mu    sync.Mutex
	plan  Plan
	state LifecycleState
}

// Activate verifies and detaches a sealed plan for lifecycle control.
func Activate(value Plan) (*Lifecycle, error) {
	err := VerifyDigest(value)
	if err != nil {
		return nil, err
	}

	return &Lifecycle{
		plan:  clonePlan(value),
		state: LifecycleReady,
	}, nil
}

// State returns the current lifecycle state.
func (l *Lifecycle) State() LifecycleState {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.state
}

// Plan returns a detached copy of the sealed plan.
func (l *Lifecycle) Plan() Plan {
	l.mu.Lock()
	defer l.mu.Unlock()

	return clonePlan(l.plan)
}

// Claim atomically checks current project state and consumes a fresh plan for
// one later execution handoff. It performs no edits and invokes no executor.
func (l *Lifecycle) Claim(current project.Fingerprint) (TransitionResult, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.state != LifecycleReady {
		return unavailableTransition(l.state), nil
	}

	freshness, err := CheckFingerprint(l.plan, current)
	if err != nil {
		return TransitionResult{}, err
	}

	if freshness.Stale {
		l.state = LifecycleStale

		return TransitionResult{
			State:       l.state,
			Diagnostics: freshness.Diagnostics,
			Changes:     freshness.Changes,
		}, nil
	}

	l.state = LifecycleConsumed

	return TransitionResult{
		State:       l.state,
		Diagnostics: []Diagnostic{},
		Changes:     []project.Change{},
	}, nil
}

// Reject invalidates a ready plan without executing it.
func (l *Lifecycle) Reject() TransitionResult {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.state != LifecycleReady {
		return unavailableTransition(l.state)
	}

	l.state = LifecycleRejected

	return TransitionResult{
		State:       l.state,
		Diagnostics: []Diagnostic{},
		Changes:     []project.Change{},
	}
}

func unavailableTransition(state LifecycleState) TransitionResult {
	return TransitionResult{
		State: state,
		Diagnostics: []Diagnostic{{
			Code:    "HMGEN-PLAN-USED",
			Field:   "state",
			Message: "plan is no longer available for execution",
		}},
		Changes: []project.Change{},
	}
}

func clonePlan(value Plan) Plan {
	result := value
	result.Domain = cloneDomain(value.Domain)
	result.Capabilities = cloneStrings(value.Capabilities)
	result.AffectedSurfaces = cloneStrings(value.AffectedSurfaces)
	result.DocumentationTargets = cloneDocumentationTargets(value.DocumentationTargets)
	result.Rules = append([]RuleRef{}, value.Rules...)
	result.Operations = cloneOperations(value.Operations)
	result.Preconditions = append([]Precondition{}, value.Preconditions...)
	result.ExpectedObservations = cloneObservations(value.ExpectedObservations)
	result.FingerprintInputs = FingerprintInputs{
		SelectedPaths:        cloneStrings(value.FingerprintInputs.SelectedPaths),
		SelectedDependencies: cloneStrings(value.FingerprintInputs.SelectedDependencies),
		PlannedSurfaces:      cloneStrings(value.FingerprintInputs.PlannedSurfaces),
	}

	result.AllowedEffects = AllowedEffects{
		Surfaces:     cloneStrings(value.AllowedEffects.Surfaces),
		Dependencies: append([]DependencyEffect{}, value.AllowedEffects.Dependencies...),
		Files:        append([]FileEffect{}, value.AllowedEffects.Files...),
	}
	if value.Application != nil {
		application := *value.Application
		result.Application = &application
	}

	if value.Target != nil {
		target := *value.Target
		target.Preserved = cloneTargetEntries(value.Target.Preserved)
		result.Target = &target
	}

	result.Units = cloneUnits(value.Units)
	result.Validation = cloneValidation(value.Validation)
	result.Exceptions = cloneExceptions(value.Exceptions)

	return result
}

func cloneOperations(values []Operation) []Operation {
	result := make([]Operation, len(values))
	for index, value := range values {
		result[index] = value
		result[index].Surfaces = cloneStrings(value.Surfaces)
		result[index].Rules = cloneStrings(value.Rules)
		result[index].DependsOn = cloneStrings(value.DependsOn)
		result[index].Files = append([]FileEffect{}, value.Files...)
	}

	return result
}

func cloneUnits(values []Unit) []Unit {
	result := make([]Unit, len(values))
	for index, value := range values {
		result[index] = value
		result[index].Domain = cloneDomain(value.Domain)
		result[index].Capabilities = cloneStrings(value.Capabilities)
		result[index].AffectedSurfaces = cloneStrings(value.AffectedSurfaces)
		result[index].Operations = cloneStrings(value.Operations)
		result[index].DependsOn = cloneStrings(value.DependsOn)
		result[index].Effects = append([]FileEffect{}, value.Effects...)
	}

	return result
}

func cloneValidation(values []ValidationObligation) []ValidationObligation {
	result := make([]ValidationObligation, len(values))
	for index, value := range values {
		result[index] = value
		result[index].Diagnostics = cloneStrings(value.Diagnostics)
		result[index].Surfaces = cloneStrings(value.Surfaces)
	}

	return result
}
