#!/usr/bin/env bash
set -euo pipefail

BASELINE=""
CANDIDATE=""
OUT_FILE=""
TIMING_REGRESSION_PERCENT="10"
SIZE_REGRESSION_PERCENT="0"
OTHER_REGRESSION_PERCENT="0"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --baseline) BASELINE="$2"; shift 2 ;;
    --candidate) CANDIDATE="$2"; shift 2 ;;
    --out-file) OUT_FILE="$2"; shift 2 ;;
    --timing-regression-percent) TIMING_REGRESSION_PERCENT="$2"; shift 2 ;;
    --size-regression-percent) SIZE_REGRESSION_PERCENT="$2"; shift 2 ;;
    --other-regression-percent) OTHER_REGRESSION_PERCENT="$2"; shift 2 ;;
    *) echo "unknown arg: $1" >&2; exit 1 ;;
  esac
done

if [[ -z "$BASELINE" || -z "$CANDIDATE" ]]; then
  echo "usage: $(basename "$0") --baseline <path> --candidate <path> [--out-file <path>] [--timing-regression-percent <n>] [--size-regression-percent <n>] [--other-regression-percent <n>]" >&2
  exit 1
fi

python3 - "$BASELINE" "$CANDIDATE" "$OUT_FILE" "$TIMING_REGRESSION_PERCENT" "$SIZE_REGRESSION_PERCENT" "$OTHER_REGRESSION_PERCENT" <<'PY'
import datetime as dt
import json
import math
import pathlib
import sys

baseline_path = pathlib.Path(sys.argv[1]).resolve()
candidate_path = pathlib.Path(sys.argv[2]).resolve()
out_file = sys.argv[3]
timing_threshold = float(sys.argv[4])
size_threshold = float(sys.argv[5])
other_threshold = float(sys.argv[6])

baseline = json.loads(baseline_path.read_text(encoding='utf-8'))
candidate = json.loads(candidate_path.read_text(encoding='utf-8'))

def collect(value, prefix=""):
    metrics = {}
    if value is None or isinstance(value, bool):
      return metrics
    if isinstance(value, dict):
      for key, child in value.items():
        child_prefix = key if not prefix else f"{prefix}.{key}"
        metrics.update(collect(child, child_prefix))
      return metrics
    if isinstance(value, list):
      for index, child in enumerate(value):
        child_prefix = f"[{index}]" if not prefix else f"{prefix}[{index}]"
        metrics.update(collect(child, child_prefix))
      return metrics
    if isinstance(value, (int, float)):
      metrics[prefix] = float(value)
    return metrics

def category(path):
    if path.endswith('_ms') or 'module_download_ms' in path:
      return 'timing'
    if path.endswith('bytes'):
      return 'size'
    return 'other'

def threshold(path):
    kind = category(path)
    if kind == 'timing':
      return timing_threshold
    if kind == 'size':
      return size_threshold
    return other_threshold

baseline_metrics = collect(baseline)
candidate_metrics = collect(candidate)
paths = sorted(set(baseline_metrics) | set(candidate_metrics))
results = []
regressions = []

for path in paths:
    baseline_has = path in baseline_metrics
    candidate_has = path in candidate_metrics
    if not baseline_has:
      results.append({'path': path, 'status': 'added', 'category': category(path), 'baseline': None, 'candidate': candidate_metrics[path], 'delta': None, 'delta_percent': None, 'threshold_percent': None})
      continue
    if not candidate_has:
      results.append({'path': path, 'status': 'removed', 'category': category(path), 'baseline': baseline_metrics[path], 'candidate': None, 'delta': None, 'delta_percent': None, 'threshold_percent': None})
      continue
    baseline_value = baseline_metrics[path]
    candidate_value = candidate_metrics[path]
    delta = candidate_value - baseline_value
    if baseline_value == 0:
      delta_percent = 0.0 if candidate_value == 0 else None
    else:
      delta_percent = (delta / baseline_value) * 100.0
    limit = threshold(path)
    if delta < 0:
      status = 'improved'
    elif delta > 0:
      status = 'regressed' if delta_percent is not None and delta_percent > limit else 'within-threshold'
    else:
      status = 'unchanged'
    record = {
      'path': path,
      'status': status,
      'category': category(path),
      'baseline': baseline_value,
      'candidate': candidate_value,
      'delta': delta,
      'delta_percent': None if delta_percent is None else round(delta_percent, 4),
      'threshold_percent': limit,
    }
    results.append(record)
    if status == 'regressed':
      regressions.append(record)

summary = {
    'baseline': str(baseline_path),
    'candidate': str(candidate_path),
    'compared_at': dt.datetime.now(dt.timezone.utc).isoformat(),
    'thresholds': {
      'timing_regression_percent': timing_threshold,
      'size_regression_percent': size_threshold,
      'other_regression_percent': other_threshold,
    },
    'counts': {
      'total_metrics': len(results),
      'regressions': len(regressions),
    },
    'results': results,
    'regressions': regressions,
  }

text = json.dumps(summary, indent=2) + '\n'
if out_file:
  pathlib.Path(out_file).write_text(text, encoding='utf-8')
print(text)
raise SystemExit(min(len(regressions), 255))
PY