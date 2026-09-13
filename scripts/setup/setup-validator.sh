#!/usr/bin/env bash
set -euo pipefail

root=${SETUP_ROOT:-$(git rev-parse --show-toplevel)}
source "${root}/scripts/setup/setup-state.sh"

common_dir=$(git -C "${root}" rev-parse --git-common-dir)
case "${common_dir}" in
  /*) shared_dir="${common_dir}/.mise/bin" ;;
  *) shared_dir="${root}/${common_dir}/.mise/bin" ;;
esac
bin_dir="${root}/.mise/bin"
validator="${shared_dir}/validate-commit-message"

mkdir -p "${shared_dir}"
(cd "${root}" && go build -o "${validator}" ./cmd/validate-commit-message)
ensure_link "${validator}" "${bin_dir}/validate-commit-message"
echo "validate-commit-message rebuilt and available at ${bin_dir}/validate-commit-message"
