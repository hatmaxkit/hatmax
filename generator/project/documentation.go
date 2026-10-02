// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package project

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	documentationRoot         = "docs"
	managedSectionStartMarker = "<!-- hatmax:generated:start -->"
	managedSectionEndMarker   = "<!-- hatmax:generated:end -->"
)

var markdownLinkPattern = regexp.MustCompile(`\[[^]]*\]\(([^)]+)\)`)

// DocumentationFile returns detached canonical documentation metadata.
func (i Inventory) DocumentationFile(target string) (DocumentationFile, bool) {
	normalized := filepath.ToSlash(filepath.Clean(filepath.FromSlash(target)))
	for _, file := range i.Documentation.Files {
		if file.Path != normalized {
			continue
		}

		result := file
		result.LocalLinks = append([]DocumentationLink{}, file.LocalLinks...)

		return result, true
	}

	return DocumentationFile{}, false
}

func inspectDocumentation(inventory *Inventory) error {
	result := DocumentationInventory{
		Root:               documentationRoot,
		Files:              []DocumentationFile{},
		OtherPaths:         []string{},
		ProtectedPaths:     []ProtectedPath{},
		ValidationCommands: []Command{},
	}

	for _, file := range inventory.files {
		if !strings.HasPrefix(file.path, documentationRoot+"/") || strings.ToLower(filepath.Ext(file.path)) != ".md" {
			continue
		}

		quadrant, subject, index, canonical := classifyDocumentationPath(file.path)
		if !canonical {
			result.OtherPaths = append(result.OtherPaths, file.path)

			continue
		}

		documentationFile, err := inspectDocumentationFile(*inventory, file, quadrant, subject, index)
		if err != nil {
			return err
		}

		result.Files = append(result.Files, documentationFile)
	}

	for _, protected := range inventory.Rules.ProtectedPaths {
		if protected.Path == documentationRoot || strings.HasPrefix(protected.Path, documentationRoot+"/") {
			result.ProtectedPaths = append(result.ProtectedPaths, protected)
		}
	}

	for _, command := range inventory.Commands {
		if isDocumentationValidationCommand(command) {
			result.ValidationCommands = append(result.ValidationCommands, cloneCommand(command))
		}
	}

	inventory.Documentation = result

	return nil
}

func inspectDocumentationFile(
	inventory Inventory,
	file fileRecord,
	quadrant string,
	subject string,
	index bool,
) (DocumentationFile, error) {
	result := DocumentationFile{
		Path:     file.path,
		Quadrant: quadrant,
		Subject:  subject,
		Index:    index,
		Size:     file.size,
	}

	if file.size > inventory.maximumFileSize {
		result.ManagedState = DocumentationUnreadable

		return result, nil
	}

	absolute, err := projectPath(inventory.Root, file.path)
	if err != nil {
		return DocumentationFile{}, err
	}

	content, err := os.ReadFile(absolute)
	if err != nil {
		return DocumentationFile{}, unexpectedReadError(file.path, err)
	}

	result.Digest, err = inventory.digestProjectFile(file.path)
	if err != nil {
		return DocumentationFile{}, err
	}

	result.ManagedState = managedSectionState(content)
	result.LocalLinks = localDocumentationLinks(inventory, file.path, content)

	return result, nil
}

func classifyDocumentationPath(filePath string) (string, string, bool, bool) {
	segments := strings.Split(filePath, "/")
	if len(segments) == 2 && segments[0] == documentationRoot && segments[1] == "README.md" {
		return "", "", true, true
	}

	if len(segments) != 3 && len(segments) != 4 {
		return "", "", false, false
	}

	quadrant, admitted := documentationQuadrant(segments[1])
	if !admitted || segments[len(segments)-1] != "README.md" {
		return "", "", false, false
	}

	if len(segments) == 3 {
		return quadrant, "", true, true
	}

	if segments[2] == "" || segments[2] == "." || segments[2] == ".." {
		return "", "", false, false
	}

	return quadrant, segments[2], false, true
}

func documentationQuadrant(directory string) (string, bool) {
	switch directory {
	case "tutorials":
		return "tutorial", true
	case "how-to":
		return "how_to", true
	case "reference":
		return "reference", true
	case "explanation":
		return "explanation", true
	default:
		return "", false
	}
}

func managedSectionState(content []byte) DocumentationManagedState {
	start := []byte(managedSectionStartMarker)
	end := []byte(managedSectionEndMarker)
	startCount := bytes.Count(content, start)
	endCount := bytes.Count(content, end)

	if startCount == 0 && endCount == 0 {
		return DocumentationUnmanaged
	}

	if startCount == 1 && endCount == 1 && bytes.Index(content, start) < bytes.Index(content, end) {
		return DocumentationManaged
	}

	return DocumentationMarkersInvalid
}

func localDocumentationLinks(inventory Inventory, source string, content []byte) []DocumentationLink {
	matches := markdownLinkPattern.FindAllSubmatch(content, -1)
	result := make([]DocumentationLink, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))

	for _, match := range matches {
		target, local := resolveDocumentationLink(source, strings.TrimSpace(string(match[1])))
		if !local {
			continue
		}

		if _, exists := seen[target]; exists {
			continue
		}

		_, exists := inventory.File(target)
		result = append(result, DocumentationLink{Target: target, Exists: exists})
		seen[target] = struct{}{}
	}

	return result
}

func resolveDocumentationLink(source, raw string) (string, bool) {
	if raw == "" || strings.HasPrefix(raw, "#") || strings.Contains(raw, "://") || strings.HasPrefix(raw, "mailto:") {
		return "", false
	}

	target := strings.Fields(raw)[0]

	target = strings.Trim(target, "<>")
	if fragment := strings.IndexAny(target, "?#"); fragment >= 0 {
		target = target[:fragment]
	}

	if target == "" {
		return "", false
	}

	resolved := path.Clean(path.Join(path.Dir(source), target))
	if strings.HasSuffix(target, "/") || path.Ext(resolved) == "" {
		resolved = path.Join(resolved, "README.md")
	}

	if resolved != documentationRoot+"/README.md" && !strings.HasPrefix(resolved, documentationRoot+"/") {
		return "", false
	}

	return resolved, true
}

func isDocumentationValidationCommand(command Command) bool {
	if command.Kind != CommandValidation {
		return false
	}

	name := strings.ToLower(command.Name)

	return strings.Contains(name, "doc") || strings.Contains(name, "markdown") || strings.Contains(name, "link") || strings.Contains(name, "readme")
}

func cloneCommand(value Command) Command {
	value.Args = append([]string{}, value.Args...)

	return value
}
