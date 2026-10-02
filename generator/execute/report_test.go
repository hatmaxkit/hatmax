// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package execute

import (
	"context"
	"errors"
	"testing"

	"hatmax.adrianpk.com/generator/project"
)

func TestValidateExecutionRunsOnlySealedCommandsInCanonicalOrder(t *testing.T) {
	root, value, manifest := appliedAddFieldProject(t)

	inventory, err := project.Inspect(context.Background(), root)
	if err != nil {
		t.Fatalf("project.Inspect() error = %v", err)
	}

	run := make([]project.CommandKind, 0, len(manifest.Commands))
	runner := func(_ context.Context, _ string, command Command) (CommandEvidence, error) {
		run = append(run, command.Kind)

		return CommandEvidence{
			Name: command.Name, Kind: command.Kind, Args: command.Args,
			WorkingDirectory: command.WorkingDirectory, ExitCode: 0,
		}, nil
	}

	report, err := validateExecutionWithRunner(context.Background(), value, manifest, inventory, runner)
	if err != nil {
		t.Fatalf("validateExecutionWithRunner() error = %v", err)
	}

	if !report.Conformance.Passed || len(report.Commands) != len(manifest.Commands) {
		t.Fatalf("report = %#v", report)
	}

	for index := 1; index < len(run); index++ {
		if commandKindPriority(run[index-1]) > commandKindPriority(run[index]) {
			t.Errorf("command kinds = %v, want generation, formatting, validation order", run)

			break
		}
	}
}

func TestValidateExecutionRejectsCommandOutsideCurrentRepositoryPolicy(t *testing.T) {
	root, value, manifest := appliedAddFieldProject(t)

	inventory, err := project.Inspect(context.Background(), root)
	if err != nil {
		t.Fatalf("project.Inspect() error = %v", err)
	}

	manifest.Commands = append(manifest.Commands, Command{
		Kind: project.CommandValidation, Name: "validation.undeclared", Args: []string{"go", "test", "./..."},
		WorkingDirectory: ".", Source: "injected",
	})
	manifest.Digest = ""

	manifest, err = Seal(manifest)
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}

	_, err = validateExecutionWithRunner(context.Background(), value, manifest, inventory, func(context.Context, string, Command) (CommandEvidence, error) {
		return CommandEvidence{}, errors.New("must not run")
	})
	requireExecutionCode(t, err, "execution_command_undeclared")
}

func TestValidateExecutionStopsAndReportsFailedCommand(t *testing.T) {
	root, value, manifest := appliedAddFieldProject(t)

	inventory, err := project.Inspect(context.Background(), root)
	if err != nil {
		t.Fatalf("project.Inspect() error = %v", err)
	}

	report, err := validateExecutionWithRunner(context.Background(), value, manifest, inventory, func(_ context.Context, _ string, command Command) (CommandEvidence, error) {
		return CommandEvidence{Name: command.Name, Kind: command.Kind, ExitCode: 2}, errors.New("command failed")
	})
	requireExecutionCode(t, err, "execution_command_failed")

	if len(report.Commands) != 1 || report.Commands[0].ExitCode != 2 {
		t.Errorf("report commands = %#v, want one failed command", report.Commands)
	}
}

func commandKindPriority(kind project.CommandKind) int {
	switch kind {
	case project.CommandGeneration:
		return 0
	case project.CommandFormatting:
		return 1
	default:
		return 2
	}
}
