<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Locale-Aware Exact Number Formatting

Date: 2026-10-18
Status: Approved
Approved: 2026-10-18
Concern: locale-aware-formatting
Inspected baseline: `3e1b31b63db934d7d8b1c1287db68b6f500a7c27`
Existing package: [format](../../../format/readme.md)
Implementation status: Completed
Completed: 2026-10-18
Validated candidate: `ff45f00bf867d1763376c21b941946c74878116a`
Published module: `hatmax.adrianpk.com@v0.6.1-0.20261018112343-ff45f00bf867`
Report: [Slice 1](../report/slices/locale-aware-formatting/slice-1-exact-formatting.md)
Plan: [Delivery plan](../plan/locale-aware-formatting.md)
Tracker: [Delivery tracker](../tracker/locale-aware-formatting.md)
Planning authorized: 2026-10-18

## Purpose

Let any HatMax application display exact integers, decimal quantities and
minor-unit amounts using an explicitly selected locale. Formatting is a
presentation operation; it must not change stored values or financial arithmetic.

The existing `format.Number` and `format.Integer` use an English printer.
The price helpers accept floating-point values and a shared currency registry.
The inspected baseline had no caller-owned locale formatter for exact decimal
input. The delivered capability lives inside `format` and preserves the
existing functions and their output.

## Caller Contract

`NewFormatter` constructs an immutable `Formatter` from an explicit BCP 47
locale. Its `Integer`, `Decimal` and `MinorUnits` methods provide this behavior:

| Operation | Input | Output |
| --- | --- | --- |
| Construct | Canonical locale tag | Formatter or a configuration error |
| Integer | Signed 64-bit integer | Localized grouping and sign, without fractional digits |
| Decimal | Canonical signed decimal string | Localized grouping and decimal separator, preserving supplied fractional digits |
| Minor-unit amount | Signed 64-bit integer and explicit decimal scale | Localized decimal amount with exactly that many fractional digits |

Construction accepts canonical ASCII tags of 1–63 bytes recognized by the
existing `golang.org/x/text/language` parser. Use the pinned locale data's normal
language/region fallback; reject malformed tags instead of guessing from process
environment. Applications choose their default locale before construction.
Support is extensible through that locale data, not an enum of five languages.

Decimal input is bounded to 128 ASCII digits excluding its optional leading
minus sign and decimal point. Require at least one integer digit, no redundant
leading zeros except zero itself, and at least one fractional digit after a
point. Reject whitespace, plus signs, exponents, grouping, NaN and infinity.
Preserve fractional zeros; normalize negative zero to unsigned zero with the
same scale. Minor-unit scale is an integer from zero through nine. This is an
explicit caller contract, not a guessed currency exponent.

## Exactness and Ownership

Never convert exact input through `float32` or `float64`. Handle negative
amounts and the minimum signed 64-bit integer without intermediate overflow.
Formatting does not round, convert currencies, apply tax, parse user input,
or infer a locale from a request, cookie or global setting.

Applications retain currency identity and decide how to label an amount. A
caller can combine an ISO currency code with the formatted minor-unit amount;
this capability does not add another currency registry or promise automatic
symbol disambiguation. The caller supplies the scale from its monetary contract.

Use the existing `golang.org/x/text` dependency and its locale conventions.
Return plain strings that remain subject to Go template escaping. Instances
must be safe for concurrent use, with no mutable process-wide current locale,
shared printer mutation, or unbounded global cache. No persistence or model
companion is required.

## Compatibility and Documentation

Existing number, price, currency-registration and template functions retain
their signatures and semantics. The new operations are additive and opt-in.
Do not change the old floating-point helpers to claim exact monetary semantics.

Update the package reference and presentation guide with complete examples of
integer, decimal and minor-unit display. Show caller-selected locales and an
unambiguous currency label. Explain the difference between display and arithmetic.

## Acceptance

| ID | Observable requirement |
| --- | --- |
| LF01 | The same value follows English, Spanish, Polish, French and German grouping/decimal conventions; a further supported locale requires no new language enum. |
| LF02 | Decimal precision and trailing fractional zeros survive formatting; large integers never lose digits through floating-point conversion. |
| LF03 | Minor-unit amounts preserve exact scale, zero, negatives and the minimum signed 64-bit value. |
| LF04 | Malformed locales, malformed decimals, excessive input and invalid scales return bounded errors without partial results. |
| LF05 | Concurrent instances with different locales produce stable independent results. |
| LF06 | Existing public helpers and their documented output remain compatible; examples are escaped normally by templates. |

Use one compact table-driven family for these deterministic transformations,
with an affected-package race check for concurrent use and existing reference
checks. A real template example establishes presentation composition. Do not
add application, provider or browser suites to verify a pure formatter, and do
not run the repository's exhaustive gate for each implementation step.
