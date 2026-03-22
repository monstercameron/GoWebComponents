#!/usr/bin/env bash
set -euo pipefail

PACKAGE=""
BASELINE_GO=""
CANDIDATE_GO=""
OUT_DIR="bin/wasm-toolchain-comparison"
TIMING_REGRESSION_PERCENT="10"
SIZE_REGRESSION_PERCENT="0"
OTHER_REGRESSION_PERCENT="0"
RELEASE_PROFILE=0
SKIP_COMPRESSION=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --package) PACKAGE="$2"; shift 2 ;;
    --baseline-go) BASELINE_GO="$2"; shift 2 ;;
    --candidate-go) CANDIDATE_GO="$2"; shift 2 ;;
    --out-dir) OUT_DIR="$2"; shift 2 ;;
    --timing-regression-percent) TIMING_REGRESSION_PERCENT="$2"; shift 2 ;;
    --size-regression-percent) SIZE_REGRESSION_PERCENT="$2"; shift 2 ;;
    --other-regression-percent) OTHER_REGRESSION_PERCENT="$2"; shift 2 ;;
    --release-profile) RELEASE_PROFILE=1; shift ;;
    --skip-compression) SKIP_COMPRESSION=1; shift ;;
    *) echo "unknown arg: $1" >&2; exit 1 ;;
  esac
done

if [[ -z "$PACKAGE" || -z "$BASELINE_GO" || -z "$CANDIDATE_GO" ]]; then
  echo "usage: $(basename "$0") --package <pkg> --baseline-go <path> --candidate-go <path> [options]" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOLVED_OUT_DIR="$(python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "$OUT_DIR")"
mkdir -p "$RESOLVED_OUT_DIR"

BASELINE_OUT_DIR="$RESOLVED_OUT_DIR/baseline"
CANDIDATE_OUT_DIR="$RESOLVED_OUT_DIR/candidate"
COMPARISON_PATH="$RESOLVED_OUT_DIR/toolchain-comparison.json"
SUMMARY_PATH="$RESOLVED_OUT_DIR/wasm-toolchain-comparison.json"

MEASURE_ARGS=(--package "$PACKAGE")
if [[ "$RELEASE_PROFILE" -eq 1 ]]; then
  MEASURE_ARGS+=(--release-profile)
fi
if [[ "$SKIP_COMPRESSION" -eq 1 ]]; then
  MEASURE_ARGS+=(--skip-compression)
fi

"$SCRIPT_DIR/measure-wasm-build.sh" "${MEASURE_ARGS[@]}" --go-executable "$BASELINE_GO" --out-dir "$BASELINE_OUT_DIR"
"$SCRIPT_DIR/measure-wasm-build.sh" "${MEASURE_ARGS[@]}" --go-executable "$CANDIDATE_GO" --out-dir "$CANDIDATE_OUT_DIR"

set +e
"$SCRIPT_DIR/compare-wasm-experiment.sh" \
  --baseline "$BASELINE_OUT_DIR/wasm-build-experiment.json" \
  --candidate "$CANDIDATE_OUT_DIR/wasm-build-experiment.json" \
  --out-file "$COMPARISON_PATH" \
  --timing-regression-percent "$TIMING_REGRESSION_PERCENT" \
  --size-regression-percent "$SIZE_REGRESSION_PERCENT" \
  --other-regression-percent "$OTHER_REGRESSION_PERCENT"
COMPARISON_EXIT=$?
set -e

export COMPARISON_EXIT
python3 - "$SUMMARY_PATH" "$PACKAGE" "$BASELINE_GO" "$CANDIDATE_GO" "$TIMING_REGRESSION_PERCENT" "$SIZE_REGRESSION_PERCENT" "$OTHER_REGRESSION_PERCENT" <<'PY'
import datetime as dt
import json
import pathlib
import subprocess
import sys

summary_path = pathlib.Path(sys.argv[1])
package = sys.argv[2]
baseline_go = sys.argv[3]
candidate_go = sys.argv[4]
timing = float(sys.argv[5])
size = float(sys.argv[6])
other = float(sys.argv[7])

summary = {
  'package': package,
  'compared_at': dt.datetime.now(dt.timezone.utc).isoformat(),
  'baseline': {
    'go_executable': baseline_go,
    'go_version': subprocess.check_output([baseline_go, 'version'], text=True).strip(),
    'manifest': 'baseline/wasm-build-experiment.json',
  },
  'candidate': {
    'go_executable': candidate_go,
    'go_version': subprocess.check_output([candidate_go, 'version'], text=True).strip(),
    'manifest': 'candidate/wasm-build-experiment.json',
  },
  'thresholds': {
    'timing_regression_percent': timing,
    'size_regression_percent': size,
    'other_regression_percent': other,
  },
  'comparison': 'toolchain-comparison.json',
  'regression_exit_code': int(__import__('os').environ['COMPARISON_EXIT']),
}
summary_path.write_text(json.dumps(summary, indent=2) + '\n', encoding='utf-8')
PY

echo "Saved toolchain comparison summary to $SUMMARY_PATH"
exit "$COMPARISON_EXIT"