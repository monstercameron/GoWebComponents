#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEST_APP_DIR="$SCRIPT_DIR/testapp"
OUTPUT_DIR="$REPO_ROOT/bin/test/testapp"
OUTPUT_WASM="$OUTPUT_DIR/main.wasm"

echo "Building test WASM app..."
mkdir -p "$OUTPUT_DIR"

pushd "$TEST_APP_DIR" >/dev/null
GOOS=js GOARCH=wasm go build -o "$OUTPUT_WASM" .
popd >/dev/null

echo "Build successful: $OUTPUT_WASM"
echo
echo "Starting test server on http://localhost:8081"
echo "Press Ctrl+C to stop"

pushd "$TEST_APP_DIR" >/dev/null
python3 -m http.server 8081