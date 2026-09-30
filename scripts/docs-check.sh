#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adrian PK
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of Hatmax. See COPYING for license terms.


set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"

required_top_level=(
  docs/README.md
  docs/tutorials
  docs/how-to
  docs/reference
  docs/explanation
)

for required_entry in "${required_top_level[@]}"; do
  if [[ ! -e "$required_entry" ]]; then
    echo "missing documentation entry: $required_entry" >&2
    exit 1
  fi
done

unexpected_top_level=$(
  find docs -mindepth 1 -maxdepth 1 \
    ! -path docs/README.md \
    ! -path docs/tutorials \
    ! -path docs/how-to \
    ! -path docs/reference \
    ! -path docs/explanation \
    -print
)
if [[ -n "$unexpected_top_level" ]]; then
  echo "unexpected top-level entry under docs:" >&2
  echo "$unexpected_top_level" >&2
  exit 1
fi

while IFS= read -r documentation_dir; do
  if [[ ! -f "$documentation_dir/README.md" ]]; then
    echo "missing documentation entrypoint: $documentation_dir/README.md" >&2
    exit 1
  fi
done < <(find docs -type d | sort)

mapfile -d '' markdown_files < <(
  find . -path './.git' -prune -o -path './.tmp' -prune -o -path './tmp' -prune \
    -o -type f -name '*.md' -print0
)

broken=0
declare -A linked_markdown=()
while IFS=$'\t' read -r source target; do
  case "$target" in
    http://*|https://*|mailto:*|'#'*)
      continue
      ;;
  esac

  local_target=${target%%#*}
  if [[ -z "$local_target" ]]; then
    continue
  fi

  resolved=$(realpath -m "$(dirname "$source")/$local_target")
  if [[ ! -e "$resolved" ]]; then
    echo "broken local link: $source -> $target" >&2
    broken=1
  elif [[ "$resolved" == *.md ]]; then
    linked_markdown["$resolved"]=1
  fi
done < <(
  perl -ne '
    while (/\]\(([^)]+)\)/g) { print "$ARGV\t$1\n" }
    while (/<(?:img|source)\b[^>]*\bsrc="([^"]+)"/g) { print "$ARGV\t$1\n" }
  ' "${markdown_files[@]}"
)

if ((broken)); then
  exit 1
fi

while IFS= read -r doc_file; do
  absolute_doc=$(realpath "$doc_file")
  if [[ -z "${linked_markdown[$absolute_doc]:-}" ]]; then
    echo "unreachable documentation page: $doc_file" >&2
    broken=1
  fi
done < <(find docs/tutorials docs/how-to docs/reference docs/explanation -type f -name '*.md' | sort)

if ((broken)); then
  exit 1
fi

for markdown_file in "${markdown_files[@]}"; do
  fence_count=$(rg -c '^```' "$markdown_file" || true)
  if ((fence_count % 2 != 0)); then
    echo "unbalanced fenced code block: $markdown_file" >&2
    broken=1
  fi
done

if ((broken)); then
  exit 1
fi

if rg -n '[[:blank:]]+$' "${markdown_files[@]}"; then
  echo "trailing whitespace found in Markdown" >&2
  exit 1
fi

if rg -n 'hatmax\.adrianpk\.com/hatmax/' \
  --glob '*.md' --glob '!tmp/**' --glob '!.tmp/**' .; then
  echo "stale Hatmax import path found" >&2
  exit 1
fi

go test -run '^$' ./examples/...
git diff --check

echo "documentation checks passed"
