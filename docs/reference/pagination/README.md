<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Pagination

`pagination` calculates page bounds. It does not read an HTTP request. The
implementation note is [pagination/readme.md](../../../pagination/readme.md).

`DefaultPageSize` is 20. `MaxPageSize` is 100.

`NewParams` replaces a page below 1 with 1. A page size of 0 or less becomes
20. A page size above 100 becomes 100.

`Offset` is `(Page - 1) * PageSize`. `Limit` is `PageSize`.

`NewResult` sets `TotalPages` to `TotalCount / PageSize`, plus one when the
division has a remainder. `HasMore` is true when `Page` is below
`TotalPages`. A zero `PageSize` panics on that division.

`HasPrevious` is true when `Page` is above 1. `PreviousPage` subtracts one,
and returns 1 on the first page. `NextPage` adds one when `HasMore` is true,
and otherwise returns `Page`.

`IsEmpty` is true when `Items` has length zero. `StartIndex` and `EndIndex`
are zero for an empty result. Otherwise `StartIndex` is
`(Page - 1) * PageSize + 1`. `EndIndex` is `Page * PageSize`, limited to
`TotalCount`.
