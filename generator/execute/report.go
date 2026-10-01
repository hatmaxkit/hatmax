// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package execute

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

const maximumCommandEvidenceBytes = 8 << 10

// CommandEvidence records one exact repository-owned command result.
type CommandEvidence struct {
	Name             string              `json:"name" yaml:"name"`
	Kind             project.CommandKind `json:"kind" yaml:"kind"`
	Args             []string            `json:"args" yaml:"args"`
	WorkingDirectory string              `json:"working_directory" yaml:"working_directory"`
	ExitCode         int                 `json:"exit_code" yaml:"exit_code"`
	Output           string              `json:"output,omitempty" yaml:"output,omitempty"`
}

// ExecutionReport records conformance and repository validation for one
// executed manifest without becoming project source of truth.
type ExecutionReport struct {
	PlanDigest     string                  `json:"plan_digest" yaml:"plan_digest"`
	ManifestDigest string                  `json:"manifest_digest" yaml:"manifest_digest"`
	BookVersion    int                     `json:"book_version" yaml:"book_version"`
	HatmaxVersion  string                  `json:"hatmax_version" yaml:"hatmax_version"`
	Surfaces       []string                `json:"surfaces" yaml:"surfaces"`
	Dependencies   []plan.DependencyEffect `json:"dependencies" yaml:"dependencies"`
	Conformance    ConformanceResult       `json:"conformance" yaml:"conformance"`
	Commands       []CommandEvidence       `json:"commands" yaml:"commands"`
	Repairs        []string                `json:"repairs" yaml:"repairs"`
	Warnings       []string                `json:"warnings" yaml:"warnings"`
}

type commandRunner func(context.Context, string, Command) (CommandEvidence, error)

// ValidateExecution checks independent conformance and then executes only the
// exact repository-owned commands sealed in the manifest.
func ValidateExecution(
	ctx context.Context,
	value plan.Plan,
	manifest Manifest,
	inventory project.Inventory,
) (ExecutionReport, error) {
	return validateExecutionWithRunner(ctx, value, manifest, inventory, runRepositoryCommand)
}

func validateExecutionWithRunner(
	ctx context.Context,
	value plan.Plan,
	manifest Manifest,
	inventory project.Inventory,
	runner commandRunner,
) (ExecutionReport, error) {
	conformance, err := CheckConformance(value, manifest, inventory)
	if err != nil {
		return ExecutionReport{}, err
	}

	report := ExecutionReport{
		PlanDigest: value.Digest, ManifestDigest: manifest.Digest,
		BookVersion: value.BookVersion, HatmaxVersion: value.HatmaxVersion,
		Surfaces:     append([]string{}, manifest.AllowedSurfaces...),
		Dependencies: append([]plan.DependencyEffect{}, value.AllowedEffects.Dependencies...),
		Conformance:  conformance,
		Commands:     []CommandEvidence{}, Repairs: []string{}, Warnings: []string{},
	}

	if !conformance.Passed {
		return report, executionError("execution_conformance_failed", "conformance", "%d required Book rules failed", len(conformance.Diagnostics))
	}

	err = validateManifestCommands(manifest.Commands, inventory.Commands)
	if err != nil {
		return report, err
	}

	commands := orderedCommands(manifest.Commands)
	for _, command := range commands {
		evidence, runErr := runner(ctx, inventory.Root, command)
		report.Commands = append(report.Commands, evidence)

		if runErr != nil {
			return report, executionError("execution_command_failed", command.Name, "%v", runErr)
		}
	}

	return report, nil
}

func validateManifestCommands(commands []Command, available []project.Command) error {
	for _, command := range commands {
		matched := false

		for _, candidate := range available {
			if command.Kind == candidate.Kind && command.Source == candidate.Source && sameStrings(command.Args, candidate.Args) {
				matched = true

				break
			}
		}

		if !matched {
			return executionError("execution_command_undeclared", command.Name, "command is not present in the current repository policy")
		}
	}

	return nil
}

func orderedCommands(commands []Command) []Command {
	result := append([]Command{}, commands...)
	priority := map[project.CommandKind]int{
		project.CommandGeneration: 0,
		project.CommandFormatting: 1,
		project.CommandValidation: 2,
	}

	sort.SliceStable(result, func(left, right int) bool {
		leftPriority := priority[result[left].Kind]

		rightPriority := priority[result[right].Kind]
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}

		return result[left].Name < result[right].Name
	})

	return result
}

func runRepositoryCommand(ctx context.Context, root string, command Command) (CommandEvidence, error) {
	workingDirectory := filepath.Join(root, filepath.FromSlash(command.WorkingDirectory))
	evidence := CommandEvidence{
		Name: command.Name, Kind: command.Kind, Args: append([]string{}, command.Args...),
		WorkingDirectory: command.WorkingDirectory, ExitCode: -1,
	}

	if len(command.Args) == 0 || strings.ContainsAny(command.Args[0], `/\\`) {
		return evidence, errors.New("command executable must be an unqualified repository-owned tool name")
	}

	process := exec.CommandContext(ctx, command.Args[0], command.Args[1:]...)
	process.Dir = workingDirectory

	var output commandOutput

	process.Stdout = &output
	process.Stderr = &output

	err := process.Run()

	evidence.Output = output.String()
	if process.ProcessState != nil {
		evidence.ExitCode = process.ProcessState.ExitCode()
	}

	return evidence, err
}
