#!/usr/bin/env bash
set -euo pipefail

root=${SETUP_ROOT:-$(git rev-parse --show-toplevel)}

if ! bash "${root}/scripts/setup/setup-bootstrap.sh"; then
  echo "Full setup stopped during bootstrap." >&2
  exit 1
fi
if ! bash "${root}/scripts/setup/setup-refresh.sh"; then
  echo "Full setup stopped during refresh." >&2
  exit 1
fi
