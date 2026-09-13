#!/usr/bin/env bash
set -euo pipefail

root=${SETUP_ROOT:-$(git rev-parse --show-toplevel)}
source "${root}/scripts/setup/environment-state.sh"
setup_environment_export

bin_dir="${root}/.mise/bin"
mkdir -p "${bin_dir}"
temporary=$(mktemp -d "${root}/.mise/.commitlint.XXXXXX")
cleanup() {
  rm -rf "${temporary}"
}
trap cleanup EXIT

GOBIN="${temporary}" go install github.com/conventionalcommit/commitlint@v0.12.0
mv "${temporary}/commitlint" "${bin_dir}/commitlint"
echo "commitlint available in the current Worktree"
