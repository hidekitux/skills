#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
common_dir="$(git rev-parse --git-common-dir)"
case "${common_dir}" in
  /*) shared_dir="${common_dir}/.mise/bin" ;;
  *) shared_dir="${root}/${common_dir}/.mise/bin" ;;
esac
bin_dir="${root}/.mise/bin"

mkdir -p "${bin_dir}"
if [[ -x "${shared_dir}/commitlint" ]]; then
  echo "Using shared commitlint at ${shared_dir}/commitlint"
elif [[ -x "${bin_dir}/commitlint" ]]; then
  echo "Using worktree commitlint at ${bin_dir}/commitlint"
else
  if [[ ! -d "${shared_dir}" ]]; then
    mkdir -p "${shared_dir}"
  fi
  GOBIN="${shared_dir}" go install github.com/conventionalcommit/commitlint@v0.12.0
  echo "Go commitlint installed at ${shared_dir}/commitlint"
fi

ln -sfn "${shared_dir}/commitlint" "${bin_dir}/commitlint"
echo "commitlint available at ${bin_dir}/commitlint"
