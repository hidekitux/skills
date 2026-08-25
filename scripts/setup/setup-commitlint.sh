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

# Build the repository-local commit-message validator beside commitlint so a
# commit never compiles code or fetches modules; only validate-commit-message
# and commitlint run during a commit. Rebuild whenever the source tree is
# newer than the installed binary so a pulled commit never leaves a stale
# validator in place.
needs_validator_build=0
if [[ ! -x "${shared_dir}/validate-commit-message" ]]; then
  needs_validator_build=1
elif [[ -n "$(find "${root}/cmd" "${root}/internal" "${root}/go.mod" "${root}/go.sum" -type f -newer "${shared_dir}/validate-commit-message" -print -quit 2>/dev/null)" ]]; then
  needs_validator_build=1
fi
if (( needs_validator_build )); then
  mkdir -p "${shared_dir}"
  (cd "${root}" && go build -o "${shared_dir}/validate-commit-message" ./cmd/validate-commit-message)
  echo "Go validate-commit-message rebuilt at ${shared_dir}/validate-commit-message"
fi

ln -sfn "${shared_dir}/validate-commit-message" "${bin_dir}/validate-commit-message"
echo "validate-commit-message available at ${bin_dir}/validate-commit-message"
