package project

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"hatmax.adrianpk.com/generator/book"
)

var scaffoldPaths = []string{
	".gitignore",
	"Makefile",
	"config.yaml",
	"go.mod",
	"go.sum",
	"main.go",
	"internal/application/application.go",
}

func TestInspectTargetClassifiesCreationTargets(t *testing.T) {
	selectedBook := loadTargetBook(t)

	tests := []struct {
		name      string
		prepare   func(*testing.T, string, string)
		admission TargetAdmission
		preserved int
		collision string
	}{
		{name: "absent", admission: TargetAbsent},
		{
			name: "empty",
			prepare: func(t *testing.T, _ string, target string) {
				t.Helper()

				err := os.Mkdir(target, 0o755)
				if err != nil {
					t.Fatalf("create target: %v", err)
				}
			},
			admission: TargetEmpty,
		},
		{
			name: "preservable repository",
			prepare: func(t *testing.T, _ string, target string) {
				t.Helper()
				writeProjectFile(t, target, "NOTICE", "preserve\n")
				initializeGit(t, target)
				runTestGit(t, target, "remote", "add", "origin", "git@github.com:alex/real-estate.git")
			},
			admission: TargetPreservable,
			preserved: 2,
		},
		{
			name: "planned collision",
			prepare: func(t *testing.T, _ string, target string) {
				t.Helper()
				writeProjectFile(t, target, "main.go", "package main\n")
			},
			admission: TargetIncompatible,
			collision: "main.go",
		},
		{
			name: "unversioned existing files",
			prepare: func(t *testing.T, _ string, target string) {
				t.Helper()
				writeProjectFile(t, target, "NOTICE", "not a repository\n")
			},
			admission: TargetIncompatible,
			collision: ".",
		},
		{
			name: "other Go module",
			prepare: func(t *testing.T, _ string, target string) {
				t.Helper()
				writeProjectFile(t, target, "go.mod", "module example.com/other\n\ngo 1.24.0\n")
			},
			admission: TargetIncompatible,
			collision: "go.mod",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parent := t.TempDir()

			target := filepath.Join(parent, "real-estate")
			if test.prepare != nil {
				test.prepare(t, parent, target)
			}

			inventory, err := InspectTarget(context.Background(), TargetRequest{
				Parent: parent, Target: target, PlannedPaths: scaffoldPaths, Book: selectedBook,
			})
			if err != nil {
				t.Fatalf("InspectTarget() error = %v", err)
			}

			if inventory.Admission != test.admission {
				t.Errorf("Admission = %q, want %q; collisions = %#v", inventory.Admission, test.admission, inventory.Collisions)
			}

			if test.preserved > 0 && len(inventory.Preserved) != test.preserved {
				t.Errorf("len(Preserved) = %d, want %d: %#v", len(inventory.Preserved), test.preserved, inventory.Preserved)
			}

			if test.collision != "" && !hasTargetCollision(inventory.Collisions, test.collision) {
				t.Errorf("Collisions = %#v, want path %q", inventory.Collisions, test.collision)
			}
		})
	}
}

func TestInspectTargetRecognizesCompatibleHatmaxProject(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "property")
	copyDirectory(t, filepath.Join("testdata", "supported"), target)

	inventory, err := InspectTarget(context.Background(), TargetRequest{
		Parent: parent, Target: target, PlannedPaths: scaffoldPaths, Book: loadTargetBook(t),
	})
	if err != nil {
		t.Fatalf("InspectTarget() error = %v", err)
	}

	if inventory.Admission != TargetCompatibleProject || inventory.ExistingProject == nil {
		t.Fatalf("Admission = %q, ExistingProject = %#v; want compatible project", inventory.Admission, inventory.ExistingProject)
	}
}

func TestInspectTargetDerivesCredentialFreeRemoteModule(t *testing.T) {
	parent := t.TempDir()
	writeProjectFile(t, parent, "NOTICE", "repository\n")
	initializeGit(t, parent)
	runTestGit(t, parent, "remote", "add", "origin", "https://token@example.com/alex/projects.git")

	target := filepath.Join(parent, "real-estate")

	inventory, err := InspectTarget(context.Background(), TargetRequest{
		Parent: parent, Target: target, PlannedPaths: scaffoldPaths, Book: loadTargetBook(t),
	})
	if err != nil {
		t.Fatalf("InspectTarget() error = %v", err)
	}

	if inventory.RemoteModulePath != "example.com/alex/projects/real-estate" {
		t.Errorf("RemoteModulePath = %q, want parent repository submodule path", inventory.RemoteModulePath)
	}

	if len(inventory.Remotes) != 1 || inventory.Remotes[0].Host != "example.com" || inventory.Remotes[0].Path != "alex/projects" {
		t.Errorf("Remotes = %#v, want sanitized origin evidence", inventory.Remotes)
	}
}

func TestTargetFingerprintTracksRelevantDrift(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "real-estate")

	beforeInventory, err := InspectTarget(context.Background(), TargetRequest{
		Parent: parent, Target: target, PlannedPaths: scaffoldPaths, Book: loadTargetBook(t),
	})
	if err != nil {
		t.Fatalf("InspectTarget() before error = %v", err)
	}

	before, err := beforeInventory.Fingerprint(2)
	if err != nil {
		t.Fatalf("Fingerprint() before error = %v", err)
	}

	writeProjectFile(t, target, "NOTICE", "preserve\n")

	afterInventory, err := InspectTarget(context.Background(), TargetRequest{
		Parent: parent, Target: target, PlannedPaths: scaffoldPaths, Book: loadTargetBook(t),
	})
	if err != nil {
		t.Fatalf("InspectTarget() after error = %v", err)
	}

	after, err := afterInventory.Fingerprint(2)
	if err != nil {
		t.Fatalf("Fingerprint() after error = %v", err)
	}

	changes := CompareFingerprints(before, after)
	if !hasChange(changes, ObservationTargetEntry, "NOTICE", "") {
		t.Fatalf("CompareFingerprints() = %#v, want added NOTICE target entry", changes)
	}
}

func TestInspectTargetRejectsUnsafeTargets(t *testing.T) {
	parent := t.TempDir()

	_, err := InspectTarget(context.Background(), TargetRequest{Parent: parent, Target: parent})
	requireProjectError(t, err, "target_path_invalid")

	_, err = InspectTarget(context.Background(), TargetRequest{Parent: parent, Target: filepath.Join(parent, "..", "outside")})
	requireProjectError(t, err, "target_path_invalid")

	target := filepath.Join(parent, "linked")

	err = os.Symlink(t.TempDir(), target)
	if err != nil {
		t.Fatalf("create target symlink: %v", err)
	}

	inventory, err := InspectTarget(context.Background(), TargetRequest{Parent: parent, Target: target})
	if err != nil {
		t.Fatalf("InspectTarget() symlink error = %v", err)
	}

	if inventory.Admission != TargetIncompatible {
		t.Errorf("Admission = %q, want incompatible symlink target", inventory.Admission)
	}
}

func loadTargetBook(t *testing.T) *book.Book {
	t.Helper()

	selectedBook, err := book.LoadDefault()
	if err != nil {
		t.Fatalf("book.LoadDefault() error = %v", err)
	}

	return selectedBook
}

func copyDirectory(t *testing.T, source, target string) {
	t.Helper()

	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relative, relErr := filepath.Rel(source, path)
		if relErr != nil {
			return relErr
		}

		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		return os.WriteFile(destination, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy directory: %v", err)
	}
}

func hasTargetCollision(values []TargetCollision, path string) bool {
	for _, value := range values {
		if value.Path == path {
			return true
		}
	}

	return false
}
