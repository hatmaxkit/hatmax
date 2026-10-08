// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const generatorGuide = "docs/tutorials/user-guide/assisted-generation.md"

func (v *verification) generatorWorkbench(discovered inventory) error {
	directory := filepath.Join(v.fixture, "generator-workbench")

	err := v.module(directory)
	if err != nil {
		return err
	}

	module := filepath.Join(directory, "go.mod")

	contents, err := os.ReadFile(module)
	if err != nil {
		return err
	}

	// The isolated module remains within Go's internal-package import boundary.
	err = writeFile(module, strings.Replace(string(contents), "example.com/docfixture", "hatmax.adrianpk.com/documentationfixture", 1))
	if err != nil {
		return err
	}

	replacements := make([]string, 0)

	for marker, input := range map[string]struct {
		page   string
		number int
	}{
		"__INSTALL__": {generatorGuide, 1}, "__OPEN__": {generatorGuide, 2},
		"__REOPEN__": {generatorGuide, 5}, "__SELECT__": {generatorGuide, 7},
		"__RESET__": {generatorGuide, 8}, "__PLAYGROUND__": {"docs/reference/generator/README.md", 3},
	} {
		body, err := sourceBlock(discovered.contents, input.page, input.number)
		if err != nil {
			return err
		}

		replacements = append(replacements, marker, strconv.Quote(body))
	}

	for marker, number := range map[string]int{"__DETAILED__": 3, "__TERSE__": 4, "__ADD_FIELD__": 6} {
		body, err := sourceBlock(discovered.contents, generatorGuide, number)
		if err != nil {
			return err
		}

		replacements = append(replacements, marker, strconv.Quote(strings.TrimSpace(body)))
	}

	body, err := sourceBlock(discovered.contents, "docs/reference/generator/README.md", 2)
	if err != nil {
		return err
	}

	prompt, err := strconv.Unquote(strings.TrimPrefix(strings.TrimSpace(body), "hm generate "))
	if err != nil {
		return fmt.Errorf("published headless request: %w", err)
	}

	replacements = append(replacements, "__CREATE_FEATURE__", strconv.Quote(prompt), "__SOURCE_ROOT__", strconv.Quote(v.root))

	install, err := sourceBlock(discovered.contents, "docs/reference/generator/README.md", 1)
	if err != nil {
		return err
	}

	guideInstall, err := sourceBlock(discovered.contents, generatorGuide, 1)
	if err != nil || install != guideInstall {
		return fmt.Errorf("source-install procedures disagree")
	}

	command, err := sourceBlock(discovered.contents, generatorGuide, 9)
	if err != nil {
		return err
	}

	field, err := strconv.Unquote(strings.TrimPrefix(strings.TrimSpace(command), "hm generate "))
	if err != nil {
		return fmt.Errorf("published follow-up request: %w", err)
	}

	guideField, err := sourceBlock(discovered.contents, generatorGuide, 6)
	if err != nil || field != strings.TrimSpace(guideField) {
		return fmt.Errorf("published timestamp requests disagree")
	}

	_, composition, found := strings.Cut(discovered.contents["internal/hatmaxcli/acceptance_test.go"], "func writeCLICompositionRoot")
	if !found {
		return fmt.Errorf("canonical existing-project fixture missing")
	}

	_, composition, found = strings.Cut(composition, "const source = `")
	if !found {
		return fmt.Errorf("canonical existing-project source missing")
	}

	composition, _, found = strings.Cut(composition, "`\n")
	if !found {
		return fmt.Errorf("canonical existing-project source boundary missing")
	}

	replacements = append(replacements, "__COMPOSITION__", strconv.Quote(composition))

	text := strings.NewReplacer(replacements...).Replace(generatorWorkflowTest + generatorCommandTest + generatorExistingTest)
	if regexp.MustCompile(`__[A-Z_]+__`).MatchString(text) {
		return fmt.Errorf("unbound published generator request")
	}

	err = writeFile(filepath.Join(directory, "workflow_test.go"), text)
	if err != nil {
		return err
	}

	for _, name := range []string{"hm", "hatmax"} {
		err = v.exec(v.root, "go", "build", "-o", filepath.Join(directory, name), "./cmd/"+name)
		if err != nil {
			return err
		}
	}

	err = v.exec(directory, "go", "mod", "tidy")
	if err != nil {
		return err
	}

	err = v.execBound(directory, 10*time.Minute, "", "go", "test", "-json", "-count=1", "-timeout=9m", "./...")
	if err != nil {
		return err
	}

	return nil
}
