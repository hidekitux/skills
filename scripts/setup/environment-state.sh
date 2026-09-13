#!/usr/bin/env bash
set -euo pipefail

setup_root=${SETUP_ROOT:-$(git rev-parse --show-toplevel)}
environment_root=${SKILLS_ENVIRONMENT_ROOT:-${setup_root}/.mise}

setup_environment_export() {
  export MISE_DATA_DIR="${environment_root}/data"
  export MISE_INSTALLS_DIR="${environment_root}/installs"
  export MISE_CACHE_DIR="${environment_root}/cache"
  export MISE_STATE_DIR="${environment_root}/state"
  export MISE_SHIMS_DIR="${environment_root}/shims"
  export GOCACHE="${environment_root}/go-build"
  export GOMODCACHE="${environment_root}/go-modcache"
  export GOPATH="${environment_root}/gopath"
  export RUFF_CACHE_DIR="${environment_root}/ruff"
  export FSLC_BIN_DIR="${environment_root}/fslc"
}

setup_environment_export_lines() {
  printf 'MISE_DATA_DIR=%s\n' "${MISE_DATA_DIR}"
  printf 'MISE_INSTALLS_DIR=%s\n' "${MISE_INSTALLS_DIR}"
  printf 'MISE_CACHE_DIR=%s\n' "${MISE_CACHE_DIR}"
  printf 'MISE_STATE_DIR=%s\n' "${MISE_STATE_DIR}"
  printf 'MISE_SHIMS_DIR=%s\n' "${MISE_SHIMS_DIR}"
  printf 'GOCACHE=%s\n' "${GOCACHE}"
  printf 'GOMODCACHE=%s\n' "${GOMODCACHE}"
  printf 'GOPATH=%s\n' "${GOPATH}"
  printf 'RUFF_CACHE_DIR=%s\n' "${RUFF_CACHE_DIR}"
  printf 'FSLC_BIN_DIR=%s\n' "${FSLC_BIN_DIR}"
}
