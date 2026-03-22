#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_DIR="$SCRIPT_DIR/dev-server"

if [[ ! -d "$SERVER_DIR" ]]; then
  echo "Error: dev server directory not found at $SERVER_DIR" >&2
  exit 1
fi

if ! command -v node >/dev/null 2>&1 || ! command -v npm >/dev/null 2>&1; then
  echo "Error: Node.js + npm are required to run tools/dev-server." >&2
  exit 1
fi

echo "Installing dev-server dependencies..."
pushd "$SERVER_DIR" >/dev/null
npm install

echo "Starting Express dev server on http://127.0.0.1:8090"
echo "Examples index: http://127.0.0.1:8090/examples/static/index.html"
echo "Counter page:   http://127.0.0.1:8090/examples/01-counter/counter.html"
echo "Press Ctrl+C to stop"

npm start