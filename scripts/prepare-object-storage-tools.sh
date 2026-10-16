#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adrian PK
# SPDX-License-Identifier: Apache-2.0
#
# This file is part of Hatmax. See LICENSE for license terms.

set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"
export GOWORK=off
go_tool="$(go env GOROOT)/bin/go"
if [[ $("$go_tool" version) != 'go version go1.27.1 '* ]]; then
  echo 'Fixture preparation requires Go 1.27.1.' >&2
  exit 1
fi

version=v0.0.0-20251015172955-9e49d5e7a648
checksum='h1:6TdolSCLSs2nwm8i0PpWDqf9iX2Ty9WQK8wmr7dCnUM='
tools="$repo_root/.tmp/object-storage/tools"
mkdir -p "$tools"
if ! git check-ignore -q "$tools/minio"; then
  echo 'Fixture tools must be ignored by Git.' >&2
  exit 1
fi

verify_binary() {
  [[ -x "$tools/minio" && -f "$tools/minio.sha256" && -f "$tools/minio.module.json" ]]
  (cd "$tools" && sha256sum -c minio.sha256)
  "$go_tool" version -m "$tools/minio" > "$tools/minio.buildinfo"
  rg -F 'go1.27.1' "$tools/minio.buildinfo" >/dev/null
  rg -F "mod" "$tools/minio.buildinfo" | rg -F "github.com/minio/minio" | rg -F "$version" | rg -F "$checksum" >/dev/null
  rg -F '"Sum": "'"$checksum"'"' "$tools/minio.module.json" >/dev/null
}

if [[ -e "$tools/minio" ]]; then
  verify_binary
  echo 'Pinned MinIO fixture executable verified.'
  exit 0
fi

echo 'start: pinned MinIO fixture build (maximum 900 seconds)'
timeout --foreground 900s env GOWORK=off GOTOOLCHAIN=local "$go_tool" mod download -json "github.com/minio/minio@$version" > "$tools/minio.module.json"
rg -F '"Sum": "'"$checksum"'"' "$tools/minio.module.json" >/dev/null
rg -F '"Hash": "9e49d5e7a648f00e26f2246f4dc28e6b07f8c84a"' "$tools/minio.module.json" >/dev/null

install_root=$(mktemp -d "$tools/install.XXXXXXXX")
trap 'rm -rf -- "$install_root"' EXIT
timeout --foreground 900s env GOWORK=off GOTOOLCHAIN=local GOFLAGS=-p=2 GOBIN="$install_root" "$go_tool" install "github.com/minio/minio@$version" > "$tools/minio.build.log" 2>&1
mv -- "$install_root/minio" "$tools/minio"
chmod 0700 "$tools/minio"
(cd "$tools" && sha256sum minio > minio.sha256)
verify_binary
echo 'finish: pinned MinIO fixture build; source, module checksum, Go build information and binary digest retained'
