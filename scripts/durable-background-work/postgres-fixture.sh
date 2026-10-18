#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adrian PK
# SPDX-License-Identifier: Apache-2.0
#
# This file is part of Hatmax. See LICENSE for license terms.
set -euo pipefail
if (( $# < 2 )); then
  echo 'Usage: postgres-fixture.sh RUN-DIRECTORY COMMAND [ARGUMENTS...]' >&2
  exit 2
fi
run_directory=$(realpath -- "$1")
shift
repository=$(git rev-parse --show-toplevel)
case "$run_directory" in
  "$repository"/.tmp/durable-background-work/*) ;;
  *) echo 'PostgreSQL requires an owned durable work run directory.' >&2; exit 2 ;;
esac
umask 077
scratch=$(mktemp -d /tmp/hatmax-durable-postgres.XXXXXXXX)
cluster="$scratch/data"
mkdir "$scratch/socket"
owned=0
cleanup() {
  local status=$?
  trap - EXIT INT TERM
  if (( owned )); then
    if ! timeout --foreground 35s pg_ctl -D "$cluster" -m fast -t 30 -w stop > "$run_directory/postgres-stop.log" 2>&1; then
      echo 'Owned PostgreSQL shutdown failed; cluster retained.' >&2
      exit 1
    fi
    if pg_ctl -D "$cluster" status >/dev/null 2>&1; then
      echo 'Owned PostgreSQL remains active; cluster retained.' >&2
      exit 1
    fi
  fi
  rm -rf -- "$scratch"
  echo 'Owned PostgreSQL stopped; temporary root removed.'
  exit "$status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
timeout --foreground 60s initdb -D "$cluster" -U postgres -A trust --no-locale > "$run_directory/postgres-init.log" 2>&1
printf "listen_addresses = ''\nunix_socket_directories = '%s'\nport = 5432\n" "$scratch/socket" >> "$cluster/postgresql.conf"
owned=1
timeout --foreground 35s pg_ctl -D "$cluster" -l "$run_directory/postgres-server.log" -t 30 -w start > "$run_directory/postgres-start.log" 2>&1
timeout --foreground 10s psql -h "$scratch/socket" -p 5432 -U postgres -d postgres -v ON_ERROR_STOP=1 -Atc 'SELECT 1' > "$run_directory/postgres-ready.log"
test "$(cat "$run_directory/postgres-ready.log")" = 1
export DB_HOST="$scratch/socket" DB_PORT=5432 DB_USER=postgres DB_PASSWORD=fixture-only DB_NAME=postgres
"$@"
