#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adrian PK
# SPDX-License-Identifier: Apache-2.0
#
# This file is part of Hatmax. See LICENSE for license terms.

# Source only after resolving product tool overrides into the owned bin directory.
# No inherited PATH directory is retained. These utilities serve the repository
# Make/shell workflows, PostgreSQL and the C compiler used by Go's race build.
native_utilities=(
  bash sh git awk sed rg find sort realpath dirname basename readlink head tail
  grep cut xargs tr cmp cp mv rm rmdir mkdir ln cat chmod touch date sleep env
  printf sha256sum mktemp id uname wc tee timeout kill gcc cc g++ c++ as ld ar
  objcopy objdump ranlib nm strip ldd getent install locale gofmt codex ps lsof nohup
)
for tool in "${native_utilities[@]}"; do
  if [[ $tool == gofmt ]]; then
    tool_path="$(dirname -- "$(realpath -- "$fixture/bin/go")")/gofmt"
  else
    tool_path=$(realpath -- "$(type -P -- "$tool")")
  fi
  ln -s "$tool_path" "$fixture/bin/$tool"
done
# The shell and Go implementations inherit this exact constrained tool lookup.
export PATH="$fixture/bin"
export GOTOOLCHAIN=local
export HATMAX_DOC_NATIVE_BIN="$fixture/bin"
manifest="$fixture/native-tools.tsv"
for tool_path in "$fixture/bin/"*; do
  resolved_tool=$(realpath -- "$tool_path")
  printf '%s\t%s\t%s\n' "$(basename -- "$tool_path")" "$resolved_tool" "$(sha256sum "$resolved_tool" | cut -d ' ' -f 1)"
done > "$manifest"
