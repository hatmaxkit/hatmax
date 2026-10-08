// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"fmt"
	"os"
	"regexp"
)

var documentationLinkPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\]\(([^)]+)\)`),
	regexp.MustCompile(`<(?:img|source)\b[^>]*\bsrc="([^"]+)"`),
}

// Keep the existing docs-check link and image coverage while using its Go toolchain.
func listDocumentationLinks(files []string) error {
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		for _, pattern := range documentationLinkPatterns {
			for _, match := range pattern.FindAllStringSubmatch(string(data), -1) {
				fmt.Printf("%s\t%s\n", file, match[1])
			}
		}
	}

	return nil
}
