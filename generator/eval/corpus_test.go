package eval

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

const corpusRoot = "testdata/corpus/v1"

type corpus struct {
	SchemaVersion int         `yaml:"schema_version"`
	CorpusVersion int         `yaml:"corpus_version"`
	TypedCases    []typedCase `yaml:"typed_cases"`
	StaleCases    []staleCase `yaml:"stale_cases"`
}

type typedCase struct {
	ID       string       `yaml:"id"`
	Category string       `yaml:"category"`
	Intent   string       `yaml:"intent"`
	Expected expectedCase `yaml:"expected"`
}

type expectedCase struct {
	Status          string   `yaml:"status"`
	EquivalentGroup string   `yaml:"equivalent_group,omitempty"`
	Diagnostics     []string `yaml:"diagnostics,omitempty"`
	Clarifications  []string `yaml:"clarifications,omitempty"`
}

type staleCase struct {
	ID                  string   `yaml:"id"`
	Intent              string   `yaml:"intent"`
	ChangedPath         string   `yaml:"changed_path"`
	ExpectedDiagnostics []string `yaml:"expected_diagnostics"`
}

func TestPlanningCorpusIsVersionedAndComplete(t *testing.T) {
	loaded := loadCorpus(t)
	if loaded.SchemaVersion != 1 || loaded.CorpusVersion != 1 {
		t.Fatalf("corpus versions = schema %d, corpus %d; want 1 and 1", loaded.SchemaVersion, loaded.CorpusVersion)
	}

	wantCategories := map[string]bool{
		"admitted":     false,
		"ambiguous":    false,
		"unsupported":  false,
		"incompatible": false,
		"invalid":      false,
	}
	seenIDs := make(map[string]struct{})
	equivalentGroups := make(map[string]int)

	for _, testCase := range loaded.TypedCases {
		assertCorpusID(t, seenIDs, testCase.ID)

		if _, exists := wantCategories[testCase.Category]; !exists {
			t.Errorf("typed case %q has unknown category %q", testCase.ID, testCase.Category)
		} else {
			wantCategories[testCase.Category] = true
		}

		assertCorpusFixture(t, testCase.Intent)

		if testCase.Expected.Status == "" {
			t.Errorf("typed case %q has no expected status", testCase.ID)
		}

		if testCase.Expected.EquivalentGroup != "" {
			equivalentGroups[testCase.Expected.EquivalentGroup]++
		}
	}

	for category, present := range wantCategories {
		if !present {
			t.Errorf("corpus has no %s typed case", category)
		}
	}

	for group, count := range equivalentGroups {
		if count < 2 {
			t.Errorf("equivalent group %q has %d case, want at least 2", group, count)
		}
	}

	if len(loaded.StaleCases) == 0 {
		t.Fatal("corpus has no stale-plan cases")
	}

	for _, testCase := range loaded.StaleCases {
		assertCorpusID(t, seenIDs, testCase.ID)
		assertCorpusFixture(t, testCase.Intent)

		if testCase.ChangedPath == "" || len(testCase.ExpectedDiagnostics) == 0 {
			t.Errorf("stale case %q is incomplete", testCase.ID)
		}
	}
}

func loadCorpus(t *testing.T) corpus {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(corpusRoot, "manifest.yaml"))
	if err != nil {
		t.Fatalf("read planning corpus: %v", err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var result corpus

	err = decoder.Decode(&result)
	if err != nil {
		t.Fatalf("decode planning corpus: %v", err)
	}

	return result
}

func assertCorpusID(t *testing.T, seen map[string]struct{}, id string) {
	t.Helper()

	if id == "" {
		t.Error("corpus case has empty ID")

		return
	}

	if _, exists := seen[id]; exists {
		t.Errorf("duplicate corpus case ID %q", id)
	}

	seen[id] = struct{}{}
}

func assertCorpusFixture(t *testing.T, path string) {
	t.Helper()

	if path == "" || filepath.IsAbs(path) || filepath.Clean(path) != filepath.FromSlash(path) {
		t.Errorf("invalid corpus fixture path %q", path)

		return
	}

	info, err := os.Stat(filepath.Join(corpusRoot, filepath.FromSlash(path)))
	if err != nil {
		t.Errorf("corpus fixture %q: %v", path, err)

		return
	}

	if !info.Mode().IsRegular() {
		t.Errorf("corpus fixture %q is not a regular file", path)
	}
}

func typedCaseByID(t *testing.T, loaded corpus, id string) typedCase {
	t.Helper()

	for _, testCase := range loaded.TypedCases {
		if testCase.ID == id {
			return testCase
		}
	}

	t.Fatalf("typed corpus case %q not found", id)

	return typedCase{}
}

func readCorpusIntent(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(corpusRoot, filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("read corpus intent %q: %v", path, err)
	}

	return data
}

func corpusCaseName(index int, id string) string {
	return fmt.Sprintf("%02d_%s", index, id)
}
