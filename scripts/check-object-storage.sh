#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adrian PK
# SPDX-License-Identifier: Apache-2.0
#
# This file is part of Hatmax. See LICENSE for license terms.

set -euo pipefail

# Only implemented modes are admitted. Full acceptance is introduced by T5.2.
if [[ $# != 2 || $1 != slice || $2 != 1 ]]; then
  echo 'Usage: check-object-storage.sh slice 1 (full acceptance is not available yet)' >&2
  exit 2
fi

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"
export GOWORK=off
if [[ $(go version) != 'go version go1.27.1 '* ]]; then
  echo 'Object storage validation requires Go 1.27.1.' >&2
  exit 1
fi

for tool in git rg timeout sha256sum golangci-lint; do
  command -v "$tool" >/dev/null
done

mkdir -p .tmp/object-storage
run_root=$(mktemp -d "$repo_root/.tmp/object-storage/slice-1.XXXXXXXX")
if ! git check-ignore -q "$run_root/receipt.txt"; then
  echo 'Object storage validation output must be ignored by Git.' >&2
  exit 1
fi

mkdir -p "$run_root/build"
export TMPDIR="$run_root/build" GOTMPDIR="$run_root/build"
export GOFLAGS="${GOFLAGS:--p=2}"
export GOLANGCI_LINT_CACHE="$repo_root/.tmp/object-storage/lint-cache"
receipt="$run_root/receipt.txt"
candidate=$(git rev-parse HEAD)
{
  printf 'slice: 1\nhead: %s\n' "$candidate"
  go version
  golangci-lint version
  sha256sum go.mod go.sum
  # These tools are prerequisites for the later complete native PostgreSQL gate.
  for tool in initdb pg_ctl psql; do
    if command -v "$tool" >/dev/null; then
      printf 'future-full-gate-prerequisite: %s available\n' "$tool"
    else
      printf 'future-full-gate-prerequisite: %s missing\n' "$tool"
    fi
  done
} > "$receipt"

check_number=0
last_log=
run_check() {
  local label=$1
  shift
  check_number=$((check_number + 1))
  last_log="$run_root/check-$check_number.log"
  printf 'start: %s\n' "$label"
  {
    printf 'start: %s\ncommand:' "$label"
    printf ' %q' "$@"
    printf '\n'
  } >> "$receipt"
  local result=0
  timeout --foreground 360s "$@" > "$last_log" 2>&1 || result=$?
  cat "$last_log"
  printf 'finish: %s exit=%s\n' "$label" "$result" | tee -a "$receipt"
  sha256sum "$last_log" >> "$receipt"
  if ((result != 0)); then
    printf 'Retained failure: %s\n' "$last_log" >&2
    return "$result"
  fi
}

check_roots() {
  local log=$1
  local package root
  while read -r package root; do
    if ! rg -F '"Action":"pass"' "$log" |
      rg -F '"Package":"hatmax.adrianpk.com/'"$package"'"' |
      rg -F '"Test":"'"$root"'"' >/dev/null; then
      printf 'Required root missing, skipped or failed: %s %s\n' "$package" "$root" >&2
      return 1
    fi
    printf 'root: %s %s pass\n' "$package" "$root" >> "$receipt"
  done <<'ROOTS'
storage TestKeys
storage TestBounds
storage/local TestObjects
storage/local TestConfinement
storage/local TestWriteFailure
ROOTS
  if rg -q '"Action":"skip"' "$log"; then
    echo 'Required storage and image/local checks must not skip.' >&2
    return 1
  fi
}

mapfile -t go_files < <(rg --files storage -g '*.go' | sort)
run_check 'storage formatting' gofmt -l "${go_files[@]}"
if [[ -s "$last_log" ]]; then
  echo 'Storage formatting differs from gofmt.' >&2
  exit 1
fi
run_check 'source licensing' make source-license-check
run_check 'runner shell syntax' bash -n scripts/check-object-storage.sh
run_check 'ordinary storage and image/local roots' go test -json -count=1 -p=1 -parallel=2 -timeout=360s ./storage/... ./image/local
check_roots "$last_log"
run_check 'race storage and image/local roots' go test -json -race -count=1 -p=1 -parallel=2 -timeout=360s ./storage/... ./image/local
check_roots "$last_log"
run_check 'affected build' go build ./storage/... ./image/local
run_check 'affected vet' go vet ./storage/... ./image/local
run_check 'configured affected lint' golangci-lint run --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 ./storage/... ./image/local

if [[ $(git rev-parse HEAD) != "$candidate" ]]; then
  echo 'Candidate changed during slice validation.' >&2
  exit 1
fi
printf 'slice: 1 pass\n' >> "$receipt"
printf 'Slice 1 passed; retained receipt: %s\n' "$receipt"
