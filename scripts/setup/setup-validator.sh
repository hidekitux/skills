#!/usr/bin/env bash
set -euo pipefail

root=${SETUP_ROOT:-$(git rev-parse --show-toplevel)}
source "${root}/scripts/setup/environment-state.sh"
setup_environment_export

bin_dir="${root}/.mise/bin"
mkdir -p "${bin_dir}"
temporary=$(mktemp -d "${root}/.mise/.validator.XXXXXX")
cleanup() {
  rm -rf "${temporary}"
}
trap cleanup EXIT

(cd "${root}" && go build -o "${temporary}/validate-commit-message" ./cmd/validate-commit-message)
mv "${temporary}/validate-commit-message" "${bin_dir}/validate-commit-message"
echo "validate-commit-message available in the current Worktree"
