#!/usr/bin/env bash
set -euo pipefail

PACKAGE="."
OUT_DIR="bin/wasm-release"
BINARY_NAME="app.wasm"
MANIFEST_NAME="wasm-release-manifest.json"
BUDGETS_PATH=""
LD_FLAGS="-s -w"
SKIP_COMPRESSION=0
KEEP_BUILD_INFO=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --package) PACKAGE="$2"; shift 2 ;;
    --out-dir) OUT_DIR="$2"; shift 2 ;;
    --binary-name) BINARY_NAME="$2"; shift 2 ;;
    --manifest-name) MANIFEST_NAME="$2"; shift 2 ;;
    --budgets-path) BUDGETS_PATH="$2"; shift 2 ;;
    --ld-flags) LD_FLAGS="$2"; shift 2 ;;
    --skip-compression) SKIP_COMPRESSION=1; shift ;;
    --keep-build-info) KEEP_BUILD_INFO=1; shift ;;
    *) echo "unknown arg: $1" >&2; exit 1 ;;
  esac
done

if ! command -v python3 >/dev/null 2>&1; then
  echo "Error: python3 is required." >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BROTLI_HELPER="$SCRIPT_DIR/write-brotli-file.mjs"

sha256_hex() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
    return
  fi
  shasum -a 256 "$1" | awk '{print $1}'
}

write_brotli() {
  local src="$1"
  local dst="$2"
  if command -v brotli >/dev/null 2>&1; then
    brotli -f -q 11 -o "$dst" "$src"
    return
  fi
  if command -v node >/dev/null 2>&1 && [[ -f "$BROTLI_HELPER" ]]; then
    node "$BROTLI_HELPER" "$src" "$dst"
    return
  fi
  return 1
}

RESOLVED_OUT_DIR="$(python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "$OUT_DIR")"
mkdir -p "$RESOLVED_OUT_DIR"

WASM_PATH="$RESOLVED_OUT_DIR/$BINARY_NAME"
GZIP_PATH="$WASM_PATH.gz"
BROTLI_PATH="$WASM_PATH.br"
MANIFEST_PATH="$RESOLVED_OUT_DIR/$MANIFEST_NAME"

BUILD_ARGS=(build -o "$WASM_PATH")
if [[ "$KEEP_BUILD_INFO" -eq 0 ]]; then
  BUILD_ARGS+=(-trimpath)
  if [[ -n "$LD_FLAGS" ]]; then
    BUILD_ARGS+=(-ldflags="$LD_FLAGS")
  fi
  BUILD_ARGS+=(-buildvcs=false)
fi
BUILD_ARGS+=("$PACKAGE")

GOOS=js GOARCH=wasm go "${BUILD_ARGS[@]}"

declare -A ARTIFACT_PATHS
ARTIFACT_PATHS[wasm]="$WASM_PATH"

if [[ "$SKIP_COMPRESSION" -eq 0 ]]; then
  gzip -n -9 -c "$WASM_PATH" > "$GZIP_PATH"
  ARTIFACT_PATHS[gzip]="$GZIP_PATH"
  if write_brotli "$WASM_PATH" "$BROTLI_PATH"; then
    ARTIFACT_PATHS[brotli]="$BROTLI_PATH"
  else
    echo "Warning: Brotli sidecar skipped because neither brotli nor node helper is available." >&2
  fi
fi

export RESOLVED_OUT_DIR WASM_PATH GZIP_PATH BROTLI_PATH MANIFEST_PATH PACKAGE LD_FLAGS BUDGETS_PATH
export KEEP_BUILD_INFO SKIP_COMPRESSION

python3 - "$RESOLVED_OUT_DIR" "$MANIFEST_PATH" "$PACKAGE" "$BUDGETS_PATH" <<'PY'
import json
import os
import pathlib
import hashlib
import sys

base_dir = pathlib.Path(sys.argv[1])
manifest_path = pathlib.Path(sys.argv[2])
package = sys.argv[3]
budgets_path = sys.argv[4]

artifact_paths = {
    'wasm': os.environ['WASM_PATH'],
}
if os.path.exists(os.environ['GZIP_PATH']):
    artifact_paths['gzip'] = os.environ['GZIP_PATH']
if os.path.exists(os.environ['BROTLI_PATH']):
    artifact_paths['brotli'] = os.environ['BROTLI_PATH']

def record(path_str):
    path = pathlib.Path(path_str)
    return {
        'path': path.relative_to(base_dir).as_posix(),
        'bytes': path.stat().st_size,
        'sha256': hashlib.sha256(path.read_bytes()).hexdigest(),
    }

artifacts = {key: record(value) for key, value in artifact_paths.items()}

if budgets_path:
    budgets = json.loads(pathlib.Path(budgets_path).read_text(encoding='utf-8'))
    checks = [
        ('raw_bytes', 'wasm', 'raw wasm'),
        ('gzip_bytes', 'gzip', 'gzip sidecar'),
        ('brotli_bytes', 'brotli', 'brotli sidecar'),
    ]
    for budget_key, artifact_key, label in checks:
        if budget_key not in budgets or artifact_key not in artifacts:
            continue
        limit = int(budgets[budget_key])
        actual = int(artifacts[artifact_key]['bytes'])
        if actual > limit:
            raise SystemExit(f'Artifact budget exceeded for {label}: {actual} bytes > {limit} bytes')

build_args = ['build', '-o', os.environ['WASM_PATH']]
if os.environ['KEEP_BUILD_INFO'] == '0':
    build_args.append('-trimpath')
    if os.environ['LD_FLAGS']:
        build_args.append(f'-ldflags={os.environ["LD_FLAGS"]}')
    build_args.append('-buildvcs=false')
build_args.append(package)

manifest = {
    'package': package,
    'goos': 'js',
    'goarch': 'wasm',
    'build_args': build_args,
    'artifacts': artifacts,
}
manifest_path.write_text(json.dumps(manifest, indent=2) + '\n', encoding='utf-8')
PY

echo "Built release wasm for $PACKAGE"
echo "Manifest: $MANIFEST_PATH"