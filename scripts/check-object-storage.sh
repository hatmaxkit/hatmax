#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adrian PK
# SPDX-License-Identifier: Apache-2.0
#
# This file is part of Hatmax. See LICENSE for license terms.

set -euo pipefail

# Slice 5 validates the delivered concern; only full mode runs make check on dev.
full_mode=0
if (( $# == 0 )); then
  full_mode=1
  slice_number=5
elif [[ $# == 2 && $1 == slice && $2 =~ ^[1-5]$ ]]; then
  slice_number=$2
else
  echo 'Usage: check-object-storage.sh [slice 1..5]' >&2
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
run_label="slice-$slice_number"
if ((full_mode)); then run_label=full; fi
run_root=$(mktemp -d "$repo_root/.tmp/object-storage/$run_label.XXXXXXXX")
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
  timeout --foreground "${check_timeout:-360s}" "$@" > "$last_log" 2>&1 || result=$?
  cat "$last_log"
  if [[ -n ${evidence_tool:-} ]]; then
    "$evidence_tool" record "$run_root" "${evidence_group:-check}" "$label" "$result" "$last_log" "$@"
  fi
  printf 'finish: %s exit=%s\n' "$label" "$result" | tee -a "$receipt"
  sha256sum "$last_log" >> "$receipt"
  if ((result != 0)); then
    printf 'Retained failure: %s\n' "$last_log" >&2
    return "$result"
  fi
}

run_group() {
  evidence_group=$1
  shift
  run_check "$@"
  evidence_group=check
}

if ((slice_number >= 5)); then
  printf 'start: build exact-candidate evidence tool\n'
  timeout --foreground 60s go build -o "$run_root/evidence" ./scripts/object-storage > "$run_root/tool-build.log" 2>&1
  evidence_tool="$run_root/evidence"
  evidence_mode=slice-5
  if ((full_mode)); then evidence_mode=full; fi
  "$evidence_tool" begin "$run_root" "$evidence_mode"
  "$evidence_tool" record "$run_root" check 'evidence tool build' 0 "$run_root/tool-build.log" go build -o "$run_root/evidence" ./scripts/object-storage
  printf 'finish: build exact-candidate evidence tool\n'
fi

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

format_paths=(storage)
if ((slice_number >= 4)); then
  format_paths+=(examples/object-storage)
fi
if ((slice_number >= 5)); then
  format_paths+=(scripts/object-storage)
fi
mapfile -t go_files < <(rg --files "${format_paths[@]}" -g '*.go' | sort)
if ((slice_number >= 4)); then
  go_files+=(scripts/documentation-conformance/object_guide_test.go scripts/documentation-conformance/object_guide_integration_test.go)
fi
if ((slice_number >= 5)); then
  go_files+=(scripts/documentation-conformance/object_acceptance_test.go)
fi
run_check 'storage formatting' gofmt -l "${go_files[@]}"
if [[ -s "$last_log" ]]; then
  echo 'Storage formatting differs from gofmt.' >&2
  exit 1
fi
run_check 'source licensing' make source-license-check
run_check 'runner shell syntax' bash -n scripts/check-object-storage.sh
run_check 'fixture tool shell syntax' bash -n scripts/prepare-object-storage-tools.sh
if ((slice_number >= 5)); then
  run_check 'PostgreSQL fixture shell syntax' bash -n scripts/object-storage/postgres-fixture.sh
fi
packages=(./storage/... ./image/local)
if ((slice_number >= 2)); then
  packages+=(./image/s3)
fi
if ((slice_number >= 4)); then
  packages=(./storage/... ./image/... ./scripts/documentation-conformance)
fi
if ((slice_number >= 5)); then
  packages+=(./scripts/object-storage)
fi
local_roots=(storage TestKeys storage TestBounds storage/local TestObjects storage/local TestConfinement storage/local TestWriteFailure)
if ((slice_number >= 3)); then
  local_roots+=(storage TestUploadBounds)
fi
if ((slice_number >= 4)); then
  local_roots+=(scripts/documentation-conformance TestObjectGuideDrift scripts/documentation-conformance TestObjectGuideControls scripts/documentation-conformance TestObjectLocalGuide scripts/documentation-conformance TestObjectImageCompatibility)
  local_roots+=(image/local TestNewStore image/local TestStore_URL image/local TestStore_PutGetDelete image/s3 TestUploadInput image/stdprocessor TestProcessorResize image/stdprocessor TestResizeByteLimit image/stdprocessor TestProcessorPixels)
fi
if ((slice_number >= 5)); then
  local_roots+=(scripts/documentation-conformance TestObjectAcceptance scripts/object-storage TestEvidenceControls)
fi
run_group ordinary 'ordinary storage and image adapter roots' go test -json -count=1 -p=1 -parallel=2 -timeout=360s "${packages[@]}"
check_roots "$last_log" "${local_roots[@]}"
run_group race 'race storage and image adapter roots' go test -json -race -count=1 -p=1 -parallel=2 -timeout=360s "${packages[@]}"
check_roots "$last_log" "${local_roots[@]}"
# Executable examples compile and lint, but have no unit-test roots. Go emits a
# package-level skip for them; keep the required test-root no-skip guard intact.
build_packages=("${packages[@]}")
if ((slice_number >= 4)); then
  build_packages+=(./examples/object-storage/...)
fi
run_check 'affected build' go build "${build_packages[@]}"
run_check 'affected vet' go vet "${build_packages[@]}"
run_check 'configured affected lint' golangci-lint run --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 "${build_packages[@]}"

if ((slice_number >= 2)); then
  # Preparation is a visible, separately bounded phase, never hidden in a root.
  printf 'start: pinned fixture tool preparation (maximum 900 seconds)\n'
  scripts/prepare-object-storage-tools.sh | tee "$run_root/tool-preparation.log"
  sha256sum "$run_root/tool-preparation.log" .tmp/object-storage/tools/minio.module.json .tmp/object-storage/tools/minio.buildinfo .tmp/object-storage/tools/minio >> "$receipt"
  printf 'finish: pinned fixture tool preparation\n'
  export HATMAX_STORAGE_MINIO="$repo_root/.tmp/object-storage/tools/minio"
  export HATMAX_STORAGE_FIXTURE_RECEIPT="$run_root/remote-fixture.txt"
  remote_selector='^Test(PrivateObjects|RemoteBounds|RemoteFailures)$'
  remote_roots=(storage/s3 TestPrivateObjects storage/s3 TestRemoteBounds storage/s3 TestRemoteFailures)
  if ((slice_number >= 3)); then
    remote_selector='^Test(PrivateObjects|RemoteBounds|RemoteFailures|SignedUploads|SigningExpiry|SnapshotCompletion)$'
    remote_roots+=(storage/s3 TestSignedUploads storage/s3 TestSigningExpiry storage/s3 TestSnapshotCompletion)
  fi
  if ((slice_number >= 5)); then
    remote_selector='^Test'
    remote_roots+=(storage/s3 TestObjectLifecycle)
  fi
  run_group remote 'private S3 integration roots' go test -json -race -tags=integration -count=1 -p=1 -parallel=2 -timeout=360s -run "$remote_selector" ./storage/s3
  check_roots "$last_log" "${remote_roots[@]}"
  if ((slice_number >= 3)); then
    if ! rg -Fxq 'scope-probe-bucket: removed' "$HATMAX_STORAGE_FIXTURE_RECEIPT"; then
      echo 'Signed upload target-scope probe cleanup was not confirmed.' >&2
      exit 1
    fi
  fi
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

if ((slice_number >= 4)); then
  run_group docs 'documentation structure, links and compiled examples' make docs-check
  export HATMAX_STORAGE_FIXTURE_RECEIPT="$run_root/guide-fixture.txt"
  check_timeout=600s
  run_group guide 'published actual signed remote workflow' go test -json -race -tags=integration,documentation -count=1 -p=1 -parallel=2 -timeout=600s -run '^TestObjectStorageGuide$' ./scripts/documentation-conformance
  unset check_timeout
  check_roots "$last_log" scripts/documentation-conformance TestObjectStorageGuide
  for outcome in 'guide: signed POST and exact final readback passed' 'guide: spool empty' 'objects-after-cleanup: 0' 'bucket: removed' 'server: stopped' 'temporary-root: removed'; do
    if ! rg -Fxq "$outcome" "$HATMAX_STORAGE_FIXTURE_RECEIPT"; then
      printf 'Missing required guide evidence: %s\n' "$outcome" >&2
      exit 1
    fi
  done
  sha256sum "$HATMAX_STORAGE_FIXTURE_RECEIPT" >> "$receipt"
  if ((slice_number >= 5)); then
    "$evidence_tool" record "$run_root" guide-child 'documented storage child results' 0 "$HATMAX_STORAGE_FIXTURE_RECEIPT.results.jsonl" go test -json -race -tags=integration,documentation -count=1 -p=1 -parallel=2 -timeout=360s -run '^TestDocumentedStorage$' ./storage/s3
  fi
  run_check 'documentation integration vet' go vet -tags=integration,documentation ./storage/s3 ./scripts/documentation-conformance
  run_check 'configured documentation integration lint' golangci-lint run --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 --build-tags=integration,documentation ./storage/s3 ./scripts/documentation-conformance
fi

if ((slice_number >= 5)); then
  check_timeout=1860s
  if ((full_mode)); then
    # TEST_FLAGS changes output, freshness and finite scheduling only. All
    # existing check prerequisites, coverage threshold and package scope remain.
    run_group repository 'complete repository gate' scripts/object-storage/postgres-fixture.sh "$run_root" make check 'TEST_FLAGS=-json -count=1 -p=1 -parallel=2 -timeout=360s' "LINT_GOCACHE=$run_root/lint-gocache"
  else
    run_group postgres-run 'native PostgreSQL smoke' scripts/object-storage/postgres-fixture.sh "$run_root" go test -json -count=1 -p=1 -timeout=120s -run '^TestStartAndStopIntegration$' ./db
  fi
  unset check_timeout
  run_group postgres 'owned PostgreSQL cleanup' "$evidence_tool" postgres "$run_root"
fi

if [[ $(git rev-parse HEAD) != "$candidate" ]]; then
  echo 'Candidate changed during slice validation.' >&2
  exit 1
fi
if ((slice_number >= 5)); then
  printf 'checks: complete; exact evidence verification follows\n' >> "$receipt"
  "$evidence_tool" verify "$run_root"
else
  printf 'slice: %s pass\n' "$slice_number" >> "$receipt"
fi
printf 'Slice %s passed; retained receipt: %s\n' "$slice_number" "$receipt"
