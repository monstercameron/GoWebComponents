#!/usr/bin/env bash
set -euo pipefail

DB_PATH="${1:-../data/atlas-commerce-os.db}"

echo "Atlas Commerce OS seed reset scaffold"

if [[ -f "$DB_PATH" ]]; then
  rm -f "$DB_PATH"
  echo "Removed existing database: $DB_PATH"
else
  echo "No database file found at $DB_PATH"
fi

echo "Next step: rebuild the local seed database once persistence is implemented."