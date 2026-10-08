#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adrian PK
# SPDX-License-Identifier: Apache-2.0
#
# This file is part of Hatmax. See LICENSE for license terms.

set -euo pipefail

if [[ $# != 2 || "$1" != slice || "$2" != 1 ]]; then
  echo 'Only slice 1 is implemented; integrated acceptance is not yet available.' >&2
  exit 2
fi

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"

# Explicit executable overrides apply to every nested repository command.
for tool in git sha256sum mktemp realpath; do
  command -v "$tool" >/dev/null
done
go_tool=$(realpath -- "$(command -v "${HATMAX_DOC_GO:-go}")")
make_tool=$(realpath -- "$(command -v "${HATMAX_DOC_MAKE:-make}")")
lint_tool=$(realpath -- "$(command -v "${HATMAX_DOC_LINT:-golangci-lint}")")
if [[ $("$go_tool" version) != 'go version go1.27.1 '* ]]; then
  echo 'Documentation conformance requires Go 1.27.1.' >&2
  exit 1
fi

mkdir -p .tmp/documentation-conformance
fixture=$(mktemp -d "$repo_root/.tmp/documentation-conformance/slice-1.XXXXXXXX")
if ! git check-ignore -q "$fixture/receipt.txt"; then
  echo 'Documentation fixture storage must be ignored by Git.' >&2
  exit 1
fi
mkdir -p "$fixture/bin" "$fixture/build" "$fixture/cache"
ln -s "$go_tool" "$fixture/bin/go"
ln -s "$make_tool" "$fixture/bin/make"
ln -s "$lint_tool" "$fixture/bin/golangci-lint"
export PATH="$fixture/bin:$PATH"
export GOWORK=off
export TMPDIR="$fixture/build" GOTMPDIR="$fixture/build"
export GOFLAGS="${GOFLAGS:--p=2}"
# Go and lint caches stay worktree-local; each invocation owns its temporary build files.
export GOCACHE="$repo_root/.tmp/documentation-conformance/gocache"

receipt="$fixture/receipt.txt"
{
  printf 'slice: 1\nhead: %s\n' "$(git rev-parse HEAD)"
  go version
  golangci-lint version
  printf 'GOFLAGS: %s\nGOWORK: off\n' "$GOFLAGS"
  sha256sum go.mod go.sum ops/default/report/documentation-conformance-coverage.md
} > "$receipt"

check_number=0
run_check() {
  local label=$1
  shift
  check_number=$((check_number + 1))
  printf 'Running %s\n' "$label"
  printf 'command: %s\nexpected: exit 0\n' "$label" >> "$receipt"
  local result=0
  "$@" > "$fixture/check-$check_number.log" 2>&1 || result=$?
  cat "$fixture/check-$check_number.log"
  printf 'exit: %s\n' "$result" >> "$receipt"
  if ((result != 0)); then
    printf 'Documentation conformance failed: %s\n' "$label" >&2
    return "$result"
  fi
}

run_check 'make docs-check' make docs-check
run_check 'make source-license-check' make source-license-check
run_check 'make lint-strict' make lint-strict
run_check 'go run ./scripts/documentation-conformance check' go run ./scripts/documentation-conformance check
run_check 'go test -count=1 -timeout=2m ./scripts/documentation-conformance' go test -count=1 -timeout=2m ./scripts/documentation-conformance
printf 'Slice 1 documentation conformance passed. Safe diagnostics retained in ignored fixture storage.\n'
