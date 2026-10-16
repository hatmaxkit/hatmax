#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adrian PK
# SPDX-License-Identifier: Apache-2.0
#
# This file is part of Hatmax. See LICENSE for license terms.

set -euo pipefail

# Only implemented modes are admitted. Full acceptance is introduced by T5.2.
if [[ $# != 2 || $1 != slice || ! $2 =~ ^[12]$ ]]; then
  echo 'Usage: check-object-storage.sh slice 1..2 (full acceptance is not available yet)' >&2
  exit 2
fi
slice_number=$2

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
run_root=$(mktemp -d "$repo_root/.tmp/object-storage/slice-$slice_number.XXXXXXXX")
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
  printf 'slice: %s\nhead: %s\n' "$slice_number" "$candidate"
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
  shift
  local package root
  while (( $# )); do
    package=$1
    root=$2
    shift 2
    if ! rg -F '"Action":"pass"' "$log" |
      rg -F '"Package":"hatmax.adrianpk.com/'"$package"'"' |
      rg -F '"Test":"'"$root"'"' >/dev/null; then
      printf 'Required root missing, skipped or failed: %s %s\n' "$package" "$root" >&2
      return 1
    fi
    printf 'root: %s %s pass\n' "$package" "$root" >> "$receipt"
  done
  if rg -q '"Action":"skip"' "$log"; then
    echo 'Required storage and image adapter checks must not skip.' >&2
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
run_check 'fixture tool shell syntax' bash -n scripts/prepare-object-storage-tools.sh
packages=(./storage/... ./image/local)
if ((slice_number >= 2)); then
  packages+=(./image/s3)
fi
local_roots=(storage TestKeys storage TestBounds storage/local TestObjects storage/local TestConfinement storage/local TestWriteFailure)
run_check 'ordinary storage and image adapter roots' go test -json -count=1 -p=1 -parallel=2 -timeout=360s "${packages[@]}"
check_roots "$last_log" "${local_roots[@]}"
run_check 'race storage and image adapter roots' go test -json -race -count=1 -p=1 -parallel=2 -timeout=360s "${packages[@]}"
check_roots "$last_log" "${local_roots[@]}"
run_check 'affected build' go build "${packages[@]}"
run_check 'affected vet' go vet "${packages[@]}"
run_check 'configured affected lint' golangci-lint run --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 "${packages[@]}"

if ((slice_number >= 2)); then
  # Preparation is a visible, separately bounded phase, never hidden in a root.
  printf 'start: pinned fixture tool preparation (maximum 900 seconds)\n'
  scripts/prepare-object-storage-tools.sh | tee "$run_root/tool-preparation.log"
  sha256sum "$run_root/tool-preparation.log" .tmp/object-storage/tools/minio.module.json .tmp/object-storage/tools/minio.buildinfo .tmp/object-storage/tools/minio >> "$receipt"
  printf 'finish: pinned fixture tool preparation\n'
  export HATMAX_STORAGE_MINIO="$repo_root/.tmp/object-storage/tools/minio"
  export HATMAX_STORAGE_FIXTURE_RECEIPT="$run_root/remote-fixture.txt"
  run_check 'private S3 integration roots' go test -json -race -tags=integration -count=1 -p=1 -parallel=2 -timeout=360s -run '^Test(PrivateObjects|RemoteBounds|RemoteFailures)$' ./storage/s3
  check_roots "$last_log" storage/s3 TestPrivateObjects storage/s3 TestRemoteBounds storage/s3 TestRemoteFailures
  for outcome in 'objects-after-cleanup: 0' 'bucket: removed' 'server: stopped' 'temporary-root: removed'; do
    if ! rg -Fxq "$outcome" "$HATMAX_STORAGE_FIXTURE_RECEIPT"; then
      printf 'Missing required fixture cleanup evidence: %s\n' "$outcome" >&2
      exit 1
    fi
  done
  sha256sum "$HATMAX_STORAGE_FIXTURE_RECEIPT" >> "$receipt"
  run_check 'integration vet' go vet -tags=integration ./storage/s3
  run_check 'configured integration lint' golangci-lint run --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 --build-tags=integration ./storage/s3
fi

if [[ $(git rev-parse HEAD) != "$candidate" ]]; then
  echo 'Candidate changed during slice validation.' >&2
  exit 1
fi
printf 'slice: %s pass\n' "$slice_number" >> "$receipt"
printf 'Slice %s passed; retained receipt: %s\n' "$slice_number" "$receipt"
