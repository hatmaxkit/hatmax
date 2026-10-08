#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adrian PK
# SPDX-License-Identifier: Apache-2.0
#
# This file is part of Hatmax. See LICENSE for license terms.

set -euo pipefail

if (( $# < 2 )); then
  echo 'Usage: data-fixture.sh OWNED-FIXTURE COMMAND [ARGUMENTS...]' >&2
  exit 2
fi
fixture=$(realpath -- "$1")
shift
repo_root=$(git rev-parse --show-toplevel)
case "$fixture" in
  "$repo_root"/.tmp/documentation-conformance/slice-3.*) ;;
  *) echo 'Data fixtures require owned Slice 3 storage.' >&2; exit 2 ;;
esac
git check-ignore -q "$fixture/postgres.log"
for tool in initdb pg_ctl psql postgres; do
  command -v "$tool" >/dev/null
done

# A short unique Unix socket avoids platform socket-path limits and shares no
# listener or mutable storage with another invocation. TCP is disabled.
socket_directory=$(mktemp -d /tmp/hatmax-doc-postgres.XXXXXXXX)
cluster="$fixture/postgres-data"
owned_cluster=0
cleanup() {
  local status=$?
  trap - EXIT
  if (( owned_cluster )); then
    if ! pg_ctl -D "$cluster" -m fast -t 20 -w stop > "$fixture/postgres-stop.log" 2>&1; then
      cat "$fixture/postgres-stop.log" >&2
      status=1
    fi
  fi
  rmdir -- "$socket_directory" || status=1
  exit "$status"
}
trap cleanup EXIT

initdb -D "$cluster" -U postgres -A trust --no-locale > "$fixture/postgres-init.log" 2>&1
{
  printf "listen_addresses = ''\nunix_socket_directories = '%s'\n" "$socket_directory"
  printf 'port = 5432\n'
} >> "$cluster/postgresql.conf"
owned_cluster=1
pg_ctl -D "$cluster" -l "$fixture/postgres.log" -t 20 -w start > "$fixture/postgres-start.log" 2>&1
psql -h "$socket_directory" -p 5432 -U postgres -d postgres -v ON_ERROR_STOP=1 -c 'CREATE ROLE dev LOGIN' > "$fixture/postgres-role.log" 2>&1
psql -h "$socket_directory" -p 5432 -U postgres -d postgres -v ON_ERROR_STOP=1 -c 'CREATE ROLE readonly LOGIN' >> "$fixture/postgres-role.log" 2>&1
psql -h "$socket_directory" -p 5432 -U postgres -d postgres -v ON_ERROR_STOP=1 -c 'CREATE DATABASE myapp OWNER dev' > "$fixture/postgres-database.log" 2>&1

export DB_HOST="$socket_directory" DB_PORT=5432 DB_USER=dev DB_PASSWORD=fixture-only DB_NAME=myapp
export HATMAX_DOC_PGDATA="$cluster"
"$@"
