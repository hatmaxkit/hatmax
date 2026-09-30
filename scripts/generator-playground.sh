#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adrian PK
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of Hatmax. See COPYING for license terms.


set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repository_root=$(cd -- "$script_dir/.." && pwd)
playground_base=${HATMAX_PLAYGROUND_BASE:-"$HOME/Projects/playground/hatmax"}
playground_name="$(date +%Y%m%d-%H%M%S)-generator-tui"
playground="$playground_base/$playground_name"
tool_cache="$repository_root/.tmp/tools/bin"
playground_bin="$playground/.tmp/bin"

for command_name in go codex golangci-lint; do
	if ! command -v "$command_name" >/dev/null; then
		echo "$command_name is required" >&2
		exit 1
	fi
done

mkdir -p "$playground_bin" "$tool_cache"

(cd "$repository_root" && go build -o "$playground_bin/hm" ./cmd/hm)

if command -v sqlc >/dev/null; then
	install -m 0755 "$(command -v sqlc)" "$playground_bin/sqlc"
else
	if [[ ! -x "$tool_cache/sqlc" ]]; then
		GOBIN="$tool_cache" go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1
	fi

	install -m 0755 "$tool_cache/sqlc" "$playground_bin/sqlc"
fi

echo "Fresh Hatmax generator playground:"
echo "$playground"

if [[ ${HATMAX_PLAYGROUND_NO_RUN:-0} == 1 ]]; then
	echo
	echo "Prepared without launching. Run:"
	echo "cd $playground && PATH=\"\$PWD/.tmp/bin:\$PATH\" ./.tmp/bin/hm"

	exit 0
fi

echo
echo "Launching the current local hm build."

cd "$playground"
PATH="$playground_bin:$PATH" exec "$playground_bin/hm"
