#!/usr/bin/env bash
set -euo pipefail

setup_root=${SETUP_ROOT:-$(git rev-parse --show-toplevel)}
environment_root=${SKILLS_ENVIRONMENT_ROOT:-${setup_root}/.mise}
shared_cache_root=${SKILLS_SHARED_CACHE_ROOT:-${XDG_CACHE_HOME:-${HOME:-/tmp}/.cache}/hidekitux-skills}

if [[ "${CI:-}" == "true" ]]; then
  environment_root=${SKILLS_ENVIRONMENT_ROOT:-${RUNNER_TEMP:-${setup_root}/.mise}/skills-worktree}
  shared_cache_root=${SKILLS_SHARED_CACHE_ROOT:-${environment_root}}
fi

setup_environment_export() {
  export MISE_DATA_DIR="${environment_root}/data"
  if [[ "${CI:-}" == "true" ]]; then
    export MISE_INSTALLS_DIR="${MISE_DATA_DIR}/installs"
    export MISE_CACHE_DIR="${MISE_DATA_DIR}/cache"
    export MISE_SHARED_INSTALL_DIRS="${MISE_INSTALLS_DIR}"
    export GOMODCACHE="${MISE_DATA_DIR}/go-modcache"
    export RUFF_CACHE_DIR="${MISE_DATA_DIR}/ruff"
  else
    export MISE_INSTALLS_DIR="${environment_root}/installs"
    export MISE_CACHE_DIR="${shared_cache_root}/mise-cache"
    export MISE_SHARED_INSTALL_DIRS="${shared_cache_root}/mise-installs"
    export GOMODCACHE="${shared_cache_root}/go-modcache"
    export RUFF_CACHE_DIR="${shared_cache_root}/ruff-cache"
  fi
  export MISE_STATE_DIR="${environment_root}/state"
  export MISE_SHIMS_DIR="${environment_root}/shims"
  export GOCACHE="${environment_root}/go-build"
  export GOPATH="${environment_root}/gopath"
  export FSLC_BIN_DIR="${environment_root}/fslc"
  export MISE_GLOBAL_CONFIG_ROOT="${environment_root}/global-config"
  export SETUP_SHARED_CACHE_ROOT="${shared_cache_root}"
  export SETUP_LOCK_ROOT="${shared_cache_root}/locks"
}

setup_tool_list() {
  printf '%s\n' go
}

setup_environment_export_lines() {
  printf 'MISE_DATA_DIR=%s\n' "${MISE_DATA_DIR}"
  printf 'MISE_INSTALLS_DIR=%s\n' "${MISE_INSTALLS_DIR}"
  printf 'MISE_CACHE_DIR=%s\n' "${MISE_CACHE_DIR}"
  printf 'MISE_SHARED_INSTALL_DIRS=%s\n' "${MISE_SHARED_INSTALL_DIRS}"
  printf 'MISE_STATE_DIR=%s\n' "${MISE_STATE_DIR}"
  printf 'MISE_SHIMS_DIR=%s\n' "${MISE_SHIMS_DIR}"
  printf 'GOCACHE=%s\n' "${GOCACHE}"
  printf 'GOMODCACHE=%s\n' "${GOMODCACHE}"
  printf 'GOPATH=%s\n' "${GOPATH}"
  printf 'RUFF_CACHE_DIR=%s\n' "${RUFF_CACHE_DIR}"
  printf 'FSLC_BIN_DIR=%s\n' "${FSLC_BIN_DIR}"
  printf 'MISE_GLOBAL_CONFIG_ROOT=%s\n' "${MISE_GLOBAL_CONFIG_ROOT}"
  printf 'SETUP_SHARED_CACHE_ROOT=%s\n' "${SETUP_SHARED_CACHE_ROOT}"
  printf 'SETUP_LOCK_ROOT=%s\n' "${SETUP_LOCK_ROOT}"
}
