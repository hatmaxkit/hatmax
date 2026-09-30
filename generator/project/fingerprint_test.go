// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package project

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var standardFingerprintRequest = FingerprintRequest{
	BookVersion:          1,
	SelectedPaths:        []string{"config.yaml"},
	SelectedDependencies: []string{"github.com/go-chi/chi/v5"},
	PlannedSurfaces:      []string{"handler", "model"},
}

func TestFingerprintIsDeterministicAndScoped(t *testing.T) {
	root := copyFixture(t, "supported")

	initial := inspectAndFingerprint(t, root, standardFingerprintRequest)
	repeated := inspectAndFingerprint(t, root, standardFingerprintRequest)

	if initial.Value != repeated.Value {
		t.Fatalf("repeated fingerprint = %q, want %q", repeated.Value, initial.Value)
	}

	if !strings.HasPrefix(initial.Value, "sha256:") {
		t.Errorf("Fingerprint.Value = %q, want sha256 prefix", initial.Value)
	}

	for index := 1; index < len(initial.Observations); index++ {
		if initial.Observations[index-1].Key > initial.Observations[index].Key {
			t.Fatalf("observations are not sorted at index %d", index)
		}
	}

	appendProjectFile(t, root, "README.md", "\nUnrelated change.\n")
	unrelatedFile := inspectAndFingerprint(t, root, standardFingerprintRequest)

	if unrelatedFile.Value != initial.Value {
		t.Error("unrelated file change invalidated fingerprint")
	}

	appendProjectFile(t, root, "go.mod", "\nrequire example.com/unrelated v1.0.0\n")
	unrelatedDependency := inspectAndFingerprint(t, root, standardFingerprintRequest)

	if unrelatedDependency.Value != initial.Value {
		t.Error("unselected dependency change invalidated fingerprint")
	}
}

func TestFingerprintClassifiesRelevantDrift(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(*testing.T, string)
		request     FingerprintRequest
		wantClass   ObservationClass
		wantPath    string
		wantSurface string
	}{
		{
			name: "selected input",
			mutate: func(t *testing.T, root string) {
				appendProjectFile(t, root, "config.yaml", "\nchanged: true\n")
			},
			request:   standardFingerprintRequest,
			wantClass: ObservationSelectedInput,
			wantPath:  "config.yaml",
		},
		{
			name: "planned surface",
			mutate: func(t *testing.T, root string) {
				appendProjectFile(t, root, "internal/feat/property/model.go", "\n// changed\n")
			},
			request:     standardFingerprintRequest,
			wantClass:   ObservationPlannedSurface,
			wantPath:    "internal/feat/property/model.go",
			wantSurface: "model",
		},
		{
			name: "selected dependency",
			mutate: func(t *testing.T, root string) {
				replaceProjectText(t, root, "go.mod", "github.com/go-chi/chi/v5 v5.2.3", "github.com/go-chi/chi/v5 v5.2.4")
			},
			request:   standardFingerprintRequest,
			wantClass: ObservationModule,
			wantPath:  "go.mod",
		},
		{
			name: "repository instructions",
			mutate: func(t *testing.T, root string) {
				appendProjectFile(t, root, "AGENTS.md", "\nNew rule.\n")
			},
			request:   standardFingerprintRequest,
			wantClass: ObservationRepositoryRule,
			wantPath:  "AGENTS.md",
		},
		{
			name: "command definition",
			mutate: func(t *testing.T, root string) {
				replaceProjectText(t, root, "Makefile", "go test ./...", "go test -race ./...")
			},
			request:   standardFingerprintRequest,
			wantClass: ObservationCommand,
			wantPath:  "Makefile",
		},
		{
			name:   "Book version",
			mutate: func(t *testing.T, root string) {},
			request: FingerprintRequest{
				BookVersion:          2,
				SelectedPaths:        standardFingerprintRequest.SelectedPaths,
				SelectedDependencies: standardFingerprintRequest.SelectedDependencies,
				PlannedSurfaces:      standardFingerprintRequest.PlannedSurfaces,
			},
			wantClass: ObservationBook,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := copyFixture(t, "supported")
			before := inspectAndFingerprint(t, root, standardFingerprintRequest)

			test.mutate(t, root)

			after := inspectAndFingerprint(t, root, test.request)
			changes := CompareFingerprints(before, after)

			if !hasChange(changes, test.wantClass, test.wantPath, test.wantSurface) {
				t.Fatalf("CompareFingerprints() = %#v, want class=%q path=%q surface=%q", changes, test.wantClass, test.wantPath, test.wantSurface)
			}
		})
	}
}

func TestFingerprintTracksMissingSelectedInput(t *testing.T) {
	root := copyFixture(t, "supported")
	request := FingerprintRequest{BookVersion: 1, SelectedPaths: []string{"optional.yaml"}}

	before := inspectAndFingerprint(t, root, request)
	writeProjectFile(t, root, "optional.yaml", "enabled: true\n")
	after := inspectAndFingerprint(t, root, request)

	changes := CompareFingerprints(before, after)
	if len(changes) != 1 || changes[0].Kind != ChangeModified || changes[0].Path != "optional.yaml" {
		t.Fatalf("CompareFingerprints() = %#v, want one modified optional.yaml", changes)
	}
}

func TestFingerprintTracksSelectedDirectory(t *testing.T) {
	root := copyFixture(t, "supported")
	request := FingerprintRequest{BookVersion: 1, SelectedPaths: []string{"assets/templates"}}

	before := inspectAndFingerprint(t, root, request)
	writeProjectFile(t, root, "assets/templates/property/form.html", "<form></form>\n")
	after := inspectAndFingerprint(t, root, request)

	changes := CompareFingerprints(before, after)
	if !hasChange(changes, ObservationSelectedInput, "assets/templates/property/form.html", "") {
		t.Fatalf("CompareFingerprints() = %#v, want added selected directory input", changes)
	}
}

func TestCompareFingerprintsClassifiesRemovedObservation(t *testing.T) {
	before := Fingerprint{Observations: []Observation{{
		Key:    "file:removed.go",
		Class:  ObservationSelectedInput,
		Path:   "removed.go",
		Digest: "sha256:before",
	}}}
	after := Fingerprint{}

	changes := CompareFingerprints(before, after)
	if len(changes) != 1 || changes[0].Kind != ChangeRemoved || changes[0].Path != "removed.go" {
		t.Fatalf("CompareFingerprints() = %#v, want one removed observation", changes)
	}
}

func TestFingerprintIncludesDirtyOverlapOnly(t *testing.T) {
	root := copyFixture(t, "supported")
	initializeGit(t, root)

	clean := inspectAndFingerprint(t, root, standardFingerprintRequest)
	appendProjectFile(t, root, "README.md", "\nUnrelated dirty change.\n")
	unrelated := inspectAndFingerprint(t, root, standardFingerprintRequest)

	if unrelated.Value != clean.Value {
		t.Error("unrelated dirty path invalidated fingerprint")
	}

	appendProjectFile(t, root, "internal/feat/property/model.go", "\n// dirty model\n")
	relevant := inspectAndFingerprint(t, root, standardFingerprintRequest)

	if !hasChange(CompareFingerprints(unrelated, relevant), ObservationDirtyOverlap, "internal/feat/property/model.go", "") {
		t.Error("relevant dirty path was not classified as dirty overlap")
	}
}

func TestFingerprintRejectsInvalidSelection(t *testing.T) {
	root := copyFixture(t, "supported")

	inventory, err := Inspect(context.Background(), root)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	tests := []struct {
		name    string
		request FingerprintRequest
		code    string
	}{
		{name: "Book version", request: FingerprintRequest{}, code: "project_book_version_invalid"},
		{name: "escaping path", request: FingerprintRequest{BookVersion: 1, SelectedPaths: []string{"../outside"}}, code: "project_path_invalid"},
		{name: "unknown surface", request: FingerprintRequest{BookVersion: 1, PlannedSurfaces: []string{"unknown"}}, code: "project_surface_unknown"},
		{name: "invalid dependency", request: FingerprintRequest{BookVersion: 1, SelectedDependencies: []string{"invalid module"}}, code: "project_dependency_invalid"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, fingerprintErr := inventory.Fingerprint(test.request)
			requireProjectError(t, fingerprintErr, test.code)
		})
	}

	symlink := filepath.Join(root, "selected-link")

	err = os.Symlink(filepath.Join(root, "config.yaml"), symlink)
	if err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	inventory, err = Inspect(context.Background(), root)
	if err != nil {
		t.Fatalf("Inspect() after symlink error = %v", err)
	}

	_, err = inventory.Fingerprint(FingerprintRequest{BookVersion: 1, SelectedPaths: []string{"selected-link"}})
	requireProjectError(t, err, "project_path_unsupported")

	limited, err := InspectWithOptions(context.Background(), root, Options{MaximumFileSize: 4})
	if err != nil {
		t.Fatalf("InspectWithOptions() error = %v", err)
	}

	_, err = limited.Fingerprint(FingerprintRequest{BookVersion: 1, SelectedPaths: []string{"config.yaml"}})
	requireProjectError(t, err, "project_limit_exceeded")
}

func inspectAndFingerprint(t *testing.T, root string, request FingerprintRequest) Fingerprint {
	t.Helper()

	inventory, err := Inspect(context.Background(), root)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	fingerprint, err := inventory.Fingerprint(request)
	if err != nil {
		t.Fatalf("Fingerprint() error = %v", err)
	}

	return fingerprint
}

func replaceProjectText(t *testing.T, root, path, oldValue, newValue string) {
	t.Helper()

	absolute := filepath.Join(root, filepath.FromSlash(path))

	data, err := os.ReadFile(absolute)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}

	content := string(data)
	if strings.Count(content, oldValue) != 1 {
		t.Fatalf("%q has %d occurrences of %q, want 1", path, strings.Count(content, oldValue), oldValue)
	}

	writeProjectFile(t, root, path, strings.Replace(content, oldValue, newValue, 1))
}

func hasChange(changes []Change, class ObservationClass, path, surface string) bool {
	for _, change := range changes {
		if change.Class == class && change.Path == path && change.Surface == surface {
			return true
		}
	}

	return false
}
