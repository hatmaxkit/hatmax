<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 2: Runtime and Presentation

Status: reviewing
Delivery set: documentation-conformance
Plan: [Delivery plan](../../../plan/documentation-conformance.md)
Tracker: [Delivery tracker](../../../tracker/documentation-conformance.md)
Branch: `docs/hatmax-runtime-presentation`
PR: `#113`
Report introduction: `79a9665a852ffd11f0383146ae745ada41f5d0da`
Activation revision: `84180a9b003fb661604f612d7015e39d03a203d0`
Rebased dev revision: `064c5490f8f97f730cd37daaf80de6492d495cec`

## Purpose

Reconcile application assembly, lifecycle, HTTP and presentation guidance with
current source and bind the published examples to reproducible local evidence.

## Delivered Behavior

The runtime chapters and package notes describe middleware-before-routes ordering,
component-owned rollback, caller-owned servers and bounded HTTP draining. Request
and presentation guidance distinguishes full/partial rendering, committed response
errors, exact HTMX redirect headers, startup-only translation/currency setup,
pagination inputs, CSRF integration and application-owned policy.

The bootstrap procedure selects a current source checkout explicitly because the
published v0.5.0 module has the older Serve contract. Its complete Go/YAML example
builds, serves the documented health JSON, handles Ctrl+C with exit 0, and leaves
its listener unavailable after shutdown. The Guide plain mode serves a full page
and name partial, accepts the matching-origin form, rejects absent/cross-site
origins, escapes submitted HTML, reports invalid input and returns an ordinary
303 redirect. An owned HTTP listener observes 200 HX-Redirect only for the exact
HX-Request value true; absent or uppercase TRUE produces 303 Location.

Coverage reconciles 448 independently discovered identities. The 141 Slice 2
bindings distinguish nine non-empty package suites, 44 compiled Go fragments,
20 executed fragment/procedure/composition bindings, ten template-rendering
bindings and 58 source-inspected pages or contextual notation. Remaining slices
retain pending receipts. Guide database execution belongs to Slice 3; the complete
Ticked composition belongs to Slice 4.

## Implementation Notes

The runner keeps the required docs, source-license and strict-lint order and adds
fresh runtime evidence for Slice 2. It resolves Go 1.27.1, retains safe diagnostics
in a unique ignored directory, and records exact HEAD, input digests, compiler
configuration, subprocess arguments/exits, log digests and process configuration.
Missing bindings, changed content/tool/head, unexpected outcomes or empty package
test execution cannot supply a pass. Build/module subprocess groups and owned
process shutdown have explicit bounds; only the invocation's processes stop.

Published Go/HTML bytes are extracted from Markdown. Typed adapters supply the
explicitly application-owned invoice methods, row templates, settings and CSRF
callback. Invoice/policy adapters establish compilation, not CRUD, database or
security-policy execution. Selected fragments additionally assert actual HTMX
attributes/headers, numeric outputs, translations/fallback, pagination, modal
configuration, form parsing and rendering. The published locale YAML is loaded
through i18n.LoadFromFS. Render has no package tests; its published template is
executed rather than receiving an empty-suite pass.

The bootstrap setup executes module init/edit/get/tidy against an immutable local
clone with the documented relative replacement. The gate substitutes the owned
clone for the moving remote download, build plus direct binary launch for the
Go run supervisor, and fresh loopback ports for 8080. Requests use a local HTTP
client with proxying and automatic redirects disabled. The Guide terminates by
owned interrupt; its main has no graceful coordinator, so bootstrap supplies that
proof. Template execution is local server rendering, without a browser DOM-swap
claim. Forwarding/proxy guidance retains its source-inspection boundary.

## Contracts Added or Changed

Slice 2 is available through scripts/check-documentation-conformance.sh slice 2;
Slice 1 remains available and integrated/future modes still reject early.
Coverage receipt names resolve to each fresh invocation's runtime-receipts.json;
they identify a reproducer rather than cached success. Structural reconciliation
also rejects changed source ownership. Evidence reconciliation checks the
maintained status against the actual verification method.

Contextual examples identify the bootstrap, Guide or illustrative invoice
composition. Error checks, valid template builders and separate Go/HTML contexts
replace invalid fragments. The contextual compiler found repeated short
declarations in render's construction block; distinct names now let the entire
block compile. Toolkit runtime code and dependencies are unchanged.

## Files of Interest

- [Bootstrap procedure](../../../../../docs/how-to/bootstrap-application/README.md)
- [Lifecycle reference](../../../../../docs/reference/application-lifecycle/README.md)
- [Guide companion](../../../../../examples/guide/README.md)
- [Coverage record](../../documentation-conformance-coverage.md)
- [Slice runner](../../../../../scripts/check-documentation-conformance.sh)
- [Runtime evidence](../../../../../scripts/documentation-conformance/runtime.go)
- [Contextual compiler](../../../../../scripts/documentation-conformance/contexts.go)
- [Negative controls](../../../../../scripts/documentation-conformance/runtime_test.go)

## Validation

Focused local checks passed while implementing:

- `GOWORK=off GOFLAGS=-p=2 go test -count=1 -timeout=2m ./scripts/documentation-conformance` — passed, including omitted/stale/misclassified/duplicate receipts, failed commands, malformed source and unaccounted command procedures.
- `GOWORK=off go run ./scripts/documentation-conformance check` — passed for 448 identities and quadrant navigation.
- `GOWORK=off GOFLAGS=-p=2 go run ./scripts/documentation-conformance evidence .tmp/documentation-conformance/slice-2.develop.Gf9INUjb/runtime-receipts.json` — passed for the 141 fresh runtime bindings after actual fixture execution.
- `bash -n scripts/check-documentation-conformance.sh` — passed.
- `GOWORK=off make lint-strict` — passed with zero issues.

The repair rebases this candidate onto current dev, including its approved
native-tooling closure amendment. Task, report-introduction and inspected
revision references identify the replayed commits. Earlier successful checks
remain historical evidence; the controller regenerates the ready gate after
the repair reference follow-up is pushed,
on the exact clean PR head, in this order:

- `GOWORK=off scripts/check-documentation-conformance.sh slice 2`
- `git diff --check`

The slice command contains the required repository checks and the focused
source/example execution above. This report's introduction commit precedes that
final head; successful exact-head results reside in controller evidence and the
canonical PR. Integrated documentation acceptance remains with Slice 7.

## Risks and Follow-ups

The approved native-tooling closure amendment is included from canonical dev; these
validators and fixture utilities use Go/shell. Slice 7 owns its final constrained
toolchain acceptance. Later slices verify database-backed Guide and Ticked journeys and complete
cross-quadrant acceptance. Conceptual chapters retain source inspection; compile
adapters and local template rendering do not establish external deployment,
authentication proof or browser DOM behavior. Inspected content changes require
new bindings and fresh receipts.
