#!/usr/bin/env bash
set -euo pipefail

EXAMPLE=""
VERBOSE=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --example) EXAMPLE="$2"; shift 2 ;;
    --verbose) VERBOSE=1; shift ;;
    *) echo "unknown arg: $1" >&2; exit 1 ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
STATIC_DIR="$SCRIPT_DIR/static"

resolve_examples_build_dir() {
  node --input-type=module -e "import { resolveWorkspaceBuildPath } from './scripts/runner-paths.mjs'; console.log(resolveWorkspaceBuildPath(process.argv[1], 'examples'));" "$REPO_ROOT"
}

BIN_DIR="$(resolve_examples_build_dir)"

echo "GoWebComponents Examples Build System"
echo "======================================"
echo

echo "[INFO] Refreshing shared Tailwind CSS"
pushd "$STATIC_DIR" >/dev/null
npm run build:css >/dev/null
popd >/dev/null

mkdir -p "$BIN_DIR"

mapfile -t EXAMPLE_DIRS < <(find "$SCRIPT_DIR" -mindepth 1 -maxdepth 1 -type d -printf '%f\n' | grep -E '(^[0-9]{2}-|^gwc-examples-site$)' | sort)
if [[ ${#EXAMPLE_DIRS[@]} -eq 0 ]]; then
  echo "[ERROR] No example directories found matching the configured build set" >&2
  exit 1
fi

if [[ -n "$EXAMPLE" ]]; then
  FILTERED=()
  for dir in "${EXAMPLE_DIRS[@]}"; do
    if [[ "$dir" == "$EXAMPLE" ]]; then
      FILTERED+=("$dir")
    fi
  done
  EXAMPLE_DIRS=("${FILTERED[@]}")
  if [[ ${#EXAMPLE_DIRS[@]} -eq 0 ]]; then
    echo "[ERROR] Example '$EXAMPLE' not found" >&2
    exit 1
  fi
fi

SUCCESSFUL=0
FAILED=0
TOTAL_SIZE=0

for example_name in "${EXAMPLE_DIRS[@]}"; do
  example_dir="$SCRIPT_DIR/$example_name"
  entry_go=""
  if [[ -f "$example_dir/main.go" ]]; then
    entry_go="$example_dir/main.go"
  elif [[ -f "$example_dir/client/main.go" ]]; then
    entry_go="$example_dir/client/main.go"
  fi

  if [[ -z "$entry_go" ]]; then
    echo "[SKIP] $example_name - no supported main.go entrypoint found"
    continue
  fi

  build_dir="$(dirname "$entry_go")"
  clean_name="${example_name#[0-9][0-9]-}"
  output_path="$BIN_DIR/$clean_name.wasm"

  echo "[BUILD] $example_name..."
  if build_output=$(cd "$build_dir" && GOOS=js GOARCH=wasm go build -o "$output_path" . 2>&1); then
    size=$(python3 -c 'import os,sys; print(os.path.getsize(sys.argv[1]))' "$output_path")
    size_kb=$(python3 -c 'import sys; print(round(int(sys.argv[1]) / 1024, 2))' "$size")
    echo "  [OK] $clean_name.wasm ($size_kb KB)"
    SUCCESSFUL=$((SUCCESSFUL + 1))
    TOTAL_SIZE=$((TOTAL_SIZE + size))
  else
    echo "  [FAIL] Failed to build $example_name" >&2
    if [[ "$VERBOSE" -eq 1 ]]; then
      echo "$build_output" >&2
    fi
    FAILED=$((FAILED + 1))
  fi
done

echo
echo "Build Summary"
echo "============="
echo
echo "Successful: $SUCCESSFUL"
echo "Failed: $FAILED"
echo "Total size: $(python3 -c 'import sys; print(round(int(sys.argv[1]) / 1024, 2))' "$TOTAL_SIZE") KB"

if [[ "$FAILED" -eq 0 ]]; then
  echo
  echo "All examples built successfully!"
  exit 0
fi

echo
echo "Some examples failed to build" >&2
exit 1