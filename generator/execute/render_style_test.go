// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package execute

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

var inlineErrorAssignment = regexp.MustCompile(`\bif\s+[^\n{]*(?:err\s*:=|err\s*=\s+)[^\n{]*\{`)

func assertDeterministicMutations(t *testing.T, first, second []Mutation) {
	t.Helper()

	if !reflect.DeepEqual(first, second) {
		t.Error("repeated rendering produced different mutations")
	}
}

func assertCanonicalRenderedGo(t *testing.T, manifest Manifest, mutations []Mutation) {
	t.Helper()

	for _, mutation := range mutations {
		target := manifestEditTarget(manifest, mutation.EditID)
		if !strings.HasSuffix(target, ".go") {
			continue
		}

		source := string(mutation.Content)
		assertNoCompressedGoLines(t, mutation.EditID, source)

		for description, fragment := range map[string]string{
			"adjacent control blocks":      "}\n\tif ",
			"return attached to a block":   "}\n\treturn ",
			"constant route concatenation": `+"/{id}"`,
		} {
			if strings.Contains(source, fragment) {
				t.Errorf("mutation %q contains %s", mutation.EditID, description)
			}
		}

		if inlineErrorAssignment.MatchString(source) {
			t.Errorf("mutation %q handles an error in an inline assignment", mutation.EditID)
		}
	}
}

func assertNoCompressedGoLines(t *testing.T, editID, source string) {
	t.Helper()

	for number, line := range strings.Split(source, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "struct {") && strings.HasSuffix(trimmed, "}") {
			t.Errorf("mutation %q line %d contains a compressed struct", editID, number+1)
		}

		if strings.HasPrefix(trimmed, "func ") && strings.Contains(trimmed, "{") && !strings.HasSuffix(trimmed, "{") {
			t.Errorf("mutation %q line %d contains a compressed function", editID, number+1)
		}

		if strings.HasPrefix(trimmed, "if ") && strings.Contains(trimmed, "{") && !strings.HasSuffix(trimmed, "{") {
			t.Errorf("mutation %q line %d contains a compressed conditional", editID, number+1)
		}
	}
}
