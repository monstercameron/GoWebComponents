#!/usr/bin/env bash
set -euo pipefail

PACKAGE="./examples/21-ui-render"
OUT_DIR="bin/wasm-build-cache-comparison"
BINARY_NAME="app.wasm"
SUMMARY_NAME="wasm-build-cache-comparison.json"
RELEASE_PROFILE=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --package) PACKAGE="$2"; shift 2 ;;
    --out-dir) OUT_DIR="$2"; shift 2 ;;
    --binary-name) BINARY_NAME="$2"; shift 2 ;;
    --summary-name) SUMMARY_NAME="$2"; shift 2 ;;
    --release-profile) RELEASE_PROFILE=1; shift ;;
    *) echo "unknown arg: $1" >&2; exit 1 ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOLVED_OUT_DIR="$(python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "$OUT_DIR")"
mkdir -p "$RESOLVED_OUT_DIR"

SHARED_GOCACHE="$RESOLVED_OUT_DIR/shared-gocache"
CI_GOCACHE="$RESOLVED_OUT_DIR/ci-gocache"
CI_GOMODCACHE="$RESOLVED_OUT_DIR/ci-gomodcache"
ISOLATED_GOCACHE="$RESOLVED_OUT_DIR/isolated-gocache"

rm -rf "$SHARED_GOCACHE" "$CI_GOCACHE" "$CI_GOMODCACHE" "$ISOLATED_GOCACHE"
mkdir -p "$SHARED_GOCACHE" "$CI_GOCACHE" "$CI_GOMODCACHE" "$ISOLATED_GOCACHE"

DEFAULT_GOCACHE="$(go env GOCACHE | tr -d '\r')"
DEFAULT_GOMODCACHE="$(go env GOMODCACHE | tr -d '\r')"
PACKAGE_DIR="$(python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "$PACKAGE")"

find_edit_file() {
  find "$PACKAGE_DIR" -maxdepth 1 -type f -name '*.go' ! -name '*_test.go' | sort | head -n 1
}

run_variant() {
  local key="$1"
  local gocache="$2"
  local gomodcache="$3"
  local small_edit="$4"
  local prepare_module_cache="$5"
  local variant_out_dir="$RESOLVED_OUT_DIR/$key"
  mkdir -p "$variant_out_dir"

  if [[ "$prepare_module_cache" == "1" ]]; then
    GOCACHE="$gocache" GOMODCACHE="$gomodcache" go mod download
  fi

  local edit_file=""
  local backup_file=""
  if [[ "$small_edit" == "1" ]]; then
    edit_file="$(find_edit_file)"
    if [[ -z "$edit_file" ]]; then
      echo "No editable Go source file found for package: $PACKAGE" >&2
      exit 1
    fi
    backup_file="$variant_out_dir/.edit-backup"
    cp "$edit_file" "$backup_file"
    printf '\n// cache experiment marker\n' >> "$edit_file"
  fi

  local measure_args=(--package "$PACKAGE" --out-dir "$variant_out_dir" --binary-name "$BINARY_NAME" --manifest-name wasm-build-experiment.json)
  if [[ "$RELEASE_PROFILE" == "1" ]]; then
    measure_args+=(--release-profile)
  fi
  GOCACHE="$gocache" GOMODCACHE="$gomodcache" "$SCRIPT_DIR/measure-wasm-build.sh" "${measure_args[@]}"

  if [[ -n "$backup_file" ]]; then
    mv "$backup_file" "$edit_file"
  fi
}

run_variant "shared-cache-cold" "$SHARED_GOCACHE" "$DEFAULT_GOMODCACHE" 0 0
run_variant "shared-cache-warm" "$SHARED_GOCACHE" "$DEFAULT_GOMODCACHE" 0 0
run_variant "shared-cache-small-edit" "$SHARED_GOCACHE" "$DEFAULT_GOMODCACHE" 1 0
run_variant "isolated-build-cache" "$ISOLATED_GOCACHE" "$DEFAULT_GOMODCACHE" 0 0
rm -rf "$CI_GOCACHE" "$CI_GOMODCACHE"
mkdir -p "$CI_GOCACHE" "$CI_GOMODCACHE"
run_variant "ci-style-cold" "$CI_GOCACHE" "$CI_GOMODCACHE" 0 1
run_variant "ci-style-warm" "$CI_GOCACHE" "$CI_GOMODCACHE" 0 0
run_variant "ci-style-small-edit" "$CI_GOCACHE" "$CI_GOMODCACHE" 1 0

python3 - "$RESOLVED_OUT_DIR" "$SUMMARY_NAME" "$PACKAGE" "$DEFAULT_GOCACHE" "$DEFAULT_GOMODCACHE" <<'PY'
import datetime as dt
import json
import pathlib
import subprocess
import sys

root = pathlib.Path(sys.argv[1])
summary_name = sys.argv[2]
package = sys.argv[3]
default_gocache = sys.argv[4]
default_gomodcache = sys.argv[5]

variants = {}
for manifest_path in sorted(root.glob('*/wasm-build-experiment.json')):
  variants[manifest_path.parent.name] = json.loads(manifest_path.read_text(encoding='utf-8'))

summary = {
  'package': package,
  'generated_at': dt.datetime.now(dt.timezone.utc).isoformat(),
  'environment': {
    'go_version': subprocess.check_output(['go', 'version'], text=True).strip(),
    'default_gocache': default_gocache,
    'default_gomodcache': default_gomodcache,
  },
  'variants': variants,
}
(root / summary_name).write_text(json.dumps(summary, indent=2) + '\n', encoding='utf-8')
PY

echo "Wrote cache comparison to $RESOLVED_OUT_DIR/$SUMMARY_NAME"