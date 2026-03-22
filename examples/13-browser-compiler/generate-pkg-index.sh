#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PKG_DIR="$SCRIPT_DIR/static/pkg/js_wasm"
OUTPUT_FILE="$SCRIPT_DIR/static/pkg/index.json"

python3 - "$PKG_DIR" "$OUTPUT_FILE" <<'PY'
import json
import pathlib
import sys

pkg_dir = pathlib.Path(sys.argv[1])
output_file = pathlib.Path(sys.argv[2])
files = sorted(str(path.relative_to(pkg_dir).with_suffix('')).replace('\\', '/') for path in pkg_dir.rglob('*.a'))
output_file.write_text(json.dumps(files, indent=2) + '\n', encoding='utf-8')
print(f"Generated index.json with {len(files)} packages")
PY