// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package execute

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"hatmax.adrianpk.com/generator/intent"
)

func TestWorkspaceStagesBoundedCreateWithoutWriting(t *testing.T) {
	root := copyExecutionFixture(t)
	value, inventory, selectedBook := executionPlan(t, root, intent.OperationCreateFeature, intent.Domain{
		Entity: "Invoice",
		Route:  "/invoices",
		Fields: []intent.Field{{Name: "number", Type: "string"}},
	}, []string{"postgres_persistence"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	workspace, err := OpenWorkspace(context.Background(), manifest, value, inventory)
	if err != nil {
		t.Fatalf("OpenWorkspace() error = %v", err)
	}

	change, err := workspace.Stage(Mutation{
		EditID:  "create.model",
		Kind:    MutationCreate,
		Content: []byte("package invoice\n"),
	})
	if err != nil {
		t.Fatalf("Stage() error = %v", err)
	}

	if change.Status != ChangeStaged || change.Target != "internal/feat/invoice/model.go" || change.BeforeDigest != "missing" {
		t.Errorf("Stage() = %#v, want staged create metadata", change)
	}

	_, err = os.Stat(filepath.Join(root, "internal", "feat", "invoice", "model.go"))
	if !os.IsNotExist(err) {
		t.Errorf("Stage() changed project, stat error = %v", err)
	}

	changes := workspace.Changes()
	if len(changes) != 1 || changes[0] != change {
		t.Errorf("Changes() = %#v, want staged change", changes)
	}
}

func TestWorkspaceStagesSemanticInsertion(t *testing.T) {
	root := copyExecutionFixture(t)
	value, inventory, selectedBook := executionPlan(t, root, intent.OperationCreateFeature, intent.Domain{
		Entity: "Invoice",
		Route:  "/invoices",
		Fields: []intent.Field{{Name: "number", Type: "string"}},
	}, []string{"postgres_persistence"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	workspace, err := OpenWorkspace(context.Background(), manifest, value, inventory)
	if err != nil {
		t.Fatalf("OpenWorkspace() error = %v", err)
	}

	change, err := workspace.Stage(Mutation{
		EditID:   "create.wiring",
		Kind:     MutationInsert,
		Anchor:   []byte("func main() {"),
		Position: InsertBefore,
		Content:  []byte("func buildInvoice() {}\n\n"),
	})
	if err != nil {
		t.Fatalf("Stage() error = %v", err)
	}

	if change.Status != ChangeStaged || change.Kind != MutationInsert {
		t.Errorf("Stage() = %#v, want staged insertion", change)
	}

	original, err := os.ReadFile(filepath.Join(root, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}

	if string(original) != "package main\n\nimport (\n\t\"context\"\n\n\t\"hatmax.adrianpk.com/app\"\n)\n\nfunc main() {\n\tapp.Setup(context.Background(), nil)\n}\n" {
		t.Error("Stage() modified insertion target")
	}
}

func TestWorkspaceRejectsUndeclaredAndInvalidMutations(t *testing.T) {
	root := copyExecutionFixture(t)
	value, inventory, selectedBook := executionPlan(t, root, intent.OperationCreateFeature, intent.Domain{
		Entity: "Invoice",
		Route:  "/invoices",
		Fields: []intent.Field{{Name: "number", Type: "string"}},
	}, []string{"postgres_persistence"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	workspace, err := OpenWorkspace(context.Background(), manifest, value, inventory)
	if err != nil {
		t.Fatalf("OpenWorkspace() error = %v", err)
	}

	_, err = workspace.Stage(Mutation{EditID: "outside.manifest", Kind: MutationCreate, Content: []byte("data")})
	requireExecutionCode(t, err, "execution_edit_undeclared")

	_, err = workspace.Stage(Mutation{EditID: "create.model", Kind: MutationReplace, Content: []byte("package invoice\n")})
	requireExecutionCode(t, err, "execution_mutation_invalid")

	_, err = workspace.Stage(Mutation{EditID: "create.model", Kind: MutationCreate, Content: []byte("not Go")})
	requireExecutionCode(t, err, "execution_postcondition_failed")
}

func TestOpenWorkspaceRejectsProjectDrift(t *testing.T) {
	root := copyExecutionFixture(t)
	value, inventory, selectedBook := executionPlan(t, root, intent.OperationCreateFeature, intent.Domain{
		Entity: "Invoice",
		Route:  "/invoices",
		Fields: []intent.Field{{Name: "number", Type: "string"}},
	}, []string{"postgres_persistence"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	appendExecutionFile(t, root, "internal/feat/property/model.go", "\n// concurrent drift\n")

	_, err = OpenWorkspace(context.Background(), manifest, value, inventory)
	requireExecutionCode(t, err, "execution_plan_stale")
}
