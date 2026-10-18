#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adrian PK
# SPDX-License-Identifier: Apache-2.0
#
# This file is part of Hatmax. See LICENSE for license terms.
set -euo pipefail
# Later slice modes and the integrated gate are enabled by their owning slices.
if [[ $# != 2 || $1 != slice || ! $2 =~ ^[12]$ ]]; then
  echo 'Usage: check-durable-background-work.sh slice 1..2' >&2
  exit 2
fi
repository=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repository"
export GOWORK=off
mkdir -p .tmp/durable-background-work
packages=(./scheduler ./scheduler/postgres)
if [[ $2 == 2 ]]; then packages=(./pubsub ./pubsub/postgres); fi
run_directory=$(mktemp -d "$repository/.tmp/durable-background-work/slice-$2.XXXXXXXX")
git check-ignore -q "$run_directory/postgres-server.log"
# Retain native output and timings; Looper owns exact-HEAD validation receipts.
scripts/durable-background-work/postgres-fixture.sh "$run_directory" bash -euo pipefail -c '
  time -p go vet "$@"
  time -p go test -race -count=1 -p=2 -timeout=180s "$@"
  time -p env GOCACHE="$PWD/.tmp/lint/gocache" golangci-lint run --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 "$@"
  time -p make docs-check
' _ "${packages[@]}"
git diff --check
