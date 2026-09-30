// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package execute

import (
	"bytes"
	"testing"
)

func TestRenderManagedMarkdownCreatesCanonicalSection(t *testing.T) {
	result, err := renderManagedMarkdown(nil, []byte("# Property Reference\n\nObserved contracts.\n"))
	if err != nil {
		t.Fatalf("renderManagedMarkdown() error = %v", err)
	}

	want := "<!-- hatmax:generated:start -->\n# Property Reference\n\nObserved contracts.\n<!-- hatmax:generated:end -->\n"
	if string(result) != want {
		t.Errorf("renderManagedMarkdown() = %q, want %q", result, want)
	}
}

func TestRenderManagedMarkdownPreservesUserContent(t *testing.T) {
	prefix := []byte("# Project handbook\n\nUser introduction.\n\n")
	suffix := []byte("\n\nUser appendix.\n")

	existing := append(append(append(append([]byte{}, prefix...), managedMarkdownStart...), []byte("\nOld generated content.\n")...), managedMarkdownEnd...)
	existing = append(existing, suffix...)

	result, err := renderManagedMarkdown(existing, []byte("# Property Reference\n\nNew generated content."))
	if err != nil {
		t.Fatalf("renderManagedMarkdown() error = %v", err)
	}

	if !bytes.HasPrefix(result, prefix) || !bytes.HasSuffix(result, suffix) {
		t.Errorf("renderManagedMarkdown() changed user-owned bytes: %q", result)
	}

	second, err := renderManagedMarkdown(result, []byte("# Property Reference\n\nNew generated content."))
	if err != nil {
		t.Fatalf("second renderManagedMarkdown() error = %v", err)
	}

	if !bytes.Equal(second, result) {
		t.Error("managed Markdown replacement is not idempotent")
	}
}

func TestRenderManagedMarkdownRejectsInvalidMarkers(t *testing.T) {
	_, err := renderManagedMarkdown([]byte("<!-- hatmax:generated:start -->\nmissing end\n"), []byte("generated"))
	requireExecutionCode(t, err, "execution_managed_section_invalid")
}
