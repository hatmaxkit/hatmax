//go:build acceptance

package hatmaxcli

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"hatmax.adrianpk.com/generator/interaction"
)

func TestTerminalSurfaceValidatesGeneratedProjectWithRealCommands(t *testing.T) {
	requireProjectAcceptanceCommand(t, "sqlc")
	requireProjectAcceptanceCommand(t, "golangci-lint")
	configureLocalProjectTests(t)

	root := copyCLIProject(t)
	writeCLICompositionRoot(t, root)
	useCurrentHatmaxModule(t, root)

	exitCode, output, errorOutput, result := runCLIRequest(
		t,
		root,
		"yes\n",
		"Create an invoice feature.",
		&cliInterpreter{interpret: interpretCLIOperation},
		nil,
	)
	if exitCode != ExitSuccess || result == nil || result.Outcome != interaction.OutcomeCompleted {
		var outcome interaction.Outcome
		var commands any
		if result != nil {
			outcome = result.Outcome
			commands = result.Report.Commands
		}

		t.Fatalf("real project generation = exit %d, outcome %q, commands %#v, stderr %q:\n%s", exitCode, outcome, commands, errorOutput, tailOutput(output))
	}

	assertRealProjectCommands(t, result)
	assertSingleMainFunction(t, root)
	assertUncompilableProjectFailsGate(t, root)
}

func configureLocalProjectTests(t *testing.T) {
	t.Helper()
	t.Setenv("GOCACHE", filepath.Join(t.TempDir(), "go-build"))
	t.Setenv("GOLANGCI_LINT_CACHE", filepath.Join(t.TempDir(), "golangci-lint"))

	if os.Getenv("DB_HOST") == "" {
		t.Setenv("GO_TEST_COMMAND", "go test -run='Test(New|Service|Handler|PostgresStoreRequires)' ./...")
	}
}

func tailOutput(output string) string {
	const limit = 4000
	if len(output) <= limit {
		return output
	}

	return output[len(output)-limit:]
}

func requireProjectAcceptanceCommand(t *testing.T, name string) {
	t.Helper()

	_, err := exec.LookPath(name)
	if err != nil {
		t.Fatalf("%s is required for real project acceptance: %v", name, err)
	}
}

func useCurrentHatmaxModule(t *testing.T, root string) {
	t.Helper()

	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve Hatmax repository root: %v", err)
	}

	commands := [][]string{
		{"mod", "edit", "-replace=hatmax.adrianpk.com=" + repositoryRoot},
		{"mod", "tidy"},
	}

	for _, arguments := range commands {
		command := exec.Command("go", arguments...)
		command.Dir = root

		output, commandErr := command.CombinedOutput()
		if commandErr != nil {
			t.Fatalf("prepare representative project with go %s: %v\n%s", strings.Join(arguments, " "), commandErr, output)
		}
	}
}

func assertRealProjectCommands(t *testing.T, result *interaction.Result) {
	t.Helper()

	want := map[string]bool{
		"generation.sqlc":   false,
		"formatting.format": false,
		"validation.check":  false,
	}

	for _, command := range result.Report.Commands {
		if _, exists := want[command.Name]; exists && command.ExitCode == 0 {
			want[command.Name] = true
		}
	}

	for name, passed := range want {
		if !passed {
			t.Errorf("real command %q did not pass: %#v", name, result.Report.Commands)
		}
	}
}

func assertSingleMainFunction(t *testing.T, root string) {
	t.Helper()

	path := filepath.Join(root, "main.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated composition root: %v", err)
	}

	parsed, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
	if err != nil {
		t.Fatalf("parse generated composition root: %v", err)
	}

	if len(parsed.Decls) == 0 {
		t.Fatal("generated composition root has no declarations")
	}

	functions := 0
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}

		functions++
		if function.Name.Name != "main" {
			t.Errorf("composition root contains helper function %q", function.Name.Name)
		}
	}

	if functions != 1 {
		t.Errorf("composition root functions = %d, want only main", functions)
	}
}

func assertUncompilableProjectFailsGate(t *testing.T, root string) {
	t.Helper()

	path := filepath.Join(root, "internal", "feat", "invoice", "service.go")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open generated service for corruption check: %v", err)
	}

	_, writeErr := file.WriteString("\nfunc uncompilable(\n")
	closeErr := file.Close()
	if writeErr != nil {
		t.Fatalf("corrupt generated service: %v", writeErr)
	}

	if closeErr != nil {
		t.Fatalf("close corrupted generated service: %v", closeErr)
	}

	command := exec.Command("make", "check")
	command.Dir = root

	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("real project gate accepted uncompilable output:\n%s", output)
	}

	if !strings.Contains(string(output), "service.go") {
		t.Errorf("failed real project gate omitted source evidence:\n%s", output)
	}
}
