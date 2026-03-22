#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEST_DIR="$SCRIPT_DIR/static/pkg/js_wasm"
mkdir -p "$DEST_DIR"

echo "Determining standard library paths..."
GOOS=js GOARCH=wasm go install std

while IFS= read -r pkg; do
  export_path="$(GOOS=js GOARCH=wasm go list -export -f '{{.Export}}' "$pkg" | tr -d '\r')"
  if [[ -z "$export_path" ]]; then
    continue
  fi
  dest_path="$DEST_DIR/$pkg.a"
  mkdir -p "$(dirname "$dest_path")"
  echo "Copying $pkg..."
  cp "$export_path" "$dest_path"
done < <(GOOS=js GOARCH=wasm go list std)

echo "Standard library copied to $DEST_DIR"