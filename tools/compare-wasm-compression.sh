#!/usr/bin/env bash
set -euo pipefail

PACKAGE="./examples/21-ui-render"
OUT_DIR="bin/wasm-compression-comparison"
BINARY_NAME="app.wasm"
SUMMARY_NAME="wasm-compression-comparison.json"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --package) PACKAGE="$2"; shift 2 ;;
    --out-dir) OUT_DIR="$2"; shift 2 ;;
    --binary-name) BINARY_NAME="$2"; shift 2 ;;
    --summary-name) SUMMARY_NAME="$2"; shift 2 ;;
    *) echo "unknown arg: $1" >&2; exit 1 ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOLVED_OUT_DIR="$(python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "$OUT_DIR")"
mkdir -p "$RESOLVED_OUT_DIR"

"$SCRIPT_DIR/measure-wasm-build.sh" --package "$PACKAGE" --out-dir "$RESOLVED_OUT_DIR/plain-raw" --binary-name "$BINARY_NAME" --manifest-name wasm-build-experiment.json --skip-compression
"$SCRIPT_DIR/measure-wasm-build.sh" --package "$PACKAGE" --out-dir "$RESOLVED_OUT_DIR/stripped-raw" --binary-name "$BINARY_NAME" --manifest-name wasm-build-experiment.json --release-profile --skip-compression
"$SCRIPT_DIR/measure-wasm-build.sh" --package "$PACKAGE" --out-dir "$RESOLVED_OUT_DIR/stripped-compressed" --binary-name "$BINARY_NAME" --manifest-name wasm-build-experiment.json --release-profile

WASM_OPT_MODE="unavailable"
WASM_OPT_LABEL=""
if command -v wasm-opt >/dev/null 2>&1; then
  WASM_OPT_MODE="path"
  WASM_OPT_LABEL="$(command -v wasm-opt)"
fi

if [[ "$WASM_OPT_MODE" != "unavailable" ]]; then
  mkdir -p "$RESOLVED_OUT_DIR/optimized-raw" "$RESOLVED_OUT_DIR/optimized-compressed"
  SOURCE_WASM="$RESOLVED_OUT_DIR/stripped-raw/$BINARY_NAME"
  OPT_RAW_WASM="$RESOLVED_OUT_DIR/optimized-raw/$BINARY_NAME"
  OPT_COMP_WASM="$RESOLVED_OUT_DIR/optimized-compressed/$BINARY_NAME"
  wasm-opt "$SOURCE_WASM" -Oz -o "$OPT_RAW_WASM"
  cp "$OPT_RAW_WASM" "$OPT_COMP_WASM"
  gzip -n -9 -c "$OPT_COMP_WASM" > "$OPT_COMP_WASM.gz"
  if command -v brotli >/dev/null 2>&1; then
    brotli -f -q 11 -o "$OPT_COMP_WASM.br" "$OPT_COMP_WASM"
  elif command -v node >/dev/null 2>&1 && [[ -f "$SCRIPT_DIR/write-brotli-file.mjs" ]]; then
    node "$SCRIPT_DIR/write-brotli-file.mjs" "$OPT_COMP_WASM" "$OPT_COMP_WASM.br"
  fi
fi

python3 - "$RESOLVED_OUT_DIR" "$SUMMARY_NAME" "$PACKAGE" "$WASM_OPT_MODE" "$WASM_OPT_LABEL" <<'PY'
import hashlib
import json
import os
import pathlib
import sys

root = pathlib.Path(sys.argv[1])
summary_name = sys.argv[2]
package = sys.argv[3]
wasm_opt_mode = sys.argv[4]
wasm_opt_label = sys.argv[5]

def read_json(path):
  return json.loads(path.read_text(encoding='utf-8'))

summary = {
  'package': package,
  'variants': {
    'plain_raw': read_json(root / 'plain-raw' / 'wasm-build-experiment.json'),
    'stripped_raw': read_json(root / 'stripped-raw' / 'wasm-build-experiment.json'),
    'stripped_compressed': read_json(root / 'stripped-compressed' / 'wasm-build-experiment.json'),
  },
  'wasm_opt': {
    'available': wasm_opt_mode != 'unavailable',
    'mode': wasm_opt_mode,
    'label': wasm_opt_label,
  },
}
if wasm_opt_mode != 'unavailable':
  summary['optimized'] = {
    'raw': str((root / 'optimized-raw').relative_to(root).as_posix()),
    'compressed': str((root / 'optimized-compressed').relative_to(root).as_posix()),
  }

(root / summary_name).write_text(json.dumps(summary, indent=2) + '\n', encoding='utf-8')
PY

echo "Wrote compression comparison to $RESOLVED_OUT_DIR/$SUMMARY_NAME"
