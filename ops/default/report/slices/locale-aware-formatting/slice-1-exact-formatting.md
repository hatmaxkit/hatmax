<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 1: Exact Localized Display

Status: reviewing
Delivery set: locale-aware-formatting
Plan: [Delivery plan](../../../plan/locale-aware-formatting.md)
Tracker: [Delivery tracker](../../../tracker/locale-aware-formatting.md)
Branch: `feat/locale-aware-formatting`
PR: `#128` ([canonical PR](https://forge.adrianpk.com/hatmax/hatmax/pulls/128))

## Purpose

Display exact integers, decimal quantities and minor-unit amounts in an explicit
caller-selected locale, independently of storage and arithmetic.

## Delivered Behavior

`format.NewFormatter` constructs an immutable locale formatter. `Integer` accepts
signed 64-bit values, `Decimal` preserves canonical decimal digits and fractional
zeros, and `MinorUnits` preserves an explicit scale from zero through nine.
Negative zero normalizes without losing scale. All methods return ordinary text.

| Criterion | Evidence |
| --- | --- |
| LF01 | `TestFormatter/display` covers English, Spanish, Polish, French, German, Indian grouping, Arabic/Persian digits and signs, and regional fallback. |
| LF02 | Exact values beyond floating-point precision, 128 digits, 127 fractional digits and trailing zeros have explicit expected results. |
| LF03 | Zero, tiny negative values, both int64 extremes and scales zero, two and nine have exact expected results. |
| LF04 | Canonical locale admission, 63-byte locale and 128-digit decimal boundaries, malformed decimals and invalid scales are exercised. Errors return no formatted result. |
| LF05 | Parallel display cases repeatedly share immutable instances across locales and operations; the configured race check covers this family. |
| LF06 | Existing number/price helpers remain unchanged. `ExampleFormatter` composes the real render/UI maps and verifies normal escaping alongside localized operations. |

## Implementation Notes

The pinned `x/text` number API has no public arbitrary-decimal input. A padded
integer supplies the locale's digit positions, grouping, decimal separator and
sign affixes. The formatter inserts validated source digits into those positions
without converting the input through floating point. Each call owns its printer.
Minor units split `strconv.FormatInt` text, avoiding negation overflow at MinInt64.
There is no global locale, cache, new dependency or currency registry.

## Contracts Added or Changed

The additive `Formatter` API accepts canonical ASCII BCP 47 tags of 1–63 bytes,
canonical decimals of at most 128 ASCII digits and explicit minor-unit scales
zero through nine. The zero-value formatter uses the root locale. Applications
own locale defaults, currency identity, scale, arithmetic and error handling.
Existing exported helpers and built-in template functions retain their behavior.

## Files of Interest

- [Formatter](../../../../../format/locale.go)
- [Behavior cases](../../../../../format/locale_test.go)
- [Executable template example](../../../../../format/locale_example_test.go)
- [Package usage](../../../../../format/readme.md)
- [API reference](../../../../../docs/reference/format/README.md)
- [Presentation guide](../../../../../docs/tutorials/user-guide/presentation-primitives.md)

## Validation

Focused implementation checks passed:

- `GOWORK=off go test ./format -run '^TestFormatter$' -count=1`
- `GOWORK=off go test ./format -run '^(TestNumber|TestInteger|TestPrice|TestPriceWithDecimals|TestPriceRange|TestRegisterCurrency|ExampleFormatter)$' -count=1`
- `GOCACHE="$PWD/.tmp/lint/gocache" golangci-lint run --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 format/locale.go format/locale_test.go format/locale_example_test.go`

The first formatter invocation took approximately 0.3 seconds; the template and
legacy-helper check took approximately 2.4 seconds. Initial assertions were
aligned with the pinned Austrian grouping and parser-recognized region fallback.
Implementation lint whitespace findings were corrected before the task commits.

The final clean PR HEAD is validated through the controller after the PR-reference
follow-up commit. Its exact commit, elapsed time and ordered command outcomes
belong to the controller receipt; the canonical PR lists successful commands:

1. `GOWORK=off go vet ./format ./render ./ui`
2. `GOWORK=off go test -race -count=1 -timeout=120s ./format ./render ./ui`
3. `GOCACHE="$PWD/.tmp/lint/gocache" golangci-lint run --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 ./format`
4. `make docs-check`
5. `git diff --check`

Task commits: `0078533f8a40999651927592e50edfa68422777a` implements T1.1;
`ca0e8d5aeb17b151ee48a001c1e92353d419b186` delivers T1.2 and introduces this
canonical report. The final PR-reference commit is the controller's validation
candidate; it does not change runtime behavior.

These are the approved affected-package and documentation checks. Integrated
candidate validation and delivery-set closure remain controller-owned.

## Risks and Follow-ups

Locale output follows the pinned `x/text` data, including nonbreaking spaces and
bidirectional sign marks. Applications must preserve those strings for display
and keep financial arithmetic in their domain types.
