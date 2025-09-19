#!/usr/bin/env bash
set -euo pipefail
: "${PGHOST:=localhost}"
: "${PGPORT:=5432}"
: "${PGUSER:=nakama}"
: "${PGDATABASE:=nakama}"
: "${PGPASSWORD:=localdev}"

for f in $(ls -1 server/migrations/*.sql | sort); do
  echo "Applying $f"
  PGPASSWORD="$PGPASSWORD" psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1 -f "$f"
  echo "Done $f"
done
