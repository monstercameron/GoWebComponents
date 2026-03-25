#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

BUILD_CLIENT=0
LISTEN_ADDR_VALUE=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --build-client)
      BUILD_CLIENT=1
      shift
      ;;
    --listen-addr)
      LISTEN_ADDR_VALUE="${2:-}"
      shift 2
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

cd "${REPO_ROOT}"

if [[ "${BUILD_CLIENT}" -eq 1 ]]; then
  "${SCRIPT_DIR}/build-client.sh"
fi

if [[ -n "${LISTEN_ADDR_VALUE}" ]]; then
  export LISTEN_ADDR="${LISTEN_ADDR_VALUE}"
fi

go run ./examples/100-ai-chat-wizard/cmd/server
