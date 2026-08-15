#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${root}"

# Check local changes whether they are staged or unstaged.
git diff --check
git diff --cached --check

head="${GITHUB_SHA:-HEAD}"
if ! git rev-parse --verify --quiet "${head}^{commit}" >/dev/null; then
  echo "No committed tree is available; checked staged and unstaged changes only."
  exit 0
fi

base="${CHECK_BASE_SHA:-}"
if [[ -z "${base}" || "${base}" =~ ^0+$ ]]; then
  empty_tree="$(git hash-object -t tree /dev/null)"
  echo "Checking whitespace in the committed tree at ${head}."
  git diff --check "${empty_tree}" "${head}"
  exit 0
fi

if ! git rev-parse --verify --quiet "${base}^{commit}" >/dev/null; then
  echo "Whitespace check base commit is unavailable: ${base}" >&2
  exit 2
fi

echo "Checking whitespace in ${base}...${head}."
git diff --check "${base}...${head}"
