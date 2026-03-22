#!/usr/bin/env bash
set -euo pipefail

PACKAGE="."
OUT_DIR="bin/wasm-build-experiment"
BINARY_NAME="app.wasm"
MANIFEST_NAME="wasm-build-experiment.json"
GO_EXECUTABLE="go"
LD_FLAGS="-s -w"
RELEASE_PROFILE=0
SKIP_COMPRESSION=0
SERVE_RELOAD_MS=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --package) PACKAGE="$2"; shift 2 ;;
    --out-dir) OUT_DIR="$2"; shift 2 ;;
    --binary-name) BINARY_NAME="$2"; shift 2 ;;
    --manifest-name) MANIFEST_NAME="$2"; shift 2 ;;
    --go-executable) GO_EXECUTABLE="$2"; shift 2 ;;
    --ld-flags) LD_FLAGS="$2"; shift 2 ;;
    --release-profile) RELEASE_PROFILE=1; shift ;;
    --skip-compression) SKIP_COMPRESSION=1; shift ;;
    --serve-reload-ms) SERVE_RELOAD_MS="$2"; shift 2 ;;
    *) echo "unknown arg: $1" >&2; exit 1 ;;
  esac
done

if ! command -v python3 >/dev/null 2>&1; then
  echo "Error: python3 is required." >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BROTLI_HELPER="$SCRIPT_DIR/write-brotli-file.mjs"

now_ms() {
  python3 - <<'PY'
import time
print(int(time.time() * 1000))
PY
}

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
if [[ "$RELEASE_PROFILE" -eq 1 ]]; then
  BUILD_ARGS+=(-trimpath)
  if [[ -n "$LD_FLAGS" ]]; then
    BUILD_ARGS+=(-ldflags="$LD_FLAGS")
  fi
  BUILD_ARGS+=(-buildvcs=false)
fi
BUILD_ARGS+=("$PACKAGE")

TOTAL_START="$(now_ms)"
BUILD_START="$(now_ms)"
GOOS=js GOARCH=wasm "$GO_EXECUTABLE" "${BUILD_ARGS[@]}"
BUILD_END="$(now_ms)"
GO_BUILD_MS=$((BUILD_END - BUILD_START))

GZIP_MS=0
BROTLI_MS=0
COMPRESSION_TOTAL_MS=0

if [[ "$SKIP_COMPRESSION" -eq 0 ]]; then
  COMP_START="$(now_ms)"
  GZIP_START="$(now_ms)"
  gzip -n -9 -c "$WASM_PATH" > "$GZIP_PATH"
  GZIP_END="$(now_ms)"
  GZIP_MS=$((GZIP_END - GZIP_START))

  if command -v brotli >/dev/null 2>&1 || { command -v node >/dev/null 2>&1 && [[ -f "$BROTLI_HELPER" ]]; }; then
    BROTLI_START="$(now_ms)"
    write_brotli "$WASM_PATH" "$BROTLI_PATH"
    BROTLI_END="$(now_ms)"
    BROTLI_MS=$((BROTLI_END - BROTLI_START))
  fi
  COMP_END="$(now_ms)"
  COMPRESSION_TOTAL_MS=$((COMP_END - COMP_START))
fi

TOTAL_END="$(now_ms)"
TOTAL_WALL_MS=$((TOTAL_END - TOTAL_START))

export RESOLVED_OUT_DIR WASM_PATH GZIP_PATH BROTLI_PATH MANIFEST_PATH PACKAGE GO_EXECUTABLE LD_FLAGS
export RELEASE_PROFILE SKIP_COMPRESSION SERVE_RELOAD_MS GO_BUILD_MS GZIP_MS BROTLI_MS COMPRESSION_TOTAL_MS TOTAL_WALL_MS

python3 - <<'PY'
import hashlib
import json
import os
import pathlib
import subprocess

base_dir = pathlib.Path(os.environ['RESOLVED_OUT_DIR'])
manifest_path = pathlib.Path(os.environ['MANIFEST_PATH'])
package = os.environ['PACKAGE']
go_executable = os.environ['GO_EXECUTABLE']

def record(path_str):
    path = pathlib.Path(path_str)
    return {
        'path': path.relative_to(base_dir).as_posix(),
        'bytes': path.stat().st_size,
        'sha256': hashlib.sha256(path.read_bytes()).hexdigest(),
    }

artifacts = {'wasm': record(os.environ['WASM_PATH'])}
if os.path.exists(os.environ['GZIP_PATH']):
    artifacts['gzip'] = record(os.environ['GZIP_PATH'])
if os.path.exists(os.environ['BROTLI_PATH']):
    artifacts['brotli'] = record(os.environ['BROTLI_PATH'])

build_args = ['build', '-o', os.environ['WASM_PATH']]
if os.environ['RELEASE_PROFILE'] == '1':
    build_args.append('-trimpath')
    if os.environ['LD_FLAGS']:
        build_args.append(f'-ldflags={os.environ["LD_FLAGS"]}')
    build_args.append('-buildvcs=false')
build_args.append(package)

phases = {
    'go_build_ms': int(os.environ['GO_BUILD_MS']),
    'compression_total_ms': int(os.environ['COMPRESSION_TOTAL_MS']),
    'total_wall_ms': int(os.environ['TOTAL_WALL_MS']),
}
if os.path.exists(os.environ['GZIP_PATH']):
    phases['gzip_ms'] = int(os.environ['GZIP_MS'])
if os.path.exists(os.environ['BROTLI_PATH']):
    phases['brotli_ms'] = int(os.environ['BROTLI_MS'])
if os.environ.get('SERVE_RELOAD_MS'):
    phases['serve_reload_ms'] = int(os.environ['SERVE_RELOAD_MS'])

manifest = {
    'package': package,
    'profile': 'release' if os.environ['RELEASE_PROFILE'] == '1' else 'debug',
    'go_executable': go_executable,
    'go_version': subprocess.check_output([go_executable, 'version'], text=True).strip(),
    'goos': 'js',
    'goarch': 'wasm',
    'build_args': build_args,
    'phases': phases,
    'artifacts': artifacts,
}
manifest_path.write_text(json.dumps(manifest, indent=2) + '\n', encoding='utf-8')
PY

echo "Measured wasm build phases for $PACKAGE"
echo "Manifest: $MANIFEST_PATH"