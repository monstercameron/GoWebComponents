#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $(basename "$0") <baseline.txt> <candidate.txt>" >&2
  exit 1
fi

if command -v benchstat >/dev/null 2>&1; then
  benchstat "$1" "$2"
  exit 0
fi

echo "benchstat is not installed. Install with: go install golang.org/x/perf/cmd/benchstat@latest"
echo "Baseline:  $1"
echo "Candidate: $2"
