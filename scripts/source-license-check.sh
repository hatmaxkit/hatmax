#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adrian PK
# SPDX-License-Identifier: Apache-2.0
#
# This file is part of Hatmax. See LICENSE for license terms.

set -euo pipefail

repository_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repository_root"

for required_file in LICENSE REUSE.toml; do
  if [[ ! -f "$required_file" ]]; then
    echo "Missing licensing file: $required_file" >&2
    exit 1
  fi
done

if ! grep -Fxq 'SPDX-License-Identifier = "Apache-2.0"' REUSE.toml ||
  ! grep -Fxq 'SPDX-FileCopyrightText = "2026 Adrian PK"' REUSE.toml; then
  echo "REUSE.toml must declare the canonical copyright and Apache-2.0." >&2
  exit 1
fi

declare -A metadata_paths=()
while IFS= read -r path; do
  metadata_paths["$path"]=1
done < <(sed -n 's/^  "\(.*\)",$/\1/p' REUSE.toml)

declare -A repository_paths=()
failures=0
headers=0
metadata=0
while IFS= read -r -d '' path; do
  repository_paths["$path"]=1

  case "$path" in
    LICENSE | REUSE.toml | LICENSES/* | third_party/*)
      continue
      ;;
  esac

  if [[ -n "${metadata_paths[$path]:-}" ]]; then
    ((metadata += 1))
    continue
  fi

  case "$path" in
    AGENTS.md | .agents/* | */testdata/* | */fixtures/* | *.json | *.png | *.sum | *.txt | */dal/*.go)
      echo "Missing content-preserving licensing metadata: $path" >&2
      failures=1
      continue
      ;;
  esac

  case "$path" in
    Makefile | */Makefile | .mailmap | .gitignore | */.gitignore | .githooks/* | *.go | *.mod | *.work | *.js | *.css | *.html | *.md | *.sh | *.sql | *.yaml | *.yml | *.toml)
      ;;
    *)
      echo "File requires a licensing policy: $path" >&2
      failures=1
      continue
      ;;
  esac

  header=$(head -n 80 "$path")
  if [[ "$header" != *"SPDX-License-Identifier: Apache-2.0"* ]] ||
    [[ "$header" != *"SPDX-FileCopyrightText: 2026 Adrian PK"* &&
       "$header" != *"SPDX-FileCopyrightText: 2024-2026 Adrian PK"* ]] ||
    [[ "$header" != *"This file is part of Hatmax. See LICENSE for license terms."* ]]; then
    echo "Missing canonical SPDX header: $path" >&2
    failures=1
  else
    ((headers += 1))
  fi
done < <(git ls-files --cached --others --exclude-standard -z | sort -zu)

for path in "${!metadata_paths[@]}"; do
  if [[ -z "${repository_paths[$path]:-}" ]]; then
    echo "Stale licensing metadata: $path" >&2
    failures=1
  fi
done

if ((failures)); then
  exit 1
fi

printf 'Source licensing passed: %s headers, %s content-preserving annotations.\n' "$headers" "$metadata"
