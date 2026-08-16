#!/usr/bin/env bash
set -euo pipefail

commitlint_bin="${COMMITLINT_BIN:-commitlint}"
root=$(git rev-parse --show-toplevel)
cd "${root}"

commits=()

lint_message() {
  local commit=$1
  echo "Checking commit ${commit}"
  local message
  message=$(git show -s --format=%B "${commit}")
  printf '%s\n' "${message}" | "${commitlint_bin}" lint
  python3 scripts/lint/validate-commit-message.py --message "$(printf '%s\n' "${message}")"
}

if [[ -n "${PR_BASE_SHA:-}" && -n "${PR_HEAD_SHA:-}" ]]; then
  while IFS= read -r commit; do
    commits+=("${commit}")
  done < <(git rev-list --no-merges "${PR_BASE_SHA}..${PR_HEAD_SHA}")
else
  before="${PUSH_BEFORE_SHA:-}"
  after="${GITHUB_SHA:-HEAD}"
  if [[ -z "${before}" || "${before}" =~ ^0+$ ]]; then
    while IFS= read -r commit; do
      commits+=("${commit}")
    done < <(git rev-list --no-merges "${after}")
  else
    while IFS= read -r commit; do
      commits+=("${commit}")
    done < <(git rev-list --no-merges "${before}..${after}")
  fi
fi

if [[ "${#commits[@]}" -eq 0 ]]; then
  echo "No commits to lint."
else
  for commit in "${commits[@]}"; do
    lint_message "${commit}"
  done
fi
