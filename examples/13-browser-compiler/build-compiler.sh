#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="$SCRIPT_DIR/static/bin"
mkdir -p "$OUTPUT_DIR"

echo "Building Go Compiler (cmd/compile)..."
GOOS=js GOARCH=wasm go build -o "$OUTPUT_DIR/compile.wasm" cmd/compile

echo "Building Go Linker (cmd/link)..."
GOOS=js GOARCH=wasm go build -o "$OUTPUT_DIR/link.wasm" cmd/link

echo "Copying wasm_exec.js..."
GOROOT_PATH="$(go env GOROOT | tr -d '\r')"
WASM_EXEC_PATH="$GOROOT_PATH/lib/wasm/wasm_exec.js"
if [[ ! -f "$WASM_EXEC_PATH" ]]; then
  WASM_EXEC_PATH="$GOROOT_PATH/misc/wasm/wasm_exec.js"
fi
cp "$WASM_EXEC_PATH" "$SCRIPT_DIR/script/wasm_exec.js"

echo "Copying standard library archives..."
"$SCRIPT_DIR/copy-std-lib.sh"

echo "Generating standard library package index..."
"$SCRIPT_DIR/generate-pkg-index.sh"

echo "Done."