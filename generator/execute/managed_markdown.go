// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package execute

import "bytes"

var (
	managedMarkdownStart = []byte("<!-- hatmax:generated:start -->")
	managedMarkdownEnd   = []byte("<!-- hatmax:generated:end -->")
)

// renderManagedMarkdown creates or replaces one Hatmax-owned Markdown section
// while preserving every byte outside existing balanced markers.
func renderManagedMarkdown(existing, managed []byte) ([]byte, error) {
	if bytes.Contains(managed, managedMarkdownStart) || bytes.Contains(managed, managedMarkdownEnd) {
		return nil, executionError("execution_managed_section_invalid", "documentation", "rendered body cannot contain ownership markers")
	}

	body := bytes.TrimRight(managed, "\n")
	if len(body) == 0 {
		return nil, executionError("execution_managed_section_invalid", "documentation", "rendered body is empty")
	}

	section := make([]byte, 0, len(managedMarkdownStart)+len(body)+len(managedMarkdownEnd)+3)
	section = append(section, managedMarkdownStart...)
	section = append(section, '\n')
	section = append(section, body...)
	section = append(section, '\n')
	section = append(section, managedMarkdownEnd...)

	if len(existing) == 0 {
		return append(section, '\n'), nil
	}

	start, end, err := managedSectionBounds(existing)
	if err != nil {
		return nil, err
	}

	result := make([]byte, 0, start+len(section)+len(existing)-end)
	result = append(result, existing[:start]...)
	result = append(result, section...)
	result = append(result, existing[end:]...)

	return result, nil
}

func managedOutsideDigest(content []byte) (string, error) {
	start, end, err := managedSectionBounds(content)
	if err != nil {
		return "", err
	}

	outside := make([]byte, 0, start+len(content)-end)
	outside = append(outside, content[:start]...)
	outside = append(outside, content[end:]...)

	return contentDigest(outside), nil
}

func managedSectionContent(content []byte) ([]byte, error) {
	start, end, err := managedSectionBounds(content)
	if err != nil {
		return nil, err
	}

	bodyStart := start + len(managedMarkdownStart)
	bodyEnd := end - len(managedMarkdownEnd)

	return content[bodyStart:bodyEnd], nil
}

func managedSectionBounds(content []byte) (int, int, error) {
	if bytes.Count(content, managedMarkdownStart) != 1 || bytes.Count(content, managedMarkdownEnd) != 1 {
		return 0, 0, executionError("execution_managed_section_invalid", "documentation", "managed Markdown requires one balanced marker pair")
	}

	start := bytes.Index(content, managedMarkdownStart)
	endStart := bytes.Index(content, managedMarkdownEnd)

	if start >= endStart {
		return 0, 0, executionError("execution_managed_section_invalid", "documentation", "managed Markdown markers are reversed")
	}

	return start, endStart + len(managedMarkdownEnd), nil
}
