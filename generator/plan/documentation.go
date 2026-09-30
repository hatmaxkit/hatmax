// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package plan

import (
	"path"
	"sort"
	"strings"
	"unicode"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/project"
)

const (
	documentationIndexRoot     = "root"
	documentationIndexQuadrant = "quadrant"
)

type documentationQuadrantLayout struct {
	directory string
	title     string
	rank      int
}

var documentationQuadrants = map[intent.DocumentationQuadrant]documentationQuadrantLayout{
	intent.DocumentationTutorial:    {directory: "tutorials", title: "Tutorials", rank: 0},
	intent.DocumentationHowTo:       {directory: "how-to", title: "How-to Guides", rank: 1},
	intent.DocumentationReference:   {directory: "reference", title: "Reference", rank: 2},
	intent.DocumentationExplanation: {directory: "explanation", title: "Explanation", rank: 3},
}

func expandDocumentationPlan(value intent.Intent, inventory project.Inventory) (*DocumentationPlan, error) {
	if value.Documentation == intent.DocumentationNotRequested {
		return nil, nil
	}

	targets := make([]DocumentationTargetEffect, 0, len(value.DocumentationTargets))
	quadrantTargets := make(map[intent.DocumentationQuadrant][]DocumentationLinkEffect)
	seenPaths := make(map[string]struct{})

	for _, target := range value.DocumentationTargets {
		layout := documentationQuadrants[target.Quadrant]
		slug := strings.ReplaceAll(target.Subject, "_", "-")
		targetPath := path.Join("docs", layout.directory, slug, "README.md")

		if _, exists := seenPaths[targetPath]; exists {
			return nil, planError("plan_documentation_target_overlap", "documentation_plan.targets", "multiple documentation targets resolve to %q", targetPath)
		}

		snapshot, err := documentationSnapshot(inventory, targetPath)
		if err != nil {
			return nil, err
		}

		title := documentationTargetTitle(target.Quadrant, target.Subject)
		targets = append(targets, DocumentationTargetEffect{
			Quadrant:   target.Quadrant,
			Subject:    target.Subject,
			ReaderGoal: target.ReaderGoal,
			Slug:       slug,
			Title:      title,
			Path:       targetPath,
			Snapshot:   snapshot,
		})
		quadrantTargets[target.Quadrant] = append(quadrantTargets[target.Quadrant], DocumentationLinkEffect{
			Title:    title,
			Target:   targetPath,
			Relative: path.Join(slug, "README.md"),
		})
		seenPaths[targetPath] = struct{}{}
	}

	sort.Slice(targets, func(left, right int) bool {
		leftLayout := documentationQuadrants[targets[left].Quadrant]
		rightLayout := documentationQuadrants[targets[right].Quadrant]

		if leftLayout.rank != rightLayout.rank {
			return leftLayout.rank < rightLayout.rank
		}

		return targets[left].Subject < targets[right].Subject
	})

	indexes, err := documentationIndexes(inventory, quadrantTargets)
	if err != nil {
		return nil, err
	}

	commands := make([]DocumentationCommand, 0, len(inventory.Documentation.ValidationCommands))
	for _, command := range inventory.Documentation.ValidationCommands {
		commands = append(commands, DocumentationCommand{
			Name: command.Name, Args: append([]string{}, command.Args...), Source: command.Source, Kind: command.Kind,
		})
	}

	return &DocumentationPlan{Targets: targets, Indexes: indexes, ValidationCommands: commands}, nil
}

func documentationIndexes(
	inventory project.Inventory,
	quadrantTargets map[intent.DocumentationQuadrant][]DocumentationLinkEffect,
) ([]DocumentationIndexEffect, error) {
	quadrants := make([]intent.DocumentationQuadrant, 0, len(quadrantTargets))
	for quadrant := range quadrantTargets {
		quadrants = append(quadrants, quadrant)
	}

	sort.Slice(quadrants, func(left, right int) bool {
		return documentationQuadrants[quadrants[left]].rank < documentationQuadrants[quadrants[right]].rank
	})

	rootSnapshot, err := documentationSnapshot(inventory, "docs/README.md")
	if err != nil {
		return nil, err
	}

	rootLinks := make([]DocumentationLinkEffect, 0, len(quadrants))
	for _, quadrant := range quadrants {
		layout := documentationQuadrants[quadrant]
		rootLinks = append(rootLinks, DocumentationLinkEffect{
			Title: layout.title, Target: path.Join("docs", layout.directory, "README.md"), Relative: path.Join(layout.directory, "README.md"),
		})
	}

	result := []DocumentationIndexEffect{{
		Kind: documentationIndexRoot, Title: "Documentation", Path: "docs/README.md", RequiredLinks: rootLinks, Snapshot: rootSnapshot,
	}}

	for _, quadrant := range quadrants {
		layout := documentationQuadrants[quadrant]
		indexPath := path.Join("docs", layout.directory, "README.md")

		snapshot, snapshotErr := documentationSnapshot(inventory, indexPath)
		if snapshotErr != nil {
			return nil, snapshotErr
		}

		links := append([]DocumentationLinkEffect{}, quadrantTargets[quadrant]...)
		sort.Slice(links, func(left, right int) bool {
			return links[left].Target < links[right].Target
		})

		result = append(result, DocumentationIndexEffect{
			Kind:          documentationIndexQuadrant,
			Quadrant:      quadrant,
			Title:         layout.title,
			Path:          indexPath,
			RequiredLinks: links,
			Snapshot:      snapshot,
		})
	}

	return result, nil
}

func documentationSnapshot(inventory project.Inventory, target string) (DocumentationSnapshot, error) {
	if protected, reason := protectedDocumentationTarget(inventory.Documentation.ProtectedPaths, target); protected {
		return DocumentationSnapshot{}, planError("plan_documentation_target_protected", target, "%s", reason)
	}

	file, exists := inventory.DocumentationFile(target)
	if !exists {
		if _, fileExists := inventory.File(target); fileExists {
			return DocumentationSnapshot{}, planError("plan_documentation_conflict", target, "canonical documentation target cannot be inspected")
		}

		return DocumentationSnapshot{Path: target}, nil
	}

	if file.ManagedState != project.DocumentationManaged {
		return DocumentationSnapshot{}, planError("plan_documentation_conflict", target, "existing documentation target is %s", file.ManagedState)
	}

	return DocumentationSnapshot{
		Path: target, Exists: true, Digest: file.Digest, ManagedState: file.ManagedState,
	}, nil
}

func protectedDocumentationTarget(values []project.ProtectedPath, target string) (bool, string) {
	for _, value := range values {
		if value.Path == target || strings.HasPrefix(value.Path, target+"/") || strings.HasPrefix(target, value.Path+"/") {
			return true, value.Reason
		}
	}

	return false, ""
}

func documentationTargetTitle(quadrant intent.DocumentationQuadrant, subject string) string {
	name := documentationTitleWords(subject)

	switch quadrant {
	case intent.DocumentationTutorial:
		return "Build " + name
	case intent.DocumentationHowTo:
		return "How to Use " + name
	case intent.DocumentationReference:
		return name + " Reference"
	case intent.DocumentationExplanation:
		return "Understanding " + name
	default:
		return name
	}
}

func documentationTitleWords(value string) string {
	words := strings.Split(value, "_")
	for index, word := range words {
		runes := []rune(word)
		if len(runes) == 0 {
			continue
		}

		runes[0] = unicode.ToUpper(runes[0])
		words[index] = string(runes)
	}

	return strings.Join(words, " ")
}
