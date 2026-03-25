#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Starting Go example server on http://127.0.0.1:8090"
echo "Examples index: http://127.0.0.1:8090/examples/"
echo "Counter page:   http://127.0.0.1:8090/examples/01-counter/"
echo "Press Ctrl+C to stop"

cd "$REPO_ROOT"
go run ./tools/gwc examples -host 127.0.0.1 -port 8090
