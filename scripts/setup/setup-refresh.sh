#!/usr/bin/env bash
set -euo pipefail

root=${SETUP_ROOT:-$(git rev-parse --show-toplevel)}
source "${root}/scripts/setup/setup-state.sh"

revision=$(git -C "${root}" rev-parse HEAD)
bootstrap_fingerprint=$(bootstrap_inputs)
validator_fingerprint=$(validator_inputs)
state_status=$(state_value status || true)
state_revision=$(state_value revision || true)
state_bootstrap=$(state_value bootstrap_inputs || true)
state_validator=$(state_value validator_inputs || true)

if [[ "${state_status}" == "ready" &&
  "${state_revision}" == "${revision}" &&
  "${state_bootstrap}" == "${bootstrap_fingerprint}" &&
  "${state_validator}" == "${validator_fingerprint}" ]]; then
  echo "Worktree setup is current for ${revision}"
  exit 0
fi

if [[ "${state_status}" != "ready" &&
  "${state_status}" != "bootstrap-ready" ||
  "${state_bootstrap}" != "${bootstrap_fingerprint}" ]]; then
  if ! bash "${root}/scripts/setup/setup-bootstrap.sh"; then
    echo "Refresh failed during bootstrap. Rerun 'mise run setup:refresh' after resolving the reported error." >&2
    exit 1
  fi
fi

if [[ "${state_validator}" != "${validator_fingerprint}" ||
  "${state_status}" != "ready" ]]; then
  if ! bash "${root}/scripts/setup/setup-validator.sh"; then
    echo "Refresh failed while preparing the commit-message validator. Rerun 'mise run setup:refresh' after resolving the reported error." >&2
    exit 1
  fi
fi

if ! bash "${root}/scripts/setup/register-local-skills.sh"; then
  echo "Refresh failed while registering local skills. Rerun 'mise run setup:refresh' after resolving the reported error." >&2
  exit 1
fi

write_state "ready" "${revision}" "${bootstrap_fingerprint}" "${validator_fingerprint}"
echo "Worktree refresh complete for ${revision}"
