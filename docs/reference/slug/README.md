<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Slug

`slug` builds an ASCII kebab-case string. The implementation note is
[slug/readme.md](../../../slug/readme.md).

`DefaultMaxLength` is 50. `UUIDPrefixLength` is 8.

`Normalize` returns `""` for empty text or a `maxLength` that is not positive.
Otherwise it removes Unicode marks, lowercases the text, turns every run of
characters outside `a-z` and `0-9` into one hyphen, and trims hyphens. A
result longer than `maxLength` is cut at the last hyphen that remains inside
that limit, or at `maxLength` when no hyphen remains, and then trimmed again.

`Generate` normalizes the text with `DefaultMaxLength` and appends `-` plus
the first 8 characters of `id.String()`. An empty normalization returns only
that prefix.
