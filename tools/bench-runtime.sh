#!/usr/bin/env bash
set -euo pipefail

PACKAGE="./internal/runtime"
COUNT="5"
BENCH="."
OUTPUT=""
EXEC=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --package) PACKAGE="$2"; shift 2 ;;
    --count) COUNT="$2"; shift 2 ;;
    --bench) BENCH="$2"; shift 2 ;;
    --output) OUTPUT="$2"; shift 2 ;;
    --exec) EXEC="$2"; shift 2 ;;
    *) echo "unknown arg: $1" >&2; exit 1 ;;
  esac
done

if [[ -z "$OUTPUT" ]]; then
  stamp="$(date +%Y%m%d-%H%M%S)"
  safe_package="${PACKAGE//[^A-Za-z0-9._-]/_}"
  OUTPUT="$(dirname "$0")/bench-${safe_package}-${stamp}.txt"
fi

cmd=(go test)
if [[ -n "$EXEC" ]]; then
  cmd+=(-exec "$EXEC")
fi
cmd+=("$PACKAGE" -run '^$' -bench "$BENCH" -benchmem -count "$COUNT")

printf 'Running:'
printf ' %q' "${cmd[@]}"
printf '\n'
"${cmd[@]}" | tee "$OUTPUT"
printf 'Wrote benchmark output to %s\n' "$OUTPUT"
