#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -eq 0 ]]; then
  echo "usage: run-mise.sh <mise arguments...>" >&2
  exit 2
fi

root=${SETUP_ROOT:-$(git rev-parse --show-toplevel)}
source "${root}/scripts/setup/environment-state.sh"
setup_environment_export

exec mise "$@"
