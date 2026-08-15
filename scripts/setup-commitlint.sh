#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
bin_dir="${root}/.mise/bin"

mkdir -p "${bin_dir}"
GOBIN="${bin_dir}" go install github.com/conventionalcommit/commitlint@v0.12.0
git config core.hooksPath .githooks
echo "Go commitlint installed at ${bin_dir}/commitlint"
echo "Git commit hooks enabled via core.hooksPath=.githooks"
