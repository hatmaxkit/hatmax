package project

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

func inspectCommands(root string, files []fileRecord) ([]Command, error) {
	result := make([]Command, 0)

	for _, file := range files {
		switch filepath.Base(file.path) {
		case "Makefile":
			commands, err := inspectMakefile(root, file.path)
			if err != nil {
				return nil, err
			}

			result = append(result, commands...)
		case "sqlc.yaml", "sqlc.yml":
			result = append(result, Command{
				Kind:   CommandGeneration,
				Name:   "sqlc",
				Args:   []string{"sqlc", "generate"},
				Source: file.path,
			})
		}
	}

	return deduplicateCommands(result), nil
}

func inspectMakefile(root, path string) ([]Command, error) {
	absolute, err := projectPath(root, path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(absolute)
	if err != nil {
		return nil, unexpectedReadError(path, err)
	}
	defer file.Close()

	result := make([]Command, 0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		targets, found := makeTargets(scanner.Text())
		if !found {
			continue
		}

		for _, target := range targets {
			kind, admitted := classifyMakeTarget(target)
			if !admitted {
				continue
			}

			result = append(result, Command{
				Kind:   kind,
				Name:   target,
				Args:   []string{"make", target},
				Source: path,
			})
		}
	}

	err = scanner.Err()
	if err != nil {
		return nil, unexpectedReadError(path, err)
	}

	return result, nil
}

func makeTargets(line string) ([]string, bool) {
	if line == "" || line[0] == '\t' || strings.HasPrefix(strings.TrimSpace(line), "#") {
		return nil, false
	}

	colon := strings.IndexByte(line, ':')
	if colon <= 0 || strings.Contains(line[:colon], "=") {
		return nil, false
	}

	targets := strings.Fields(line[:colon])
	for _, target := range targets {
		if strings.ContainsAny(target, "%$/") {
			return nil, false
		}
	}

	return targets, len(targets) > 0
}

func classifyMakeTarget(target string) (CommandKind, bool) {
	lower := strings.ToLower(target)
	switch {
	case lower == "fmt" || strings.Contains(lower, "format"):
		return CommandFormatting, true
	case strings.Contains(lower, "generate") || strings.Contains(lower, "sqlc") || strings.Contains(lower, "templ"):
		return CommandGeneration, true
	case strings.Contains(lower, "test") || strings.Contains(lower, "check") || strings.Contains(lower, "lint") || lower == "vet" || lower == "ci":
		return CommandValidation, true
	default:
		return "", false
	}
}

func deduplicateCommands(commands []Command) []Command {
	seen := make(map[string]struct{}, len(commands))

	result := make([]Command, 0, len(commands))
	for _, command := range commands {
		key := string(command.Kind) + "\x00" + formatCommand(command)
		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}

		result = append(result, command)
	}

	return result
}
