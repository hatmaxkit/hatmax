package conversation

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"runtime"
	"strings"
)

// ResolveScope derives an opaque stable key from one canonical absolute path.
// The path is not retained in the returned value.
func ResolveScope(kind ScopeKind, path string) (Scope, error) {
	if kind != ScopeProject && kind != ScopePreProject {
		return Scope{}, modelError("conversation_scope_invalid", "scope.kind", "unknown scope kind %q", kind)
	}

	if strings.TrimSpace(path) == "" {
		return Scope{}, modelError("conversation_scope_invalid", "scope.path", "scope path is required")
	}

	absolute, err := filepath.Abs(path)
	if err != nil {
		return Scope{}, modelError("conversation_scope_invalid", "scope.path", "resolve absolute scope path: %v", err)
	}

	canonical := filepath.Clean(absolute)
	if runtime.GOOS == "windows" {
		canonical = strings.ToLower(canonical)
	}

	digest := sha256.New()
	digest.Write([]byte(kind))
	digest.Write([]byte{0})
	digest.Write([]byte(canonical))

	return Scope{Key: hex.EncodeToString(digest.Sum(nil)), Kind: kind}, nil
}
