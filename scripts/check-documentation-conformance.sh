#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adrian PK
# SPDX-License-Identifier: Apache-2.0
#
# This file is part of Hatmax. See LICENSE for license terms.

set -euo pipefail

if [[ $# != 2 || "$1" != slice || ("$2" != 1 && "$2" != 2 && "$2" != 3 && "$2" != 4 && "$2" != 5 && "$2" != 6) ]]; then
  echo 'Only slices 1 through 6 are implemented; integrated acceptance is not yet available.' >&2
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
slice_number=$2
fixture=$(mktemp -d "$repo_root/.tmp/documentation-conformance/slice-$slice_number.XXXXXXXX")
if ! git check-ignore -q "$fixture/receipt.txt"; then
  echo 'Documentation fixture storage must be ignored by Git.' >&2
  exit 1
fi
mkdir -p "$fixture/bin" "$fixture/build" "$fixture/cache"
ln -s "$go_tool" "$fixture/bin/go"
ln -s "$make_tool" "$fixture/bin/make"
ln -s "$lint_tool" "$fixture/bin/golangci-lint"
if ((slice_number >= 3)); then
  sqlc_command=${HATMAX_DOC_SQLC:-sqlc}
  if [[ -z ${HATMAX_DOC_SQLC:-} ]] && ! command -v sqlc >/dev/null; then
    sqlc_command="$repo_root/.tmp/documentation-conformance/tools/sqlc"
  fi
  sqlc_tool=$(realpath -- "$(command -v "$sqlc_command")")
  ln -s "$sqlc_tool" "$fixture/bin/sqlc"
  for tool in initdb pg_ctl postgres psql createuser createdb; do
    tool_command=$tool
    if [[ -n ${HATMAX_DOC_PG_BINDIR:-} ]]; then
      tool_command="$HATMAX_DOC_PG_BINDIR/$tool"
    fi
    tool_path=$(realpath -- "$(command -v "$tool_command")")
    ln -s "$tool_path" "$fixture/bin/$tool"
  done
fi
if ((slice_number >= 4)); then
  chromium_command=${HATMAX_DOC_CHROMIUM:-chromium}
  # Arch's launcher is a shell script. Prefer its actual native image so the
  # receipt binds executable bytes as well as the observed browser version.
  if [[ -z ${HATMAX_DOC_CHROMIUM:-} && -x /usr/lib/chromium/chromium ]]; then
    chromium_command=/usr/lib/chromium/chromium
  fi
  export CHROMIUM_BIN
  CHROMIUM_BIN=$(realpath -- "$(command -v "$chromium_command")")
fi
export PATH="$fixture/bin:$PATH"
export GOWORK=off
export TMPDIR="$fixture/build" GOTMPDIR="$fixture/build"
export GOFLAGS="${GOFLAGS:--p=2}"
# Go and lint caches stay worktree-local; each invocation owns its temporary build files.
export GOCACHE="$repo_root/.tmp/documentation-conformance/gocache"

receipt="$fixture/receipt.txt"
{
  printf 'slice: %s\nhead: %s\n' "$slice_number" "$(git rev-parse HEAD)"
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
  {
    printf 'label: %s\ncommand:' "$label"
    printf ' %q' "$@"
    printf '\nexpected: exit 0\n'
  } >> "$receipt"
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
if ((slice_number >= 2)); then
  runtime_fixture=$fixture
  if ((slice_number >= 3)); then
    runtime_fixture=$(mktemp -d "$repo_root/.tmp/documentation-conformance/slice-2.cumulative.XXXXXXXX")
  fi
  run_check 'go run ./scripts/documentation-conformance runtime <owned-fixture>' go run ./scripts/documentation-conformance runtime "$runtime_fixture"
  run_check 'go run ./scripts/documentation-conformance evidence <owned-fixture>/runtime-receipts.json' go run ./scripts/documentation-conformance evidence "$runtime_fixture/runtime-receipts.json"
fi
if ((slice_number >= 3)); then
  data_fixture=$fixture
  if ((slice_number >= 4)); then
    data_fixture=$(mktemp -d "$repo_root/.tmp/documentation-conformance/slice-3.cumulative.XXXXXXXX")
  fi
  run_check 'owned PostgreSQL and go run ./scripts/documentation-conformance data <owned-fixture>' scripts/documentation-conformance/data-fixture.sh "$data_fixture" go run ./scripts/documentation-conformance data "$data_fixture"
  run_check 'go run ./scripts/documentation-conformance data-evidence <owned-fixture>/data-receipts.json' go run ./scripts/documentation-conformance data-evidence "$data_fixture/data-receipts.json"
fi
if ((slice_number >= 4)); then
  identity_fixture=$fixture
  if ((slice_number >= 5)); then
    identity_fixture=$(mktemp -d "$repo_root/.tmp/documentation-conformance/slice-4.cumulative.XXXXXXXX")
  fi
  run_check 'owned PostgreSQL and go run ./scripts/documentation-conformance identity <owned-fixture>' scripts/documentation-conformance/data-fixture.sh "$identity_fixture" go run ./scripts/documentation-conformance identity "$identity_fixture"
  run_check 'go run ./scripts/documentation-conformance identity-evidence <owned-fixture>/identity-receipts.json' go run ./scripts/documentation-conformance identity-evidence "$identity_fixture/identity-receipts.json"
fi
if ((slice_number >= 5)); then
  infrastructure_fixture=$fixture
  if ((slice_number >= 6)); then
    infrastructure_fixture=$(mktemp -d "$repo_root/.tmp/documentation-conformance/slice-5.cumulative.XXXXXXXX")
  fi
  run_check 'owned PostgreSQL and go run ./scripts/documentation-conformance infrastructure <owned-fixture>' scripts/documentation-conformance/data-fixture.sh "$infrastructure_fixture" go run ./scripts/documentation-conformance infrastructure "$infrastructure_fixture"
  run_check 'go run ./scripts/documentation-conformance infrastructure-evidence <owned-fixture>/infrastructure-receipts.json' go run ./scripts/documentation-conformance infrastructure-evidence "$infrastructure_fixture/infrastructure-receipts.json"
fi
if ((slice_number >= 6)); then
  run_check 'owned PostgreSQL and go run ./scripts/documentation-conformance generator <owned-fixture>' scripts/documentation-conformance/data-fixture.sh "$fixture" go run ./scripts/documentation-conformance generator "$fixture"
  run_check 'go run ./scripts/documentation-conformance generator-evidence <owned-fixture>/generator-receipts.json' go run ./scripts/documentation-conformance generator-evidence "$fixture/generator-receipts.json"
fi
if ((slice_number == 6)); then
  printf 'Slice 6 evidence checks passed; blocked generator workflows remain unresolved. Safe diagnostics retained in ignored fixture storage.\n'
else
  printf 'Slice %s documentation conformance passed. Safe diagnostics retained in ignored fixture storage.\n' "$slice_number"
fi
