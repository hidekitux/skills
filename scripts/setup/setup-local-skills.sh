#!/usr/bin/env bash
set -euo pipefail

root=${SETUP_ROOT:-$(git rev-parse --show-toplevel)}

bash "${root}/scripts/setup/setup-refresh.sh"
