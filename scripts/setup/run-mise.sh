#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -eq 0 ]]; then
  echo "usage: run-mise.sh <mise arguments...>" >&2
  exit 2
fi

root=${SETUP_ROOT:-$(git rev-parse --show-toplevel)}
source "${root}/scripts/setup/environment-state.sh"
setup_environment_export
source "${root}/scripts/setup/setup-state.sh"

setup_command=0
selected_tools="repository-configured-tools"
if [[ "${1:-}" == "run" && "${2:-}" == setup:* ]]; then
  setup_command=1
  selected_tools=$(setup_tool_list | paste -sd, -)
elif [[ "${1:-}" == "install" ]]; then
  selected_tools="${*:2}"
  [[ -n "${selected_tools}" ]] || selected_tools="repository-configured-tools"
fi

if [[ "${setup_command}" == "1" || "${1:-}" == "install" ]]; then
  setup_lock_acquire
fi

started_at=$(date +%s)
if [[ "${setup_command}" == "1" ]]; then
  if ! mise install go; then
    echo "mise setup: required tool installation failed; rerun the same setup command after resolving the reported error" >&2
    exit 1
  fi
  export MISE_AUTO_INSTALL=0
fi

set +e
mise "$@"
status=$?
set -e

elapsed=$(( $(date +%s) - started_at ))
if [[ "${setup_command}" == "1" || "${1:-}" == "install" ]]; then
  if [[ "${status}" == "0" ]]; then
    echo "mise setup: config=repository-only tools=${selected_tools} reuse=shared-compatible elapsed=${elapsed}s recovery=rerun-the-same-command" >&2
  else
    echo "mise setup: config=repository-only tools=${selected_tools} reuse=shared-compatible elapsed=${elapsed}s recovery=rerun-the-same-command-after-resolving-the-error" >&2
  fi
fi

setup_lock_release
exit "${status}"
