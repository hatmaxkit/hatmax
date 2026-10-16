// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
)

type manifest struct {
	Version                                    int
	Mode, Head, Tree, Branch, Origin, Source   string
	Go, SDK, S3SDK, Ledger, EvidenceExecutable string
	Roots, RepositoryRoots                     []root
}

type step struct {
	Index             int
	Group, Label, Log string
	Exit              int
	Arguments         []string
	SHA256            string
}

type result struct {
	Criterion string
	Outcome   string
	Roots     []root
}

type receipt struct {
	ManifestSHA256 string
	Status         string
	Steps          []step
	Results        []result
	DeclaredSkips  []string
	Artifacts      map[string]string
}

func output(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, name, args...)
	command.WaitDelay = 5 * time.Second

	data, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("evidence identity command %s: %w", name, err)
	}

	return strings.TrimSpace(string(data)), nil
}

func hashFile(name string) (string, error) {
	file, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()

	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func sourceDigest() (string, error) {
	listing, err := output("git", "ls-files", "-z")
	if err != nil {
		return "", err
	}

	hash := sha256.New()

	for _, name := range strings.Split(strings.TrimSuffix(listing, "\x00"), "\x00") {
		digest, err := hashFile(name)
		if err != nil {
			return "", err
		}

		_, _ = fmt.Fprintf(hash, "%s\x00%s\n", name, digest)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func identity(mode string) (manifest, error) {
	m := manifest{Version: 1, Mode: mode, Go: runtime.Version()}
	if mode != "slice-5" && mode != "full" || m.Go != "go1.27.1" {
		return m, fmt.Errorf("unsupported evidence mode or toolchain")
	}

	commands := []struct {
		target *string
		args   []string
	}{
		{&m.Head, []string{"rev-parse", "HEAD"}},
		{&m.Tree, []string{"rev-parse", "HEAD^{tree}"}},
		{&m.Branch, []string{"branch", "--show-current"}},
		{&m.Origin, []string{"remote", "get-url", "origin"}},
	}
	for _, command := range commands {
		value, err := output("git", command.args...)
		if err != nil {
			return m, err
		}

		*command.target = value
	}

	status, err := output("git", "status", "--porcelain", "--untracked-files=all")
	if err != nil || status != "" || m.Origin != "ssh://forge@zeus:2222/hatmax/hatmax.git" || m.Branch == "" || mode == "full" && m.Branch != "dev" {
		return m, fmt.Errorf("evidence requires a clean canonical candidate and the full gate requires dev")
	}

	remote, err := output("git", "ls-remote", "origin", "refs/heads/"+m.Branch)
	if err != nil || remote != m.Head+"\trefs/heads/"+m.Branch {
		return m, fmt.Errorf("candidate is not the exact pushed branch head")
	}

	m.Source, err = sourceDigest()
	if err != nil {
		return m, err
	}

	m.Ledger, err = hashFile("scripts/object-storage/ledger.go")
	if err != nil {
		return m, err
	}

	executable, err := os.Executable()
	if err != nil {
		return m, err
	}

	m.EvidenceExecutable, err = hashFile(executable)
	if err != nil {
		return m, err
	}

	m.SDK, err = output("go", "list", "-m", "-f", "{{.Version}}", "github.com/aws/aws-sdk-go-v2")
	if err != nil {
		return m, err
	}

	m.S3SDK, err = output("go", "list", "-m", "-f", "{{.Version}}", "github.com/aws/aws-sdk-go-v2/service/s3")
	if err != nil {
		return m, err
	}

	m.Roots, err = discoverRoots()
	if err != nil {
		return m, err
	}

	err = checkLedger(m.Roots)
	if err != nil {
		return m, err
	}

	if mode == "full" {
		m.RepositoryRoots, err = listedRoots("", []string{"./..."})
	}

	return m, err
}

func writeJSON(name string, value any) error {
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(value)
	closeErr := file.Close()

	if err != nil {
		return err
	}

	return closeErr
}

func readJSON(name string, value any) error {
	file, err := os.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(io.LimitReader(file, 16<<20)).Decode(value)
}

func evidenceDirectory(directory string) (string, error) {
	abs, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return "", err
	}

	abs, err = filepath.Abs(abs)
	if err != nil {
		return "", err
	}

	root, err := output("git", "rev-parse", "--show-toplevel")
	if err != nil || !strings.HasPrefix(abs, filepath.Join(root, ".tmp", "object-storage")+string(os.PathSeparator)) {
		return "", fmt.Errorf("evidence requires an owned ignored run directory")
	}

	_, err = output("git", "check-ignore", filepath.Join(abs, "manifest.json"))
	if err != nil {
		return "", fmt.Errorf("evidence output must be ignored")
	}

	return abs, nil
}

func beginEvidence(directory, mode string) error {
	directory, err := evidenceDirectory(directory)
	if err != nil {
		return err
	}

	m, err := identity(mode)
	if err != nil {
		return err
	}

	return writeJSON(filepath.Join(directory, "manifest.json"), m)
}

func recordStep(directory, group, label, status, log string, arguments []string) error {
	directory, err := evidenceDirectory(directory)
	if err != nil {
		return err
	}

	path, err := filepath.Abs(log)
	if err != nil || !strings.HasPrefix(path, directory+string(os.PathSeparator)) || len(arguments) == 0 {
		return fmt.Errorf("step requires exact arguments and an owned log")
	}

	exit, err := strconv.Atoi(status)
	if err != nil {
		return err
	}

	files, err := filepath.Glob(filepath.Join(directory, "step-*.json"))
	if err != nil || len(files) >= 128 {
		return fmt.Errorf("invalid or excessive evidence step count")
	}

	digest, err := hashFile(path)
	if err != nil {
		return err
	}

	s := step{Index: len(files) + 1, Group: group, Label: label, Log: path, Exit: exit, Arguments: arguments, SHA256: digest}

	return writeJSON(filepath.Join(directory, fmt.Sprintf("step-%03d.json", s.Index)), s)
}

type testEvent struct{ Action, Package, Test string }

func declaredSkip(event testEvent) bool {
	key := strings.TrimPrefix(event.Package, "hatmax.adrianpk.com/") + ":" + event.Test

	return key == "generator/backend/codex:TestAuthenticatedLiveSmoke" || key == "scheduler:TestPanicProcess" ||
		strings.HasPrefix(key, "htmx:FuzzOOBEscape/") || strings.HasPrefix(key, "ui:FuzzUIURLs/")
}

func rootResults(reader io.Reader, roots []root, repository bool) ([]string, error) {
	statuses := make(map[string]string)

	var declared []string

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64<<10), 8<<20)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 || line[0] != '{' {
			continue
		}

		var event testEvent

		err := json.Unmarshal(line, &event)
		if err != nil {
			return nil, fmt.Errorf("malformed test result event")
		}

		if event.Package == "" || event.Action == "" {
			continue
		}

		if event.Action == "fail" {
			return nil, fmt.Errorf("failed result: %s %s", event.Package, event.Test)
		}

		if event.Test == "" {
			continue // No-test packages are compilation outcomes, not skipped roots.
		}

		key := strings.TrimPrefix(event.Package, "hatmax.adrianpk.com/") + ":" + event.Test
		if event.Action == "skip" {
			if !repository || !declaredSkip(event) {
				return nil, fmt.Errorf("unexpected required skip: %s", key)
			}

			declared = append(declared, key)
		}

		if event.Action == "pass" || event.Action == "skip" {
			statuses[key] = event.Action
		}
	}

	err := scanner.Err()
	if err != nil {
		return nil, err
	}

	for _, r := range roots {
		status := statuses[r.Package+":"+r.Name]
		if status != "pass" && !(repository && status == "skip" && declaredSkip(testEvent{Package: r.Package, Test: r.Name})) {
			return nil, fmt.Errorf("missing successful root result: %s", r.key())
		}
	}

	return declared, nil
}

func checkFixture(path, binary string, guide bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	text := string(data)

	required := []string{"source: " + fixtureSource, "module: " + fixtureModule, "module-sum: " + fixtureSum, "go: go1.27.1", "binary-sha256: " + binary, "objects-after-cleanup: 0", "bucket: removed", "server: stopped", "temporary-root: removed"}
	if guide {
		required = append(required, "guide: signed POST and exact final readback passed", "guide: spool empty")
	} else {
		required = append(required, "scope-probe-bucket: removed", "lifecycle: local and remote snapshots, failed replacement, cancellation and absence passed")
	}

	for _, line := range required {
		if !slices.Contains(strings.Split(text, "\n"), line) {
			return fmt.Errorf("missing fixture identity/cleanup: %s", line)
		}
	}

	return nil
}

func verifyEvidence(directory string) error {
	directory, err := evidenceDirectory(directory)
	if err != nil {
		return err
	}

	var recorded manifest

	err = readJSON(filepath.Join(directory, "manifest.json"), &recorded)
	if err != nil {
		return err
	}

	current, err := identity(recorded.Mode)
	if err != nil || !reflect.DeepEqual(recorded, current) {
		return fmt.Errorf("candidate/source/root/tool identity drift: %v", err)
	}

	final := receipt{Status: "focused passed; full gate pending", Artifacts: make(map[string]string)}

	final.ManifestSHA256, err = hashFile(filepath.Join(directory, "manifest.json"))
	if err != nil {
		return err
	}

	final.Steps, final.DeclaredSkips, err = verifySteps(directory, recorded)
	if err != nil {
		return err
	}

	if recorded.Mode == "full" {
		final.Status = "accepted"
	}

	binary, err := hashFile(".tmp/object-storage/tools/minio")
	if err != nil {
		return err
	}

	for _, name := range []string{"remote-fixture.txt", "guide-fixture.txt"} {
		err := checkFixture(filepath.Join(directory, name), binary, name == "guide-fixture.txt")
		if err != nil {
			return err
		}
	}

	err = checkPostgres(directory)
	if err != nil {
		return err
	}

	for _, name := range []string{"receipt.txt", "remote-fixture.txt", "guide-fixture.txt", "guide-fixture.txt.results.jsonl", "postgres-fixture.txt", "tool-preparation.log"} {
		digest, err := hashFile(filepath.Join(directory, name))
		if err != nil {
			return err
		}

		final.Artifacts[name] = digest
	}

	for _, criterion := range criteria {
		outcome := "passed"
		if criterion.ID == "OS10" && recorded.Mode != "full" {
			outcome = "pending full integrated gate"
		}

		final.Results = append(final.Results, result{Criterion: criterion.ID, Outcome: outcome, Roots: criterion.Roots})
	}

	return writeJSON(filepath.Join(directory, "acceptance.json"), final)
}

func checkPostgres(directory string) error {
	data, err := os.ReadFile(filepath.Join(directory, "postgres-fixture.txt"))
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range []string{"database: owned native Unix socket; TCP disabled", "database: ready", "database: stopped", "temporary-root: removed"} {
		if !slices.Contains(lines, line) {
			return fmt.Errorf("missing PostgreSQL lifecycle evidence: %s", line)
		}
	}

	hashes := 0

	for _, line := range lines {
		expected, path, found := strings.Cut(line, "  ")
		if !found || len(expected) != 64 {
			continue
		}

		actual, err := hashFile(path)
		if err != nil || actual != expected {
			return fmt.Errorf("PostgreSQL tool/log digest changed: %s", path)
		}

		hashes++
	}

	if hashes < 9 {
		return fmt.Errorf("PostgreSQL tool/log identity evidence is incomplete")
	}

	fmt.Println("Owned PostgreSQL readiness and cleanup passed.")

	return nil
}

func verifySteps(directory string, recorded manifest) ([]step, []string, error) {
	var (
		steps         []step
		declaredSkips []string
	)

	files, err := filepath.Glob(filepath.Join(directory, "step-*.json"))

	labels := expectedLabels(recorded.Mode)
	if err != nil || len(files) != len(labels) {
		return nil, nil, fmt.Errorf("missing command evidence")
	}

	groups := make(map[string]bool)

	for index, path := range files {
		var s step

		err := readJSON(path, &s)
		if err != nil {
			return nil, nil, err
		}

		digest, err := hashFile(s.Log)
		if err != nil || s.Index != index+1 || s.Label != labels[index] || s.Exit != 0 || digest != s.SHA256 || len(s.Arguments) == 0 {
			return nil, nil, fmt.Errorf("failed/missing/changed command or log: %s", path)
		}

		var required []root

		group := s.Group
		if group == "race" {
			group = "ordinary"
		}

		for _, r := range recorded.Roots {
			if r.Group == group {
				required = append(required, r)
			}
		}

		if s.Group == "repository" {
			if len(s.Arguments) != 6 || s.Arguments[0] != "scripts/object-storage/postgres-fixture.sh" || s.Arguments[1] != directory || s.Arguments[2] != "make" || s.Arguments[3] != "check" || s.Arguments[4] != "TEST_FLAGS=-json -count=1 -p=1 -parallel=2 -timeout=360s" || s.Arguments[5] != "LINT_GOCACHE="+filepath.Join(directory, "lint-gocache") {
				return nil, nil, fmt.Errorf("complete repository gate command drift")
			}

			required = recorded.RepositoryRoots
		}

		if len(required) > 0 {
			log, err := os.Open(s.Log)
			if err != nil {
				return nil, nil, err
			}

			declared, resultErr := rootResults(log, required, s.Group == "repository")

			closeErr := log.Close()
			if resultErr != nil || closeErr != nil {
				return nil, nil, fmt.Errorf("result evidence %s: %v / %v", s.Label, resultErr, closeErr)
			}

			declaredSkips = append(declaredSkips, declared...)
		}

		groups[s.Group] = true
		steps = append(steps, s)
	}

	for _, required := range []string{"ordinary", "race", "remote", "guide", "guide-child", "docs", "postgres"} {
		if !groups[required] {
			return nil, nil, fmt.Errorf("required command group missing: %s", required)
		}
	}

	if recorded.Mode == "full" {
		if !groups["repository"] || len(recorded.RepositoryRoots) == 0 {
			return nil, nil, fmt.Errorf("complete existing repository gate is required")
		}
	}

	return steps, declaredSkips, nil
}
