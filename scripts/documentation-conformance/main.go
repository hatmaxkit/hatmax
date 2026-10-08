// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

// Command documentation-conformance inventories source and checks coverage.
package main

import (
	"crypto/sha256"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

const coveragePath = "ops/default/report/documentation-conformance-coverage.md"

type row struct {
	id, source, revision, digest, owner, page, need, method, receipt, status string
	slice                                                                    int
}

type inventory struct {
	contents map[string]string
	rows     []row
}

var references = map[string]string{
	"app": "application-lifecycle", "auth": "authentication", "config": "configuration",
	"crypto": "crypto", "db": "database", "fake": "fake", "format": "format",
	"htmx": "htmx", "i18n": "i18n", "image": "image", "log": "logging",
	"mailer": "mailer", "middleware": "middleware", "modal": "modal", "model": "model",
	"pagination": "pagination", "pubsub": "pubsub", "render": "rendering",
	"scheduler": "scheduler", "seed": "seed", "settings": "configuration", "slug": "slug",
	"telemetry": "telemetry", "testhelper": "testhelper", "ui": "ui", "validation": "validation",
	"web": "http", "generator": "generator", "cmd": "generator", "internal": "generator",
}

func main() {
	err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 2 && args[0] == "generator" {
		return runGenerator(args[1])
	}

	if len(args) == 2 && args[0] == "generator-evidence" {
		return checkGeneratorEvidence(args[1])
	}

	if len(args) == 2 && args[0] == "generator-workbench" {
		v, discovered, err := newGroupVerification(args[1], 6)
		if err != nil {
			return err
		}

		return v.generatorWorkbench(discovered)
	}

	if len(args) == 2 && args[0] == "infrastructure" {
		return runInfrastructure(args[1])
	}

	if len(args) == 2 && args[0] == "infrastructure-evidence" {
		return checkInfrastructureEvidence(args[1])
	}

	if len(args) == 2 && args[0] == "infrastructure-contexts" {
		v, discovered, err := newGroupVerification(args[1], 5)
		if err != nil {
			return err
		}

		err = compileInfrastructureContexts(v, discovered)
		if err == nil {
			fmt.Printf("Infrastructure contexts passed: %d exact Go fragments; workflow receipts remain pending.\n", len(v.receipt.Proofs))
		}

		return err
	}

	if len(args) == 2 && args[0] == "identity" {
		return runIdentity(args[1])
	}

	if len(args) == 2 && args[0] == "identity-evidence" {
		return checkIdentityEvidence(args[1])
	}

	if len(args) == 2 && args[0] == "identity-ticked" {
		v, discovered, err := newGroupVerification(args[1], 4)
		if err != nil {
			return err
		}

		return v.identityTicked(discovered)
	}

	if len(args) == 2 && args[0] == "identity-contexts" {
		v, discovered, err := newGroupVerification(args[1], 4)
		if err != nil {
			return err
		}

		err = compileIdentityContexts(v, discovered)
		if err == nil {
			fmt.Printf("Identity/crypto contexts passed: %d exact Go fragments; complete Slice 4 workflow receipts remain pending.\n", len(v.receipt.Proofs))
		}

		return err
	}

	if len(args) == 2 && args[0] == "data" {
		return runData(args[1])
	}

	if len(args) == 2 && args[0] == "data-evidence" {
		return checkDataEvidence(args[1])
	}

	if len(args) == 2 && args[0] == "data-workbench" {
		return runDataWorkbench(args[1])
	}

	if len(args) == 2 && args[0] == "data-contexts" {
		return runDataContexts(args[1])
	}

	if len(args) == 2 && args[0] == "runtime" {
		return runRuntime(args[1])
	}

	if len(args) == 2 && args[0] == "evidence" {
		return checkEvidence(args[1])
	}

	if len(args) != 1 || (args[0] != "inventory" && args[0] != "check") {
		return fmt.Errorf("usage: documentation-conformance inventory|check|runtime FIXTURE|evidence RECEIPT|data-contexts FIXTURE|data-workbench FIXTURE|data FIXTURE|data-evidence RECEIPT|identity-contexts FIXTURE|identity-ticked FIXTURE|identity FIXTURE|identity-evidence RECEIPT|infrastructure-contexts FIXTURE|infrastructure FIXTURE|infrastructure-evidence RECEIPT|generator-workbench FIXTURE|generator FIXTURE|generator-evidence RECEIPT")
	}

	discovered, err := discover()
	if err != nil {
		return err
	}

	if args[0] == "inventory" {
		revision, err := command("git", "rev-parse", "HEAD")
		if err != nil {
			return err
		}

		fmt.Print(render(discovered.rows, strings.TrimSpace(revision)))

		return nil
	}

	text, err := os.ReadFile(coveragePath)
	if err != nil {
		return err
	}

	recorded, err := parseRows(string(text))
	if err != nil {
		return err
	}

	revisions := make(map[string]bool)
	for _, r := range recorded {
		if revisions[r.revision] {
			continue
		}

		_, err := command("git", "cat-file", "-e", r.revision+"^{commit}")
		if err != nil {
			return fmt.Errorf("unknown inspected revision: %s: %w", r.id, err)
		}

		revisions[r.revision] = true
	}

	err = reconcile(discovered.rows, recorded)
	if err != nil {
		return err
	}

	err = checkNavigation(discovered.contents)
	if err != nil {
		return err
	}

	fmt.Printf("Coverage and navigation passed: %d source/page/example identities.\n", len(recorded))

	return nil
}

func command(name string, args ...string) (string, error) {
	output, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s failed: %w: %s", name, err, output)
	}

	return string(output), nil
}

func excluded(file string) bool {
	parts := strings.Split(file, "/")

	return slices.Contains(parts, "testdata") || slices.Contains(parts, "fixtures") ||
		strings.HasPrefix(file, "third_party/") || strings.HasPrefix(file, "scripts/")
}

func productPage(file string) bool {
	return strings.HasSuffix(strings.ToLower(file), ".md") && !excluded(file) &&
		!strings.HasPrefix(file, "ops/") && !strings.HasPrefix(file, ".") && file != "AGENTS.md"
}

func discover() (inventory, error) {
	// Include added index entries and non-ignored additions so new pages cannot hide.
	listing, err := command("git", "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err != nil {
		return inventory{}, err
	}

	files := strings.Split(strings.TrimSuffix(listing, "\x00"), "\x00")
	slices.Sort(files)
	files = slices.Compact(files)
	result := inventory{contents: make(map[string]string)}
	packages := make(map[string][]string)
	examples := make(map[string][]string)

	for _, file := range files {
		if strings.HasPrefix(file, "examples/") {
			parts := strings.Split(file, "/")
			root := strings.Join(parts[:2], "/")
			examples[root] = append(examples[root], file)
		}

		if excluded(file) {
			if strings.HasPrefix(file, "examples/") {
				content, err := os.ReadFile(file)
				if err != nil {
					return inventory{}, err
				}

				result.contents[file] = string(content)
			}

			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			return inventory{}, err
		}

		result.contents[file] = string(content)
		if strings.HasSuffix(file, ".go") {
			packages[path.Dir(file)] = append(packages[path.Dir(file)], file)
		}

		if productPage(file) {
			result.add("page:"+file, file, []string{file}, pageOwner(file), file, readerNeed(file), "source comparison and navigation")

			for index, block := range blocks(string(content)) {
				source := fmt.Sprintf("%s#block-%d", file, index+1)
				method := blockMethod(block)
				result.add("example:"+source, source, nil, pageOwner(file), file, "Validate "+method+" in the page's stated composition", method)

				result.rows[len(result.rows)-1].digest = digest(block)
				if file == "examples/guide/README.md" && strings.Contains(block, "GUIDE_DATABASE_ENABLED=true") {
					result.rows[len(result.rows)-1].owner = "db"
					result.rows[len(result.rows)-1].slice = 3
				}

				if file == "examples/guide/README.md" && strings.Contains(block, "/greeting") {
					result.rows[len(result.rows)-1].owner = "settings"
					result.rows[len(result.rows)-1].slice = 3
				}

				if strings.HasPrefix(block, "```sql") && pageOwner(file) == "app" {
					result.rows[len(result.rows)-1].owner = "db"
					result.rows[len(result.rows)-1].slice = 3
				}
			}

			if strings.HasPrefix(file, "docs/how-to/") || (strings.HasPrefix(file, "docs/tutorials/user-guide/") && !strings.HasSuffix(file, "/README.md")) {
				result.add("walkthrough:"+file, file, []string{file}, pageOwner(file), file, "Execute the complete published reader procedure", "real local walkthrough with owned fixtures")
			}
		}

		if strings.HasPrefix(file, "generator/book/") && !strings.HasSuffix(file, ".go") {
			result.add("book:"+file, file, []string{file}, "generator", referencePage("generator"), "Check selected Book content and capability boundaries", "Book selection, schema and generated conformance")
		} else if strings.HasSuffix(file, ".yaml") && !strings.HasPrefix(file, ".") {
			result.add("configuration:"+file, file, []string{file}, "config", referencePage("config"), "Check effective example configuration and tool settings", "real validator in the example composition")
		}
	}

	for directory, sources := range packages {
		owner := strings.Split(directory, "/")[0]
		kind := "package:"
		page := referencePage(owner)

		if strings.HasPrefix(directory, "examples/") {
			kind = "example-package:"
			owner = exampleOwner(directory)
			parts := strings.Split(directory, "/")
			page = strings.Join(parts[:2], "/") + "/README.md"
		} else if strings.Contains("/"+directory+"/", "/internal/") {
			kind = "implementation:"
		}

		for _, source := range sources {
			if strings.HasSuffix(source, "_test.go") {
				continue
			}

			parsed, err := parser.ParseFile(token.NewFileSet(), source, result.contents[source], parser.PackageClauseOnly)
			if err != nil {
				return inventory{}, err
			}

			if parsed.Name.Name == "main" {
				kind = "command:"
			}
		}

		result.add(kind+directory, directory, sources, owner, page, "Understand the delivered API and source ownership", "exported source and compiled or executed composition")
	}

	for directory, sources := range examples {
		result.add("example-composition:"+directory, directory, sources, exampleOwner(directory), directory+"/README.md", "Run the companion in the owning slice mode; later slices verify remaining workflows", "real example composition; later slices verify each workflow")
	}

	for _, source := range []string{"config/config.go", "scheduler/config.go", "settings/setting.go"} {
		if _, found := result.contents[source]; found {
			owner := strings.Split(source, "/")[0]
			result.add("configuration-api:"+source, source, []string{source}, owner, referencePage(owner), "Check public fields, defaults and validation", "effective values through current validators")
		}
	}

	for _, source := range []string{"internal/hatmaxcli/app.go", "internal/hatmaxtui/model.go"} {
		if _, found := result.contents[source]; found {
			result.add("help:"+source, source, []string{source}, "generator", referencePage("generator"), "Use current terminal commands and interactions", "actual executable help and supported interactions")
		}
	}

	result.add("dependencies:go.mod", "go.mod", []string{"go.mod", "go.sum"}, "generator", referencePage("generator"), "Reproduce source dependency and toolchain inputs", "module and checksum identity")
	slices.SortFunc(result.rows, func(a, b row) int { return strings.Compare(a.id, b.id) })

	return result, nil
}

func (i *inventory) add(id, source string, files []string, owner, page, need, method string) {
	var bound strings.Builder
	for _, file := range files {
		fmt.Fprintf(&bound, "%s\x00%s\x00", file, i.contents[file])
	}

	i.rows = append(i.rows, row{id: id, source: source, digest: digest(bound.String()), owner: owner,
		page: page, need: need, method: method, slice: ownerSlice(owner), receipt: "pending", status: "inventoried"})
}

func digest(text string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(text)))
}

func referencePage(owner string) string {
	name, found := references[owner]
	if !found {
		return "missing"
	}

	return "docs/reference/" + name + "/README.md"
}

func ownerSlice(owner string) int {
	switch owner {
	case "app", "web", "middleware", "render", "htmx", "ui", "modal", "format", "pagination", "i18n":
		return 2
	case "config", "settings", "db", "model", "validation", "seed", "log":
		return 3
	case "auth", "crypto":
		return 4
	case "mailer", "image", "pubsub", "scheduler", "telemetry", "testhelper", "fake", "slug":
		return 5
	case "generator", "cmd", "internal":
		return 6
	default:
		return 7
	}
}

func exampleOwner(directory string) string {
	switch {
	case directory == "examples/ticked":
		return "auth"
	case strings.Contains(directory, "/auth"), strings.Contains(directory, "/audit"), strings.Contains(directory, "/web"):
		return "auth"
	case strings.Contains(directory, "/dal"), strings.Contains(directory, "/list"):
		return "model"
	default:
		return "app"
	}
}

func pageOwner(file string) string {
	if !strings.HasPrefix(file, "docs/") {
		if strings.HasPrefix(file, "examples/") {
			return exampleOwner(path.Dir(file))
		}

		return strings.Split(file, "/")[0]
	}

	subject := path.Base(path.Dir(file))
	if strings.HasPrefix(file, "docs/tutorials/user-guide/") && path.Base(file) != "README.md" {
		subject = strings.TrimSuffix(path.Base(file), ".md")
	}

	switch subject {
	case "orientation", "application-anatomy", "lifecycle-and-wiring", "component-order-and-startup", "application-lifecycle", "bootstrap-application", "interfaces-and-adapters", "feature-anatomy":
		return "app"
	case "request-boundary", "http", "failure-and-security-boundaries":
		return "web"
	case "pages-and-partials", "rendering":
		return "render"
	case "server-rendered-htmx":
		return "htmx"
	case "presentation-primitives":
		return "ui"
	case "configuration", "configuration-boundaries", "configuration-and-runtime-settings":
		return "config"
	case "database", "persistence-and-migrations", "connect-postgres", "apply-migrations", "postgres-first":
		return "db"
	case "models-and-data-flow":
		return "model"
	case "forms-and-validation":
		return "validation"
	case "logging":
		return "log"
	case "authentication", "identity-and-sessions", "add-authentication", "manage-authenticators", "recover-account", "authentication-proof-and-recovery":
		return "auth"
	case "configure-mailer":
		return "mailer"
	case "store-images", "gallery":
		return "image"
	case "use-pubsub":
		return "pubsub"
	case "run-background-jobs", "events-and-background-work":
		return "scheduler"
	case "test-with-postgres", "testing-and-evolution":
		return "testhelper"
	case "application-services":
		return "mailer"
	case "assisted-generation":
		return "generator"
	default:
		if _, found := references[subject]; found {
			return subject
		}

		return "navigation"
	}
}

func readerNeed(file string) string {
	switch {
	case strings.HasPrefix(file, "docs/tutorials/"):
		return "Tutorial: learn one connected application-building step"
	case strings.HasPrefix(file, "docs/how-to/"):
		return "How-to: complete one known task and observe its result"
	case strings.HasPrefix(file, "docs/reference/"):
		return "Reference: find current contracts, limits and public names"
	case strings.HasPrefix(file, "docs/explanation/"):
		return "Explanation: understand implemented boundaries and tradeoffs"
	default:
		return "Product entrypoint or package/example implementation note"
	}
}

var fencePattern = regexp.MustCompile("(?m)^(`{3,}|~{3,})([^\\n]*)\\n")

func blocks(text string) []string {
	var result []string

	matches := fencePattern.FindAllStringIndex(text, -1)
	for index := 0; index+1 < len(matches); index += 2 {
		result = append(result, text[matches[index][0]:matches[index+1][1]])
	}

	return result
}

func blockMethod(block string) string {
	first := strings.TrimSpace(strings.SplitN(block, "\n", 2)[0])

	language := strings.TrimLeft(first, "`~")
	switch language {
	case "go":
		if strings.Contains(block, "package main") && strings.Contains(block, "func main()") {
			return "executable Go example"
		}

		return "contextual Go fragment (owning composition required)"
	case "bash", "sh", "shell", "console":
		return "published command procedure"
	case "yaml", "yml", "json", "toml":
		return "configuration fragment (effective validator required)"
	case "html", "sql", "javascript", "js":
		return "contextual template or query (owning composition required)"
	default:
		return "contextual output or notation (source comparison required)"
	}
}

func render(rows []row, revision string) string {
	var output strings.Builder
	output.WriteString("<!--\nSPDX-FileCopyrightText: 2026 Adrian PK\nSPDX-License-Identifier: Apache-2.0\n\nThis file is part of Hatmax. See LICENSE for license terms.\n-->\n\n# Hatmax Documentation Conformance Coverage\n\n")
	fmt.Fprintf(&output, "Status: Inventoried; behavioral verification pending\nActivation revision: `%s`\n\n", revision)
	output.WriteString("## Inventory Contract\n\n")
	output.WriteString("Discover Go package directories, executables, example compositions, public configuration APIs, example YAML, embedded Book data, terminal help, dependencies and product Markdown independently from this table. Include nested and internal implementation owners so public terminal and example behavior has a source binding. Exclude repository operations, agent rules, third-party material, validation tooling, and testdata/fixtures from product-page discovery; these are not published product documentation. Package source bindings include tracked tests. Complete example bindings additionally include assets, SQL migrations, configuration and browser test fixtures. Book bindings include both delivered releases and their contextual positive/negative fragments.\n\n")
	output.WriteString("Every row records its inspected revision and SHA-256 content identity. Directory digests bind sorted source paths and bytes; page and file digests bind path and bytes; block digests bind exact fenced content. Newly added or removed sources, pages and blocks invalidate reconciliation. Later slices must reconcile changed bindings and retain only receipts whose source, snippet, tool and configuration inputs still match.\n\n")
	output.WriteString("`inventoried` means identified and assigned, not behaviorally verified. `pending` receipts cannot establish execution. Missing surface documentation stays `missing` until its owner supplies coverage. Contextual fragments require the complete owning composition; executable commands require observed outcomes. A walkthrough row covers instructions outside fenced blocks. Owning slices 2 through 6 perform source/API comparison and real execution; Slice 7 reconciles navigation and complete acceptance.\n\n")
	output.WriteString("Reproduce the initial table with `GOWORK=off go run ./scripts/documentation-conformance inventory`. This writes a fresh inventory to stdout and does not preserve later verification receipts. Validate the maintained table with `GOWORK=off go run ./scripts/documentation-conformance check`.\n\n")
	output.WriteString("## Coverage Rows\n\n| ID | Source owner | Inspected revision | SHA-256 | Owner | Slice | Page | Reader need | Verification | Receipt | Status |\n| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")

	for _, r := range rows {
		fmt.Fprintf(&output, "| %s | %s | %s | %s | %s | %d | %s | %s | %s | %s | %s |\n", r.id, r.source, revision, r.digest, r.owner, r.slice, r.page, r.need, r.method, r.receipt, r.status)
	}

	return output.String()
}

func parseRows(text string) ([]row, error) {
	var rows []row

	for _, line := range strings.Split(text, "\n") {
		if !strings.HasPrefix(line, "| ") || strings.HasPrefix(line, "| ID ") || strings.HasPrefix(line, "| ---") {
			continue
		}

		fields := strings.Split(strings.Trim(line, "|"), "|")
		if len(fields) != 11 {
			return nil, fmt.Errorf("invalid coverage row: %s", line)
		}

		for index := range fields {
			fields[index] = strings.TrimSpace(fields[index])
			if fields[index] == "" {
				return nil, fmt.Errorf("empty coverage field: %s", fields[0])
			}
		}

		number, err := strconv.Atoi(fields[5])
		if err != nil || number < 1 || number > 7 {
			return nil, fmt.Errorf("invalid slice: %s", fields[0])
		}

		rows = append(rows, row{id: fields[0], source: fields[1], revision: fields[2], digest: fields[3], owner: fields[4], slice: number, page: fields[6], need: fields[7], method: fields[8], receipt: fields[9], status: fields[10]})
	}

	return rows, nil
}

var revisionPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

func reconcile(discovered, recorded []row) error {
	known := make(map[string]row)
	for _, r := range recorded {
		if _, duplicate := known[r.id]; duplicate {
			return fmt.Errorf("duplicate identity: %s", r.id)
		}

		if !revisionPattern.MatchString(r.revision) {
			return fmt.Errorf("invalid inspected revision: %s", r.id)
		}

		verified := r.slice == 2 && r.receipt == "slice-2/runtime-receipts.json" &&
			slices.Contains([]string{"source-inspected", "compiled", "rendered", "executed", "package-tests"}, r.status)

		verified = verified || (r.slice == 3 && r.receipt == "slice-3/data-receipts.json" &&
			slices.Contains([]string{"source-inspected", "compiled", "executed", "package-tests"}, r.status))

		verified = verified || (r.slice == 4 && r.receipt == "slice-4/identity-receipts.json" &&
			slices.Contains([]string{"source-inspected", "compiled", "executed", "package-tests"}, r.status))

		verified = verified || (r.slice == 5 && r.receipt == "slice-5/infrastructure-receipts.json" &&
			slices.Contains([]string{"source-inspected", "compiled", "executed", "package-tests"}, r.status))

		verified = verified || (r.slice == 6 && r.receipt == "slice-6/generator-receipts.json" &&
			slices.Contains([]string{"source-inspected", "executed", "package-tests", "blocked"}, r.status))
		if (r.status != "inventoried" || r.receipt != "pending") && !verified {
			return fmt.Errorf("unsupported status or receipt; future rows require pending receipts only: %s", r.id)
		}

		known[r.id] = r
	}

	for _, current := range discovered {
		previous, found := known[current.id]
		if !found {
			return fmt.Errorf("unaccounted surface/page/example: %s", current.id)
		}

		if previous.source != current.source || previous.digest != current.digest {
			return fmt.Errorf("source drift: %s", current.id)
		}

		if previous.owner != current.owner || previous.slice != current.slice {
			return fmt.Errorf("source ownership drift: %s", current.id)
		}

		if previous.page != "missing" {
			_, err := os.Stat(previous.page)
			if err != nil {
				return fmt.Errorf("missing coverage page: %s: %w", current.id, err)
			}
		}

		delete(known, current.id)
	}

	if len(known) != 0 {
		return fmt.Errorf("stale coverage identities: %d", len(known))
	}

	return nil
}

var inlineLink = regexp.MustCompile(`\]\(([^)]+)\)`)
var referenceLink = regexp.MustCompile(`(?m)^\s*\[[^\]]+\]:\s*(\S+)`)
var heading = regexp.MustCompile(`^ {0,3}#{1,6}\s+(.+?)\s*#*\s*$`)
var htmlAnchor = regexp.MustCompile(`(?:id|name)=["']([^"']+)["']`)

func anchors(text string) map[string]bool {
	result := make(map[string]bool)
	counts := make(map[string]int)
	fenced := false

	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") || strings.HasPrefix(strings.TrimSpace(line), "~~~") {
			fenced = !fenced

			continue
		}

		if fenced {
			continue
		}

		match := heading.FindStringSubmatch(line)
		if len(match) == 0 {
			continue
		}

		var slug strings.Builder

		for _, char := range strings.ToLower(match[1]) {
			switch {
			case unicode.IsLetter(char), unicode.IsNumber(char), char == '-', char == '_':
				slug.WriteRune(char)
			case unicode.IsSpace(char):
				slug.WriteRune('-')
			}
		}

		name := slug.String()
		count := counts[name]

		counts[name]++
		if count > 0 {
			name += "-" + strconv.Itoa(count)
		}

		result[name] = true
	}

	for _, match := range htmlAnchor.FindAllStringSubmatch(text, -1) {
		result[match[1]] = true
	}

	return result
}

func checkNavigation(contents map[string]string) error {
	edges := make(map[string][]string)

	for source, text := range contents {
		if !productPage(source) {
			continue
		}

		links := append(inlineLink.FindAllStringSubmatch(text, -1), referenceLink.FindAllStringSubmatch(text, -1)...)
		for _, match := range links {
			target := strings.Trim(match[1], "<>")
			if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
				continue
			}

			file, anchor, _ := strings.Cut(target, "#")

			destination := source
			if file != "" {
				destination = path.Clean(path.Join(path.Dir(source), file))
			}

			_, err := os.Stat(destination)
			if err != nil {
				return fmt.Errorf("broken local link: %s -> %s", source, target)
			}

			if anchor != "" && strings.HasSuffix(strings.ToLower(destination), ".md") && !anchors(contents[destination])[anchor] {
				return fmt.Errorf("broken heading anchor: %s -> %s", source, target)
			}

			edges[source] = append(edges[source], destination)
		}
	}
	// Require a path from each quadrant index; a circular orphan is not reachable.
	for _, quadrant := range []string{"tutorials", "how-to", "reference", "explanation"} {
		root := "docs/" + quadrant + "/README.md"
		visited := make(map[string]bool)

		queue := []string{root}
		for len(queue) != 0 {
			next := queue[0]
			queue = queue[1:]

			if visited[next] {
				continue
			}

			visited[next] = true
			queue = append(queue, edges[next]...)
		}

		for file := range contents {
			if strings.HasPrefix(file, "docs/"+quadrant+"/") && productPage(file) && !visited[file] {
				return fmt.Errorf("unreachable from quadrant index: %s", file)
			}
		}
	}

	return nil
}
