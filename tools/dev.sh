#!/bin/bash

# Development server wrapper for path-based Go WASM apps.
# Builds the live-reload server and runs it against a specific main.go or app directory.

set -euo pipefail

MAIN="${1:-}"
ROOT="${2:-}"
INDEX="${3:-}"
OUTPUT="${4:-}"
HOST="${HOST:-127.0.0.1}"
PORT="${PORT:-8080}"
NO_HOT_RELOAD="${NO_HOT_RELOAD:-0}"

if [[ -z "$MAIN" ]]; then
  echo "Usage: ./tools/dev.sh <path-to-main.go-or-app-dir> [root] [index] [output]" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SERVER_DIR="$SCRIPT_DIR/livereload"
BINARY_PATH="$SERVER_DIR/livereload"

if ! command -v go >/dev/null 2>&1; then
  echo "Error: Go is required to run the dev server." >&2
  exit 1
fi

pushd "$SERVER_DIR" >/dev/null
go build -o livereload .
popd >/dev/null

if [[ -z "$ROOT" ]]; then
  ROOT="$(dirname "$MAIN")"
fi

args=(
  -main "$MAIN"
  -root "$ROOT"
  -host "$HOST"
  -port "$PORT"
)

if [[ -n "$INDEX" ]]; then
  args+=(-index "$INDEX")
fi
if [[ -n "$OUTPUT" ]]; then
  args+=(-output "$OUTPUT")
fi
if [[ "$NO_HOT_RELOAD" == "1" ]]; then
  args+=(-hot=false)
fi

echo "Starting hot-reload dev server..."
echo "Main:  $MAIN"
echo "Root:  $ROOT"
echo "Press Ctrl+C to stop"

pushd "$REPO_ROOT" >/dev/null
"$BINARY_PATH" "${args[@]}"
popd >/dev/null
