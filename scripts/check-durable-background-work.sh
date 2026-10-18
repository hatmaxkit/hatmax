#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adrian PK
# SPDX-License-Identifier: Apache-2.0
#
# This file is part of Hatmax. See LICENSE for license terms.
set -euo pipefail
mode=full
if (( $# != 0 )); then
  if [[ $# != 2 || $1 != slice || ! $2 =~ ^[123]$ ]]; then
    echo 'Usage: check-durable-background-work.sh [slice 1..3]' >&2
    exit 2
  fi
  mode="slice-$2"
fi
repository=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repository"
export GOWORK=off
mkdir -p .tmp/durable-background-work
packages=(./scheduler ./scheduler/postgres ./pubsub ./pubsub/postgres)
if [[ $mode == slice-1 ]]; then packages=(./scheduler ./scheduler/postgres); fi
if [[ $mode == slice-2 ]]; then packages=(./pubsub ./pubsub/postgres); fi
run_directory=$(mktemp -d "$repository/.tmp/durable-background-work/$mode.XXXXXXXX")
git check-ignore -q "$run_directory/postgres-server.log"
export DURABLE_WORK_MODE="$mode" DURABLE_WORK_TEST_LOG="$run_directory/tests.log"
# Keep one fixture and native output. Looper owns exact-HEAD validation receipts.
scripts/durable-background-work/postgres-fixture.sh "$run_directory" bash -euo pipefail -c '
  time -p go vet "$@"
  test_flags=(-race -count=1 -p=2 -timeout=180s)
  if [[ $DURABLE_WORK_MODE == slice-3 ]]; then
    test_flags+=(-run "^(TestWorkOperations|TestLeasedRuns|TestBoundedDeliveries)$")
  fi
  if [[ $DURABLE_WORK_MODE == slice-3 || $DURABLE_WORK_MODE == full ]]; then
    time -p go test -v "${test_flags[@]}" "$@" | tee "$DURABLE_WORK_TEST_LOG"
    awk -f scripts/durable-background-work/required-roots.awk "$DURABLE_WORK_TEST_LOG"
  else
    time -p go test "${test_flags[@]}" "$@"
  fi
  time -p env GOCACHE="$PWD/.tmp/lint/gocache" golangci-lint run --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 "$@"
  time -p make docs-check
' _ "${packages[@]}"
git diff --check
