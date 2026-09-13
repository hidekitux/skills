#!/usr/bin/env bash
set -euo pipefail

root=${SETUP_ROOT:-$(git rev-parse --show-toplevel)}
source "${root}/scripts/setup/setup-state.sh"

bootstrap_fingerprint=$(bootstrap_inputs)
state_status=$(state_value status || true)
state_bootstrap=$(state_value bootstrap_inputs || true)
if [[ "${state_status}" == "ready" || "${state_status}" == "bootstrap-ready" ]] &&
  [[ "${state_bootstrap}" == "${bootstrap_fingerprint}" ]]; then
  echo "Bootstrap is current for $(git -C "${root}" rev-parse HEAD)"
  exit 0
fi

current_hooks_path=$(git -C "${root}" config --local --get core.hooksPath || true)
if [[ "${current_hooks_path}" != ".githooks" ]]; then
  git -C "${root}" config core.hooksPath .githooks
  echo "Git commit hooks enabled via core.hooksPath=.githooks"
else
  echo "Git commit hooks already enabled via core.hooksPath=.githooks"
fi

if ! bash "${root}/scripts/setup/setup-commitlint.sh"; then
  echo "Bootstrap failed while preparing commitlint. Rerun 'mise run setup:bootstrap' after resolving the reported error." >&2
  exit 1
fi

revision=$(git -C "${root}" rev-parse HEAD)
write_state "bootstrap-ready" "${revision}" "${bootstrap_fingerprint}" ""
echo "Bootstrap ready for ${revision}"
