#!/usr/bin/env bash
set -euo pipefail

commitlint_bin="${COMMITLINT_BIN:-commitlint}"

lint_message() {
  local commit=$1
  echo "Checking commit ${commit}"
  git show -s --format=%B "${commit}" | "${commitlint_bin}" lint
}

if [[ -n "${PR_BASE_SHA:-}" && -n "${PR_HEAD_SHA:-}" ]]; then
  mapfile -t commits < <(git rev-list --no-merges "${PR_BASE_SHA}..${PR_HEAD_SHA}")
else
  before="${PUSH_BEFORE_SHA:-}"
  after="${GITHUB_SHA:-HEAD}"
  if [[ -z "${before}" || "${before}" =~ ^0+$ ]]; then
    mapfile -t commits < <(git rev-list --no-merges "${after}")
  else
    mapfile -t commits < <(git rev-list --no-merges "${before}..${after}")
  fi
fi

if [[ "${#commits[@]}" -eq 0 ]]; then
  echo "No commits to lint."
else
  for commit in "${commits[@]}"; do
    lint_message "${commit}"
  done
fi

if [[ -n "${PR_TITLE:-}" ]]; then
  echo "Checking pull request title"
  printf '%s\n' "${PR_TITLE}" | "${commitlint_bin}" lint
fi
