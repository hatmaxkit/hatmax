package project

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func copyFixture(t *testing.T, name string) string {
	t.Helper()

	source := filepath.Join("testdata", name)
	target := filepath.Join(t.TempDir(), filepath.Base(name))

	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
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
		t.Fatalf("copy fixture %q: %v", name, err)
	}

	return target
}

func writeProjectFile(t *testing.T, root, path, content string) {
	t.Helper()

	absolute := filepath.Join(root, filepath.FromSlash(path))

	err := os.MkdirAll(filepath.Dir(absolute), 0o755)
	if err != nil {
		t.Fatalf("create parent for %q: %v", path, err)
	}

	err = os.WriteFile(absolute, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func appendProjectFile(t *testing.T, root, path, content string) {
	t.Helper()

	absolute := filepath.Join(root, filepath.FromSlash(path))

	file, err := os.OpenFile(absolute, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open %q: %v", path, err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		t.Fatalf("append %q: %v", path, err)
	}
}

func initializeGit(t *testing.T, root string) {
	t.Helper()

	runTestGit(t, root, "init", "--quiet")
	runTestGit(t, root, "add", ".")
	runTestGit(
		t,
		root,
		"-c", "user.name=Hatmax Test",
		"-c", "user.email=hatmax@example.invalid",
		"commit", "--quiet", "-m", "fixture",
	)
}

func runTestGit(t *testing.T, root string, arguments ...string) {
	t.Helper()

	commandArguments := append([]string{"-C", root}, arguments...)
	command := exec.Command("git", commandArguments...)

	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", arguments, err, output)
	}
}

func requireProjectError(t *testing.T, err error, code string) {
	t.Helper()

	if err == nil {
		t.Fatalf("error = nil, want project error %q", code)
	}

	var projectErr Error
	if !errors.As(err, &projectErr) {
		t.Fatalf("error = %v, want project Error", err)
	}

	if projectErr.Code != code {
		t.Fatalf("error code = %q, want %q", projectErr.Code, code)
	}
}

func hasString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}

	return false
}

func hasProtectedPath(values []ProtectedPath, expected string) bool {
	for _, value := range values {
		if value.Path == expected {
			return true
		}
	}

	return false
}

func hasCommand(commands []Command, name string) bool {
	for _, command := range commands {
		if command.Name == name {
			return true
		}
	}

	return false
}
