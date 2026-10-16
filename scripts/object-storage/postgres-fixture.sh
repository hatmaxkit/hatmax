#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Adrian PK
# SPDX-License-Identifier: Apache-2.0
#
# This file is part of Hatmax. See LICENSE for license terms.

set -euo pipefail

if (( $# < 2 )); then
  echo 'Usage: postgres-fixture.sh OWNED-RUN COMMAND [ARGUMENTS...]' >&2
  exit 2
fi
run_directory=$(realpath -- "$1")
shift
repository=$(git rev-parse --show-toplevel)
case "$run_directory" in
  "$repository"/.tmp/object-storage/*) ;;
  *) echo 'PostgreSQL requires owned object storage run output.' >&2; exit 2 ;;
esac
git check-ignore -q "$run_directory/postgres-fixture.txt"
for tool in initdb pg_ctl postgres psql timeout sha256sum; do
  command -v "$tool" >/dev/null
done

receipt="$run_directory/postgres-fixture.txt"
if [[ -e "$receipt" ]]; then
  echo 'PostgreSQL evidence already exists; use a fresh owned run.' >&2
  exit 2
fi
umask 077
scratch=$(mktemp -d /tmp/hatmax-storage-postgres.XXXXXXXX)
cluster="$scratch/data"
socket_directory="$scratch/socket"
mkdir "$socket_directory"
owned=0

cleanup() {
  local status=$?
  trap - EXIT INT TERM
  if (( owned )); then
    if ! timeout --foreground 35s pg_ctl -D "$cluster" -m fast -t 30 -w stop > "$run_directory/postgres-stop.log" 2>&1; then
      echo 'Owned PostgreSQL shutdown failed; cluster retained.' >&2
      status=1
    fi
    if pg_ctl -D "$cluster" status > /dev/null 2>&1; then
      echo 'Owned PostgreSQL is still running; cluster retained.' >&2
      exit 1
    fi
  fi
  printf 'database: stopped\n' >> "$receipt"
  sha256sum "$run_directory"/postgres-*.log >> "$receipt"
  rm -rf -- "$scratch"
  if [[ -e "$scratch" ]]; then
    echo 'Owned PostgreSQL root removal was not confirmed.' >&2
    status=1
  else
    printf 'temporary-root: removed\n' >> "$receipt"
  fi
  exit "$status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

{
  printf 'database: owned native Unix socket; TCP disabled\n'
  for tool in initdb pg_ctl postgres psql; do
    "$tool" --version
    sha256sum "$(command -v "$tool")"
  done
} > "$receipt"
timeout --foreground 60s initdb -D "$cluster" -U postgres -A trust --no-locale > "$run_directory/postgres-init.log" 2>&1
{
  printf "listen_addresses = ''\nunix_socket_directories = '%s'\n" "$socket_directory"
  printf 'port = 5432\n'
} >> "$cluster/postgresql.conf"
owned=1
timeout --foreground 35s pg_ctl -D "$cluster" -l "$run_directory/postgres-server.log" -t 30 -w start > "$run_directory/postgres-start.log" 2>&1
timeout --foreground 10s psql -h "$socket_directory" -p 5432 -U postgres -d postgres -v ON_ERROR_STOP=1 -Atc 'SELECT 1' > "$run_directory/postgres-ready.log" 2>&1
if [[ $(cat "$run_directory/postgres-ready.log") != 1 ]]; then
  echo 'Owned PostgreSQL readiness failed.' >&2
  exit 1
fi
printf 'database: ready\n' >> "$receipt"

# The socket lives in a private mode-0700 directory; no system database or Docker
# fallback is used. Existing tests supply their own private schemas and cleanup.
export DB_HOST="$socket_directory" DB_PORT=5432 DB_USER=postgres DB_PASSWORD=fixture-only DB_NAME=postgres
timeout --foreground 1800s "$@"
