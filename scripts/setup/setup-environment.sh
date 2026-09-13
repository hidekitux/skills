#!/usr/bin/env bash
set -euo pipefail

root=${SETUP_ROOT:-$(git rev-parse --show-toplevel)}
source "${root}/scripts/setup/environment-state.sh"
setup_environment_export

mkdir -p \
  "${MISE_DATA_DIR}" \
  "${MISE_INSTALLS_DIR}" \
  "${MISE_CACHE_DIR}" \
  "${MISE_STATE_DIR}" \
  "${MISE_SHIMS_DIR}" \
  "${GOCACHE}" \
  "${GOMODCACHE}" \
  "${GOPATH}" \
  "${RUFF_CACHE_DIR}" \
  "${FSLC_BIN_DIR}"

if [[ -n "${GITHUB_ENV:-}" ]]; then
  setup_environment_export_lines >> "${GITHUB_ENV}"
fi

echo "Worktree environment prepared before the next mise invocation"
