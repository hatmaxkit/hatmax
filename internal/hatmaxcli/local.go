package hatmaxcli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"hatmax.adrianpk.com/generator/backend/codex"
	"hatmax.adrianpk.com/generator/interaction"
)

// NewCodexCoordinator assembles the production coordinator around the local
// resident Codex App Server adapter.
func NewCodexCoordinator(
	root string,
	approver interaction.Approver,
	clarifier interaction.Clarifier,
) (Runner, error) {
	identity, err := opaqueProjectIdentity(root)
	if err != nil {
		return nil, err
	}

	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user cache directory: %w", err)
	}

	interpreter, err := codex.NewLocalInterpreter(codex.InterpreterConfig{
		ContextRoot:     filepath.Join(cacheRoot, "hatmax", "codex"),
		ProjectIdentity: identity,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize Codex interpreter: %w", err)
	}

	coordinator, err := interaction.New(interaction.Config{
		Interpreter: interpreter,
		Approver:    approver,
		Clarifier:   clarifier,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize interaction coordinator: %w", err)
	}

	return coordinator, nil
}

func opaqueProjectIdentity(root string) (string, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve project root: %w", err)
	}

	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve canonical project root: %w", err)
	}

	digest := sha256.Sum256([]byte(filepath.Clean(canonical)))

	return "sha256:" + hex.EncodeToString(digest[:]), nil
}
