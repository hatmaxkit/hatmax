// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

// Package execute prepares and applies bounded edits derived from sealed
// Hatmax generator plans.
package execute

import (
	"fmt"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

// CurrentSchemaVersion is the execution manifest schema understood by this
// package.
const CurrentSchemaVersion = 4

// EditKind identifies one structured project mutation family.
type EditKind string

const (
	// EditCreateFile creates one new file from a Book-owned recipe.
	EditCreateFile EditKind = "create_file"
	// EditUpdateGo updates a named Go declaration or composition list.
	EditUpdateGo EditKind = "update_go"
	// EditUpdateSQL updates a migration or query definition.
	EditUpdateSQL EditKind = "update_sql"
	// EditUpdateTemplate updates a named server-rendered template.
	EditUpdateTemplate EditKind = "update_template"
	// EditUpdateConfiguration updates an admitted repository configuration.
	EditUpdateConfiguration EditKind = "update_configuration"
	// EditUpdateTest updates a test owned by the planned behavior.
	EditUpdateTest EditKind = "update_test"
	// EditUpdateMarkdown updates one Hatmax-managed Markdown section.
	EditUpdateMarkdown EditKind = "update_markdown"
)

// ConditionKind identifies one verifiable edit condition.
type ConditionKind string

const (
	// ConditionPathAbsent requires that a target does not exist.
	ConditionPathAbsent ConditionKind = "path_absent"
	// ConditionPathPresent requires that a target exists.
	ConditionPathPresent ConditionKind = "path_present"
	// ConditionPathDigest requires exact previously inspected content.
	ConditionPathDigest ConditionKind = "path_digest"
	// ConditionGoParses requires syntactically valid Go source.
	ConditionGoParses ConditionKind = "go_parses"
	// ConditionContentContains requires a named semantic marker.
	ConditionContentContains ConditionKind = "content_contains"
	// ConditionCommandAvailable requires a repository-owned tool.
	ConditionCommandAvailable ConditionKind = "command_available"
	// ConditionManagedOutsideDigest requires user-owned Markdown outside the
	// managed section to remain byte-for-byte unchanged.
	ConditionManagedOutsideDigest ConditionKind = "managed_outside_digest"
)

// Severity classifies an execution diagnostic.
type Severity string

const (
	// SeverityError prevents execution or reports failed execution.
	SeverityError Severity = "error"
	// SeverityWarning records a non-blocking conformance observation.
	SeverityWarning Severity = "warning"
)

// Condition is one explicit precondition or postcondition for an edit.
type Condition struct {
	Kind  ConditionKind `json:"kind" yaml:"kind"`
	Value string        `json:"value,omitempty" yaml:"value,omitempty"`
}

// ImplementationSlot identifies application-specific content a renderer may
// obtain from the sealed plan. It does not authorize a new dependency,
// surface, or target.
type ImplementationSlot struct {
	Name   string `json:"name" yaml:"name"`
	Source string `json:"source" yaml:"source"`
}

// Obligation binds one edit to a selected logical plan operation and its
// Book-owned rules. One file edit may satisfy several obligations.
type Obligation struct {
	Operation string     `json:"operation" yaml:"operation"`
	Owner     plan.Owner `json:"owner" yaml:"owner"`
	Rules     []string   `json:"rules" yaml:"rules"`
}

// Edit is one ordered, Book-owned project mutation.
type Edit struct {
	ID             string               `json:"id" yaml:"id"`
	Kind           EditKind             `json:"kind" yaml:"kind"`
	Surface        string               `json:"surface" yaml:"surface"`
	Target         string               `json:"target" yaml:"target"`
	Recipe         string               `json:"recipe" yaml:"recipe"`
	Obligations    []Obligation         `json:"obligations" yaml:"obligations"`
	DependsOn      []string             `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Preconditions  []Condition          `json:"preconditions" yaml:"preconditions"`
	Postconditions []Condition          `json:"postconditions" yaml:"postconditions"`
	Slots          []ImplementationSlot `json:"implementation_slots,omitempty" yaml:"implementation_slots,omitempty"`
}

// Command is one bounded repository-owned command selected for execution.
type Command struct {
	Kind             project.CommandKind `json:"kind" yaml:"kind"`
	Name             string              `json:"name" yaml:"name"`
	Args             []string            `json:"args" yaml:"args"`
	WorkingDirectory string              `json:"working_directory" yaml:"working_directory"`
	Source           string              `json:"source" yaml:"source"`
}

// Manifest is the complete deterministic mutation contract prepared from one
// sealed plan and one compatible project inventory.
type Manifest struct {
	SchemaVersion      int              `json:"schema_version" yaml:"schema_version"`
	PlanDigest         string           `json:"plan_digest" yaml:"plan_digest"`
	ProjectFingerprint string           `json:"project_fingerprint,omitempty" yaml:"project_fingerprint,omitempty"`
	SourceFingerprint  string           `json:"source_fingerprint,omitempty" yaml:"source_fingerprint,omitempty"`
	Intent             intent.Operation `json:"intent" yaml:"intent"`
	Feature            string           `json:"feature" yaml:"feature"`
	TargetPath         string           `json:"target_path,omitempty" yaml:"target_path,omitempty"`
	PreservedPaths     []string         `json:"preserved_paths,omitempty" yaml:"preserved_paths,omitempty"`
	AllowedSurfaces    []string         `json:"allowed_surfaces" yaml:"allowed_surfaces"`
	Edits              []Edit           `json:"edits" yaml:"edits"`
	Commands           []Command        `json:"commands,omitempty" yaml:"commands,omitempty"`
	Digest             string           `json:"digest,omitempty" yaml:"digest,omitempty"`
}

// Diagnostic describes one stable execution or conformance failure.
type Diagnostic struct {
	Code                 string   `json:"code" yaml:"code"`
	Rule                 string   `json:"rule,omitempty" yaml:"rule,omitempty"`
	Severity             Severity `json:"severity" yaml:"severity"`
	Surface              string   `json:"surface,omitempty" yaml:"surface,omitempty"`
	Location             string   `json:"location,omitempty" yaml:"location,omitempty"`
	Observed             string   `json:"observed" yaml:"observed"`
	Expected             string   `json:"expected" yaml:"expected"`
	RepairableWithinPlan bool     `json:"repairable_within_plan" yaml:"repairable_within_plan"`
}

// Error describes an invalid or incompatible execution contract.
type Error struct {
	Code    string
	Path    string
	Message string
}

func (e Error) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}

	return fmt.Sprintf("%s at %s: %s", e.Code, e.Path, e.Message)
}

func executionError(code, path, format string, arguments ...any) error {
	return Error{
		Code:    code,
		Path:    path,
		Message: fmt.Sprintf(format, arguments...),
	}
}
